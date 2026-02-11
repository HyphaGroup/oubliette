# Change: Tech Debt Sweep

## Why

After the Droid runtime removal and streaming pipeline rewrite, the codebase has accumulated dead code, stale paths, deprecated patterns, and orphaned systems that no longer serve a purpose. Per the rip-and-replace philosophy, these need to go.

## What Changes

### 1. Dead Files

| File | Reason |
|------|--------|
| `internal/config/config.go` | Empty file, only deprecation comment |
| `internal/container/types.go` | Empty file, only comment |
| `internal/container/Dockerfile` | Stale Dockerfile superseded by `containers/base/Dockerfile` |
| `containers/osint/` | Stale container type, never referenced at runtime |

### 2. Remove Legacy Token Scope (`ScopeReadOnly`)

`ScopeReadOnly = "read-only"` is deprecated in favor of `ScopeAdminRO`. Still referenced in ~18 locations across auth types, MCP handlers, permissions, and tests. Remove the constant and all fallback paths that check for it.

### 3. Remove Legacy Tool Scope System

All registered tools now use `Target`/`Access` for permissions. The old `Scope` field on `ToolDef`, the `toolScopeAdmin`/`toolScopeWrite`/`toolScopeRead` constants, `isToolAllowedForTokenScope()`, and the fallback path in `IsToolAllowed()` are dead code. ~25 references across registry, permissions, tools, and tests.

### 4. Remove Dead `.factory/` Code Paths

The `.factory/` directory was the Droid runtime config location, now fully removed from container setup. These stale paths remain:

- `readFinalResponseFromSession()` reads from `.factory/sessions/` — OpenCode doesn't use this path. The function is dead (only called from `handleSessionEvents` for completed sessions, where it returns empty). Remove entirely.
- Stale comments referencing `.factory/` paths in `handlers_session.go` and `cmd/server/main.go`

### 5. Remove Unused Audit Operations

Only `OpProjectCreate`, `OpProjectDelete`, `OpTokenCreate`, `OpTokenRevoke` are called. Remove: `OpSessionCreate`, `OpSessionEnd`, `OpSessionMessage`, `OpWorkspaceCreate`, `OpWorkspaceDelete`, `OpContainerStart`, `OpContainerStop`, `OpImageRebuild`.

### 6. Remove Unused Metrics Functions

`RecordToolCall`, `SetContainersRunning`, `SetProjectsTotal`, `RecordEventDrop` are defined but never called. Remove along with their gauge/counter variables.

### 7. Replace `os.IsNotExist` Anti-Pattern

24 locations across 9 files use `os.IsNotExist(err)`. The project's anti-pattern list calls this out — should use `errors.Is(err, fs.ErrNotExist)`.

### 8. Clean Up `osint` References

Comments, type annotations, and test fixtures still reference `osint` as a container type. Remove all references after deleting `containers/osint/`.

### 9. Remove `credential_refs.provider` TODO

`handlers_container.go:357` has a TODO for project-specific credential refs. This was a Droid-era concept. OpenCode resolves credentials from `oubliette.jsonc` directly. Remove the TODO comment.

## Impact

- No external API changes
- Affected packages: `auth`, `mcp`, `audit`, `metrics`, `config`, `container`, `session`, `project`, `backup`, `cleanup`, `schedule`
- Net reduction: ~200-300 lines
