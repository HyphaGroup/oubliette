package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/HyphaGroup/oubliette/internal/agent"
	"github.com/HyphaGroup/oubliette/internal/validation"
)

// Manager handles agent session lifecycle
type Manager struct {
	sessionsBaseDir string
	agentRuntime    agent.Runtime
	oublietteMCPURL string
	// Persistent session index for O(1) lookups
	persistentIndex *SessionIndex
	// Per-session locks for thread-safe metadata operations
	sessionLocks *SessionLockMap
}

// NewManager creates a new session manager
// Note: Call LoadIndex() after creation to restore persistent index
func NewManager(sessionsBaseDir string, agentRuntime agent.Runtime, oublietteMCPURL string) *Manager {
	// Use parent of sessionsBaseDir for data directory (sessionsBaseDir is typically .../projects)
	dataDir := filepath.Join(filepath.Dir(sessionsBaseDir), "data")
	return &Manager{
		sessionsBaseDir: sessionsBaseDir,
		agentRuntime:    agentRuntime,
		oublietteMCPURL: oublietteMCPURL,
		persistentIndex: NewSessionIndex(dataDir),
		sessionLocks:    NewSessionLockMap(),
	}
}

// LoadIndex loads the persistent session index from disk
// Should be called once at startup
func (m *Manager) LoadIndex() error {
	return m.persistentIndex.Load()
}

// SaveIndex persists the session index to disk
// Called automatically on session updates, but can be called manually
func (m *Manager) SaveIndex() error {
	return m.persistentIndex.Save()
}

// indexSession adds a session to the persistent index
func (m *Manager) indexSession(session *Session) {
	m.persistentIndex.Add(&SessionIndexEntry{
		SessionID:   session.SessionID,
		ProjectID:   session.ProjectID,
		WorkspaceID: session.WorkspaceID,
		Status:      session.Status,
	})
	// Best-effort save - don't fail the operation if save fails
	_ = m.persistentIndex.Save()
}

// updateIndexStatus updates the status in the index
func (m *Manager) updateIndexStatus(sessionID string, status Status) {
	m.persistentIndex.UpdateStatus(sessionID, status)
	_ = m.persistentIndex.Save()
}

// lookupProject returns the project ID for a session (O(1) lookup)
func (m *Manager) lookupProject(sessionID string) (string, bool) {
	entry, ok := m.persistentIndex.Get(sessionID)
	if ok {
		return entry.ProjectID, true
	}
	return "", false
}

// AgentRuntime returns the agent runtime for direct execution
func (m *Manager) AgentRuntime() agent.Runtime {
	return m.agentRuntime
}

// Create spawns a new agent session
func (m *Manager) Create(ctx context.Context, projectID, containerID, prompt string, opts StartOptions) (*Session, error) {
	sessionID := generateSessionID()
	sessionsDir := filepath.Join(m.sessionsBaseDir, projectID, "sessions")

	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create sessions directory: %w", err)
	}

	// Workspace ID is required - handlers must resolve it before calling
	workspaceID := opts.WorkspaceID
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace_id is required - caller must resolve workspace before creating session")
	}

	session := &Session{
		SessionID:      sessionID,
		ProjectID:      projectID,
		WorkspaceID:    workspaceID,
		ContainerID:    containerID,
		Status:         StatusActive,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Turns:          []Turn{},
		TotalCost:      Cost{},
		Model:          opts.Model,
		AutonomyLevel:  opts.AutonomyLevel,
		ReasoningLevel: opts.ReasoningLevel,
	}

	// Working directory depends on workspace isolation setting
	var workingDir string
	if opts.WorkspaceIsolation {
		// Isolated mode: /workspace is mounted to workspaces/, so workingDir is /workspace/<uuid>
		workingDir = filepath.Join("/workspace", workspaceID)
	} else {
		// Non-isolated mode: /workspace is mounted to project root
		workingDir = filepath.Join("/workspace", "workspaces", workspaceID)
	}

	// Execute first turn via agent runtime (single-turn, non-interactive)
	// For interactive streaming sessions, use CreateBidirectionalSession instead
	req := &agent.ExecuteRequest{
		Prompt:        prompt,
		ContainerID:   containerID,
		WorkingDir:    workingDir,
		SessionID:     sessionID,
		ProjectID:     projectID,
		Depth:         0, // Prime session is at depth 0
		Model:         opts.Model,
		AutonomyLevel: opts.AutonomyLevel,
		EnabledTools:  opts.ToolsAllowed,
		DisabledTools: opts.ToolsDisallowed,
		SystemPrompt:  opts.AppendSystemPrompt,
	}

	// Determine which runtime to use (override or manager's default)
	runtime := m.agentRuntime
	if opts.RuntimeOverride != nil {
		if rt, ok := opts.RuntimeOverride.(agent.Runtime); ok {
			runtime = rt
		}
	}

	resp, err := runtime.Execute(ctx, req)
	if err != nil {
		session.Status = StatusFailed
		return nil, err
	}

	// Record turn
	turn := Turn{
		TurnNumber:  1,
		Prompt:      prompt,
		StartedAt:   time.Now().Add(-time.Duration(resp.DurationMs) * time.Millisecond),
		CompletedAt: time.Now(),
		Output: TurnOutput{
			Text:     resp.Result,
			ExitCode: 0,
		},
		Cost: Cost{
			InputTokens:  resp.InputTokens,
			OutputTokens: resp.OutputTokens,
		},
	}

	session.RuntimeSessionID = resp.SessionID
	session.Turns = append(session.Turns, turn)
	session.TotalCost.InputTokens = resp.InputTokens
	session.TotalCost.OutputTokens = resp.OutputTokens

	// Use locked save even for new sessions for consistency
	if err := m.saveSessionLocked(session); err != nil {
		return nil, err
	}

	return session, nil
}

