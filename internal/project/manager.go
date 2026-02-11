package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	agentconfig "github.com/HyphaGroup/oubliette/internal/agent/config"
	"github.com/HyphaGroup/oubliette/internal/config"
	"github.com/HyphaGroup/oubliette/internal/validation"
	"github.com/google/uuid"
)

// NewManager creates a new project manager
func NewManager(projectsDir string, defaultMaxDepth, defaultMaxAgents int, defaultMaxCostUSD float64) *Manager {
	// Template dir is sibling to projects dir
	templateDir := filepath.Join(filepath.Dir(projectsDir), "template")
	return &Manager{
		projectsDir:       projectsDir,
		templateDir:       templateDir,
		defaultMaxDepth:   defaultMaxDepth,
		defaultMaxAgents:  defaultMaxAgents,
		defaultMaxCostUSD: defaultMaxCostUSD,
	}
}

// SetModelRegistry sets the model configuration for the manager
func (m *Manager) SetModelRegistry(registry *ModelRegistry) {
	m.modelRegistry = registry
}

// SetConfigDefaults sets the config defaults for the manager
func (m *Manager) SetConfigDefaults(defaults *config.ConfigDefaultsConfig) {
	m.configDefaults = defaults
}

// SetContainers sets the container type -> image name mapping
func (m *Manager) SetContainers(containers map[string]string) {
	m.containers = containers
}

// GetImageNameForType returns the image name for a container type
func (m *Manager) GetImageNameForType(containerType string) string {
	if m.containers != nil {
		if imageName, ok := m.containers[containerType]; ok {
			return imageName
		}
	}
	// Fallback to local naming convention
	return fmt.Sprintf("oubliette-%s:latest", containerType)
}

// GetMaxDepth returns the max recursion depth for a project (project override or env default)
func (m *Manager) GetMaxDepth(project *Project) int {
	if project.RecursionConfig != nil && project.RecursionConfig.MaxDepth != nil {
		return *project.RecursionConfig.MaxDepth
	}
	return m.defaultMaxDepth
}

// GetMaxAgents returns the max agents per session for a project (project override or env default)
func (m *Manager) GetMaxAgents(project *Project) int {
	if project.RecursionConfig != nil && project.RecursionConfig.MaxAgents != nil {
		return *project.RecursionConfig.MaxAgents
	}
	return m.defaultMaxAgents
}

// GetMaxCostUSD returns the max cost in USD for a project (project override or env default)
func (m *Manager) GetMaxCostUSD(project *Project) float64 {
	if project.RecursionConfig != nil && project.RecursionConfig.MaxCostUSD != nil {
		return *project.RecursionConfig.MaxCostUSD
	}
	return m.defaultMaxCostUSD
}

// GetDefaultMaxDepth returns the server default max recursion depth
func (m *Manager) GetDefaultMaxDepth() int {
	return m.defaultMaxDepth
}

// GetDefaultMaxAgents returns the server default max agents per session
func (m *Manager) GetDefaultMaxAgents() int {
	return m.defaultMaxAgents
}

// GetDefaultMaxCostUSD returns the server default max cost in USD
func (m *Manager) GetDefaultMaxCostUSD() float64 {
	return m.defaultMaxCostUSD
}

// SetSessionChecker sets the active session checker for safe deletion
func (m *Manager) SetSessionChecker(checker ActiveSessionChecker) {
	m.sessionChecker = checker
}

