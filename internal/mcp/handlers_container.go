package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/HyphaGroup/oubliette/internal/config"
	"github.com/HyphaGroup/oubliette/internal/container"
	"github.com/HyphaGroup/oubliette/internal/logger"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Container Operation Handlers

type SpawnContainerParams struct {
	ProjectID string `json:"project_id"`
}

func (s *Server) handleSpawnContainer(ctx context.Context, request *mcp.CallToolRequest, params *SpawnContainerParams) (*mcp.CallToolResult, any, error) {
	if params.ProjectID == "" {
		return nil, nil, fmt.Errorf("project_id is required")
	}

	authCtx, err := requireProjectAccess(ctx, params.ProjectID)
	if err != nil {
		return nil, nil, err
	}
	if !authCtx.CanWrite() {
		return nil, nil, fmt.Errorf("read-only access, cannot start container")
	}

	logger.Info("Spawning container for project: %s", params.ProjectID)

	proj, err := s.projectMgr.Get(params.ProjectID)
	if err != nil {
		logger.Error("Failed to get project %s: %v", params.ProjectID, err)
		return nil, nil, err
	}

	containerName := fmt.Sprintf("oubliette-%s", params.ProjectID[:8])
	containerID, err := s.createAndStartContainer(ctx, containerName, proj.ImageName, params.ProjectID)
	if err != nil {
		logger.Error("Failed to spawn container for %s: %v", params.ProjectID, err)
		return nil, nil, err
	}

	logger.Info("Container spawned successfully for %s: %s", params.ProjectID, containerID[:12])

	result := "✅ Container started successfully\n\n"
	result += fmt.Sprintf("Container ID: %s\n", containerID[:12])
	result += fmt.Sprintf("Project: %s\n", params.ProjectID)
	result += "Mode: long-lived (use container_stop to stop)\n"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: result},
		},
	}, nil, nil
}

type ExecCommandParams struct {
	ProjectID  string `json:"project_id"`
	Command    string `json:"command"`
	WorkingDir string `json:"working_dir,omitempty"`
}

func (s *Server) handleExecCommand(ctx context.Context, request *mcp.CallToolRequest, params *ExecCommandParams) (*mcp.CallToolResult, any, error) {
	if params.ProjectID == "" {
		return nil, nil, fmt.Errorf("project_id is required")
	}
	if params.Command == "" {
		return nil, nil, fmt.Errorf("command is required")
	}

	containerName := fmt.Sprintf("oubliette-%s", params.ProjectID[:8])

	execResult, err := s.runtime.Exec(ctx, containerName, container.ExecConfig{
		Cmd:          []string{"/bin/bash", "-c", params.Command},
		WorkingDir:   params.WorkingDir,
		AttachStdout: true,
		AttachStderr: true,
	})

	var output string
	if execResult != nil {
		output = execResult.Stdout
		if execResult.Stderr != "" {
			output += "\nSTDERR:\n" + execResult.Stderr
		}
	}

	if err != nil {
		result := fmt.Sprintf("Command failed: %v\n\nOutput:\n%s", err, output)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: output},
		},
	}, nil, nil
}

type StopContainerParams struct {
	ProjectID string `json:"project_id"`
}

func (s *Server) handleStopContainer(ctx context.Context, request *mcp.CallToolRequest, params *StopContainerParams) (*mcp.CallToolResult, any, error) {
	if params.ProjectID == "" {
		return nil, nil, fmt.Errorf("project_id is required")
	}

	logger.Info("Stopping container for project: %s", params.ProjectID)

	containerName := fmt.Sprintf("oubliette-%s", params.ProjectID[:8])

	if err := s.runtime.Stop(ctx, containerName); err != nil {
		logger.Error("Failed to stop container for %s: %v", params.ProjectID, err)
		return nil, nil, err
	}

	logger.Info("Container stopped successfully for project: %s", params.ProjectID)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Container for project '%s' stopped", params.ProjectID)},
		},
	}, nil, nil
}

type GetLogsParams struct {
	ProjectID string `json:"project_id"`
}

func (s *Server) handleGetLogs(ctx context.Context, request *mcp.CallToolRequest, params *GetLogsParams) (*mcp.CallToolResult, any, error) {
	if params.ProjectID == "" {
		return nil, nil, fmt.Errorf("project_id is required")
	}

	containerName := fmt.Sprintf("oubliette-%s", params.ProjectID[:8])

	logs, err := s.runtime.Logs(ctx, containerName, container.LogsOptions{Tail: "1000"})
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: logs},
		},
	}, nil, nil
}

