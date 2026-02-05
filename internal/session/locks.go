package session

import (
	"sync"
)

// SessionLockMap provides per-session RWMutex for thread-safe metadata operations.
// This prevents race conditions when multiple concurrent operations (spawn child,
// send message, mark complete) access the same session.
type SessionLockMap struct {
	locks sync.Map // sessionID -> *sync.RWMutex
}

// NewSessionLockMap creates a new session lock map
func NewSessionLockMap() *SessionLockMap {
	return &SessionLockMap{}
}

// getOrCreateLock returns the lock for a session, creating one if needed
func (m *SessionLockMap) getOrCreateLock(sessionID string) *sync.RWMutex {
	lock, _ := m.locks.LoadOrStore(sessionID, &sync.RWMutex{})
	rwMutex, _ := lock.(*sync.RWMutex)
	return rwMutex
}

// Lock acquires an exclusive write lock for a session
func (m *SessionLockMap) Lock(sessionID string) {
	m.getOrCreateLock(sessionID).Lock()
}

// Unlock releases the write lock for a session
func (m *SessionLockMap) Unlock(sessionID string) {
	m.getOrCreateLock(sessionID).Unlock()
}

// RLock acquires a read lock for a session
func (m *SessionLockMap) RLock(sessionID string) {
	m.getOrCreateLock(sessionID).RLock()
}

// RUnlock releases the read lock for a session
func (m *SessionLockMap) RUnlock(sessionID string) {
	m.getOrCreateLock(sessionID).RUnlock()
}

// Delete removes the lock for a session (call after session cleanup)
func (m *SessionLockMap) Delete(sessionID string) {
	m.locks.Delete(sessionID)
}
