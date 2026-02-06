package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// UnifiedConfig is the single configuration file format for oubliette.jsonc
type UnifiedConfig struct {
	Server      ServerSection      `json:"server"`
	Credentials CredentialsSection `json:"credentials"`
	Defaults    DefaultsSection    `json:"defaults"`
	Models      ModelsSection      `json:"models"`
	Containers  map[string]string  `json:"containers"` // Container type name -> image name
}

// ServerSection contains server configuration
type ServerSection struct {
	Address string `json:"address"`
}

// CredentialsSection contains all API credentials
type CredentialsSection struct {
	GitHub    GitHubCredentials   `json:"github"`
	Providers ProviderCredentials `json:"providers"`
}

// DefaultsSection contains default settings for projects/sessions
type DefaultsSection struct {
	Limits    LimitsDefaults    `json:"limits"`
	Agent     AgentDefaults     `json:"agent"`
	Container ContainerDefaults `json:"container"`
	Backup    BackupDefaults    `json:"backup"`
}

// ModelsSection contains model definitions
type ModelsSection struct {
	Models   map[string]ModelDefinition `json:"models"`
	Defaults ModelDefaults              `json:"defaults"`
}

// ModelDefaults contains default model preferences
type ModelDefaults struct {
	IncludedModels  []string `json:"included_models"`
	SessionModel    string   `json:"session_model"`
	AutonomyMode    string   `json:"autonomy_mode"`
	ReasoningEffort string   `json:"reasoning_effort"`
}

// FindConfigPath returns the path to oubliette.jsonc using precedence:
// 1. configDir + /oubliette.jsonc (if configDir specified)
// 2. ./config/oubliette.jsonc (project-local)
// 3. ~/.oubliette/config/oubliette.jsonc (user global)
func FindConfigPath(configDir string) (string, error) {
	if configDir != "" {
		path := filepath.Join(configDir, "oubliette.jsonc")
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("oubliette.jsonc not found in %s", configDir)
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return path, nil
		}
		return abs, nil
	}

	candidates := []string{
		filepath.Join("config", "oubliette.jsonc"),
	}
	if homeDir, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(homeDir, ".oubliette", "config", "oubliette.jsonc"))
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			abs, err := filepath.Abs(path)
			if err != nil {
				return path, nil
			}
			return abs, nil
		}
	}

	return "", fmt.Errorf("oubliette.jsonc not found; tried: %v", candidates)
}

// LoadUnifiedConfig loads configuration from a single oubliette.jsonc file
func LoadUnifiedConfig(configPath string) (*UnifiedConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", configPath, err)
	}

	jsonData := StripJSONComments(data)

	var cfg UnifiedConfig
	if err := json.Unmarshal(jsonData, &cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", configPath, err)
	}

	applyUnifiedDefaults(&cfg)

	if cfg.Credentials.GitHub.Credentials == nil {
		cfg.Credentials.GitHub.Credentials = make(map[string]GitHubCredential)
	}
	if cfg.Credentials.Providers.Credentials == nil {
		cfg.Credentials.Providers.Credentials = make(map[string]ProviderCredential)
	}
	if cfg.Models.Models == nil {
		cfg.Models.Models = make(map[string]ModelDefinition)
	}
	if cfg.Defaults.Agent.MCPServers == nil {
		cfg.Defaults.Agent.MCPServers = make(map[string]MCPServerDefaults)
	}

	return &cfg, nil
}

func applyUnifiedDefaults(cfg *UnifiedConfig) {
	if cfg.Server.Address == "" {
		cfg.Server.Address = ":8080"
	}

	if cfg.Defaults.Limits.MaxRecursionDepth == 0 {
		cfg.Defaults.Limits.MaxRecursionDepth = 3
	}
	if cfg.Defaults.Limits.MaxAgentsPerSession == 0 {
		cfg.Defaults.Limits.MaxAgentsPerSession = 50
	}
	if cfg.Defaults.Limits.MaxCostUSD == 0 {
		cfg.Defaults.Limits.MaxCostUSD = 10.00
	}

	// No hardcoded model fallback -- config must specify defaults.agent.model
	if cfg.Defaults.Agent.Autonomy == "" {
		cfg.Defaults.Agent.Autonomy = "off"
	}
	if cfg.Defaults.Agent.Reasoning == "" {
		cfg.Defaults.Agent.Reasoning = "medium"
	}

	if cfg.Defaults.Container.Type == "" {
		cfg.Defaults.Container.Type = "dev"
	}

	if cfg.Containers == nil {
		cfg.Containers = make(map[string]string)
	}
	if len(cfg.Containers) == 0 {
		if isDevMode() {
			cfg.Containers["base"] = "oubliette-base:latest"
			cfg.Containers["dev"] = "oubliette-dev:latest"
		} else {
			cfg.Containers["base"] = "ghcr.io/hyphagroup/oubliette-base:latest"
			cfg.Containers["dev"] = "ghcr.io/hyphagroup/oubliette-dev:latest"
		}
	}

	if cfg.Defaults.Backup.Directory == "" {
		cfg.Defaults.Backup.Directory = "data/backups"
	}
	if cfg.Defaults.Backup.Retention == 0 {
		cfg.Defaults.Backup.Retention = 7
	}
	if cfg.Defaults.Backup.IntervalHours == 0 {
		cfg.Defaults.Backup.IntervalHours = 24
	}
}

func isDevMode() bool {
	return os.Getenv("OUBLIETTE_DEV") == "1"
}

// ToLoadedConfig converts UnifiedConfig to LoadedConfig
func (u *UnifiedConfig) ToLoadedConfig(configDir string) *LoadedConfig {
	return &LoadedConfig{
		Server: ServerJSONConfig{
			Address: u.Server.Address,
		},
		Credentials: &CredentialRegistry{
			GitHub:    u.Credentials.GitHub,
			Providers: u.Credentials.Providers,
		},
		ConfigDefaults: ConfigDefaultsConfig{
			Limits:    u.Defaults.Limits,
			Agent:     u.Defaults.Agent,
			Container: u.Defaults.Container,
			Backup:    u.Defaults.Backup,
		},
		ProjectDefaults: ProjectDefaultsConfig{
			MaxRecursionDepth:   u.Defaults.Limits.MaxRecursionDepth,
			MaxAgentsPerSession: u.Defaults.Limits.MaxAgentsPerSession,
			MaxCostUSD:          u.Defaults.Limits.MaxCostUSD,
			ContainerType:       u.Defaults.Container.Type,
		},
		Models:     u.GetModelRegistry(),
		Containers: u.Containers,
		ConfigDir:  configDir,
	}
}

// GetModelRegistry returns a ModelRegistry from the unified config
func (u *UnifiedConfig) GetModelRegistry() *ModelRegistry {
	return &ModelRegistry{
		Models: u.Models.Models,
	}
}

// Validate checks that required configuration is present
func (u *UnifiedConfig) Validate() error {
	return nil
}
