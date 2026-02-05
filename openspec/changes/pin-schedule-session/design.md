## Context

Scheduled tasks need persistent session affinity so users can:
1. Review task output across multiple runs
2. Follow up on scheduled task conversations
3. Track the history of a recurring task

Current behavior spawns/resumes sessions opportunistically based on what's active, making task output discoverable only through logs or multiple MCP calls.

## Goals / Non-Goals

**Goals:**
- 1:1 relationship between schedule target and session
- Automatic session resumption when closed
- Session ID visible in schedule list/get responses
- Last execution output directly in schedule responses (no extra calls needed)
- Full execution history queryable via `action: "history"`

**Non-Goals:**
- Multiple sessions per schedule target
- Session rotation or archival policies
- Automatic history pruning (can add retention policy later)

## Decisions

### Decision 1: Session ID and last output stored on ScheduleTarget, not Schedule

**Rationale:** A schedule can have multiple targets (different projects/workspaces). Each target needs its own session since sessions are project+workspace scoped. Each target also has its own execution output.

**Schema change:**
```sql
ALTER TABLE schedule_targets ADD COLUMN session_id TEXT;
ALTER TABLE schedule_targets ADD COLUMN last_executed_at DATETIME;
ALTER TABLE schedule_targets ADD COLUMN last_output TEXT;
```

### Decision 2: Resume closed sessions via ResumeBidirectionalSession

**Rationale:** The session manager already has `ResumeBidirectionalSession` which takes an existing session and resumes it with the agent runtime. This handles the "session closed but still on disk" case.

**Flow:**
```
executeScheduleTarget(schedule, target):
  if target.SessionID != "":
    # Try to use active session
    if activeSess := activeSessions.Get(target.SessionID); activeSess != nil && activeSess.IsRunning():
      output := activeSess.SendMessageAndWaitForOutput(prompt)
      store.UpdateTargetExecution(target.ID, output)
      return target.SessionID, output
    
    # Session not active - resume from disk
    if existingSession := sessionMgr.Load(target.SessionID); existingSession != nil:
      output := resumeAndSendMessage(existingSession, prompt)
      store.UpdateTargetExecution(target.ID, output)
      return target.SessionID, output
  
  # First run for this target - spawn and pin
  sess, output := spawnNewSession(...)
  store.UpdateTargetSession(target.ID, sess.SessionID, output)
  return sess.SessionID, output
```

### Decision 3: Capture output from streaming session

**Rationale:** The executor streams events. To capture the final output, we need to either:
- Wait for completion and read the last turn from the session
- Use `FinalResponseFetcher` which already exists on ActiveSession

We'll use the existing `FinalResponseFetcher` mechanism after the message completes, or read the last turn's output from the session file.

### Decision 4: session_behavior=new clears the pinned session

**Rationale:** If `session_behavior=new` is set, the user explicitly wants a fresh session each run. In this case, we should clear the pinned session ID and spawn fresh.

### Decision 5: Execution history stored in separate table

**Rationale:** A dedicated `schedule_executions` table provides:
- Clean queryable history independent of session turns
- Ability to track failed and skipped executions (not just successes)
- Separation of concerns - schedule metadata vs execution log

**Schema:**
```sql
CREATE TABLE schedule_executions (
  id TEXT PRIMARY KEY,
  schedule_id TEXT NOT NULL,
  target_id TEXT NOT NULL,
  session_id TEXT,
  executed_at DATETIME NOT NULL,
  status TEXT NOT NULL,  -- success, failed, skipped
  output TEXT,
  error TEXT,
  duration_ms INTEGER,
  FOREIGN KEY (schedule_id) REFERENCES schedules(id) ON DELETE CASCADE
);
CREATE INDEX idx_executions_schedule ON schedule_executions(schedule_id, executed_at DESC);
```

**Query via `action: "history"`:**
```json
{
  "action": "history",
  "schedule_id": "sched_123",
  "limit": 10
}
```

## Risks / Trade-offs

**Risk:** Session disk storage grows unbounded
- **Mitigation:** Existing session cleanup mechanisms apply. This change doesn't affect retention.

**Risk:** Pinned session may fail to resume (e.g., container image changed)
- **Mitigation:** If resume fails, spawn new session and update the pinned ID. Log warning.

**Risk:** Execution history table grows unbounded
- **Mitigation:** For MVP, no automatic pruning. Can add `max_history` config or `action: "prune"` later. Index on `(schedule_id, executed_at DESC)` keeps queries fast.

## Migration Plan

None. Per project philosophy: **NO BACKWARDS COMPATIBILITY**.

Schema changes are applied on server startup. The new columns and table are created fresh. Any existing schedule data from development/testing should be wiped - users reinstall from scratch.

## Open Questions

None - design is straightforward given existing session resumption infrastructure.
