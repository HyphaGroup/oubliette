package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/HyphaGroup/oubliette/internal/audit"
	"github.com/HyphaGroup/oubliette/internal/auth"
	"github.com/HyphaGroup/oubliette/internal/container"
	"github.com/HyphaGroup/oubliette/internal/logger"
	"github.com/HyphaGroup/oubliette/internal/project"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Project Management Handlers

// CredentialRefs specifies which credentials to use for a project
type CredentialRefs struct {
	GitHub   string `json:"github,omitempty"`
	Provider string `json:"provider,omitempty"`
}

type CreateProjectParams struct {
	Name               string          `json:"name"`
	Description        string          `json:"description,omitempty"`
	GitHubToken        string          `json:"github_token,omitempty"`    // Direct token (overrides credential_refs)
	CredentialRefs     *CredentialRefs `json:"credential_refs,omitempty"` // Named credential references
	RemoteURL          string          `json:"remote_url,omitempty"`
	InitGit            *bool           `json:"init_git,omitempty"`
	Languages          []string        `json:"languages,omitempty"`
	WorkspaceIsolation *bool           `json:"workspace_isolation,omitempty"`
	ProtectedPaths     []string        `json:"protected_paths,omitempty"`

	// Project limits (optional, falls back to server defaults)
	MaxRecursionDepth   *int     `json:"max_recursion_depth,omitempty"`
	MaxAgentsPerSession *int     `json:"max_agents_per_session,omitempty"`
	MaxCostUSD          *float64 `json:"max_cost_usd,omitempty"`

	// Agent configuration
	Model         string         `json:"model,omitempty"`          // Model shorthand (e.g., "sonnet") or full ID
	Autonomy      string         `json:"autonomy,omitempty"`       // off, low, medium, high
	Reasoning     string         `json:"reasoning,omitempty"`      // off, low, medium, high
	DisabledTools []string       `json:"disabled_tools,omitempty"` // Tools to disable
	MCPServers    map[string]any `json:"mcp_servers,omitempty"`    // Additional MCP servers
	Permissions   map[string]any `json:"permissions,omitempty"`    // Custom permissions (OpenCode format)

	// Container configuration
	ContainerType string `json:"container_type,omitempty"` // base, dev, osint (default: dev)
}

