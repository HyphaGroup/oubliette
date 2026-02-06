# Change: Pin Scheduled Tasks to Dedicated Sessions

## Why

Currently, scheduled tasks either resume an active session or spawn a new one on each run, making it difficult to track and follow up on scheduled task output. Users cannot easily discover which session a scheduled task used or review its conversation history. Getting the output requires multiple MCP tool calls. There's also no way to see execution history (past runs, failures, skipped executions).

## What Changes

- Each schedule target stores a `session_id` that persists across runs
- Each schedule target stores `last_executed_at` and `last_output` from the most recent run
- New `schedule_executions` table tracks full execution history with status and output
- When a schedule runs:
  - If the pinned session exists and is active → send message to it
  - If the pinned session exists but is closed → resume/restart that session, then send message
  - If no pinned session exists → spawn new session and store its ID
  - After execution, store the output text and timestamp on the target
  - Record execution in history table (success/failed/skipped with output or error)
- `schedule_list` and `schedule_get` return per target:
  - `session_id` - for accessing full conversation history
  - `last_executed_at` - when it last ran
  - `last_output` - the agent's response text from the last run
- New `action: "history"` returns past executions with output/errors

## Impact

- Affected specs: `scheduled-tasks`
- Affected code:
  - `internal/schedule/types.go` - Add `SessionID`, `LastExecutedAt`, `LastOutput` to `ScheduleTarget`; add `Execution` type
  - `internal/schedule/store.go` - Add `schedule_executions` table; persist and load new columns
  - `internal/mcp/server.go` - Update `executeScheduleTarget` to capture output and record executions
  - `internal/mcp/handlers_schedule.go` - Include new fields in responses; add `history` action