// Create creates a new project with all necessary directories
func (m *Manager) Create(req CreateProjectRequest) (*Project, error) {
	// Generate UUIDs for project and default workspace
	projectID := uuid.New().String()
	defaultWorkspaceID := uuid.New().String()
	projectDir := filepath.Join(m.projectsDir, projectID)

	// UUID-based directories should never collide, but check anyway
	if _, err := os.Stat(projectDir); err == nil {
		return nil, fmt.Errorf("project directory already exists (UUID collision): %s", projectID)
	}

	// Create project directory structure
	dirs := []string{
		projectDir,
		// Workspaces container
		filepath.Join(projectDir, "workspaces"),
		// Cache directories
		filepath.Join(projectDir, "cache"),
		filepath.Join(projectDir, "cache", "npm"),
		filepath.Join(projectDir, "cache", "pip"),
		filepath.Join(projectDir, "cache", "maven"),
		filepath.Join(projectDir, "cache", "gradle"),
		filepath.Join(projectDir, "history"),
		filepath.Join(projectDir, "ssh"),
		// Sessions directory (for metadata storage)
		filepath.Join(projectDir, "sessions"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Copy templates to project
	templateBaseDir := filepath.Join(filepath.Dir(m.projectsDir), "template")

	// Copy template/openspec/ to project/openspec/ (for OpenSpec spec-driven workflow)
	templateOpenspecDir := filepath.Join(templateBaseDir, "openspec")
	projectOpenspecDir := filepath.Join(projectDir, "openspec")
	if _, err := os.Stat(templateOpenspecDir); err == nil {
		if err := m.copyDir(templateOpenspecDir, projectOpenspecDir); err != nil {
			return nil, fmt.Errorf("failed to copy template openspec: %w", err)
		}
	}

	// Copy template/AGENTS.md to project/AGENTS.md (root stub for AI assistants)
	templateAgentsMD := filepath.Join(templateBaseDir, "AGENTS.md")
	projectAgentsMD := filepath.Join(projectDir, "AGENTS.md")
	if _, err := os.Stat(templateAgentsMD); err == nil {
		if err := m.copyFile(templateAgentsMD, projectAgentsMD); err != nil {
			return nil, fmt.Errorf("failed to copy template AGENTS.md: %w", err)
		}
	}

	// Set SSH directory permissions
	sshDir := filepath.Join(projectDir, "ssh")
	if err := os.Chmod(sshDir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to set SSH directory permissions: %w", err)
	}

	// GitHub token is resolved in MCP layer (via account registry)
	// Just use what was passed in the request
	githubToken := req.GitHubToken

	// Build canonical config (new config system)
	createdAt := time.Now()
	canonicalConfig := m.buildCanonicalConfig(projectID, defaultWorkspaceID, req, createdAt)

	// Save canonical config (projects/<id>/config.json)
	if err := m.saveCanonicalConfig(canonicalConfig); err != nil {
		return nil, fmt.Errorf("failed to save canonical config: %w", err)
	}

	// Generate runtime configs for both runtimes
	if err := m.generateRuntimeConfigs(canonicalConfig); err != nil {
		return nil, fmt.Errorf("failed to generate runtime configs: %w", err)
	}

	// Build recursion config for Project struct (legacy compatibility)
	var recursionConfig *RecursionConfig
	if req.MaxRecursionDepth != nil || req.MaxAgentsPerSession != nil || req.MaxCostUSD != nil {
		recursionConfig = &RecursionConfig{
			MaxDepth:   req.MaxRecursionDepth,
			MaxAgents:  req.MaxAgentsPerSession,
			MaxCostUSD: req.MaxCostUSD,
		}
	}

	// Create project metadata (Project struct for API responses - populated from canonical config)
	project := &Project{
		ID:                 canonicalConfig.ID,
		Name:               canonicalConfig.Name,
		Description:        canonicalConfig.Description,
		DefaultWorkspaceID: canonicalConfig.DefaultWorkspaceID,
		CreatedAt:          canonicalConfig.CreatedAt,
		GitHubToken:        githubToken,
		RemoteURL:          req.RemoteURL,
		ImageName:          canonicalConfig.Container.ImageName,
		ContainerType:      canonicalConfig.Container.Type,
		Model:              canonicalConfig.Agent.Model,
		WorkspaceIsolation: canonicalConfig.WorkspaceIsolation,
		ProtectedPaths:     canonicalConfig.ProtectedPaths,
		RecursionConfig:    recursionConfig,
	}

	// Save project metadata (legacy metadata.json for backwards compatibility during transition)
	if err := m.saveMetadata(project); err != nil {
		return nil, fmt.Errorf("failed to save project metadata: %w", err)
	}

	// Create default workspace
	if _, err := m.CreateWorkspace(projectID, defaultWorkspaceID, "", "project_create"); err != nil {
		return nil, fmt.Errorf("failed to create default workspace: %w", err)
	}

	// Create .env file with GitHub token in default workspace
	if githubToken != "" {
		envPath := filepath.Join(projectDir, "workspaces", defaultWorkspaceID, ".env")
		envContent := fmt.Sprintf("GH_TOKEN=%s\n", githubToken)
		if err := os.WriteFile(envPath, []byte(envContent), 0o600); err != nil {
			return nil, fmt.Errorf("failed to create .env file: %w", err)
		}
	}

	// Initialize git repository in default workspace if requested
	if req.InitGit {
		workspaceDir := filepath.Join(projectDir, "workspaces", defaultWorkspaceID)

		cmd := exec.Command("git", "init")
		cmd.Dir = workspaceDir
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("failed to initialize git: %w", err)
		}

		if req.RemoteURL != "" {
			cmd = exec.Command("git", "remote", "add", "origin", req.RemoteURL)
			cmd.Dir = workspaceDir
			if err := cmd.Run(); err != nil {
				return nil, fmt.Errorf("failed to set git remote: %w", err)
			}
		}

		gitignoreContent := `.env
.venv
node_modules/
__pycache__/
*.pyc
.DS_Store
`
		gitignorePath := filepath.Join(workspaceDir, ".gitignore")
		if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0o644); err != nil {
			return nil, fmt.Errorf("failed to create .gitignore: %w", err)
		}
	}

	return project, nil
}

