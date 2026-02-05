# Design: Expose Oubliette Tools to Container Droids

## Component Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Container (Apple/Docker)                  │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  Droid                                                 │  │
│  │  - Sees oubliette_* tools via MCP                     │  │
│  │  - Calls them like any other tool                     │  │
│  └───────────────────┬───────────────────────────────────┘  │
│                      │ stdio (MCP)                          │
│  ┌───────────────────▼───────────────────────────────────┐  │
│  │  oubliette-client                                      │  │
│  │  - Detects OUBLIETTE_API_KEY env var                  │  │
│  │  - Requests tool list via socket                      │  │
│  │  - Registers oubliette_* tools with MCP server        │  │
│  │  - Forwards tool calls to socket                      │  │
│  └───────────────────┬───────────────────────────────────┘  │
│                      │ unix socket (/mcp/relay.sock)        │
└──────────────────────┼──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│  oubliette-server (host)                                     │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  SocketHandler                                          │ │
│  │  - New method: oubliette_tools (discovery)             │ │
│  │  - New method: oubliette_call_tool (execution)         │ │
│  └───────────────────┬────────────────────────────────────┘ │
│                      │                                       │
│  ┌───────────────────▼────────────────────────────────────┐ │
│  │  Auth Middleware                                        │ │
│  │  - Validates API key                                    │ │
│  │  - Returns available tools based on scope              │ │
│  └───────────────────┬────────────────────────────────────┘ │
│                      │                                       │
│  ┌───────────────────▼────────────────────────────────────┐ │
│  │  Tool Handlers (existing)                               │ │
│  │  - project_create, token_create, etc.                  │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## Socket Protocol Extensions

### Method: `oubliette_tools`

Request available tools for an API key.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "oubliette_tools",
  "params": {
    "api_key": "oub_xxx..."
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "tools": [
      {
        "name": "project_create",
        "description": "Create new project...",
        "inputSchema": {"type": "object", "properties": {...}}
      },
      {
        "name": "project_list",
        "description": "List all projects...",
        "inputSchema": {"type": "object", "properties": {...}}
      }
    ]
  }
}
```

**Error (invalid key):**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32001,
    "message": "invalid or expired API key"
  }
}
```

### Method: `oubliette_call_tool`

Execute an Oubliette tool with API key auth.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "oubliette_call_tool",
  "params": {
    "api_key": "oub_xxx...",
    "tool": "project_create",
    "arguments": {
      "name": "my-project",
      "description": "A new project"
    }
  }
}
```

**Response (success):**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "content": [{"type": "text", "text": "{\"id\": \"proj_xxx\", ...}"}],
    "isError": false
  }
}
```

**Response (tool error):**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "content": [{"type": "text", "text": "project already exists"}],
    "isError": true
  }
}
```

## Tool Filtering by Scope

Token scopes determine which tools are available:

| Scope | Available Tools |
|-------|----------------|
| `admin` | All tools |
| `write` | project_*, container_*, session_*, workspace_*, image_rebuild |
| `read` | project_list, project_get, session_list, session_get, session_events, workspace_list, config_limits |

Tools like `token_create`, `token_list`, `token_revoke` require `admin` scope.

## oubliette-client Changes

### Startup Flow

```go
func main() {
    // ... existing setup ...

    // Check for Oubliette API key
    oublietteKey := os.Getenv("OUBLIETTE_API_KEY")
    if oublietteKey != "" {
        logf("OUBLIETTE_API_KEY detected, requesting tools...")
        if err := requestOublietteTools(oublietteKey); err != nil {
            logf("WARNING: failed to get oubliette tools: %v", err)
        }
    }

    // Wait for caller_tools_config...
    // Start MCP server...
}

