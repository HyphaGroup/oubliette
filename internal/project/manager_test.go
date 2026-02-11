package project

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// Ensure strings is imported
var _ = strings.Contains

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()

	mgr := NewManager(tmpDir, 5, 10, 100.0)

	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}
	if mgr.projectsDir != tmpDir {
		t.Errorf("projectsDir = %q, want %q", mgr.projectsDir, tmpDir)
	}
	if mgr.defaultMaxDepth != 5 {
		t.Errorf("defaultMaxDepth = %d, want 5", mgr.defaultMaxDepth)
	}
	if mgr.defaultMaxAgents != 10 {
		t.Errorf("defaultMaxAgents = %d, want 10", mgr.defaultMaxAgents)
	}
	if mgr.defaultMaxCostUSD != 100.0 {
		t.Errorf("defaultMaxCostUSD = %f, want 100.0", mgr.defaultMaxCostUSD)
	}
}

func TestManagerGetMaxDepth(t *testing.T) {
	mgr := NewManager(t.TempDir(), 5, 10, 100.0)

	t.Run("use default", func(t *testing.T) {
		project := &Project{}
		if got := mgr.GetMaxDepth(project); got != 5 {
			t.Errorf("GetMaxDepth() = %d, want 5", got)
		}
	})

	t.Run("use project override", func(t *testing.T) {
		maxDepth := 3
		project := &Project{
			RecursionConfig: &RecursionConfig{
				MaxDepth: &maxDepth,
			},
		}
		if got := mgr.GetMaxDepth(project); got != 3 {
			t.Errorf("GetMaxDepth() = %d, want 3", got)
		}
	})
}

func TestManagerGetMaxAgents(t *testing.T) {
	mgr := NewManager(t.TempDir(), 5, 10, 100.0)

	t.Run("use default", func(t *testing.T) {
		project := &Project{}
		if got := mgr.GetMaxAgents(project); got != 10 {
			t.Errorf("GetMaxAgents() = %d, want 10", got)
		}
	})

	t.Run("use project override", func(t *testing.T) {
		maxAgents := 20
		project := &Project{
			RecursionConfig: &RecursionConfig{
				MaxAgents: &maxAgents,
			},
		}
		if got := mgr.GetMaxAgents(project); got != 20 {
			t.Errorf("GetMaxAgents() = %d, want 20", got)
		}
	})
}

func TestManagerGetMaxCostUSD(t *testing.T) {
	mgr := NewManager(t.TempDir(), 5, 10, 100.0)

	t.Run("use default", func(t *testing.T) {
		project := &Project{}
		if got := mgr.GetMaxCostUSD(project); got != 100.0 {
			t.Errorf("GetMaxCostUSD() = %f, want 100.0", got)
		}
	})

	t.Run("use project override", func(t *testing.T) {
		maxCost := 50.0
		project := &Project{
			RecursionConfig: &RecursionConfig{
				MaxCostUSD: &maxCost,
			},
		}
		if got := mgr.GetMaxCostUSD(project); got != 50.0 {
			t.Errorf("GetMaxCostUSD() = %f, want 50.0", got)
		}
	})
}

func TestManagerGet(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, 5, 10, 100.0)

	// Create a project manually
	projectID := "550e8400-e29b-41d4-a716-446655440000"
	projectDir := filepath.Join(tmpDir, projectID)
	_ = os.MkdirAll(projectDir, 0o755)

	project := &Project{
		ID:        projectID,
		Name:      "Test Project",
		CreatedAt: time.Now(),
		ImageName: "oubliette:latest",
	}
	data, _ := json.Marshal(project)
	_ = os.WriteFile(filepath.Join(projectDir, "metadata.json"), data, 0o644)

	t.Run("get existing project", func(t *testing.T) {
		got, err := mgr.Get(projectID)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if got.ID != projectID {
			t.Errorf("ID = %q, want %q", got.ID, projectID)
		}
		if got.Name != "Test Project" {
			t.Errorf("Name = %q, want %q", got.Name, "Test Project")
		}
	})

	t.Run("get with custom Dockerfile", func(t *testing.T) {
		// Create Dockerfile
		_ = os.WriteFile(filepath.Join(projectDir, "Dockerfile"), []byte("FROM alpine"), 0o644)

		got, err := mgr.Get(projectID)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if !got.HasDockerfile {
			t.Error("expected HasDockerfile = true")
		}
		if !strings.HasPrefix(got.ImageName, "oubliette-") {
			t.Errorf("ImageName = %q, expected oubliette-* prefix", got.ImageName)
		}
	})

	t.Run("get non-existent project", func(t *testing.T) {
		_, err := mgr.Get("550e8400-e29b-41d4-a716-446655440001")
		if err == nil {
			t.Error("expected error for non-existent project")
		}
	})

	t.Run("invalid project ID", func(t *testing.T) {
		_, err := mgr.Get("../escape")
		if err == nil {
			t.Error("expected error for invalid project ID")
		}
	})
}