// Continue adds a turn to existing session
func (m *Manager) Continue(ctx context.Context, sessionID, prompt string) (*Turn, error) {
	// Lock session for the entire operation (read-modify-write)
	m.sessionLocks.Lock(sessionID)
	defer m.sessionLocks.Unlock(sessionID)

	session, err := m.Load(sessionID)
	if err != nil {
		return nil, err
	}

	if session.Status != StatusActive {
		return nil, fmt.Errorf("session %s is not active", sessionID)
	}

	req := &agent.ExecuteRequest{
		Prompt:        prompt,
		ContainerID:   session.ContainerID,
		WorkingDir:    "/workspace",
		SessionID:     session.RuntimeSessionID, // Continue agent session
		Model:         session.Model,
		AutonomyLevel: session.AutonomyLevel,
	}

	resp, err := m.agentRuntime.Execute(ctx, req)
	if err != nil {
		session.Status = StatusFailed
		return nil, err
	}

	turn := Turn{
		TurnNumber:  len(session.Turns) + 1,
		Prompt:      prompt,
		StartedAt:   time.Now().Add(-time.Duration(resp.DurationMs) * time.Millisecond),
		CompletedAt: time.Now(),
		Output: TurnOutput{
			Text:     resp.Result,
			ExitCode: 0,
		},
		Cost: Cost{
			InputTokens:  resp.InputTokens,
			OutputTokens: resp.OutputTokens,
		},
	}

	session.Turns = append(session.Turns, turn)
	session.UpdatedAt = time.Now()
	session.TotalCost.InputTokens += resp.InputTokens
	session.TotalCost.OutputTokens += resp.OutputTokens

	// Use internal saveSession since we already hold the lock
	if err := m.saveSession(session); err != nil {
		return nil, err
	}

	return &turn, nil
}

// Load retrieves a session by ID
// Uses in-memory index for O(1) lookup when available, falls back to directory scan
func (m *Manager) Load(sessionID string) (*Session, error) {
	if err := validation.ValidateSessionID(sessionID); err != nil {
		return nil, err
	}

	// Try indexed lookup first (O(1))
	if projectID, ok := m.lookupProject(sessionID); ok {
		sessionPath := filepath.Join(m.sessionsBaseDir, projectID, "sessions", sessionID+".json")
		if session, err := m.loadSessionFromFile(sessionPath); err == nil {
			return session, nil
		}
		// Index was stale, remove and fall through to scan
		m.persistentIndex.Remove(sessionID)
	}

	// Fall back to directory scan (O(n) - populates index for future lookups)
	entries, err := os.ReadDir(m.sessionsBaseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read sessions base directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		sessionPath := filepath.Join(m.sessionsBaseDir, entry.Name(), "sessions", sessionID+".json")
		if _, err := os.Stat(sessionPath); err == nil {
			session, err := m.loadSessionFromFile(sessionPath)
			if err == nil {
				// Add to persistent index for future O(1) lookups
				m.indexSession(session)
			}
			return session, err
		}
	}

	return nil, fmt.Errorf("session %s not found", sessionID)
}

// List returns all sessions for a project
func (m *Manager) List(projectID string, statusFilter *Status) ([]*SessionSummary, error) {
	if err := validation.ValidateProjectID(projectID); err != nil {
		return nil, err
	}

	sessionsDir := filepath.Join(m.sessionsBaseDir, projectID, "sessions")

	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []*SessionSummary{}, nil
		}
		return nil, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	var summaries []*SessionSummary
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		sessionPath := filepath.Join(sessionsDir, entry.Name())
		session, err := m.loadSessionFromFile(sessionPath)
		if err != nil {
			continue
		}

		if statusFilter != nil && session.Status != *statusFilter {
			continue
		}

		summaries = append(summaries, session.ToSummary())
	}

	return summaries, nil
}