// ContainerRefreshParams for the container_refresh tool
type ContainerRefreshParams struct {
	ProjectID     string `json:"project_id,omitempty"`
	ContainerType string `json:"container_type,omitempty"`
}

// handleContainerRefresh pulls the latest image and restarts the container
// Fails if there are active sessions for the project
func (s *Server) handleContainerRefresh(ctx context.Context, request *mcp.CallToolRequest, params *ContainerRefreshParams) (*mcp.CallToolResult, any, error) {
	// Require project_id or container_type
	if params.ProjectID == "" && params.ContainerType == "" {
		return nil, nil, fmt.Errorf("either project_id or container_type is required")
	}

	if s.imageManager == nil {
		return nil, nil, fmt.Errorf("image manager not configured")
	}

	// If container_type is specified, just ensure image exists (pull if needed)
	if params.ContainerType != "" {
		if !s.imageManager.IsValidType(params.ContainerType) {
			return nil, nil, fmt.Errorf("invalid container_type: %s (valid types: %v)", params.ContainerType, s.imageManager.ValidTypes())
		}

		imageName, _ := s.imageManager.GetImageName(params.ContainerType)

		// Pull the image
		logger.Info("Refreshing container type %s (image: %s)", params.ContainerType, imageName)
		if err := s.runtime.Pull(ctx, imageName); err != nil {
			return nil, nil, fmt.Errorf("failed to pull image %s: %w", imageName, err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("✅ Container type '%s' refreshed successfully (image: %s)", params.ContainerType, imageName)},
			},
		}, nil, nil
	}

	// Project-specific refresh
	proj, err := s.projectMgr.Get(params.ProjectID)
	if err != nil {
		return nil, nil, err
	}

	// Check for active sessions - fail if any exist
	if s.activeSessions.CountByProject(params.ProjectID) > 0 {
		return nil, nil, fmt.Errorf("cannot refresh container: project has active sessions (stop sessions first)")
	}

	containerName := fmt.Sprintf("oubliette-%s", params.ProjectID[:8])

	// Pull the image
	logger.Info("Refreshing container for project %s (image: %s)", params.ProjectID, proj.ImageName)
	if err := s.runtime.Pull(ctx, proj.ImageName); err != nil {
		return nil, nil, fmt.Errorf("failed to pull image %s: %w", proj.ImageName, err)
	}

	// Stop and remove old container
	_ = s.runtime.Stop(ctx, containerName)
	_ = s.runtime.Remove(ctx, containerName, true)

	// Start fresh container
	containerID, err := s.createAndStartContainer(ctx, containerName, proj.ImageName, params.ProjectID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start refreshed container: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("✅ Container refreshed successfully\n\nContainer ID: %s\nImage: %s", containerID[:12], proj.ImageName)},
		},
	}, nil, nil
}

