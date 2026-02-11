package session

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "projects")

	mgr := NewManager(sessionsDir, nil, "http://localhost:8080/mcp")

	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}
	if mgr.sessionsBaseDir != sessionsDir {
		t.Errorf("sessionsBaseDir = %q, want %q", mgr.sessionsBaseDir, sessionsDir)
	}
	if mgr.persistentIndex == nil {
		t.Error("expected non-nil persistentIndex")
	}
	if mgr.sessionLocks == nil {
		t.Error("expected non-nil sessionLocks")
	}
}

func TestManagerLoadIndex(t *testing.T) {
	tmpDir := t.TempDir()
	// NewManager uses filepath.Dir(sessionsBaseDir) as parent, then adds "data"
	// So if sessionsBaseDir = tmpDir/projects, dataDir = tmpDir/data
	sessionsDir := filepath.Join(tmpDir, "projects")
	dataDir := filepath.Join(tmpDir, "data")
	_ = os.MkdirAll(dataDir, 0o755)

	// Create index file with some entries (note: file format is array not map)
	indexData := []*SessionIndexEntry{
		{SessionID: "gogol_20250101_120000_abc12345", ProjectID: "550e8400-e29b-41d4-a716-446655440001", Status: StatusActive},
		{SessionID: "gogol_20250101_120001_def67890", ProjectID: "550e8400-e29b-41d4-a716-446655440002", Status: StatusCompleted},
	}
	indexBytes, _ := json.Marshal(indexData)
	_ = os.WriteFile(filepath.Join(dataDir, "sessions_index.json"), indexBytes, 0o644)

	mgr := NewManager(sessionsDir, nil, "http://localhost:8080/mcp")

	if err := mgr.LoadIndex(); err != nil {
		t.Fatalf("LoadIndex() error = %v", err)
	}

	// Verify entries were loaded
	entry, ok := mgr.persistentIndex.Get("gogol_20250101_120000_abc12345")
	if !ok {
		t.Error("expected to find session in index")
	}
	if entry.ProjectID != "550e8400-e29b-41d4-a716-446655440001" {
		t.Errorf("session ProjectID = %q, want %q", entry.ProjectID, "550e8400-e29b-41d4-a716-446655440001")
	}
}

func TestManagerSaveSession(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "projects")
	projectID := "550e8400-e29b-41d4-a716-446655440001"
	sessionID := "gogol_20250101_120000_abc12345"

	// Create session directory
	sessDir := filepath.Join(sessionsDir, projectID, "sessions")
	_ = os.MkdirAll(sessDir, 0o755)

	mgr := NewManager(sessionsDir, nil, "http://localhost:8080/mcp")

	session := &Session{
		SessionID:   sessionID,
		ProjectID:   projectID,
		WorkspaceID: "workspace-1",
		Status:      StatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Turns: []Turn{
			{TurnNumber: 1, Prompt: "Hello"},
		},
	}

	if err := mgr.SaveSession(session); err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}

	// Verify file was created
	sessionPath := filepath.Join(sessDir, sessionID+".json")
	if _, err := os.Stat(sessionPath); err != nil {
		t.Errorf("session file not created: %v", err)
	}

	// Verify session was indexed
	entry, ok := mgr.persistentIndex.Get(sessionID)
	if !ok {
		t.Error("session not added to index")
	}
	if entry.ProjectID != projectID {
		t.Errorf("indexed ProjectID = %q, want %q", entry.ProjectID, projectID)
	}
}

