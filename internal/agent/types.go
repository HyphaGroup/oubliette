// Package agent provides the agent runtime abstraction layer.
//
// types.go - Shared types for agent communication
//
// This file contains:
// - StreamEventType and StreamEvent for normalized event streaming
// - ExecuteRequest for agent execution parameters
//
// StreamEvent provides a common format that all runtime implementations
// must convert their native events into. This enables consistent
// event handling regardless of the backend.

package agent

// StreamEventType represents the type of streaming event
type StreamEventType string

const (
	StreamEventSystem     StreamEventType = "system"
	StreamEventMessage    StreamEventType = "message"
	StreamEventDelta      StreamEventType = "delta"
	StreamEventToolCall   StreamEventType = "tool_call"
	StreamEventToolResult StreamEventType = "tool_result"
	StreamEventCompletion StreamEventType = "completion"
	StreamEventError      StreamEventType = "error"
)

// StreamEvent represents a single event in agent streaming output
// This is a normalized type that works across all agent backends
type StreamEvent struct {
	Type      StreamEventType `json:"type"`
	Subtype   string          `json:"subtype,omitempty"`
	SessionID string          `json:"session_id,omitempty"`

	// Message fields
	Role string `json:"role,omitempty"`
	ID   string `json:"id,omitempty"`
	Text string `json:"text,omitempty"`

	// Tool call fields
	MessageID  string                 `json:"messageId,omitempty"`
	ToolID     string                 `json:"toolId,omitempty"`
	ToolName   string                 `json:"toolName,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`

	// Tool result fields
	IsError bool   `json:"isError,omitempty"`
	Value   string `json:"value,omitempty"`

	// Completion fields (final event)
	FinalText  string `json:"finalText,omitempty"`
	NumTurns   int    `json:"numTurns,omitempty"`
	DurationMs int    `json:"durationMs,omitempty"`

	Timestamp int64 `json:"timestamp,omitempty"`

	// Raw data for backend-specific fields
	Raw map[string]interface{} `json:"-"`
}

// ExecuteRequest contains parameters for agent execution
type ExecuteRequest struct {
	// Required
	Prompt      string
	ContainerID string
	WorkingDir  string

	// Session management
	SessionID string // Empty for new, set for continuation
	ProjectID string // Project ID for session identity
	Depth     int    // Recursion depth

	// Agent configuration
	Model          string // Model identifier
	AutonomyLevel  string // Autonomy/permission level
	ReasoningLevel string // Reasoning/thinking level

	// Tool control
	EnabledTools  []string
	DisabledTools []string

	// Mode flags
	StreamJSONRPC bool // Bidirectional streaming protocol
	UseSpec       bool // Planning mode

	// Context
	SystemPrompt string
}
