package schedule

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTestStore(t *testing.T) (*Store, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "schedule_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	store, err := NewStore(dir)
	if err != nil {
		_ = os.RemoveAll(dir)
		t.Fatalf("Failed to create store: %v", err)
	}
	return store, func() {
		_ = store.Close()
		_ = os.RemoveAll(dir)
	}
}

func TestStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	sched := &Schedule{
		Name:            "test-schedule",
		CronExpr:        "0 * * * *",
		Prompt:          "Run daily check",
		Enabled:         true,
		OverlapBehavior: OverlapSkip,
		SessionBehavior: SessionResume,
		CreatorTokenID:  "test-token",
		CreatorScope:    "admin",
		Targets: []ScheduleTarget{
			{ProjectID: "proj-1"},
			{ProjectID: "proj-2", WorkspaceID: "ws-1"},
		},
	}

	err := store.Create(sched)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if sched.ID == "" {
		t.Error("Create() should set ID")
	}
	if sched.CreatedAt.IsZero() {
		t.Error("Create() should set CreatedAt")
	}
	if sched.NextRunAt == nil {
		t.Error("Create() should calculate NextRunAt for enabled schedule")
	}

	// Verify targets have IDs
	for i, target := range sched.Targets {
		if target.ID == "" {
			t.Errorf("Target %d should have ID", i)
		}
	}
}

func TestStore_CreateInvalidCron(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	sched := &Schedule{
		Name:           "invalid-schedule",
		CronExpr:       "invalid cron",
		Prompt:         "test",
		CreatorTokenID: "test",
		CreatorScope:   "admin",
	}

	err := store.Create(sched)
	if err == nil {
		t.Error("Create() with invalid cron should return error")
	}
}

func TestStore_Get(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Create
	sched := &Schedule{
		Name:            "test",
		CronExpr:        "0 0 * * *",
		Prompt:          "test prompt",
		Enabled:         true,
		OverlapBehavior: OverlapParallel,
		SessionBehavior: SessionNew,
		CreatorTokenID:  "tok",
		CreatorScope:    "admin",
		Targets:         []ScheduleTarget{{ProjectID: "p1"}},
	}
	if err := store.Create(sched); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Get
	got, err := store.Get(sched.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.Name != sched.Name {
		t.Errorf("Get().Name = %v, want %v", got.Name, sched.Name)
	}
	if got.OverlapBehavior != OverlapParallel {
		t.Errorf("Get().OverlapBehavior = %v, want %v", got.OverlapBehavior, OverlapParallel)
	}
	if got.SessionBehavior != SessionNew {
		t.Errorf("Get().SessionBehavior = %v, want %v", got.SessionBehavior, SessionNew)
	}
	if len(got.Targets) != 1 {
		t.Errorf("Get().Targets length = %d, want 1", len(got.Targets))
	}
}

func TestStore_GetNotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, err := store.Get("nonexistent")
	if err != ErrScheduleNotFound {
		t.Errorf("Get() error = %v, want ErrScheduleNotFound", err)
	}
}

