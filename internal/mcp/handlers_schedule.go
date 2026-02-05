package mcp

import (
	"context"
	"fmt"

	"github.com/HyphaGroup/oubliette/internal/auth"
	"github.com/HyphaGroup/oubliette/internal/schedule"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Schedule Management Handlers

type ScheduleCreateParams struct {
	Name            string                    `json:"name"`
	CronExpr        string                    `json:"cron_expr"`
	Prompt          string                    `json:"prompt"`
	Targets         []ScheduleTargetParams    `json:"targets"`
	Enabled         *bool                     `json:"enabled,omitempty"`
	OverlapBehavior *schedule.OverlapBehavior `json:"overlap_behavior,omitempty"`
	SessionBehavior *schedule.SessionBehavior `json:"session_behavior,omitempty"`
}

type ScheduleTargetParams struct {
	ProjectID   string `json:"project_id"`
	WorkspaceID string `json:"workspace_id,omitempty"`
}

func (s *Server) handleScheduleCreate(ctx context.Context, request *mcp.CallToolRequest, params *ScheduleCreateParams) (*mcp.CallToolResult, any, error) {
	authCtx, err := requireAuth(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Validate required fields
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

	// Validate token can access all targets
	for _, target := range params.Targets {
		if !authCtx.CanAccessProject(target.ProjectID) {
			return nil, nil, fmt.Errorf("access denied to project %s", target.ProjectID)
		}
	}

	// Build schedule
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

	// Convert targets
	for _, t := range params.Targets {
		sched.Targets = append(sched.Targets, schedule.ScheduleTarget{
			ProjectID:   t.ProjectID,
			WorkspaceID: t.WorkspaceID,
		})
	}

	// Create in store
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
		Content: []mcp.Content{
			&mcp.TextContent{Text: result},
		},
	}, sched, nil
}

type ScheduleListParams struct {
	ProjectID string `json:"project_id,omitempty"`
}

func (s *Server) handleScheduleList(ctx context.Context, request *mcp.CallToolRequest, params *ScheduleListParams) (*mcp.CallToolResult, any, error) {
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

	// Filter by access if not admin
	if !auth.IsAdminScope(authCtx.Token.Scope) {
		projectID := auth.ExtractProjectID(authCtx.Token.Scope)
		var filtered []*schedule.Schedule
		for _, sched := range schedules {
			// Check if any target belongs to accessible project
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
			Content: []mcp.Content{
				&mcp.TextContent{Text: "No schedules found."},
			},
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
		Content: []mcp.Content{
			&mcp.TextContent{Text: result},
		},
	}, schedules, nil
}

type ScheduleGetParams struct {
	ScheduleID string `json:"schedule_id"`
}

func (s *Server) handleScheduleGet(ctx context.Context, request *mcp.CallToolRequest, params *ScheduleGetParams) (*mcp.CallToolResult, any, error) {
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

	// Check access
	if !auth.IsAdminScope(authCtx.Token.Scope) {
		projectID := auth.ExtractProjectID(authCtx.Token.Scope)
		hasAccess := false
		for _, target := range sched.Targets {
			if target.ProjectID == projectID {
				hasAccess = true
				break
			}
		}
		if !hasAccess {
			return nil, nil, fmt.Errorf("access denied to schedule %s", params.ScheduleID)
		}
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
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: result},
		},
	}, sched, nil
}

type ScheduleUpdateParams struct {
	ScheduleID      string                    `json:"schedule_id"`
	Name            *string                   `json:"name,omitempty"`
	CronExpr        *string                   `json:"cron_expr,omitempty"`
	Prompt          *string                   `json:"prompt,omitempty"`
	Enabled         *bool                     `json:"enabled,omitempty"`
	OverlapBehavior *schedule.OverlapBehavior `json:"overlap_behavior,omitempty"`
	SessionBehavior *schedule.SessionBehavior `json:"session_behavior,omitempty"`
	Targets         []ScheduleTargetParams    `json:"targets,omitempty"`
}