func TestManagerLoad(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "projects")
	projectID := "550e8400-e29b-41d4-a716-446655440001"
	sessionID := "gogol_20250101_120000_abc12345"

	// Create session file
	sessDir := filepath.Join(sessionsDir, projectID, "sessions")
	_ = os.MkdirAll(sessDir, 0o755)

	session := &Session{
		SessionID:   sessionID,
		ProjectID:   projectID,
		WorkspaceID: "660e8400-e29b-41d4-a716-446655440001",
		Status:      StatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	sessionData, _ := json.Marshal(session)
	_ = os.WriteFile(filepath.Join(sessDir, sessionID+".json"), sessionData, 0o644)

	mgr := NewManager(sessionsDir, nil, "http://localhost:8080/mcp")

	t.Run("load existing session", func(t *testing.T) {
		loaded, err := mgr.Load(sessionID)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if loaded.SessionID != sessionID {
			t.Errorf("SessionID = %q, want %q", loaded.SessionID, sessionID)
		}
		if loaded.ProjectID != projectID {
			t.Errorf("ProjectID = %q, want %q", loaded.ProjectID, projectID)
		}
	})

	t.Run("load indexed session", func(t *testing.T) {
		// Session should now be indexed - reload should be O(1)
		loaded, err := mgr.Load(sessionID)
		if err != nil {
			t.Fatalf("Load() cached error = %v", err)
		}
		if loaded.SessionID != sessionID {
			t.Errorf("SessionID = %q, want %q", loaded.SessionID, sessionID)
		}
	})

	t.Run("load non-existent session", func(t *testing.T) {
		_, err := mgr.Load("gogol_20250101_000000_00000000")
		if err == nil {
			t.Error("expected error for non-existent session")
		}
	})

	t.Run("invalid session ID", func(t *testing.T) {
		_, err := mgr.Load("invalid")
		if err == nil {
			t.Error("expected error for invalid session ID")
		}
	})
}

func TestManagerList(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "projects")
	projectID := "550e8400-e29b-41d4-a716-446655440001"

	sessDir := filepath.Join(sessionsDir, projectID, "sessions")
	_ = os.MkdirAll(sessDir, 0o755)

	// Create multiple sessions
	sessions := []*Session{
		{SessionID: "gogol_20250101_100000_aaa11111", ProjectID: projectID, Status: StatusActive, CreatedAt: time.Now()},
		{SessionID: "gogol_20250101_110000_bbb22222", ProjectID: projectID, Status: StatusCompleted, CreatedAt: time.Now()},
		{SessionID: "gogol_20250101_120000_ccc33333", ProjectID: projectID, Status: StatusFailed, CreatedAt: time.Now()},
	}

	for _, s := range sessions {
		data, _ := json.Marshal(s)
		_ = os.WriteFile(filepath.Join(sessDir, s.SessionID+".json"), data, 0o644)
	}

	mgr := NewManager(sessionsDir, nil, "http://localhost:8080/mcp")

	t.Run("list all sessions", func(t *testing.T) {
		list, err := mgr.List(projectID, nil)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(list) != 3 {
			t.Errorf("len(list) = %d, want 3", len(list))
		}
	})

	t.Run("filter by status", func(t *testing.T) {
		active := StatusActive
		list, err := mgr.List(projectID, &active)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(list) != 1 {
			t.Errorf("len(filtered) = %d, want 1", len(list))
		}
		if list[0].Status != StatusActive {
			t.Errorf("filtered status = %q, want %q", list[0].Status, StatusActive)
		}
	})

	t.Run("list non-existent project", func(t *testing.T) {
		list, err := mgr.List("550e8400-e29b-41d4-a716-446655440099", nil)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(list) != 0 {
			t.Errorf("len(list) = %d, want 0", len(list))
		}
	})

	t.Run("invalid project ID", func(t *testing.T) {
		_, err := mgr.List("../escape", nil)
		if err == nil {
			t.Error("expected error for invalid project ID")
		}
	})
}

func TestManagerEnd(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "projects")
	projectID := "550e8400-e29b-41d4-a716-446655440001"
	sessionID := "gogol_20250101_120000_abc12345"

	sessDir := filepath.Join(sessionsDir, projectID, "sessions")
	_ = os.MkdirAll(sessDir, 0o755)

	session := &Session{
		SessionID: sessionID,
		ProjectID: projectID,
		Status:    StatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	data, _ := json.Marshal(session)
	_ = os.WriteFile(filepath.Join(sessDir, sessionID+".json"), data, 0o644)

	mgr := NewManager(sessionsDir, nil, "http://localhost:8080/mcp")

	t.Run("end active session", func(t *testing.T) {
		err := mgr.End(sessionID)
		if err != nil {
			t.Fatalf("End() error = %v", err)
		}

		// Verify status changed
		loaded, _ := mgr.Load(sessionID)
		if loaded.Status != StatusCompleted {
			t.Errorf("Status = %q, want %q", loaded.Status, StatusCompleted)
		}
	})

	t.Run("end non-existent session", func(t *testing.T) {
		err := mgr.End("gogol_20250101_000000_00000000")
		if err == nil {
			t.Error("expected error for non-existent session")
		}
	})

	t.Run("invalid session ID", func(t *testing.T) {
		err := mgr.End("invalid")
		if err == nil {
			t.Error("expected error for invalid session ID")
		}
	})
}

