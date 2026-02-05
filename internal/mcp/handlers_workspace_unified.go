package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// WorkspaceParams is the unified params struct for the workspace tool
type WorkspaceParams struct {
	Action string `json:"action"` // Required: list, delete

	// For both actions
	ProjectID   string `json:"project_id,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
}

var workspaceActions = []string{"list", "delete"}

// handleWorkspace is the unified handler for the workspace tool
func (s *Server) handleWorkspace(ctx context.Context, request *mcp.CallToolRequest, params *WorkspaceParams) (*mcp.CallToolResult, any, error) {
	if params.Action == "" {
		return nil, nil, missingActionError("workspace", workspaceActions)
	}

	switch params.Action {
	case "list":
		return s.workspaceList(ctx, request, params)
	case "delete":
		return s.workspaceDelete(ctx, request, params)
	default:
		return nil, nil, actionError("workspace", params.Action, workspaceActions)
	}
}

func (s *Server) workspaceList(ctx context.Context, request *mcp.CallToolRequest, params *WorkspaceParams) (*mcp.CallToolResult, any, error) {
	return s.handleWorkspaceList(ctx, request, &WorkspaceListParams{ProjectID: params.ProjectID})
}

func (s *Server) workspaceDelete(ctx context.Context, request *mcp.CallToolRequest, params *WorkspaceParams) (*mcp.CallToolResult, any, error) {
	return s.handleWorkspaceDelete(ctx, request, &WorkspaceDeleteParams{
		ProjectID:   params.ProjectID,
		WorkspaceID: params.WorkspaceID,
	})
}
