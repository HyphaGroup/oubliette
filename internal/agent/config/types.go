// Package config provides canonical project configuration types and
// translation to runtime-specific formats (Droid, OpenCode).
package config

import (
	"fmt"
	"time"
)

// ProjectConfig is the canonical configuration for a project.
// Stored as projects/<id>/config.json and serves as the single source of truth.
type ProjectConfig struct {
	// Project identity
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Description        string    `json:"description,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	DefaultWorkspaceID string    `json:"default_workspace_id"`

	// Container settings
	Container ContainerConfig `json:"container"`

	// Agent runtime settings
	Agent AgentConfig `json:"agent"`

	// Resource limits
	Limits LimitsConfig `json:"limits"`

	// Isolation settings
	WorkspaceIsolation bool     `json:"workspace_isolation,omitempty"`
	ProtectedPaths     []string `json:"protected_paths,omitempty"`

	// Credential references (names, not values)
	CredentialRefs *CredentialRefs `json:"credential_refs,omitempty"`
}

// CredentialRefs specifies which credentials to use for a project
type CredentialRefs struct {
	Factory  string `json:"factory,omitempty"`
	GitHub   string `json:"github,omitempty"`
	Provider string `json:"provider,omitempty"`
}

// ContainerConfig defines container settings
type ContainerConfig struct {
	Type          string `json:"type"`                     // base, dev, osint
	ImageName     string `json:"image_name"`               // e.g., oubliette-dev:latest
	HasDockerfile bool   `json:"has_dockerfile,omitempty"` // true if custom Dockerfile exists
	Status        string `json:"status,omitempty"`         // runtime state
	ID            string `json:"id,omitempty"`             // runtime container ID
}

// AgentConfig defines agent runtime settings
type AgentConfig struct {
	Runtime       string               `json:"runtime"`                  // droid, opencode
	Model         string               `json:"model"`                    // e.g., claude-sonnet-4-5-20250929
	Autonomy      string               `json:"autonomy"`                 // off, low, medium, high
	Reasoning     string               `json:"reasoning,omitempty"`      // off, low, medium, high
	DisabledTools []string             `json:"disabled_tools,omitempty"` // tools to disable
	MCPServers    map[string]MCPServer `json:"mcp_servers"`              // MCP server definitions
	Permissions   map[string]any       `json:"permissions,omitempty"`    // OpenCode-style permissions override
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

// Valid autonomy levels
var ValidAutonomyLevels = []string{"off", "low", "medium", "high"}

// Valid reasoning levels
var ValidReasoningLevels = []string{"off", "low", "medium", "high"}

// Validate checks that required fields are present and valid
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
	if c.Agent.Runtime == "" {
		return fmt.Errorf("agent.runtime is required")
	}
	if c.Agent.Model == "" {
		return fmt.Errorf("agent.model is required")
	}

	// Validate autonomy level
	if !isValidLevel(c.Agent.Autonomy, ValidAutonomyLevels) {
		return fmt.Errorf("invalid autonomy level %q, must be one of: %v", c.Agent.Autonomy, ValidAutonomyLevels)
	}

	// Validate reasoning level if set
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
