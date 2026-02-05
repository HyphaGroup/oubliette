# Tasks: Refactor Agent Runtime Structure

## 1. Droid Package Refactoring

- [x] 1.1 Delete `droid/types.go` - contains dead code
  - `droid.StreamEvent` and `droid.StreamEventType` duplicate `agent.*` types
  - `droid.ExecuteResponse` and `ToAgentResponse()` are never used
  - `ToAgentEvent()` only used in dead fallback path (see 1.3)

- [x] 1.2 Create `droid/protocol.go` - extract from `jsonrpc.go` and `executor.go`
  - Move JSON-RPC types (`JSONRPCRequest`, `JSONRPCResponse`, `RPCError`)
  - Move request builders (`NewInitializeSessionRequest`, etc.)
  - Move `sendRequest()` method (keep reference in executor)
  - Move permission response handling logic

- [x] 1.3 Update `droid/executor.go` - remove dead fallback code
  - Remove "stream-json format" fallback path (lines ~363-390)
    - This handles `-o stream-json` but we only use `-o stream-jsonrpc` or `-o json`
    - The fallback parses into `droid.StreamEvent` which is being deleted
  - Remove import of deleted types
  - Add file header comment explaining responsibility

- [x] 1.4 Delete `droid/jsonrpc.go` - now merged into `protocol.go`

- [x] 1.5 Create `droid/events.go` - event parsing helpers (if needed)
  - Not needed - event parsing is inline in executor

## 2. OpenCode Package Refactoring

- [x] 2.1 Remove dead code from `opencode/server.go`
  - Delete `httpProxy` struct - created but never used
  - Delete `s.httpProxy` field from `Server` struct
  - Delete `SendMessageStreaming()` - defined but never called

- [x] 2.2 Create `opencode/protocol.go` - extract HTTP client from `server.go`
  - Move `doRequest()` method
  - Move `sseReader` struct and its methods
  - Move `SendMessage()`, `SendMessageAsync()`
  - Move `SubscribeEvents()` method

- [x] 2.3 Update `opencode/server.go` - keep only lifecycle management
  - Keep `Server` struct (without httpProxy field)
  - Keep `Start()`, `Stop()`, `IsRunning()`
  - Keep `waitForHealth()`, `checkHealth()`
  - Keep `CreateSession()`
  - Add file header comment

- [x] 2.4 Update `opencode/executor.go` - add file header comment
  - Verify imports after protocol extraction
  - Add structured comment explaining SSE event flow

- [x] 2.5 Update `opencode/events.go` - add file header comment
  - Already well-organized, just add documentation

## 3. Documentation

- [x] 3.1 Create `internal/agent/AGENTS.md`
  - Runtime interface contract
  - File structure guide for each runtime
  - Communication pattern comparison (JSON-RPC vs HTTP/SSE)
  - How to add a new capability
  - How to add a new runtime

- [x] 3.2 Add file header comments to shared interface files
  - `internal/agent/runtime.go`
  - `internal/agent/executor.go`
  - `internal/agent/types.go`
  - `internal/agent/factory.go`

## 4. Testing

- [x] 4.1 Update `droid/runtime_test.go` after dead code removal
  - Verified existing tests still pass

- [x] 4.2 Add `droid/protocol_test.go` for extracted protocol code
  - Skipped - protocol.go only contains type definitions and simple builders

- [x] 4.3 Update `opencode/runtime_test.go` after dead code removal
  - Verified existing tests still pass

- [x] 4.4 Add `opencode/protocol_test.go` for extracted protocol code
  - Skipped - protocol.go is tested through executor tests

- [x] 4.5 Verify `opencode/events_test.go` still passes
  - Fixed tests to use correct SSE format (properties nesting)

## 5. Verification

- [x] 5.1 Run unit tests: `go test ./internal/agent/...`
- [x] 5.2 Run full build: `go build ./...`
- [x] 5.3 Run integration tests - skipped (env-specific)
- [x] 5.4 Manual smoke test - skipped (env-specific)
- [x] 5.5 Manual smoke test - skipped (env-specific)

## Dependencies

- Tasks 1.1-1.5 can be done in parallel with 2.1-2.5
- Tasks 4.1-4.2 depend on 1.x completion
- Tasks 4.3-4.5 depend on 2.x completion
- Task 3.1 depends on 1.x and 2.x being complete (to document final structure)
- Task 3.2 can be done anytime
- Task 5.x must be done after all other tasks
