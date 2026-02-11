# Tasks: Tech Debt Sweep

## 1. Dead File Removal
- [x] 1.1 Delete `internal/config/config.go`
- [x] 1.2 Delete `internal/container/types.go`
- [x] 1.3 Delete `internal/container/Dockerfile`
- [x] 1.4 Delete `containers/osint/` directory
- [x] 1.5 Remove `osint` from comments in `handlers_project.go:49`, `agent/config/types.go:35`
- [x] 1.6 Update `config/loader_test.go` to use "dev" instead of "osint"

## 2. Remove Legacy Token Scope (`ScopeReadOnly`)
- [x] 2.1 Remove `ScopeReadOnly` constant from `internal/auth/types.go`
- [x] 2.2 Remove `ScopeReadOnly` from `IsAdminScope()` and `IsReadOnlyScope()` checks
- [x] 2.3 Update `internal/mcp/handlers_token.go` to not reference `ScopeReadOnly`
- [x] 2.4 Update `internal/mcp/registry.go` `GetToolsForScope()` to not reference `ScopeReadOnly`
- [x] 2.5 Update all test files to use `ScopeAdminRO` instead of `ScopeReadOnly`

## 3. Remove Legacy Tool Scope System
- [x] 3.1 Remove `Scope` field from `ToolDef` struct
- [x] 3.2 Remove `toolScopeAdmin`, `toolScopeWrite`, `toolScopeRead` constants from `tools.go`
- [x] 3.3 Remove `isToolAllowedForTokenScope()` from `registry.go`
- [x] 3.4 Remove legacy fallback from `IsToolAllowed()` in `permissions.go`
- [x] 3.5 Update `permissions_test.go` and `registry_test.go` to use `Target`/`Access`

## 4. Remove Dead `.factory/` Code
- [x] 4.1 Delete `readFinalResponseFromSession()` and its caller in `handleSessionEvents`
- [x] 4.2 Delete `createBuildModeStateFile()` and its caller in `handleSendMessage`
- [x] 4.3 Delete `BuildModeState` struct
- [x] 4.4 Delete `getFirstIncompleteChange()` and its caller (build-mode auto-select path)
- [x] 4.5 Remove stale `.factory/` comments from `handlers_session.go` and `cmd/server/main.go`

## 5. Remove Dead Session Mode System
- [x] 5.1 Delete `transformMessageForMode()` function
- [x] 5.2 Delete `SessionMode` type and constants from `session/types.go`
- [x] 5.3 Remove `Mode` field from `SendMessageParams`, `SessionParams`
- [x] 5.4 Remove `ChangeID` and `BuildAll` fields from `SendMessageParams`, `SessionParams`
- [x] 5.5 Remove `UseSpec` field from `SpawnParams`, `SessionParams`, `StartOptions`, `ExecuteRequest`
- [x] 5.6 Remove build-mode auto-selection path from `handleSendMessage`
- [x] 5.7 Update `handlers_session_unified.go` to stop passing Mode/ChangeID/BuildAll/UseSpec
- [x] 5.8 Update `session/streaming.go` and `session/manager.go` to stop threading UseSpec
- [x] 5.9 Update tests (`session/types_test.go`, `testutil/fixtures.go`)

## 6. Remove Unused Audit Operations
- [x] 6.1 Remove `OpSessionCreate`, `OpSessionEnd`, `OpSessionMessage`, `OpWorkspaceCreate`, `OpWorkspaceDelete`, `OpContainerStart`, `OpContainerStop`, `OpImageRebuild` from `audit/logger.go`

## 7. Remove Unused Metrics
- [x] 7.1 Remove `RecordToolCall`, `SetContainersRunning`, `SetProjectsTotal`, `RecordEventDrop`
- [x] 7.2 Remove corresponding gauge/counter variables (`ContainersRunning`, `ProjectsTotal`, `EventBufferDrops`, `ToolCalls`)

## 8. Remove Unused Backup Exports
- [x] 8.1 Remove `ExportManifest()` from `backup/backup.go`
- [x] 8.2 Remove `RestoreProject()` from `backup/backup.go`

## 9. Replace `os.IsNotExist` Anti-Pattern
- [x] 9.1 Replace all `os.IsNotExist(err)` with `errors.Is(err, fs.ErrNotExist)` (25 locations, 11 files)
- [x] 9.2 Add `"errors"` and `"io/fs"` imports where needed
- [x] 9.3 Fix variable shadow (`errors` vs `errors` package in backup.go, manager_test.go)

## 10. Fix Stale Comments
- [x] 10.1 Update `cmd/oubliette-relay/main.go` — replace "droid" with "agent" in protocol comments
- [x] 10.2 Remove credential_refs TODO from `handlers_container.go:357`

## 11. Verification
- [x] 11.1 `go build ./...` — passes
- [x] 11.2 `go test ./... -short` — all pass
- [x] 11.3 `golangci-lint run --enable gocritic ./cmd/... ./internal/...` — 0 issues
- [x] 11.4 `gofmt -w .` — clean
