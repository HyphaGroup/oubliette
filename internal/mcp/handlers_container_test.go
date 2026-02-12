package mcp

import (
	"testing"
)

func TestContainerParams(t *testing.T) {
	params := ContainerParams{
		Action:    "exec",
		ProjectID: "550e8400-e29b-41d4-a716-446655440000",
		Command:   "ls -la",
	}

	if params.Action != "exec" {
		t.Errorf("Action = %q, want %q", params.Action, "exec")
	}
	if params.ProjectID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("ProjectID = %q, want UUID", params.ProjectID)
	}
	if params.Command != "ls -la" {
		t.Errorf("Command = %q, want %q", params.Command, "ls -la")
	}
}

func TestContainerParams_Minimal(t *testing.T) {
	params := ContainerParams{
		Action:    "start",
		ProjectID: "project-1",
	}

	if params.WorkingDir != "" {
		t.Errorf("WorkingDir = %q, want empty", params.WorkingDir)
	}
	if params.Command != "" {
		t.Errorf("Command = %q, want empty", params.Command)
	}
}

func TestContainerRefreshParams(t *testing.T) {
	t.Run("with project", func(t *testing.T) {
		params := ContainerRefreshParams{
			ProjectID: "project-1",
		}
		if params.ProjectID != "project-1" {
			t.Errorf("ProjectID = %q, want %q", params.ProjectID, "project-1")
		}
	})

	t.Run("with container_type", func(t *testing.T) {
		params := ContainerRefreshParams{
			ContainerType: "dev",
		}
		if params.ContainerType != "dev" {
			t.Errorf("ContainerType = %q, want %q", params.ContainerType, "dev")
		}
	})
}