// Get retrieves project information by UUID
func (m *Manager) Get(projectID string) (*Project, error) {
	if err := validation.ValidateProjectID(projectID); err != nil {
		return nil, err
	}

	// Acquire read lock for this project
	m.projectLocks.RLock(projectID)
	defer m.projectLocks.RUnlock(projectID)

	metadataPath := filepath.Join(m.projectsDir, projectID, "metadata.json")

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("project %s not found", projectID)
		}
		return nil, fmt.Errorf("failed to read project metadata: %w", err)
	}

	var project Project
	if err := json.Unmarshal(data, &project); err != nil {
		return nil, fmt.Errorf("failed to parse project metadata: %w", err)
	}

	// Check if custom Dockerfile exists
	dockerfilePath := filepath.Join(m.projectsDir, projectID, "Dockerfile")
	if _, err := os.Stat(dockerfilePath); err == nil {
		project.HasDockerfile = true
		project.ImageName = fmt.Sprintf("oubliette-%s:latest", projectID[:8])
	}

	return &project, nil
}

// List returns all projects with optional filtering
func (m *Manager) List(filter *ListProjectsFilter) ([]*Project, error) {
	entries, err := os.ReadDir(m.projectsDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []*Project{}, nil
		}
		return nil, fmt.Errorf("failed to read projects directory: %w", err)
	}

	var projects []*Project
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip non-UUID directories (e.g., .index.json or other files)
		if len(entry.Name()) != 36 {
			continue
		}

		project, err := m.Get(entry.Name())
		if err != nil {
			// Skip projects with invalid metadata
			continue
		}

		// Apply name filter if specified
		if filter != nil && filter.NameContains != "" {
			if !strings.Contains(strings.ToLower(project.Name), strings.ToLower(filter.NameContains)) {
				continue
			}
		}

		projects = append(projects, project)

		// Apply limit if specified
		if filter != nil && filter.Limit > 0 && len(projects) >= filter.Limit {
			break
		}
	}

	return projects, nil
}

