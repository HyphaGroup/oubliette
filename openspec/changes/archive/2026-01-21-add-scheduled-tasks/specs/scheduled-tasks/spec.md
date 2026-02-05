# Scheduled Tasks

Cron-based scheduling for recurring agent task execution across projects and workspaces.

## ADDED Requirements

### Requirement: Schedule Creation

The system SHALL allow creating scheduled tasks with cron expressions, prompts, and target project/workspaces.

#### Scenario: Create schedule with single target
- Given: An admin token
- When: Calling `schedule_create` with name "Daily Review", cron "0 9 * * *", prompt "Review code changes", and target project "proj-123"
- Then: A schedule is created with ID prefix "sched_"
- And: The schedule is enabled by default
- And: next_run_at is calculated based on cron expression

#### Scenario: Create schedule with multiple targets
- Given: An admin token
- When: Calling `schedule_create` with targets for project "proj-123" and project "proj-456" workspace "ws-789"
- Then: The schedule stores both targets
- And: Execution triggers for all targets on each run

#### Scenario: Project-scoped token can only target own project
- Given: A token with scope "project:proj-123"
- When: Calling `schedule_create` with target project "proj-456"
- Then: The request is denied with "access denied"

### Requirement: Schedule Listing

The system SHALL list schedules filtered by token scope and optional project filter.

#### Scenario: Admin lists all schedules
- Given: An admin token
- When: Calling `schedule_list` with no filters
- Then: All schedules are returned

#### Scenario: Project-scoped token lists schedules
- Given: A token with scope "project:proj-123"
- When: Calling `schedule_list` with no filters
- Then: Only schedules targeting project "proj-123" are returned

#### Scenario: Filter by project
- Given: An admin token
- When: Calling `schedule_list` with project_id "proj-123"
- Then: Only schedules with at least one target for "proj-123" are returned

### Requirement: Schedule Updates

The system SHALL allow updating schedule properties including enabled status, cron, prompt, and targets.

#### Scenario: Pause a schedule
- Given: An enabled schedule "sched-123"
- When: Calling `schedule_update` with enabled=false
- Then: The schedule is paused
- And: It will not trigger until re-enabled

#### Scenario: Update cron expression
- Given: A schedule with cron "0 9 * * *"
- When: Calling `schedule_update` with cron "0 */4 * * *"
- Then: The cron expression is updated
- And: next_run_at is recalculated

#### Scenario: Update targets
- Given: A schedule targeting project "proj-123"
- When: Calling `schedule_update` with targets for "proj-456"
- Then: The targets are replaced (not merged)

### Requirement: Schedule Deletion

The system SHALL allow deleting schedules with proper authorization.

#### Scenario: Delete own schedule
- Given: A schedule created by token "tok-123"
- When: The same token calls `schedule_delete`
- Then: The schedule is removed

#### Scenario: Admin can delete any schedule
- Given: A schedule created by token "tok-123"
- When: An admin token calls `schedule_delete`
- Then: The schedule is removed

### Requirement: Manual Trigger

The system SHALL allow manually triggering a scheduled task immediately.

#### Scenario: Trigger schedule manually
- Given: An enabled schedule "sched-123"
- When: Calling `schedule_trigger` with schedule_id "sched-123"
- Then: The schedule executes immediately for all targets
- And: last_run_at is updated
- And: Session IDs are returned for each target

#### Scenario: Trigger disabled schedule
- Given: A disabled schedule "sched-123"
- When: Calling `schedule_trigger` with schedule_id "sched-123"
- Then: The schedule executes (manual trigger ignores enabled status)

### Requirement: Cron-Based Execution

The system SHALL evaluate cron expressions and trigger schedules at the appropriate times.

#### Scenario: Schedule triggers on time
- Given: A schedule with cron "0 9 * * *" (daily at 9am UTC)
- When: The time reaches 09:00 UTC
- Then: The schedule executes for all targets
- And: last_run_at is set to the execution time
- And: next_run_at is set to the next 09:00 UTC

#### Scenario: Disabled schedule does not trigger
- Given: A disabled schedule
- When: The scheduled time arrives
- Then: The schedule does not execute

### Requirement: Overlap Behavior

The system SHALL handle concurrent execution based on the schedule's overlap_behavior setting.

#### Scenario: Skip overlapping execution
- Given: A schedule with overlap_behavior "skip"
- And: The previous execution is still running
- When: The next scheduled time arrives
- Then: The execution is skipped
- And: A log entry is emitted

#### Scenario: Parallel overlapping execution
- Given: A schedule with overlap_behavior "parallel"
- And: The previous execution is still running
- When: The next scheduled time arrives
- Then: A new execution starts in parallel

### Requirement: Session Behavior

The system SHALL control session creation based on the schedule's session_behavior setting.

#### Scenario: Resume existing session
- Given: A schedule with session_behavior "resume" (default)
- And: An active session exists for the target workspace
- When: The schedule triggers
- Then: The existing session receives the prompt

#### Scenario: Always new session
- Given: A schedule with session_behavior "new"
- And: An active session exists for the target workspace
- When: The schedule triggers
- Then: A new session is spawned

### Requirement: Persistence

The system SHALL persist schedules to disk and reload them on server restart.

#### Scenario: Schedules survive restart
- Given: A schedule "sched-123" exists
- When: The server restarts
- Then: The schedule is loaded from disk
- And: Execution resumes based on cron expression

### Requirement: Logging

The system SHALL emit log entries for schedule execution events.

#### Scenario: Log successful execution
- Given: A schedule triggers successfully
- Then: A log entry is emitted with schedule ID, target, and session ID

#### Scenario: Log failed execution
- Given: A schedule trigger fails (e.g., project not found)
- Then: A log entry is emitted with schedule ID, target, and error message