func TestManagerAddChildSession(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "projects")
	projectID := "550e8400-e29b-41d4-a716-446655440001"
	parentID := "gogol_20250101_120000_aabbccdd"
	childID := "gogol_20250101_120001_eeff0011"

	sessDir := filepath.Join(sessionsDir, projectID, "sessions")
	_ = os.MkdirAll(sessDir, 0o755)

	parent := &Session{
		SessionID: parentID,
		ProjectID: projectID,
		Status:    StatusActive,
		Depth:     0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	child := &Session{
		SessionID: childID,
		ProjectID: projectID,
		Status:    StatusActive,
		Depth:     1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	parentData, _ := json.Marshal(parent)
	childData, _ := json.Marshal(child)
	_ = os.WriteFile(filepath.Join(sessDir, parentID+".json"), parentData, 0o644)
	_ = os.WriteFile(filepath.Join(sessDir, childID+".json"), childData, 0o644)

	mgr := NewManager(sessionsDir, nil, "http://localhost:8080/mcp")

	t.Run("add valid child", func(t *testing.T) {
		err := mgr.AddChildSession(parentID, childID)
		if err != nil {
			t.Fatalf("AddChildSession() error = %v", err)
		}

		loaded, _ := mgr.Load(parentID)
		if len(loaded.ChildSessions) != 1 {
			t.Errorf("ChildSessions count = %d, want 1", len(loaded.ChildSessions))
		}
		if loaded.ChildSessions[0] != childID {
			t.Errorf("ChildSessions[0] = %q, want %q", loaded.ChildSessions[0], childID)
		}
	})

	t.Run("invalid depth", func(t *testing.T) {
		wrongDepthID := "gogol_20250101_120002_22334455"
		wrongDepth := &Session{
			SessionID: wrongDepthID,
			ProjectID: projectID,
			Depth:     5, // Wrong depth
		}
		wrongDepthData, _ := json.Marshal(wrongDepth)
		_ = os.WriteFile(filepath.Join(sessDir, wrongDepthID+".json"), wrongDepthData, 0o644)

		err := mgr.AddChildSession(parentID, wrongDepthID)
		if err == nil {
			t.Error("expected error for invalid depth")
		}
	})
}

func TestManagerRecoverStaleSessions(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "projects")
	projectID := "550e8400-e29b-41d4-a716-446655440001"

	sessDir := filepath.Join(sessionsDir, projectID, "sessions")
	_ = os.MkdirAll(sessDir, 0o755)

	now := time.Now()
	staleTime := now.Add(-2 * time.Hour) // 2 hours ago

	sessions := []*Session{
		{SessionID: "gogol_20250101_100000_aaa11111", ProjectID: projectID, Status: StatusActive, UpdatedAt: staleTime},
		{SessionID: "gogol_20250101_110000_bbb22222", ProjectID: projectID, Status: StatusActive, UpdatedAt: staleTime},
		{SessionID: "gogol_20250101_120000_ccc33333", ProjectID: projectID, Status: StatusActive, UpdatedAt: now},
		{SessionID: "gogol_20250101_130000_ddd44444", ProjectID: projectID, Status: StatusCompleted, UpdatedAt: staleTime},
	}

	for _, s := range sessions {
		data, _ := json.Marshal(s)
		_ = os.WriteFile(filepath.Join(sessDir, s.SessionID+".json"), data, 0o644)
	}

	mgr := NewManager(sessionsDir, nil, "http://localhost:8080/mcp")

	t.Run("recover via filesystem scan", func(t *testing.T) {
		recovered, err := mgr.RecoverStaleSessions(time.Hour)
		if err != nil {
			t.Fatalf("RecoverStaleSessions() error = %v", err)
		}
		if recovered != 2 {
			t.Errorf("recovered = %d, want 2", recovered)
		}

		// Verify stale sessions are now failed
		stale1, err := mgr.Load("gogol_20250101_100000_aaa11111")
		if err != nil {
			t.Fatalf("Load stale1 error = %v", err)
		}
		if stale1.Status != StatusFailed {
			t.Errorf("stale1 status = %q, want %q", stale1.Status, StatusFailed)
		}

		// Verify fresh session is still active
		fresh, err := mgr.Load("gogol_20250101_120000_ccc33333")
		if err != nil {
			t.Fatalf("Load fresh error = %v", err)
		}
		if fresh.Status != StatusActive {
			t.Errorf("fresh status = %q, want %q", fresh.Status, StatusActive)
		}
	})

	t.Run("recover via index", func(t *testing.T) {
		// Reset stale sessions to active
		for _, id := range []string{"gogol_20250101_100000_aaa11111", "gogol_20250101_110000_bbb22222"} {
			s, _ := mgr.Load(id)
			s.Status = StatusActive
			s.UpdatedAt = staleTime
			_ = mgr.SaveSession(s)
		}

		// Index is now populated - recovery should use fast path
		recovered, err := mgr.RecoverStaleSessions(time.Hour)
		if err != nil {
			t.Fatalf("RecoverStaleSessions() error = %v", err)
		}
		if recovered != 2 {
			t.Errorf("recovered = %d, want 2", recovered)
		}
	})
}