func (s *Server) handleCreateProject(ctx context.Context, request *mcp.CallToolRequest, params *CreateProjectParams) (*mcp.CallToolResult, any, error) {
	authCtx, err := requireWriteAccess(ctx)
	if err != nil {
		return nil, nil, err
	}

	if params.Name == "" {
		return nil, nil, fmt.Errorf("name is required")
	}

	logger.Info("Creating project: %s", params.Name)

	// Helper to get token info for audit
	tokenID, tokenScope := getTokenInfo(authCtx)

	initGit := true
	if params.InitGit != nil {
		initGit = *params.InitGit
	}

	workspaceIsolation := false
	if params.WorkspaceIsolation != nil {
		workspaceIsolation = *params.WorkspaceIsolation
	}

	// Resolve GitHub token: explicit token > credential_refs > default
	githubToken := params.GitHubToken
	if githubToken == "" && s.credentials != nil {
		if params.CredentialRefs != nil && params.CredentialRefs.GitHub != "" {
			// Validate and lookup specified credential
			if !s.credentials.HasGitHubCredential(params.CredentialRefs.GitHub) {
				return nil, nil, fmt.Errorf("unknown github credential: %s (use project_options to list available credentials)", params.CredentialRefs.GitHub)
			}
			githubToken, _ = s.credentials.GetGitHubToken(params.CredentialRefs.GitHub)
		} else {
			// Use default credential
			githubToken, _ = s.credentials.GetDefaultGitHubToken()
		}
	}

	// Validate autonomy field
	if params.Autonomy != "" {
		validAutonomy := []string{"off", "low", "medium", "high"}
		found := false
		for _, a := range validAutonomy {
			if params.Autonomy == a {
				found = true
				break
			}
		}
		if !found {
			return nil, nil, fmt.Errorf("invalid autonomy: %s (must be one of: off, low, medium, high)", params.Autonomy)
		}
	}

	// Validate reasoning field
	if params.Reasoning != "" {
		validReasoning := []string{"off", "low", "medium", "high"}
		found := false
		for _, r := range validReasoning {
			if params.Reasoning == r {
				found = true
				break
			}
		}
		if !found {
			return nil, nil, fmt.Errorf("invalid reasoning: %s (must be one of: off, low, medium, high)", params.Reasoning)
		}
	}

	// Validate container_type if provided
	if params.ContainerType != "" {
		if s.imageManager == nil {
			return nil, nil, fmt.Errorf("container types not configured")
		}
		if !s.imageManager.IsValidType(params.ContainerType) {
			return nil, nil, fmt.Errorf("invalid container_type: %s (valid types: %v)", params.ContainerType, s.imageManager.ValidTypes())
		}
	}

	// Convert MCP servers from map[string]any to map[string]AgentMCPServer
	var mcpServers map[string]project.AgentMCPServer
	if len(params.MCPServers) > 0 {
		mcpServers = make(map[string]project.AgentMCPServer)
		for name, serverData := range params.MCPServers {
			// serverData should be a map with type, command, args, url, env, etc.
			serverMap, ok := serverData.(map[string]any)
			if !ok {
				continue
			}
			server := project.AgentMCPServer{}
			if t, ok := serverMap["type"].(string); ok {
				server.Type = t
			}
			if cmd, ok := serverMap["command"].(string); ok {
				server.Command = cmd
			}
			if args, ok := serverMap["args"].([]any); ok {
				for _, a := range args {
					if s, ok := a.(string); ok {
						server.Args = append(server.Args, s)
					}
				}
			}
			if url, ok := serverMap["url"].(string); ok {
				server.URL = url
			}
			if env, ok := serverMap["env"].(map[string]any); ok {
				server.Env = make(map[string]string)
				for k, v := range env {
					if s, ok := v.(string); ok {
						server.Env[k] = s
					}
				}
			}
			mcpServers[name] = server
		}
	}

	// Convert credential refs to project type
	var credRefs *project.CredentialRefs
	if params.CredentialRefs != nil {
		credRefs = &project.CredentialRefs{
			GitHub:   params.CredentialRefs.GitHub,
			Provider: params.CredentialRefs.Provider,
		}
	}

	req := project.CreateProjectRequest{
		Name:                params.Name,
		Description:         params.Description,
		GitHubToken:         githubToken,
		RemoteURL:           params.RemoteURL,
		InitGit:             initGit,
		Languages:           params.Languages,
		WorkspaceIsolation:  workspaceIsolation,
		ProtectedPaths:      params.ProtectedPaths,
		MaxRecursionDepth:   params.MaxRecursionDepth,
		MaxAgentsPerSession: params.MaxAgentsPerSession,
		MaxCostUSD:          params.MaxCostUSD,
		Model:               params.Model,
		Autonomy:            params.Autonomy,
		Reasoning:           params.Reasoning,
		DisabledTools:       params.DisabledTools,
		MCPServers:          mcpServers,
		Permissions:         params.Permissions,
		ContainerType:       params.ContainerType,
		CredentialRefs:      credRefs,
	}

	proj, err := s.projectMgr.Create(req)
	if err != nil {
		logger.Error("Failed to create project %s: %v", params.Name, err)
		audit.LogFailure(audit.OpProjectCreate, tokenID, tokenScope, "", err)
		return nil, nil, err
	}

	audit.LogSuccess(audit.OpProjectCreate, tokenID, tokenScope, proj.ID)
	logger.Info("Project created successfully: %s (ID: %s)", proj.Name, proj.ID)

	// Auto-start container after project creation
	logger.Info("Starting container for new project: %s", proj.ID)

	containerName := fmt.Sprintf("oubliette-%s", proj.ID[:8])
	containerID, err := s.createAndStartContainer(ctx, containerName, proj.ImageName, proj.ID)
	if err != nil {
		logger.Error("Failed to start container for project %s: %v", proj.ID, err)
		result := fmt.Sprintf("✅ Project '%s' created successfully!\n\n", proj.Name)
		result += fmt.Sprintf("ID: %s\n", proj.ID)
		result += fmt.Sprintf("⚠️  Warning: Failed to start container: %v\n\n", err)
		result += fmt.Sprintf("Workspace: %s\n", s.projectMgr.GetWorkspaceDir(proj.ID))
		result += fmt.Sprintf("Image: %s\n", proj.ImageName)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, nil, nil
	}

	result := fmt.Sprintf("✅ Project '%s' created successfully!\n\n", proj.Name)
	result += fmt.Sprintf("ID: %s\n", proj.ID)
	result += fmt.Sprintf("Container: %s (started)\n", containerID[:12])
	result += fmt.Sprintf("Workspace: %s\n", s.projectMgr.GetWorkspaceDir(proj.ID))
	result += fmt.Sprintf("Image: %s\n", proj.ImageName)

	if proj.RemoteURL != "" {
		result += fmt.Sprintf("Git remote: %s\n", proj.RemoteURL)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: result},
		},
	}, nil, nil
}

