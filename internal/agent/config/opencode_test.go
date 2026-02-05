package config

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestToOpenCodeConfig(t *testing.T) {
	cfg := &AgentConfig{
		Runtime:   "opencode",
		Model:     "claude-sonnet-4-5-20250929",
		Autonomy:  "high",
		Reasoning: "medium",
		MCPServers: map[string]MCPServer{
			"oubliette-parent": {
				Type:    "stdio",
				Command: "/usr/local/bin/oubliette-client",
				Args:    []string{"/mcp/relay.sock"},
				Env:     map[string]string{"DEBUG": "1"},
			},
			"remote-api": {
				Type:    "http",
				URL:     "https://api.example.com/mcp",
				Headers: map[string]string{"Authorization": "Bearer token"},
			},
		},
		DisabledTools: []string{"dangerous_tool"},
	}

	result := ToOpenCodeConfig(cfg)

	// Check model translation
	if result.Model != "anthropic/claude-sonnet-4-5-20250929" {
		t.Errorf("model: got %q, want %q", result.Model, "anthropic/claude-sonnet-4-5-20250929")
	}

	// Check stdio -> local translation
	local, ok := result.MCP["oubliette-parent"]
	if !ok {
		t.Fatal("missing oubliette-parent server")
	}
	if local.Type != "local" {
		t.Errorf("type: got %q, want %q", local.Type, "local")
	}
	expectedCmd := []string{"/usr/local/bin/oubliette-client", "/mcp/relay.sock"}
	if !reflect.DeepEqual(local.Command, expectedCmd) {
		t.Errorf("command: got %v, want %v", local.Command, expectedCmd)
	}
	if local.Environment["DEBUG"] != "1" {
		t.Error("environment not translated")
	}
	if !local.Enabled {
		t.Error("expected enabled to be true")
	}

	// Check http -> remote translation
	remote, ok := result.MCP["remote-api"]
	if !ok {
		t.Fatal("missing remote-api server")
	}
	if remote.Type != "remote" {
		t.Errorf("type: got %q, want %q", remote.Type, "remote")
	}
	if remote.URL != "https://api.example.com/mcp" {
		t.Error("url not set")
	}
	if remote.Headers["Authorization"] != "Bearer token" {
		t.Error("headers not translated")
	}

	// Check disabled tools
	if !result.Tools["dangerous_tool"] == false {
		t.Error("disabled tool should be false")
	}
}