func TestManagerList(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, 5, 10, 100.0)

	// Create multiple projects
	projects := []struct {
		id   string
		name string
	}{
		{"550e8400-e29b-41d4-a716-446655440001", "Alpha Project"},
		{"550e8400-e29b-41d4-a716-446655440002", "Beta Project"},
		{"550e8400-e29b-41d4-a716-446655440003", "Alpha Two"},
	}

	for _, p := range projects {
		projectDir := filepath.Join(tmpDir, p.id)
		_ = os.MkdirAll(projectDir, 0o755)
		project := &Project{ID: p.id, Name: p.name, CreatedAt: time.Now()}
		data, _ := json.Marshal(project)
		_ = os.WriteFile(filepath.Join(projectDir, "metadata.json"), data, 0o644)
	}

	t.Run("list all", func(t *testing.T) {
		list, err := mgr.List(nil)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(list) != 3 {
			t.Errorf("len(list) = %d, want 3", len(list))
		}
	})

	t.Run("filter by name", func(t *testing.T) {
		filter := &ListProjectsFilter{NameContains: "alpha"}
		list, err := mgr.List(filter)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(list) != 2 {
			t.Errorf("len(list) = %d, want 2", len(list))
		}
	})

	t.Run("limit results", func(t *testing.T) {
		filter := &ListProjectsFilter{Limit: 2}
		list, err := mgr.List(filter)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(list) != 2 {
			t.Errorf("len(list) = %d, want 2", len(list))
		}
	})

	t.Run("empty directory", func(t *testing.T) {
		emptyMgr := NewManager(filepath.Join(tmpDir, "empty"), 5, 10, 100.0)
		list, err := emptyMgr.List(nil)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(list) != 0 {
			t.Errorf("len(list) = %d, want 0", len(list))
		}
	})
}

func TestManagerDelete(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, 5, 10, 100.0)

	projectID := "550e8400-e29b-41d4-a716-446655440000"
	projectDir := filepath.Join(tmpDir, projectID)
	_ = os.MkdirAll(projectDir, 0o755)
	project := &Project{ID: projectID, Name: "Test"}
	data, _ := json.Marshal(project)
	_ = os.WriteFile(filepath.Join(projectDir, "metadata.json"), data, 0o644)

	t.Run("delete existing project", func(t *testing.T) {
		err := mgr.Delete(projectID)
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		// Verify deleted
		if _, err := os.Stat(projectDir); !errors.Is(err, fs.ErrNotExist) {
			t.Error("project directory should be deleted")
		}
	})

	t.Run("delete non-existent", func(t *testing.T) {
		err := mgr.Delete("550e8400-e29b-41d4-a716-446655440001")
		if err == nil {
			t.Error("expected error for non-existent project")
		}
	})

	t.Run("invalid project ID", func(t *testing.T) {
		err := mgr.Delete("../escape")
		if err == nil {
			t.Error("expected error for invalid project ID")
		}
	})
}

// MockSessionChecker implements ActiveSessionChecker for testing
type MockSessionChecker struct {
	projectsWithSessions   map[string]bool
	workspacesWithSessions map[string]bool
}

func (m *MockSessionChecker) HasActiveSessionsForProject(projectID string) bool {
	return m.projectsWithSessions[projectID]
}

