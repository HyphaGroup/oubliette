package cleanup

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig("/test/projects")

	if cfg.ProjectsDir != "/test/projects" {
		t.Errorf("ProjectsDir = %q, want %q", cfg.ProjectsDir, "/test/projects")
	}
	if cfg.Interval != 5*time.Minute {
		t.Errorf("Interval = %v, want %v", cfg.Interval, 5*time.Minute)
	}
	if cfg.SessionRetention != 1*time.Hour {
		t.Errorf("SessionRetention = %v, want %v", cfg.SessionRetention, 1*time.Hour)
	}
	if cfg.DiskWarnPercent != 80.0 {
		t.Errorf("DiskWarnPercent = %f, want 80.0", cfg.DiskWarnPercent)
	}
	if cfg.DiskErrorPercent != 90.0 {
		t.Errorf("DiskErrorPercent = %f, want 90.0", cfg.DiskErrorPercent)
	}
}

func TestNew(t *testing.T) {
	cfg := Config{
		ProjectsDir:      "/custom/projects",
		Interval:         10 * time.Minute,
		SessionRetention: 2 * time.Hour,
		DiskWarnPercent:  75.0,
		DiskErrorPercent: 85.0,
	}

	cleaner := New(cfg)

	if cleaner.projectsDir != "/custom/projects" {
		t.Errorf("projectsDir = %q, want %q", cleaner.projectsDir, "/custom/projects")
	}
	if cleaner.interval != 10*time.Minute {
		t.Errorf("interval = %v, want %v", cleaner.interval, 10*time.Minute)
	}
	if cleaner.retention != 2*time.Hour {
		t.Errorf("retention = %v, want %v", cleaner.retention, 2*time.Hour)
	}
	if cleaner.diskWarn != 75.0 {
		t.Errorf("diskWarn = %f, want 75.0", cleaner.diskWarn)
	}
	if cleaner.diskError != 85.0 {
		t.Errorf("diskError = %f, want 85.0", cleaner.diskError)
	}
}

func TestCleaner_StartStop(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		ProjectsDir:      tmpDir,
		Interval:         100 * time.Millisecond, // Fast for testing
		SessionRetention: 1 * time.Hour,
		DiskWarnPercent:  80.0,
		DiskErrorPercent: 90.0,
	}

	cleaner := New(cfg)
	cleaner.Start()

	// Give it time to run at least once
	time.Sleep(150 * time.Millisecond)

	cleaner.Stop()

	// Verify it stopped (no panic, no hanging)
}

func TestCleaner_CleanupTmpFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some .tmp files with different ages
	oldTmpFile := filepath.Join(tmpDir, "old.tmp")
	newTmpFile := filepath.Join(tmpDir, "new.tmp")
	regularFile := filepath.Join(tmpDir, "regular.txt")

	_ = os.WriteFile(oldTmpFile, []byte("old"), 0o644)
	_ = os.WriteFile(newTmpFile, []byte("new"), 0o644)
	_ = os.WriteFile(regularFile, []byte("keep"), 0o644)

	// Make old file appear old
	oldTime := time.Now().Add(-2 * time.Hour)
	_ = os.Chtimes(oldTmpFile, oldTime, oldTime)

	cfg := Config{
		ProjectsDir:      tmpDir,
		Interval:         1 * time.Hour, // Won't run during test
		SessionRetention: 1 * time.Hour,
		DiskWarnPercent:  80.0,
		DiskErrorPercent: 90.0,
	}

	cleaner := New(cfg)
	cleaner.cleanupTmpFiles()

	// Old .tmp should be removed
	if _, err := os.Stat(oldTmpFile); !errors.Is(err, fs.ErrNotExist) {
		t.Error("old .tmp file should have been removed")
	}

	// New .tmp should still exist
	if _, err := os.Stat(newTmpFile); err != nil {
		t.Error("new .tmp file should still exist")
	}

	// Regular file should still exist
	if _, err := os.Stat(regularFile); err != nil {
		t.Error("regular file should still exist")
	}
}

func TestCleaner_CleanupTmpFiles_Nested(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested directory structure
	nestedDir := filepath.Join(tmpDir, "project1", "sessions")
	_ = os.MkdirAll(nestedDir, 0o755)

	nestedTmpFile := filepath.Join(nestedDir, "nested.tmp")
	_ = os.WriteFile(nestedTmpFile, []byte("nested"), 0o644)

	// Make it old
	oldTime := time.Now().Add(-2 * time.Hour)
	_ = os.Chtimes(nestedTmpFile, oldTime, oldTime)

	cfg := Config{
		ProjectsDir:      tmpDir,
		SessionRetention: 1 * time.Hour,
	}

	cleaner := New(cfg)
	cleaner.cleanupTmpFiles()

	// Nested old .tmp should be removed
	if _, err := os.Stat(nestedTmpFile); !errors.Is(err, fs.ErrNotExist) {
		t.Error("nested old .tmp file should have been removed")
	}
}