// Delete removes a project and all its data
// Returns error if there are active sessions for the project
func (m *Manager) Delete(projectID string) error {
	if err := validation.ValidateProjectID(projectID); err != nil {
		return err
	}

	// Check for active sessions before deletion
	if m.sessionChecker != nil && m.sessionChecker.HasActiveSessionsForProject(projectID) {
		return fmt.Errorf("cannot delete project %s: has active sessions", projectID)
	}

	// Acquire write lock for this project
	m.projectLocks.Lock(projectID)
	defer m.projectLocks.Unlock(projectID)

	projectDir := filepath.Join(m.projectsDir, projectID)

	if _, err := os.Stat(projectDir); errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("project %s not found", projectID)
	}

	return os.RemoveAll(projectDir)
	// Note: We intentionally don't call m.projectLocks.Delete(projectID) here
	// because the deferred Unlock would try to unlock a non-existent lock.
	// The sync.Map entry is small and will be garbage collected if the
	// projectID is reused.
}

// GetWorkspaceDir returns the default workspace directory path for a project
func (m *Manager) GetWorkspaceDir(projectID string) string {
	project, err := m.Get(projectID)
	if err != nil {
		// Fallback - shouldn't happen
		return filepath.Join(m.projectsDir, projectID, "workspaces")
	}
	return filepath.Join(m.projectsDir, projectID, "workspaces", project.DefaultWorkspaceID)
}

// GetWorkspacePath returns the path for a specific workspace
func (m *Manager) GetWorkspacePath(projectID, workspaceID string) string {
	return filepath.Join(m.projectsDir, projectID, "workspaces", workspaceID)
}

// CreateWorkspace creates a new workspace with the given ID
func (m *Manager) CreateWorkspace(projectID, workspaceID, externalID, source string) (*WorkspaceMetadata, error) {
	if err := validation.ValidateProjectID(projectID); err != nil {
		return nil, err
	}

	if workspaceID == "" {
		workspaceID = uuid.New().String()
	} else if err := validation.ValidateWorkspaceID(workspaceID); err != nil {
		return nil, err
	}

	workspaceDir := filepath.Join(m.projectsDir, projectID, "workspaces", workspaceID)

	// Check if workspace already exists
	if _, err := os.Stat(workspaceDir); err == nil {
		// Workspace exists, return existing metadata
		return m.GetWorkspaceMetadata(projectID, workspaceID)
	}

	// Create workspace directories
	dirs := []string{
		workspaceDir,
		filepath.Join(workspaceDir, ".rlm-context"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create workspace directory %s: %w", dir, err)
		}
	}

	// Copy project AGENTS.md to workspace when isolation enabled
	// (agent won't be able to access project root in isolated mode)
	proj, err := m.Get(projectID)
	if err == nil && proj.WorkspaceIsolation {
		projectAgentsFile := filepath.Join(m.projectsDir, projectID, "AGENTS.md")
		workspaceAgentsFile := filepath.Join(workspaceDir, "AGENTS.md")
		// Only copy if project has AGENTS.md and workspace doesn't already have one
		if _, err := os.Stat(projectAgentsFile); err == nil {
			if _, err := os.Stat(workspaceAgentsFile); errors.Is(err, fs.ErrNotExist) {
				if err := m.copyFile(projectAgentsFile, workspaceAgentsFile); err != nil {
					// Log but don't fail - AGENTS.md is optional
					fmt.Printf("Warning: failed to copy AGENTS.md to workspace: %v\n", err)
				}
			}
		}
	}

	// Copy project openspec/ to workspace if it exists (for OpenSpec spec-driven workflow)
	projectOpenspecDir := filepath.Join(m.projectsDir, projectID, "openspec")
	workspaceOpenspecDir := filepath.Join(workspaceDir, "openspec")
	if _, err := os.Stat(projectOpenspecDir); err == nil {
		if err := m.copyDir(projectOpenspecDir, workspaceOpenspecDir); err != nil {
			// Log but don't fail - openspec is optional
			fmt.Printf("Warning: failed to copy openspec to workspace: %v\n", err)
		}
	}

	// Create workspace metadata
	metadata := &WorkspaceMetadata{
		ID:         workspaceID,
		CreatedAt:  time.Now(),
		ExternalID: externalID,
		Source:     source,
	}

	if err := m.saveWorkspaceMetadata(projectID, metadata); err != nil {
		return nil, fmt.Errorf("failed to save workspace metadata: %w", err)
	}

	return metadata, nil
}

