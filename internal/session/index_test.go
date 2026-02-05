package session

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestSessionIndex_AddAndGet(t *testing.T) {
	idx := NewSessionIndex(t.TempDir())

	tests := []struct {
		name    string
		entry   *SessionIndexEntry
		wantGet bool
	}{
		{
			name: "add and get single entry",
			entry: &SessionIndexEntry{
				SessionID:   "sess-1",
				ProjectID:   "proj-1",
				WorkspaceID: "ws-1",
				Status:      StatusActive,
			},
			wantGet: true,
		},
		{
			name: "add entry with different status",
			entry: &SessionIndexEntry{
				SessionID:   "sess-2",
				ProjectID:   "proj-1",
				WorkspaceID: "ws-1",
				Status:      StatusCompleted,
			},
			wantGet: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx.Add(tt.entry)

			got, ok := idx.Get(tt.entry.SessionID)
			if ok != tt.wantGet {
				t.Errorf("Get() ok = %v, want %v", ok, tt.wantGet)
			}
			if ok && got.SessionID != tt.entry.SessionID {
				t.Errorf("Get() SessionID = %v, want %v", got.SessionID, tt.entry.SessionID)
			}
		})
	}
}

func TestSessionIndex_GetNonExistent(t *testing.T) {
	idx := NewSessionIndex(t.TempDir())

	_, ok := idx.Get("non-existent")
	if ok {
		t.Error("Get() should return false for non-existent session")
	}
}

func TestSessionIndex_GetByProject(t *testing.T) {
	idx := NewSessionIndex(t.TempDir())

	// Add sessions to different projects
	idx.Add(&SessionIndexEntry{SessionID: "sess-1", ProjectID: "proj-1", WorkspaceID: "ws-1", Status: StatusActive})
	idx.Add(&SessionIndexEntry{SessionID: "sess-2", ProjectID: "proj-1", WorkspaceID: "ws-1", Status: StatusActive})
	idx.Add(&SessionIndexEntry{SessionID: "sess-3", ProjectID: "proj-2", WorkspaceID: "ws-1", Status: StatusActive})

	tests := []struct {
		name      string
		projectID string
		wantCount int
	}{
		{"project with 2 sessions", "proj-1", 2},
		{"project with 1 session", "proj-2", 1},
		{"project with no sessions", "proj-3", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessions := idx.GetByProject(tt.projectID)
			if len(sessions) != tt.wantCount {
				t.Errorf("GetByProject() count = %v, want %v", len(sessions), tt.wantCount)
			}
		})
	}
}

func TestSessionIndex_GetByStatus(t *testing.T) {
	idx := NewSessionIndex(t.TempDir())

	idx.Add(&SessionIndexEntry{SessionID: "sess-1", ProjectID: "proj-1", WorkspaceID: "ws-1", Status: StatusActive})
	idx.Add(&SessionIndexEntry{SessionID: "sess-2", ProjectID: "proj-1", WorkspaceID: "ws-1", Status: StatusActive})
	idx.Add(&SessionIndexEntry{SessionID: "sess-3", ProjectID: "proj-1", WorkspaceID: "ws-1", Status: StatusCompleted})
	idx.Add(&SessionIndexEntry{SessionID: "sess-4", ProjectID: "proj-1", WorkspaceID: "ws-1", Status: StatusFailed})

	tests := []struct {
		name      string
		status    Status
		wantCount int
	}{
		{"active sessions", StatusActive, 2},
		{"completed sessions", StatusCompleted, 1},
		{"failed sessions", StatusFailed, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessions := idx.GetByStatus(tt.status)
			if len(sessions) != tt.wantCount {
				t.Errorf("GetByStatus() count = %v, want %v", len(sessions), tt.wantCount)
			}
		})
	}
}

func TestSessionIndex_UpdateStatus(t *testing.T) {
	idx := NewSessionIndex(t.TempDir())

	idx.Add(&SessionIndexEntry{SessionID: "sess-1", ProjectID: "proj-1", WorkspaceID: "ws-1", Status: StatusActive})

	// Update status
	ok := idx.UpdateStatus("sess-1", StatusCompleted)
	if !ok {
		t.Error("UpdateStatus() should return true for existing session")
	}

	// Verify status changed
	entry, _ := idx.Get("sess-1")
	if entry.Status != StatusCompleted {
		t.Errorf("Status = %v, want %v", entry.Status, StatusCompleted)
	}

	// Verify status indices updated
	active := idx.GetByStatus(StatusActive)
	if len(active) != 0 {
		t.Errorf("GetByStatus(Active) count = %v, want 0", len(active))
	}

	completed := idx.GetByStatus(StatusCompleted)
	if len(completed) != 1 {
		t.Errorf("GetByStatus(Completed) count = %v, want 1", len(completed))
	}
}

func TestSessionIndex_UpdateStatus_NonExistent(t *testing.T) {
	idx := NewSessionIndex(t.TempDir())

	ok := idx.UpdateStatus("non-existent", StatusCompleted)
	if ok {
		t.Error("UpdateStatus() should return false for non-existent session")
	}
}

