// oubliette-client is an MCP server that runs inside containers.
// It provides tools to agents via stdio and communicates with the
// parent Oubliette server via the relay socket.
//
// It supports caller tool relay, where tools declared by the parent caller
// are registered as MCP tools and forwarded through the socket.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CallerToolDefinition defines a tool that can be called on the external caller
type CallerToolDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	InputSchema any    `json:"inputSchema,omitempty"`
}

// parentConn manages the connection to the parent Oubliette via relay socket
type parentConn struct {
	conn        net.Conn
	reader      *bufio.Reader
	mu          sync.Mutex
	nextID      int
	pending     map[int]chan json.RawMessage
	callerID    string // ID of the caller (e.g., "myapp")
	callerTools []CallerToolDefinition
	configReady chan struct{} // Signals when initial config is received
}

var parent *parentConn
var logFile *os.File
var mcpServer *mcp.Server  // Global reference for dynamic tool registration
var oublietteAPIKey string // API key for Oubliette tool access

func logf(format string, args ...any) {
	if logFile != nil {
		_, _ = fmt.Fprintf(logFile, format+"\n", args...)
		_ = logFile.Sync()
	}
}

func main() {
	// Open log file for debugging (stderr may be /dev/null)
	var err error
	logFile, err = os.OpenFile("/tmp/oubliette-client.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		logFile = nil // Fall back to no logging
	}
	logf("oubliette-client starting")
	socketPath := "/mcp/relay.sock"
	if len(os.Args) > 1 {
		socketPath = os.Args[1]
	}

	// Connect to relay socket as downstream
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "oubliette-client: failed to connect to %s: %v\n", socketPath, err)
		os.Exit(1)
	}

	// Send downstream header
	projectID := os.Getenv("OUBLIETTE_PROJECT_ID")
	if projectID == "" {
		projectID = "unknown"
	}
	header := fmt.Sprintf("OUBLIETTE-DOWNSTREAM %s\n", projectID)
	if _, err := conn.Write([]byte(header)); err != nil {
		_ = conn.Close()
		fmt.Fprintf(os.Stderr, "oubliette-client: failed to send header: %v\n", err)
		os.Exit(1)
	}

	// Initialize parent connection
	parent = &parentConn{
		conn:        conn,
		reader:      bufio.NewReader(conn),
		pending:     make(map[int]chan json.RawMessage),
		configReady: make(chan struct{}, 1), // Buffered so signal isn't lost
	}

	// Start reading from parent socket in background
	// This will signal configReady when caller_tools_config is received
	go parent.readLoop()

	// Create MCP server with tools
	mcpServer = mcp.NewServer(&mcp.Implementation{
		Name:    "oubliette-parent",
		Version: "0.1.0",
	}, &mcp.ServerOptions{
		HasTools: true,
	})

	// Add base tools (only session_message for unauthenticated access)
	// Authenticated clients get additional tools via OUBLIETTE_API_KEY
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "session_message",
		Description: "Send a message to a child session. Creates a new session if session_id not provided. Returns the final assistant response.",
	}, handleSessionMessage)

	// Wait for caller_tools_config before starting MCP server
	// This ensures caller tools are registered before Droid queries tools/list
	logf("waiting for caller_tools_config from parent...")
	select {
	case <-parent.configReady:
		logf("caller_tools_config received, starting MCP server")
	case <-time.After(5 * time.Second):
		logf("timeout waiting for caller_tools_config, starting MCP server anyway")
	}

	// Debug: Log tool count before starting server
	parent.mu.Lock()
	toolCount := len(parent.callerTools)
	callerID := parent.callerID
	parent.mu.Unlock()
	logf("DEBUG: Starting MCP server with %d caller tools from %s", toolCount, callerID)

	// Check for Oubliette API key and request tools if available
	oublietteAPIKey = os.Getenv("OUBLIETTE_API_KEY")
	if oublietteAPIKey != "" {
		logf("OUBLIETTE_API_KEY detected, requesting tools...")
		if err := requestOublietteTools(); err != nil {
			logf("WARNING: failed to get oubliette tools: %v", err)
		}
	}

	// Run MCP server over stdio
	if err := mcpServer.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		fmt.Fprintf(os.Stderr, "oubliette-client: server error: %v\n", err)
		os.Exit(1)
	}
}

// Tool input/output types
type SessionMessageInput struct {
	Message   string `json:"message" jsonschema:"message to send to the child session"`
	SessionID string `json:"session_id,omitempty" jsonschema:"optional session ID to continue; creates new if not provided"`
}

type SessionMessageOutput struct {
	SessionID string `json:"session_id"`
	Result    string `json:"result"`
	Spawned   bool   `json:"spawned"`
}