// GetWorkspaceMetadata retrieves metadata for a workspace
func (m *Manager) GetWorkspaceMetadata(projectID, workspaceID string) (*WorkspaceMetadata, error) {
	if err := validation.ValidateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validation.ValidateWorkspaceID(workspaceID); err != nil {
		return nil, err
	}

	metadataPath := filepath.Join(m.projectsDir, projectID, "workspaces", workspaceID, "metadata.json")

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("workspace %s not found", workspaceID)
		}
		return nil, fmt.Errorf("failed to read workspace metadata: %w", err)
	}

	var metadata WorkspaceMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse workspace metadata: %w", err)
	}

	return &metadata, nil
}

// saveWorkspaceMetadata persists workspace metadata to disk
// Uses project lock since workspaces are scoped to projects
func (m *Manager) saveWorkspaceMetadata(projectID string, metadata *WorkspaceMetadata) error {
	// Acquire write lock for this project
	m.projectLocks.Lock(projectID)
	defer m.projectLocks.Unlock(projectID)

	metadataPath := filepath.Join(m.projectsDir, projectID, "workspaces", metadata.ID, "metadata.json")

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal workspace metadata: %w", err)
	}

	// Use atomic write pattern
	tmpPath := metadataPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write workspace metadata: %w", err)
	}

	if err := os.Rename(tmpPath, metadataPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to rename workspace metadata: %w", err)
	}

	return nil
}

// UpdateWorkspaceLastSession updates the last_session_at timestamp
func (m *Manager) UpdateWorkspaceLastSession(projectID, workspaceID string) error {
	metadata, err := m.GetWorkspaceMetadata(projectID, workspaceID)
	if err != nil {
		return err
	}

	metadata.LastSessionAt = time.Now()
	return m.saveWorkspaceMetadata(projectID, metadata)
}

// ListWorkspaces returns all workspace metadata for a project
func (m *Manager) ListWorkspaces(projectID string) ([]*WorkspaceMetadata, error) {
	workspacesDir := filepath.Join(m.projectsDir, projectID, "workspaces")
	entries, err := os.ReadDir(workspacesDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []*WorkspaceMetadata{}, nil
		}
		return nil, fmt.Errorf("failed to read workspaces directory: %w", err)
	}

	var workspaces []*WorkspaceMetadata
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		metadata, err := m.GetWorkspaceMetadata(projectID, e.Name())
		if err != nil {
			// Skip workspaces with invalid metadata
			continue
		}
		workspaces = append(workspaces, metadata)
	}
	return workspaces, nil
}

// DeleteWorkspace removes a workspace and all its data
// Returns error if there are active sessions using this workspace
func (m *Manager) DeleteWorkspace(projectID, workspaceID string) error {
	if err := validation.ValidateProjectID(projectID); err != nil {
		return err
	}
	if err := validation.ValidateWorkspaceID(workspaceID); err != nil {
		return err
	}

	// Check for active sessions before deletion
	if m.sessionChecker != nil && m.sessionChecker.HasActiveSessionsForWorkspace(projectID, workspaceID) {
		return fmt.Errorf("cannot delete workspace %s: has active sessions", workspaceID)
	}

	// Cannot delete default workspace
	project, err := m.Get(projectID)
	if err != nil {
		return err
	}
	if workspaceID == project.DefaultWorkspaceID {
		return fmt.Errorf("cannot delete default workspace")
	}

	workspacePath := m.GetWorkspacePath(projectID, workspaceID)
	if _, err := os.Stat(workspacePath); errors.Is(err, fs.ErrNotExist) {
		// Idempotent - already deleted
		return nil
	}

	return os.RemoveAll(workspacePath)
}