type ListProjectsParams struct {
	NameContains *string `json:"name_contains,omitempty"`
	Limit        *int    `json:"limit,omitempty"`
}

func (s *Server) handleListProjects(ctx context.Context, request *mcp.CallToolRequest, params *ListProjectsParams) (*mcp.CallToolResult, any, error) {
	if _, err := requireAuth(ctx); err != nil {
		return nil, nil, err
	}

	var filter *project.ListProjectsFilter
	if params != nil && (params.NameContains != nil || params.Limit != nil) {
		filter = &project.ListProjectsFilter{}
		if params.NameContains != nil {
			filter.NameContains = *params.NameContains
		}
		if params.Limit != nil {
			filter.Limit = *params.Limit
		}
	}

	projects, err := s.projectMgr.List(filter)
	if err != nil {
		return nil, nil, err
	}

	if len(projects) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "No projects found"},
			},
		}, nil, nil
	}

	result := fmt.Sprintf("Found %d project(s):\n\n", len(projects))
	for _, proj := range projects {
		result += fmt.Sprintf("• %s (ID: %s)\n", proj.Name, proj.ID)
		if proj.Description != "" {
			result += fmt.Sprintf("  Description: %s\n", proj.Description)
		}
		result += fmt.Sprintf("  Image: %s\n", proj.ImageName)
		if proj.RemoteURL != "" {
			result += fmt.Sprintf("  Remote: %s\n", proj.RemoteURL)
		}

		containerName := fmt.Sprintf("oubliette-%s", proj.ID[:8])
		if status, err := s.runtime.Status(ctx, containerName); err == nil {
			result += fmt.Sprintf("  Container: %s\n", status)
		} else {
			result += "  Container: not running\n"
		}

		result += fmt.Sprintf("  Created: %s\n\n", proj.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: result},
		},
	}, nil, nil
}

type GetProjectParams struct {
	ProjectID string `json:"project_id"`
}

func (s *Server) handleGetProject(ctx context.Context, request *mcp.CallToolRequest, params *GetProjectParams) (*mcp.CallToolResult, any, error) {
	if params.ProjectID == "" {
		return nil, nil, fmt.Errorf("project_id is required")
	}

	if _, err := requireProjectAccess(ctx, params.ProjectID); err != nil {
		return nil, nil, err
	}

	proj, err := s.projectMgr.Get(params.ProjectID)
	if err != nil {
		return nil, nil, err
	}

	result := fmt.Sprintf("Project: %s\n", proj.Name)
	result += fmt.Sprintf("ID: %s\n\n", proj.ID)
	if proj.Description != "" {
		result += fmt.Sprintf("Description: %s\n", proj.Description)
	}
	result += fmt.Sprintf("Image: %s\n", proj.ImageName)
	result += fmt.Sprintf("Custom Dockerfile: %v\n", proj.HasDockerfile)
	result += fmt.Sprintf("Workspace: %s\n", s.projectMgr.GetWorkspaceDir(proj.ID))

	containerName := fmt.Sprintf("oubliette-%s", proj.ID[:8])
	if status, err := s.runtime.Status(ctx, containerName); err == nil {
		result += fmt.Sprintf("Container Status: %s\n", status)
	} else {
		result += "Container Status: not running\n"
	}

	if proj.RemoteURL != "" {
		result += fmt.Sprintf("Git remote: %s\n", proj.RemoteURL)
	}

	result += fmt.Sprintf("Created: %s\n", proj.CreatedAt.Format("2006-01-02 15:04:05"))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: result},
		},
	}, nil, nil
}

