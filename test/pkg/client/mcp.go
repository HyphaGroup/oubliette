package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPClient implements a client for Oubliette's MCP server
type MCPClient struct {
	serverURL string
	authToken string
	client    *mcp.Client
	session   *mcp.ClientSession
	ctx       context.Context
}

// Tool represents an MCP tool definition
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema,omitempty"`
}

// ToolResult represents the result of a tool invocation
type ToolResult struct {
	Content  []mcp.Content
	IsError  bool
	Metadata map[string]interface{}
}

// NewMCPClient creates a new MCP client for the given server URL
func NewMCPClient(serverURL string) *MCPClient {
	return &MCPClient{
		serverURL: serverURL,
		ctx:       context.Background(),
	}
}

// SetAuthToken sets the Bearer token for authentication
func (c *MCPClient) SetAuthToken(token string) {
	c.authToken = token
}

// authTransport wraps http.RoundTripper to add auth header
type authTransport struct {
	base  http.RoundTripper
	token string
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.token != "" {
		req.Header.Set("Authorization", "Bearer "+t.token)
	}
	return t.base.RoundTrip(req)
}

// Connect establishes a connection to the MCP server
func (c *MCPClient) Connect() error {
	// Create MCP client
	c.client = mcp.NewClient(&mcp.Implementation{
		Name:    "oubliette-test",
		Version: "0.1.0",
	}, nil)

	// Create HTTP client with auth if token is set
	httpClient := &http.Client{
		Timeout: 0, // No timeout for long-running operations
	}
	if c.authToken != "" {
		httpClient.Transport = &authTransport{
			base:  http.DefaultTransport,
			token: c.authToken,
		}
	}

	// Create streamable HTTP transport
	transport := &mcp.StreamableClientTransport{
		Endpoint:   c.serverURL,
		HTTPClient: httpClient,
	}

	// Connect to server
	session, err := c.client.Connect(c.ctx, transport, nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.session = session
	return nil
}

// ListTools retrieves all available tools from the server
func (c *MCPClient) ListTools() ([]Tool, error) {
	if c.session == nil {
		return nil, fmt.Errorf("not connected - call Connect() first")
	}

	// List tools using the session
	result, err := c.session.ListTools(c.ctx, &mcp.ListToolsParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	// Convert to our Tool type
	tools := make([]Tool, len(result.Tools))
	for i, t := range result.Tools {
		var inputSchema map[string]interface{}
		if t.InputSchema != nil {
			if schema, ok := t.InputSchema.(map[string]interface{}); ok {
				inputSchema = schema
			}
		}
		tools[i] = Tool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: inputSchema,
		}
	}

	return tools, nil
}

// InvokeTool calls the specified tool with the given parameters
func (c *MCPClient) InvokeTool(name string, params map[string]interface{}) (*ToolResult, error) {
	if c.session == nil {
		return nil, fmt.Errorf("not connected - call Connect() first")
	}

	// Call tool using the session
	result, err := c.session.CallTool(c.ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: params,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to call tool: %w", err)
	}

	return &ToolResult{
		Content:  result.Content,
		IsError:  result.IsError,
		Metadata: result.Meta,
	}, nil
}

// Close closes the client session
func (c *MCPClient) Close() error {
	if c.session != nil {
		return c.session.Close()
	}
	return nil
}

// GetToolContent extracts text content from a ToolResult
func (r *ToolResult) GetToolContent() string {
	if r == nil {
		return ""
	}

	var result string
	for _, content := range r.Content {
		// Check if it's a TextContent
		if textContent, ok := content.(*mcp.TextContent); ok {
			if result != "" {
				result += "\n"
			}
			result += textContent.Text
		}
	}

	return result
}
