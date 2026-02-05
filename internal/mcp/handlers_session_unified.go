package mcp

import (
	"context"

	"github.com/HyphaGroup/oubliette/internal/session"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// SessionParams is the unified params struct for the session tool
type SessionParams struct {
	Action string `json:"action"` // Required: spawn, message, get, list, end, events, cleanup

	// Common
	ProjectID   string `json:"project_id,omitempty"`
	SessionID   string `json:"session_id,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`

	// For spawn
	Prompt             string         `json:"prompt,omitempty"`
	CreateWorkspace    bool           `json:"create_workspace,omitempty"`
	NewSession         bool           `json:"new_session,omitempty"`
	ExternalID         string         `json:"external_id,omitempty"`
	Source             string         `json:"source,omitempty"`
	Context            map[string]any `json:"context,omitempty"`
	Model              string         `json:"model,omitempty"`
	AutonomyLevel      string         `json:"autonomy_level,omitempty"`
	ReasoningLevel     string         `json:"reasoning_level,omitempty"`
	ToolsAllowed       []string       `json:"tools_allowed,omitempty"`
	ToolsDisallowed    []string       `json:"tools_disallowed,omitempty"`
	AppendSystemPrompt string         `json:"append_system_prompt,omitempty"`
	UseSpec            bool           `json:"use_spec,omitempty"`

	// For message
	Message     string                         `json:"message,omitempty"`
	Attachments []Attachment                   `json:"attachments,omitempty"`
	CallerTools []session.CallerToolDefinition `json:"caller_tools,omitempty"`
	CallerID    string                         `json:"caller_id,omitempty"`
	Mode        string                         `json:"mode,omitempty"`
	ChangeID    string                         `json:"change_id,omitempty"`
	BuildAll    bool                           `json:"build_all,omitempty"`

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

// handleSession is the unified handler for the session tool
func (s *Server) handleSession(ctx context.Context, request *mcp.CallToolRequest, params *SessionParams) (*mcp.CallToolResult, any, error) {
	if params.Action == "" {
		return nil, nil, missingActionError("session", sessionActions)
	}

	switch params.Action {
	case "spawn":
		return s.sessionSpawn(ctx, request, params)
	case "message":
		return s.sessionMessage(ctx, request, params)
	case "get":
		return s.sessionGet(ctx, request, params)
	case "list":
		return s.sessionList(ctx, request, params)
	case "end":
		return s.sessionEnd(ctx, request, params)
	case "events":
		return s.sessionEvents(ctx, request, params)
	case "cleanup":
		return s.sessionCleanup(ctx, request, params)
	default:
		return nil, nil, actionError("session", params.Action, sessionActions)
	}
}

func (s *Server) sessionSpawn(ctx context.Context, request *mcp.CallToolRequest, params *SessionParams) (*mcp.CallToolResult, any, error) {
	return s.handleSpawn(ctx, request, &SpawnParams{
		ProjectID:          params.ProjectID,
		WorkspaceID:        params.WorkspaceID,
		Prompt:             params.Prompt,
		CreateWorkspace:    params.CreateWorkspace,
		NewSession:         params.NewSession,
		ExternalID:         params.ExternalID,
		Source:             params.Source,
		Context:            params.Context,
		Model:              params.Model,
		AutonomyLevel:      params.AutonomyLevel,
		ReasoningLevel:     params.ReasoningLevel,
		ToolsAllowed:       params.ToolsAllowed,
		ToolsDisallowed:    params.ToolsDisallowed,
		AppendSystemPrompt: params.AppendSystemPrompt,
		UseSpec:            params.UseSpec,
	})
}

func (s *Server) sessionMessage(ctx context.Context, request *mcp.CallToolRequest, params *SessionParams) (*mcp.CallToolResult, any, error) {
	return s.handleSendMessage(ctx, request, &SendMessageParams{
		ProjectID:          params.ProjectID,
		WorkspaceID:        params.WorkspaceID,
		Message:            params.Message,
		Attachments:        params.Attachments,
		CallerTools:        params.CallerTools,
		CallerID:           params.CallerID,
		Context:            params.Context,
		Model:              params.Model,
		AutonomyLevel:      params.AutonomyLevel,
		ReasoningLevel:     params.ReasoningLevel,
		ToolsAllowed:       params.ToolsAllowed,
		ToolsDisallowed:    params.ToolsDisallowed,
		AppendSystemPrompt: params.AppendSystemPrompt,
		CreateWorkspace:    params.CreateWorkspace,
		ExternalID:         params.ExternalID,
		Source:             params.Source,
		Mode:               params.Mode,
		ChangeID:           params.ChangeID,
		BuildAll:           params.BuildAll,
	})
}

func (s *Server) sessionGet(ctx context.Context, request *mcp.CallToolRequest, params *SessionParams) (*mcp.CallToolResult, any, error) {
	return s.handleGetSession(ctx, request, &GetSessionParams{SessionID: params.SessionID})
}

func (s *Server) sessionList(ctx context.Context, request *mcp.CallToolRequest, params *SessionParams) (*mcp.CallToolResult, any, error) {
	return s.handleListSessions(ctx, request, &ListSessionsParams{
		ProjectID: params.ProjectID,
		Status:    params.Status,
	})
}

func (s *Server) sessionEnd(ctx context.Context, request *mcp.CallToolRequest, params *SessionParams) (*mcp.CallToolResult, any, error) {
	return s.handleEndSession(ctx, request, &EndSessionParams{SessionID: params.SessionID})
}

func (s *Server) sessionEvents(ctx context.Context, request *mcp.CallToolRequest, params *SessionParams) (*mcp.CallToolResult, any, error) {
	return s.handleSessionEvents(ctx, request, &SessionEventsParams{
		SessionID:       params.SessionID,
		SinceIndex:      params.SinceIndex,
		MaxEvents:       params.MaxEvents,
		IncludeChildren: params.IncludeChildren,
	})
}

func (s *Server) sessionCleanup(ctx context.Context, request *mcp.CallToolRequest, params *SessionParams) (*mcp.CallToolResult, any, error) {
	return s.handleSessionCleanup(ctx, request, &SessionCleanupParams{
		ProjectID:   params.ProjectID,
		MaxAgeHours: params.MaxAgeHours,
	})
}
