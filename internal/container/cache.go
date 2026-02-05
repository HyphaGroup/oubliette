package container

import (
	"context"
	"sync"
	"time"
)

// CachedRuntime wraps a Runtime and caches Status() calls with TTL.
// This reduces Docker/Apple Container API calls when listing projects
// or checking container status repeatedly.
type CachedRuntime struct {
	Runtime
	cache    map[string]*statusCacheEntry
	mu       sync.RWMutex
	ttl      time.Duration
	cleanupC chan struct{}
	closed   bool
}

type statusCacheEntry struct {
	status    ContainerStatus
	expiresAt time.Time
}

// NewCachedRuntime creates a new cached runtime wrapper with the specified TTL
func NewCachedRuntime(r Runtime, ttl time.Duration) *CachedRuntime {
	if ttl <= 0 {
		ttl = 5 * time.Second // Default 5 second TTL
	}
	cr := &CachedRuntime{
		Runtime:  r,
		cache:    make(map[string]*statusCacheEntry),
		ttl:      ttl,
		cleanupC: make(chan struct{}),
	}
	go cr.cleanupLoop()
	return cr
}

// Status returns the cached status or fetches from underlying runtime
func (cr *CachedRuntime) Status(ctx context.Context, containerID string) (ContainerStatus, error) {
	// Check cache first
	cr.mu.RLock()
	if entry, ok := cr.cache[containerID]; ok && time.Now().Before(entry.expiresAt) {
		status := entry.status
		cr.mu.RUnlock()
		return status, nil
	}
	cr.mu.RUnlock()

	// Cache miss - fetch from runtime
	status, err := cr.Runtime.Status(ctx, containerID)
	if err != nil {
		return status, err
	}

	// Store in cache
	cr.mu.Lock()
	cr.cache[containerID] = &statusCacheEntry{
		status:    status,
		expiresAt: time.Now().Add(cr.ttl),
	}
	cr.mu.Unlock()

	return status, nil
}

// InvalidateStatus removes a container from the cache
// Call this when container state changes (start, stop, remove)
func (cr *CachedRuntime) InvalidateStatus(containerID string) {
	cr.mu.Lock()
	delete(cr.cache, containerID)
	cr.mu.Unlock()
}

// InvalidateAll clears the entire cache
func (cr *CachedRuntime) InvalidateAll() {
	cr.mu.Lock()
	cr.cache = make(map[string]*statusCacheEntry)
	cr.mu.Unlock()
}

// Override lifecycle methods to invalidate cache on state changes

func (cr *CachedRuntime) Start(ctx context.Context, containerID string) error {
	err := cr.Runtime.Start(ctx, containerID)
	cr.InvalidateStatus(containerID)
	return err
}

func (cr *CachedRuntime) Stop(ctx context.Context, containerID string) error {
	err := cr.Runtime.Stop(ctx, containerID)
	cr.InvalidateStatus(containerID)
	return err
}

func (cr *CachedRuntime) Remove(ctx context.Context, containerID string, force bool) error {
	err := cr.Runtime.Remove(ctx, containerID, force)
	cr.InvalidateStatus(containerID)
	return err
}

func (cr *CachedRuntime) Create(ctx context.Context, config CreateConfig) (string, error) {
	id, err := cr.Runtime.Create(ctx, config)
	if err == nil {
		// New container starts in "created" state
		cr.mu.Lock()
		cr.cache[id] = &statusCacheEntry{
			status:    StatusCreated,
			expiresAt: time.Now().Add(cr.ttl),
		}
		// Also cache by name if provided
		if config.Name != "" {
			cr.cache[config.Name] = &statusCacheEntry{
				status:    StatusCreated,
				expiresAt: time.Now().Add(cr.ttl),
			}
		}
		cr.mu.Unlock()
	}
	return id, err
}

// Close stops the cleanup goroutine and closes the underlying runtime
func (cr *CachedRuntime) Close() error {
	cr.mu.Lock()
	if cr.closed {
		cr.mu.Unlock()
		return nil
	}
	cr.closed = true
	cr.mu.Unlock()

	close(cr.cleanupC)
	return cr.Runtime.Close()
}

// cleanupLoop periodically removes expired entries
func (cr *CachedRuntime) cleanupLoop() {
	ticker := time.NewTicker(cr.ttl * 2)
	defer ticker.Stop()

	for {
		select {
		case <-cr.cleanupC:
			return
		case <-ticker.C:
			cr.cleanup()
		}
	}
}

func (cr *CachedRuntime) cleanup() {
	now := time.Now()
	cr.mu.Lock()
	for id, entry := range cr.cache {
		if now.After(entry.expiresAt) {
			delete(cr.cache, id)
		}
	}
	cr.mu.Unlock()
}

// CacheStats returns cache statistics for monitoring
func (cr *CachedRuntime) CacheStats() (size int, ttl time.Duration) {
	cr.mu.RLock()
	size = len(cr.cache)
	cr.mu.RUnlock()
	return size, cr.ttl
}