func handleSessionMessage(ctx context.Context, req *mcp.CallToolRequest, input SessionMessageInput) (*mcp.CallToolResult, any, error) {
	if input.Message == "" {
		return nil, SessionMessageOutput{}, fmt.Errorf("message is required")
	}

	// Call parent to spawn/send to session (returns immediately with session_id)
	result, err := callParent("session_message", map[string]any{
		"message":    input.Message,
		"session_id": input.SessionID,
	})
	if err != nil {
		return nil, SessionMessageOutput{}, fmt.Errorf("failed to send message: %w", err)
	}

	// Parse the response to get session_id
	var spawnResp struct {
		SessionID string `json:"session_id"`
		Spawned   bool   `json:"spawned"`
	}
	if err := json.Unmarshal(result, &spawnResp); err != nil {
		return nil, SessionMessageOutput{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Poll session_events until we get the result
	finalResult, err := waitForSessionResult(ctx, spawnResp.SessionID)
	if err != nil {
		return nil, SessionMessageOutput{}, fmt.Errorf("failed to get result: %w", err)
	}

	return nil, SessionMessageOutput{
		SessionID: spawnResp.SessionID,
		Result:    finalResult,
		Spawned:   spawnResp.Spawned,
	}, nil
}

// waitForSessionResult polls session_events until the session completes
func waitForSessionResult(ctx context.Context, sessionID string) (string, error) {
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-timeout:
			return "", fmt.Errorf("timeout waiting for session result")
		case <-ticker.C:
			result, err := callParent("session_events", map[string]any{
				"session_id":  sessionID,
				"since_index": 0,
			})
			if err != nil {
				continue // Retry on error
			}

			var events struct {
				SessionID string `json:"session_id"`
				Status    string `json:"status"`
				Events    []struct {
					Type string `json:"type"`
					Role string `json:"role,omitempty"`
					Text string `json:"text,omitempty"`
				} `json:"events"`
				Completed bool `json:"completed"`
				Failed    bool `json:"failed"`
			}
			if err := json.Unmarshal(result, &events); err != nil {
				continue
			}

			// Check if completed
			if events.Completed {
				// Find the assistant message
				for _, event := range events.Events {
					if event.Type == "message" && event.Role == "assistant" {
						return event.Text, nil
					}
				}
				return "Session completed", nil
			}

			if events.Failed {
				for _, event := range events.Events {
					if event.Type == "error" {
						return "", fmt.Errorf("session failed: %s", event.Text)
					}
				}
				return "", fmt.Errorf("session failed")
			}
		}
	}
}

// callParent sends a JSON-RPC request to the parent Oubliette server via the relay
func callParent(method string, params any) (json.RawMessage, error) {
	parent.mu.Lock()
	id := parent.nextID
	parent.nextID++

	// Create response channel
	respChan := make(chan json.RawMessage, 1)
	parent.pending[id] = respChan
	parent.mu.Unlock()

	defer func() {
		parent.mu.Lock()
		delete(parent.pending, id)
		parent.mu.Unlock()
	}()

	// Build request
	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
	}
	if params != nil {
		req["params"] = params
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')

	// Send request
	parent.mu.Lock()
	_, err = parent.conn.Write(data)
	parent.mu.Unlock()
	if err != nil {
		return nil, err
	}

	// Wait for response with timeout
	select {
	case result := <-respChan:
		return result, nil
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("timeout waiting for response")
	}
}

// readLoop reads responses from parent and dispatches to waiting callers.
// It also handles notifications like caller_tools_config.
func (p *parentConn) readLoop() {
	logf("readLoop started")
	decoder := json.NewDecoder(p.reader)
	for {
		var msg struct {
			// Common fields
			JSONRPC string `json:"jsonrpc"`
			ID      *int   `json:"id,omitempty"` // nil for notifications

			// Response fields
			Result json.RawMessage `json:"result,omitempty"`
			Error  *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error,omitempty"`

			// Notification fields
			Type   string          `json:"type,omitempty"`
			Method string          `json:"method,omitempty"`
			Params json.RawMessage `json:"params,omitempty"`
		}

		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				fmt.Fprintf(os.Stderr, "oubliette-client: connection closed\n")
			} else {
				fmt.Fprintf(os.Stderr, "oubliette-client: read error: %v\n", err)
			}
			return
		}

		logf("received message: ID=%v Type=%s Method=%s", msg.ID, msg.Type, msg.Method)

		// Handle notifications (no ID)
		if msg.ID == nil {
			logf("notification received, Type=%s Method=%s", msg.Type, msg.Method)
			if msg.Type == "caller_tools_config" || msg.Method == "caller_tools_config" {
				logf("handling caller_tools_config")
				p.handleCallerToolsConfig(msg.Params)
			}
			continue
		}

		// Handle responses
		p.mu.Lock()
		if ch, ok := p.pending[*msg.ID]; ok {
			if msg.Error != nil {
				// Send error as JSON
				errJSON, _ := json.Marshal(map[string]string{"error": msg.Error.Message})
				ch <- errJSON
			} else {
				ch <- msg.Result
			}
		}
		p.mu.Unlock()
	}
}

