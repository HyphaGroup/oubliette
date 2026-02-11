# Tasks: Tech Debt Sweep

## 1. Dead File Removal
- [ ] 1.1 Delete `internal/config/config.go`
- [ ] 1.2 Delete `internal/container/types.go`
- [ ] 1.3 Delete `internal/container/Dockerfile`
- [ ] 1.4 Delete `containers/osint/` directory
- [ ] 1.5 Remove `osint` from comments in `handlers_project.go:49`, `agent/config/types.go:35`
- [ ] 1.6 Update `config/loader_test.go` to use "base" or "dev" instead of "osint"

## 2. Remove Legacy Token Scope (`ScopeReadOnly`)
- [ ] 2.1 Remove `ScopeReadOnly` constant from `internal/auth/types.go`
- [ ] 2.2 Remove `ScopeReadOnly` from `IsAdminScope()` and `IsReadOnlyScope()` checks
- [ ] 2.3 Update `internal/mcp/handlers_token.go` to not reference `ScopeReadOnly`
- [ ] 2.4 Update `internal/mcp/registry.go` `GetToolsForScope()` to not reference `ScopeReadOnly`
- [ ] 2.5 Update all test files to use `ScopeAdminRO` instead of `ScopeReadOnly`

## 3. Remove Legacy Tool Scope System
- [ ] 3.1 Remove `Scope` field from `ToolDef` struct
- [ ] 3.2 Remove `toolScopeAdmin`, `toolScopeWrite`, `toolScopeRead` constants from `tools.go`
- [ ] 3.3 Remove `isToolAllowedForTokenScope()` from `registry.go`
- [ ] 3.4 Remove legacy fallback from `IsToolAllowed()` in `permissions.go`
- [ ] 3.5 Update `permissions_test.go` and `registry_test.go` to use `Target`/`Access`

## 4. Remove Dead `.factory/` Code
- [ ] 4.1 Delete `readFinalResponseFromSession()` and its caller in `handleSessionEvents`
- [ ] 4.2 Delete `createBuildModeStateFile()` and its caller in `handleSendMessage`
- [ ] 4.3 Delete `BuildModeState` struct
- [ ] 4.4 Delete `getFirstIncompleteChange()` and its caller (build-mode auto-select path)
- [ ] 4.5 Remove stale `.factory/` comments from `handlers_session.go` and `cmd/server/main.go`

## 5. Remove Dead Session Mode System
- [ ] 5.1 Delete `transformMessageForMode()` function
- [ ] 5.2 Delete `SessionMode` type and constants (`ModeInteractive`, `ModePlan`, `ModeBuild`) from `session/types.go`
- [ ] 5.3 Remove `Mode` field from `ActiveSession`, `SendMessageParams`, `SessionParams`
- [ ] 5.4 Remove `ChangeID` and `BuildAll` fields from `SendMessageParams`, `SessionParams`
- [ ] 5.5 Remove `UseSpec` field from `SpawnParams`, `SessionParams`, `RuntimeRequest`, `StartOptions`
- [ ] 5.6 Remove build-mode auto-selection path from `handleSendMessage` (lines ~766-780)
- [ ] 5.7 Update `handlers_session_unified.go` to stop passing Mode/ChangeID/BuildAll/UseSpec
- [ ] 5.8 Update `session/streaming.go` and `session/manager.go` to stop threading UseSpec
- [ ] 5.9 Update tests (`session/types_test.go`, `testutil/fixtures.go`)

## 6. Remove Unused Audit Operations
- [ ] 6.1 Remove `OpSessionCreate`, `OpSessionEnd`, `OpSessionMessage`, `OpWorkspaceCreate`, `OpWorkspaceDelete`, `OpContainerStart`, `OpContainerStop`, `OpImageRebuild` from `audit/logger.go`

## 7. Remove Unused Metrics
- [ ] 7.1 Remove `RecordToolCall`, `SetContainersRunning`, `SetProjectsTotal`, `RecordEventDrop`
- [ ] 7.2 Remove corresponding gauge/counter variables

## 8. Remove Unused Backup Exports
- [ ] 8.1 Remove `ExportManifest()` from `backup/backup.go`
- [ ] 8.2 Remove `RestoreProject()` from `backup/backup.go`

## 9. Replace `os.IsNotExist` Anti-Pattern
- [ ] 9.1 Replace all `os.IsNotExist(err)` with `errors.Is(err, fs.ErrNotExist)` (24 locations, 9 files)
- [ ] 9.2 Add `"errors"` and `"io/fs"` imports where needed

## 10. Fix Stale Comments
- [ ] 10.1 Update `cmd/oubliette-relay/main.go` — replace "droid" with "agent" in protocol comments
- [ ] 10.2 Remove credential_refs TODO from `handlers_container.go:357`

## 11. Verification
- [ ] 11.1 `go build ./...`
- [ ] 11.2 `go test ./... -short`
- [ ] 11.3 `golangci-lint run --enable gocritic ./cmd/... ./internal/...`
- [ ] 11.4 `gofmt -l .` returns nothing
