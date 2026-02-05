# Tasks: Consolidate MCP Tools

## 1. Create Action Dispatch Infrastructure

- [x] 1.1 Create action validation helper in `internal/mcp/actions.go`:
  ```go
  func validateAction(action string, valid []string) error
  func actionError(tool, action string, valid []string) error
  ```

- [x] 1.2 Update `Registry` to support action-based tools (if needed)

## 2. Consolidate Project Tools

- [x] 2.1 Create unified `ProjectParams` struct with action field
- [x] 2.2 Create `handleProject()` dispatcher
- [x] 2.3 Rename existing handlers to private: `projectCreate()`, `projectList()`, `projectGet()`, `projectDelete()`
- [x] 2.4 Update `tools_registry.go` - remove 4 registrations, add 1 `project` tool
- [x] 2.5 Update integration tests in `test/pkg/suites/project_test.go`

## 3. Consolidate Container Tools

- [x] 3.1 Create unified `ContainerParams` struct with action field
- [x] 3.2 Create `handleContainer()` dispatcher
- [x] 3.3 Rename existing handlers to private: `containerStart()`, `containerStop()`, `containerLogs()`, `containerExec()`
- [x] 3.4 Update `tools_registry.go` - remove 4 registrations, add 1 `container` tool
- [x] 3.5 Update integration tests in `test/pkg/suites/container_test.go`

## 4. Consolidate Session Tools

- [x] 4.1 Create unified `SessionParams` struct with action field
- [x] 4.2 Create `handleSession()` dispatcher
- [x] 4.3 Rename existing handlers to private: `sessionSpawn()`, `sessionMessage()`, `sessionGet()`, `sessionList()`, `sessionEnd()`, `sessionEvents()`, `sessionCleanup()`
- [x] 4.4 Update `tools_registry.go` - remove 7 registrations, add 1 `session` tool
- [x] 4.5 Move `caller_tool_response` handler - stays separate
- [x] 4.6 Update integration tests in `test/pkg/suites/session_test.go`

## 5. Consolidate Workspace Tools

- [x] 5.1 Create unified `WorkspaceParams` struct with action field
- [x] 5.2 Create `handleWorkspace()` dispatcher
- [x] 5.3 Rename existing handlers to private: `workspaceList()`, `workspaceDelete()`
- [x] 5.4 Update `tools_registry.go` - remove 2 registrations, add 1 `workspace` tool
- [x] 5.5 Update integration tests in `test/pkg/suites/workspace_test.go`

## 6. Consolidate Token Tools

- [x] 6.1 Create unified `TokenParams` struct with action field
- [x] 6.2 Create `handleToken()` dispatcher
- [x] 6.3 Rename existing handlers to private: `tokenCreate()`, `tokenList()`, `tokenRevoke()`
- [x] 6.4 Update `tools_registry.go` - remove 3 registrations, add 1 `token` tool
- [x] 6.5 Update integration tests in `test/pkg/suites/token_test.go`

## 7. Consolidate Schedule Tools

- [x] 7.1 Create unified `ScheduleParams` struct with action field
- [x] 7.2 Create `handleSchedule()` dispatcher
- [x] 7.3 Rename existing handlers to private: `scheduleCreate()`, `scheduleList()`, `scheduleGet()`, `scheduleUpdate()`, `scheduleDelete()`, `scheduleTrigger()`
- [x] 7.4 Update `tools_registry.go` - remove 6 registrations, add 1 `schedule` tool
- [x] 7.5 Update integration tests in `test/pkg/suites/schedule_test.go`

## 8. Update Standalone Tools

- [x] 8.1 Verify `project_options` unchanged
- [x] 8.2 Verify `project_changes` unchanged
- [x] 8.3 Verify `project_tasks` unchanged
- [x] 8.4 Verify `image_rebuild` unchanged
- [x] 8.5 Verify `caller_tool_response` unchanged
- [x] 8.6 Verify `config_limits` unchanged

## 9. Clean Up Dead Code

- [x] 9.1 Remove old handler function signatures (now private)
- [x] 9.2 Remove old tool registrations from `tools_registry.go`
- [x] 9.3 Search for any remaining references to old tool names
- [x] 9.4 Run `./build.sh` to verify no compile errors

## 10. Testing and Verification

- [x] 10.1 Run integration tests: `cd test/cmd && go run . --test`
- [x] 10.2 Verify 100% MCP tool coverage: `cd test/cmd && go run . --coverage-report`
- [x] 10.3 Verify tool count is now 12: `grep -c 'Name:' internal/mcp/tools_registry.go`
- [x] 10.4 Manual smoke test with mcp-cli

## 11. Documentation

- [x] 11.1 Update `docs/MCP_TOOLS.md` with new tool structure
- [x] 11.2 Update examples in documentation
