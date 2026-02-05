package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ContainerParams is the unified params struct for the container tool
type ContainerParams struct {
	Action string `json:"action"` // Required: start, stop, logs, exec

	// For all actions
	ProjectID string `json:"project_id,omitempty"`

	// For exec
	Command    string `json:"command,omitempty"`
	WorkingDir string `json:"working_dir,omitempty"`
}

var containerActions = []string{"start", "stop", "logs", "exec"}

// handleContainer is the unified handler for the container tool
func (s *Server) handleContainer(ctx context.Context, request *mcp.CallToolRequest, params *ContainerParams) (*mcp.CallToolResult, any, error) {
	if params.Action == "" {
		return nil, nil, missingActionError("container", containerActions)
	}

	switch params.Action {
	case "start":
		return s.containerStart(ctx, request, params)
	case "stop":
		return s.containerStop(ctx, request, params)
	case "logs":
		return s.containerLogs(ctx, request, params)
	case "exec":
		return s.containerExec(ctx, request, params)
	default:
		return nil, nil, actionError("container", params.Action, containerActions)
	}
}

func (s *Server) containerStart(ctx context.Context, request *mcp.CallToolRequest, params *ContainerParams) (*mcp.CallToolResult, any, error) {
	return s.handleSpawnContainer(ctx, request, &SpawnContainerParams{ProjectID: params.ProjectID})
}

func (s *Server) containerStop(ctx context.Context, request *mcp.CallToolRequest, params *ContainerParams) (*mcp.CallToolResult, any, error) {
	return s.handleStopContainer(ctx, request, &StopContainerParams{ProjectID: params.ProjectID})
}

func (s *Server) containerLogs(ctx context.Context, request *mcp.CallToolRequest, params *ContainerParams) (*mcp.CallToolResult, any, error) {
	return s.handleGetLogs(ctx, request, &GetLogsParams{ProjectID: params.ProjectID})
}

func (s *Server) containerExec(ctx context.Context, request *mcp.CallToolRequest, params *ContainerParams) (*mcp.CallToolResult, any, error) {
	return s.handleExecCommand(ctx, request, &ExecCommandParams{
		ProjectID:  params.ProjectID,
		Command:    params.Command,
		WorkingDir: params.WorkingDir,
	})
}
