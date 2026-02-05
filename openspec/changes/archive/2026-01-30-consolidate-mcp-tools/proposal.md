# Change: Consolidate MCP Tools

## Why

Currently 32 MCP tools with repetitive CRUD patterns:
- `project_create`, `project_list`, `project_get`, `project_delete`
- `session_spawn`, `session_message`, `session_get`, `session_list`, `session_end`, ...
- `token_create`, `token_list`, `token_revoke`
- etc.

This creates:
1. Large tool list makes discovery harder for LLM callers
2. Repetitive handler boilerplate
3. Inconsistent naming (some use verbs, some use nouns)

## What Changes

Consolidate CRUD tools into resource-based tools with `action` parameter:

```json
// Before: 4 separate tools
{"tool": "project_create", "params": {"name": "foo"}}
{"tool": "project_get", "params": {"project_id": "foo"}}

// After: 1 tool with action
{"tool": "project", "params": {"action": "create", "name": "foo"}}
{"tool": "project", "params": {"action": "get", "project_id": "foo"}}
```

**Consolidation Map:**

| Current (26 tools) | Consolidated (6 tools) |
|-------------------|------------------------|
| `project_create`, `project_list`, `project_get`, `project_delete` | `project` with actions: `create`, `list`, `get`, `delete` |
| `container_start`, `container_stop`, `container_logs`, `container_exec` | `container` with actions: `start`, `stop`, `logs`, `exec` |
| `session_spawn`, `session_message`, `session_get`, `session_list`, `session_end`, `session_events`, `session_cleanup` | `session` with actions: `spawn`, `message`, `get`, `list`, `end`, `events`, `cleanup` |
| `workspace_list`, `workspace_delete` | `workspace` with actions: `list`, `delete` |
| `token_create`, `token_list`, `token_revoke` | `token` with actions: `create`, `list`, `revoke` |
| `schedule_create`, `schedule_list`, `schedule_get`, `schedule_update`, `schedule_delete`, `schedule_trigger` | `schedule` with actions: `create`, `list`, `get`, `update`, `delete`, `trigger` |

**Tools that remain separate (6):**
- `project_options` - Discovery/meta tool
- `project_changes` - OpenSpec integration
- `project_tasks` - OpenSpec integration
- `image_rebuild` - Global infrastructure
- `caller_tool_response` - Event response pattern
- `config_limits` - Config inspection

**Result: 32 tools → 12 tools (62% reduction)**

## Impact

- **BREAKING**: All tool names change for consolidated tools
- Callers must update to use `action` parameter
- Handler logic restructured with action dispatch

## Files Changed

**Modified:**
- `internal/mcp/tools_registry.go` - New consolidated registrations
- `internal/mcp/handlers_project.go` - Action dispatch for project tool
- `internal/mcp/handlers_container.go` - Action dispatch for container tool
- `internal/mcp/handlers_session.go` - Action dispatch for session tool
- `internal/mcp/handlers_workspace.go` - Action dispatch for workspace tool
- `internal/mcp/handlers_token.go` - Action dispatch for token tool
- `internal/mcp/handlers_schedule.go` - Action dispatch for schedule tool
- `test/pkg/suites/*.go` - Update all test tool calls

**Deleted:**
- Nothing deleted, just consolidated
