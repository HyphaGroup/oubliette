package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ProjectParams is the unified params struct for the project tool
type ProjectParams struct {
	Action string `json:"action"` // Required: create, list, get, delete

	// For create
	Name                string          `json:"name,omitempty"`
	Description         string          `json:"description,omitempty"`
	GitHubToken         string          `json:"github_token,omitempty"`
	CredentialRefs      *CredentialRefs `json:"credential_refs,omitempty"`
	RemoteURL           string          `json:"remote_url,omitempty"`
	InitGit             *bool           `json:"init_git,omitempty"`
	Languages           []string        `json:"languages,omitempty"`
	WorkspaceIsolation  *bool           `json:"workspace_isolation,omitempty"`
	ProtectedPaths      []string        `json:"protected_paths,omitempty"`
	MaxRecursionDepth   *int            `json:"max_recursion_depth,omitempty"`
	MaxAgentsPerSession *int            `json:"max_agents_per_session,omitempty"`
	MaxCostUSD          *float64        `json:"max_cost_usd,omitempty"`
	Model               string          `json:"model,omitempty"`
	Autonomy            string          `json:"autonomy,omitempty"`
	Reasoning           string          `json:"reasoning,omitempty"`
	DisabledTools       []string        `json:"disabled_tools,omitempty"`
	MCPServers          map[string]any  `json:"mcp_servers,omitempty"`
	Permissions         map[string]any  `json:"permissions,omitempty"`
	ContainerType       string          `json:"container_type,omitempty"`

	// For list
	NameContains *string `json:"name_contains,omitempty"`
	Limit        *int    `json:"limit,omitempty"`

	// For get, delete
	ProjectID string `json:"project_id,omitempty"`
}

var projectActions = []string{"create", "list", "get", "delete", "options"}

// handleProject is the unified handler for the project tool
func (s *Server) handleProject(ctx context.Context, request *mcp.CallToolRequest, params *ProjectParams) (*mcp.CallToolResult, any, error) {
	if params.Action == "" {
		return nil, nil, missingActionError("project", projectActions)
	}

	switch params.Action {
	case "create":
		return s.projectCreate(ctx, request, params)
	case "list":
		return s.projectList(ctx, request, params)
	case "get":
		return s.projectGet(ctx, request, params)
	case "delete":
		return s.projectDelete(ctx, request, params)
	case "options":
		return s.handleProjectOptions(ctx, request, &ProjectOptionsParams{})
	default:
		return nil, nil, actionError("project", params.Action, projectActions)
	}
}

// projectCreate handles the create action
func (s *Server) projectCreate(ctx context.Context, request *mcp.CallToolRequest, params *ProjectParams) (*mcp.CallToolResult, any, error) {
	// Convert to original params and delegate
	createParams := &CreateProjectParams{
		Name:                params.Name,
		Description:         params.Description,
		GitHubToken:         params.GitHubToken,
		CredentialRefs:      params.CredentialRefs,
		RemoteURL:           params.RemoteURL,
		InitGit:             params.InitGit,
		Languages:           params.Languages,
		WorkspaceIsolation:  params.WorkspaceIsolation,
		ProtectedPaths:      params.ProtectedPaths,
		MaxRecursionDepth:   params.MaxRecursionDepth,
		MaxAgentsPerSession: params.MaxAgentsPerSession,
		MaxCostUSD:          params.MaxCostUSD,
		Model:               params.Model,
		Autonomy:            params.Autonomy,
		Reasoning:           params.Reasoning,
		DisabledTools:       params.DisabledTools,
		MCPServers:          params.MCPServers,
		Permissions:         params.Permissions,
		ContainerType:       params.ContainerType,
	}
	return s.handleCreateProject(ctx, request, createParams)
}

// projectList handles the list action
func (s *Server) projectList(ctx context.Context, request *mcp.CallToolRequest, params *ProjectParams) (*mcp.CallToolResult, any, error) {
	listParams := &ListProjectsParams{
		NameContains: params.NameContains,
		Limit:        params.Limit,
	}
	return s.handleListProjects(ctx, request, listParams)
}

// projectGet handles the get action
func (s *Server) projectGet(ctx context.Context, request *mcp.CallToolRequest, params *ProjectParams) (*mcp.CallToolResult, any, error) {
	getParams := &GetProjectParams{
		ProjectID: params.ProjectID,
	}
	return s.handleGetProject(ctx, request, getParams)
}

// projectDelete handles the delete action
func (s *Server) projectDelete(ctx context.Context, request *mcp.CallToolRequest, params *ProjectParams) (*mcp.CallToolResult, any, error) {
	deleteParams := &DeleteProjectParams{
		ProjectID: params.ProjectID,
	}
	return s.handleDeleteProject(ctx, request, deleteParams)
}
