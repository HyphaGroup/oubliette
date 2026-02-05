package project

import (
	"sync"
)

// ProjectLockMap provides per-project RWMutex for thread-safe metadata operations
type ProjectLockMap struct {
	locks sync.Map // projectID -> *sync.RWMutex
}

// getOrCreateLock returns the lock for a project, creating one if needed
func (m *ProjectLockMap) getOrCreateLock(projectID string) *sync.RWMutex {
	lock, _ := m.locks.LoadOrStore(projectID, &sync.RWMutex{})
	rwMutex, _ := lock.(*sync.RWMutex)
	return rwMutex
}

// Lock acquires an exclusive write lock for a project
func (m *ProjectLockMap) Lock(projectID string) {
	m.getOrCreateLock(projectID).Lock()
}

// Unlock releases the write lock for a project
func (m *ProjectLockMap) Unlock(projectID string) {
	m.getOrCreateLock(projectID).Unlock()
}

// RLock acquires a read lock for a project
func (m *ProjectLockMap) RLock(projectID string) {
	m.getOrCreateLock(projectID).RLock()
}

// RUnlock releases the read lock for a project
func (m *ProjectLockMap) RUnlock(projectID string) {
	m.getOrCreateLock(projectID).RUnlock()
}

// Delete removes the lock for a project (call after project deletion)
func (m *ProjectLockMap) Delete(projectID string) {
	m.locks.Delete(projectID)
}
