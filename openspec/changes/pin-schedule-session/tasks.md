## 1. Schema Update
- [x] 1.1 Add columns to `schedule_targets` table in `internal/schedule/store.go`:
  - `session_id TEXT`
  - `last_executed_at DATETIME`
  - `last_output TEXT`
- [x] 1.2 Add `schedule_executions` table in `internal/schedule/store.go`:
  - `id`, `schedule_id`, `target_id`, `session_id`
  - `executed_at`, `status`, `output`, `error`, `duration_ms`
- [x] 1.3 Add fields to `ScheduleTarget` struct in `internal/schedule/types.go`:
  - `SessionID string`
  - `LastExecutedAt *time.Time`
  - `LastOutput string`
- [x] 1.4 Add `Execution` struct in `internal/schedule/types.go`
- [x] 1.5 Update `Create`, `Get`, `List` methods to read new target columns
- [x] 1.6 Add `UpdateTargetExecution(targetID, sessionID, output string)` method
- [x] 1.7 Add `RecordExecution(execution *Execution)` method
- [x] 1.8 Add `ListExecutions(scheduleID string, limit int)` method

## 2. Execution Logic
- [x] 2.1 Update `executeScheduleTarget` in `internal/mcp/server.go` to:
  - Check for pinned session first
  - Resume closed sessions via `ResumeBidirectionalSession`
  - Store new session ID when spawning
- [x] 2.2 Capture execution output after message completes:
  - Wait for session to return to idle
  - Read last turn output from session
  - Store output via `UpdateTargetExecution`
- [x] 2.3 Record execution in history table with status/output/error
- [x] 2.4 Handle `session_behavior=new` by clearing pinned session before spawn
- [x] 2.5 Handle resume failures gracefully (spawn new, update pin, log warning)
- [x] 2.6 Record skipped executions (overlap policy) in history

## 3. API Response Updates
- [x] 3.1 Include in `schedule_get` response (per target):
  - `session_id`
  - `last_executed_at`
  - `last_output`
- [x] 3.2 Include same fields in `schedule_list` response (per target)
- [x] 3.3 Return `session_id` and `output` from `schedule_trigger` response
- [x] 3.4 Add `action: "history"` handler in `internal/mcp/handlers_schedule.go`
- [x] 3.5 Update unified handler to route `history` action

## 4. Testing
- [x] 4.1 Add integration test: schedule runs, verify session_id and last_output stored
- [x] 4.2 Add integration test: schedule runs again, verify same session_id, new output
- [x] 4.3 Add integration test: end session, schedule runs, verify session resumed
- [x] 4.4 Add integration test: session_behavior=new spawns fresh session each time
- [x] 4.5 Add integration test: schedule_get returns last_output without extra calls
- [x] 4.6 Add integration test: action=history returns past executions
- [x] 4.7 Add integration test: failed execution recorded in history with error
- [x] 4.8 Verify 100% MCP tool coverage: `cd test/cmd && go run . --coverage-report`
