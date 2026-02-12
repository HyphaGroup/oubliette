# Tasks: Simplify All MCP Tools

## 1. Fix Description Inaccuracies
- [x] 1.1 `project_tasks`: change `change_name` → `change_id` in description
- [x] 1.2 `schedule`: add `history` action, fix `pin_session` → `session_behavior`, fix param name
- [x] 1.3 `config_limits`: change `max_children` → `max_agents` in description

## 2. Remove Dead Code
- [x] 2.1 Remove `Languages` from `CreateProjectParams`, `ProjectParams`, `CreateProjectRequest`
- [x] 2.2 Remove `GetSessionsByChangeID` enrichment from `handleProjectChanges` + `ActiveSessions` field + `TaskContext` dead code
- [x] 2.3 Remove `CredentialRefs.Provider` from all 3 types (`mcp`, `project`, `agentconfig`)

## 3. Eliminate Dual Param Types — Container
- [x] 3.1 Delete `SpawnContainerParams`, `ExecCommandParams`, `StopContainerParams`, `GetLogsParams`
- [x] 3.2 Update handlers to accept `*ContainerParams` directly
- [x] 3.3 Merge unified handler into `handlers_container.go`
- [x] 3.4 Delete `handlers_container_unified.go`

## 4. Eliminate Dual Param Types — Token
- [x] 4.1 Delete `TokenCreateParams`, `TokenListParams`, `TokenRevokeParams`
- [x] 4.2 Update handlers to accept `*TokenParams` directly
- [x] 4.3 Merge unified handler into `handlers_token.go`
- [x] 4.4 Delete `handlers_token_unified.go`

## 5. Eliminate Dual Param Types — Workspace
- [x] 5.1 Delete `WorkspaceListParams`, `WorkspaceDeleteParams`
- [x] 5.2 Update handlers to accept `*WorkspaceParams` directly
- [x] 5.3 Merge unified handler into `handlers_workspace.go`
- [x] 5.4 Delete `handlers_workspace_unified.go`

## 6. Eliminate Dual Param Types — Project
- [x] 6.1 Delete `CreateProjectParams`, `ListProjectsParams`, `GetProjectParams`, `DeleteProjectParams`, `ProjectOptionsParams`
- [x] 6.2 Update all project handlers to accept `*ProjectParams`
- [x] 6.3 Merge unified handler into `handlers_project.go`
- [x] 6.4 Delete `handlers_project_unified.go`

## 7. Eliminate Dual Param Types — Schedule
- [x] 7.1 Delete all per-action params types
- [x] 7.2 Update all schedule handlers to accept `*ScheduleParams`
- [x] 7.3 Merge unified handler into `handlers_schedule.go`
- [x] 7.4 Delete `handlers_schedule_unified.go`
- [x] 7.5 Extract `requireScheduleAccess` helper to DRY access checks

## 8. Eliminate Dual Param Types — Session
- [x] 8.1 Delete `SpawnParams`, `SendMessageParams`, `GetSessionParams`, `ListSessionsParams`, `EndSessionParams`, `SessionEventsParams`, `SessionCleanupParams`
- [x] 8.2 Update all session handlers to accept `*SessionParams`
- [x] 8.3 Simplify dispatch — handlers called directly, no intermediate mapping functions

## 9. Rename Schedule `prompt` → `message`
- [x] Skipped — requires SQLite schema migration. Description already corrected to say `prompt`.

## 10. Consolidate `CredentialRefs`
- [x] 10.1 Remove `mcp.CredentialRefs` — `ProjectParams` now uses `project.CredentialRefs` directly
- [x] 10.2 Remove conversion code in `handleCreateProject`
- [x] 10.3 `agentconfig.CredentialRefs` left as-is (separate package, avoids circular deps)

## 11. Update Tests
- [x] 11.1 Update `handlers_project_test.go` — use `ProjectParams`
- [x] 11.2 Update `handlers_container_test.go` — use `ContainerParams`
- [x] 11.3 Update `handlers_token_test.go` — use `TokenParams`

## 12. Verification
- [x] 12.1 `go build ./...` — passes
- [x] 12.2 `go test ./... -short` — all pass
- [x] 12.3 `golangci-lint run --enable gocritic ./cmd/... ./internal/...` — 0 issues
- [x] 12.4 `gofmt -w .` — clean