func TestStore_List(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Create multiple schedules
	for i := 0; i < 3; i++ {
		sched := &Schedule{
			Name:           "test",
			CronExpr:       "* * * * *",
			Prompt:         "p",
			Enabled:        i%2 == 0,
			CreatorTokenID: "t",
			CreatorScope:   "admin",
			Targets:        []ScheduleTarget{{ProjectID: "p1"}},
		}
		if err := store.Create(sched); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// List all
	all, err := store.List(nil)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(all) != 3 {
		t.Errorf("List() returned %d, want 3", len(all))
	}

	// List enabled only
	enabled := true
	filtered, err := store.List(&ListFilter{Enabled: &enabled})
	if err != nil {
		t.Fatalf("List(enabled=true) error = %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("List(enabled=true) returned %d, want 2", len(filtered))
	}
}

func TestStore_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	sched := &Schedule{
		Name:           "original",
		CronExpr:       "0 0 * * *",
		Prompt:         "original prompt",
		Enabled:        true,
		CreatorTokenID: "t",
		CreatorScope:   "admin",
		Targets:        []ScheduleTarget{{ProjectID: "p1"}},
	}
	if err := store.Create(sched); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Update name
	newName := "updated"
	if err := store.Update(sched.ID, &ScheduleUpdate{Name: &newName}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, _ := store.Get(sched.ID)
	if got.Name != "updated" {
		t.Errorf("After Update, Name = %v, want updated", got.Name)
	}

	// Update cron (should recalculate next_run_at)
	newCron := "0 12 * * *"
	if err := store.Update(sched.ID, &ScheduleUpdate{CronExpr: &newCron}); err != nil {
		t.Fatalf("Update cron error = %v", err)
	}

	got, _ = store.Get(sched.ID)
	if got.CronExpr != "0 12 * * *" {
		t.Errorf("After Update, CronExpr = %v, want 0 12 * * *", got.CronExpr)
	}
}

func TestStore_UpdateInvalidCron(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	sched := &Schedule{
		Name:           "test",
		CronExpr:       "0 0 * * *",
		Prompt:         "p",
		CreatorTokenID: "t",
		CreatorScope:   "admin",
		Targets:        []ScheduleTarget{{ProjectID: "p1"}},
	}
	if err := store.Create(sched); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	invalidCron := "invalid"
	err := store.Update(sched.ID, &ScheduleUpdate{CronExpr: &invalidCron})
	if err == nil {
		t.Error("Update() with invalid cron should return error")
	}
}

func TestStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	sched := &Schedule{
		Name:           "to-delete",
		CronExpr:       "0 0 * * *",
		Prompt:         "p",
		CreatorTokenID: "t",
		CreatorScope:   "admin",
		Targets:        []ScheduleTarget{{ProjectID: "p1"}, {ProjectID: "p2"}},
	}
	if err := store.Create(sched); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := store.Delete(sched.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := store.Get(sched.ID)
	if err != ErrScheduleNotFound {
		t.Errorf("Get() after Delete error = %v, want ErrScheduleNotFound", err)
	}
}

func TestStore_DeleteNotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Delete("nonexistent")
	if err != ErrScheduleNotFound {
		t.Errorf("Delete() error = %v, want ErrScheduleNotFound", err)
	}
}

func TestStore_ListDue(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	// Create enabled schedule with past next_run
	sched1 := &Schedule{
		Name:           "due",
		CronExpr:       "* * * * *",
		Prompt:         "p",
		Enabled:        true,
		CreatorTokenID: "t",
		CreatorScope:   "admin",
		Targets:        []ScheduleTarget{{ProjectID: "p1"}},
	}
	if err := store.Create(sched1); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	// Manually set next_run to past
	_, _ = store.db.Exec("UPDATE schedules SET next_run_at = ? WHERE id = ?", past, sched1.ID)

	// Create disabled schedule with past next_run
	sched2 := &Schedule{
		Name:           "disabled",
		CronExpr:       "* * * * *",
		Prompt:         "p",
		Enabled:        false,
		CreatorTokenID: "t",
		CreatorScope:   "admin",
		Targets:        []ScheduleTarget{{ProjectID: "p1"}},
	}
	if err := store.Create(sched2); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	_, _ = store.db.Exec("UPDATE schedules SET next_run_at = ? WHERE id = ?", past, sched2.ID)

	// Create enabled schedule with future next_run
	sched3 := &Schedule{
		Name:           "future",
		CronExpr:       "* * * * *",
		Prompt:         "p",
		Enabled:        true,
		CreatorTokenID: "t",
		CreatorScope:   "admin",
		Targets:        []ScheduleTarget{{ProjectID: "p1"}},
	}
	if err := store.Create(sched3); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	_, _ = store.db.Exec("UPDATE schedules SET next_run_at = ? WHERE id = ?", future, sched3.ID)

	// ListDue should only return enabled + past due
	due, err := store.ListDue(now)
	if err != nil {
		t.Fatalf("ListDue() error = %v", err)
	}

	if len(due) != 1 {
		t.Errorf("ListDue() returned %d, want 1", len(due))
	}
	if len(due) > 0 && due[0].ID != sched1.ID {
		t.Errorf("ListDue() returned wrong schedule")
	}
}

func TestStore_UpdateRunTimes(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	sched := &Schedule{
		Name:           "test",
		CronExpr:       "0 0 * * *",
		Prompt:         "p",
		Enabled:        true,
		CreatorTokenID: "t",
		CreatorScope:   "admin",
		Targets:        []ScheduleTarget{{ProjectID: "p1"}},
	}
	if err := store.Create(sched); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	lastRun := time.Now()
	nextRun := lastRun.Add(24 * time.Hour)

	if err := store.UpdateRunTimes(sched.ID, lastRun, nextRun); err != nil {
		t.Fatalf("UpdateRunTimes() error = %v", err)
	}

	got, _ := store.Get(sched.ID)
	if got.LastRunAt == nil {
		t.Error("LastRunAt should be set")
	}
	if got.NextRunAt == nil {
		t.Error("NextRunAt should be set")
	}
}

func TestStore_DatabaseFile(t *testing.T) {
	dir, err := os.MkdirTemp("", "schedule_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	_ = store.Close()

	// Verify file exists
	dbPath := filepath.Join(dir, "schedules.db")
	if _, err := os.Stat(dbPath); errors.Is(err, fs.ErrNotExist) {
		t.Error("Database file should be created")
	}
}
