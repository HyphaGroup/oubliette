package session

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// SessionIndex provides persistent O(1) session lookups.
// The index maps sessionID -> SessionIndexEntry for fast retrieval
// of session location (project, workspace) and status.
type SessionIndex struct {
	// entries maps sessionID -> entry
	entries map[string]*SessionIndexEntry
	// byProject maps projectID -> set of sessionIDs
	byProject map[string]map[string]bool
	// byStatus maps status -> set of sessionIDs for fast filtering
	byStatus map[Status]map[string]bool
	mu       sync.RWMutex
	filePath string
}

// SessionIndexEntry contains the indexed data for a session
type SessionIndexEntry struct {
	SessionID   string `json:"session_id"`
	ProjectID   string `json:"project_id"`
	WorkspaceID string `json:"workspace_id"`
	Status      Status `json:"status"`
}

// NewSessionIndex creates a new session index
func NewSessionIndex(dataDir string) *SessionIndex {
	return &SessionIndex{
		entries:   make(map[string]*SessionIndexEntry),
		byProject: make(map[string]map[string]bool),
		byStatus:  make(map[Status]map[string]bool),
		filePath:  filepath.Join(dataDir, "sessions_index.json"),
	}
}

// Load reads the index from disk
func (idx *SessionIndex) Load() error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	data, err := os.ReadFile(idx.filePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// No index file yet, start fresh
			return nil
		}
		return err
	}

	var entries []*SessionIndexEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}

	// Rebuild in-memory indices
	idx.entries = make(map[string]*SessionIndexEntry, len(entries))
	idx.byProject = make(map[string]map[string]bool)
	idx.byStatus = make(map[Status]map[string]bool)

	for _, entry := range entries {
		idx.entries[entry.SessionID] = entry
		idx.addToProjectIndex(entry.SessionID, entry.ProjectID)
		idx.addToStatusIndex(entry.SessionID, entry.Status)
	}

	return nil
}

// Save writes the index to disk atomically
func (idx *SessionIndex) Save() error {
	idx.mu.RLock()
	entries := make([]*SessionIndexEntry, 0, len(idx.entries))
	for _, entry := range idx.entries {
		entries = append(entries, entry)
	}
	idx.mu.RUnlock()

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(idx.filePath), 0o755); err != nil {
		return err
	}

	// Atomic write
	tmpPath := idx.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, idx.filePath)
}

// Add adds or updates a session in the index
func (idx *SessionIndex) Add(entry *SessionIndexEntry) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Remove from old status index if updating
	if old, exists := idx.entries[entry.SessionID]; exists {
		idx.removeFromStatusIndex(entry.SessionID, old.Status)
	}

	idx.entries[entry.SessionID] = entry
	idx.addToProjectIndex(entry.SessionID, entry.ProjectID)
	idx.addToStatusIndex(entry.SessionID, entry.Status)
}

// Get retrieves a session entry by ID (O(1))
func (idx *SessionIndex) Get(sessionID string) (*SessionIndexEntry, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	entry, ok := idx.entries[sessionID]
	return entry, ok
}

// GetByProject returns all session IDs for a project
func (idx *SessionIndex) GetByProject(projectID string) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	sessions := idx.byProject[projectID]
	result := make([]string, 0, len(sessions))
	for sessionID := range sessions {
		result = append(result, sessionID)
	}
	return result
}

// GetByStatus returns all session IDs with a given status
func (idx *SessionIndex) GetByStatus(status Status) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	sessions := idx.byStatus[status]
	result := make([]string, 0, len(sessions))
	for sessionID := range sessions {
		result = append(result, sessionID)
	}
	return result
}

// UpdateStatus updates the status of a session in the index
func (idx *SessionIndex) UpdateStatus(sessionID string, newStatus Status) bool {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	entry, exists := idx.entries[sessionID]
	if !exists {
		return false
	}

	idx.removeFromStatusIndex(sessionID, entry.Status)
	entry.Status = newStatus
	idx.addToStatusIndex(sessionID, newStatus)
	return true
}

// Remove removes a session from the index
func (idx *SessionIndex) Remove(sessionID string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	entry, exists := idx.entries[sessionID]
	if !exists {
		return
	}

	idx.removeFromProjectIndex(sessionID, entry.ProjectID)
	idx.removeFromStatusIndex(sessionID, entry.Status)
	delete(idx.entries, sessionID)
}

// Count returns the total number of indexed sessions
func (idx *SessionIndex) Count() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.entries)
}

// internal helper methods (must be called with lock held)

func (idx *SessionIndex) addToProjectIndex(sessionID, projectID string) {
	if idx.byProject[projectID] == nil {
		idx.byProject[projectID] = make(map[string]bool)
	}
	idx.byProject[projectID][sessionID] = true
}

func (idx *SessionIndex) removeFromProjectIndex(sessionID, projectID string) {
	if idx.byProject[projectID] != nil {
		delete(idx.byProject[projectID], sessionID)
	}
}

func (idx *SessionIndex) addToStatusIndex(sessionID string, status Status) {
	if idx.byStatus[status] == nil {
		idx.byStatus[status] = make(map[string]bool)
	}
	idx.byStatus[status][sessionID] = true
}

func (idx *SessionIndex) removeFromStatusIndex(sessionID string, status Status) {
	if idx.byStatus[status] != nil {
		delete(idx.byStatus[status], sessionID)
	}
}
