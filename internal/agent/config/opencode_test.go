package config

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestToOpenCodeConfig(t *testing.T) {
	cfg := &AgentConfig{
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

	if result.Model != "anthropic/claude-sonnet-4-5-20250929" {
		t.Errorf("model: got %q, want %q", result.Model, "anthropic/claude-sonnet-4-5-20250929")
	}

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

	remote, ok := result.MCP["remote-api"]
	if !ok {
		t.Fatal("missing remote-api server")
	}
	if remote.Type != "remote" {
		t.Errorf("type: got %q, want %q", remote.Type, "remote")
	}

	if !result.Tools["dangerous_tool"] == false {
		t.Error("disabled tool should be false")
	}

	// Reasoning should NOT be in provider config (handled per-message via variant)
	if result.Provider != nil {
		t.Error("expected no provider config (reasoning is per-message)")
	}
}

func TestToOpenCodeConfigWithExtraHeaders(t *testing.T) {
	cfg := &AgentConfig{
		Model:    "claude-opus-4-6",
		Autonomy: "off",
		ExtraHeaders: map[string]string{
			"anthropic-beta": "context-1m-2025-08-07",
		},
		MCPServers: map[string]MCPServer{},
	}

	result := ToOpenCodeConfig(cfg)

	if result.Provider == nil {
		t.Fatal("expected provider config for extra headers")
	}
	pc, ok := result.Provider["anthropic"]
	if !ok {
		t.Fatal("missing anthropic provider config")
	}
	mo, ok := pc.Models["claude-opus-4-6"]
	if !ok {
		t.Fatal("missing model options for claude-opus-4-6")
	}
	if mo.Headers["anthropic-beta"] != "context-1m-2025-08-07" {
		t.Errorf("headers: got %v, want anthropic-beta header", mo.Headers)
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if !strings.Contains(string(data), `"anthropic-beta"`) {
		t.Error("anthropic-beta header not in serialized output")
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
		{"gemini-3-pro", "google/gemini-3-pro"},
		{"anthropic/claude-sonnet", "anthropic/claude-sonnet"},
		{"custom-model", "custom-model"},
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
		wantType string
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

	s := string(data)
	if !strings.Contains(s, `"$schema"`) {
		t.Error("missing $schema key")
	}
	if !strings.Contains(s, `"mcp"`) {
		t.Error("missing mcp key")
	}

	var parsed OpenCodeConfig
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if parsed.Model != cfg.Model {
		t.Errorf("model mismatch: got %q, want %q", parsed.Model, cfg.Model)
	}
}