type DeleteProjectParams struct {
	ProjectID string `json:"project_id"`
}

func (s *Server) handleDeleteProject(ctx context.Context, request *mcp.CallToolRequest, params *DeleteProjectParams) (*mcp.CallToolResult, any, error) {
	if params.ProjectID == "" {
		return nil, nil, fmt.Errorf("project_id is required")
	}

	authCtx, err := requireProjectAccess(ctx, params.ProjectID)
	if err != nil {
		return nil, nil, err
	}
	if !authCtx.CanWrite() {
		return nil, nil, fmt.Errorf("read-only access, cannot delete project")
	}

	tokenID, tokenScope := getTokenInfo(authCtx)
	logger.Info("Deleting project: %s", params.ProjectID)

	containerName := fmt.Sprintf("oubliette-%s", params.ProjectID[:8])
	_ = s.runtime.Stop(ctx, containerName)
	_ = s.runtime.Remove(ctx, containerName, true)

	if err := CleanupSocketDir(params.ProjectID); err != nil {
		logger.Error("Failed to cleanup socket dir for project %s: %v", params.ProjectID, err)
	}

	if err := s.projectMgr.Delete(params.ProjectID); err != nil {
		logger.Error("Failed to delete project %s: %v", params.ProjectID, err)
		audit.LogFailure(audit.OpProjectDelete, tokenID, tokenScope, params.ProjectID, err)
		return nil, nil, err
	}

	audit.LogSuccess(audit.OpProjectDelete, tokenID, tokenScope, params.ProjectID)
	logger.Info("Project deleted successfully: %s", params.ProjectID)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Project '%s' deleted successfully", params.ProjectID)},
		},
	}, nil, nil
}

// getTokenInfo extracts token ID and scope from auth context
func getTokenInfo(authCtx *auth.AuthContext) (string, string) {
	if authCtx == nil || authCtx.Token == nil {
		return "", ""
	}
	return authCtx.Token.ID, authCtx.Token.Scope
}

// ProjectChangesParams for the project_changes tool
type ProjectChangesParams struct {
	ProjectID string `json:"project_id"`
}

// ProjectChangesResult mirrors OpenSpec list --json output with session info
type ProjectChangesResult struct {
	ProjectID string               `json:"project_id"`
	Changes   []OpenSpecChangeInfo `json:"changes"`
}

// OpenSpecChangeInfo represents a single change from openspec list --json
type OpenSpecChangeInfo struct {
	Name           string   `json:"name"`
	CompletedTasks int      `json:"completedTasks"`
	TotalTasks     int      `json:"totalTasks"`
	LastModified   string   `json:"lastModified"`
	Status         string   `json:"status"`
	ActiveSessions []string `json:"active_sessions,omitempty"`
}

