package mcp

import (
	"context"

	"github.com/HyphaGroup/oubliette/internal/schedule"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ScheduleParams is the unified params struct for the schedule tool
type ScheduleParams struct {
	Action string `json:"action"` // Required: create, list, get, update, delete, trigger, history

	// For create/update
	Name            string                    `json:"name,omitempty"`
	CronExpr        string                    `json:"cron_expr,omitempty"`
	Prompt          string                    `json:"prompt,omitempty"`
	Targets         []ScheduleTargetParams    `json:"targets,omitempty"`
	Enabled         *bool                     `json:"enabled,omitempty"`
	OverlapBehavior *schedule.OverlapBehavior `json:"overlap_behavior,omitempty"`
	SessionBehavior *schedule.SessionBehavior `json:"session_behavior,omitempty"`

	// For get, update, delete, trigger, history
	ScheduleID string `json:"schedule_id,omitempty"`

	// For list
	ProjectID string `json:"project_id,omitempty"`

	// For history
	Limit int `json:"limit,omitempty"`
}

var scheduleActions = []string{"create", "list", "get", "update", "delete", "trigger", "history"}

// handleSchedule is the unified handler for the schedule tool
func (s *Server) handleSchedule(ctx context.Context, request *mcp.CallToolRequest, params *ScheduleParams) (*mcp.CallToolResult, any, error) {
	if params.Action == "" {
		return nil, nil, missingActionError("schedule", scheduleActions)
	}

	switch params.Action {
	case "create":
		return s.scheduleCreate(ctx, request, params)
	case "list":
		return s.scheduleList(ctx, request, params)
	case "get":
		return s.scheduleGet(ctx, request, params)
	case "update":
		return s.scheduleUpdate(ctx, request, params)
	case "delete":
		return s.scheduleDelete(ctx, request, params)
	case "trigger":
		return s.scheduleTrigger(ctx, request, params)
	case "history":
		return s.scheduleHistory(ctx, request, params)
	default:
		return nil, nil, actionError("schedule", params.Action, scheduleActions)
	}
}

func (s *Server) scheduleCreate(ctx context.Context, request *mcp.CallToolRequest, params *ScheduleParams) (*mcp.CallToolResult, any, error) {
	return s.handleScheduleCreate(ctx, request, &ScheduleCreateParams{
		Name:            params.Name,
		CronExpr:        params.CronExpr,
		Prompt:          params.Prompt,
		Targets:         params.Targets,
		Enabled:         params.Enabled,
		OverlapBehavior: params.OverlapBehavior,
		SessionBehavior: params.SessionBehavior,
	})
}

func (s *Server) scheduleList(ctx context.Context, request *mcp.CallToolRequest, params *ScheduleParams) (*mcp.CallToolResult, any, error) {
	return s.handleScheduleList(ctx, request, &ScheduleListParams{ProjectID: params.ProjectID})
}

func (s *Server) scheduleGet(ctx context.Context, request *mcp.CallToolRequest, params *ScheduleParams) (*mcp.CallToolResult, any, error) {
	return s.handleScheduleGet(ctx, request, &ScheduleGetParams{ScheduleID: params.ScheduleID})
}

func (s *Server) scheduleUpdate(ctx context.Context, request *mcp.CallToolRequest, params *ScheduleParams) (*mcp.CallToolResult, any, error) {
	// Convert string fields to pointers for update
	var name, cronExpr, prompt *string
	if params.Name != "" {
		name = &params.Name
	}
	if params.CronExpr != "" {
		cronExpr = &params.CronExpr
	}
	if params.Prompt != "" {
		prompt = &params.Prompt
	}

	return s.handleScheduleUpdate(ctx, request, &ScheduleUpdateParams{
		ScheduleID:      params.ScheduleID,
		Name:            name,
		CronExpr:        cronExpr,
		Prompt:          prompt,
		Targets:         params.Targets,
		Enabled:         params.Enabled,
		OverlapBehavior: params.OverlapBehavior,
		SessionBehavior: params.SessionBehavior,
	})
}

func (s *Server) scheduleDelete(ctx context.Context, request *mcp.CallToolRequest, params *ScheduleParams) (*mcp.CallToolResult, any, error) {
	return s.handleScheduleDelete(ctx, request, &ScheduleDeleteParams{ScheduleID: params.ScheduleID})
}

func (s *Server) scheduleTrigger(ctx context.Context, request *mcp.CallToolRequest, params *ScheduleParams) (*mcp.CallToolResult, any, error) {
	return s.handleScheduleTrigger(ctx, request, &ScheduleTriggerParams{ScheduleID: params.ScheduleID})
}

func (s *Server) scheduleHistory(ctx context.Context, request *mcp.CallToolRequest, params *ScheduleParams) (*mcp.CallToolResult, any, error) {
	return s.handleScheduleHistory(ctx, request, &ScheduleHistoryParams{ScheduleID: params.ScheduleID, Limit: params.Limit})
}
