package session

import (
	"sync"
	"testing"
	"time"

	"github.com/HyphaGroup/oubliette/internal/agent"
)

func TestActiveStatus_Constants(t *testing.T) {
	tests := []struct {
		status ActiveStatus
		want   string
	}{
		{ActiveStatusRunning, "running"},
		{ActiveStatusPaused, "paused"},
		{ActiveStatusCompleted, "completed"},
		{ActiveStatusFailed, "failed"},
		{ActiveStatusTimedOut, "timed_out"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("ActiveStatus = %q, want %q", tt.status, tt.want)
		}
	}
}

func TestNewActiveSession(t *testing.T) {
	sess := NewActiveSession("sess-1", "proj-1", "ws-1", "container-1", nil)

	if sess.SessionID != "sess-1" {
		t.Errorf("SessionID = %q, want %q", sess.SessionID, "sess-1")
	}
	if sess.ProjectID != "proj-1" {
		t.Errorf("ProjectID = %q, want %q", sess.ProjectID, "proj-1")
	}
	if sess.WorkspaceID != "ws-1" {
		t.Errorf("WorkspaceID = %q, want %q", sess.WorkspaceID, "ws-1")
	}
	if sess.ContainerID != "container-1" {
		t.Errorf("ContainerID = %q, want %q", sess.ContainerID, "container-1")
	}
	if sess.Status != ActiveStatusRunning {
		t.Errorf("Status = %q, want %q", sess.Status, ActiveStatusRunning)
	}
	if sess.EventBuffer == nil {
		t.Error("EventBuffer should be initialized")
	}
	if sess.StartedAt.IsZero() {
		t.Error("StartedAt should be set")
	}
	if sess.LastActivity.IsZero() {
		t.Error("LastActivity should be set")
	}
}

func TestActiveSession_SendMessage_NoExecutor(t *testing.T) {
	sess := NewActiveSession("sess-1", "proj-1", "ws-1", "container-1", nil)

	err := sess.SendMessage("hello")
	if err == nil {
		t.Error("expected error when executor is nil")
	}
}

func TestActiveSession_GetEvents(t *testing.T) {
	sess := NewActiveSession("sess-1", "proj-1", "ws-1", "container-1", nil)

	// Add some events to buffer
	sess.EventBuffer.Append(&agent.StreamEvent{Type: "message", Text: "test1"})
	sess.EventBuffer.Append(&agent.StreamEvent{Type: "message", Text: "test2"})

	events, err := sess.GetEvents(-1)
	if err != nil {
		t.Fatalf("GetEvents() error = %v", err)
	}
	if len(events) != 2 {
		t.Errorf("len(events) = %d, want 2", len(events))
	}
}

func TestActiveSession_GetExecutor(t *testing.T) {
	sess := NewActiveSession("sess-1", "proj-1", "ws-1", "container-1", nil)

	// Initially nil
	if sess.GetExecutor() != nil {
		t.Error("expected nil executor")
	}
}

func TestActiveSession_CloseExecutor(t *testing.T) {
	sess := NewActiveSession("sess-1", "proj-1", "ws-1", "container-1", nil)

	// Should not panic with nil executor
	sess.CloseExecutor()

	if sess.GetExecutor() != nil {
		t.Error("executor should be nil after close")
	}
}

func TestActiveSession_IsRunning(t *testing.T) {
	sess := NewActiveSession("sess-1", "proj-1", "ws-1", "container-1", nil)

	if !sess.IsRunning() {
		t.Error("new session should be running")
	}

	sess.SetStatus(ActiveStatusCompleted, nil)
	if sess.IsRunning() {
		t.Error("completed session should not be running")
	}
}

func TestActiveSession_SetStatus(t *testing.T) {
	sess := NewActiveSession("sess-1", "proj-1", "ws-1", "container-1", nil)

	sess.SetStatus(ActiveStatusFailed, nil)
	if sess.GetStatus() != ActiveStatusFailed {
		t.Errorf("GetStatus() = %q, want %q", sess.GetStatus(), ActiveStatusFailed)
	}

	sess.SetStatus(ActiveStatusCompleted, nil)
	if sess.GetStatus() != ActiveStatusCompleted {
		t.Errorf("GetStatus() = %q, want %q", sess.GetStatus(), ActiveStatusCompleted)
	}
}

