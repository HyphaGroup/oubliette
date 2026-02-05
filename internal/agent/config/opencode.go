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
	Options map[string]any `json:"options,omitempty"`
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

	// Add provider-specific reasoning config if needed
	if cfg.Reasoning != "" && cfg.Reasoning != "off" {
		providerConfig := translateReasoningToOpenCode(cfg.Reasoning, model)
		if providerConfig != nil {
			config.Provider = providerConfig
		}
	}

	return config
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

// translateReasoningToOpenCode returns provider-specific reasoning configuration
func translateReasoningToOpenCode(reasoning, model string) map[string]ProviderConfig {
	// Extract provider from model (e.g., "anthropic/claude-sonnet..." -> "anthropic")
	provider := ""
	modelName := model
	if idx := strings.Index(model, "/"); idx != -1 {
		provider = model[:idx]
		modelName = model[idx+1:]
	} else {
		// Detect provider from model name prefix
		switch {
		case strings.HasPrefix(model, "claude-"):
			provider = "anthropic"
		case strings.HasPrefix(model, "gpt-"):
			provider = "openai"
		case strings.HasPrefix(model, "gemini-"):
			provider = "google"
		default:
			return nil // Unknown provider
		}
	}

	var options map[string]any
	switch provider {
	case "anthropic":
		// Anthropic uses thinking.budgetTokens
		switch reasoning {
		case "low":
			options = map[string]any{
				"thinking": map[string]any{
					"type":         "enabled",
					"budgetTokens": 4000,
				},
			}
		case "medium":
			options = map[string]any{
				"thinking": map[string]any{
					"type":         "enabled",
					"budgetTokens": 16000,
				},
			}
		case "high":
			options = map[string]any{
				"thinking": map[string]any{
					"type":         "enabled",
					"budgetTokens": 32000,
				},
			}
		default:
			return nil
		}
	case "openai":
		// OpenAI uses reasoningEffort
		switch reasoning {
		case "low":
			options = map[string]any{"reasoningEffort": "low"}
		case "medium":
			options = map[string]any{"reasoningEffort": "medium"}
		case "high":
			options = map[string]any{"reasoningEffort": "high"}
		default:
			return nil
		}
	case "google":
		// Google uses variant (low/high only)
		switch reasoning {
		case "low":
			options = map[string]any{"variant": "low"}
		case "medium", "high":
			options = map[string]any{"variant": "high"}
		default:
			return nil
		}
	default:
		return nil
	}

	return map[string]ProviderConfig{
		provider: {
			Models: map[string]ModelOptions{
				modelName: {Options: options},
			},
		},
	}
}
