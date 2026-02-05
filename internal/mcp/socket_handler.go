// socket_handler.go handles JSON-RPC communication over unix sockets.
// This provides tool execution for oubliette-client running inside containers.
//
// Architecture:
// - Container runs oubliette-relay which listens on /mcp/relay.sock
// - Socket is published to host via --publish-socket
// - oubliette-client (MCP server for droid) connects to relay as "downstream"
// - This handler connects to the published socket as "upstream"
// - Relay pairs upstream/downstream and pipes bytes
// - oubliette-client sends tool requests (session_spawn, project_list)
// - This handler processes requests and returns results
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/HyphaGroup/oubliette/internal/agent"
	"github.com/HyphaGroup/oubliette/internal/logger"
	"github.com/HyphaGroup/oubliette/internal/session"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// childSession tracks an async child session execution
type childSession struct {
	SessionID   string
	Status      string // "running", "completed", "failed"
	Result      string
	Error       string
	StartedAt   time.Time
	CompletedAt time.Time
	// Context for nested spawning
	WorkspaceID string
	ContainerID string
	ProjectID   string
}

// SocketHandler manages upstream connections to container relay sockets
type SocketHandler struct {
	server        *Server
	connections   map[string]net.Conn      // sessionID -> connection
	childSessions map[string]*childSession // child sessionID -> session state
	mu            sync.RWMutex
	childMu       sync.RWMutex
	childCounter  int
}

// NewSocketHandler creates a new socket handler
func NewSocketHandler(server *Server) *SocketHandler {
	return &SocketHandler{
		server:        server,
		connections:   make(map[string]net.Conn),
		childSessions: make(map[string]*childSession),
	}
}

// ConnectSession connects to the relay for a session and starts handling requests.
// Called when a droid session starts. Runs in a goroutine.
// The context is used for cancellation propagation - when ctx is cancelled, the connection closes.
func (h *SocketHandler) ConnectSession(ctx context.Context, projectID, sessionID string, depth int) error {
	socketPath := SocketPath(projectID)

	// Wait for socket to appear
	if err := waitForSocket(socketPath, SocketConnectTimeout); err != nil {
		return fmt.Errorf("socket not ready: %w", err)
	}

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to relay: %w", err)
	}

	// Close connection when context is cancelled
	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	// Send upstream header
	header := fmt.Sprintf("OUBLIETTE-UPSTREAM %s %s %d\n", sessionID, projectID, depth)
	if _, err := conn.Write([]byte(header)); err != nil {
		_ = conn.Close()
		return fmt.Errorf("failed to send header: %w", err)
	}

	// Track connection
	h.mu.Lock()
	h.connections[sessionID] = conn
	h.mu.Unlock()

	logger.Info("Connected to relay as upstream for session %s", sessionID)

	// Handle requests from oubliette-client - derive context with MCP headers
	reqCtx := WithMCPHeaders(ctx, sessionID, projectID, depth)

	h.handleRequests(reqCtx, bufio.NewReader(conn), conn, sessionID, projectID, depth)

	// Cleanup when done
	h.mu.Lock()
	delete(h.connections, sessionID)
	h.mu.Unlock()

	return nil
}

// CloseSession closes the connection for a session
func (h *SocketHandler) CloseSession(sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if conn, exists := h.connections[sessionID]; exists {
		_ = conn.Close()
		delete(h.connections, sessionID)
		logger.Info("Closed socket connection for session %s", sessionID)
	}
}

// Close closes all connections
func (h *SocketHandler) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for sessionID, conn := range h.connections {
		_ = conn.Close()
		delete(h.connections, sessionID)
	}
}

// sendCallerToolsConfig sends a caller_tools_config notification if the session has caller tools
func (h *SocketHandler) sendCallerToolsConfig(conn net.Conn, sessionID string) {
	activeSess, ok := h.server.activeSessions.Get(sessionID)
	if !ok {
		return
	}

	callerID, tools := activeSess.GetCallerTools()
	if callerID == "" || len(tools) == 0 {
		return
	}

	// Send caller_tools_config notification
	notification := map[string]any{
		"jsonrpc": "2.0",
		"type":    "caller_tools_config",
		"params": map[string]any{
			"caller_id": callerID,
			"tools":     tools,
		},
	}

	data, err := json.Marshal(notification)
	if err != nil {
		logger.Error("Failed to marshal caller_tools_config: %v", err)
		return
	}
	data = append(data, '\n')

	if _, err := conn.Write(data); err != nil {
		logger.Error("Failed to send caller_tools_config for session %s: %v", sessionID, err)
		return
	}

	logger.Info("Sent caller_tools_config to session %s with caller_id: %s, tools: %d", sessionID, callerID, len(tools))
}

