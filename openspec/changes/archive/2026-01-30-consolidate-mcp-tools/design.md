# Design: Consolidate MCP Tools

## Context

MCP tools follow common CRUD patterns but are exposed as separate tools. This inflates the tool count and makes the API surface harder to navigate.

## Goals

1. Reduce tool count from 32 to 12 (62% reduction)
2. Consistent `action` parameter pattern across all resource tools
3. Maintain exact same functionality - just reorganized
4. Clear error messages for invalid actions

## Non-Goals

- Changing any business logic
- Adding new capabilities
- Changing response formats

## Decisions

### Action Parameter Pattern

Every consolidated tool has a required `action` parameter as the first parameter:

```json
{
  "tool": "project",
  "params": {
    "action": "create",
    "name": "my-project",
    "source_url": "https://github.com/..."
  }
}
```

Actions are validated against an enum. Invalid action returns error:
```
unknown action 'foo' for project tool; valid actions: create, list, get, delete
```

### Handler Structure

Each consolidated handler dispatches to action-specific sub-handlers:

```go
func (s *Server) handleProject(ctx context.Context, req *mcp.CallToolRequest, params *ProjectParams) (*mcp.CallToolResult, any, error) {
    switch params.Action {
    case "create":
        return s.projectCreate(ctx, params)
    case "list":
        return s.projectList(ctx, params)
    case "get":
        return s.projectGet(ctx, params)
    case "delete":
        return s.projectDelete(ctx, params)
    default:
        return nil, nil, fmt.Errorf("unknown action '%s' for project tool; valid actions: create, list, get, delete", params.Action)
    }
}
```

Sub-handlers are private methods (lowercase) that contain the actual logic - mostly unchanged from current handlers.

### Parameter Validation

Parameters are validated per-action. The params struct includes all possible fields, but validation happens in sub-handlers:

```go
type ProjectParams struct {
    Action    string `json:"action"`              // Required: create, list, get, delete
    
    // For create
    Name      string `json:"name,omitempty"`
    SourceURL string `json:"source_url,omitempty"`
    
    // For get, delete
    ProjectID string `json:"project_id,omitempty"`
}
```

### Tool Definition Schema

Tool schemas list all parameters with descriptions indicating which actions use them:

```go
Register(r, ToolDef{
    Name:        "project",
    Description: "Manage projects. Actions: create, list, get, delete",
    Target:      TargetGlobal, // Most permissive for the tool
    Access:      AccessWrite,  // Most permissive for the tool
}, s.handleProject)
```

Access control is checked per-action in the handler, not at registration.

### Access Control Per Action

Since different actions have different access levels, we check in the handler:

```go
func (s *Server) handleProject(ctx context.Context, req *mcp.CallToolRequest, params *ProjectParams) (*mcp.CallToolResult, any, error) {
    switch params.Action {
    case "list":
        // Read access - no additional check needed
    case "create", "delete":
        // Write access checked at registration level
    }
    // ...
}
```

For tools where actions span read/write/admin, register at the lowest common denominator and check in handler.

### Consolidated Tool Definitions

| Tool | Actions | Target | Access |
|------|---------|--------|--------|
| `project` | create, list, get, delete | Global | Write (list/get check Read) |
| `container` | start, stop, logs, exec | Project | Write (logs checks Read) |
| `session` | spawn, message, get, list, end, events, cleanup | Project | Write (get/list/events check Read) |
| `workspace` | list, delete | Project | Write (list checks Read) |
| `token` | create, list, revoke | Global | Admin |
| `schedule` | create, list, get, update, delete, trigger | Global | Write (list/get check Read) |

### Tools That Remain Separate

These don't fit the CRUD pattern well:

1. **`project_options`** - Meta/discovery, doesn't operate on a project
2. **`project_changes`** - OpenSpec-specific, could be `openspec` tool later
3. **`project_tasks`** - OpenSpec-specific, could be `openspec` tool later
4. **`image_rebuild`** - Infrastructure operation, not resource CRUD
5. **`caller_tool_response`** - Event callback, not resource CRUD
6. **`config_limits`** - Read-only inspection, could merge into `project` as action later

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| Breaking change for all callers | Clean break, update tests as reference |
| Action validation overhead | Minimal - single switch statement |
| Mixed access levels per tool | Per-action access checks in handler |

## Alternative Considered

**Keep separate tools, just rename for consistency**: Rejected because it doesn't reduce tool count or improve discoverability.
