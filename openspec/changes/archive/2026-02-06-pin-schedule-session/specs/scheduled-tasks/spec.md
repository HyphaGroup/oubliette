## MODIFIED Requirements

### Requirement: Session Behavior

The system SHALL control session creation based on the schedule's session_behavior setting, with each target maintaining a pinned session for continuity.

#### Scenario: Resume existing session
- Given: A schedule with session_behavior "resume" (default)
- And: An active session exists for the target workspace
- When: The schedule triggers
- Then: The existing session receives the prompt

#### Scenario: Resume closed pinned session
- Given: A schedule with session_behavior "resume"
- And: The target has a pinned session_id
- And: That session is not active (was previously ended)
- When: The schedule triggers
- Then: The pinned session is resumed
- And: The session receives the prompt

#### Scenario: First run pins session
- Given: A schedule target with no pinned session_id
- When: The schedule triggers for the first time
- Then: A new session is spawned
- And: The session_id is stored on the target
- And: Subsequent runs use this pinned session

#### Scenario: Always new session
- Given: A schedule with session_behavior "new"
- And: A pinned session exists for the target
- When: The schedule triggers
- Then: A new session is spawned
- And: The pinned session_id is updated to the new session

#### Scenario: Resume failure spawns new session
- Given: A schedule target with a pinned session_id
- And: The pinned session cannot be resumed (e.g., corrupted, incompatible)
- When: The schedule triggers
- Then: A new session is spawned
- And: The pinned session_id is updated
- And: A warning is logged

## ADDED Requirements

### Requirement: Execution Output Visibility

The system SHALL expose the last execution details for each schedule target so users can review task output without additional calls.

#### Scenario: schedule_get returns last execution
- Given: A schedule target that has executed at least once
- When: Calling schedule_get with schedule_id
- Then: Each target in the response includes:
  - `session_id` - the pinned session for full history
  - `last_executed_at` - timestamp of last run
  - `last_output` - the agent's response text from last run

#### Scenario: schedule_list returns last execution
- Given: Schedules with targets that have executed
- When: Calling schedule_list
- Then: Each schedule's targets include `session_id`, `last_executed_at`, and `last_output`

#### Scenario: schedule_trigger returns output
- Given: A schedule executes successfully
- When: Calling schedule_trigger
- Then: The response includes `session_id` and `output` for each target that executed

#### Scenario: Never-executed target has null fields
- Given: A schedule target that has never executed
- When: Calling schedule_get
- Then: The target's `session_id`, `last_executed_at`, and `last_output` are null/empty

### Requirement: Execution History

The system SHALL track execution history for each schedule, including successful runs, failures, and skipped executions.

#### Scenario: Query execution history
- Given: A schedule that has executed multiple times
- When: Calling schedule with action "history" and schedule_id
- Then: Returns a list of executions in reverse chronological order
- And: Each execution includes `executed_at`, `status`, `output` or `error`

#### Scenario: History includes failed executions
- Given: A schedule that had a failed execution (e.g., project not found)
- When: Calling schedule with action "history"
- Then: The failed execution appears in history with status "failed" and error message

#### Scenario: History includes skipped executions
- Given: A schedule with overlap_behavior "skip"
- And: An execution was skipped because previous run was still active
- When: Calling schedule with action "history"
- Then: The skipped execution appears in history with status "skipped"

#### Scenario: History respects limit parameter
- Given: A schedule with 100 executions
- When: Calling schedule with action "history" and limit 10
- Then: Only the 10 most recent executions are returned

#### Scenario: History filtered by access
- Given: A project-scoped token
- When: Calling schedule with action "history" for a schedule targeting that project
- Then: Only executions for accessible targets are returned
