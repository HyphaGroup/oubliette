# Spec: Session Cleanup

## ADDED Requirements

### Requirement: Session cleanup by age

The system MUST provide a way to delete old session metadata files based on age.

#### Scenario: Clean up old sessions for a project
- **GIVEN** a project with session files of various ages
- **WHEN** `CleanupOldSessions` is called with `maxAge = 24h`
- **THEN** sessions older than 24 hours are deleted
- **AND** sessions newer than 24 hours are preserved
- **AND** the count of deleted sessions is returned

#### Scenario: Preserve active sessions
- **GIVEN** a project with an active session older than maxAge
- **WHEN** `CleanupOldSessions` is called
- **THEN** the active session is NOT deleted (status != completed/failed/timed_out)

#### Scenario: Clean up sessions across all projects
- **GIVEN** multiple projects with old sessions
- **WHEN** `CleanupAllOldSessions` is called with `maxAge = 7d`
- **THEN** old sessions are deleted from all projects
- **AND** a map of project_id -> deleted_count is returned

### Requirement: MCP tool for session cleanup

The system MUST expose session cleanup via MCP for manual invocation.

#### Scenario: Clean up single project via MCP
- **GIVEN** an authenticated user with write access
- **WHEN** `session_cleanup` is called with `project_id` and `max_age_hours: 24`
- **THEN** old sessions for that project are deleted
- **AND** the response includes the count of deleted sessions

#### Scenario: Clean up all projects via MCP
- **GIVEN** an authenticated admin user
- **WHEN** `session_cleanup` is called without `project_id`
- **THEN** old sessions for all projects are deleted
- **AND** the response includes counts per project

#### Scenario: Reject cleanup without write access
- **GIVEN** a user with read-only access
- **WHEN** `session_cleanup` is called
- **THEN** the request is rejected with "read-only access" error
