package session

import (
	"time"
)

// Status represents the state of a session
type Status string

const (
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

// Session represents an AI agent session (gogol)
type Session struct {
	SessionID        string    `json:"session_id"`
	ProjectID        string    `json:"project_id"`
	WorkspaceID      string    `json:"workspace_id"` // UUID of workspace used
	ContainerID      string    `json:"container_id"`
	Status           Status    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	RuntimeSessionID string    `json:"runtime_session_id"` // For session continuation
	Model            string    `json:"model"`              // claude-opus-4-5-20251101, gpt-5.1, etc.
	AutonomyLevel    string    `json:"autonomy_level"`     // low, medium, high
	ReasoningLevel   string    `json:"reasoning_level"`    // off, low, medium, high
	Turns            []Turn    `json:"turns"`
	TotalCost        Cost      `json:"total_cost"`
	// Recursion hierarchy fields
	ParentSessionID *string                `json:"parent_session_id,omitempty"`
	ChildSessions   []string               `json:"child_sessions,omitempty"`
	Depth           int                    `json:"depth"`
	ExplorationID   string                 `json:"exploration_id,omitempty"`
	TaskContext     map[string]interface{} `json:"task_context,omitempty"`
	ToolsAllowed    []string               `json:"tools_allowed,omitempty"`
}

// Turn represents a single interaction in a session
type Turn struct {
	TurnNumber  int        `json:"turn_number"`
	Prompt      string     `json:"prompt"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt time.Time  `json:"completed_at"`
	Output      TurnOutput `json:"output"`
	Cost        Cost       `json:"cost"`
}

// TurnOutput contains the result of a turn
type TurnOutput struct {
	Text          string `json:"text"`
	ExitCode      int    `json:"exit_code"`
	Error         string `json:"error,omitempty"`
	StreamingFile string `json:"streaming_file,omitempty"` // Path to streaming output file
	// Note: files_modified and commands_run removed (not provided by runtime)
}

// Cost represents API usage cost
type Cost struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// SessionSummary is a lightweight view of a session
type SessionSummary struct {
	SessionID   string    `json:"session_id"`
	ProjectID   string    `json:"project_id"`
	WorkspaceID string    `json:"workspace_id"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	TurnCount   int       `json:"turn_count"`
	LastPrompt  string    `json:"last_prompt,omitempty"`
}

// ToSummary converts a Session to SessionSummary
func (s *Session) ToSummary() *SessionSummary {
	summary := &SessionSummary{
		SessionID:   s.SessionID,
		ProjectID:   s.ProjectID,
		WorkspaceID: s.WorkspaceID,
		Status:      s.Status,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
		TurnCount:   len(s.Turns),
	}
	if len(s.Turns) > 0 {
		summary.LastPrompt = s.Turns[len(s.Turns)-1].Prompt
	}
	return summary
}

// StartOptions contains options for starting a new session
type StartOptions struct {
	Model          string // Model to use (e.g., claude-opus-4-5-20251101)
	AutonomyLevel  string // low, medium, high, skip-permissions-unsafe
	ReasoningLevel string // off, low, medium, high
	WorkspaceID    string // Workspace identifier (default, experiment-001, etc.)

	ToolsAllowed       []string // Whitelist of allowed tools
	ToolsDisallowed    []string // Blacklist of disallowed tools
	WorkspaceIsolation bool     // When true, workingDir is /workspace/<uuid> instead of /workspace/workspaces/<uuid>

	RuntimeOverride interface{} // agent.Runtime - use this runtime instead of manager's default (interface to avoid circular import)
}
