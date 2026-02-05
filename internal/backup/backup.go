// Package backup provides backup and restore functionality for Oubliette.
package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/HyphaGroup/oubliette/internal/logger"
)

// Manager handles backup and restore operations.
type Manager struct {
	projectsDir string
	backupDir   string
	retention   int
	interval    time.Duration
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// Config holds backup configuration.
type Config struct {
	ProjectsDir string
	BackupDir   string
	Retention   int           // Number of backups to keep
	Interval    time.Duration // How often to run backups (0 = disabled)
}

// Snapshot represents a backup snapshot.
type Snapshot struct {
	Timestamp time.Time `json:"timestamp"`
	ProjectID string    `json:"project_id"`
	Filename  string    `json:"filename"`
	SizeBytes int64     `json:"size_bytes"`
}

// New creates a new backup Manager.
func New(cfg Config) (*Manager, error) {
	if err := os.MkdirAll(cfg.BackupDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	return &Manager{
		projectsDir: cfg.ProjectsDir,
		backupDir:   cfg.BackupDir,
		retention:   cfg.Retention,
		interval:    cfg.Interval,
	}, nil
}

// Start begins periodic backup if interval > 0.
func (m *Manager) Start() {
	if m.interval <= 0 {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.wg.Add(1)

	go func() {
		defer m.wg.Done()
		ticker := time.NewTicker(m.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := m.BackupAll(); err != nil {
					logger.Printf("⚠️  Backup failed: %v", err)
				}
			}
		}
	}()

	logger.Printf("📦 Backup automation started (interval=%v, retention=%d)", m.interval, m.retention)
}

// Stop halts periodic backup.
func (m *Manager) Stop() {
	if m.cancel != nil {
		m.cancel()
		m.wg.Wait()
		logger.Println("📦 Backup automation stopped")
	}
}

// BackupProject creates a backup of a single project.
func (m *Manager) BackupProject(projectID string) (*Snapshot, error) {
	projectPath := filepath.Join(m.projectsDir, projectID)
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("project not found: %s", projectID)
	}

	timestamp := time.Now()
	filename := fmt.Sprintf("%s_%s.tar.gz", projectID, timestamp.Format("20060102_150405"))
	backupPath := filepath.Join(m.backupDir, filename)

	// Create backup file
	file, err := os.Create(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup file: %w", err)
	}
	defer func() { _ = file.Close() }()

	gw := gzip.NewWriter(file)
	defer func() { _ = gw.Close() }()

	tw := tar.NewWriter(gw)
	defer func() { _ = tw.Close() }()

	// Walk project directory and add files
	err = filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip workspace directories (just backup metadata)
		relPath, _ := filepath.Rel(projectPath, path)
		if strings.HasPrefix(relPath, "workspace") && info.IsDir() && relPath != "workspace" {
			return filepath.SkipDir
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.Join(projectID, relPath)

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer func() { _ = f.Close() }()
			if _, err := io.Copy(tw, f); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		_ = os.Remove(backupPath)
		return nil, fmt.Errorf("failed to create backup: %w", err)
	}

	// Get file size
	stat, _ := os.Stat(backupPath)

	snapshot := &Snapshot{
		Timestamp: timestamp,
		ProjectID: projectID,
		Filename:  filename,
		SizeBytes: stat.Size(),
	}

	logger.Printf("📦 Created backup: %s (%d bytes)", filename, stat.Size())

	// Enforce retention policy
	m.enforceRetention(projectID)

	return snapshot, nil
}

// BackupAll creates backups of all projects.
func (m *Manager) BackupAll() error {
	entries, err := os.ReadDir(m.projectsDir)
	if err != nil {
		return fmt.Errorf("failed to read projects directory: %w", err)
	}

	var errors []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if it's a valid project (has metadata.json)
		metaPath := filepath.Join(m.projectsDir, entry.Name(), "metadata.json")
		if _, err := os.Stat(metaPath); os.IsNotExist(err) {
			continue
		}

		if _, err := m.BackupProject(entry.Name()); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", entry.Name(), err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("backup errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// RestoreProject restores a project from backup.
func (m *Manager) RestoreProject(filename string) error {
	backupPath := filepath.Join(m.backupDir, filename)
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup not found: %s", filename)
	}

	file, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup: %w", err)
	}
	defer func() { _ = file.Close() }()

	gr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to decompress backup: %w", err)
	}
	defer func() { _ = gr.Close() }()

	tr := tar.NewReader(gr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read backup: %w", err)
		}

		targetPath := filepath.Join(m.projectsDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}
			f, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				_ = f.Close()
				return fmt.Errorf("failed to write file: %w", err)
			}
			_ = f.Close()
		}
	}

	logger.Printf("📦 Restored from backup: %s", filename)
	return nil
}

// ListSnapshots returns all available snapshots, optionally filtered by project.
func (m *Manager) ListSnapshots(projectID string) ([]Snapshot, error) {
	entries, err := os.ReadDir(m.backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var snapshots []Snapshot
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tar.gz") {
			continue
		}

		// Parse filename: projectID_YYYYMMDD_HHMMSS.tar.gz
		name := strings.TrimSuffix(entry.Name(), ".tar.gz")
		parts := strings.Split(name, "_")
		if len(parts) < 3 {
			continue
		}

		pid := strings.Join(parts[:len(parts)-2], "_")
		if projectID != "" && pid != projectID {
			continue
		}

		dateStr := parts[len(parts)-2] + "_" + parts[len(parts)-1]
		timestamp, err := time.Parse("20060102_150405", dateStr)
		if err != nil {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		snapshots = append(snapshots, Snapshot{
			Timestamp: timestamp,
			ProjectID: pid,
			Filename:  entry.Name(),
			SizeBytes: info.Size(),
		})
	}

	// Sort by timestamp descending
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Timestamp.After(snapshots[j].Timestamp)
	})

	return snapshots, nil
}

// enforceRetention removes old backups beyond retention limit.
func (m *Manager) enforceRetention(projectID string) {
	snapshots, err := m.ListSnapshots(projectID)
	if err != nil {
		return
	}

	if len(snapshots) <= m.retention {
		return
	}

	// Remove oldest backups
	for i := m.retention; i < len(snapshots); i++ {
		backupPath := filepath.Join(m.backupDir, snapshots[i].Filename)
		if err := os.Remove(backupPath); err == nil {
			logger.Printf("📦 Removed old backup: %s", snapshots[i].Filename)
		}
	}
}

// ExportManifest creates a JSON manifest of all snapshots.
func (m *Manager) ExportManifest() ([]byte, error) {
	snapshots, err := m.ListSnapshots("")
	if err != nil {
		return nil, err
	}

	manifest := struct {
		ExportedAt time.Time  `json:"exported_at"`
		BackupDir  string     `json:"backup_dir"`
		Snapshots  []Snapshot `json:"snapshots"`
	}{
		ExportedAt: time.Now(),
		BackupDir:  m.backupDir,
		Snapshots:  snapshots,
	}

	return json.MarshalIndent(manifest, "", "  ")
}
