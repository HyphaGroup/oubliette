# Design: Token Scope Refactoring

## Current State

### Scope Constants (internal/auth/types.go)
```go
const (
    ScopeAdmin    = "admin"
    ScopeReadOnly = "read-only"
)
func ScopeProject(projectID string) string // returns "project:<uuid>"
```

### Tool Scope Constants (internal/mcp/tools.go)
```go
const (
    toolScopeAdmin = "admin" // admin only
    toolScopeWrite = "write" // admin + write
    toolScopeRead  = "read"  // all scopes
)
```

### Permission Check (internal/mcp/registry.go)
```go
func isToolAllowedForTokenScope(toolScope, tokenScope string) bool {
    if tokenScope == auth.ScopeAdmin {
        return true
    }
    switch toolScope {
    case toolScopeAdmin:
        return tokenScope == auth.ScopeAdmin
    case toolScopeWrite:
        return tokenScope != auth.ScopeReadOnly
    case toolScopeRead:
        return true
    }
    return false
}
```

## Proposed Design

### New Scope Format

```
admin           - Full access to everything
admin:ro        - Read-only access to everything
project:<uuid>  - Full access to one project
project:<uuid>:ro - Read-only access to one project
```

### New Tool Permission Model

Replace single `Scope` string with structured permissions:

```go
// internal/mcp/tools.go

type ToolTarget string
const (
    TargetGlobal  ToolTarget = "global"  // System-wide operation
    TargetProject ToolTarget = "project" // Operates on a specific project
)

type ToolAccess string
const (
    AccessRead  ToolAccess = "read"
    AccessWrite ToolAccess = "write"
    AccessAdmin ToolAccess = "admin" // Admin-only (token management)
)

// ToolDef updated
type ToolDef struct {
    Name        string
    Description string
    Target      ToolTarget // What does this tool operate on?
    Access      ToolAccess // What level of access is required?
    InputSchema map[string]any
}
```

### Permission Logic

```go
func isToolAllowed(tool *ToolDef, tokenScope string, projectID string) bool {
    // Parse token scope
    isAdmin := tokenScope == "admin"
    isAdminRO := tokenScope == "admin:ro"
    isProjectScope := strings.HasPrefix(tokenScope, "project:")
    isReadOnly := strings.HasSuffix(tokenScope, ":ro")
    
    scopeProjectID := ""
    if isProjectScope {
        // Extract project ID: "project:<uuid>" or "project:<uuid>:ro"
        parts := strings.Split(tokenScope, ":")
        scopeProjectID = parts[1]
    }
    
    // Admin-only tools (token management)
    if tool.Access == AccessAdmin {
        return isAdmin // Only full admin, not admin:ro
    }
    
    // Write access check
    if tool.Access == AccessWrite && isReadOnly {
        return false
    }
    
    // Global tools accessible by admin scopes
    if tool.Target == TargetGlobal {
        return isAdmin || isAdminRO
    }
    
    // Project tools need matching project or admin scope
    if tool.Target == TargetProject {
        if isAdmin || isAdminRO {
            return true
        }
        if isProjectScope && scopeProjectID == projectID {
            return true
        }
    }
    
    return false
}
```

### Tool Classifications