// End marks a session as completed
func (m *Manager) End(sessionID string) error {
	if err := validation.ValidateSessionID(sessionID); err != nil {
		return err
	}

	// Lock session for the entire operation (read-modify-write)
	m.sessionLocks.Lock(sessionID)
	defer m.sessionLocks.Unlock(sessionID)

	session, err := m.Load(sessionID)
	if err != nil {
		return err
	}

	session.Status = StatusCompleted
	session.UpdatedAt = time.Now()

	// Use internal saveSession since we already hold the lock
	return m.saveSession(session)
}

// saveSession writes session to disk atomically (internal, assumes caller holds lock)
func (m *Manager) saveSession(session *Session) error {
	sessionsDir := filepath.Join(m.sessionsBaseDir, session.ProjectID, "sessions")
	sessionPath := filepath.Join(sessionsDir, session.SessionID+".json")

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	tmpPath := sessionPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	// Remove extended attributes that might prevent rename on macOS
	// This handles com.apple.provenance and similar attributes
	_ = exec.Command("xattr", "-c", tmpPath).Run() // Ignore errors, best effort

	if err := os.Rename(tmpPath, sessionPath); err != nil {
		return fmt.Errorf("failed to rename session file: %w", err)
	}

	// Add to persistent index for O(1) lookups
	m.indexSession(session)

	return nil
}

// saveSessionLocked writes session to disk atomically with locking
func (m *Manager) saveSessionLocked(session *Session) error {
	m.sessionLocks.Lock(session.SessionID)
	defer m.sessionLocks.Unlock(session.SessionID)
	return m.saveSession(session)
}

// loadSessionFromFile reads a session from disk
func (m *Manager) loadSessionFromFile(path string) (*Session, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to parse session file: %w", err)
	}

	return &session, nil
}

// SaveSession exposes saveSession for external use with locking
func (m *Manager) SaveSession(session *Session) error {
	return m.saveSessionLocked(session)
}

// AddChildSession adds a child session ID to a parent session
// Validates that both sessions exist and child's depth is exactly parent's depth + 1
func (m *Manager) AddChildSession(parentSessionID, childSessionID string) error {
	// Lock parent session for the entire operation (read-modify-write)
	m.sessionLocks.Lock(parentSessionID)
	defer m.sessionLocks.Unlock(parentSessionID)

	parent, err := m.Load(parentSessionID)
	if err != nil {
		return fmt.Errorf("failed to load parent session: %w", err)
	}

	child, err := m.Load(childSessionID)
	if err != nil {
		return fmt.Errorf("failed to load child session: %w", err)
	}

	// Validate depth consistency
	expectedChildDepth := parent.Depth + 1
	if child.Depth != expectedChildDepth {
		return fmt.Errorf("depth mismatch: child depth %d != expected %d (parent depth + 1)", child.Depth, expectedChildDepth)
	}

	parent.ChildSessions = append(parent.ChildSessions, childSessionID)
	parent.UpdatedAt = time.Now()

	// Use internal saveSession since we already hold the lock
	return m.saveSession(parent)
}

// generateSessionID creates a unique session identifier
func generateSessionID() string {
	timestamp := time.Now().Format("20060102_150405")
	randomBytes := make([]byte, 4)
	_, _ = rand.Read(randomBytes)
	randomHex := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("gogol_%s_%s", timestamp, randomHex)
}

// GenerateExplorationID creates a unique exploration identifier
func GenerateExplorationID() string {
	timestamp := time.Now().Format("20060102")
	randomBytes := make([]byte, 4)
	_, _ = rand.Read(randomBytes)
	randomHex := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("exp_%s_%s", timestamp, randomHex)
}

