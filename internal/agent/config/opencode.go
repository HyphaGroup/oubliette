package config

import "strings"

// OpenCodeConfig is the opencode.json format for OpenCode runtime
type OpenCodeConfig struct {
	Schema     string                    `json:"$schema,omitempty"`
	Model      string                    `json:"model"`
	Permission any                       `json:"permission"` // string or map
	Tools      map[string]bool           `json:"tools,omitempty"`
	MCP        map[string]OpenCodeMCP    `json:"mcp"`
	Provider   map[string]ProviderConfig `json:"provider,omitempty"`
}

// OpenCodeMCP is a single MCP server in OpenCode format
type OpenCodeMCP struct {
	Type        string            `json:"type"`                  // local, remote
	Command     []string          `json:"command,omitempty"`     // array for local (command + args combined)
	URL         string            `json:"url,omitempty"`         // for remote
	Headers     map[string]string `json:"headers,omitempty"`     // for remote
	Environment map[string]string `json:"environment,omitempty"` // env vars (not "env")
	Enabled     bool              `json:"enabled"`
}

// ProviderConfig is provider-specific configuration for OpenCode
type ProviderConfig struct {
	Models map[string]ModelOptions `json:"models,omitempty"`
}

// ModelOptions are model-specific options (e.g., thinking, reasoningEffort)
type ModelOptions struct {
	Options map[string]any    `json:"options,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// ToOpenCodeConfig converts canonical agent config to OpenCode format
func ToOpenCodeConfig(cfg *AgentConfig) *OpenCodeConfig {
	// Translate MCP servers
	mcp := make(map[string]OpenCodeMCP)
	for name, srv := range cfg.MCPServers {
		ocMCP := OpenCodeMCP{
			Enabled: !srv.Disabled,
		}
		switch srv.Type {
		case "stdio":
			ocMCP.Type = "local"
			// Combine command + args into array
			ocMCP.Command = append([]string{srv.Command}, srv.Args...)
			ocMCP.Environment = srv.Env
		case "http":
			ocMCP.Type = "remote"
			ocMCP.URL = srv.URL
			ocMCP.Headers = srv.Headers
		}
		mcp[name] = ocMCP
	}

	// Translate model (add provider prefix)
	model := translateModelToOpenCodeFormat(cfg.Model)

	// Translate autonomy to permissions
	permission := translateAutonomyToPermissions(cfg.Autonomy, cfg.Permissions)

	// Translate disabled tools
	var tools map[string]bool
	if len(cfg.DisabledTools) > 0 {
		tools = make(map[string]bool)
		for _, t := range cfg.DisabledTools {
			tools[t] = false
		}
	}

	config := &OpenCodeConfig{
		Schema:     "https://opencode.ai/config.json",
		Model:      model,
		Permission: permission,
		Tools:      tools,
		MCP:        mcp,
	}

	// Merge extra headers into provider model config (reasoning is handled per-message via variant)
	if len(cfg.ExtraHeaders) > 0 {
		provider, modelName := splitProviderModel(model)
		if provider != "" {
			if config.Provider == nil {
				config.Provider = make(map[string]ProviderConfig)
			}
			pc := config.Provider[provider]
			if pc.Models == nil {
				pc.Models = make(map[string]ModelOptions)
			}
			mo := pc.Models[modelName]
			mo.Headers = cfg.ExtraHeaders
			pc.Models[modelName] = mo
			config.Provider[provider] = pc
		}
	}

	return config
}

// splitProviderModel splits "anthropic/claude-opus-4-6" into provider and model name
func splitProviderModel(model string) (string, string) {
	if idx := strings.Index(model, "/"); idx != -1 {
		return model[:idx], model[idx+1:]
	}
	switch {
	case strings.HasPrefix(model, "claude-"):
		return "anthropic", model
	case strings.HasPrefix(model, "gpt-"):
		return "openai", model
	case strings.HasPrefix(model, "gemini-"):
		return "google", model
	}
	return "", model
}

// translateModelToOpenCodeFormat adds provider prefix to model ID
func translateModelToOpenCodeFormat(model string) string {
	// Already has provider prefix
	if strings.Contains(model, "/") {
		return model
	}

	switch {
	case strings.HasPrefix(model, "claude-"):
		return "anthropic/" + model
	case strings.HasPrefix(model, "gpt-"):
		return "openai/" + model
	case strings.HasPrefix(model, "gemini-"):
		return "google/" + model
	default:
		return model
	}
}

// translateAutonomyToPermissions converts autonomy level to OpenCode permissions
func translateAutonomyToPermissions(autonomy string, custom map[string]any) any {
	if len(custom) > 0 {
		return custom // Use custom permissions if provided
	}

	switch autonomy {
	case "off":
		return "allow" // String "allow" = unrestricted
	case "high":
		return map[string]any{
			"*":                  "allow",
			"external_directory": "ask",
			"doom_loop":          "ask",
		}
	case "medium":
		return map[string]any{
			"read": "allow",
			"edit": "allow",
			"bash": map[string]string{
				"*":     "ask",
				"git *": "allow",
			},
		}
	case "low":
		return map[string]any{
			"read": "allow",
			"edit": "ask",
			"bash": "ask",
		}
	default:
		return "allow"
	}
}