// handleRequests processes JSON-RPC requests from oubliette-client
func (h *SocketHandler) handleRequests(ctx context.Context, reader *bufio.Reader, conn net.Conn, sessionID, projectID string, depth int) {
	decoder := json.NewDecoder(reader)
	logger.Info("SocketHandler: listening for requests from session %s (waiting for tool calls)", sessionID)

	// Send caller_tools_config notification if session has caller tools
	h.sendCallerToolsConfig(conn, sessionID)

	for {
		var request JSONRPCRequest
		logger.Info("SocketHandler: waiting for next request from session %s...", sessionID)
		if err := decoder.Decode(&request); err != nil {
			if err.Error() == "EOF" {
				logger.Info("SocketHandler: connection closed (EOF) for session %s", sessionID)
			} else {
				logger.Error("JSON decode error for session %s: %v (type: %T)", sessionID, err, err)
			}
			return
		}

		logger.Info("SocketHandler: received %s (id=%v) for session %s", request.Method, request.ID, sessionID)

		// Process the request
		response := h.processRequest(ctx, &request, sessionID, projectID, depth)

		// Write response
		responseBytes, _ := json.Marshal(response)
		responseBytes = append(responseBytes, '\n')
		if _, err := conn.Write(responseBytes); err != nil {
			logger.Error("Failed to write response for session %s: %v", sessionID, err)
			return
		}
		logger.Info("SocketHandler: sent response for %s", request.Method)
	}
}

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      any           `json:"id"`
	Result  any           `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (h *SocketHandler) processRequest(ctx context.Context, req *JSONRPCRequest, sessionID, projectID string, depth int) *JSONRPCResponse {
	switch req.Method {
	case "session_message":
		return h.handleSessionMessage(ctx, req, sessionID, projectID, depth)
	case "session_events":
		return h.handleSessionEvents(ctx, req)
	case "project_list":
		return h.handleProjectList(ctx, req)
	case "caller_tool":
		return h.handleCallerTool(ctx, req, sessionID)
	case "oubliette_tools":
		return h.handleOublietteTools(ctx, req)
	case "oubliette_call_tool":
		return h.handleOublietteCallTool(ctx, req)
	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32601,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
		}
	}
}

func (h *SocketHandler) handleSessionMessage(ctx context.Context, req *JSONRPCRequest, parentSessionID, projectID string, depth int) *JSONRPCResponse {
	var params struct {
		Message   string `json:"message"`
		SessionID string `json:"session_id,omitempty"`
	}
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &JSONRPCError{
					Code:    -32602,
					Message: "Invalid params: " + err.Error(),
				},
			}
		}
	}

	if params.Message == "" {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32602,
				Message: "message is required",
			},
		}
	}

	logger.Info("SocketHandler: session_message called with message: %s (parent: %s, depth: %d)", params.Message, parentSessionID, depth)

	// Get parent session info (workspace, container) - check both active sessions and child sessions
	var workspaceID, containerID string

	// First try the main active sessions manager
	if parentSession, ok := h.server.activeSessions.Get(parentSessionID); ok {
		workspaceID = parentSession.WorkspaceID
		containerID = parentSession.ContainerID
	} else {
		// Check if this is a child session calling to spawn a grandchild
		h.childMu.RLock()
		childSess, isChild := h.childSessions[parentSessionID]
		h.childMu.RUnlock()

		if isChild && childSess.WorkspaceID != "" {
			workspaceID = childSess.WorkspaceID
			containerID = childSess.ContainerID
		} else {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &JSONRPCError{
					Code:    -32000,
					Message: fmt.Sprintf("Parent session %s not found", parentSessionID),
				},
			}
		}
	}

	// Check recursion depth using project config
	proj, err := h.server.projectMgr.Get(projectID)
	if err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32000,
				Message: fmt.Sprintf("Project %s not found: %v", projectID, err),
			},
		}
	}
	maxDepth := h.server.projectMgr.GetMaxDepth(proj)
	if depth >= maxDepth {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32000,
				Message: fmt.Sprintf("Max recursion depth (%d) exceeded", maxDepth),
			},
		}
	}

	// Generate child session ID
	h.childMu.Lock()
	h.childCounter++
	childSessionID := fmt.Sprintf("child_%s_%d", parentSessionID, h.childCounter)

	// Create child session record with context for potential grandchildren
	child := &childSession{
		SessionID:   childSessionID,
		Status:      "running",
		StartedAt:   time.Now(),
		WorkspaceID: workspaceID,
		ContainerID: containerID,
		ProjectID:   projectID,
	}
	h.childSessions[childSessionID] = child
	h.childMu.Unlock()

	logger.Info("Spawning async child session %s for message: %s", childSessionID, params.Message)

	// Execute asynchronously with streaming (MCP enabled)
	// Child needs its own upstream connection to the relay so its oubliette-client can communicate
	go func() {
		childDepth := depth + 1
		// Working directory depends on workspace isolation setting
		var workingDir string
		if proj.WorkspaceIsolation {
			workingDir = fmt.Sprintf("/workspace/%s", workspaceID)
		} else {
			workingDir = fmt.Sprintf("/workspace/workspaces/%s", workspaceID)
		}

		// Connect upstream for the child session BEFORE starting the executor
		// This way when child's oubliette-client connects as downstream, it will pair with our upstream
		socketPath := SocketPath(projectID)

		childConn, err := net.Dial("unix", socketPath)
		if err != nil {
			h.childMu.Lock()
			if cs, ok := h.childSessions[childSessionID]; ok {
				cs.CompletedAt = time.Now()
				cs.Status = "failed"
				cs.Error = fmt.Sprintf("failed to connect upstream for child: %v", err)
			}
			h.childMu.Unlock()
			logger.Error("Child session %s failed to connect upstream: %v", childSessionID, err)
			return
		}

		// Send upstream header for child session
		header := fmt.Sprintf("OUBLIETTE-UPSTREAM %s %s %d\n", childSessionID, projectID, childDepth)
		if _, err := childConn.Write([]byte(header)); err != nil {
			_ = childConn.Close()
			h.childMu.Lock()
			if cs, ok := h.childSessions[childSessionID]; ok {
				cs.CompletedAt = time.Now()
				cs.Status = "failed"
				cs.Error = fmt.Sprintf("failed to send child header: %v", err)
			}
			h.childMu.Unlock()
			logger.Error("Child session %s failed to send header: %v", childSessionID, err)
			return
		}

		logger.Info("Connected upstream for child session %s (depth %d)", childSessionID, childDepth)

		// Create child context with timeout - derived from parent context for proper cancellation
		// When parent context is cancelled, child operations should also cancel
		childCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()

		// Add MCP headers to child context
		childCtx = WithMCPHeaders(childCtx, childSessionID, projectID, childDepth)

		// Close connection when child context is cancelled
		go func() {
			<-childCtx.Done()
			_ = childConn.Close()
		}()

		// Handle child's MCP requests in a separate goroutine
		go func() {
			defer func() { _ = childConn.Close() }()
			h.handleRequests(childCtx, bufio.NewReader(childConn), childConn, childSessionID, projectID, childDepth)
			logger.Info("Child session %s MCP handler finished", childSessionID)
		}()

		// Now start the streaming executor - child's oubliette-client will pair with our upstream
		execReq := &agent.ExecuteRequest{
			Prompt:        params.Message,
			ContainerID:   containerID,
			WorkingDir:    workingDir,
			ProjectID:     projectID,
			Depth:         childDepth,
			StreamJSONRPC: true, // Streaming with MCP enabled
		}

		// Get runtime for this project (may be different from server default)
		projectRuntime := h.server.GetRuntimeForProject(proj)
		executor, err := projectRuntime.ExecuteStreaming(childCtx, execReq)
		if err != nil {
			_ = childConn.Close() // Close the upstream connection
			h.childMu.Lock()
			if cs, ok := h.childSessions[childSessionID]; ok {
				cs.CompletedAt = time.Now()
				cs.Status = "failed"
				cs.Error = fmt.Sprintf("failed to start streaming session: %v", err)
			}
			h.childMu.Unlock()
			logger.Error("Child session %s failed to start: %v", childSessionID, err)
			return
		}
		defer func() { _ = executor.Close() }()

		// Collect events until completion
		var finalResult string
		for event := range executor.Events() {
			if event.Type == agent.StreamEventCompletion {
				finalResult = event.FinalText
				break
			}
			// Could also collect message events if we want intermediate results
			if event.Type == agent.StreamEventMessage && event.Role == "assistant" {
				finalResult = event.Text // Keep updating with latest assistant text
			}
		}

		// Check for errors
		select {
		case err := <-executor.Errors():
			h.childMu.Lock()
			if cs, ok := h.childSessions[childSessionID]; ok {
				cs.CompletedAt = time.Now()
				cs.Status = "failed"
				cs.Error = err.Error()
			}
			h.childMu.Unlock()
			logger.Error("Child session %s failed: %v", childSessionID, err)
			return
		default:
		}

		// Mark as completed
		h.childMu.Lock()
		if cs, ok := h.childSessions[childSessionID]; ok {
			cs.CompletedAt = time.Now()
			cs.Status = "completed"
			cs.Result = finalResult
			logger.Info("Child session %s completed with result length: %d", childSessionID, len(finalResult))
		}
		h.childMu.Unlock()
	}()

	// Return immediately with session ID
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"session_id": childSessionID,
			"spawned":    true,
		},
	}
}

// handleSessionEvents returns events/status from a child session
func (h *SocketHandler) handleSessionEvents(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		SessionID  string `json:"session_id"`
		SinceIndex int    `json:"since_index"`
	}
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &JSONRPCError{
					Code:    -32602,
					Message: "Invalid params: " + err.Error(),
				},
			}
		}
	}

	// Look up child session
	h.childMu.RLock()
	child, ok := h.childSessions[params.SessionID]
	h.childMu.RUnlock()

	if !ok {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32000,
				Message: fmt.Sprintf("Session %s not found", params.SessionID),
			},
		}
	}

	// Build events based on status
	events := []map[string]any{}
	completed := false
	failed := false

	switch child.Status {
	case "completed":
		completed = true
		// Add a message event with the result
		events = append(events, map[string]any{
			"index": 1,
			"type":  "message",
			"role":  "assistant",
			"text":  child.Result,
		})
	case "failed":
		failed = true
		events = append(events, map[string]any{
			"index": 1,
			"type":  "error",
			"text":  child.Error,
		})
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"session_id": params.SessionID,
			"status":     child.Status,
			"last_index": len(events),
			"events":     events,
			"completed":  completed,
			"failed":     failed,
		},
	}
}
func (h *SocketHandler) handleProjectList(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	logger.Info("SocketHandler: project_list called")

	projects, err := h.server.projectMgr.List(nil)
	if err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32000,
				Message: "Failed to list projects: " + err.Error(),
			},
		}
	}

	// Convert to response format
	projectList := make([]map[string]any, 0, len(projects))
	for _, p := range projects {
		projectList = append(projectList, map[string]any{
			"id":          p.ID,
			"name":        p.Name,
			"created_at":  p.CreatedAt,
			"description": p.Description,
		})
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"projects": projectList,
		},
	}
}

// Caller tool relay constants
const (
	callerToolTimeout = 60 * time.Second
)

// handleCallerTool handles requests to execute tools on the external caller.
// It pushes a caller_tool_request event via SSE and waits for the caller to respond.
func (h *SocketHandler) handleCallerTool(ctx context.Context, req *JSONRPCRequest, sessionID string) *JSONRPCResponse {
	var params struct {
		Tool      string         `json:"tool"`
		Arguments map[string]any `json:"arguments"`
	}
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &JSONRPCError{
					Code:    -32602,
					Message: "Invalid params: " + err.Error(),
				},
			}
		}
	}

	if params.Tool == "" {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32602,
				Message: "tool is required",
			},
		}
	}

	// Get the active session
	activeSess, ok := h.server.activeSessions.Get(sessionID)
	if !ok {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32000,
				Message: fmt.Sprintf("Session %s not found", sessionID),
			},
		}
	}

	// Check if caller tools are configured
	if !activeSess.HasCallerTools() {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32000,
				Message: "No caller tools configured for this session",
			},
		}
	}

	// Get MCP session for SSE push
	mcpSession := activeSess.GetMCPSession()
	if mcpSession == nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32000,
				Message: "No MCP client connected to receive tool request",
			},
		}
	}

	// Generate unique request ID
	requestID := uuid.New().String()

	// Register pending request
	responseCh := activeSess.RegisterCallerRequest(requestID)

	// Push caller_tool_request event via SSE
	if err := h.pushCallerToolRequest(ctx, activeSess, requestID, params.Tool, params.Arguments); err != nil {
		activeSess.CancelCallerRequest(requestID)
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32000,
				Message: fmt.Sprintf("Failed to push tool request: %v", err),
			},
		}
	}

	logger.Info("Pushed caller_tool_request for session %s: tool=%s, request_id=%s", sessionID, params.Tool, requestID)

	// Wait for response with timeout
	select {
	case response, ok := <-responseCh:
		if !ok {
			// Channel was closed (cancelled)
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &JSONRPCError{
					Code:    -32000,
					Message: "Request was cancelled",
				},
			}
		}
		if response.Error != "" {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &JSONRPCError{
					Code:    -32000,
					Message: response.Error,
				},
			}
		}
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  response.Result,
		}

	case <-time.After(callerToolTimeout):
		activeSess.CancelCallerRequest(requestID)
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32000,
				Message: fmt.Sprintf("Caller tool request timed out after %v", callerToolTimeout),
			},
		}

	case <-ctx.Done():
		activeSess.CancelCallerRequest(requestID)
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32000,
				Message: "Request cancelled",
			},
		}
	}
}

// pushCallerToolRequest pushes a caller_tool_request event to the MCP client via Log()
func (h *SocketHandler) pushCallerToolRequest(ctx context.Context, activeSess *session.ActiveSession, requestID, tool string, arguments map[string]any) error {
	mcpSession := activeSess.GetMCPSession()
	if mcpSession == nil {
		return fmt.Errorf("no MCP client connected")
	}

	// Build event data
	eventData := map[string]any{
		"type":       "caller_tool_request",
		"session_id": activeSess.SessionID,
		"request_id": requestID,
		"tool":       tool,
		"arguments":  arguments,
	}

	params := &mcp.LoggingMessageParams{
		Logger: "oubliette.caller_tool",
		Level:  "info",
		Data:   eventData,
	}

	return mcpSession.Log(ctx, params)
}

// handleOublietteTools returns the list of available Oubliette tools for a given API key
func (h *SocketHandler) handleOublietteTools(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		APIKey string `json:"api_key"`
	}
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &JSONRPCError{
					Code:    -32602,
					Message: "Invalid params: " + err.Error(),
				},
			}
		}
	}

	if params.APIKey == "" {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32602,
				Message: "api_key is required",
			},
		}
	}

	// Validate API key
	token, err := h.server.authStore.ValidateToken(params.APIKey)
	if err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32001,
				Message: "invalid or expired API key",
			},
		}
	}

	// Get tools for this scope
	tools := h.server.getToolsForScope(token.Scope)

	logger.Info("SocketHandler: oubliette_tools returned %d tools for scope %s", len(tools), token.Scope)

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"tools": tools,
		},
	}
}

// handleOublietteCallTool executes an Oubliette tool with API key auth
func (h *SocketHandler) handleOublietteCallTool(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		APIKey    string         `json:"api_key"`
		Tool      string         `json:"tool"`
		Arguments map[string]any `json:"arguments"`
	}
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &JSONRPCError{
					Code:    -32602,
					Message: "Invalid params: " + err.Error(),
				},
			}
		}
	}

	if params.APIKey == "" {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32602,
				Message: "api_key is required",
			},
		}
	}

	if params.Tool == "" {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32602,
				Message: "tool is required",
			},
		}
	}

	// Validate API key
	token, err := h.server.authStore.ValidateToken(params.APIKey)
	if err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32001,
				Message: "invalid or expired API key",
			},
		}
	}

	// Extract project ID from arguments for project-scoped permission check
	projectID := ExtractProjectIDFromArgs(params.Arguments)

	// Check tool is allowed for scope (with project check for project-scoped tokens)
	if !h.server.isToolAllowedWithProject(params.Tool, token.Scope, projectID) {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32002,
				Message: fmt.Sprintf("tool %s not allowed for token scope %s", params.Tool, token.Scope),
			},
		}
	}

	logger.Info("SocketHandler: oubliette_call_tool called for tool=%s, scope=%s, project=%s", params.Tool, token.Scope, projectID)

	// Dispatch to tool handler
	result, err := h.server.dispatchToolCall(ctx, params.Tool, params.Arguments, token)
	if err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32000,
				Message: err.Error(),
			},
		}
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}
