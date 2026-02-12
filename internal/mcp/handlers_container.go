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

// ContainerParams is the params struct for the container tool
type ContainerParams struct {
	Action string `json:"action"` // Required: start, stop, logs, exec

	ProjectID  string `json:"project_id,omitempty"`
	Command    string `json:"command,omitempty"`
	WorkingDir string `json:"working_dir,omitempty"`
}

var containerActions = []string{"start", "stop", "logs", "exec"}

func (s *Server) handleContainer(ctx context.Context, request *mcp.CallToolRequest, params *ContainerParams) (*mcp.CallToolResult, any, error) {
	if params.Action == "" {
		return nil, nil, missingActionError("container", containerActions)
	}

	switch params.Action {
	case "start":
		return s.handleContainerStart(ctx, request, params)
	case "stop":
		return s.handleContainerStop(ctx, request, params)
	case "logs":
		return s.handleContainerLogs(ctx, request, params)
	case "exec":
		return s.handleContainerExec(ctx, request, params)
	default:
		return nil, nil, actionError("container", params.Action, containerActions)
	}
}

func (s *Server) handleContainerStart(ctx context.Context, request *mcp.CallToolRequest, params *ContainerParams) (*mcp.CallToolResult, any, error) {
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

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: result},
		},
	}, nil, nil
}

func (s *Server) handleContainerExec(ctx context.Context, request *mcp.CallToolRequest, params *ContainerParams) (*mcp.CallToolResult, any, error) {
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

func (s *Server) handleContainerStop(ctx context.Context, request *mcp.CallToolRequest, params *ContainerParams) (*mcp.CallToolResult, any, error) {
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

func (s *Server) handleContainerLogs(ctx context.Context, request *mcp.CallToolRequest, params *ContainerParams) (*mcp.CallToolResult, any, error) {
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

func (s *Server) handleContainerRefresh(ctx context.Context, request *mcp.CallToolRequest, params *ContainerRefreshParams) (*mcp.CallToolResult, any, error) {
	if params.ProjectID == "" && params.ContainerType == "" {
		return nil, nil, fmt.Errorf("either project_id or container_type is required")
	}

	if s.imageManager == nil {
		return nil, nil, fmt.Errorf("image manager not configured")
	}

	if params.ContainerType != "" {
		if !s.imageManager.IsValidType(params.ContainerType) {
			return nil, nil, fmt.Errorf("invalid container_type: %s (valid types: %v)", params.ContainerType, s.imageManager.ValidTypes())
		}

		imageName, _ := s.imageManager.GetImageName(params.ContainerType)

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

	proj, err := s.projectMgr.Get(params.ProjectID)
	if err != nil {
		return nil, nil, err
	}

	if s.activeSessions.CountByProject(params.ProjectID) > 0 {
		return nil, nil, fmt.Errorf("cannot refresh container: project has active sessions (stop sessions first)")
	}

	containerName := fmt.Sprintf("oubliette-%s", params.ProjectID[:8])

	logger.Info("Refreshing container for project %s (image: %s)", params.ProjectID, proj.ImageName)
	if err := s.runtime.Pull(ctx, proj.ImageName); err != nil {
		return nil, nil, fmt.Errorf("failed to pull image %s: %w", proj.ImageName, err)
	}

	_ = s.runtime.Stop(ctx, containerName)
	_ = s.runtime.Remove(ctx, containerName, true)

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

func (s *Server) createAndStartContainer(ctx context.Context, containerName, imageName, projectName string) (string, error) {
	exists, err := s.runtime.ImageExists(ctx, imageName)
	if err != nil {
		logger.Error("Failed to check image %s: %v", imageName, err)
	} else if !exists {
		logger.Info("Image %s not found, pulling...", imageName)
		if err := s.runtime.Pull(ctx, imageName); err != nil {
			return "", fmt.Errorf("failed to pull image %s: %w", imageName, err)
		}
		logger.Info("Pulled image %s successfully", imageName)
	}

	_ = s.runtime.Stop(ctx, containerName)
	_ = s.runtime.Remove(ctx, containerName, true)

	projectDir := s.projectMgr.GetProjectDir(projectName)

	_ = os.MkdirAll(filepath.Join(projectDir, "cache", "npm"), 0o755)
	_ = os.MkdirAll(filepath.Join(projectDir, "cache", "pip"), 0o755)
	_ = os.MkdirAll(filepath.Join(projectDir, "cache", "maven"), 0o755)
	_ = os.MkdirAll(filepath.Join(projectDir, "cache", "gradle"), 0o755)
	_ = os.MkdirAll(filepath.Join(projectDir, "history"), 0o755)
	_ = os.MkdirAll(filepath.Join(projectDir, "ssh"), 0o755)

	if err := EnsureSocketDir(projectName); err != nil {
		logger.Error("Failed to create socket directory: %v", err)
		return "", fmt.Errorf("failed to create socket directory: %w", err)
	}

	proj, err := s.projectMgr.Get(projectName)
	if err != nil {
		return "", fmt.Errorf("failed to get project: %w", err)
	}

	var workspaceSource string
	var additionalMounts []container.Mount
	if proj.WorkspaceIsolation {
		workspaceSource = filepath.Join(projectDir, "workspaces")
		_ = os.MkdirAll(workspaceSource, 0o755)

		if len(proj.ProtectedPaths) > 0 {
			for _, protPath := range proj.ProtectedPaths {
				srcPath := filepath.Join(projectDir, protPath)
				if _, err := os.Stat(srcPath); err == nil {
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
		workspaceSource = projectDir
	}

	mounts := []container.Mount{
		{Type: container.MountTypeBind, Source: workspaceSource, Target: "/workspace"},
		{Type: container.MountTypeBind, Source: filepath.Join(projectDir, "cache", "npm"), Target: "/home/gogol/.npm"},
		{Type: container.MountTypeBind, Source: filepath.Join(projectDir, "cache", "pip"), Target: "/home/gogol/.cache/pip"},
		{Type: container.MountTypeBind, Source: filepath.Join(projectDir, "cache", "maven"), Target: "/home/gogol/.m2"},
		{Type: container.MountTypeBind, Source: filepath.Join(projectDir, "cache", "gradle"), Target: "/home/gogol/.gradle"},
		{Type: container.MountTypeBind, Source: filepath.Join(projectDir, "history"), Target: "/home/gogol/.shell_history"},
		{Type: container.MountTypeBind, Source: filepath.Join(projectDir, "ssh"), Target: "/home/gogol/.ssh"},
	}

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
