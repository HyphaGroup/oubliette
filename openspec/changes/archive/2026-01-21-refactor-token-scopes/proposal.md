# Proposal: Refactor Token Scopes

## Summary

Replace the current `admin`/`write`/`read-only` token scope model with a cleaner two-tier system:
- **Admin scope** with read-write (`admin`) or read-only (`admin:ro`) access to all projects
- **Project scope** with read-write (`project:<uuid>`) or read-only (`project:<uuid>:ro`) access to a single project

## Motivation

The current scoping model has issues:
1. A `write` token can create/delete ANY project - effectively admin-level access
2. No way to restrict a token to a single project
3. `read-only` is global but rarely useful (can see everything but do nothing)
4. Tool-to-scope mapping is implicit and hard to track

## Proposed Scope Model

| Scope | Format | Access |
|-------|--------|--------|
| Admin RW | `admin` | All tools, all projects |
| Admin RO | `admin:ro` | Read-only tools, all projects |
| Project RW | `project:<uuid>` | Read/write tools for one project |
| Project RO | `project:<uuid>:ro` | Read-only tools for one project |

## Tool Permission Model

Each tool declares what it needs:
- **Target**: `global` (system-wide) or `project` (operates on a project)
- **Access**: `read` or `write`

```go
Register(r, ToolDef{
    Name:   "project_delete",
    Target: TargetProject,  // operates on a project
    Access: AccessWrite,    // modifies data
}, handler)
```

Permission logic:
- `admin` → all tools
- `admin:ro` → tools with `Access: read` only
- `project:<uuid>` → project-targeted tools for that project + global read tools
- `project:<uuid>:ro` → project-targeted read tools for that project + global read tools

## Tool Classification

**Global + Admin-only** (token management):
- `token_create`, `token_list`, `token_revoke`

**Global + Write** (system operations):
- `image_rebuild`

**Global + Read** (system info):
- `project_list`, `project_options`

**Project + Write**:
- `project_create`, `project_delete`
- `container_start`, `container_exec`, `container_stop`
- `session_spawn`, `session_message`, `session_end`, `session_cleanup`
- `workspace_delete`, `caller_tool_response`

**Project + Read**:
- `project_get`, `project_changes`, `project_tasks`
- `container_logs`
- `session_get`, `session_list`, `session_events`
- `workspace_list`, `config_limits`

## Migration

- Existing `admin` tokens → remain `admin`
- Existing `read-only` tokens → become `admin:ro`
- Existing `project:<uuid>` tokens → remain `project:<uuid>` (already supported)
- No `write` scope exists in practice (not exposed in CLI)

## Benefits

1. **Least privilege**: Give container agents only access to their project
2. **Clear mental model**: Admin vs project, read vs write
3. **Explicit tool permissions**: Easy to audit what each scope can do
4. **Future-proof**: Easy to add new permission dimensions if needed
