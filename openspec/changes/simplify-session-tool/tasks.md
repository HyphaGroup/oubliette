# Tasks: Simplify Session Tool

## 1. Remove Dead/Redundant Parameters and Code
- [ ] 1.1 Remove `mode`, `change_id`, `build_all` from `SendMessageParams` and `SessionParams`
- [ ] 1.2 Remove `use_spec` from `SpawnParams`, `SessionParams`, `StartOptions`, `ExecuteRequest`
- [ ] 1.3 Remove `append_system_prompt` from `SpawnParams`, `SendMessageParams`, `SessionParams`, `StartOptions`, `ExecuteRequest`
- [ ] 1.4 Remove `SystemPrompt` concatenation from `opencode/runtime.go` (both `Execute` and `ExecuteStreaming`)
- [ ] 1.5 Inline child session context preamble directly into prompt in `handleSpawnChild`
- [ ] 1.6 Delete `transformMessageForMode()` function
- [ ] 1.7 Delete `getFirstIncompleteChange()` function
- [ ] 1.8 Delete `createBuildModeStateFile()` function and `BuildModeState` struct
- [ ] 1.9 Delete `SessionMode` type and constants from `session/types.go`
- [ ] 1.10 Delete `readFinalResponseFromSession()` and remove `FinalResponse` from `SessionEventsResult`
- [ ] 1.11 Remove build-mode auto-selection path from `handleSendMessage` (lines ~766-780)
- [ ] 1.12 Remove `Mode`/`ChangeID`/`BuildAll` fields from `TaskContext` struct on `ActiveSession`
- [ ] 1.13 Remove `SetTaskContext` calls that reference mode/changeID in `handleSendMessage`
- [ ] 1.14 Update `handlers_session_unified.go` to stop passing dead params
- [ ] 1.15 Update `session/streaming.go` and `session/manager.go` to stop threading `UseSpec` and `AppendSystemPrompt`

## 2. Rename Parameters
- [ ] 2.1 Rename `prompt` to `message` on `SpawnParams` (keep `prompt` as alias via both fields)
- [ ] 2.2 Update `SessionParams` unified struct to match
- [ ] 2.3 Update `handlers_session_unified.go` mapping functions

## 3. Update Tool Description
- [ ] 3.1 Replace session tool description in `tools_registry.go` with rich action-level guidance
- [ ] 3.2 Add per-action parameter hints in the JSON schema description fields

## 4. Clean Up Stale Comments
- [ ] 4.1 Remove `.factory/` path comments from `handlers_session.go`
- [ ] 4.2 Remove "OpenSpec session modes" section comments from `SendMessageParams`
- [ ] 4.3 Remove `SystemPrompt` field comment from `agent/types.go`
- [ ] 4.3 Update file-level comment block (DEPTH TRACKING, etc.) to remove mode references

## 5. Update Tests
- [ ] 5.1 Update `session/types_test.go` — remove `UseSpec` from test fixtures
- [ ] 5.2 Update `testutil/fixtures.go` — remove dead fields
- [ ] 5.3 Update any handler tests that reference mode/build params

## 6. Verification
- [ ] 6.1 `go build ./...`
- [ ] 6.2 `go test ./... -short`
- [ ] 6.3 `golangci-lint run --enable gocritic ./cmd/... ./internal/...`
- [ ] 6.4 Smoke test via MCP Inspector: spawn, message, events, get, list, end
