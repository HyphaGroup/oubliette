package project

import (
	"time"

	agentconfig "github.com/HyphaGroup/oubliette/internal/agent/config"
	"github.com/HyphaGroup/oubliette/internal/config"
)

// ModelRegistry is an alias for config.ModelRegistry
type ModelRegistry = config.ModelRegistry

// AgentMCPServer is an alias for the canonical MCP server type
type AgentMCPServer = agentconfig.MCPServer

// RecursionConfig defines limits for recursive gogol spawning
type RecursionConfig struct {
	MaxDepth   *int     `json:"max_depth,omitempty"`
	MaxAgents  *int     `json:"max_agents,omitempty"`
	MaxCostUSD *float64 `json:"max_cost_usd,omitempty"`
}

// CredentialRefs specifies which credentials to use for a project
type CredentialRefs struct {
	GitHub string `json:"github,omitempty"`
}

// Project represents a managed project
type Project struct {
	ID                 string           `json:"id"`
	Name               string           `json:"name"`
	Description        string           `json:"description"`
	DefaultWorkspaceID string           `json:"default_workspace_id"`
	CreatedAt          time.Time        `json:"created_at"`
	GitHubToken        string           `json:"-"`
	RemoteURL          string           `json:"remote_url,omitempty"`
	ImageName          string           `json:"image_name"`
	ContainerType      string           `json:"container_type"`
	Model              string           `json:"model,omitempty"`
	HasDockerfile      bool             `json:"has_dockerfile"`
	ContainerID        string           `json:"container_id,omitempty"`
	ContainerStatus    string           `json:"container_status,omitempty"`
	RecursionConfig    *RecursionConfig `json:"recursion_config,omitempty"`
	WorkspaceIsolation bool             `json:"workspace_isolation,omitempty"`
	ProtectedPaths     []string         `json:"protected_paths,omitempty"`
	CredentialRefs     *CredentialRefs  `json:"credential_refs,omitempty"`
}

// WorkspaceMetadata contains metadata about a workspace
type WorkspaceMetadata struct {
	ID            string    `json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	LastSessionAt time.Time `json:"last_session_at,omitempty"`
	ExternalID    string    `json:"external_id,omitempty"`
	Source        string    `json:"source,omitempty"`
}

// CreateProjectRequest contains parameters for creating a project
type CreateProjectRequest struct {
	Name               string
	Description        string
	GitHubToken        string
	RemoteURL          string
	InitGit            bool
	WorkspaceIsolation bool
	ProtectedPaths     []string

	MaxRecursionDepth   *int
	MaxAgentsPerSession *int
	MaxCostUSD          *float64

	Model         string
	Autonomy      string
	Reasoning     string
	DisabledTools []string
	MCPServers    map[string]AgentMCPServer
	Permissions   map[string]any

	ContainerType  string
	CredentialRefs *CredentialRefs
}

// ListProjectsFilter contains optional filters for listing projects
type ListProjectsFilter struct {
	NameContains string
	Limit        int
}

// ActiveSessionChecker is an interface for checking active sessions
type ActiveSessionChecker interface {
	HasActiveSessionsForProject(projectID string) bool
	HasActiveSessionsForWorkspace(projectID, workspaceID string) bool
}

// Manager handles project CRUD operations
type Manager struct {
	projectsDir       string
	templateDir       string
	defaultMaxDepth   int
	defaultMaxAgents  int
	defaultMaxCostUSD float64
	projectLocks      ProjectLockMap
	sessionChecker    ActiveSessionChecker
	modelRegistry     *ModelRegistry
	configDefaults    *config.ConfigDefaultsConfig
	containers        map[string]string
}
