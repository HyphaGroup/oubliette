package config

import (
	"fmt"
	"path/filepath"
)

// ServerJSONConfig holds server settings
type ServerJSONConfig struct {
	Address      string          `json:"address"`
	AgentRuntime string          `json:"agent_runtime"` // auto, droid, opencode
	Droid        DroidJSONConfig `json:"droid"`
}

type DroidJSONConfig struct {
	DefaultModel string `json:"default_model"`
}

// ConfigDefaultsConfig holds default settings for projects/sessions
type ConfigDefaultsConfig struct {
	Limits    LimitsDefaults    `json:"limits"`
	Agent     AgentDefaults     `json:"agent"`
	Container ContainerDefaults `json:"container"`
	Backup    BackupDefaults    `json:"backup"`
}

// LimitsDefaults contains default resource limits
type LimitsDefaults struct {
	MaxRecursionDepth   int     `json:"max_recursion_depth"`
	MaxAgentsPerSession int     `json:"max_agents_per_session"`
	MaxCostUSD          float64 `json:"max_cost_usd"`
}

// AgentDefaults contains default agent configuration
type AgentDefaults struct {
	Runtime    string                       `json:"runtime"`
	Model      string                       `json:"model"`
	Autonomy   string                       `json:"autonomy"`
	Reasoning  string                       `json:"reasoning"`
	MCPServers map[string]MCPServerDefaults `json:"mcp_servers"`
}

// MCPServerDefaults is an MCP server definition in defaults
type MCPServerDefaults struct {
	Type    string   `json:"type"`
	Command string   `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
	URL     string   `json:"url,omitempty"`
}

// ContainerDefaults contains default container configuration
type ContainerDefaults struct {
	Type string `json:"type"`
}

// BackupDefaults contains default backup configuration
type BackupDefaults struct {
	Enabled       bool   `json:"enabled"`
	Directory     string `json:"directory"`
	Retention     int    `json:"retention"`
	IntervalHours int    `json:"interval_hours"`
}

// ProjectDefaultsConfig is kept for backward compatibility with project manager
type ProjectDefaultsConfig struct {
	MaxRecursionDepth   int     `json:"max_recursion_depth"`
	MaxAgentsPerSession int     `json:"max_agents_per_session"`
	MaxCostUSD          float64 `json:"max_cost_usd"`
	ContainerType       string  `json:"container_type"`
}

// LoadedConfig holds all configuration loaded from oubliette.jsonc
type LoadedConfig struct {
	Server          ServerJSONConfig
	Credentials     *CredentialRegistry
	ConfigDefaults  ConfigDefaultsConfig
	ProjectDefaults ProjectDefaultsConfig
	Models          *ModelRegistry
	Containers      map[string]string // Container type name -> image name
	ConfigDir       string
}

// DefaultConfigDefaults returns default configuration values
func DefaultConfigDefaults() ConfigDefaultsConfig {
	return ConfigDefaultsConfig{
		Limits: LimitsDefaults{
			MaxRecursionDepth:   3,
			MaxAgentsPerSession: 50,
			MaxCostUSD:          10.00,
		},
		Agent: AgentDefaults{
			Runtime:   "droid",
			Model:     "sonnet",
			Autonomy:  "off",
			Reasoning: "medium",
			MCPServers: map[string]MCPServerDefaults{
				"oubliette-parent": {
					Type:    "stdio",
					Command: "/usr/local/bin/oubliette-client",
					Args:    []string{"/mcp/relay.sock"},
				},
			},
		},
		Container: ContainerDefaults{
			Type: "dev",
		},
		Backup: BackupDefaults{
			Enabled:       false,
			Directory:     "data/backups",
			Retention:     7,
			IntervalHours: 24,
		},
	}
}

// LoadAll loads configuration from oubliette.jsonc
func LoadAll(configDir string) (*LoadedConfig, error) {
	configPath, err := FindConfigPath(configDir)
	if err != nil {
		return nil, err
	}

	unified, err := LoadUnifiedConfig(configPath)
	if err != nil {
		return nil, err
	}

	return unified.ToLoadedConfig(filepath.Dir(configPath)), nil
}

// HasFactoryAPIKey returns true if a Factory API key is configured
func (c *LoadedConfig) HasFactoryAPIKey() bool {
	key, ok := c.Credentials.GetDefaultFactoryKey()
	return ok && key != ""
}

// Validate checks that required configuration is present
func (c *LoadedConfig) Validate() error {
	if !c.HasFactoryAPIKey() {
		return fmt.Errorf("factory API key is required: add to oubliette.jsonc")
	}
	return nil
}