func (m *MockSessionChecker) HasActiveSessionsForWorkspace(projectID, workspaceID string) bool {
	return m.workspacesWithSessions[projectID+"/"+workspaceID]
}

func TestManagerDeleteWithActiveSessions(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, 5, 10, 100.0)

	checker := &MockSessionChecker{
		projectsWithSessions: map[string]bool{"550e8400-e29b-41d4-a716-446655440000": true},
	}
	mgr.SetSessionChecker(checker)

	projectID := "550e8400-e29b-41d4-a716-446655440000"
	projectDir := filepath.Join(tmpDir, projectID)
	_ = os.MkdirAll(projectDir, 0o755)
	project := &Project{ID: projectID, Name: "Test"}
	data, _ := json.Marshal(project)
	_ = os.WriteFile(filepath.Join(projectDir, "metadata.json"), data, 0o644)

	err := mgr.Delete(projectID)
	if err == nil {
		t.Error("expected error when deleting project with active sessions")
	}
	if !strings.Contains(err.Error(), "active sessions") {
		t.Errorf("error should mention active sessions, got: %v", err)
	}
}

func TestManagerWorkspaceOperations(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, 5, 10, 100.0)

	projectID := "550e8400-e29b-41d4-a716-446655440000"
	defaultWorkspaceID := "660e8400-e29b-41d4-a716-446655440000"
	projectDir := filepath.Join(tmpDir, projectID)
	workspacesDir := filepath.Join(projectDir, "workspaces")
	_ = os.MkdirAll(workspacesDir, 0o755)

	// Create project metadata
	project := &Project{
		ID:                 projectID,
		Name:               "Test",
		DefaultWorkspaceID: defaultWorkspaceID,
	}
	data, _ := json.Marshal(project)
	_ = os.WriteFile(filepath.Join(projectDir, "metadata.json"), data, 0o644)

	t.Run("create workspace", func(t *testing.T) {
		workspaceID := "770e8400-e29b-41d4-a716-446655440000"
		ws, err := mgr.CreateWorkspace(projectID, workspaceID, "ext-123", "test")
		if err != nil {
			t.Fatalf("CreateWorkspace() error = %v", err)
		}
		if ws.ID != workspaceID {
			t.Errorf("ID = %q, want %q", ws.ID, workspaceID)
		}
		if ws.ExternalID != "ext-123" {
			t.Errorf("ExternalID = %q, want %q", ws.ExternalID, "ext-123")
		}
		if ws.Source != "test" {
			t.Errorf("Source = %q, want %q", ws.Source, "test")
		}
	})

	t.Run("create workspace with auto ID", func(t *testing.T) {
		ws, err := mgr.CreateWorkspace(projectID, "", "", "")
		if err != nil {
			t.Fatalf("CreateWorkspace() error = %v", err)
		}
		if ws.ID == "" {
			t.Error("expected auto-generated workspace ID")
		}
		// Should be a UUID
		if len(ws.ID) != 36 {
			t.Errorf("auto-generated ID should be UUID, got %q", ws.ID)
		}
	})

	t.Run("create existing workspace returns metadata", func(t *testing.T) {
		workspaceID := "880e8400-e29b-41d4-a716-446655440000"
		ws1, _ := mgr.CreateWorkspace(projectID, workspaceID, "first", "test1")
		ws2, err := mgr.CreateWorkspace(projectID, workspaceID, "second", "test2")
		if err != nil {
			t.Fatalf("CreateWorkspace() error = %v", err)
		}
		// Should return existing, not overwrite
		if ws2.ExternalID != ws1.ExternalID {
			t.Error("expected existing workspace to be returned")
		}
	})

	t.Run("get workspace metadata", func(t *testing.T) {
		workspaceID := "990e8400-e29b-41d4-a716-446655440000"
		_, _ = mgr.CreateWorkspace(projectID, workspaceID, "ext", "src")

		metadata, err := mgr.GetWorkspaceMetadata(projectID, workspaceID)
		if err != nil {
			t.Fatalf("GetWorkspaceMetadata() error = %v", err)
		}
		if metadata.ID != workspaceID {
			t.Errorf("ID = %q, want %q", metadata.ID, workspaceID)
		}
	})

	t.Run("get non-existent workspace", func(t *testing.T) {
		_, err := mgr.GetWorkspaceMetadata(projectID, "aa0e8400-e29b-41d4-a716-446655440000")
		if err == nil {
			t.Error("expected error for non-existent workspace")
		}
	})

	t.Run("list workspaces", func(t *testing.T) {
		list, err := mgr.ListWorkspaces(projectID)
		if err != nil {
			t.Fatalf("ListWorkspaces() error = %v", err)
		}
		if len(list) < 3 {
			t.Errorf("len(list) = %d, want >= 3", len(list))
		}
	})

	t.Run("workspace exists", func(t *testing.T) {
		if !mgr.WorkspaceExists(projectID, "770e8400-e29b-41d4-a716-446655440000") {
			t.Error("expected workspace to exist")
		}
		if mgr.WorkspaceExists(projectID, "nonexistent") {
			t.Error("expected workspace to not exist")
		}
	})

	t.Run("update workspace last session", func(t *testing.T) {
		workspaceID := "770e8400-e29b-41d4-a716-446655440000"
		err := mgr.UpdateWorkspaceLastSession(projectID, workspaceID)
		if err != nil {
			t.Fatalf("UpdateWorkspaceLastSession() error = %v", err)
		}

		metadata, _ := mgr.GetWorkspaceMetadata(projectID, workspaceID)
		if metadata.LastSessionAt.IsZero() {
			t.Error("LastSessionAt should be set")
		}
	})

	t.Run("delete workspace", func(t *testing.T) {
		workspaceID := "bb0e8400-e29b-41d4-a716-446655440000"
		_, _ = mgr.CreateWorkspace(projectID, workspaceID, "", "")

		err := mgr.DeleteWorkspace(projectID, workspaceID)
		if err != nil {
			t.Fatalf("DeleteWorkspace() error = %v", err)
		}

		if mgr.WorkspaceExists(projectID, workspaceID) {
			t.Error("workspace should be deleted")
		}
	})

	t.Run("cannot delete default workspace", func(t *testing.T) {
		err := mgr.DeleteWorkspace(projectID, defaultWorkspaceID)
		if err == nil {
			t.Error("expected error when deleting default workspace")
		}
	})

	t.Run("delete non-existent workspace is idempotent", func(t *testing.T) {
		err := mgr.DeleteWorkspace(projectID, "cc0e8400-e29b-41d4-a716-446655440000")
		if err != nil {
			t.Errorf("DeleteWorkspace() error = %v, expected nil (idempotent)", err)
		}
	})
}

