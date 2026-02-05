package mcp

import (
	"testing"
)

func TestCreateProjectParams(t *testing.T) {
	initGit := true
	params := CreateProjectParams{
		Name:        "test-project",
		Description: "A test project",
		GitHubToken: "gh_token",
		RemoteURL:   "https://github.com/test/repo",
		InitGit:     &initGit,
		Languages:   []string{"go", "python"},
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
	if params.RemoteURL != "https://github.com/test/repo" {
		t.Errorf("RemoteURL = %q, want %q", params.RemoteURL, "https://github.com/test/repo")
	}
	if params.InitGit == nil || !*params.InitGit {
		t.Error("InitGit should be true")
	}
	if len(params.Languages) != 2 {
		t.Errorf("Languages count = %d, want 2", len(params.Languages))
	}
}

func TestCreateProjectParams_Defaults(t *testing.T) {
	params := CreateProjectParams{
		Name: "minimal",
	}

	if params.Name != "minimal" {
		t.Errorf("Name = %q, want %q", params.Name, "minimal")
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
	if params.Languages != nil {
		t.Error("Languages should be nil by default")
	}
}

func TestListProjectsParams(t *testing.T) {
	nameContains := "test"
	limit := 10
	params := ListProjectsParams{
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

func TestListProjectsParams_Empty(t *testing.T) {
	params := ListProjectsParams{}

	if params.NameContains != nil {
		t.Error("NameContains should be nil")
	}
	if params.Limit != nil {
		t.Error("Limit should be nil")
	}
}

func TestGetProjectParams(t *testing.T) {
	params := GetProjectParams{
		ProjectID: "550e8400-e29b-41d4-a716-446655440000",
	}

	if params.ProjectID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("ProjectID = %q, want UUID", params.ProjectID)
	}
}

func TestDeleteProjectParams(t *testing.T) {
	params := DeleteProjectParams{
		ProjectID: "550e8400-e29b-41d4-a716-446655440000",
	}

	if params.ProjectID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("ProjectID = %q, want UUID", params.ProjectID)
	}
}
