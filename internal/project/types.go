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
	Factory  string `json:"factory,omitempty"`
	GitHub   string `json:"github,omitempty"`
	Provider string `json:"provider,omitempty"`
}

// Project represents a managed project
type Project struct {
	ID                 string           `json:"id"`   // UUID - primary key
	Name               string           `json:"name"` // Display name only
	Description        string           `json:"description"`
	DefaultWorkspaceID string           `json:"default_workspace_id"` // UUID of default workspace
	CreatedAt          time.Time        `json:"created_at"`
	GitHubToken        string           `json:"-"` // Never serialize tokens
	RemoteURL          string           `json:"remote_url,omitempty"`
	ImageName          string           `json:"image_name"`
	ContainerType      string           `json:"container_type"`          // base, dev, osint (default: dev)
	AgentRuntime       string           `json:"agent_runtime,omitempty"` // droid, opencode, or empty for server default
	Model              string           `json:"model,omitempty"`         // Model in provider/id format (e.g., "anthropic/claude-sonnet-4-5")
	HasDockerfile      bool             `json:"has_dockerfile"`
	ContainerID        string           `json:"container_id,omitempty"`
	ContainerStatus    string           `json:"container_status,omitempty"`
	RecursionConfig    *RecursionConfig `json:"recursion_config,omitempty"`
	WorkspaceIsolation bool             `json:"workspace_isolation,omitempty"` // When true, mount only workspace dir
	ProtectedPaths     []string         `json:"protected_paths,omitempty"`     // Paths mounted read-only when isolated
	CredentialRefs     *CredentialRefs  `json:"credential_refs,omitempty"`     // Named credential references
}

// WorkspaceMetadata contains metadata about a workspace
type WorkspaceMetadata struct {
	ID            string    `json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	LastSessionAt time.Time `json:"last_session_at,omitempty"`
	ExternalID    string    `json:"external_id,omitempty"` // Caller's identifier (e.g., user UUID from external system)
	Source        string    `json:"source,omitempty"`      // Which system created the workspace
}

// CreateProjectRequest contains parameters for creating a project
type CreateProjectRequest struct {
	Name               string // Display name
	Description        string
	GitHubToken        string
	RemoteURL          string
	InitGit            bool
	Languages          []string
	WorkspaceIsolation bool     // When true, mount only workspace dir in containers
	ProtectedPaths     []string // Paths mounted read-only when isolated

	// Project limits (optional, uses server defaults if not set)
	MaxRecursionDepth   *int
	MaxAgentsPerSession *int
	MaxCostUSD          *float64

	// Agent configuration
	AgentRuntime  string                    // droid, opencode, or empty for server default
	Model         string                    // Model shorthand (e.g., "sonnet") or full ID
	Autonomy      string                    // off, low, medium, high
	Reasoning     string                    // off, low, medium, high
	DisabledTools []string                  // Tools to disable
	MCPServers    map[string]AgentMCPServer // Additional MCP servers
	Permissions   map[string]any            // Custom permissions (OpenCode format)

	// Container configuration
	ContainerType string // base, dev, osint (default: dev)

	// Credential references
	CredentialRefs *CredentialRefs
}

// ListProjectsFilter contains optional filters for listing projects
type ListProjectsFilter struct {
	NameContains string // Case-insensitive substring match
	Limit        int    // Max results (0 = no limit)
}

// ActiveSessionChecker is an interface for checking active sessions
// Implemented by the MCP server to allow project manager to check for active sessions
type ActiveSessionChecker interface {
	HasActiveSessionsForProject(projectID string) bool
	HasActiveSessionsForWorkspace(projectID, workspaceID string) bool
}

// Manager handles project CRUD operations
type Manager struct {
	projectsDir string
	templateDir string // Path to template directory
	// Recursion defaults from environment (legacy - use configDefaults instead)
	defaultMaxDepth   int
	defaultMaxAgents  int
	defaultMaxCostUSD float64
	// Per-project locks to prevent concurrent metadata writes
	projectLocks ProjectLockMap
	// Optional checker for active sessions (set by MCP server)
	sessionChecker ActiveSessionChecker
	// Model configuration (optional - set via SetModelConfig)
	modelRegistry *ModelRegistry
	// Config defaults (new format with agent config)
	configDefaults *config.ConfigDefaultsConfig
	// Container type -> image name mapping (set via SetContainers)
	containers map[string]string
}