func TestManagerDeleteWorkspaceWithActiveSessions(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, 5, 10, 100.0)

	projectID := "550e8400-e29b-41d4-a716-446655440000"
	workspaceID := "660e8400-e29b-41d4-a716-446655440000"

	checker := &MockSessionChecker{
		workspacesWithSessions: map[string]bool{projectID + "/" + workspaceID: true},
	}
	mgr.SetSessionChecker(checker)

	projectDir := filepath.Join(tmpDir, projectID)
	_ = os.MkdirAll(filepath.Join(projectDir, "workspaces"), 0o755)
	project := &Project{ID: projectID, Name: "Test", DefaultWorkspaceID: "other"}
	data, _ := json.Marshal(project)
	_ = os.WriteFile(filepath.Join(projectDir, "metadata.json"), data, 0o644)

	_, _ = mgr.CreateWorkspace(projectID, workspaceID, "", "")

	err := mgr.DeleteWorkspace(projectID, workspaceID)
	if err == nil {
		t.Error("expected error when deleting workspace with active sessions")
	}
}

func TestManagerGetWorkspaceDir(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, 5, 10, 100.0)

	projectID := "550e8400-e29b-41d4-a716-446655440000"
	defaultWorkspaceID := "660e8400-e29b-41d4-a716-446655440000"
	projectDir := filepath.Join(tmpDir, projectID)
	_ = os.MkdirAll(projectDir, 0o755)

	project := &Project{
		ID:                 projectID,
		DefaultWorkspaceID: defaultWorkspaceID,
	}
	data, _ := json.Marshal(project)
	_ = os.WriteFile(filepath.Join(projectDir, "metadata.json"), data, 0o644)

	dir := mgr.GetWorkspaceDir(projectID)
	expected := filepath.Join(tmpDir, projectID, "workspaces", defaultWorkspaceID)
	if dir != expected {
		t.Errorf("GetWorkspaceDir() = %q, want %q", dir, expected)
	}
}

