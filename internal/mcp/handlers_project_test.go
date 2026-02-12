package mcp

import (
	"testing"
)

func TestProjectParams_Create(t *testing.T) {
	initGit := true
	params := ProjectParams{
		Action:      "create",
		Name:        "test-project",
		Description: "A test project",
		GitHubToken: "gh_token",
		RemoteURL:   "https://github.com/test/repo",
		InitGit:     &initGit,
	}

	if params.Name != "test-project" {
		t.Errorf("Name = %q, want %q", params.Name, "test-project")
	}
	if params.Description != "A test project" {
		t.Errorf("Description = %q, want %q", params.Description, "A test project")
	}
	if params.GitHubToken != "gh_token" {
		t.Errorf("GitHubToken = %q, want %q", params.GitHubToken, "gh_token")
	}
	if params.InitGit == nil || !*params.InitGit {
		t.Error("InitGit should be true")
	}
}

func TestProjectParams_Defaults(t *testing.T) {
	params := ProjectParams{
		Action: "create",
		Name:   "minimal",
	}

	if params.Description != "" {
		t.Errorf("Description = %q, want empty", params.Description)
	}
	if params.GitHubToken != "" {
		t.Errorf("GitHubToken = %q, want empty", params.GitHubToken)
	}
	if params.InitGit != nil {
		t.Error("InitGit should be nil by default")
	}
}

func TestProjectParams_List(t *testing.T) {
	nameContains := "test"
	limit := 10
	params := ProjectParams{
		Action:       "list",
		NameContains: &nameContains,
		Limit:        &limit,
	}

	if params.NameContains == nil || *params.NameContains != "test" {
		t.Error("NameContains should be 'test'")
	}
	if params.Limit == nil || *params.Limit != 10 {
		t.Error("Limit should be 10")
	}
}

func TestProjectParams_Get(t *testing.T) {
	params := ProjectParams{
		Action:    "get",
		ProjectID: "550e8400-e29b-41d4-a716-446655440000",
	}

	if params.ProjectID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("ProjectID = %q, want UUID", params.ProjectID)
	}
}