func (s *Server) handleProjectChanges(ctx context.Context, request *mcp.CallToolRequest, params *ProjectChangesParams) (*mcp.CallToolResult, any, error) {
	if params.ProjectID == "" {
		return nil, nil, fmt.Errorf("project_id is required")
	}

	_, err := requireProjectAccess(ctx, params.ProjectID)
	if err != nil {
		return nil, nil, err
	}

	// Execute openspec list --json in the container
	containerName := fmt.Sprintf("oubliette-%s", params.ProjectID[:8])

	// Run openspec list --json --sort name (working dir is /workspace inside container)
	execResult, err := s.runtime.Exec(ctx, containerName, container.ExecConfig{
		Cmd:          []string{"bash", "-c", "source ~/.nvm/nvm.sh && cd /workspace && OPENSPEC_TELEMETRY=0 openspec list --json --sort name 2>/dev/null || echo '{\"changes\":[]}'"},
		WorkingDir:   "/workspace",
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		logger.Error("Failed to run openspec list for project %s: %v", params.ProjectID, err)
		return nil, nil, fmt.Errorf("failed to get changes: %w", err)
	}
	output := execResult.Stdout
	logger.Info("OpenSpec list output for project %s: stdout=%q stderr=%q exit=%d", params.ProjectID, output, execResult.Stderr, execResult.ExitCode)

	// Parse the OpenSpec output
	var openspecOutput struct {
		Changes []struct {
			Name           string `json:"name"`
			CompletedTasks int    `json:"completedTasks"`
			TotalTasks     int    `json:"totalTasks"`
			LastModified   string `json:"lastModified"`
			Status         string `json:"status"`
		} `json:"changes"`
	}
	if err := json.Unmarshal([]byte(output), &openspecOutput); err != nil {
		// If parsing fails, return raw output
		result := fmt.Sprintf(`{"project_id": "%s", "raw_output": %s}`, params.ProjectID, output)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: result}},
		}, nil, nil
	}

	// Enrich with active session info
	enrichedChanges := make([]OpenSpecChangeInfo, 0, len(openspecOutput.Changes))
	for _, change := range openspecOutput.Changes {
		changeInfo := OpenSpecChangeInfo{
			Name:           change.Name,
			CompletedTasks: change.CompletedTasks,
			TotalTasks:     change.TotalTasks,
			LastModified:   change.LastModified,
			Status:         change.Status,
		}

		// Get active sessions working on this change
		activeSessions := s.activeSessions.GetSessionsByChangeID(params.ProjectID, change.Name)
		if len(activeSessions) > 0 {
			sessionIDs := make([]string, 0, len(activeSessions))
			for _, sess := range activeSessions {
				sessionIDs = append(sessionIDs, sess.SessionID)
			}
			changeInfo.ActiveSessions = sessionIDs
		}

		enrichedChanges = append(enrichedChanges, changeInfo)
	}

	result := ProjectChangesResult{
		ProjectID: params.ProjectID,
		Changes:   enrichedChanges,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
	}, result, nil
}

// ProjectTasksParams for the project_tasks tool
type ProjectTasksParams struct {
	ProjectID string `json:"project_id"`
	ChangeID  string `json:"change_id"`
}

func (s *Server) handleProjectTasks(ctx context.Context, request *mcp.CallToolRequest, params *ProjectTasksParams) (*mcp.CallToolResult, any, error) {
	if params.ProjectID == "" {
		return nil, nil, fmt.Errorf("project_id is required")
	}
	if params.ChangeID == "" {
		return nil, nil, fmt.Errorf("change_id is required")
	}

	_, err := requireProjectAccess(ctx, params.ProjectID)
	if err != nil {
		return nil, nil, err
	}

	// Execute openspec instructions apply --json in the container
	containerName := fmt.Sprintf("oubliette-%s", params.ProjectID[:8])

	// Run openspec instructions apply --change <change_id> --json (working dir is /workspace inside container)
	cmd := fmt.Sprintf(
		"source ~/.nvm/nvm.sh && cd /workspace && OPENSPEC_TELEMETRY=0 openspec instructions apply --change %s --json 2>/dev/null || echo '{\"error\": \"change not found\"}'",
		params.ChangeID,
	)
	execResult, err := s.runtime.Exec(ctx, containerName, container.ExecConfig{
		Cmd:          []string{"bash", "-c", cmd},
		WorkingDir:   "/workspace",
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		logger.Error("Failed to run openspec instructions for project %s change %s: %v", params.ProjectID, params.ChangeID, err)
		return nil, nil, fmt.Errorf("failed to get tasks: %w", err)
	}
	output := execResult.Stdout

	// Return the raw JSON output
	result := fmt.Sprintf(`{"project_id": "%s", "change_id": "%s", "raw_output": %s}`, params.ProjectID, params.ChangeID, output)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: result},
		},
	}, nil, nil
}

// Project Options - returns available configuration options for project creation

type ProjectOptionsParams struct{}

type ProjectOptionsResponse struct {
	Defaults       ProjectDefaultsResponse     `json:"defaults"`
	Credentials    *CredentialsOptionsResponse `json:"credentials,omitempty"`
	Models         *ModelsOptionsResponse      `json:"models,omitempty"`
	ContainerTypes *ContainerTypesResponse     `json:"container_types,omitempty"`
	TokenScopes    *TokenScopesResponse        `json:"token_scopes,omitempty"`
}

type TokenScopesResponse struct {
	Formats []TokenScopeFormat `json:"formats"`
}

