# Tasks: Add Session Cleanup

## Session Manager

- [x] 1. Add `CleanupOldSessions(projectID string, maxAge time.Duration) (int, error)` method
  - Scan sessions directory for `.json` files
  - Parse session metadata to get `completed_at` or `updated_at`
  - Delete files older than `maxAge`
  - Return count of deleted sessions
  - Skip active sessions (check `status` field)

- [x] 2. Add `CleanupAllOldSessions(maxAge time.Duration) (map[string]int, error)` method
  - Iterate all projects
  - Call `CleanupOldSessions` for each
  - Return map of project_id -> deleted count

## MCP Tool

- [x] 3. Add `session_cleanup` MCP tool
  - Parameters: `project_id` (optional), `max_age_hours` (default: 24)
  - If `project_id` provided, clean that project only
  - Otherwise, clean all projects
  - Return summary of deleted sessions per project
  - Requires write access

## Optional: Auto-cleanup

- [x] 4. Add startup cleanup (optional, config-driven)
  - Fixed existing cleanup system to work with flat JSON session files
  - Existing config already supports: `cleanup.enabled`, `cleanup.retention_minutes`
  - Runs at startup and periodically (default: every 5 minutes)

## Testing

- [x] 5. Add unit tests for `CleanupOldSessions`
  - Test cleanup of old sessions
  - Test preservation of recent sessions
  - Test preservation of active sessions

- [x] 6. Add integration test for `session_cleanup` tool
  - Added test_session_cleanup to test/pkg/suites/session.go

## Documentation

- [x] 7. Update AGENTS.md with `session_cleanup` tool documentation
  - Added session_cleanup to MCP Tools for Sessions section