func TestManagerConcurrentSave(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "projects")
	projectID := "550e8400-e29b-41d4-a716-446655440001"
	sessionID := "gogol_20250101_120000_abc12345"

	sessDir := filepath.Join(sessionsDir, projectID, "sessions")
	_ = os.MkdirAll(sessDir, 0o755)

	session := &Session{
		SessionID: sessionID,
		ProjectID: projectID,
		Status:    StatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Turns:     []Turn{},
	}
	data, _ := json.Marshal(session)
	_ = os.WriteFile(filepath.Join(sessDir, sessionID+".json"), data, 0o644)

	mgr := NewManager(sessionsDir, nil, "http://localhost:8080/mcp")

	// Run concurrent saves
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s, _ := mgr.Load(sessionID)
			s.Turns = append(s.Turns, Turn{TurnNumber: i, Prompt: "test"})
			s.UpdatedAt = time.Now()
			_ = mgr.SaveSession(s)
		}(i)
	}
	wg.Wait()

	// Verify file is valid JSON
	loaded, err := mgr.Load(sessionID)
	if err != nil {
		t.Fatalf("Load after concurrent saves error = %v", err)
	}
	// We can't guarantee turn count due to race conditions in reads,
	// but file should be valid and have at least some turns
	if len(loaded.Turns) == 0 {
		t.Error("expected at least some turns after concurrent saves")
	}
}

func TestGenerateSessionID(t *testing.T) {
	id1 := generateSessionID()
	id2 := generateSessionID()

	if id1 == id2 {
		t.Error("expected unique session IDs")
	}

	// Should start with "gogol_"
	if len(id1) < 6 || id1[:6] != "gogol_" {
		t.Errorf("session ID should start with 'gogol_', got %q", id1)
	}
}

func TestGenerateExplorationID(t *testing.T) {
	id1 := GenerateExplorationID()
	id2 := GenerateExplorationID()

	if id1 == id2 {
		t.Error("expected unique exploration IDs")
	}

	// Should start with "exp_"
	if len(id1) < 4 || id1[:4] != "exp_" {
		t.Errorf("exploration ID should start with 'exp_', got %q", id1)
	}
}

func TestLoadSessionFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "projects")

	mgr := NewManager(sessionsDir, nil, "http://localhost:8080/mcp")

	t.Run("valid file", func(t *testing.T) {
		session := &Session{
			SessionID: "test-session",
			ProjectID: "test-project",
			Status:    StatusActive,
		}
		data, _ := json.Marshal(session)
		filePath := filepath.Join(tmpDir, "valid.json")
		_ = os.WriteFile(filePath, data, 0o644)

		loaded, err := mgr.loadSessionFromFile(filePath)
		if err != nil {
			t.Fatalf("loadSessionFromFile() error = %v", err)
		}
		if loaded.SessionID != "test-session" {
			t.Errorf("SessionID = %q, want %q", loaded.SessionID, "test-session")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "invalid.json")
		_ = os.WriteFile(filePath, []byte("not json"), 0o644)

		_, err := mgr.loadSessionFromFile(filePath)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := mgr.loadSessionFromFile(filepath.Join(tmpDir, "missing.json"))
		if err == nil {
			t.Error("expected error for missing file")
		}
	})
}

func TestIndexSessionAndLookup(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "projects")

	mgr := NewManager(sessionsDir, nil, "http://localhost:8080/mcp")

	session := &Session{
		SessionID:   "gogol_20250101_120000_abc12345",
		ProjectID:   "550e8400-e29b-41d4-a716-446655440001",
		WorkspaceID: "660e8400-e29b-41d4-a716-446655440001",
		Status:      StatusActive,
	}

	// Index the session
	mgr.indexSession(session)

	// Lookup should work
	projectID, ok := mgr.lookupProject(session.SessionID)
	if !ok {
		t.Error("expected lookup to succeed")
	}
	if projectID != "550e8400-e29b-41d4-a716-446655440001" {
		t.Errorf("projectID = %q, want %q", projectID, "550e8400-e29b-41d4-a716-446655440001")
	}

	// Non-existent lookup
	_, ok = mgr.lookupProject("gogol_20250101_000000_00000000")
	if ok {
		t.Error("expected lookup to fail for non-existent session")
	}
}

func TestUpdateIndexStatus(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "projects")

	mgr := NewManager(sessionsDir, nil, "http://localhost:8080/mcp")

	session := &Session{
		SessionID: "gogol_20250101_120000_abc12345",
		ProjectID: "550e8400-e29b-41d4-a716-446655440001",
		Status:    StatusActive,
	}

	mgr.indexSession(session)

	// Update status
	mgr.updateIndexStatus(session.SessionID, StatusCompleted)

	// Verify via GetByStatus
	active := mgr.persistentIndex.GetByStatus(StatusActive)
	if len(active) != 0 {
		t.Errorf("expected 0 active sessions, got %d", len(active))
	}

	completed := mgr.persistentIndex.GetByStatus(StatusCompleted)
	if len(completed) != 1 {
		t.Errorf("expected 1 completed session, got %d", len(completed))
	}
}

