# Tasks: Tech Debt Sweep

## 1. Dead File Removal
- [ ] 1.1 Delete `internal/config/config.go`
- [ ] 1.2 Delete `internal/container/types.go`
- [ ] 1.3 Delete `internal/container/Dockerfile` (stale, superseded by `containers/base/Dockerfile`)
- [ ] 1.4 Delete `containers/osint/` directory
- [ ] 1.5 Remove `osint` from comments/annotations in `handlers_project.go`, `config/types.go`, `loader_test.go`

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
- [ ] 4.2 Remove `FinalResponse` field from `SessionEventsResult` (or repurpose to use event buffer)
- [ ] 4.3 Remove stale `.factory/` comments from `handlers_session.go` and `cmd/server/main.go`

## 5. Remove Unused Audit Operations
- [ ] 5.1 Remove `OpSessionCreate`, `OpSessionEnd`, `OpSessionMessage`, `OpWorkspaceCreate`, `OpWorkspaceDelete`, `OpContainerStart`, `OpContainerStop`, `OpImageRebuild` from `audit/logger.go`

## 6. Remove Unused Metrics
- [ ] 6.1 Remove `RecordToolCall`, `SetContainersRunning`, `SetProjectsTotal`, `RecordEventDrop`
- [ ] 6.2 Remove corresponding gauge/counter variables

## 7. Replace `os.IsNotExist` Anti-Pattern
- [ ] 7.1 Replace all `os.IsNotExist(err)` with `errors.Is(err, fs.ErrNotExist)` (24 locations, 9 files)

## 8. Remove `credential_refs` TODO
- [ ] 8.1 Remove TODO comment from `handlers_container.go:357`

## 9. Verification
- [ ] 9.1 `go build ./...`
- [ ] 9.2 `go test ./... -short`
- [ ] 9.3 `golangci-lint run --enable gocritic ./cmd/... ./internal/...`
- [ ] 9.4 `gofmt -l .` returns nothing