func TestManagerGetWorkspacePath(t *testing.T) {
	mgr := NewManager("/projects", 5, 10, 100.0)

	path := mgr.GetWorkspacePath("project-1", "workspace-1")
	expected := "/projects/project-1/workspaces/workspace-1"
	if path != expected {
		t.Errorf("GetWorkspacePath() = %q, want %q", path, expected)
	}
}

func TestManagerGetProjectDir(t *testing.T) {
	mgr := NewManager("/projects", 5, 10, 100.0)

	dir := mgr.GetProjectDir("project-1")
	expected := "/projects/project-1"
	if dir != expected {
		t.Errorf("GetProjectDir() = %q, want %q", dir, expected)
	}
}

func TestManagerGetSessionsDir(t *testing.T) {
	mgr := NewManager("/projects", 5, 10, 100.0)

	dir := mgr.GetSessionsDir("project-1")
	expected := "/projects/project-1/sessions"
	if dir != expected {
		t.Errorf("GetSessionsDir() = %q, want %q", dir, expected)
	}
}

func TestManagerConcurrentGet(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, 5, 10, 100.0)

	projectID := "550e8400-e29b-41d4-a716-446655440000"
	projectDir := filepath.Join(tmpDir, projectID)
	_ = os.MkdirAll(projectDir, 0o755)

	project := &Project{ID: projectID, Name: "Test", CreatedAt: time.Now()}
	data, _ := json.Marshal(project)
	_ = os.WriteFile(filepath.Join(projectDir, "metadata.json"), data, 0o644)

	var wg sync.WaitGroup
	errs := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := mgr.Get(projectID)
			if err != nil {
				errs <- err
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent Get() error = %v", err)
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, 5, 10, 100.0)

	src := filepath.Join(tmpDir, "source.txt")
	dst := filepath.Join(tmpDir, "dest.txt")

	content := []byte("test content")
	_ = os.WriteFile(src, content, 0o644)

	err := mgr.copyFile(src, dst)
	if err != nil {
		t.Fatalf("copyFile() error = %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("content = %q, want %q", string(got), string(content))
	}
}

func TestCopyDir(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, 5, 10, 100.0)

	srcDir := filepath.Join(tmpDir, "src")
	dstDir := filepath.Join(tmpDir, "dst")

	// Create source structure
	_ = os.MkdirAll(filepath.Join(srcDir, "subdir"), 0o755)
	_ = os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0o644)
	_ = os.WriteFile(filepath.Join(srcDir, "subdir", "file2.txt"), []byte("content2"), 0o644)

	err := mgr.copyDir(srcDir, dstDir)
	if err != nil {
		t.Fatalf("copyDir() error = %v", err)
	}

	// Verify structure
	if _, err := os.Stat(filepath.Join(dstDir, "file1.txt")); err != nil {
		t.Error("file1.txt should exist")
	}
	if _, err := os.Stat(filepath.Join(dstDir, "subdir", "file2.txt")); err != nil {
		t.Error("subdir/file2.txt should exist")
	}

	// Verify content
	got, _ := os.ReadFile(filepath.Join(dstDir, "subdir", "file2.txt"))
	if string(got) != "content2" {
		t.Errorf("file2.txt content = %q, want %q", string(got), "content2")
	}
}

