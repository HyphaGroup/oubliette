package schedule

import (
	"time"
)

// OverlapBehavior defines what to do if a previous run is still active
type OverlapBehavior string

const (
	OverlapSkip     OverlapBehavior = "skip"     // Don't start if previous still running
	OverlapQueue    OverlapBehavior = "queue"    // Queue for later (MVP: skip with warning)
	OverlapParallel OverlapBehavior = "parallel" // Allow concurrent execution
)

// SessionBehavior defines how to handle session creation
type SessionBehavior string

const (
	SessionResume SessionBehavior = "resume" // Resume existing or spawn new (default)
	SessionNew    SessionBehavior = "new"    // Always create fresh session
)

// Schedule represents a scheduled task that executes prompts on a cron schedule
type Schedule struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	CronExpr        string           `json:"cron_expr"`        // Standard 5-field cron expression
	Prompt          string           `json:"prompt"`           // Message to send to agent
	Enabled         bool             `json:"enabled"`          // Can be paused/resumed
	OverlapBehavior OverlapBehavior  `json:"overlap_behavior"` // What to do if previous run active
	SessionBehavior SessionBehavior  `json:"session_behavior"` // resume or new
	Targets         []ScheduleTarget `json:"targets"`          // Project/workspace pairs to execute on
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	LastRunAt       *time.Time       `json:"last_run_at,omitempty"`
	NextRunAt       *time.Time       `json:"next_run_at,omitempty"`
	CreatorTokenID  string           `json:"creator_token_id"` // Token that created this schedule
	CreatorScope    string           `json:"creator_scope"`    // Scope of creating token for auth
}

// ScheduleTarget represents a project/workspace pair to execute on
type ScheduleTarget struct {
	ID             string     `json:"id"`
	ScheduleID     string     `json:"schedule_id"`
	ProjectID      string     `json:"project_id"`
	WorkspaceID    string     `json:"workspace_id,omitempty"`      // Optional, uses default if empty
	SessionID      string     `json:"session_id,omitempty"`        // Pinned session for this target
	LastExecutedAt *time.Time `json:"last_executed_at,omitempty"`  // When this target last ran
	LastOutput     string     `json:"last_output,omitempty"`       // Output from last execution
}

// ExecutionStatus represents the outcome of a schedule execution
type ExecutionStatus string

const (
	ExecutionSuccess ExecutionStatus = "success"
	ExecutionFailed  ExecutionStatus = "failed"
	ExecutionSkipped ExecutionStatus = "skipped"
)

// Execution represents a single execution of a scheduled task
type Execution struct {
	ID          string          `json:"id"`
	ScheduleID  string          `json:"schedule_id"`
	TargetID    string          `json:"target_id"`
	SessionID   string          `json:"session_id,omitempty"`
	ExecutedAt  time.Time       `json:"executed_at"`
	Status      ExecutionStatus `json:"status"`
	Output      string          `json:"output,omitempty"`
	Error       string          `json:"error,omitempty"`
	DurationMs  int64           `json:"duration_ms,omitempty"`
}

// ScheduleUpdate contains optional fields for updating a schedule
type ScheduleUpdate struct {
	Name            *string          `json:"name,omitempty"`
	CronExpr        *string          `json:"cron_expr,omitempty"`
	Prompt          *string          `json:"prompt,omitempty"`
	Enabled         *bool            `json:"enabled,omitempty"`
	OverlapBehavior *OverlapBehavior `json:"overlap_behavior,omitempty"`
	SessionBehavior *SessionBehavior `json:"session_behavior,omitempty"`
	Targets         []ScheduleTarget `json:"targets,omitempty"` // If set, replaces all targets
}

// ListFilter contains optional filters for listing schedules
type ListFilter struct {
	ProjectID string // Filter to schedules targeting this project
	Enabled   *bool  // Filter by enabled status
}

// IsValidOverlapBehavior checks if the overlap behavior is valid
func IsValidOverlapBehavior(b OverlapBehavior) bool {
	return b == OverlapSkip || b == OverlapQueue || b == OverlapParallel
}

// IsValidSessionBehavior checks if the session behavior is valid
func IsValidSessionBehavior(b SessionBehavior) bool {
	return b == SessionResume || b == SessionNew
}