func requestOublietteTools(apiKey string) error {
    result, err := callParent("oubliette_tools", map[string]any{
        "api_key": apiKey,
    })
    if err != nil {
        return err
    }
    
    var resp struct {
        Tools []ToolDefinition `json:"tools"`
    }
    if err := json.Unmarshal(result, &resp); err != nil {
        return err
    }
    
    // Store key for later tool calls
    oublietteAPIKey = apiKey
    
    // Register tools with oubliette_ prefix
    for _, tool := range resp.Tools {
        registerOublietteTool(tool)
    }
    
    return nil
}
```

### Tool Registration

```go
func registerOublietteTool(tool ToolDefinition) {
    toolName := "oubliette_" + tool.Name
    
    mcp.AddTool(mcpServer, &mcp.Tool{
        Name:        toolName,
        Description: tool.Description,
        InputSchema: tool.InputSchema,
    }, func(ctx context.Context, req *mcp.CallToolRequest, input map[string]any) (*mcp.CallToolResult, any, error) {
        return handleOublietteToolCall(ctx, tool.Name, input)
    })
}

func handleOublietteToolCall(ctx context.Context, toolName string, args map[string]any) (*mcp.CallToolResult, any, error) {
    result, err := callParent("oubliette_call_tool", map[string]any{
        "api_key":   oublietteAPIKey,
        "tool":      toolName,
        "arguments": args,
    })
    if err != nil {
        return nil, nil, err
    }
    
    // Parse and return result
    var resp struct {
        Content []map[string]any `json:"content"`
        IsError bool             `json:"isError"`
    }
    json.Unmarshal(result, &resp)
    
    // Convert to MCP result format
    // ...
}
```

## Server-Side Implementation

### SocketHandler Changes

```go
func (h *SocketHandler) processRequest(ctx context.Context, req *JSONRPCRequest, sessionID, projectID string, depth int) *JSONRPCResponse {
    switch req.Method {
    // ... existing methods ...
    case "oubliette_tools":
        return h.handleOublietteTools(ctx, req)
    case "oubliette_call_tool":
        return h.handleOublietteCallTool(ctx, req)
    // ...
    }
}

func (h *SocketHandler) handleOublietteTools(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
    var params struct {
        APIKey string `json:"api_key"`
    }
    json.Unmarshal(req.Params, &params)
    
    // Validate API key and get scope
    token, err := h.server.authManager.ValidateToken(params.APIKey)
    if err != nil {
        return errorResponse(req.ID, -32001, "invalid or expired API key")
    }
    
    // Get tools for this scope
    tools := h.server.getToolsForScope(token.Scope)
    
    return successResponse(req.ID, map[string]any{"tools": tools})
}

func (h *SocketHandler) handleOublietteCallTool(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
    var params struct {
        APIKey    string         `json:"api_key"`
        Tool      string         `json:"tool"`
        Arguments map[string]any `json:"arguments"`
    }
    json.Unmarshal(req.Params, &params)
    
    // Validate API key
    token, err := h.server.authManager.ValidateToken(params.APIKey)
    if err != nil {
        return errorResponse(req.ID, -32001, "invalid or expired API key")
    }
    
    // Check tool is allowed for scope
    if !h.server.isToolAllowedForScope(params.Tool, token.Scope) {
        return errorResponse(req.ID, -32002, "tool not allowed for this token scope")
    }
    
    // Build fake MCP request and dispatch to handler
    result, err := h.server.dispatchToolCall(ctx, params.Tool, params.Arguments, token)
    if err != nil {
        return errorResponse(req.ID, -32000, err.Error())
    }
    
    return successResponse(req.ID, result)
}
```

## Security Considerations

1. **API Key in Environment**: Key is passed via env var, never logged or persisted
2. **Per-Call Validation**: Key validated on every tool call, not cached
3. **Scope Enforcement**: Token scopes limit available tools
4. **No Elevation**: Container can't gain more access than the key provides
5. **Audit Trail**: Tool calls logged with token ID for tracing

## Testing Strategy

1. **Unit tests**: Tool filtering by scope
2. **Integration tests**: 
   - Discover tools with valid key
   - Discover tools with invalid key (should fail)
   - Call tool with valid key
   - Call tool with insufficient scope (should fail)
   - Call tool with invalid key (should fail)
3. **End-to-end**: Droid creates a project via oubliette_project_create
