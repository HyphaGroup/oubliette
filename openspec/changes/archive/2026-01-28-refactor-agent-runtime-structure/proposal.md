# Change: Refactor Agent Runtime Structure for Comparability

## Why

The Droid and OpenCode agent runtimes implement the same `agent.Runtime` and `agent.StreamingExecutor` interfaces but have inconsistent file organization and naming. This makes it difficult to:

1. Compare how each runtime handles specific capabilities (MCP tools, permissions, events)
2. Understand what code to modify when adding new runtime features
3. Onboard new contributors or add future runtimes

The runtimes have fundamentally different communication patterns (CLI/stdin-stdout vs HTTP/SSE), but the *responsibilities* are the same. Aligning file structure around responsibilities rather than implementation details improves maintainability.

## What Changes

### Dead Code Removal

**Droid package:**
- Delete `types.go` entirely - contains unused duplicate types:
  - `droid.StreamEvent` / `droid.StreamEventType` duplicate `agent.*` types
  - `droid.ExecuteResponse` and `ToAgentResponse()` are never called
  - `ToAgentEvent()` only used in dead fallback code path
- Remove "stream-json format" fallback in `executor.go` (lines ~363-390)
  - Handles `-o stream-json` output format, but we only use `-o stream-jsonrpc` or `-o json`
  - This is the only code that uses the types being deleted

**OpenCode package:**
- Delete `httpProxy` struct and field - created but never used
- Delete `SendMessageStreaming()` method - defined but never called

### Code Organization

Normalize both runtime packages to have parallel file structure:

| File | Responsibility | Droid (CLI) | OpenCode (HTTP) |
|------|----------------|-------------|-----------------|
| `runtime.go` | `agent.Runtime` interface impl | ✓ exists | ✓ exists |
| `executor.go` | `agent.StreamingExecutor` impl | ✓ exists (cleanup) | ✓ exists |
| `protocol.go` | Communication layer | NEW (from jsonrpc.go) | NEW (from server.go) |
| `events.go` | Event parsing helpers | MAYBE (see tasks) | ✓ exists |
| `command.go` | CLI command building | ✓ exists | N/A |
| `server.go` | Server lifecycle | N/A | ✓ exists (cleanup) |

### Documentation

Add `internal/agent/AGENTS.md` with:
- Runtime interface contract documentation
- File structure guide for each runtime
- How to add a new runtime
- Communication pattern diagrams

### Code Comments

Add structured comments to each file explaining:
- Purpose and responsibility
- Key types/functions
- Relationship to other files
- Runtime-specific behavior

## Impact

- **Affected code**: `internal/agent/droid/`, `internal/agent/opencode/`
- **No breaking changes**: All interfaces remain the same
- **No behavioral changes**: Pure refactoring of file organization
- **New file**: `internal/agent/AGENTS.md`

## Non-Goals

- Adding new capabilities to either runtime
- Changing the Runtime or StreamingExecutor interfaces
- Unifying the communication protocols (they're fundamentally different)
- Performance optimization
