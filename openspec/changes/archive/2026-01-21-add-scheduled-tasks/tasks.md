# Tasks: Add Scheduled Tasks

## Phase 1: Core Types and Store

- [x] 1.1 Create `internal/schedule/types.go` with `Schedule`, `ScheduleTarget` structs
- [x] 1.2 Create `internal/schedule/store.go` with SQLite persistence (following auth/store.go pattern):
  - Constructor opens/creates `data/schedules.db`
  - `migrate()` creates tables and indexes
  - `Create(schedule)` - insert with transaction
  - `Get(id)` - load single schedule with targets
  - `List(filter)` - list with optional project filter
  - `Update(id, updates)` - partial updates
  - `Delete(id)` - remove schedule (CASCADE deletes targets)
  - `ListDue(now)` - enabled schedules where next_run_at <= now
  - `UpdateRunTimes(id, lastRun, nextRun)` - update timestamps
  - `Close()` - close database connection

## Phase 2: Cron Parsing

- [x] 2.1 Add `github.com/robfig/cron/v3` dependency
- [x] 2.2 Create `internal/schedule/cron.go`:
  - `ParseCron(expr)` - validate and parse cron expression
  - `NextRun(expr, after)` - calculate next run time
- [x] 2.3 Add validation in store Create/Update for cron expressions

## Phase 3: Schedule Runner

- [x] 3.1 Create `internal/schedule/runner.go` with `ScheduleRunner`:
  - Background goroutine with 1-minute ticker
  - Checks enabled schedules against current time
  - Tracks running executions per schedule
- [x] 3.2 Implement overlap behavior (skip, queue, parallel):
  - `skip` - don't start if previous still running
  - `queue` - for MVP, log warning and skip (full queue later)
  - `parallel` - allow concurrent execution
- [x] 3.3 Implement target execution:
  - Call `session_message` logic internally for each target
  - Handle session_behavior (resume vs new)
  - Log execution results

## Phase 4: MCP Tools

- [x] 4.1 Create `internal/mcp/handlers_schedule.go`
- [x] 4.2 Implement `schedule_create`:
  - Validate cron expression
  - Validate token can access all targets
  - Store creator token ID and scope
  - Register tool with Target=Global, Access=Write
- [x] 4.3 Implement `schedule_list`:
  - Filter by token scope (admin sees all, project scope sees own)
  - Optional project_id filter
  - Register tool with Target=Global, Access=Read
- [x] 4.4 Implement `schedule_get`:
  - Validate token can access schedule
  - Register tool with Target=Global, Access=Read
- [x] 4.5 Implement `schedule_update`:
  - Partial updates via pointer fields
  - Recalculate next_run_at if cron changes
  - Register tool with Target=Global, Access=Write
- [x] 4.6 Implement `schedule_delete`:
  - Validate token can access schedule
  - Register tool with Target=Global, Access=Write
- [x] 4.7 Implement `schedule_trigger`:
  - Execute immediately regardless of enabled status
  - Return session IDs for each target
  - Register tool with Target=Global, Access=Write

## Phase 5: Server Integration

- [x] 5.1 Add `ScheduleManager` and `ScheduleRunner` to `Server` struct
- [x] 5.2 Initialize schedule components in `NewServer()`
- [x] 5.3 Start `ScheduleRunner` goroutine on server start
- [x] 5.4 Graceful shutdown: stop runner, wait for in-flight executions

## Phase 6: Testing

- [x] 6.1 Add unit tests for cron parsing in `internal/schedule/cron_test.go`
- [x] 6.2 Add unit tests for store CRUD in `internal/schedule/store_test.go`
- [x] 6.3 Add integration tests in `test/pkg/suites/schedule.go`:
  - `test_schedule_create_and_list`
  - `test_schedule_update_enabled`
  - `test_schedule_delete`
  - `test_schedule_trigger_manual`
  - `test_schedule_project_scope_restriction`

## Phase 7: Documentation

- [x] 7.1 Update `AGENTS.md` with schedule tools documentation
- [x] 7.2 Update `README.md` with scheduled tasks section
- [x] 7.3 Add example schedules in documentation