// WorkspaceExists checks if a workspace exists
func (m *Manager) WorkspaceExists(projectID, workspaceID string) bool {
	workspacePath := m.GetWorkspacePath(projectID, workspaceID)
	_, err := os.Stat(workspacePath)
	return err == nil
}

// GetProjectDir returns the root project directory
func (m *Manager) GetProjectDir(projectID string) string {
	return filepath.Join(m.projectsDir, projectID)
}

// GetSessionsDir returns the sessions directory for a project
func (m *Manager) GetSessionsDir(projectID string) string {
	return filepath.Join(m.projectsDir, projectID, "sessions")
}

// saveMetadata persists project metadata to disk
// Note: Caller should hold the project lock
func (m *Manager) saveMetadata(project *Project) error {
	metadataPath := filepath.Join(m.projectsDir, project.ID, "metadata.json")

	data, err := json.MarshalIndent(project, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal project metadata: %w", err)
	}

	// Use atomic write pattern: write to temp file, then rename
	tmpPath := metadataPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	if err := os.Rename(tmpPath, metadataPath); err != nil {
		_ = os.Remove(tmpPath) // Clean up on failure
		return fmt.Errorf("failed to rename metadata file: %w", err)
	}

	return nil
}

// copyDir recursively copies a directory tree
func (m *Manager) copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		// Target path
		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			// Create directory
			return os.MkdirAll(targetPath, info.Mode())
		}

		// Copy file
		return m.copyFile(path, targetPath)
	})
}

// copyFile copies a single file
func (m *Manager) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = dstFile.Close() }()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// Preserve permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

// buildCanonicalConfig creates a ProjectConfig from a CreateProjectRequest and defaults
func (m *Manager) buildCanonicalConfig(projectID, defaultWorkspaceID string, req CreateProjectRequest, createdAt time.Time) *agentconfig.ProjectConfig {
	// Get defaults
	defaults := m.getConfigDefaults()

	// Determine container type
	containerType := req.ContainerType
	if containerType == "" {
		containerType = defaults.Container.Type
	}

	// Determine model
	modelShorthand := req.Model
	if modelShorthand == "" {
		modelShorthand = defaults.Agent.Model
	}
	// Resolve shorthand model name to full model ID and look up metadata
	model := modelShorthand
	var modelDef *config.ModelDefinition
	if m.modelRegistry != nil {
		model = m.modelRegistry.ResolveModel(modelShorthand)
		if def, ok := m.modelRegistry.GetModel(modelShorthand); ok {
			modelDef = &def
		}
	}

	// Determine autonomy
	autonomy := req.Autonomy
	if autonomy == "" {
		autonomy = defaults.Agent.Autonomy
	}

	// Determine reasoning
	reasoning := req.Reasoning
	if reasoning == "" {
		reasoning = defaults.Agent.Reasoning
	}

	// Build MCP servers from defaults + request
	mcpServers := make(map[string]agentconfig.MCPServer)
	for name, srv := range defaults.Agent.MCPServers {
		mcpServers[name] = agentconfig.MCPServer{
			Type:    srv.Type,
			Command: srv.Command,
			Args:    srv.Args,
			URL:     srv.URL,
		}
	}
	for name, srv := range req.MCPServers {
		mcpServers[name] = srv
	}

	// Build limits
	maxDepth := defaults.Limits.MaxRecursionDepth
	if req.MaxRecursionDepth != nil {
		maxDepth = *req.MaxRecursionDepth
	}
	maxAgents := defaults.Limits.MaxAgentsPerSession
	if req.MaxAgentsPerSession != nil {
		maxAgents = *req.MaxAgentsPerSession
	}
	maxCost := defaults.Limits.MaxCostUSD
	if req.MaxCostUSD != nil {
		maxCost = *req.MaxCostUSD
	}

	return &agentconfig.ProjectConfig{
		ID:                 projectID,
		Name:               req.Name,
		Description:        req.Description,
		CreatedAt:          createdAt,
		DefaultWorkspaceID: defaultWorkspaceID,
		Container: agentconfig.ContainerConfig{
			Type:      containerType,
			ImageName: m.GetImageNameForType(containerType),
		},
		Agent: agentconfig.AgentConfig{
			Model:         model,
			Autonomy:      autonomy,
			Reasoning:     reasoning,
			DisabledTools: req.DisabledTools,
			MCPServers:    mcpServers,
			Permissions:   req.Permissions,
			ModelProvider: modelDefField(modelDef, func(d *config.ModelDefinition) string { return d.Provider }),
			ModelDisplay:  modelDefField(modelDef, func(d *config.ModelDefinition) string { return d.DisplayName }),
			ModelBaseURL:  modelDefField(modelDef, func(d *config.ModelDefinition) string { return d.BaseURL }),
			ModelMaxOut:   modelDefInt(modelDef, func(d *config.ModelDefinition) int { return d.MaxOutputTokens }),
			ExtraHeaders:  modelDefHeaders(modelDef),
		},
		Limits: agentconfig.LimitsConfig{
			MaxRecursionDepth:   maxDepth,
			MaxAgentsPerSession: maxAgents,
			MaxCostUSD:          maxCost,
		},
		WorkspaceIsolation: req.WorkspaceIsolation,
		ProtectedPaths:     req.ProtectedPaths,
		CredentialRefs:     convertCredentialRefs(req.CredentialRefs),
	}
}