// handleCallerToolsConfig processes a caller_tools_config notification and registers caller tools
func (p *parentConn) handleCallerToolsConfig(params json.RawMessage) {
	var config struct {
		CallerID string                 `json:"caller_id"`
		Tools    []CallerToolDefinition `json:"tools"`
	}
	if err := json.Unmarshal(params, &config); err != nil {
		logf("failed to parse caller_tools_config: %v", err)
		fmt.Fprintf(os.Stderr, "oubliette-client: failed to parse caller_tools_config: %v\n", err)
		return
	}

	if config.CallerID == "" || len(config.Tools) == 0 {
		logf("caller_tools_config: no caller_id or tools")
		return
	}

	p.mu.Lock()
	p.callerID = config.CallerID
	p.callerTools = config.Tools
	p.mu.Unlock()

	logf("caller_tools_config: caller_id=%s, tools=%d", config.CallerID, len(config.Tools))
	fmt.Fprintf(os.Stderr, "oubliette-client: registering %d caller tools from %s\n", len(config.Tools), config.CallerID)

	// Register each tool with the MCP server
	for _, tool := range config.Tools {
		registerCallerTool(config.CallerID, tool)
	}

	// Signal that config is ready (non-blocking in case already signaled or no one waiting)
	select {
	case p.configReady <- struct{}{}:
		logf("signaled configReady")
	default:
		logf("configReady already signaled or closed")
	}
}

// CallerToolInput is the generic input type for dynamically registered caller tools
type CallerToolInput map[string]any

// CallerToolOutput is the generic output type for dynamically registered caller tools
type CallerToolOutput struct {
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

// registerCallerTool registers a caller tool with the MCP server
func registerCallerTool(callerID string, tool CallerToolDefinition) {
	// Create prefixed tool name: {caller_id}_{tool_name}
	toolName := fmt.Sprintf("%s_%s", callerID, tool.Name)

	// Convert inputSchema from any to *jsonschema.Schema
	// The SDK requires a proper jsonschema.Schema, not a raw map
	var schema *jsonschema.Schema
	if tool.InputSchema != nil {
		// Marshal to JSON and unmarshal to jsonschema.Schema
		schemaBytes, err := json.Marshal(tool.InputSchema)
		if err != nil {
			logf("ERROR: failed to marshal input schema for %s: %v", toolName, err)
			return
		}
		schema = &jsonschema.Schema{}
		if err := json.Unmarshal(schemaBytes, schema); err != nil {
			logf("ERROR: failed to unmarshal input schema for %s: %v", toolName, err)
			return
		}
	} else {
		// Default to empty object schema
		schema = &jsonschema.Schema{Type: "object"}
	}

	// Ensure Type is "object" as required by MCP SDK
	if schema.Type == "" {
		schema.Type = "object"
	}

	// Store the original tool name for the handler (closure captures it)
	originalToolName := tool.Name

	logf("registering caller tool: %s (original: %s, schema type: %s)", toolName, originalToolName, schema.Type)

	// Register the tool with the MCP server using mcp.AddTool
	// Wrap in recover to catch any panics from the SDK
	func() {
		defer func() {
			if r := recover(); r != nil {
				logf("PANIC registering tool %s: %v", toolName, r)
				fmt.Fprintf(os.Stderr, "oubliette-client: panic registering tool %s: %v\n", toolName, r)
			}
		}()
		mcp.AddTool(mcpServer, &mcp.Tool{
			Name:        toolName,
			Description: tool.Description,
			InputSchema: schema,
		}, func(ctx context.Context, req *mcp.CallToolRequest, input CallerToolInput) (*mcp.CallToolResult, any, error) {
			return handleCallerToolCall(ctx, originalToolName, input)
		})
	}()
}

// handleCallerToolCall handles a call to a caller tool by forwarding to the parent
func handleCallerToolCall(ctx context.Context, toolName string, args CallerToolInput) (*mcp.CallToolResult, any, error) {
	logf("caller tool call: %s args=%v", toolName, args)

	// Call parent with caller_tool method
	result, err := callParent("caller_tool", map[string]any{
		"tool":      toolName,
		"arguments": args,
	})
	if err != nil {
		return nil, CallerToolOutput{Error: err.Error()}, nil
	}

	// Check if result is an error
	var errResp struct {
		Error string `json:"error,omitempty"`
	}
	if json.Unmarshal(result, &errResp) == nil && errResp.Error != "" {
		return nil, CallerToolOutput{Error: errResp.Error}, nil
	}

	// Return the raw result
	var output any
	_ = json.Unmarshal(result, &output)
	return nil, CallerToolOutput{Result: output}, nil
}

// OublietteToolDefinition defines an Oubliette tool returned by oubliette_tools
type OublietteToolDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	InputSchema any    `json:"inputSchema,omitempty"`
}