func TestActiveSession_LastActivityTime(t *testing.T) {
	sess := NewActiveSession("sess-1", "proj-1", "ws-1", "container-1", nil)

	initialTime := sess.LastActivityTime()
	if initialTime.IsZero() {
		t.Error("LastActivityTime should not be zero")
	}

	// Activity time should be recent
	if time.Since(initialTime) > time.Second {
		t.Error("LastActivityTime should be within last second")
	}
}

func TestActiveSession_ConcurrentAccess(t *testing.T) {
	sess := NewActiveSession("sess-1", "proj-1", "ws-1", "container-1", nil)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(4)
		go func() {
			defer wg.Done()
			sess.IsRunning()
		}()
		go func() {
			defer wg.Done()
			sess.GetStatus()
		}()
		go func() {
			defer wg.Done()
			sess.LastActivityTime()
		}()
		go func() {
			defer wg.Done()
			sess.GetExecutor()
		}()
	}
	wg.Wait()
}

func TestNewActiveSessionManager(t *testing.T) {
	mgr := NewActiveSessionManager(10, time.Hour)
	defer mgr.Close()

	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}
	if mgr.maxPerProj != 10 {
		t.Errorf("maxPerProj = %d, want 10", mgr.maxPerProj)
	}
	if mgr.idleTimeout != time.Hour {
		t.Errorf("idleTimeout = %v, want %v", mgr.idleTimeout, time.Hour)
	}
}

func TestNewActiveSessionManager_Defaults(t *testing.T) {
	mgr := NewActiveSessionManager(0, 0) // Use defaults
	defer mgr.Close()

	if mgr.maxPerProj != DefaultMaxActiveSessions {
		t.Errorf("maxPerProj = %d, want %d", mgr.maxPerProj, DefaultMaxActiveSessions)
	}
	if mgr.idleTimeout != DefaultSessionIdleTimeout {
		t.Errorf("idleTimeout = %v, want %v", mgr.idleTimeout, DefaultSessionIdleTimeout)
	}
}

func TestActiveSessionManager_Register(t *testing.T) {
	mgr := NewActiveSessionManager(5, time.Hour)
	defer mgr.Close()

	sess := NewActiveSession("sess-1", "proj-1", "ws-1", "container-1", nil)
	err := mgr.Register(sess)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if mgr.Count() != 1 {
		t.Errorf("Count() = %d, want 1", mgr.Count())
	}
}

func TestActiveSessionManager_Register_MaxPerProject(t *testing.T) {
	mgr := NewActiveSessionManager(2, time.Hour)
	defer mgr.Close()

	// Register max sessions for project
	for i := 0; i < 2; i++ {
		sess := NewActiveSession("sess-"+string(rune('a'+i)), "proj-1", "ws-"+string(rune('a'+i)), "container-1", nil)
		err := mgr.Register(sess)
		if err != nil {
			t.Fatalf("Register() error = %v", err)
		}
	}

	// Try to register one more - should fail
	sess := NewActiveSession("sess-c", "proj-1", "ws-c", "container-1", nil)
	err := mgr.Register(sess)
	if err == nil {
		t.Error("expected error when max sessions reached")
	}

	// Different project should still work
	sess2 := NewActiveSession("sess-d", "proj-2", "ws-d", "container-1", nil)
	err = mgr.Register(sess2)
	if err != nil {
		t.Errorf("Register() for different project error = %v", err)
	}
}

func TestActiveSessionManager_Get(t *testing.T) {
	mgr := NewActiveSessionManager(5, time.Hour)
	defer mgr.Close()

	sess := NewActiveSession("sess-1", "proj-1", "ws-1", "container-1", nil)
	_ = mgr.Register(sess)

	t.Run("existing session", func(t *testing.T) {
		got, ok := mgr.Get("sess-1")
		if !ok {
			t.Error("expected to find session")
		}
		if got.SessionID != "sess-1" {
			t.Errorf("SessionID = %q, want %q", got.SessionID, "sess-1")
		}
	})

	t.Run("non-existent session", func(t *testing.T) {
		_, ok := mgr.Get("nonexistent")
		if ok {
			t.Error("expected not to find session")
		}
	})
}

