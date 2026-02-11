# Change: Tech Debt Sweep

## Why

After the Droid runtime removal and streaming pipeline rewrite, the codebase has accumulated dead code, stale paths, deprecated patterns, and orphaned systems that no longer serve a purpose. Per the rip-and-replace philosophy, these need to go.

## What Changes

### 1. Dead Files

| File | Reason |
|------|--------|
| `internal/config/config.go` | Empty file, only deprecation comment |
| `internal/container/types.go` | Empty file, only comment |
| `internal/container/Dockerfile` | Stale 159-line Dockerfile superseded by `containers/base/Dockerfile`. References Factory Droid CLI. Nothing builds from it. |
| `containers/osint/` | Stale container type, never referenced at runtime |

### 2. Remove Legacy Token Scope (`ScopeReadOnly`)

`ScopeReadOnly = "read-only"` is deprecated in favor of `ScopeAdminRO`. Still referenced in ~18 locations across auth types, MCP handlers, permissions, and tests. Remove the constant and all fallback paths that check for it.

### 3. Remove Legacy Tool Scope System

All registered tools now use `Target`/`Access` for permissions. The old `Scope` field on `ToolDef`, the `toolScopeAdmin`/`toolScopeWrite`/`toolScopeRead` constants, `isToolAllowedForTokenScope()`, and the fallback path in `IsToolAllowed()` are dead code. ~25 references across registry, permissions, tools, and tests.

### 4. Remove Dead `.factory/` Code Paths

The `.factory/` directory was the Droid runtime config location, now fully removed from container setup. These stale paths remain:

- **`readFinalResponseFromSession()`** — reads from `.factory/sessions/` (Droid JSONL path). OpenCode doesn't use this. Dead function. Remove + its caller in `handleSessionEvents`.
- **`createBuildModeStateFile()`** — writes `.factory/build-mode.json` for stop hooks. Stop hooks were deleted. Nothing reads this file. Dead function. Remove + its caller in `handleSendMessage`.
- **`BuildModeState` struct** — only used by `createBuildModeStateFile`. Dead type. Remove.
- **`getFirstIncompleteChange()`** — execs `openspec list --json` to auto-select changes for build mode. Only called from the build-mode path in `handleSendMessage`. Dead with build mode removal.
- Stale `.factory/` comments in `handlers_session.go` (lines 745, 1086-1090) and `cmd/server/main.go` (line 848).

### 5. Remove Dead Session Mode System

`transformMessageForMode()` prepends `/openspec-proposal` or `/openspec-apply` to messages. These were Droid slash-commands that OpenCode doesn't support. The entire mode system is dead:

- **`transformMessageForMode()`** — prepends Droid slash-commands. Dead function.
- **`SessionMode` type** and constants (`ModeInteractive`, `ModePlan`, `ModeBuild`) in `session/types.go`. Dead type.
- **`Mode` field** on `ActiveSession`, `SendMessageParams`, `SessionParams` — unused by OpenCode runtime.
- **`ChangeID`/`BuildAll` fields** on `SendMessageParams`, `SessionParams` — only meaningful for build mode. Dead.
- **`UseSpec` field** on `SpawnParams`, `SessionParams`, `RuntimeRequest`, `StartOptions` — threaded through types but never read by OpenCode runtime. Dead.
- The entire build-mode auto-selection path in `handleSendMessage` (lines 766-780).

### 6. Remove Unused Audit Operations

Only `OpProjectCreate`, `OpProjectDelete`, `OpTokenCreate`, `OpTokenRevoke` are called. Remove: `OpSessionCreate`, `OpSessionEnd`, `OpSessionMessage`, `OpWorkspaceCreate`, `OpWorkspaceDelete`, `OpContainerStart`, `OpContainerStop`, `OpImageRebuild`.

### 7. Remove Unused Metrics Functions

`RecordToolCall`, `SetContainersRunning`, `SetProjectsTotal`, `RecordEventDrop` are defined but never called. Remove along with their gauge/counter variables.

### 8. Remove Unused Backup Exports

`ExportManifest()` and `RestoreProject()` are defined but never called from any handler or CLI command. Dead exports.

### 9. Replace `os.IsNotExist` Anti-Pattern

24 locations across 9 files use `os.IsNotExist(err)`. The project's anti-pattern list calls this out — should use `errors.Is(err, fs.ErrNotExist)`.

### 10. Clean Up `osint` References

Comments and test fixtures still reference `osint` as a container type:
- `handlers_project.go:49` — comment lists "base, dev, osint"
- `agent/config/types.go:35` — comment lists "base, dev, osint"
- `config/loader_test.go` — 4 test values use "osint"

### 11. Fix Stale Comments

- `cmd/oubliette-relay/main.go` lines 2, 5, 73 — references "droid" in protocol comments. Should say "agent" or "OpenCode".
- `handlers_container.go:357` — TODO for project-specific credential refs (Droid-era concept). Remove.

## Impact

- No external API changes (Mode/ChangeID/BuildAll/UseSpec are optional JSON fields that become no-ops then get removed)
- Affected packages: `auth`, `mcp`, `audit`, `metrics`, `config`, `container`, `session`, `project`, `backup`, `cleanup`, `schedule`
- Net reduction: ~400-500 lines