func (s *Server) handleScheduleUpdate(ctx context.Context, request *mcp.CallToolRequest, params *ScheduleUpdateParams) (*mcp.CallToolResult, any, error) {
	authCtx, err := requireAuth(ctx)
	if err != nil {
		return nil, nil, err
	}

	if params.ScheduleID == "" {
		return nil, nil, fmt.Errorf("schedule_id is required")
	}

	// Get existing schedule to check access
	sched, err := s.scheduleStore.Get(params.ScheduleID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	// Check access - must have access to existing targets
	if !auth.IsAdminScope(authCtx.Token.Scope) {
		projectID := auth.ExtractProjectID(authCtx.Token.Scope)
		hasAccess := false
		for _, target := range sched.Targets {
			if target.ProjectID == projectID {
				hasAccess = true
				break
			}
		}
		if !hasAccess {
			return nil, nil, fmt.Errorf("access denied to schedule %s", params.ScheduleID)
		}
	}

	// Build update
	update := &schedule.ScheduleUpdate{
		Name:            params.Name,
		CronExpr:        params.CronExpr,
		Prompt:          params.Prompt,
		Enabled:         params.Enabled,
		OverlapBehavior: params.OverlapBehavior,
		SessionBehavior: params.SessionBehavior,
	}

	// Validate behaviors if provided
	if params.OverlapBehavior != nil && !schedule.IsValidOverlapBehavior(*params.OverlapBehavior) {
		return nil, nil, fmt.Errorf("invalid overlap_behavior: %s", *params.OverlapBehavior)
	}
	if params.SessionBehavior != nil && !schedule.IsValidSessionBehavior(*params.SessionBehavior) {
		return nil, nil, fmt.Errorf("invalid session_behavior: %s", *params.SessionBehavior)
	}

	// Convert and validate targets if provided
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
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("✅ Schedule %s updated successfully.", params.ScheduleID)},
		},
	}, nil, nil
}

type ScheduleDeleteParams struct {
	ScheduleID string `json:"schedule_id"`
}

func (s *Server) handleScheduleDelete(ctx context.Context, request *mcp.CallToolRequest, params *ScheduleDeleteParams) (*mcp.CallToolResult, any, error) {
	authCtx, err := requireAuth(ctx)
	if err != nil {
		return nil, nil, err
	}

	if params.ScheduleID == "" {
		return nil, nil, fmt.Errorf("schedule_id is required")
	}

	// Get existing schedule to check access
	sched, err := s.scheduleStore.Get(params.ScheduleID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	// Check access
	if !auth.IsAdminScope(authCtx.Token.Scope) {
		projectID := auth.ExtractProjectID(authCtx.Token.Scope)
		hasAccess := false
		for _, target := range sched.Targets {
			if target.ProjectID == projectID {
				hasAccess = true
				break
			}
		}
		if !hasAccess {
			return nil, nil, fmt.Errorf("access denied to schedule %s", params.ScheduleID)
		}
	}

	if err := s.scheduleStore.Delete(params.ScheduleID); err != nil {
		return nil, nil, fmt.Errorf("failed to delete schedule: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("✅ Schedule %s deleted successfully.", params.ScheduleID)},
		},
	}, nil, nil
}

type ScheduleTriggerParams struct {
	ScheduleID string `json:"schedule_id"`
}

func (s *Server) handleScheduleTrigger(ctx context.Context, request *mcp.CallToolRequest, params *ScheduleTriggerParams) (*mcp.CallToolResult, any, error) {
	authCtx, err := requireAuth(ctx)
	if err != nil {
		return nil, nil, err
	}

	if params.ScheduleID == "" {
		return nil, nil, fmt.Errorf("schedule_id is required")
	}

	// Get schedule to check access and get details
	sched, err := s.scheduleStore.Get(params.ScheduleID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	// Check access
	if !auth.IsAdminScope(authCtx.Token.Scope) {
		projectID := auth.ExtractProjectID(authCtx.Token.Scope)
		hasAccess := false
		for _, target := range sched.Targets {
			if target.ProjectID == projectID {
				hasAccess = true
				break
			}
		}
		if !hasAccess {
			return nil, nil, fmt.Errorf("access denied to schedule %s", params.ScheduleID)
		}
	}

	// Trigger via runner
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
		Content: []mcp.Content{
			&mcp.TextContent{Text: result},
		},
	}, sessionIDs, nil
}