func TestManagerCreate(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, 5, 10, 100.0)

	t.Run("create basic project", func(t *testing.T) {
		req := CreateProjectRequest{
			Name:        "Test Project",
			Description: "A test project",
			InitGit:     false,
		}

		proj, err := mgr.Create(req)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if proj.Name != "Test Project" {
			t.Errorf("Name = %q, want %q", proj.Name, "Test Project")
		}
		if proj.Description != "A test project" {
			t.Errorf("Description = %q, want %q", proj.Description, "A test project")
		}
		if proj.ID == "" {
			t.Error("expected non-empty project ID")
		}
		if proj.DefaultWorkspaceID == "" {
			t.Error("expected non-empty default workspace ID")
		}

		// Verify project directory was created
		projectDir := filepath.Join(tmpDir, proj.ID)
		if _, err := os.Stat(projectDir); err != nil {
			t.Error("project directory should exist")
		}

		// Verify workspaces directory was created
		workspacesDir := filepath.Join(projectDir, "workspaces")
		if _, err := os.Stat(workspacesDir); err != nil {
			t.Error("workspaces directory should exist")
		}

		// Verify metadata file was created
		metadataFile := filepath.Join(projectDir, "metadata.json")
		if _, err := os.Stat(metadataFile); err != nil {
			t.Error("metadata.json should exist")
		}
	})

	t.Run("create with custom token", func(t *testing.T) {
		req := CreateProjectRequest{
			Name:        "Project with token",
			GitHubToken: "custom-token",
			InitGit:     false,
		}

		proj, err := mgr.Create(req)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// GitHubToken should not be serialized but check .env file
		envPath := filepath.Join(tmpDir, proj.ID, "workspaces", proj.DefaultWorkspaceID, ".env")
		envContent, err := os.ReadFile(envPath)
		if err != nil {
			t.Fatalf("failed to read .env file: %v", err)
		}
		if !strings.Contains(string(envContent), "GH_TOKEN=custom-token") {
			t.Error(".env should contain custom token")
		}
	})

	t.Run("create with remote URL", func(t *testing.T) {
		req := CreateProjectRequest{
			Name:      "Project with remote",
			RemoteURL: "https://github.com/test/repo",
			InitGit:   false,
		}

		proj, err := mgr.Create(req)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if proj.RemoteURL != "https://github.com/test/repo" {
			t.Errorf("RemoteURL = %q, want %q", proj.RemoteURL, "https://github.com/test/repo")
		}
	})
}

func TestRecursionConfigTypes(t *testing.T) {
	maxDepth := 5
	maxAgents := 20
	maxCost := 50.0

	cfg := &RecursionConfig{
		MaxDepth:   &maxDepth,
		MaxAgents:  &maxAgents,
		MaxCostUSD: &maxCost,
	}

	if cfg.MaxDepth == nil || *cfg.MaxDepth != 5 {
		t.Error("MaxDepth should be 5")
	}
	if cfg.MaxAgents == nil || *cfg.MaxAgents != 20 {
		t.Error("MaxAgents should be 20")
	}
	if cfg.MaxCostUSD == nil || *cfg.MaxCostUSD != 50.0 {
		t.Error("MaxCostUSD should be 50.0")
	}
}

func TestProjectTypes(t *testing.T) {
	now := time.Now()
	proj := &Project{
		ID:                 "550e8400-e29b-41d4-a716-446655440000",
		Name:               "Test",
		Description:        "Desc",
		DefaultWorkspaceID: "660e8400-e29b-41d4-a716-446655440000",
		CreatedAt:          now,
		RemoteURL:          "https://github.com/test/repo",
		ImageName:          "oubliette:latest",
		HasDockerfile:      true,
		ContainerID:        "abc123",
		ContainerStatus:    "running",
	}

	if proj.ID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Error("ID mismatch")
	}
	if proj.Name != "Test" {
		t.Error("Name mismatch")
	}
	if !proj.HasDockerfile {
		t.Error("HasDockerfile should be true")
	}
}

func TestWorkspaceMetadataTypes(t *testing.T) {
	now := time.Now()
	meta := &WorkspaceMetadata{
		ID:            "workspace-id",
		CreatedAt:     now,
		LastSessionAt: now,
		ExternalID:    "ext-123",
		Source:        "test",
	}

	if meta.ID != "workspace-id" {
		t.Error("ID mismatch")
	}
	if meta.ExternalID != "ext-123" {
		t.Error("ExternalID mismatch")
	}
	if meta.Source != "test" {
		t.Error("Source mismatch")
	}
}