func TestCleanupOldSessions(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "projects")
	projectID := "550e8400-e29b-41d4-a716-446655440001"

	sessDir := filepath.Join(sessionsDir, projectID, "sessions")
	_ = os.MkdirAll(sessDir, 0o755)

	now := time.Now()
	oldTime := now.Add(-48 * time.Hour)   // 2 days ago
	recentTime := now.Add(-1 * time.Hour) // 1 hour ago

	// Create sessions with different ages and statuses
	sessions := []*Session{
		{SessionID: "gogol_20250101_100000_old_done", ProjectID: projectID, Status: StatusCompleted, UpdatedAt: oldTime},
		{SessionID: "gogol_20250101_110000_old_fail", ProjectID: projectID, Status: StatusFailed, UpdatedAt: oldTime},
		{SessionID: "gogol_20250101_120000_old_active", ProjectID: projectID, Status: StatusActive, UpdatedAt: oldTime},
		{SessionID: "gogol_20250101_130000_recent_done", ProjectID: projectID, Status: StatusCompleted, UpdatedAt: recentTime},
	}

	for _, s := range sessions {
		data, _ := json.Marshal(s)
		filePath := filepath.Join(sessDir, s.SessionID+".json")
		_ = os.WriteFile(filePath, data, 0o644)
		// Set file modification time to match UpdatedAt
		_ = os.Chtimes(filePath, s.UpdatedAt, s.UpdatedAt)
	}

	mgr := NewManager(sessionsDir, nil, "http://localhost:8080/mcp")
	// Index all sessions
	for _, s := range sessions {
		mgr.indexSession(s)
	}

	t.Run("cleanup old completed sessions", func(t *testing.T) {
		deleted, err := mgr.CleanupOldSessions(projectID, 24*time.Hour)
		if err != nil {
			t.Fatalf("CleanupOldSessions() error = %v", err)
		}
		// Should delete 2: old_done and old_fail (not old_active or recent_done)
		if deleted != 2 {
			t.Errorf("deleted = %d, want 2", deleted)
		}

		// Verify old completed sessions are gone
		if _, err := os.Stat(filepath.Join(sessDir, "gogol_20250101_100000_old_done.json")); !errors.Is(err, fs.ErrNotExist) {
			t.Error("expected old_done session file to be deleted")
		}
		if _, err := os.Stat(filepath.Join(sessDir, "gogol_20250101_110000_old_fail.json")); !errors.Is(err, fs.ErrNotExist) {
			t.Error("expected old_fail session file to be deleted")
		}

		// Verify active session is preserved (even if old)
		if _, err := os.Stat(filepath.Join(sessDir, "gogol_20250101_120000_old_active.json")); err != nil {
			t.Error("expected old_active session file to be preserved")
		}

		// Verify recent session is preserved
		if _, err := os.Stat(filepath.Join(sessDir, "gogol_20250101_130000_recent_done.json")); err != nil {
			t.Error("expected recent_done session file to be preserved")
		}
	})

	t.Run("cleanup non-existent project", func(t *testing.T) {
		deleted, err := mgr.CleanupOldSessions("550e8400-e29b-41d4-a716-446655440099", 24*time.Hour)
		if err != nil {
			t.Fatalf("CleanupOldSessions() error = %v", err)
		}
		if deleted != 0 {
			t.Errorf("deleted = %d, want 0", deleted)
		}
	})

	t.Run("invalid project ID", func(t *testing.T) {
		_, err := mgr.CleanupOldSessions("../escape", 24*time.Hour)
		if err == nil {
			t.Error("expected error for invalid project ID")
		}
	})
}

func TestCleanupAllOldSessions(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "projects")

	projectID1 := "550e8400-e29b-41d4-a716-446655440001"
	projectID2 := "550e8400-e29b-41d4-a716-446655440002"

	sessDir1 := filepath.Join(sessionsDir, projectID1, "sessions")
	sessDir2 := filepath.Join(sessionsDir, projectID2, "sessions")
	_ = os.MkdirAll(sessDir1, 0o755)
	_ = os.MkdirAll(sessDir2, 0o755)

	oldTime := time.Now().Add(-48 * time.Hour)

	// Create old completed sessions in both projects
	sessions := []*Session{
		{SessionID: "gogol_20250101_100000_proj1_old", ProjectID: projectID1, Status: StatusCompleted, UpdatedAt: oldTime},
		{SessionID: "gogol_20250101_110000_proj2_old1", ProjectID: projectID2, Status: StatusCompleted, UpdatedAt: oldTime},
		{SessionID: "gogol_20250101_120000_proj2_old2", ProjectID: projectID2, Status: StatusFailed, UpdatedAt: oldTime},
	}

	for _, s := range sessions {
		data, _ := json.Marshal(s)
		var filePath string
		if s.ProjectID == projectID1 {
			filePath = filepath.Join(sessDir1, s.SessionID+".json")
		} else {
			filePath = filepath.Join(sessDir2, s.SessionID+".json")
		}
		_ = os.WriteFile(filePath, data, 0o644)
		_ = os.Chtimes(filePath, s.UpdatedAt, s.UpdatedAt)
	}

	mgr := NewManager(sessionsDir, nil, "http://localhost:8080/mcp")

	results, err := mgr.CleanupAllOldSessions(24 * time.Hour)
	if err != nil {
		t.Fatalf("CleanupAllOldSessions() error = %v", err)
	}

	// Should have results for both projects
	if len(results) != 2 {
		t.Errorf("len(results) = %d, want 2", len(results))
	}

	if results[projectID1] != 1 {
		t.Errorf("results[project1] = %d, want 1", results[projectID1])
	}
	if results[projectID2] != 2 {
		t.Errorf("results[project2] = %d, want 2", results[projectID2])
	}
}