// convertCredentialRefs converts project.CredentialRefs to agentconfig.CredentialRefs
func convertCredentialRefs(refs *CredentialRefs) *agentconfig.CredentialRefs {
	if refs == nil {
		return nil
	}
	return &agentconfig.CredentialRefs{
		GitHub:   refs.GitHub,
		Provider: refs.Provider,
	}
}

// getConfigDefaults returns config defaults, using coded defaults if not set
func (m *Manager) getConfigDefaults() config.ConfigDefaultsConfig {
	if m.configDefaults != nil {
		return *m.configDefaults
	}
	return config.DefaultConfigDefaults()
}

// saveCanonicalConfig saves the canonical config to projects/<id>/config.json
func (m *Manager) saveCanonicalConfig(cfg *agentconfig.ProjectConfig) error {
	configPath := filepath.Join(m.projectsDir, cfg.ID, "config.json")

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Use atomic write pattern
	tmpPath := configPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	if err := os.Rename(tmpPath, configPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to rename config: %w", err)
	}

	return nil
}

// generateRuntimeConfigs generates OpenCode runtime config from canonical config
func (m *Manager) generateRuntimeConfigs(cfg *agentconfig.ProjectConfig) error {
	projectDir := filepath.Join(m.projectsDir, cfg.ID)

	// Ensure .opencode directories exist
	opencodeDirs := []string{
		filepath.Join(projectDir, ".opencode"),
		filepath.Join(projectDir, ".opencode", "agents"),
		filepath.Join(projectDir, ".opencode", "skills"),
	}
	for _, dir := range opencodeDirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create .opencode directory %s: %w", dir, err)
		}
	}

	// Generate OpenCode config
	openCodeCfg := agentconfig.ToOpenCodeConfig(&cfg.Agent)
	if err := m.writeJSON(filepath.Join(projectDir, "opencode.json"), openCodeCfg); err != nil {
		return fmt.Errorf("failed to write opencode config: %w", err)
	}

	return nil
}

func modelDefField(def *config.ModelDefinition, f func(*config.ModelDefinition) string) string {
	if def == nil {
		return ""
	}
	return f(def)
}

func modelDefInt(def *config.ModelDefinition, f func(*config.ModelDefinition) int) int {
	if def == nil {
		return 0
	}
	return f(def)
}

func modelDefHeaders(def *config.ModelDefinition) map[string]string {
	if def == nil || len(def.ExtraHeaders) == 0 {
		return nil
	}
	return def.ExtraHeaders
}

// writeJSON writes data as formatted JSON to a file atomically
func (m *Manager) writeJSON(path string, data any) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Atomic write
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, jsonData, 0o644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}