func TestTranslateModelToOpenCodeFormat(t *testing.T) {
	tests := []struct {
		model string
		want  string
	}{
		{"claude-sonnet-4-5-20250929", "anthropic/claude-sonnet-4-5-20250929"},
		{"claude-opus-4-5", "anthropic/claude-opus-4-5"},
		{"gpt-5.1", "openai/gpt-5.1"},
		{"gpt-5.1-codex", "openai/gpt-5.1-codex"},
		{"gemini-3-pro", "google/gemini-3-pro"},
		{"anthropic/claude-sonnet", "anthropic/claude-sonnet"}, // already has prefix
		{"custom-model", "custom-model"},                       // unknown, unchanged
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := translateModelToOpenCodeFormat(tt.model)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTranslateAutonomyToPermissions(t *testing.T) {
	tests := []struct {
		name     string
		autonomy string
		custom   map[string]any
		wantType string // "string" or "map"
	}{
		{"off returns allow string", "off", nil, "string"},
		{"high returns map", "high", nil, "map"},
		{"medium returns map", "medium", nil, "map"},
		{"low returns map", "low", nil, "map"},
		{"custom overrides", "high", map[string]any{"custom": "value"}, "map"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := translateAutonomyToPermissions(tt.autonomy, tt.custom)

			switch tt.wantType {
			case "string":
				if s, ok := result.(string); !ok || s != "allow" {
					t.Errorf("expected string 'allow', got %T: %v", result, result)
				}
			case "map":
				if _, ok := result.(map[string]any); !ok {
					t.Errorf("expected map, got %T", result)
				}
			}

			// Custom should be returned as-is
			if tt.custom != nil {
				if m, ok := result.(map[string]any); ok {
					if m["custom"] != "value" {
						t.Error("custom permissions not preserved")
					}
				}
			}
		})
	}
}

func TestTranslateReasoningToOpenCode(t *testing.T) {
	tests := []struct {
		name      string
		reasoning string
		model     string
		wantNil   bool
		checkFn   func(t *testing.T, result map[string]ProviderConfig)
	}{
		{
			name:      "anthropic medium",
			reasoning: "medium",
			model:     "anthropic/claude-sonnet-4-5-20250929",
			checkFn: func(t *testing.T, result map[string]ProviderConfig) {
				pc, ok := result["anthropic"]
				if !ok {
					t.Fatal("missing anthropic provider")
				}
				mo, ok := pc.Models["claude-sonnet-4-5-20250929"]
				if !ok {
					t.Fatal("missing model options")
				}
				thinking, ok := mo.Options["thinking"].(map[string]any)
				if !ok {
					t.Fatal("missing thinking config")
				}
				if thinking["budgetTokens"] != 16000 {
					t.Errorf("budgetTokens: got %v, want 16000", thinking["budgetTokens"])
				}
			},
		},
		{
			name:      "anthropic high",
			reasoning: "high",
			model:     "claude-opus-4-5",
			checkFn: func(t *testing.T, result map[string]ProviderConfig) {
				thinking := result["anthropic"].Models["claude-opus-4-5"].Options["thinking"].(map[string]any)
				if thinking["budgetTokens"] != 32000 {
					t.Errorf("budgetTokens: got %v, want 32000", thinking["budgetTokens"])
				}
			},
		},
		{
			name:      "openai medium",
			reasoning: "medium",
			model:     "openai/gpt-5.1",
			checkFn: func(t *testing.T, result map[string]ProviderConfig) {
				effort := result["openai"].Models["gpt-5.1"].Options["reasoningEffort"]
				if effort != "medium" {
					t.Errorf("reasoningEffort: got %v, want medium", effort)
				}
			},
		},
		{
			name:      "google high",
			reasoning: "high",
			model:     "gemini-3-pro",
			checkFn: func(t *testing.T, result map[string]ProviderConfig) {
				variant := result["google"].Models["gemini-3-pro"].Options["variant"]
				if variant != "high" {
					t.Errorf("variant: got %v, want high", variant)
				}
			},
		},
		{
			name:      "off returns nil",
			reasoning: "off",
			model:     "claude-sonnet",
			wantNil:   true,
		},
		{
			name:      "unknown provider returns nil",
			reasoning: "medium",
			model:     "unknown-model",
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := translateReasoningToOpenCode(tt.reasoning, tt.model)
			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}
			if result == nil {
				t.Fatal("unexpected nil result")
			}
			if tt.checkFn != nil {
				tt.checkFn(t, result)
			}
		})
	}
}

func TestOpenCodeConfigSerialization(t *testing.T) {
	cfg := &OpenCodeConfig{
		Schema:     "https://opencode.ai/config.json",
		Model:      "anthropic/claude-sonnet",
		Permission: "allow",
		MCP: map[string]OpenCodeMCP{
			"test": {
				Type:    "local",
				Command: []string{"/bin/test"},
				Enabled: true,
			},
		},
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	// Verify key names match OpenCode format
	s := string(data)
	if !containsString(s, `"$schema"`) {
		t.Error("missing $schema key")
	}
	if !containsString(s, `"mcp"`) {
		t.Error("missing mcp key (should not be mcpServers)")
	}
	if !containsString(s, `"permission"`) {
		t.Error("missing permission key")
	}

	// Parse back
	var parsed OpenCodeConfig
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if parsed.Model != cfg.Model {
		t.Errorf("model mismatch: got %q, want %q", parsed.Model, cfg.Model)
	}
}
