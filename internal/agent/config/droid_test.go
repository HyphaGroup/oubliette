package config

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestToDroidMCPConfig(t *testing.T) {
	cfg := &AgentConfig{
		MCPServers: map[string]MCPServer{
			"oubliette-parent": {
				Type:    "stdio",
				Command: "/usr/local/bin/oubliette-client",
				Args:    []string{"/mcp/relay.sock"},
				Env:     map[string]string{"DEBUG": "1"},
			},
			"remote-api": {
				Type:     "http",
				URL:      "https://api.example.com/mcp",
				Disabled: true,
			},
		},
	}

	result := ToDroidMCPConfig(cfg)

	// Check stdio server
	stdio, ok := result.MCPServers["oubliette-parent"]
	if !ok {
		t.Fatal("missing oubliette-parent server")
	}
	if stdio.Type != "stdio" {
		t.Errorf("type: got %q, want %q", stdio.Type, "stdio")
	}
	if stdio.Command != "/usr/local/bin/oubliette-client" {
		t.Errorf("command mismatch")
	}
	if !reflect.DeepEqual(stdio.Args, []string{"/mcp/relay.sock"}) {
		t.Errorf("args mismatch: got %v", stdio.Args)
	}
	if stdio.Env["DEBUG"] != "1" {
		t.Errorf("env mismatch")
	}

	// Check http server
	http, ok := result.MCPServers["remote-api"]
	if !ok {
		t.Fatal("missing remote-api server")
	}
	if http.Type != "http" {
		t.Errorf("type: got %q, want %q", http.Type, "http")
	}
	if http.URL != "https://api.example.com/mcp" {
		t.Errorf("url mismatch")
	}
	if !http.Disabled {
		t.Error("expected disabled to be true")
	}
}

func TestToDroidSettings(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *AgentConfig
		wantModel string
		wantMode  string
	}{
		{
			name: "autonomy off",
			cfg: &AgentConfig{
				Model:     "claude-sonnet-4-5",
				Autonomy:  "off",
				Reasoning: "medium",
			},
			wantModel: "claude-sonnet-4-5",
			wantMode:  "auto-high",
		},
		{
			name: "autonomy high",
			cfg: &AgentConfig{
				Model:    "gpt-5.1",
				Autonomy: "high",
			},
			wantModel: "gpt-5.1",
			wantMode:  "auto-high",
		},
		{
			name: "autonomy medium",
			cfg: &AgentConfig{
				Model:    "claude-sonnet",
				Autonomy: "medium",
			},
			wantModel: "claude-sonnet",
			wantMode:  "auto-medium",
		},
		{
			name: "autonomy low",
			cfg: &AgentConfig{
				Model:    "claude-haiku",
				Autonomy: "low",
			},
			wantModel: "claude-haiku",
			wantMode:  "auto-low",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToDroidSettings(tt.cfg)
			if result.SessionDefaultSettings.Model != tt.wantModel {
				t.Errorf("model: got %q, want %q", result.SessionDefaultSettings.Model, tt.wantModel)
			}
			if result.SessionDefaultSettings.AutonomyMode != tt.wantMode {
				t.Errorf("autonomyMode: got %q, want %q", result.SessionDefaultSettings.AutonomyMode, tt.wantMode)
			}
		})
	}
}

func TestTranslateAutonomyToDroidFlags(t *testing.T) {
	tests := []struct {
		autonomy string
		want     []string
	}{
		{"off", []string{"--skip-permissions-unsafe"}},
		{"high", []string{"--auto", "high"}},
		{"medium", []string{"--auto", "medium"}},
		{"low", []string{"--auto", "low"}},
		{"unknown", []string{"--skip-permissions-unsafe"}},
	}

	for _, tt := range tests {
		t.Run(tt.autonomy, func(t *testing.T) {
			got := TranslateAutonomyToDroidFlags(tt.autonomy)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTranslateReasoningToDroidFlags(t *testing.T) {
	tests := []struct {
		reasoning string
		want      []string
	}{
		{"off", []string{"-r", "off"}},
		{"low", []string{"-r", "low"}},
		{"medium", []string{"-r", "medium"}},
		{"high", []string{"-r", "high"}},
		{"unknown", []string{"-r", "medium"}}, // default
	}

	for _, tt := range tests {
		t.Run(tt.reasoning, func(t *testing.T) {
			got := TranslateReasoningToDroidFlags(tt.reasoning)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDroidMCPConfigSerialization(t *testing.T) {
	cfg := &DroidMCPConfig{
		MCPServers: map[string]DroidMCPServer{
			"test": {
				Type:    "stdio",
				Command: "/bin/test",
				Args:    []string{"--flag"},
			},
		},
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	// Verify key name is "mcpServers" (Droid format)
	if !containsString(string(data), `"mcpServers"`) {
		t.Errorf("expected mcpServers key, got: %s", data)
	}
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