type TokenScopeFormat struct {
	Format      string `json:"format"`
	Description string `json:"description"`
	Example     string `json:"example,omitempty"`
}

type ProjectDefaultsResponse struct {
	MaxRecursionDepth   int     `json:"max_recursion_depth"`
	MaxAgentsPerSession int     `json:"max_agents_per_session"`
	MaxCostUSD          float64 `json:"max_cost_usd"`
}

type CredentialsOptionsResponse struct {
	GitHub    []CredentialOptionInfo         `json:"github"`
	Providers []ProviderCredentialOptionInfo `json:"providers"`
}

type CredentialOptionInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsDefault   bool   `json:"is_default,omitempty"`
}

type ProviderCredentialOptionInfo struct {
	Name        string `json:"name"`
	Provider    string `json:"provider"`
	Description string `json:"description"`
	IsDefault   bool   `json:"is_default,omitempty"`
}

type ModelsOptionsResponse struct {
	Available []ModelOptionInfo `json:"available"`
}

type ModelOptionInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Provider    string `json:"provider"`
}

type ContainerTypesResponse struct {
	Available []ContainerTypeInfo `json:"available"`
	Default   string              `json:"default"`
}

type ContainerTypeInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (s *Server) handleProjectOptions(ctx context.Context, request *mcp.CallToolRequest, params *ProjectOptionsParams) (*mcp.CallToolResult, any, error) {
	if _, err := requireAuth(ctx); err != nil {
		return nil, nil, err
	}

	// Get defaults from project manager
	response := ProjectOptionsResponse{
		Defaults: ProjectDefaultsResponse{
			MaxRecursionDepth:   s.projectMgr.GetDefaultMaxDepth(),
			MaxAgentsPerSession: s.projectMgr.GetDefaultMaxAgents(),
			MaxCostUSD:          s.projectMgr.GetDefaultMaxCostUSD(),
		},
	}

	// Add credentials if configured
	if s.credentials != nil {
		credsList := s.credentials.ListCredentials()

		github := make([]CredentialOptionInfo, len(credsList.GitHub))
		for i, c := range credsList.GitHub {
			github[i] = CredentialOptionInfo{
				Name:        c.Name,
				Description: c.Description,
				IsDefault:   c.IsDefault,
			}
		}

		providers := make([]ProviderCredentialOptionInfo, len(credsList.Providers))
		for i, c := range credsList.Providers {
			providers[i] = ProviderCredentialOptionInfo{
				Name:        c.Name,
				Provider:    c.Provider,
				Description: c.Description,
				IsDefault:   c.IsDefault,
			}
		}

		response.Credentials = &CredentialsOptionsResponse{
			GitHub:    github,
			Providers: providers,
		}
	}

	// Add models if configured
	if s.modelRegistry != nil && len(s.modelRegistry.Models) > 0 {
		models := s.modelRegistry.ListModels()
		available := make([]ModelOptionInfo, len(models))
		for i, m := range models {
			available[i] = ModelOptionInfo{
				Name:        m.Name,
				DisplayName: m.DisplayName,
				Provider:    m.Provider,
			}
		}
		response.Models = &ModelsOptionsResponse{
			Available: available,
		}
	}

	// Add container types from ImageManager
	if s.imageManager != nil {
		types := s.imageManager.ValidTypes()
		available := make([]ContainerTypeInfo, len(types))
		for i, t := range types {
			available[i] = ContainerTypeInfo{
				Name:        t,
				Description: fmt.Sprintf("Container type '%s'", t),
			}
		}
		response.ContainerTypes = &ContainerTypesResponse{
			Available: available,
			Default:   "dev",
		}
	}

	// Add token scope formats
	response.TokenScopes = &TokenScopesResponse{
		Formats: []TokenScopeFormat{
			{Format: "admin", Description: "Full access to all tools and all projects", Example: "admin"},
			{Format: "admin:ro", Description: "Read-only access to all tools and all projects", Example: "admin:ro"},
			{Format: "project:<uuid>", Description: "Full access to one project only", Example: "project:proj_abc123"},
			{Format: "project:<uuid>:ro", Description: "Read-only access to one project only", Example: "project:proj_abc123:ro"},
		},
	}

	data, _ := json.MarshalIndent(response, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(data)},
		},
	}, response, nil
}