func TestCleaner_DiskUsage(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		ProjectsDir: tmpDir,
	}

	cleaner := New(cfg)
	used, total, percent, err := cleaner.DiskUsage()

	if err != nil {
		t.Fatalf("DiskUsage() error = %v", err)
	}

	if total == 0 {
		t.Error("total bytes should be > 0")
	}
	if used > total {
		t.Error("used bytes should be <= total bytes")
	}
	if percent < 0 || percent > 100 {
		t.Errorf("percent = %f, should be between 0 and 100", percent)
	}
}

func TestCleaner_DiskUsage_InvalidPath(t *testing.T) {
	cfg := Config{
		ProjectsDir: "/nonexistent/path/that/does/not/exist",
	}

	cleaner := New(cfg)
	_, _, _, err := cleaner.DiskUsage()

	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestCleaner_CheckDiskUsage(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		ProjectsDir:      tmpDir,
		DiskWarnPercent:  80.0,
		DiskErrorPercent: 90.0,
	}

	cleaner := New(cfg)

	// This should not panic - just logs warnings if disk is high
	cleaner.checkDiskUsage()
}

func TestCleaner_RunCleanup(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		ProjectsDir:      tmpDir,
		SessionRetention: 1 * time.Hour,
		DiskWarnPercent:  80.0,
		DiskErrorPercent: 90.0,
	}

	cleaner := New(cfg)

	// Should run all cleanup tasks without panic
	cleaner.runCleanup()
}

func TestCleaner_CleanupOldSessions(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a project with sessions (flat JSON files, not directories)
	projectDir := filepath.Join(tmpDir, "project-1")
	sessionsDir := filepath.Join(projectDir, "sessions")
	_ = os.MkdirAll(sessionsDir, 0o755)

	oldTime := time.Now().Add(-2 * time.Hour)

	// Create old completed session file
	oldSessionFile := filepath.Join(sessionsDir, "old-session.json")
	_ = os.WriteFile(oldSessionFile, []byte(`{"status":"completed"}`), 0o644)
	_ = os.Chtimes(oldSessionFile, oldTime, oldTime)

	// Create new completed session file
	newSessionFile := filepath.Join(sessionsDir, "new-session.json")
	_ = os.WriteFile(newSessionFile, []byte(`{"status":"completed"}`), 0o644)

	// Create old active session file (should not be removed)
	activeSessionFile := filepath.Join(sessionsDir, "active-session.json")
	_ = os.WriteFile(activeSessionFile, []byte(`{"status":"active"}`), 0o644)
	_ = os.Chtimes(activeSessionFile, oldTime, oldTime)

	cfg := Config{
		ProjectsDir:      tmpDir,
		SessionRetention: 1 * time.Hour,
	}

	cleaner := New(cfg)
	cleaner.cleanupOldSessions()

	// Old completed session file should be removed
	if _, err := os.Stat(oldSessionFile); !errors.Is(err, fs.ErrNotExist) {
		t.Error("old completed session should have been removed")
	}

	// New completed session file should still exist
	if _, err := os.Stat(newSessionFile); err != nil {
		t.Error("new completed session should still exist")
	}

	// Active session file should still exist
	if _, err := os.Stat(activeSessionFile); err != nil {
		t.Error("active session should still exist even if old")
	}
}

func TestCleaner_CleanupOldSessions_FailedStatus(t *testing.T) {
	tmpDir := t.TempDir()

	projectDir := filepath.Join(tmpDir, "project-1")
	sessionsDir := filepath.Join(projectDir, "sessions")
	_ = os.MkdirAll(sessionsDir, 0o755)

	oldTime := time.Now().Add(-2 * time.Hour)

	// Create old failed session file (should be cleaned up)
	failedSessionFile := filepath.Join(sessionsDir, "failed-session.json")
	_ = os.WriteFile(failedSessionFile, []byte(`{"status":"failed"}`), 0o644)
	_ = os.Chtimes(failedSessionFile, oldTime, oldTime)

	cfg := Config{
		ProjectsDir:      tmpDir,
		SessionRetention: 1 * time.Hour,
	}

	cleaner := New(cfg)
	cleaner.cleanupOldSessions()

	// Old failed session file should be removed
	if _, err := os.Stat(failedSessionFile); !errors.Is(err, fs.ErrNotExist) {
		t.Error("old failed session should have been removed")
	}
}