func TestSessionIndex_Remove(t *testing.T) {
	idx := NewSessionIndex(t.TempDir())

	idx.Add(&SessionIndexEntry{SessionID: "sess-1", ProjectID: "proj-1", WorkspaceID: "ws-1", Status: StatusActive})

	// Remove session
	idx.Remove("sess-1")

	// Verify removed
	_, ok := idx.Get("sess-1")
	if ok {
		t.Error("Get() should return false after Remove()")
	}

	// Verify indices cleaned up
	if len(idx.GetByProject("proj-1")) != 0 {
		t.Error("GetByProject() should return empty after Remove()")
	}
	if len(idx.GetByStatus(StatusActive)) != 0 {
		t.Error("GetByStatus() should return empty after Remove()")
	}
}

func TestSessionIndex_Remove_NonExistent(t *testing.T) {
	idx := NewSessionIndex(t.TempDir())

	// Should not panic
	idx.Remove("non-existent")
}

func TestSessionIndex_Count(t *testing.T) {
	idx := NewSessionIndex(t.TempDir())

	if idx.Count() != 0 {
		t.Error("Count() should be 0 for empty index")
	}

	idx.Add(&SessionIndexEntry{SessionID: "sess-1", ProjectID: "proj-1", WorkspaceID: "ws-1", Status: StatusActive})
	idx.Add(&SessionIndexEntry{SessionID: "sess-2", ProjectID: "proj-1", WorkspaceID: "ws-1", Status: StatusActive})

	if idx.Count() != 2 {
		t.Errorf("Count() = %v, want 2", idx.Count())
	}

	idx.Remove("sess-1")
	if idx.Count() != 1 {
		t.Errorf("Count() = %v, want 1 after remove", idx.Count())
	}
}

func TestSessionIndex_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	idx := NewSessionIndex(dir)

	// Add entries
	idx.Add(&SessionIndexEntry{SessionID: "sess-1", ProjectID: "proj-1", WorkspaceID: "ws-1", Status: StatusActive})
	idx.Add(&SessionIndexEntry{SessionID: "sess-2", ProjectID: "proj-1", WorkspaceID: "ws-2", Status: StatusCompleted})

	// Save
	if err := idx.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	indexPath := filepath.Join(dir, "sessions_index.json")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Fatal("Index file not created")
	}

	// Create new index and load
	idx2 := NewSessionIndex(dir)
	if err := idx2.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify entries loaded
	if idx2.Count() != 2 {
		t.Errorf("Loaded index Count() = %v, want 2", idx2.Count())
	}

	entry, ok := idx2.Get("sess-1")
	if !ok {
		t.Fatal("Get() should return true for loaded session")
	}
	if entry.Status != StatusActive {
		t.Errorf("Loaded entry Status = %v, want Active", entry.Status)
	}

	// Verify secondary indices rebuilt
	if len(idx2.GetByProject("proj-1")) != 2 {
		t.Error("GetByProject() should work after Load()")
	}
	if len(idx2.GetByStatus(StatusActive)) != 1 {
		t.Error("GetByStatus() should work after Load()")
	}
}

func TestSessionIndex_Load_NoFile(t *testing.T) {
	idx := NewSessionIndex(t.TempDir())

	// Should not error on missing file
	if err := idx.Load(); err != nil {
		t.Errorf("Load() should not error on missing file, got: %v", err)
	}

	if idx.Count() != 0 {
		t.Error("Count() should be 0 after loading missing file")
	}
}

func TestSessionIndex_ConcurrentAccess(t *testing.T) {
	idx := NewSessionIndex(t.TempDir())
	var wg sync.WaitGroup

	// Concurrent adds
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			idx.Add(&SessionIndexEntry{
				SessionID:   "sess-" + string(rune('0'+i%10)),
				ProjectID:   "proj-1",
				WorkspaceID: "ws-1",
				Status:      StatusActive,
			})
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			idx.GetByProject("proj-1")
			idx.GetByStatus(StatusActive)
			idx.Count()
		}()
	}

	wg.Wait()

	// Should not panic or produce inconsistent state
	if idx.Count() < 1 {
		t.Error("Index should have at least 1 entry")
	}
}

func TestSessionIndex_Add_UpdatesExisting(t *testing.T) {
	idx := NewSessionIndex(t.TempDir())

	// Add entry
	idx.Add(&SessionIndexEntry{SessionID: "sess-1", ProjectID: "proj-1", WorkspaceID: "ws-1", Status: StatusActive})

	// Add again with different status (update)
	idx.Add(&SessionIndexEntry{SessionID: "sess-1", ProjectID: "proj-1", WorkspaceID: "ws-1", Status: StatusCompleted})

	// Should update, not create duplicate
	if idx.Count() != 1 {
		t.Errorf("Count() = %v, want 1 (no duplicate)", idx.Count())
	}

	// Status should be updated
	entry, _ := idx.Get("sess-1")
	if entry.Status != StatusCompleted {
		t.Errorf("Status = %v, want Completed", entry.Status)
	}

	// Old status index should be cleaned
	if len(idx.GetByStatus(StatusActive)) != 0 {
		t.Error("Old status index not cleaned on update")
	}
}
