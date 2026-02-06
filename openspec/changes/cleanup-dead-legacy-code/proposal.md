# Change: Clean Up Dead, Legacy, and Unfinished Code

## Why

The codebase has accumulated dead code, deprecated constructs, unfinished placeholders, stale container artifacts, and unused metrics/audit operations. Per the project's rip-and-replace philosophy, these should be eliminated immediately rather than left to rot.

## What Changes

### Dead/Deprecated Code Removal

1. **`internal/config/config.go`** — Empty file with only a deprecation comment. No functions, no references. Delete it.

2. **`internal/container/types.go`** — Empty file with only a comment ("now config-driven"). No types, no references. Delete it.

3. **`ScopeReadOnly` ("read-only") legacy token scope** — Deprecated constant still referenced in ~20 locations across auth, MCP handlers, tests. All tools now use Target/Access model; `ScopeAdminRO` replaced it. Remove `ScopeReadOnly` and all fallback paths. Rewrite `IsAdminScope()` and `IsReadOnlyScope()` to not reference it.

4. **Legacy `Scope` field on `ToolDef`** — All registered tools now use `Target`/`Access`. The `Scope` field, `isToolAllowedForTokenScope()`, `toolScopeAdmin/toolScopeWrite/toolScopeRead` constants, and the fallback path in `IsToolAllowed()` are dead code. Remove them. Update tests that still use the legacy `Scope` field.

5. **`RuntimeFactory` placeholder in `internal/agent/factory.go`** — `CreateRuntime()`, `createDroidRuntime()`, and `createOpenCodeRuntime()` all return errors saying "use X directly". `NewFactory()` is never called from `cmd/server/main.go`. `DetectRuntimeType()` is also unused outside tests. Remove the factory struct and placeholder methods; keep `RuntimeType` constants and `FactoryConfig` if still referenced.

6. **Unused audit operations** — `OpSessionCreate`, `OpSessionEnd`, `OpSessionMessage`, `OpWorkspaceCreate`, `OpWorkspaceDelete`, `OpContainerStart`, `OpContainerStop`, `OpImageRebuild` are defined but never called. Only `OpProjectCreate`, `OpProjectDelete`, `OpTokenCreate`, `OpTokenRevoke` are used. Remove unused operations.

7. **Unused metrics functions** — `RecordToolCall`, `SetContainersRunning`, `SetProjectsTotal`, `RecordEventDrop` are defined but never called. `ToolCalls`, `ContainersRunning`, `ProjectsTotal`, `EventBufferDrops` gauges/counters are unused. Remove them.

8. **Unused backup methods** — `ExportManifest()` and `RestoreProject()` are defined but never called from any handler or CLI command. Remove them.

9. **`containers/osint/` directory** — Contains a stale Dockerfile and metadata.json for an OSINT container type. Container types are now config-driven (base/dev only). No code references `osint` at runtime, but leftover `osint` references exist in comments, test fixtures, and type annotations. Remove the directory and clean up all `osint` references.

### Unfinished Code (TODOs)

10. **`StreamingExecutor.Cancel()` TODO** — `internal/agent/opencode/executor.go:96` returns `nil` with `// TODO: Call abort endpoint`. Either implement it or document that OpenCode has no abort API and return an appropriate error.

11. **`credential_refs.provider` TODO** — `internal/mcp/handlers_container.go:361` has a TODO for project-specific credential refs. The credential_refs spec exists and the field is threaded through types, but container startup only uses the global default provider credential. Implement or explicitly document as deferred.

### Test/Build Issues

12. **Unused import in `models_test.go`** — `encoding/json` is imported but not used, causing `go test ./internal/config/` to fail.

13. **Failing `TestFindConfigPath` test** — The "error when config not found" subtest fails because `FindConfigPath` falls through to `./config/oubliette.jsonc` (which exists in the repo working directory). The test needs to either chdir or mock the fallback.

14. **`os.IsNotExist` anti-pattern** — Used in 20+ locations. The project's anti-pattern list explicitly calls out `os.IsNotExist` as deprecated; should use `errors.Is(err, fs.ErrNotExist)`.

## Impact

- Affected specs: `tools` (remove legacy Scope field from tool registration)
- Affected code: `internal/auth`, `internal/mcp`, `internal/agent`, `internal/config`, `internal/container`, `internal/metrics`, `internal/audit`, `internal/backup`, `containers/osint/`
- No breaking external API changes (all changes are internal)