// OublietteToolOutput is the generic output type for oubliette tools
type OublietteToolOutput struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content,omitempty"`
	IsError bool   `json:"isError,omitempty"`
	Error   string `json:"error,omitempty"`
}

// requestOublietteTools requests available tools from Oubliette server via socket
func requestOublietteTools() error {
	result, err := callParent("oubliette_tools", map[string]any{
		"api_key": oublietteAPIKey,
	})
	if err != nil {
		return fmt.Errorf("failed to call oubliette_tools: %w", err)
	}

	// Check for error in response
	var errResp struct {
		Error string `json:"error,omitempty"`
	}
	if json.Unmarshal(result, &errResp) == nil && errResp.Error != "" {
		return fmt.Errorf("oubliette_tools error: %s", errResp.Error)
	}

	// Parse tools response
	var resp struct {
		Tools []OublietteToolDefinition `json:"tools"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return fmt.Errorf("failed to parse oubliette_tools response: %w", err)
	}

	logf("oubliette_tools returned %d tools", len(resp.Tools))

	// Register each tool with oubliette_ prefix
	for _, tool := range resp.Tools {
		registerOublietteTool(tool)
	}

	return nil
}

// registerOublietteTool registers an Oubliette tool with the MCP server
func registerOublietteTool(tool OublietteToolDefinition) {
	// Create prefixed tool name: oubliette_{tool_name}
	toolName := fmt.Sprintf("oubliette_%s", tool.Name)

	// Convert inputSchema from any to *jsonschema.Schema
	var schema *jsonschema.Schema
	if tool.InputSchema != nil {
		schemaBytes, err := json.Marshal(tool.InputSchema)
		if err != nil {
			logf("ERROR: failed to marshal input schema for %s: %v", toolName, err)
			return
		}
		schema = &jsonschema.Schema{}
		if err := json.Unmarshal(schemaBytes, schema); err != nil {
			logf("ERROR: failed to unmarshal input schema for %s: %v", toolName, err)
			return
		}
	} else {
		schema = &jsonschema.Schema{Type: "object"}
	}

	if schema.Type == "" {
		schema.Type = "object"
	}

	// Store the original tool name for the handler
	originalToolName := tool.Name

	logf("registering oubliette tool: %s (original: %s)", toolName, originalToolName)

	// Register the tool with the MCP server
	func() {
		defer func() {
			if r := recover(); r != nil {
				logf("PANIC registering oubliette tool %s: %v", toolName, r)
			}
		}()
		mcp.AddTool(mcpServer, &mcp.Tool{
			Name:        toolName,
			Description: tool.Description,
			InputSchema: schema,
		}, func(ctx context.Context, req *mcp.CallToolRequest, input CallerToolInput) (*mcp.CallToolResult, any, error) {
			return handleOublietteToolCall(ctx, originalToolName, input)
		})
	}()
}

// handleOublietteToolCall handles a call to an Oubliette tool by forwarding to the parent
func handleOublietteToolCall(ctx context.Context, toolName string, args CallerToolInput) (*mcp.CallToolResult, any, error) {
	logf("oubliette tool call: %s args=%v", toolName, args)

	// Call parent with oubliette_call_tool method
	result, err := callParent("oubliette_call_tool", map[string]any{
		"api_key":   oublietteAPIKey,
		"tool":      toolName,
		"arguments": args,
	})
	if err != nil {
		return nil, OublietteToolOutput{Error: err.Error(), IsError: true}, nil
	}

	// Parse the response
	var output OublietteToolOutput
	if err := json.Unmarshal(result, &output); err != nil {
		return nil, OublietteToolOutput{Error: fmt.Sprintf("failed to parse response: %v", err), IsError: true}, nil
	}

	// Check for error in JSON-RPC response (different from tool error)
	var errResp struct {
		Error string `json:"error,omitempty"`
	}
	if json.Unmarshal(result, &errResp) == nil && errResp.Error != "" {
		return nil, OublietteToolOutput{Error: errResp.Error, IsError: true}, nil
	}

	return nil, output, nil
}