func TestActiveSessionManager_GetByWorkspace(t *testing.T) {
	mgr := NewActiveSessionManager(5, time.Hour)
	defer mgr.Close()

	sess := NewActiveSession("sess-1", "proj-1", "ws-1", "container-1", nil)
	_ = mgr.Register(sess)

	t.Run("existing workspace", func(t *testing.T) {
		got, ok := mgr.GetByWorkspace("proj-1", "ws-1")
		if !ok {
			t.Error("expected to find session by workspace")
		}
		if got.SessionID != "sess-1" {
			t.Errorf("SessionID = %q, want %q", got.SessionID, "sess-1")
		}
	})

	t.Run("wrong workspace", func(t *testing.T) {
		_, ok := mgr.GetByWorkspace("proj-1", "ws-other")
		if ok {
			t.Error("expected not to find session for wrong workspace")
		}
	})

	t.Run("wrong project", func(t *testing.T) {
		_, ok := mgr.GetByWorkspace("proj-other", "ws-1")
		if ok {
			t.Error("expected not to find session for wrong project")
		}
	})
}

func TestActiveSessionManager_Remove(t *testing.T) {
	mgr := NewActiveSessionManager(5, time.Hour)
	defer mgr.Close()

	sess := NewActiveSession("sess-1", "proj-1", "ws-1", "container-1", nil)
	_ = mgr.Register(sess)

	mgr.Remove("sess-1")

	if mgr.Count() != 0 {
		t.Errorf("Count() = %d, want 0", mgr.Count())
	}

	_, ok := mgr.Get("sess-1")
	if ok {
		t.Error("session should be removed")
	}

	_, ok = mgr.GetByWorkspace("proj-1", "ws-1")
	if ok {
		t.Error("workspace index should be cleared")
	}
}

func TestActiveSessionManager_Remove_NonExistent(t *testing.T) {
	mgr := NewActiveSessionManager(5, time.Hour)
	defer mgr.Close()

	// Should not panic
	mgr.Remove("nonexistent")
}

func TestActiveSessionManager_SendMessage_NotFound(t *testing.T) {
	mgr := NewActiveSessionManager(5, time.Hour)
	defer mgr.Close()

	err := mgr.SendMessage("nonexistent", "hello")
	if err == nil {
		t.Error("expected error for non-existent session")
	}
}

func TestActiveSessionManager_SendMessage_NotRunning(t *testing.T) {
	mgr := NewActiveSessionManager(5, time.Hour)
	defer mgr.Close()

	sess := NewActiveSession("sess-1", "proj-1", "ws-1", "container-1", nil)
	_ = mgr.Register(sess)
	sess.SetStatus(ActiveStatusCompleted, nil)

	err := mgr.SendMessage("sess-1", "hello")
	if err == nil {
		t.Error("expected error for completed session")
	}
}

func TestActiveSessionManager_GetEvents_NotFound(t *testing.T) {
	mgr := NewActiveSessionManager(5, time.Hour)
	defer mgr.Close()

	_, err := mgr.GetEvents("nonexistent", -1)
	if err == nil {
		t.Error("expected error for non-existent session")
	}
}

func TestActiveSessionManager_GetLastEventIndex(t *testing.T) {
	mgr := NewActiveSessionManager(5, time.Hour)
	defer mgr.Close()

	sess := NewActiveSession("sess-1", "proj-1", "ws-1", "container-1", nil)
	_ = mgr.Register(sess)

	// Add some events
	sess.EventBuffer.Append(&agent.StreamEvent{Type: "message", Text: "test"})

	idx, err := mgr.GetLastEventIndex("sess-1")
	if err != nil {
		t.Fatalf("GetLastEventIndex() error = %v", err)
	}
	if idx != 0 {
		t.Errorf("LastEventIndex = %d, want 0", idx)
	}
}

func TestActiveSessionManager_GetLastEventIndex_NotFound(t *testing.T) {
	mgr := NewActiveSessionManager(5, time.Hour)
	defer mgr.Close()

	_, err := mgr.GetLastEventIndex("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent session")
	}
}

