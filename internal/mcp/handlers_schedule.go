package mcp

import (
	"context"
	"fmt"

	"github.com/HyphaGroup/oubliette/internal/auth"
	"github.com/HyphaGroup/oubliette/internal/schedule"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ScheduleParams is the params struct for the schedule tool
type ScheduleParams struct {
	Action string `json:"action"` // Required: create, list, get, update, delete, trigger, history

	Name            string                    `json:"name,omitempty"`
	CronExpr        string                    `json:"cron_expr,omitempty"`
	Prompt          string                    `json:"prompt,omitempty"`
	Targets         []ScheduleTargetParams    `json:"targets,omitempty"`
	Enabled         *bool                     `json:"enabled,omitempty"`
	OverlapBehavior *schedule.OverlapBehavior `json:"overlap_behavior,omitempty"`
	SessionBehavior *schedule.SessionBehavior `json:"session_behavior,omitempty"`
	ScheduleID      string                    `json:"schedule_id,omitempty"`
	ProjectID       string                    `json:"project_id,omitempty"`
	Limit           int                       `json:"limit,omitempty"`
}

type ScheduleTargetParams struct {
	ProjectID   string `json:"project_id"`
	WorkspaceID string `json:"workspace_id,omitempty"`
}

var scheduleActions = []string{"create", "list", "get", "update", "delete", "trigger", "history"}

func (s *Server) handleSchedule(ctx context.Context, request *mcp.CallToolRequest, params *ScheduleParams) (*mcp.CallToolResult, any, error) {
	if params.Action == "" {
		return nil, nil, missingActionError("schedule", scheduleActions)
	}

	switch params.Action {
	case "create":
		return s.handleScheduleCreate(ctx, request, params)
	case "list":
		return s.handleScheduleList(ctx, request, params)
	case "get":
		return s.handleScheduleGet(ctx, request, params)
	case "update":
		return s.handleScheduleUpdate(ctx, request, params)
	case "delete":
		return s.handleScheduleDelete(ctx, request, params)
	case "trigger":
		return s.handleScheduleTrigger(ctx, request, params)
	case "history":
		return s.handleScheduleHistory(ctx, request, params)
	default:
		return nil, nil, actionError("schedule", params.Action, scheduleActions)
	}
}

func (s *Server) handleScheduleCreate(ctx context.Context, request *mcp.CallToolRequest, params *ScheduleParams) (*mcp.CallToolResult, any, error) {
	authCtx, err := requireAuth(ctx)
	if err != nil {
		return nil, nil, err
	}

	if params.Name == "" {
		return nil, nil, fmt.Errorf("name is required")
	}
	if params.CronExpr == "" {
		return nil, nil, fmt.Errorf("cron_expr is required")
	}
	if params.Prompt == "" {
		return nil, nil, fmt.Errorf("prompt is required")
	}
	if len(params.Targets) == 0 {
		return nil, nil, fmt.Errorf("at least one target is required")
	}

	for _, target := range params.Targets {
		if !authCtx.CanAccessProject(target.ProjectID) {
			return nil, nil, fmt.Errorf("access denied to project %s", target.ProjectID)
		}
	}

	sched := &schedule.Schedule{
		Name:            params.Name,
		CronExpr:        params.CronExpr,
		Prompt:          params.Prompt,
		Enabled:         true,
		OverlapBehavior: schedule.OverlapSkip,
		SessionBehavior: schedule.SessionResume,
		CreatorTokenID:  authCtx.Token.ID,
		CreatorScope:    authCtx.Token.Scope,
	}

	if params.Enabled != nil {
		sched.Enabled = *params.Enabled
	}
	if params.OverlapBehavior != nil {
		if !schedule.IsValidOverlapBehavior(*params.OverlapBehavior) {
			return nil, nil, fmt.Errorf("invalid overlap_behavior: %s", *params.OverlapBehavior)
		}
		sched.OverlapBehavior = *params.OverlapBehavior
	}
	if params.SessionBehavior != nil {
		if !schedule.IsValidSessionBehavior(*params.SessionBehavior) {
			return nil, nil, fmt.Errorf("invalid session_behavior: %s", *params.SessionBehavior)
		}
		sched.SessionBehavior = *params.SessionBehavior
	}

	for _, t := range params.Targets {
		sched.Targets = append(sched.Targets, schedule.ScheduleTarget{
			ProjectID:   t.ProjectID,
			WorkspaceID: t.WorkspaceID,
		})
	}

	if err := s.scheduleStore.Create(sched); err != nil {
		return nil, nil, fmt.Errorf("failed to create schedule: %w", err)
	}

	result := "✅ Schedule created successfully!\n\n"
	result += fmt.Sprintf("ID:       %s\n", sched.ID)
	result += fmt.Sprintf("Name:     %s\n", sched.Name)
	result += fmt.Sprintf("Cron:     %s\n", sched.CronExpr)
	result += fmt.Sprintf("Targets:  %d project(s)\n", len(sched.Targets))
	result += fmt.Sprintf("Enabled:  %v\n", sched.Enabled)
	if sched.NextRunAt != nil {
		result += fmt.Sprintf("Next Run: %s\n", sched.NextRunAt.Format("2006-01-02 15:04:05"))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, sched, nil
}

func (s *Server) handleScheduleList(ctx context.Context, request *mcp.CallToolRequest, params *ScheduleParams) (*mcp.CallToolResult, any, error) {
	authCtx, err := requireAuth(ctx)
	if err != nil {
		return nil, nil, err
	}

	filter := &schedule.ListFilter{}
	if params.ProjectID != "" {
		if !authCtx.CanAccessProject(params.ProjectID) {
			return nil, nil, fmt.Errorf("access denied to project %s", params.ProjectID)
		}
		filter.ProjectID = params.ProjectID
	}

	schedules, err := s.scheduleStore.List(filter)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list schedules: %w", err)
	}

	if !auth.IsAdminScope(authCtx.Token.Scope) {
		projectID := auth.ExtractProjectID(authCtx.Token.Scope)
		var filtered []*schedule.Schedule
		for _, sched := range schedules {
			for _, target := range sched.Targets {
				if target.ProjectID == projectID {
					filtered = append(filtered, sched)
					break
				}
			}
		}
		schedules = filtered
	}

	if len(schedules) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "No schedules found."}},
		}, nil, nil
	}

	result := fmt.Sprintf("Found %d schedule(s):\n\n", len(schedules))
	for _, sched := range schedules {
		status := "enabled"
		if !sched.Enabled {
			status = "disabled"
		}
		result += fmt.Sprintf("• %s (%s)\n", sched.Name, sched.ID)
		result += fmt.Sprintf("  Cron:     %s\n", sched.CronExpr)
		result += fmt.Sprintf("  Status:   %s\n", status)
		result += fmt.Sprintf("  Targets:  %d project(s)\n", len(sched.Targets))
		if sched.NextRunAt != nil {
			result += fmt.Sprintf("  Next Run: %s\n", sched.NextRunAt.Format("2006-01-02 15:04"))
		}
		result += "\n"
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, schedules, nil
}