// RecoverStaleSessions scans for sessions that were marked active but are now stale
// (e.g., due to server crash). Sessions older than maxAge are marked as failed.
//
// RECOVERY ALGORITHM:
//
// This function should be called on server startup to clean up sessions that were
// left in "active" state due to an ungraceful shutdown (crash, kill -9, etc.).
//
// Algorithm:
//  1. Scan all project directories under sessionsBaseDir
//  2. For each project, scan all session metadata files (*.json)
//  3. For each session with status="active":
//     - If UpdatedAt < (now - maxAge), mark as "failed"
//     - UpdatedAt serves as a heartbeat - active sessions update it periodically
//  4. Return count of recovered (transitioned to failed) sessions
//
// Why maxAge matters:
//   - Too short: May mark legitimately running sessions as failed
//   - Too long: Stale sessions sit around longer before cleanup
//   - Default: 30 minutes (reasonable for long-running agent tasks)
//
// Performance considerations:
//   - Uses persistent session index for O(1) lookup of active sessions
//   - Only loads metadata for sessions marked active in index
//   - Falls back to filesystem scan if index is empty (first run or corruption)
//
// Thread safety:
//   - Safe to call from single goroutine (server startup)
//   - Acquires per-session locks for modifications
func (m *Manager) RecoverStaleSessions(maxAge time.Duration) (recovered int, err error) {
	now := time.Now()
	cutoff := now.Add(-maxAge)

	// Try index-based recovery first (fast path)
	activeSessions := m.persistentIndex.GetByStatus(StatusActive)
	if len(activeSessions) > 0 {
		for _, sessionID := range activeSessions {
			entry, ok := m.persistentIndex.Get(sessionID)
			if !ok {
				continue
			}

			// Load session metadata to check UpdatedAt
			sessionPath := filepath.Join(m.sessionsBaseDir, entry.ProjectID, "sessions", sessionID+".json")
			session, err := m.loadSessionFromFile(sessionPath)
			if err != nil {
				// Session file missing, remove from index
				m.persistentIndex.Remove(sessionID)
				continue
			}

			// Check if session is stale
			if session.UpdatedAt.Before(cutoff) {
				m.sessionLocks.Lock(sessionID)
				session.Status = StatusFailed
				session.UpdatedAt = now
				err := m.saveSession(session)
				m.sessionLocks.Unlock(sessionID)
				if err != nil {
					continue
				}
				recovered++
			}
		}
		return recovered, nil
	}

	// Fall back to filesystem scan if index is empty (first run or corruption)
	entries, err := os.ReadDir(m.sessionsBaseDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to read sessions base directory: %w", err)
	}

	for _, projectEntry := range entries {
		if !projectEntry.IsDir() {
			continue
		}

		sessionsDir := filepath.Join(m.sessionsBaseDir, projectEntry.Name(), "sessions")
		sessionFiles, err := os.ReadDir(sessionsDir)
		if err != nil {
			continue
		}

		for _, sessionFile := range sessionFiles {
			if !strings.HasSuffix(sessionFile.Name(), ".json") {
				continue
			}

			sessionPath := filepath.Join(sessionsDir, sessionFile.Name())
			session, err := m.loadSessionFromFile(sessionPath)
			if err != nil {
				continue
			}

			// Index all sessions for future lookups
			m.indexSession(session)

			// Check if session was active but is now stale
			if session.Status == StatusActive && session.UpdatedAt.Before(cutoff) {
				m.sessionLocks.Lock(session.SessionID)
				session.Status = StatusFailed
				session.UpdatedAt = now
				err := m.saveSession(session)
				m.sessionLocks.Unlock(session.SessionID)
				if err != nil {
					continue
				}
				recovered++
			}
		}
	}

	return recovered, nil
}

// CleanupOldSessions deletes session metadata files older than maxAge for a specific project.
// Only deletes sessions that are not in "active" status.
// Returns the count of deleted sessions.
func (m *Manager) CleanupOldSessions(projectID string, maxAge time.Duration) (int, error) {
	if err := validation.ValidateProjectID(projectID); err != nil {
		return 0, err
	}

	sessionsDir := filepath.Join(m.sessionsBaseDir, projectID, "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	cutoff := time.Now().Add(-maxAge)
	deleted := 0

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		sessionPath := filepath.Join(sessionsDir, entry.Name())
		session, err := m.loadSessionFromFile(sessionPath)
		if err != nil {
			// Can't parse, skip
			continue
		}

		// Never delete active sessions
		if session.Status == StatusActive {
			continue
		}

		// Check if session is old enough to delete
		if session.UpdatedAt.Before(cutoff) {
			// Remove from index first
			m.persistentIndex.Remove(session.SessionID)

			// Delete the file
			if err := os.Remove(sessionPath); err != nil {
				continue
			}
			deleted++
		}
	}

	// Save index after bulk removal
	_ = m.persistentIndex.Save()

	return deleted, nil
}

// CleanupAllOldSessions deletes old session metadata files across all projects.
// Returns a map of project_id -> deleted count.
func (m *Manager) CleanupAllOldSessions(maxAge time.Duration) (map[string]int, error) {
	results := make(map[string]int)

	entries, err := os.ReadDir(m.sessionsBaseDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return results, nil
		}
		return nil, fmt.Errorf("failed to read projects directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectID := entry.Name()
		deleted, err := m.CleanupOldSessions(projectID, maxAge)
		if err != nil {
			// Log but continue with other projects
			continue
		}

		if deleted > 0 {
			results[projectID] = deleted
		}
	}

	return results, nil
}
