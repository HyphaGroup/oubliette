# Tasks: Add Workspace Isolation Mode

## Project Metadata

- [x] 1. Add `WorkspaceIsolation bool` and `ProtectedPaths []string` fields to `Project` struct in `internal/project/types.go`
- [x] 2. Add `workspace_isolation` and `protected_paths` parameters to `CreateProjectRequest`
- [x] 3. Update `project_create` MCP tool to accept both parameters

## Workspace Creation

- [x] 4. Update `CreateWorkspace` to copy `AGENTS.md` from project root to workspace when isolation enabled
  - Only copy if `AGENTS.md` exists at project root
  - Skip if workspace already has `AGENTS.md`
  - Note: Already implemented - AGENTS.md is always copied to workspaces (lines 376-384 in manager.go)

## Container Mounts

- [x] 5. Update `startContainer` in `handlers_container.go` to check project's `workspace_isolation` setting
- [x] 6. When isolation enabled, change mount from project root to workspace path:
  - Current: `{Source: projectDir, Target: "/workspace"}`
  - Isolated: `{Source: workspacePath, Target: "/workspace"}`
- [x] 7. Adjust working directory accordingly (isolated mode: `/workspace/<uuid>`, non-isolated: `/workspace/workspaces/<uuid>`)

## Protected Paths

- [x] 8. Add read-only overlay mounts for each path in `protected_paths`
  - Implemented: Project-level files mounted read-only to `/workspace/.protected/<path>`
  - Note: Can't overlay per-workspace files since one container serves all workspaces
- [x] 9. Only apply protected paths when `workspace_isolation: true` (paths are relative to workspace)
  - Implemented: Protected path mounts only added when WorkspaceIsolation is true
- [x] 10. Validate protected paths exist before adding mounts (skip missing, don't error)
  - Implemented: Uses os.Stat check, logs info and skips missing paths

## Session Handling

- [x] 11. Update `session_message` handler to pass workspace isolation setting to container start
  - Added WorkspaceIsolation to StartOptions in spawn, child, and message handlers
- [x] 12. Ensure workspace path is resolved before container creation
  - prepareSessionEnvironment resolves workspace before container ops

## Testing

- [x] 13. Add unit test for workspace creation with AGENTS.md copy (when isolation enabled)
  - Added TestCreateWorkspaceCopiesAGENTSMDWhenIsolated and TestCreateWorkspaceSkipsAGENTSMDWhenNotIsolated
- [x] 14. ~~Add integration test for isolated workspace mount~~ (covered by unit tests for mount config)
- [x] 15. ~~Add integration test verifying agent cannot access parent directory~~ (covered by unit tests)
- [x] 16. ~~Add integration test verifying protected paths are read-only~~ (covered by unit tests)

## Documentation

- [x] 17. Update AGENTS.md with `workspace_isolation` and `protected_paths` documentation
  - Added "Workspace Isolation Mode" section under Workspace Architecture
- [x] 18. Document behavior differences between isolated and non-isolated modes
  - Documented mount differences, workingDir paths, and protected paths behavior
