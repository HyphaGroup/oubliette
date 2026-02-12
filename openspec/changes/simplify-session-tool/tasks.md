# Tasks: Simplify Session Tool

## 1. Remove Dead/Redundant Parameters and Code
- [x] 1.1 Remove `mode`, `change_id`, `build_all` from `SendMessageParams` and `SessionParams`
- [x] 1.2 Remove `use_spec` from `SpawnParams`, `SessionParams`, `StartOptions`, `ExecuteRequest`
- [x] 1.3 Remove `append_system_prompt` from `SpawnParams`, `SendMessageParams`, `SessionParams`, `StartOptions`, `ExecuteRequest`
- [x] 1.4 Remove `SystemPrompt` concatenation from `opencode/runtime.go` (both `Execute` and `ExecuteStreaming`)
- [x] 1.5 Inline child session context preamble directly into prompt in `handleSpawnChild`
- [x] 1.6 Delete `transformMessageForMode()` function
- [x] 1.7 Delete `getFirstIncompleteChange()` function
- [x] 1.8 Delete `createBuildModeStateFile()` function and `BuildModeState` struct
- [x] 1.9 Delete `SessionMode` type and constants from `session/types.go`
- [x] 1.10 Delete `readFinalResponseFromSession()` and remove `FinalResponse` from `SessionEventsResult`
- [x] 1.11 Remove build-mode auto-selection path from `handleSendMessage`
- [x] 1.12 Remove `Mode`/`ChangeID`/`BuildAll` fields from `TaskContext` struct on `ActiveSession`
- [x] 1.13 Remove `SetTaskContext` calls that reference mode/changeID in `handleSendMessage`
- [x] 1.14 Update `handlers_session_unified.go` to stop passing dead params
- [x] 1.15 Update `session/streaming.go` and `session/manager.go` to stop threading `UseSpec` and `AppendSystemPrompt`

## 2. Rename Parameters
- [x] 2.1 Rename `prompt` to `message` on `SpawnParams` (keep `prompt` as alias via both fields + GetMessage() helper)
- [x] 2.2 Update `SessionParams` unified struct to match
- [x] 2.3 Update `handlers_session_unified.go` mapping functions

## 3. Add Rich Tool Descriptions (ALL tools)
- [x] 3.1 `session` — actions, key behaviors (auto-resume, model defaults), per-action param guidance
- [x] 3.2 `project` — actions (create/list/get/delete/options), key params (repo_url, container_type, model)
- [x] 3.3 `container` — actions (start/stop/logs/exec), when to use each
- [x] 3.4 `container_refresh` — what it does, when it fails (active sessions)
- [x] 3.5 `workspace` — actions (list/delete), what workspaces are
- [x] 3.6 `token` — actions (create/list/revoke), scope formats
- [x] 3.7 `schedule` — actions (create/list/get/update/delete/trigger), cron format, session pinning
- [x] 3.8 `config_limits` — what recursion limits are, when to check
- [x] 3.9 `caller_tool_response` — how tool relay works, when this is needed
- [x] 3.10 `project_changes` — what OpenSpec changes are, output format
- [x] 3.11 `project_tasks` — what OpenSpec tasks are, output format

## 4. Clean Up Stale Comments
- [x] 4.1 Remove `.factory/` path comments from `handlers_session.go` (already done in tech-debt sweep)
- [x] 4.2 Remove "OpenSpec session modes" section comments from `SendMessageParams` (already done)
- [x] 4.3 Remove `SystemPrompt` field comment from `agent/types.go`
- [x] 4.4 Update file-level comment block (DEPTH TRACKING, etc.) to remove mode references (already clean)

## 5. Update Tests
- [x] 5.1 Update `session/types_test.go` — remove `AppendSystemPrompt` from test fixtures
- [x] 5.2 Update `testutil/fixtures.go` — remove dead fields (already clean)
- [x] 5.3 Update any handler tests that reference mode/build params (already clean)

## 6. Verification
- [x] 6.1 `go build ./...` — passes
- [x] 6.2 `go test ./... -short` — all pass
- [x] 6.3 `golangci-lint run --enable gocritic ./cmd/... ./internal/...` — 0 issues
- [ ] 6.4 Smoke test via MCP Inspector: spawn, message, events, get, list, end
