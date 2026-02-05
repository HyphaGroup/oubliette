# Design: Scheduled Tasks

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      MCP Server                              │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                  Schedule Tools                          ││
│  │  schedule_create, schedule_list, schedule_update, etc.  ││
│  └────────────────────────┬────────────────────────────────┘│
│                           ↓                                  │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                 ScheduleManager                          ││
│  │  - CRUD operations                                       ││
│  │  - Token scope validation                                ││
│  │  - SQLite persistence (data/schedules.db)                ││
│  └────────────────────────┬────────────────────────────────┘│
│                           ↓                                  │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                 ScheduleRunner                           ││
│  │  - Background goroutine                                  ││
│  │  - Evaluates cron expressions                            ││
│  │  - Tracks running executions                             ││
│  │  - Calls session_message for each target                 ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

## Data Model

### Schedule

```go
type Schedule struct {
    ID          string           `json:"id"`           // sched_<uuid>
    Name        string           `json:"name"`
    Description string           `json:"description,omitempty"`
    Cron        string           `json:"cron"`         // 5-field cron expression
    Prompt      string           `json:"prompt"`
    Targets     []ScheduleTarget `json:"targets"`
    
    // Behavior
    Enabled         bool   `json:"enabled"`
    OverlapBehavior string `json:"overlap_behavior"` // skip, queue, parallel
    SessionBehavior string `json:"session_behavior"` // resume, new
    
    // Auth
    CreatedBy   string `json:"created_by"`   // Token ID that created this
    TokenScope  string `json:"token_scope"`  // Scope at creation time
    
    // Timestamps
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
    LastRunAt   *time.Time `json:"last_run_at,omitempty"`
    NextRunAt   *time.Time `json:"next_run_at,omitempty"`
}

type ScheduleTarget struct {
    ProjectID   string `json:"project_id"`
    WorkspaceID string `json:"workspace_id,omitempty"` // Empty = default workspace
}
```

### SQLite Storage

Database at `data/schedules.db` (following the pattern from `data/auth.db`):

```sql
CREATE TABLE IF NOT EXISTS schedules (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    cron TEXT NOT NULL,
    prompt TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    overlap_behavior TEXT NOT NULL DEFAULT 'skip',
    session_behavior TEXT NOT NULL DEFAULT 'resume',
    created_by TEXT NOT NULL,
    token_scope TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_run_at DATETIME,
    next_run_at DATETIME
);

CREATE TABLE IF NOT EXISTS schedule_targets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    schedule_id TEXT NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
    project_id TEXT NOT NULL,
    workspace_id TEXT,
    UNIQUE(schedule_id, project_id, workspace_id)
);

CREATE INDEX IF NOT EXISTS idx_schedules_enabled ON schedules(enabled);
CREATE INDEX IF NOT EXISTS idx_schedules_next_run ON schedules(next_run_at);
CREATE INDEX IF NOT EXISTS idx_schedule_targets_project ON schedule_targets(project_id);
```

Benefits over file-based:
- Atomic updates with transactions
- Efficient queries (enabled schedules, by project)
- No index/file sync issues
- Consistent with existing auth.db pattern

## Cron Parsing

Use `github.com/robfig/cron/v3` for cron expression parsing:
- Standard 5-field format: `minute hour day month weekday`
- Examples:
  - `0 9 * * *` - Daily at 9:00 AM UTC
  - `*/30 * * * *` - Every 30 minutes
  - `0 0 * * 0` - Weekly on Sunday at midnight

## Scheduler Loop

```go
func (r *ScheduleRunner) Run(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case now := <-ticker.C:
            r.checkAndExecute(now)
        }
    }
}

func (r *ScheduleRunner) checkAndExecute(now time.Time) {
    schedules, _ := r.manager.ListDue(now)
    for _, sched := range schedules {
        go r.execute(sched)
    }
}
```

## Overlap Handling

```go
func (r *ScheduleRunner) execute(sched *Schedule) {
    r.mu.Lock()
    running, exists := r.running[sched.ID]
    
    switch sched.OverlapBehavior {
    case "skip":
        if exists && running > 0 {
            r.mu.Unlock()
            logger.Info("Skipping schedule %s: previous run still active", sched.ID)
            return
        }
    case "queue":
        // Queue implementation would need a channel per schedule
        // For MVP, treat as skip with warning
        if exists && running > 0 {
            r.mu.Unlock()
            logger.Warn("Queue not implemented, skipping schedule %s", sched.ID)
            return
        }
    case "parallel":
        // Allow concurrent execution
    }
    
    r.running[sched.ID]++
    r.mu.Unlock()
    
    defer func() {
        r.mu.Lock()
        r.running[sched.ID]--
        r.mu.Unlock()
    }()
    
    r.executeTargets(sched)
}
```