func TestCreateWorkspaceCopiesAGENTSMDWhenIsolated(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, 5, 10, 100.0)

	// Create project directory structure (use valid UUID)
	projectID := "12345678-1234-1234-1234-123456789abc"
	projectDir := filepath.Join(tmpDir, projectID)
	_ = os.MkdirAll(filepath.Join(projectDir, "workspaces"), 0o755)
	_ = os.MkdirAll(filepath.Join(projectDir, ".factory"), 0o755)

	// Create project metadata with workspace isolation enabled
	proj := &Project{
		ID:                 projectID,
		Name:               "test-project",
		WorkspaceIsolation: true,
	}
	metadataPath := filepath.Join(projectDir, "metadata.json")
	projData, _ := json.Marshal(proj)
	_ = os.WriteFile(metadataPath, projData, 0o644)

	// Create AGENTS.md at project root
	agentsContent := "# Test AGENTS.md\nThis is a test."
	if err := os.WriteFile(filepath.Join(projectDir, "AGENTS.md"), []byte(agentsContent), 0o644); err != nil {
		t.Fatalf("failed to create AGENTS.md: %v", err)
	}

	// Create workspace (use valid UUID)
	workspaceID := "abcdef12-3456-7890-abcd-ef1234567890"
	meta, err := mgr.CreateWorkspace(projectID, workspaceID, "ext-id", "test")
	if err != nil {
		t.Fatalf("CreateWorkspace failed: %v", err)
	}

	if meta.ID != workspaceID {
		t.Errorf("workspace ID = %q, want %q", meta.ID, workspaceID)
	}

	// Verify AGENTS.md was copied to workspace (isolation enabled)
	workspaceAgents := filepath.Join(projectDir, "workspaces", workspaceID, "AGENTS.md")
	data, err := os.ReadFile(workspaceAgents)
	if err != nil {
		t.Fatalf("failed to read workspace AGENTS.md: %v (should be copied when isolation enabled)", err)
	}
	if string(data) != agentsContent {
		t.Errorf("workspace AGENTS.md content = %q, want %q", string(data), agentsContent)
	}
}

func TestCreateWorkspaceSkipsAGENTSMDWhenNotIsolated(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, 5, 10, 100.0)

	// Create project directory structure (use valid UUID)
	projectID := "11111111-1111-1111-1111-111111111111"
	projectDir := filepath.Join(tmpDir, projectID)
	_ = os.MkdirAll(filepath.Join(projectDir, "workspaces"), 0o755)
	_ = os.MkdirAll(filepath.Join(projectDir, ".factory"), 0o755)

	// Create project metadata WITHOUT workspace isolation
	proj := &Project{
		ID:                 projectID,
		Name:               "test-project-no-iso",
		WorkspaceIsolation: false,
	}
	metadataPath := filepath.Join(projectDir, "metadata.json")
	projData, _ := json.Marshal(proj)
	_ = os.WriteFile(metadataPath, projData, 0o644)

	// Create AGENTS.md at project root
	if err := os.WriteFile(filepath.Join(projectDir, "AGENTS.md"), []byte("# Test"), 0o644); err != nil {
		t.Fatalf("failed to create AGENTS.md: %v", err)
	}

	// Create workspace (use valid UUID)
	workspaceID := "bcdef123-4567-890a-bcde-f12345678901"
	_, err := mgr.CreateWorkspace(projectID, workspaceID, "ext-id", "test")
	if err != nil {
		t.Fatalf("CreateWorkspace failed: %v", err)
	}

	// Verify AGENTS.md was NOT copied to workspace (isolation disabled)
	workspaceAgents := filepath.Join(projectDir, "workspaces", workspaceID, "AGENTS.md")
	if _, err := os.Stat(workspaceAgents); err == nil {
		t.Error("AGENTS.md should NOT be copied when workspace isolation is disabled")
	}
}
