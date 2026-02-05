# Change: Use Per-Project Agent Runtime

## Why

The `add-agent-runtime-abstraction` change introduced:
- `agent_runtime` field in `Project` metadata and `CreateProjectRequest`
- Server-level runtime selection (`config/server.json`)
- Factory function to create Droid or OpenCode runtimes

However, the per-project `agent_runtime` setting is **stored but never used**. When sessions spawn, they always use the server-wide runtime from `session.Manager.agentRuntime`, ignoring the project's configured runtime.

Users expect that setting `"agent_runtime": "opencode"` on project creation will cause that project's sessions to use OpenCode, not Droid.

## What Changes

- **MODIFIED** Session spawning to check project's `agent_runtime` field
- **ADDED** Runtime factory accessible at spawn time (to create runtime per-project if different from server default)
- **MODIFIED** `session.Manager` to support runtime override per-spawn

**NOT changing:**
- Server-level default runtime (remains in `config/server.json`)
- Project metadata schema (already has `agent_runtime` field)
- `project_create` API (already accepts `agent_runtime` parameter)

## Impact

- **Affected specs**: Extends `agent-runtime` capability (from `add-agent-runtime-abstraction`)
- **Affected code**:
  - `internal/session/manager.go` - Accept optional runtime override in spawn methods
  - `internal/session/streaming.go` - Pass runtime override through
  - `internal/mcp/handlers_session.go` - Resolve project runtime before spawning
  - `internal/mcp/server.go` - Expose runtime factory for per-project runtime creation
- **Breaking changes**: None (additive - default behavior unchanged)
