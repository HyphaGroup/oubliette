# MCP Tool Development

This document covers developing MCP tools for Oubliette, including the caller tool relay and exposing tools to container droids.

## Adding a New Tool

### 1. Define Parameters

Create the parameter struct in `internal/mcp/types.go`:

```go
type NewFeatureParams struct {
    ProjectID   string `json:"project_id"`
    RequiredArg string `json:"required_arg"`
    OptionalArg *int   `json:"optional_arg,omitempty"`
}
```

### 2. Implement Handler

Add handler in `internal/mcp/handlers_*.go`:

```go
func (s *Server) handleNewFeature(
    ctx context.Context,
    request *mcp.CallToolRequest,
    params *NewFeatureParams,
) (*mcp.CallToolResult, any, error) {
    // 1. Extract MCP context
    mcpCtx := ExtractMCPContext(ctx)

    // 2. Validate parameters
    if params.RequiredArg == "" {
        return nil, nil, fmt.Errorf("required_arg is required")
    }

    // 3. Delegate to manager
    result, err := s.manager.DoOperation(ctx, params)
    if err != nil {
        return nil, nil, fmt.Errorf("failed: %w", err)
    }

    // 4. Format response
    return formatResponse(result), result, nil
}
```

### 3. Register Tool

In `internal/mcp/server.go`:

```go
mcp.AddTool(server, &mcp.Tool{
    Name:        "example_new_feature",
    Description: "Does X to accomplish Y",
    InputSchema: buildSchema(NewFeatureParams{}),
}, s.handleNewFeature)
```

### 4. Add Tests

In `test/pkg/suites/<appropriate>.go`:

```go
{
    Name:        "test_new_feature",
    Description: "Test new feature functionality",
    Tags:        []string{"feature", "category"},
    Execute: func(ctx *testpkg.TestContext) error {
        result, err := ctx.Client.InvokeTool("example_new_feature", map[string]any{
            "project_id":   projectID,
            "required_arg": "value",
        })
        ctx.Assertions.AssertNoError(err, "Tool should succeed")
        ctx.Assertions.AssertFalse(result.IsError, "Should not return error")
        return nil
    },
}
```

## Available MCP Tools

Oubliette uses unified tools with an `action` parameter for related operations.

### Unified Tools

#### `project` - Project Management
| Action | Description |
|--------|-------------|
| `create` | Create new project with config |
| `list` | List all projects |
| `get` | Get project details |
| `delete` | Delete project and resources |
| `options` | Get available models, accounts, container types |

```json
{"action": "create", "name": "my-project", "description": "..."}
{"action": "list"}
{"action": "get", "project_id": "..."}
{"action": "delete", "project_id": "..."}
{"action": "options"}
```

#### `container` - Container Management
| Action | Description |
|--------|-------------|
| `start` | Start project container |
| `stop` | Stop project container |
| `exec` | Execute command in container |
| `logs` | Get container logs |

```json
{"action": "start", "project_id": "..."}
{"action": "stop", "project_id": "..."}
{"action": "exec", "project_id": "...", "command": "ls -la"}
{"action": "logs", "project_id": "..."}
```

#### `session` - Session Management
| Action | Description |
|--------|-------------|
| `spawn` | Spawn or resume session |
| `message` | Send message to active session |
| `get` | Get session details |
| `list` | List sessions for a project |
| `end` | End session gracefully |
| `events` | Retrieve buffered events |
| `cleanup` | Delete old session metadata |

```json
{"action": "spawn", "project_id": "...", "prompt": "..."}
{"action": "message", "project_id": "...", "message": "..."}
{"action": "get", "session_id": "..."}
{"action": "list", "project_id": "..."}
{"action": "end", "session_id": "..."}
{"action": "events", "session_id": "...", "since_index": 0}
{"action": "cleanup", "project_id": "...", "max_age_hours": 24}
```

#### `workspace` - Workspace Management
| Action | Description |
|--------|-------------|
| `list` | List all workspaces |
| `delete` | Delete workspace |

```json
{"action": "list", "project_id": "..."}
{"action": "delete", "project_id": "...", "workspace_id": "..."}
```

#### `token` - API Token Management
| Action | Description |
|--------|-------------|
| `create` | Create API token |
| `list` | List tokens |
| `revoke` | Revoke token |

```json
{"action": "create", "name": "my-token", "scope": "admin"}
{"action": "list"}
{"action": "revoke", "token_id": "..."}
```

#### `schedule` - Scheduled Task Management
| Action | Description |
|--------|-------------|
| `create` | Create scheduled task |
| `list` | List schedules |
| `get` | Get schedule details |
| `update` | Update schedule |
| `delete` | Delete schedule |
| `trigger` | Manually trigger schedule |

```json
{"action": "create", "name": "...", "cron_expr": "0 * * * *", "prompt": "...", "targets": [...]}
{"action": "list"}
{"action": "get", "schedule_id": "..."}
{"action": "update", "schedule_id": "...", "enabled": false}
{"action": "delete", "schedule_id": "..."}
{"action": "trigger", "schedule_id": "..."}
```

