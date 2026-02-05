package config

// DroidMCPConfig is the .factory/mcp.json format for Droid runtime
type DroidMCPConfig struct {
	MCPServers map[string]DroidMCPServer `json:"mcpServers"`
}

// DroidMCPServer is a single MCP server in Droid format
type DroidMCPServer struct {
	Type     string            `json:"type"`              // stdio, http
	Command  string            `json:"command,omitempty"` // for stdio
	Args     []string          `json:"args,omitempty"`    // for stdio
	URL      string            `json:"url,omitempty"`     // for http
	Env      map[string]string `json:"env,omitempty"`
	Disabled bool              `json:"disabled,omitempty"`
}

// DroidSettings is the .factory/settings.json format for Droid runtime
type DroidSettings struct {
	SessionDefaultSettings DroidSessionDefaults `json:"sessionDefaultSettings"`
	CustomModels           []DroidCustomModel   `json:"customModels,omitempty"`
	LogoAnimation          string               `json:"logoAnimation,omitempty"`
	IncludeCoAuthoredBy    bool                 `json:"includeCoAuthoredByDroid"`
	AllowBackgroundProcs   bool                 `json:"allowBackgroundProcesses"`
	CloudSessionSync       bool                 `json:"cloudSessionSync"`
	Hooks                  map[string]any       `json:"hooks,omitempty"`
}

// DroidSessionDefaults are session default settings for Droid
type DroidSessionDefaults struct {
	AutonomyMode    string `json:"autonomyMode,omitempty"`
	Model           string `json:"model,omitempty"`
	ReasoningEffort string `json:"reasoningEffort,omitempty"`
}

// DroidCustomModel is a custom model definition for Droid
type DroidCustomModel struct {
	ID              string `json:"id"`
	Model           string `json:"model"`
	Name            string `json:"name"`
	BaseURL         string `json:"baseUrl"`
	APIKey          string `json:"apiKey"`
	MaxOutputTokens int    `json:"maxOutputTokens"`
	Provider        string `json:"provider"`
}

// ToDroidMCPConfig converts canonical MCP config to Droid format
func ToDroidMCPConfig(cfg *AgentConfig) *DroidMCPConfig {
	servers := make(map[string]DroidMCPServer)
	for name, srv := range cfg.MCPServers {
		servers[name] = DroidMCPServer{
			Type:     srv.Type, // stdio/http same in canonical and Droid
			Command:  srv.Command,
			Args:     srv.Args,
			URL:      srv.URL,
			Env:      srv.Env,
			Disabled: srv.Disabled,
		}
	}
	return &DroidMCPConfig{MCPServers: servers}
}

// ToDroidSettings converts canonical agent config to Droid settings format
func ToDroidSettings(cfg *AgentConfig) *DroidSettings {
	settings := &DroidSettings{
		SessionDefaultSettings: DroidSessionDefaults{
			Model:           cfg.Model,
			ReasoningEffort: cfg.Reasoning,
		},
		CloudSessionSync: true,
	}

	// Map autonomy to Droid autonomyMode
	switch cfg.Autonomy {
	case "off":
		settings.SessionDefaultSettings.AutonomyMode = "auto-high" // closest to unrestricted
	case "high":
		settings.SessionDefaultSettings.AutonomyMode = "auto-high"
	case "medium":
		settings.SessionDefaultSettings.AutonomyMode = "auto-medium"
	case "low":
		settings.SessionDefaultSettings.AutonomyMode = "auto-low"
	default:
		settings.SessionDefaultSettings.AutonomyMode = "auto-high"
	}

	return settings
}

// TranslateAutonomyToDroidFlags returns CLI flags for Droid autonomy level
func TranslateAutonomyToDroidFlags(autonomy string) []string {
	switch autonomy {
	case "off":
		return []string{"--skip-permissions-unsafe"}
	case "high":
		return []string{"--auto", "high"}
	case "medium":
		return []string{"--auto", "medium"}
	case "low":
		return []string{"--auto", "low"}
	default:
		return []string{"--skip-permissions-unsafe"}
	}
}

// TranslateReasoningToDroidFlags returns CLI flags for Droid reasoning level
func TranslateReasoningToDroidFlags(reasoning string) []string {
	switch reasoning {
	case "off":
		return []string{"-r", "off"}
	case "low":
		return []string{"-r", "low"}
	case "medium":
		return []string{"-r", "medium"}
	case "high":
		return []string{"-r", "high"}
	default:
		return []string{"-r", "medium"}
	}
}
