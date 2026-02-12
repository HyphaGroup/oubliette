# Change: Simplify All MCP Tools

## Why

After the session tool cleanup, the same patterns of dead params, structural bloat, and inaccurate descriptions exist across every other tool. This sweep addresses all of them in one pass.

## Findings

### 1. Dual Param Type Anti-Pattern (ALL tools)

Every unified tool has **two layers of param structs** and a mapping function that copies field-by-field between them. Example for `container`:

```
ContainerParams (unified) → containerExec() → ExecCommandParams (original) → handleExecCommand()
```

The unified struct IS the params. The original per-action structs and their mapping functions are pure boilerplate. The unified handler should dispatch directly to the underlying handler using the unified params, or the underlying handlers should accept the unified struct directly.

**Files affected**: All 6 `*_unified.go` files + corresponding `handlers_*.go` files.

**Approach**: Eliminate the per-action param structs. Have the underlying handlers accept the unified params directly (or inline the logic into the unified handler methods).

### 2. Dead/Unused Fields

| Tool | Field | Issue |
|------|-------|-------|
| `project` (create) | `languages` | Accepted on API, stored in `CreateProjectRequest`, but **never read** by project manager. Not stored on `Project`. Dead. |
| `project` (create) | `init_git` | Default is `true`. The `*bool` pointer type is unnecessary ceremony -- just use `bool` with `omitempty`. |
| `project` (create) | `workspace_isolation` | Same `*bool` issue. |

### 3. Description Inaccuracies

| Tool | Issue |
|------|-------|
| `project_tasks` | Description says `change_name` but param is `change_id` |
| `schedule` | Description lists `message` param but actual field is `prompt` |
| `schedule` | Description mentions `pin_session=true` which doesn't exist (it's `session_behavior`) |
| `schedule` | `history` action exists in handler but missing from description |
| `config_limits` | Description says "max_children" but actual field is "max_agents" |

### 4. Schedule `prompt` → `message` Rename

The session tool renamed `prompt` → `message`. Schedules still use `prompt` for the same concept (task text sent to the agent). Should be `message` for consistency.

### 5. `CredentialRefs` Duplication

`CredentialRefs` is defined **three times** in three packages:
- `mcp.CredentialRefs` (handlers_project.go:19)
- `project.CredentialRefs` (project/types.go:24)
- `agentconfig.CredentialRefs` (agent/config/types.go:28)

Plus a `convertCredentialRefs` mapping function in project/manager.go. Should be one canonical type.

### 6. `CredentialRefs.Provider` Field

The `Provider` field on `CredentialRefs` is threaded through all 3 types but is never used for credential lookup -- only `GitHub` is used. The handler only calls `HasGitHubCredential()` / `GetGitHubToken()`. The `Provider` field does nothing.

Wait -- checking: the provider credential is looked up via `GetDefaultProviderCredential()` in container startup, not via `CredentialRefs.Provider`. So the `Provider` ref field is dead.

### 7. `project_changes` ActiveSessions Enrichment

`handleProjectChanges` calls `s.activeSessions.GetSessionsByChangeID()` which depends on the now-deleted `ChangeID` field on sessions (removed in tech-debt-sweep). This enrichment is dead code.

### 8. Unused `*mcp.CallToolRequest` Parameter

Every handler signature includes `request *mcp.CallToolRequest` but almost none use it. This is structural (required by the generic `Register` function signature) and can't be removed without refactoring the registry pattern. **Not worth changing** -- it's a common Go pattern for handler signatures.

## What Changes

### Phase 1: Fix descriptions (zero-risk)
Fix the 5 inaccurate descriptions listed above.

### Phase 2: Remove dead code
- Remove `Languages` field from `CreateProjectParams`, `ProjectParams`, `CreateProjectRequest`
- Remove dead `GetSessionsByChangeID` enrichment from `handleProjectChanges`
- Remove `CredentialRefs.Provider` from all 3 types + the mapping function

### Phase 3: Eliminate dual param types
For each tool, collapse the per-action param structs into the unified struct:
- `container`: Remove `SpawnContainerParams`, `ExecCommandParams`, `StopContainerParams`, `GetLogsParams`. Handlers use `ContainerParams` directly.
- `token`: Remove `TokenCreateParams`, `TokenListParams`, `TokenRevokeParams`. Handlers use `TokenParams` directly.
- `workspace`: Remove `WorkspaceListParams`, `WorkspaceDeleteParams`. Handlers use `WorkspaceParams` directly.
- `project`: Remove `CreateProjectParams`, `ListProjectsParams`, `GetProjectParams`, `DeleteProjectParams`. Handlers use `ProjectParams` directly.
- `schedule`: Remove `ScheduleCreateParams`, `ScheduleListParams`, `ScheduleGetParams`, `ScheduleUpdateParams`, `ScheduleDeleteParams`, `ScheduleTriggerParams`, `ScheduleHistoryParams`. Handlers use `ScheduleParams` directly.
- `session`: Remove `SpawnParams`, `SendMessageParams`, `GetSessionParams`, `ListSessionsParams`, `EndSessionParams`, `SessionEventsParams`, `SessionCleanupParams`. Handlers use `SessionParams` directly.

After this, the `*_unified.go` mapping functions become the handlers themselves, and the `*_unified.go` files can be merged into the main handler files.

### Phase 4: Rename schedule `prompt` → `message`
Rename `Prompt` to `Message` on `ScheduleCreateParams`, `ScheduleUpdateParams`, `ScheduleParams`, and the `Schedule` domain type. Update all references.

### Phase 5: Consolidate `CredentialRefs`
Keep one canonical type in `project/types.go`. Remove the duplicates in `mcp` and `agentconfig` packages. Update imports.

## Impact

- **Breaking**: Schedule API `prompt` → `message` (no external callers yet)
- **Net reduction**: ~600-800 lines of mapping boilerplate
- **Better accuracy**: All 11 tool descriptions corrected
- **Simpler codebase**: One param type per tool instead of two