func TestActiveSessionManager_ListByProject(t *testing.T) {
	mgr := NewActiveSessionManager(10, time.Hour)
	defer mgr.Close()

	// Register sessions for two projects
	_ = mgr.Register(NewActiveSession("sess-1", "proj-1", "ws-1", "container-1", nil))
	_ = mgr.Register(NewActiveSession("sess-2", "proj-1", "ws-2", "container-1", nil))
	_ = mgr.Register(NewActiveSession("sess-3", "proj-2", "ws-3", "container-1", nil))

	proj1Sessions := mgr.ListByProject("proj-1")
	if len(proj1Sessions) != 2 {
		t.Errorf("ListByProject(proj-1) count = %d, want 2", len(proj1Sessions))
	}

	proj2Sessions := mgr.ListByProject("proj-2")
	if len(proj2Sessions) != 1 {
		t.Errorf("ListByProject(proj-2) count = %d, want 1", len(proj2Sessions))
	}

	noSessions := mgr.ListByProject("proj-3")
	if len(noSessions) != 0 {
		t.Errorf("ListByProject(proj-3) count = %d, want 0", len(noSessions))
	}
}

func TestActiveSessionManager_Count(t *testing.T) {
	mgr := NewActiveSessionManager(10, time.Hour)
	defer mgr.Close()

	if mgr.Count() != 0 {
		t.Errorf("initial Count() = %d, want 0", mgr.Count())
	}

	_ = mgr.Register(NewActiveSession("sess-1", "proj-1", "ws-1", "container-1", nil))
	_ = mgr.Register(NewActiveSession("sess-2", "proj-1", "ws-2", "container-1", nil))

	if mgr.Count() != 2 {
		t.Errorf("Count() = %d, want 2", mgr.Count())
	}
}

func TestActiveSessionManager_CountByProject(t *testing.T) {
	mgr := NewActiveSessionManager(10, time.Hour)
	defer mgr.Close()

	_ = mgr.Register(NewActiveSession("sess-1", "proj-1", "ws-1", "container-1", nil))
	_ = mgr.Register(NewActiveSession("sess-2", "proj-1", "ws-2", "container-1", nil))
	_ = mgr.Register(NewActiveSession("sess-3", "proj-2", "ws-3", "container-1", nil))

	if mgr.CountByProject("proj-1") != 2 {
		t.Errorf("CountByProject(proj-1) = %d, want 2", mgr.CountByProject("proj-1"))
	}
	if mgr.CountByProject("proj-2") != 1 {
		t.Errorf("CountByProject(proj-2) = %d, want 1", mgr.CountByProject("proj-2"))
	}
	if mgr.CountByProject("proj-3") != 0 {
		t.Errorf("CountByProject(proj-3) = %d, want 0", mgr.CountByProject("proj-3"))
	}
}

func TestActiveSessionManager_Close(t *testing.T) {
	mgr := NewActiveSessionManager(5, time.Hour)

	_ = mgr.Register(NewActiveSession("sess-1", "proj-1", "ws-1", "container-1", nil))
	_ = mgr.Register(NewActiveSession("sess-2", "proj-1", "ws-2", "container-1", nil))

	mgr.Close()

	if mgr.Count() != 0 {
		t.Errorf("Count() after Close = %d, want 0", mgr.Count())
	}
}

func TestActiveSessionManager_ConcurrentAccess(t *testing.T) {
	mgr := NewActiveSessionManager(100, time.Hour)
	defer mgr.Close()

	var wg sync.WaitGroup

	// Register sessions concurrently
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sess := NewActiveSession(
				"sess-"+string(rune('a'+i)),
				"proj-1",
				"ws-"+string(rune('a'+i)),
				"container-1",
				nil,
			)
			_ = mgr.Register(sess)
		}(i)
	}
	wg.Wait()

	// Concurrent reads
	for i := 0; i < 50; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			mgr.Count()
		}()
		go func() {
			defer wg.Done()
			mgr.ListByProject("proj-1")
		}()
		go func(i int) {
			defer wg.Done()
			mgr.Get("sess-" + string(rune('a'+i%20)))
		}(i)
	}
	wg.Wait()
}