### Standalone Tools

| Tool | Description |
|------|-------------|
| `project_changes` | List OpenSpec changes for a project |
| `project_tasks` | Get task details for an OpenSpec change |
| `image_rebuild` | Rebuild container image |
| `caller_tool_response` | Respond to caller tool request |
| `config_limits` | Get recursion limits |

---

## Caller Tool Relay

The Caller Tool Relay enables agents inside Oubliette containers to call tools on the external caller that initiated the session.

### How It Works

```
Droid → oubliette-client → socket → oubliette-server → SSE event → Caller
                                                                      ↓
Droid ← oubliette-client ← socket ← oubliette-server ← MCP call ← Caller executes
```

1. **Caller declares tools**: Pass `caller_id` and `caller_tools` in `session_message` context
2. **Agent sees prefixed tools**: Tools appear as `{caller_id}_{tool_name}` (e.g., `myapp_send_notification`)
3. **Agent calls tool**: Request flows through socket to oubliette-server
4. **Server pushes SSE event**: `caller_tool_request` event sent to caller
5. **Caller executes and responds**: Caller calls `caller_tool_response` MCP tool
6. **Result returns to agent**: Response flows back through socket

### Configuration

Pass caller tools in `session_message` context:

```json
{
  "project_id": "...",
  "message": "...",
  "context": {
    "caller_id": "myapp",
    "caller_tools": [
      {
        "name": "send_notification",
        "description": "Send a notification to the user",
        "inputSchema": {
          "type": "object",
          "properties": {
            "message": {"type": "string"},
            "recipients": {"type": "array", "items": {"type": "string"}}
          },
          "required": ["message", "recipients"]
        }
      }
    ]
  }
}
```

### SSE Event: caller_tool_request

```json
{
  "type": "caller_tool_request",
  "session_id": "gogol_xxx",
  "request_id": "uuid-v4",
  "tool": "send_notification",
  "arguments": {"message": "Hello", "recipients": ["user@example.com"]}
}
```

### MCP Tool: caller_tool_response

```json
{
  "name": "caller_tool_response",
  "arguments": {
    "session_id": "gogol_xxx",
    "request_id": "uuid-v4",
    "result": {"status": "sent"},
    "error": null
  }
}
```

### Timeout

Default: 60 seconds. If caller doesn't respond, the agent receives an error.

---

## Exposing Oubliette Tools to Container Agents

Agents inside containers can access Oubliette's MCP tools when configured with an API key.

### How It Works

```
Droid → oubliette-client → socket → oubliette-server → Tool Handler
                                                            ↓
Droid ← oubliette-client ← socket ← oubliette-server ← Result
```

1. **API key provided**: Set `OUBLIETTE_API_KEY` environment variable in container
2. **Client discovers tools**: On startup, oubliette-client calls `oubliette_tools`
3. **Tools registered**: Tools appear as `oubliette_{tool_name}` (e.g., `oubliette_project_create`)
4. **Agent calls tool**: Request routed through socket with API key
5. **Server validates and executes**: API key checked, scope enforced

### Token Scope Model

| Scope Format | Description |
|--------------|-------------|
| `admin` | Full access to all tools and projects |
| `admin:ro` | Read-only access to all tools and projects |
| `project:<uuid>` | Full access to one project only |
| `project:<uuid>:ro` | Read-only access to one project only |

### Tool Permission Model

| Access Level | Tools |
|--------------|-------|
| **Admin-only** | `token_create`, `token_list`, `token_revoke` |
| **Global + Write** | `project_create`, `image_rebuild` |
| **Global + Read** | `project_list`, `project_options` |
| **Project + Write** | `project_delete`, `container_*`, `session_*`, `workspace_delete` |
| **Project + Read** | `project_get`, `session_get`, `session_list`, `workspace_list` |

### Tool Naming

All Oubliette tools are prefixed with `oubliette_` inside containers:
- `oubliette_project` (with action: create, list, get, delete, options)
- `oubliette_session` (with action: spawn, message, get, list, end, events, cleanup)
- `oubliette_container` (with action: start, stop, exec, logs)
- `oubliette_workspace` (with action: list, delete)
- `oubliette_token` (admin only, with action: create, list, revoke)
- `oubliette_schedule` (with action: create, list, get, update, delete, trigger)

### Socket Methods

**`oubliette_tools`** - Discover available tools:
```json
{"jsonrpc": "2.0", "id": 1, "method": "oubliette_tools", "params": {"api_key": "oub_xxx..."}}
```

**`oubliette_call_tool`** - Execute a tool:
```json
{"jsonrpc": "2.0", "id": 2, "method": "oubliette_call_tool", "params": {"api_key": "oub_xxx...", "tool": "project_list", "arguments": {}}}
```

### Use Cases

- **Admin agent**: Create and manage projects on behalf of users
- **Orchestrator agent**: Spawn sessions across multiple projects
- **Self-improving agent**: Create new workspaces and branch experiments