func (s *Server) handleScheduleGet(ctx context.Context, request *mcp.CallToolRequest, params *ScheduleParams) (*mcp.CallToolResult, any, error) {
	authCtx, err := requireAuth(ctx)
	if err != nil {
		return nil, nil, err
	}

	if params.ScheduleID == "" {
		return nil, nil, fmt.Errorf("schedule_id is required")
	}

	sched, err := s.scheduleStore.Get(params.ScheduleID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	if err := requireScheduleAccess(authCtx, sched); err != nil {
		return nil, nil, err
	}

	status := "enabled"
	if !sched.Enabled {
		status = "disabled"
	}

	result := fmt.Sprintf("Schedule: %s\n\n", sched.Name)
	result += fmt.Sprintf("ID:              %s\n", sched.ID)
	result += fmt.Sprintf("Cron:            %s\n", sched.CronExpr)
	result += fmt.Sprintf("Status:          %s\n", status)
	result += fmt.Sprintf("Overlap:         %s\n", sched.OverlapBehavior)
	result += fmt.Sprintf("Session:         %s\n", sched.SessionBehavior)
	result += fmt.Sprintf("Created:         %s\n", sched.CreatedAt.Format("2006-01-02 15:04"))
	if sched.LastRunAt != nil {
		result += fmt.Sprintf("Last Run:        %s\n", sched.LastRunAt.Format("2006-01-02 15:04"))
	}
	if sched.NextRunAt != nil {
		result += fmt.Sprintf("Next Run:        %s\n", sched.NextRunAt.Format("2006-01-02 15:04"))
	}
	result += fmt.Sprintf("\nPrompt:\n%s\n", sched.Prompt)
	result += fmt.Sprintf("\nTargets (%d):\n", len(sched.Targets))
	for _, t := range sched.Targets {
		if t.WorkspaceID != "" {
			result += fmt.Sprintf("  • %s (workspace: %s)\n", t.ProjectID, t.WorkspaceID)
		} else {
			result += fmt.Sprintf("  • %s (default workspace)\n", t.ProjectID)
		}
		if t.SessionID != "" {
			result += fmt.Sprintf("    Session: %s\n", t.SessionID)
		}
		if t.LastExecutedAt != nil {
			result += fmt.Sprintf("    Last Run: %s\n", t.LastExecutedAt.Format("2006-01-02 15:04:05"))
		}
		if t.LastOutput != "" {
			output := t.LastOutput
			if len(output) > 200 {
				output = output[:200] + "..."
			}
			result += fmt.Sprintf("    Output: %s\n", output)
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, sched, nil
}

func (s *Server) handleScheduleUpdate(ctx context.Context, request *mcp.CallToolRequest, params *ScheduleParams) (*mcp.CallToolResult, any, error) {
	authCtx, err := requireAuth(ctx)
	if err != nil {
		return nil, nil, err
	}

	if params.ScheduleID == "" {
		return nil, nil, fmt.Errorf("schedule_id is required")
	}

	sched, err := s.scheduleStore.Get(params.ScheduleID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	if err := requireScheduleAccess(authCtx, sched); err != nil {
		return nil, nil, err
	}

	// Convert non-empty strings to pointers for partial update
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

	update := &schedule.ScheduleUpdate{
		Name:            name,
		CronExpr:        cronExpr,
		Prompt:          prompt,
		Enabled:         params.Enabled,
		OverlapBehavior: params.OverlapBehavior,
		SessionBehavior: params.SessionBehavior,
	}

	if params.OverlapBehavior != nil && !schedule.IsValidOverlapBehavior(*params.OverlapBehavior) {
		return nil, nil, fmt.Errorf("invalid overlap_behavior: %s", *params.OverlapBehavior)
	}
	if params.SessionBehavior != nil && !schedule.IsValidSessionBehavior(*params.SessionBehavior) {
		return nil, nil, fmt.Errorf("invalid session_behavior: %s", *params.SessionBehavior)
	}

	if len(params.Targets) > 0 {
		for _, target := range params.Targets {
			if !authCtx.CanAccessProject(target.ProjectID) {
				return nil, nil, fmt.Errorf("access denied to project %s", target.ProjectID)
			}
		}
		for _, t := range params.Targets {
			update.Targets = append(update.Targets, schedule.ScheduleTarget{
				ProjectID:   t.ProjectID,
				WorkspaceID: t.WorkspaceID,
			})
		}
	}

	if err := s.scheduleStore.Update(params.ScheduleID, update); err != nil {
		return nil, nil, fmt.Errorf("failed to update schedule: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("✅ Schedule %s updated successfully.", params.ScheduleID)}},
	}, nil, nil
}

func (s *Server) handleScheduleDelete(ctx context.Context, request *mcp.CallToolRequest, params *ScheduleParams) (*mcp.CallToolResult, any, error) {
	authCtx, err := requireAuth(ctx)
	if err != nil {
		return nil, nil, err
	}

	if params.ScheduleID == "" {
		return nil, nil, fmt.Errorf("schedule_id is required")
	}

	sched, err := s.scheduleStore.Get(params.ScheduleID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	if err := requireScheduleAccess(authCtx, sched); err != nil {
		return nil, nil, err
	}

	if err := s.scheduleStore.Delete(params.ScheduleID); err != nil {
		return nil, nil, fmt.Errorf("failed to delete schedule: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("✅ Schedule %s deleted successfully.", params.ScheduleID)}},
	}, nil, nil
}

func (s *Server) handleScheduleTrigger(ctx context.Context, request *mcp.CallToolRequest, params *ScheduleParams) (*mcp.CallToolResult, any, error) {
	authCtx, err := requireAuth(ctx)
	if err != nil {
		return nil, nil, err
	}

	if params.ScheduleID == "" {
		return nil, nil, fmt.Errorf("schedule_id is required")
	}

	sched, err := s.scheduleStore.Get(params.ScheduleID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	if err := requireScheduleAccess(authCtx, sched); err != nil {
		return nil, nil, err
	}

	if s.scheduleRunner == nil {
		return nil, nil, fmt.Errorf("schedule runner not initialized")
	}

	sessionIDs, err := s.scheduleRunner.TriggerNow(sched)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to trigger schedule: %w", err)
	}

	result := fmt.Sprintf("✅ Schedule %s triggered successfully!\n\n", sched.Name)
	result += fmt.Sprintf("Sessions created: %d\n", len(sessionIDs))
	for i, id := range sessionIDs {
		result += fmt.Sprintf("  %d. %s\n", i+1, id)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, sessionIDs, nil
}

func (s *Server) handleScheduleHistory(ctx context.Context, request *mcp.CallToolRequest, params *ScheduleParams) (*mcp.CallToolResult, any, error) {
	authCtx, err := requireAuth(ctx)
	if err != nil {
		return nil, nil, err
	}

	if params.ScheduleID == "" {
		return nil, nil, fmt.Errorf("schedule_id is required")
	}

	sched, err := s.scheduleStore.Get(params.ScheduleID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	if err := requireScheduleAccess(authCtx, sched); err != nil {
		return nil, nil, err
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}

	executions, err := s.scheduleStore.ListExecutions(params.ScheduleID, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list executions: %w", err)
	}

	if len(executions) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("No execution history found for schedule %s.", params.ScheduleID)}},
		}, nil, nil
	}

	result := fmt.Sprintf("Execution history for %s (%d executions):\n\n", sched.Name, len(executions))
	for _, exec := range executions {
		result += fmt.Sprintf("• %s [%s]\n", exec.ExecutedAt.Format("2006-01-02 15:04:05"), exec.Status)
		if exec.SessionID != "" {
			result += fmt.Sprintf("  Session: %s\n", exec.SessionID)
		}
		if exec.DurationMs > 0 {
			result += fmt.Sprintf("  Duration: %dms\n", exec.DurationMs)
		}
		if exec.Error != "" {
			result += fmt.Sprintf("  Error: %s\n", exec.Error)
		}
		if exec.Output != "" {
			output := exec.Output
			if len(output) > 200 {
				output = output[:200] + "..."
			}
			result += fmt.Sprintf("  Output: %s\n", output)
		}
		result += "\n"
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, executions, nil
}

// requireScheduleAccess checks if the auth context has access to a schedule
func requireScheduleAccess(authCtx *auth.AuthContext, sched *schedule.Schedule) error {
	if auth.IsAdminScope(authCtx.Token.Scope) {
		return nil
	}
	projectID := auth.ExtractProjectID(authCtx.Token.Scope)
	for _, target := range sched.Targets {
		if target.ProjectID == projectID {
			return nil
		}
	}
	return fmt.Errorf("access denied to schedule %s", sched.ID)
}
