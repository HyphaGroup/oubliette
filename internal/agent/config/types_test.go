package config

import (
	"encoding/json"
	"testing"
	"time"
)

func TestProjectConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ProjectConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			cfg: ProjectConfig{
				ID:                 "proj-123",
				Name:               "test-project",
				DefaultWorkspaceID: "ws-456",
				Container:          ContainerConfig{Type: "dev"},
				Agent: AgentConfig{
					Runtime:  "droid",
					Model:    "claude-sonnet-4-5",
					Autonomy: "high",
				},
			},
			wantErr: false,
		},
		{
			name: "missing id",
			cfg: ProjectConfig{
				Name:               "test",
				DefaultWorkspaceID: "ws-456",
				Container:          ContainerConfig{Type: "dev"},
				Agent:              AgentConfig{Runtime: "droid", Model: "m", Autonomy: "high"},
			},
			wantErr: true,
			errMsg:  "project id is required",
		},
		{
			name: "invalid autonomy",
			cfg: ProjectConfig{
				ID:                 "proj-123",
				Name:               "test",
				DefaultWorkspaceID: "ws-456",
				Container:          ContainerConfig{Type: "dev"},
				Agent:              AgentConfig{Runtime: "droid", Model: "m", Autonomy: "invalid"},
			},
			wantErr: true,
			errMsg:  "invalid autonomy level",
		},
		{
			name: "valid autonomy off",
			cfg: ProjectConfig{
				ID:                 "proj-123",
				Name:               "test",
				DefaultWorkspaceID: "ws-456",
				Container:          ContainerConfig{Type: "dev"},
				Agent:              AgentConfig{Runtime: "droid", Model: "m", Autonomy: "off"},
			},
			wantErr: false,
		},
		{
			name: "invalid reasoning",
			cfg: ProjectConfig{
				ID:                 "proj-123",
				Name:               "test",
				DefaultWorkspaceID: "ws-456",
				Container:          ContainerConfig{Type: "dev"},
				Agent:              AgentConfig{Runtime: "droid", Model: "m", Autonomy: "high", Reasoning: "invalid"},
			},
			wantErr: true,
			errMsg:  "invalid reasoning level",
		},
		{
			name: "valid reasoning medium",
			cfg: ProjectConfig{
				ID:                 "proj-123",
				Name:               "test",
				DefaultWorkspaceID: "ws-456",
				Container:          ContainerConfig{Type: "dev"},
				Agent:              AgentConfig{Runtime: "droid", Model: "m", Autonomy: "high", Reasoning: "medium"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestProjectConfigSerialization(t *testing.T) {
	cfg := ProjectConfig{
		ID:                 "proj-abc123",
		Name:               "my-project",
		Description:        "Test project",
		CreatedAt:          time.Date(2025, 1, 28, 10, 0, 0, 0, time.UTC),
		DefaultWorkspaceID: "ws-def456",
		Container: ContainerConfig{
			Type:      "dev",
			ImageName: "oubliette-dev:latest",
		},
		Agent: AgentConfig{
			Runtime:   "droid",
			Model:     "claude-sonnet-4-5-20250929",
			Autonomy:  "high",
			Reasoning: "medium",
			MCPServers: map[string]MCPServer{
				"oubliette-parent": {
					Type:    "stdio",
					Command: "/usr/local/bin/oubliette-client",
					Args:    []string{"/mcp/relay.sock"},
				},
			},
		},
		Limits: LimitsConfig{
			MaxRecursionDepth:   3,
			MaxAgentsPerSession: 50,
			MaxCostUSD:          10.0,
		},
	}

	// Serialize
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Deserialize
	var parsed ProjectConfig
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Verify round-trip
	if parsed.ID != cfg.ID {
		t.Errorf("ID mismatch: got %q, want %q", parsed.ID, cfg.ID)
	}
	if parsed.Agent.Model != cfg.Agent.Model {
		t.Errorf("Model mismatch: got %q, want %q", parsed.Agent.Model, cfg.Agent.Model)
	}
	if parsed.Agent.Reasoning != cfg.Agent.Reasoning {
		t.Errorf("Reasoning mismatch: got %q, want %q", parsed.Agent.Reasoning, cfg.Agent.Reasoning)
	}
	if len(parsed.Agent.MCPServers) != 1 {
		t.Errorf("MCPServers count mismatch: got %d, want 1", len(parsed.Agent.MCPServers))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || substr == "" ||
		(s != "" && substr != "" && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
