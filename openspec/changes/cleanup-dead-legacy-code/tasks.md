# Tasks: Clean Up Dead, Legacy, and Unfinished Code

## 1. Dead File Removal
- [ ] 1.1 Delete `internal/config/config.go` (empty deprecated file)
- [ ] 1.2 Delete `internal/container/types.go` (empty comment-only file)
- [ ] 1.3 Delete `containers/osint/` directory (stale container type)
- [ ] 1.4 Remove `osint` references from comments, type annotations, test fixtures (`loader_test.go`, `types.go`)

## 2. Remove Legacy Token Scope
- [ ] 2.1 Remove `ScopeReadOnly` constant from `internal/auth/types.go`
- [ ] 2.2 Remove `ScopeReadOnly` from `IsAdminScope()` and `IsReadOnlyScope()` fallback checks
- [ ] 2.3 Update `internal/mcp/handlers_token.go` to use `ScopeAdminRO` instead of `ScopeReadOnly`
- [ ] 2.4 Update `internal/mcp/registry.go` `isToolAllowedForTokenScope()` to not reference `ScopeReadOnly`
- [ ] 2.5 Update `cmd/server/main.go` to use `ScopeAdminRO` instead of `ScopeReadOnly`
- [ ] 2.6 Update all test files referencing `ScopeReadOnly` to use `ScopeAdminRO`

## 3. Remove Legacy Tool Scope System
- [ ] 3.1 Remove `Scope` field from `ToolDef` struct in `internal/mcp/registry.go`
- [ ] 3.2 Remove `toolScopeAdmin`, `toolScopeWrite`, `toolScopeRead` constants from `internal/mcp/tools.go`
- [ ] 3.3 Remove `isToolAllowedForTokenScope()` function from `internal/mcp/registry.go`
- [ ] 3.4 Remove legacy fallback path from `IsToolAllowed()` in `internal/mcp/permissions.go`
- [ ] 3.5 Update `internal/mcp/permissions_test.go` `TestIsToolAllowed_LegacyScope` — remove or rewrite using Target/Access
- [ ] 3.6 Update `internal/mcp/registry_test.go` tests that use legacy `Scope` field

## 4. Remove Unused Agent Factory Placeholder
- [ ] 4.1 Remove `RuntimeFactory` struct, `NewFactory()`, `CreateRuntime()`, `createDroidRuntime()`, `createOpenCodeRuntime()` from `internal/agent/factory.go`
- [ ] 4.2 Keep `RuntimeType` constants, `FactoryConfig` struct, and `DetectRuntimeType()` if referenced; remove if not
- [ ] 4.3 Update `internal/agent/factory_test.go` accordingly

## 5. Remove Unused Audit Operations
- [ ] 5.1 Remove `OpSessionCreate`, `OpSessionEnd`, `OpSessionMessage`, `OpWorkspaceCreate`, `OpWorkspaceDelete`, `OpContainerStart`, `OpContainerStop`, `OpImageRebuild` from `internal/audit/logger.go`

## 6. Remove Unused Metrics
- [ ] 6.1 Remove `RecordToolCall`, `SetContainersRunning`, `SetProjectsTotal`, `RecordEventDrop` functions from `internal/metrics/metrics.go`
- [ ] 6.2 Remove corresponding gauge/counter/histogram variables: `ToolCalls`, `ContainersRunning`, `ProjectsTotal`, `EventBufferDrops`

## 7. Remove Unused Backup Methods
- [ ] 7.1 Remove `ExportManifest()` from `internal/backup/backup.go`
- [ ] 7.2 Remove `RestoreProject()` from `internal/backup/backup.go`

## 8. Fix Unfinished TODOs (requires user input)
- [ ] 8.1 `StreamingExecutor.Cancel()` — decide: implement OpenCode abort or return explicit "not supported" error
- [ ] 8.2 `credential_refs.provider` TODO — decide: implement per-project provider credentials in container startup or remove TODO and document as out-of-scope

## 9. Fix Test/Build Issues
- [ ] 9.1 Remove unused `encoding/json` import from `internal/config/models_test.go`
- [ ] 9.2 Fix `TestFindConfigPath` "error when config not found" subtest (needs isolation from repo working dir)

## 10. Replace `os.IsNotExist` Anti-Pattern
- [ ] 10.1 Replace all `os.IsNotExist(err)` with `errors.Is(err, fs.ErrNotExist)` across `internal/` (20+ locations)

## 11. Verification
- [ ] 11.1 Run `./build.sh` — must succeed
- [ ] 11.2 Run `go test ./... -short` — all tests must pass
- [ ] 11.3 Run `go vet ./...` — no issues
- [ ] 11.4 Run `cd test/cmd && go run . --coverage-report` — must be 100%
