package mcp

import (
	"context"

	"github.com/HyphaGroup/oubliette/internal/session"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// SessionParams is the params struct for the session tool
type SessionParams struct {
	Action string `json:"action"` // Required: spawn, message, get, list, end, events, cleanup

	// Common
	ProjectID   string `json:"project_id,omitempty"`
	SessionID   string `json:"session_id,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	Message     string `json:"message,omitempty"`

	// For spawn and message
	CreateWorkspace bool           `json:"create_workspace,omitempty"`
	NewSession      bool           `json:"new_session,omitempty"`
	ExternalID      string         `json:"external_id,omitempty"`
	Source          string         `json:"source,omitempty"`
	Context         map[string]any `json:"context,omitempty"`
	Model           string         `json:"model,omitempty"`
	AutonomyLevel   string         `json:"autonomy_level,omitempty"`
	ReasoningLevel  string         `json:"reasoning_level,omitempty"`
	ToolsAllowed    []string       `json:"tools_allowed,omitempty"`
	ToolsDisallowed []string       `json:"tools_disallowed,omitempty"`

	// For message only
	Attachments []Attachment                   `json:"attachments,omitempty"`
	CallerTools []session.CallerToolDefinition `json:"caller_tools,omitempty"`
	CallerID    string                         `json:"caller_id,omitempty"`

	// For list
	Status string `json:"status,omitempty"`

	// For events
	SinceIndex      *int `json:"since_index,omitempty"`
	MaxEvents       *int `json:"max_events,omitempty"`
	IncludeChildren bool `json:"include_children,omitempty"`

	// For cleanup
	MaxAgeHours *int `json:"max_age_hours,omitempty"`
}

var sessionActions = []string{"spawn", "message", "get", "list", "end", "events", "cleanup"}

func (s *Server) handleSession(ctx context.Context, request *mcp.CallToolRequest, params *SessionParams) (*mcp.CallToolResult, any, error) {
	if params.Action == "" {
		return nil, nil, missingActionError("session", sessionActions)
	}

	switch params.Action {
	case "spawn":
		return s.handleSpawn(ctx, request, params)
	case "message":
		return s.handleSendMessage(ctx, request, params)
	case "get":
		return s.handleGetSession(ctx, request, params)
	case "list":
		return s.handleListSessions(ctx, request, params)
	case "end":
		return s.handleEndSession(ctx, request, params)
	case "events":
		return s.handleSessionEvents(ctx, request, params)
	case "cleanup":
		return s.handleSessionCleanup(ctx, request, params)
	default:
		return nil, nil, actionError("session", params.Action, sessionActions)
	}
}