```go
// Token management - admin only
Register(r, ToolDef{Name: "token_create", Target: TargetGlobal, Access: AccessAdmin}, ...)
Register(r, ToolDef{Name: "token_list",   Target: TargetGlobal, Access: AccessAdmin}, ...)
Register(r, ToolDef{Name: "token_revoke", Target: TargetGlobal, Access: AccessAdmin}, ...)

// Global system operations
Register(r, ToolDef{Name: "project_list",    Target: TargetGlobal, Access: AccessRead}, ...)
Register(r, ToolDef{Name: "project_options", Target: TargetGlobal, Access: AccessRead}, ...)
Register(r, ToolDef{Name: "project_create",  Target: TargetGlobal, Access: AccessWrite}, ...) // Creates new project
Register(r, ToolDef{Name: "image_rebuild",   Target: TargetGlobal, Access: AccessWrite}, ...)

// Project-scoped operations
Register(r, ToolDef{Name: "project_get",     Target: TargetProject, Access: AccessRead}, ...)
Register(r, ToolDef{Name: "project_delete",  Target: TargetProject, Access: AccessWrite}, ...)
Register(r, ToolDef{Name: "project_changes", Target: TargetProject, Access: AccessRead}, ...)
Register(r, ToolDef{Name: "project_tasks",   Target: TargetProject, Access: AccessRead}, ...)

Register(r, ToolDef{Name: "container_start", Target: TargetProject, Access: AccessWrite}, ...)
Register(r, ToolDef{Name: "container_exec",  Target: TargetProject, Access: AccessWrite}, ...)
Register(r, ToolDef{Name: "container_stop",  Target: TargetProject, Access: AccessWrite}, ...)
Register(r, ToolDef{Name: "container_logs",  Target: TargetProject, Access: AccessRead}, ...)

Register(r, ToolDef{Name: "session_spawn",   Target: TargetProject, Access: AccessWrite}, ...)
Register(r, ToolDef{Name: "session_message", Target: TargetProject, Access: AccessWrite}, ...)
Register(r, ToolDef{Name: "session_get",     Target: TargetProject, Access: AccessRead}, ...)
Register(r, ToolDef{Name: "session_list",    Target: TargetProject, Access: AccessRead}, ...)
Register(r, ToolDef{Name: "session_events",  Target: TargetProject, Access: AccessRead}, ...)
Register(r, ToolDef{Name: "session_end",     Target: TargetProject, Access: AccessWrite}, ...)
Register(r, ToolDef{Name: "session_cleanup", Target: TargetProject, Access: AccessWrite}, ...)

Register(r, ToolDef{Name: "workspace_list",   Target: TargetProject, Access: AccessRead}, ...)
Register(r, ToolDef{Name: "workspace_delete", Target: TargetProject, Access: AccessWrite}, ...)

Register(r, ToolDef{Name: "config_limits",         Target: TargetProject, Access: AccessRead}, ...)
Register(r, ToolDef{Name: "caller_tool_response",  Target: TargetProject, Access: AccessWrite}, ...)
```

### Extracting Project ID from Tool Call

For project-scoped permission checks, we need the project ID from the request. Options:

1. **From parameters** - Most tools have `project_id` parameter
2. **From session context** - Session tools may derive project from session ID
3. **From MCP context** - Headers may contain project context

Recommended: Check parameters first, then session lookup, then MCP context.

```go
func extractProjectID(toolName string, args map[string]any, ctx context.Context) string {
    // Direct project_id parameter
    if pid, ok := args["project_id"].(string); ok && pid != "" {
        return pid
    }
    
    // Session ID -> lookup project
    if sid, ok := args["session_id"].(string); ok && sid != "" {
        if sess, err := sessionMgr.Load(sid); err == nil {
            return sess.ProjectID
        }
    }
    
    // MCP context (for child sessions)
    if mcpCtx := ExtractMCPContext(ctx); mcpCtx.ProjectID != "" {
        return mcpCtx.ProjectID
    }
    
    return ""
}
```

### Migration Strategy

1. Keep `ScopeAdmin` and `ScopeReadOnly` constants for backwards compatibility
2. Map old scopes to new behavior:
   - `admin` → `admin` (unchanged)
   - `read-only` → `admin:ro` (treated as admin read-only)
   - `project:<uuid>` → `project:<uuid>` (unchanged, gains proper enforcement)
3. Update `oubliette-token create` CLI to accept new scope formats
4. Update `project_options` to document available scopes

### Testing Strategy

1. Unit tests for `isToolAllowed` with all scope/tool combinations
2. Integration tests for each scope type accessing various tools
3. Negative tests ensuring project tokens can't access other projects
