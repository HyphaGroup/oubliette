// Package config provides canonical project configuration types and
// translation to OpenCode runtime format.
package config

import (
	"fmt"
	"time"
)

// ProjectConfig is the canonical configuration for a project.
// Stored as projects/<id>/config.json and serves as the single source of truth.
type ProjectConfig struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Description        string    `json:"description,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	DefaultWorkspaceID string    `json:"default_workspace_id"`

	Container          ContainerConfig `json:"container"`
	Agent              AgentConfig     `json:"agent"`
	Limits             LimitsConfig    `json:"limits"`
	WorkspaceIsolation bool            `json:"workspace_isolation,omitempty"`
	ProtectedPaths     []string        `json:"protected_paths,omitempty"`
	CredentialRefs     *CredentialRefs `json:"credential_refs,omitempty"`
}

// CredentialRefs specifies which credentials to use for a project
type CredentialRefs struct {
	GitHub string `json:"github,omitempty"`
}

// ContainerConfig defines container settings
type ContainerConfig struct {
	Type          string `json:"type"`                     // base, dev
	ImageName     string `json:"image_name"`               // e.g., oubliette-dev:latest
	HasDockerfile bool   `json:"has_dockerfile,omitempty"` // true if custom Dockerfile exists
	Status        string `json:"status,omitempty"`         // runtime state
	ID            string `json:"id,omitempty"`             // runtime container ID
}

// AgentConfig defines agent runtime settings
type AgentConfig struct {
	Model         string               `json:"model"`                    // e.g., claude-opus-4-6
	ModelProvider string               `json:"model_provider,omitempty"` // anthropic, openai, google
	ModelDisplay  string               `json:"model_display,omitempty"`  // human-friendly name
	ModelBaseURL  string               `json:"model_base_url,omitempty"` // API endpoint
	ModelMaxOut   int                  `json:"model_max_output,omitempty"`
	ExtraHeaders  map[string]string    `json:"extra_headers,omitempty"` // HTTP headers for API requests
	Autonomy      string               `json:"autonomy"`                // off, low, medium, high
	Reasoning     string               `json:"reasoning,omitempty"`     // off, low, medium, high
	DisabledTools []string             `json:"disabled_tools,omitempty"`
	MCPServers    map[string]MCPServer `json:"mcp_servers"`
	Permissions   map[string]any       `json:"permissions,omitempty"` // OpenCode-style permissions override
}

// MCPServer defines an MCP server configuration in canonical format
type MCPServer struct {
	Type     string            `json:"type"`              // stdio, http
	Command  string            `json:"command,omitempty"` // for stdio
	Args     []string          `json:"args,omitempty"`    // for stdio
	URL      string            `json:"url,omitempty"`     // for http
	Headers  map[string]string `json:"headers,omitempty"` // for http
	Env      map[string]string `json:"env,omitempty"`     // environment variables
	Disabled bool              `json:"disabled,omitempty"`
}

// LimitsConfig defines resource limits
type LimitsConfig struct {
	MaxRecursionDepth   int     `json:"max_recursion_depth"`
	MaxAgentsPerSession int     `json:"max_agents_per_session"`
	MaxCostUSD          float64 `json:"max_cost_usd"`
}

var ValidAutonomyLevels = []string{"off", "low", "medium", "high"}
var ValidReasoningLevels = []string{"off", "low", "medium", "high"}

func (c *ProjectConfig) Validate() error {
	if c.ID == "" {
		return fmt.Errorf("project id is required")
	}
	if c.Name == "" {
		return fmt.Errorf("project name is required")
	}
	if c.DefaultWorkspaceID == "" {
		return fmt.Errorf("default_workspace_id is required")
	}
	if c.Container.Type == "" {
		return fmt.Errorf("container.type is required")
	}
	if c.Agent.Model == "" {
		return fmt.Errorf("agent.model is required")
	}
	if !isValidLevel(c.Agent.Autonomy, ValidAutonomyLevels) {
		return fmt.Errorf("invalid autonomy level %q, must be one of: %v", c.Agent.Autonomy, ValidAutonomyLevels)
	}
	if c.Agent.Reasoning != "" && !isValidLevel(c.Agent.Reasoning, ValidReasoningLevels) {
		return fmt.Errorf("invalid reasoning level %q, must be one of: %v", c.Agent.Reasoning, ValidReasoningLevels)
	}
	return nil
}

func isValidLevel(level string, valid []string) bool {
	for _, v := range valid {
		if level == v {
			return true
		}
	}
	return false
}