// createAndStartContainer creates and starts a container for a project
func (s *Server) createAndStartContainer(ctx context.Context, containerName, imageName, projectName string) (string, error) {
	// Check if image exists
	exists, err := s.runtime.ImageExists(ctx, imageName)
	if err != nil {
		logger.Error("Failed to check image %s: %v", imageName, err)
		// Continue anyway - the create will fail if image truly doesn't exist
	} else if !exists {
		// Try to pull the image
		logger.Info("Image %s not found, pulling...", imageName)
		if err := s.runtime.Pull(ctx, imageName); err != nil {
			return "", fmt.Errorf("failed to pull image %s: %w", imageName, err)
		}
		logger.Info("Pulled image %s successfully", imageName)
	}

	// Remove any existing container with the same name
	_ = s.runtime.Stop(ctx, containerName)
	_ = s.runtime.Remove(ctx, containerName, true)

	projectDir := s.projectMgr.GetProjectDir(projectName)
	projectFactoryDir := filepath.Join(projectDir, ".factory")

	// Ensure directories exist
	_ = os.MkdirAll(projectFactoryDir, 0o755)
	_ = os.MkdirAll(filepath.Join(projectDir, "cache", "npm"), 0o755)
	_ = os.MkdirAll(filepath.Join(projectDir, "cache", "pip"), 0o755)
	_ = os.MkdirAll(filepath.Join(projectDir, "cache", "maven"), 0o755)
	_ = os.MkdirAll(filepath.Join(projectDir, "cache", "gradle"), 0o755)
	_ = os.MkdirAll(filepath.Join(projectDir, "history"), 0o755)
	_ = os.MkdirAll(filepath.Join(projectDir, "ssh"), 0o755)

	// Ensure socket directory exists on host
	if err := EnsureSocketDir(projectName); err != nil {
		logger.Error("Failed to create socket directory: %v", err)
		return "", fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Get project to check workspace isolation setting
	proj, err := s.projectMgr.Get(projectName)
	if err != nil {
		return "", fmt.Errorf("failed to get project: %w", err)
	}

	// Determine workspace mount based on isolation setting
	var workspaceSource string
	var additionalMounts []container.Mount
	if proj.WorkspaceIsolation {
		// Isolated mode: mount only workspaces directory
		// Sessions will have workingDir /workspace/<workspace_id>
		workspaceSource = filepath.Join(projectDir, "workspaces")
		_ = os.MkdirAll(workspaceSource, 0o755)

		// Add read-only mounts for protected paths
		// Note: Protected paths are mounted from the project root as shared read-only resources
		// They appear at /workspace/.protected/<path> and workspaces can reference them
		if len(proj.ProtectedPaths) > 0 {
			for _, protPath := range proj.ProtectedPaths {
				srcPath := filepath.Join(projectDir, protPath)
				// Validate path exists before adding mount (skip missing)
				if _, err := os.Stat(srcPath); err == nil {
					// Mount to a .protected directory to make them accessible read-only
					targetPath := filepath.Join("/workspace/.protected", protPath)
					additionalMounts = append(additionalMounts, container.Mount{
						Type:     container.MountTypeBind,
						Source:   srcPath,
						Target:   targetPath,
						ReadOnly: true,
					})
					logger.Info("Added read-only protected mount: %s -> %s", srcPath, targetPath)
				} else {
					logger.Info("Skipping protected path %s (not found)", protPath)
				}
			}
		}
	} else {
		// Non-isolated mode: mount full project directory
		// Sessions will have workingDir /workspace/workspaces/<workspace_id>
		workspaceSource = projectDir
	}

	// Build mount list
	mounts := []container.Mount{
		{Type: container.MountTypeBind, Source: projectFactoryDir, Target: "/home/gogol/.factory"},
		{Type: container.MountTypeBind, Source: workspaceSource, Target: "/workspace"},
		{Type: container.MountTypeBind, Source: filepath.Join(projectDir, "cache", "npm"), Target: "/home/gogol/.npm"},
		{Type: container.MountTypeBind, Source: filepath.Join(projectDir, "cache", "pip"), Target: "/home/gogol/.cache/pip"},
		{Type: container.MountTypeBind, Source: filepath.Join(projectDir, "cache", "maven"), Target: "/home/gogol/.m2"},
		{Type: container.MountTypeBind, Source: filepath.Join(projectDir, "cache", "gradle"), Target: "/home/gogol/.gradle"},
		{Type: container.MountTypeBind, Source: filepath.Join(projectDir, "history"), Target: "/home/gogol/.shell_history"},
		{Type: container.MountTypeBind, Source: filepath.Join(projectDir, "ssh"), Target: "/home/gogol/.ssh"},
	}

	// Note: config.json and opencode.json are already inside projectDir which is mounted at /workspace
	// They don't need separate mounts. The .factory/ directory is mounted at /home/gogol/.factory
	// so agents can read configs but writing to /workspace/.factory won't affect the real configs.

	// Add protected path mounts (read-only)
	mounts = append(mounts, additionalMounts...)

	cfg := container.CreateConfig{
		Name:       containerName,
		Image:      imageName,
		Cmd:        []string{"sleep", "infinity"},
		WorkingDir: "/workspace",
		Env: []string{
			"HOME=/home/gogol",
			"HISTFILE=/home/gogol/.shell_history/bash_history",
			"MAX_MCP_OUTPUT_TOKENS=50000",
			fmt.Sprintf("OUBLIETTE_PROJECT_ID=%s", projectName),
			fmt.Sprintf("OUBLIETTE_WORKSPACE_ISOLATION=%t", proj.WorkspaceIsolation),
		},
		Mounts: mounts,
		PublishedSockets: []container.PublishedSocket{
			{
				HostPath:      SocketPath(projectName),
				ContainerPath: "/mcp/relay.sock",
			},
		},
		Init:        true,
		NetworkMode: "bridge",
		Memory:      s.containerMemory,
		CPUs:        s.containerCPUs,
	}

	// Inject provider credentials from unified registry
	// TODO: Support project-specific credential refs (credential_refs.provider)
	if s.credentials != nil {
		if provCred, ok := s.credentials.GetDefaultProviderCredential(); ok && provCred.APIKey != "" {
			envVar := config.ProviderEnvVar(provCred.Provider)
			if envVar != "" {
				cfg.Env = append(cfg.Env, fmt.Sprintf("%s=%s", envVar, provCred.APIKey))
			}
		}
	}

	containerID, err := s.runtime.Create(ctx, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	if err := s.runtime.Start(ctx, containerID); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	return containerID, nil
}