## Target Execution

For each target, internally invoke the same logic as `session_message`:

```go
func (r *ScheduleRunner) executeTargets(sched *Schedule) {
    for _, target := range sched.Targets {
        logger.Info("Executing schedule %s for project %s workspace %s",
            sched.ID, target.ProjectID, target.WorkspaceID)
        
        // Use internal session_message logic
        params := &SendMessageParams{
            ProjectID:   target.ProjectID,
            WorkspaceID: target.WorkspaceID,
            Message:     sched.Prompt,
            NewSession:  sched.SessionBehavior == "new",
        }
        
        // Execute with the original creator's permissions
        result, err := r.server.sendMessageInternal(ctx, params)
        if err != nil {
            logger.Error("Schedule %s target %s/%s failed: %v",
                sched.ID, target.ProjectID, target.WorkspaceID, err)
        } else {
            logger.Info("Schedule %s target %s/%s started session %s",
                sched.ID, target.ProjectID, target.WorkspaceID, result.SessionID)
        }
    }
    
    // Update last_run_at and next_run_at
    r.manager.UpdateRunTimes(sched.ID)
}
```

## MCP Tool Parameters

### schedule_create
```go
type ScheduleCreateParams struct {
    Name            string           `json:"name"`
    Description     string           `json:"description,omitempty"`
    Cron            string           `json:"cron"`
    Prompt          string           `json:"prompt"`
    Targets         []ScheduleTarget `json:"targets"`
    OverlapBehavior string           `json:"overlap_behavior,omitempty"` // default: skip
    SessionBehavior string           `json:"session_behavior,omitempty"` // default: resume
    Enabled         bool             `json:"enabled,omitempty"`          // default: true
}
```

### schedule_update
```go
type ScheduleUpdateParams struct {
    ScheduleID      string            `json:"schedule_id"`
    Name            *string           `json:"name,omitempty"`
    Description     *string           `json:"description,omitempty"`
    Cron            *string           `json:"cron,omitempty"`
    Prompt          *string           `json:"prompt,omitempty"`
    Targets         *[]ScheduleTarget `json:"targets,omitempty"`
    OverlapBehavior *string           `json:"overlap_behavior,omitempty"`
    SessionBehavior *string           `json:"session_behavior,omitempty"`
    Enabled         *bool             `json:"enabled,omitempty"`
}
```

### schedule_list
```go
type ScheduleListParams struct {
    ProjectID string `json:"project_id,omitempty"` // Filter by project
    Enabled   *bool  `json:"enabled,omitempty"`    // Filter by enabled status
}
```

## Authorization Flow

```go
func (s *Server) handleScheduleCreate(ctx context.Context, params *ScheduleCreateParams) error {
    authCtx, err := requireAuth(ctx)
    if err != nil {
        return err
    }
    
    // Validate all targets are accessible
    for _, target := range params.Targets {
        if !authCtx.CanAccessProject(target.ProjectID) {
            return fmt.Errorf("cannot create schedule for project %s: access denied", target.ProjectID)
        }
    }
    
    // Store the creating token's scope for future access checks
    schedule := &Schedule{
        // ...
        CreatedBy:  authCtx.Token.ID,
        TokenScope: authCtx.Token.Scope,
    }
    
    return s.scheduleMgr.Create(schedule)
}
```

## Startup & Shutdown

```go
func NewServer(cfg *ServerConfig) *Server {
    s := &Server{...}
    
    // Initialize schedule manager and runner
    s.scheduleMgr = schedule.NewManager(cfg.DataDir)
    s.scheduleRunner = schedule.NewRunner(s.scheduleMgr, s)
    
    return s
}

func (s *Server) Start(ctx context.Context) error {
    // ... existing startup ...
    
    // Start scheduler in background
    go s.scheduleRunner.Run(ctx)
    
    return nil
}
```

## File Structure

```
internal/
└── schedule/
    ├── store.go        # SQLite persistence (following auth/store.go pattern)
    ├── manager.go      # ScheduleManager - CRUD with store
    ├── runner.go       # ScheduleRunner - Background execution
    ├── types.go        # Schedule, ScheduleTarget types
    └── cron.go         # Cron parsing helpers

internal/mcp/
└── handlers_schedule.go  # MCP tool handlers
```

## Implementation Notes

1. **Runner precision**: 1-minute ticker is sufficient; cron expressions don't support sub-minute
2. **Time zone**: All times in UTC; clients can convert for display
3. **Missed runs**: If server was down, don't catch up - just run at next scheduled time
4. **Logging**: Each execution logs to server log with schedule ID, target, session ID, result
5. **Foreign keys**: Enable with `PRAGMA foreign_keys = ON` on connection
