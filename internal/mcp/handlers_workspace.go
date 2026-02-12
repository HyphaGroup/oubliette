package mcp

import (
	"context"
	"fmt"

	"github.com/HyphaGroup/oubliette/internal/logger"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// WorkspaceParams is the params struct for the workspace tool
type WorkspaceParams struct {
	Action string `json:"action"` // Required: list, delete

	ProjectID   string `json:"project_id,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
}

var workspaceActions = []string{"list", "delete"}

func (s *Server) handleWorkspace(ctx context.Context, request *mcp.CallToolRequest, params *WorkspaceParams) (*mcp.CallToolResult, any, error) {
	if params.Action == "" {
		return nil, nil, missingActionError("workspace", workspaceActions)
	}

	switch params.Action {
	case "list":
		return s.handleWorkspaceList(ctx, request, params)
	case "delete":
		return s.handleWorkspaceDelete(ctx, request, params)
	default:
		return nil, nil, actionError("workspace", params.Action, workspaceActions)
	}
}

func (s *Server) handleWorkspaceList(ctx context.Context, request *mcp.CallToolRequest, params *WorkspaceParams) (*mcp.CallToolResult, any, error) {
	if params.ProjectID == "" {
		return nil, nil, fmt.Errorf("project_id is required")
	}

	if _, err := requireProjectAccess(ctx, params.ProjectID); err != nil {
		return nil, nil, err
	}

	proj, err := s.projectMgr.Get(params.ProjectID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load project: %w", err)
	}

	workspaces, err := s.projectMgr.ListWorkspaces(params.ProjectID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list workspaces: %w", err)
	}

	if len(workspaces) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "No workspaces found."},
			},
		}, nil, nil
	}

	result := fmt.Sprintf("Found %d workspace(s):\n\n", len(workspaces))
	for _, ws := range workspaces {
		isDefault := ws.ID == proj.DefaultWorkspaceID
		if isDefault {
			result += fmt.Sprintf("• %s (default)\n", ws.ID)
		} else {
			result += fmt.Sprintf("• %s\n", ws.ID)
		}
		result += fmt.Sprintf("  Created: %s\n", ws.CreatedAt.Format("2006-01-02 15:04"))
		if !ws.LastSessionAt.IsZero() {
			result += fmt.Sprintf("  Last session: %s\n", ws.LastSessionAt.Format("2006-01-02 15:04"))
		}
		if ws.ExternalID != "" {
			result += fmt.Sprintf("  External ID: %s\n", ws.ExternalID)
		}
		if ws.Source != "" {
			result += fmt.Sprintf("  Source: %s\n", ws.Source)
		}
		result += "\n"
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: result},
		},
	}, workspaces, nil
}

func (s *Server) handleWorkspaceDelete(ctx context.Context, request *mcp.CallToolRequest, params *WorkspaceParams) (*mcp.CallToolResult, any, error) {
	if params.ProjectID == "" {
		return nil, nil, fmt.Errorf("project_id is required")
	}
	if params.WorkspaceID == "" {
		return nil, nil, fmt.Errorf("workspace_id is required")
	}

	authCtx, err := requireProjectAccess(ctx, params.ProjectID)
	if err != nil {
		return nil, nil, err
	}
	if !authCtx.CanWrite() {
		return nil, nil, fmt.Errorf("read-only access, cannot delete workspaces")
	}

	if err := s.projectMgr.DeleteWorkspace(params.ProjectID, params.WorkspaceID); err != nil {
		return nil, nil, fmt.Errorf("failed to delete workspace: %w", err)
	}

	logger.Info("Workspace deleted: %s in project %s", params.WorkspaceID, params.ProjectID)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("✅ Workspace %s deleted successfully.", params.WorkspaceID)},
		},
	}, nil, nil
}
