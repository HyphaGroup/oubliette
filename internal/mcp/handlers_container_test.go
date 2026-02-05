package mcp

import (
	"testing"
)

func TestSpawnContainerParams(t *testing.T) {
	params := SpawnContainerParams{
		ProjectID: "550e8400-e29b-41d4-a716-446655440000",
	}

	if params.ProjectID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("ProjectID = %q, want UUID", params.ProjectID)
	}
}

func TestExecCommandParams(t *testing.T) {
	params := ExecCommandParams{
		ProjectID:  "550e8400-e29b-41d4-a716-446655440000",
		Command:    "ls -la",
		WorkingDir: "/workspace",
	}

	if params.ProjectID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("ProjectID = %q, want UUID", params.ProjectID)
	}
	if params.Command != "ls -la" {
		t.Errorf("Command = %q, want %q", params.Command, "ls -la")
	}
	if params.WorkingDir != "/workspace" {
		t.Errorf("WorkingDir = %q, want %q", params.WorkingDir, "/workspace")
	}
}

func TestExecCommandParams_Minimal(t *testing.T) {
	params := ExecCommandParams{
		ProjectID: "project-1",
		Command:   "echo hello",
	}

	if params.WorkingDir != "" {
		t.Errorf("WorkingDir = %q, want empty", params.WorkingDir)
	}
}

func TestStopContainerParams(t *testing.T) {
	params := StopContainerParams{
		ProjectID: "550e8400-e29b-41d4-a716-446655440000",
	}

	if params.ProjectID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("ProjectID = %q, want UUID", params.ProjectID)
	}
}

func TestGetLogsParams(t *testing.T) {
	params := GetLogsParams{
		ProjectID: "550e8400-e29b-41d4-a716-446655440000",
	}

	if params.ProjectID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("ProjectID = %q, want UUID", params.ProjectID)
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
