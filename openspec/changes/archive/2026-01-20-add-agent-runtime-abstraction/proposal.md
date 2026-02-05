# Change: Add Agent Runtime Abstraction (Droid + OpenCode)

## Why

Oubliette is tightly coupled to Factory Droid as its AI execution engine. The `internal/droid/` package is directly referenced throughout session management, making it impossible to use alternative backends like OpenCode. This creates vendor lock-in and prevents local development without a Factory API key.

We need an abstraction layer similar to the existing `container.Runtime` interface that allows Docker and Apple Container to be used interchangeably.

## What Changes

- **ADDED** `internal/agent/` package with `Runtime` and `StreamingExecutor` interfaces
- **ADDED** Normalized `StreamEvent` type that works across all backends
- **ADDED** `internal/agent/droid/` implementation wrapping existing Droid code
- **ADDED** `internal/agent/opencode/` implementation using OpenCode SDK
- **MODIFIED** `session.Manager` to accept `agent.Runtime` interface instead of `*droid.Manager`
- **MODIFIED** `session.ActiveSession` to use `agent.StreamingExecutor` interface
- **MODIFIED** `session.EventBuffer` to use `*agent.StreamEvent`
- **ADDED** Runtime configuration in `config/server.json` (replaces env var)
- **ADDED** Factory function with auto-detection (prefer Droid if Factory API key configured)
- **ADDED** `agent_runtime` option in `project_options` response
- **ADDED** Per-project runtime override in `project_create`

## Impact

- **Affected specs**: agent-runtime (new capability)
- **Affected code**:
  - `internal/droid/` → moves to `internal/agent/droid/`
  - `internal/session/manager.go` - interface change
  - `internal/session/active.go` - interface change  
  - `internal/session/event_buffer.go` - type change
  - `internal/mcp/handlers_session.go` - use new interface
  - `cmd/server/main.go` - factory initialization
- **Related changes**:
  - `migrate-config-to-files` - provides `config/server.json` where runtime is configured
  - `add-model-configuration` - provides `config/models.json` for API keys

## Configuration Integration

Runtime selection lives in `config/server.json`:
```json
{
  "address": ":8080",
  "agent_runtime": "auto",
  "droid": {
    "default_model": "claude-sonnet-4-5"
  }
}
```

Values: `"auto"` (default), `"droid"`, `"opencode"`

Auto-detection logic:
1. If `config/factory.json` exists with valid API key → use Droid
2. Otherwise → use OpenCode

Per-project override via `project_create`:
```json
{
  "name": "local-dev-project",
  "agent_runtime": "opencode"
}
```

## Success Criteria

- Both Droid and OpenCode runtimes pass identical integration test suite
- Existing Droid functionality unchanged (no regression)
- OpenCode server lifecycle managed per-project (starts with container)
- Auto-detection works correctly based on config/ files
- `project_options` includes available runtimes
