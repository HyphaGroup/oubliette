// Package cleanup provides background resource cleanup for Oubliette.
package cleanup

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/HyphaGroup/oubliette/internal/logger"
)

// Cleaner performs periodic resource cleanup.
type Cleaner struct {
	projectsDir string
	interval    time.Duration
	retention   time.Duration
	diskWarn    float64
	diskError   float64
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// Config holds cleanup configuration.
type Config struct {
	ProjectsDir      string
	Interval         time.Duration // How often to run cleanup
	SessionRetention time.Duration // How long to keep completed sessions
	DiskWarnPercent  float64       // Warn at this disk usage percentage
	DiskErrorPercent float64       // Error at this disk usage percentage
}

// DefaultConfig returns sensible defaults.
func DefaultConfig(projectsDir string) Config {
	return Config{
		ProjectsDir:      projectsDir,
		Interval:         5 * time.Minute,
		SessionRetention: 1 * time.Hour,
		DiskWarnPercent:  80.0,
		DiskErrorPercent: 90.0,
	}
}

// New creates a new Cleaner with the given configuration.
func New(cfg Config) *Cleaner {
	return &Cleaner{
		projectsDir: cfg.ProjectsDir,
		interval:    cfg.Interval,
		retention:   cfg.SessionRetention,
		diskWarn:    cfg.DiskWarnPercent,
		diskError:   cfg.DiskErrorPercent,
	}
}

// Start begins the periodic cleanup loop.
func (c *Cleaner) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	c.wg.Add(1)

	go func() {
		defer c.wg.Done()
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()

		// Run immediately on start
		c.runCleanup()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.runCleanup()
			}
		}
	}()

	logger.Printf("🧹 Cleanup started (interval=%v, retention=%v)", c.interval, c.retention)
}

// Stop halts the cleanup loop.
func (c *Cleaner) Stop() {
	if c.cancel != nil {
		c.cancel()
		c.wg.Wait()
		logger.Println("🧹 Cleanup stopped")
	}
}

// runCleanup performs all cleanup tasks.
func (c *Cleaner) runCleanup() {
	c.cleanupTmpFiles()
	c.cleanupOldSessions()
	c.checkDiskUsage()
}

// cleanupTmpFiles removes orphaned .tmp files older than retention.
func (c *Cleaner) cleanupTmpFiles() {
	cutoff := time.Now().Add(-c.retention)
	var removed int

	err := filepath.Walk(c.projectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Look for .tmp files
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".tmp") {
			if info.ModTime().Before(cutoff) {
				if err := os.Remove(path); err == nil {
					removed++
				}
			}
		}
		return nil
	})

	if err != nil {
		logger.Printf("⚠️  Cleanup walk error: %v", err)
	}
	if removed > 0 {
		logger.Printf("🧹 Removed %d orphaned .tmp files", removed)
	}
}

// cleanupOldSessions removes completed session metadata files older than retention.
// Sessions are stored as flat JSON files: sessions/<session_id>.json
func (c *Cleaner) cleanupOldSessions() {
	cutoff := time.Now().Add(-c.retention)
	var removed int

	// Walk project directories looking for sessions
	entries, err := os.ReadDir(c.projectsDir)
	if err != nil {
		return
	}

	for _, projectEntry := range entries {
		if !projectEntry.IsDir() {
			continue
		}

		sessionsDir := filepath.Join(c.projectsDir, projectEntry.Name(), "sessions")
		sessionFiles, err := os.ReadDir(sessionsDir)
		if err != nil {
			continue
		}

		for _, sessionFile := range sessionFiles {
			// Skip directories and non-JSON files
			if sessionFile.IsDir() || !strings.HasSuffix(sessionFile.Name(), ".json") {
				continue
			}

			sessionPath := filepath.Join(sessionsDir, sessionFile.Name())

			// Read session metadata
			data, err := os.ReadFile(sessionPath)
			if err != nil {
				continue
			}

			// Check if session is completed or failed (not active)
			status := string(data)
			isCompleted := strings.Contains(status, `"status":"completed"`) ||
				strings.Contains(status, `"status":"failed"`)

			if !isCompleted {
				continue // Never delete active sessions
			}

			// Check file modification time
			info, err := sessionFile.Info()
			if err != nil {
				continue
			}

			if info.ModTime().Before(cutoff) {
				if err := os.Remove(sessionPath); err == nil {
					removed++
				}
			}
		}
	}

	if removed > 0 {
		logger.Printf("🧹 Cleaned up %d old session files", removed)
	}
}

// checkDiskUsage monitors disk usage and logs warnings.
func (c *Cleaner) checkDiskUsage() {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(c.projectsDir, &stat); err != nil {
		return
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	used := total - free
	usedPercent := float64(used) / float64(total) * 100

	if usedPercent >= c.diskError {
		logger.Printf("🔴 CRITICAL: Disk usage at %.1f%% (projects dir)", usedPercent)
	} else if usedPercent >= c.diskWarn {
		logger.Printf("🟠 WARNING: Disk usage at %.1f%% (projects dir)", usedPercent)
	}
}

// DiskUsage returns current disk usage stats.
func (c *Cleaner) DiskUsage() (usedBytes, totalBytes uint64, usedPercent float64, err error) {
	var stat syscall.Statfs_t
	if err = syscall.Statfs(c.projectsDir, &stat); err != nil {
		return
	}

	totalBytes = stat.Blocks * uint64(stat.Bsize)
	freeBytes := stat.Bfree * uint64(stat.Bsize)
	usedBytes = totalBytes - freeBytes
	usedPercent = float64(usedBytes) / float64(totalBytes) * 100
	return
}
