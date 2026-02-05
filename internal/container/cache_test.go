package container

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockRuntimeForCache is a minimal mock for cache testing
type mockRuntimeForCache struct {
	statusCalls atomic.Int32
	statusValue ContainerStatus
	statusError error
	startCalls  atomic.Int32
	startError  error
	stopCalls   atomic.Int32
	stopError   error
	removeCalls atomic.Int32
	removeError error
	createCalls atomic.Int32
	createError error
	createID    string
}

func (m *mockRuntimeForCache) Create(ctx context.Context, config CreateConfig) (string, error) {
	m.createCalls.Add(1)
	if m.createError != nil {
		return "", m.createError
	}
	if m.createID != "" {
		return m.createID, nil
	}
	return "mock-" + config.Name, nil
}

func (m *mockRuntimeForCache) Start(ctx context.Context, containerID string) error {
	m.startCalls.Add(1)
	return m.startError
}

func (m *mockRuntimeForCache) Stop(ctx context.Context, containerID string) error {
	m.stopCalls.Add(1)
	return m.stopError
}

func (m *mockRuntimeForCache) Remove(ctx context.Context, containerID string, force bool) error {
	m.removeCalls.Add(1)
	return m.removeError
}

func (m *mockRuntimeForCache) Exec(ctx context.Context, containerID string, config ExecConfig) (*ExecResult, error) {
	return &ExecResult{}, nil
}

func (m *mockRuntimeForCache) ExecInteractive(ctx context.Context, containerID string, config ExecConfig) (*InteractiveExec, error) {
	return nil, errors.New("not implemented")
}

func (m *mockRuntimeForCache) Inspect(ctx context.Context, containerID string) (*ContainerInfo, error) {
	return nil, errors.New("not implemented")
}

func (m *mockRuntimeForCache) Logs(ctx context.Context, containerID string, opts LogsOptions) (string, error) {
	return "", nil
}

func (m *mockRuntimeForCache) Status(ctx context.Context, containerID string) (ContainerStatus, error) {
	m.statusCalls.Add(1)
	if m.statusError != nil {
		return StatusUnknown, m.statusError
	}
	return m.statusValue, nil
}

func (m *mockRuntimeForCache) Build(ctx context.Context, config BuildConfig) error {
	return nil
}

func (m *mockRuntimeForCache) ImageExists(ctx context.Context, imageName string) (bool, error) {
	return true, nil
}

func (m *mockRuntimeForCache) Pull(ctx context.Context, imageName string) error {
	return nil
}

func (m *mockRuntimeForCache) Ping(ctx context.Context) error {
	return nil
}

func (m *mockRuntimeForCache) Close() error {
	return nil
}

func (m *mockRuntimeForCache) Name() string {
	return "mock"
}

func (m *mockRuntimeForCache) IsAvailable() bool {
	return true
}

func TestCachedRuntime_StatusCaching(t *testing.T) {
	mock := &mockRuntimeForCache{statusValue: StatusRunning}
	cr := NewCachedRuntime(mock, 100*time.Millisecond)
	defer func() { _ = cr.Close() }()

	ctx := context.Background()

	// First call should hit underlying runtime
	status, err := cr.Status(ctx, "container-1")
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status != StatusRunning {
		t.Errorf("Status() = %v, want Running", status)
	}
	if mock.statusCalls.Load() != 1 {
		t.Errorf("statusCalls = %v, want 1", mock.statusCalls.Load())
	}

	// Second call should be cached
	status, err = cr.Status(ctx, "container-1")
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status != StatusRunning {
		t.Errorf("Status() = %v, want Running", status)
	}
	if mock.statusCalls.Load() != 1 {
		t.Errorf("statusCalls = %v, want 1 (cached)", mock.statusCalls.Load())
	}
}

func TestCachedRuntime_TTLExpiry(t *testing.T) {
	mock := &mockRuntimeForCache{statusValue: StatusRunning}
	ttl := 50 * time.Millisecond
	cr := NewCachedRuntime(mock, ttl)
	defer func() { _ = cr.Close() }()

	ctx := context.Background()

	// First call
	_, _ = cr.Status(ctx, "container-1")
	if mock.statusCalls.Load() != 1 {
		t.Errorf("statusCalls = %v, want 1", mock.statusCalls.Load())
	}

	// Wait for TTL to expire
	time.Sleep(ttl + 10*time.Millisecond)

	// Should hit underlying runtime again
	_, _ = cr.Status(ctx, "container-1")
	if mock.statusCalls.Load() != 2 {
		t.Errorf("statusCalls = %v, want 2 (after TTL)", mock.statusCalls.Load())
	}
}

func TestCachedRuntime_InvalidateStatus(t *testing.T) {
	mock := &mockRuntimeForCache{statusValue: StatusRunning}
	cr := NewCachedRuntime(mock, 10*time.Second) // Long TTL
	defer func() { _ = cr.Close() }()

	ctx := context.Background()

	// First call
	_, _ = cr.Status(ctx, "container-1")
	if mock.statusCalls.Load() != 1 {
		t.Errorf("statusCalls = %v, want 1", mock.statusCalls.Load())
	}

	// Invalidate
	cr.InvalidateStatus("container-1")

	// Should hit underlying runtime again
	_, _ = cr.Status(ctx, "container-1")
	if mock.statusCalls.Load() != 2 {
		t.Errorf("statusCalls = %v, want 2 (after invalidate)", mock.statusCalls.Load())
	}
}

func TestCachedRuntime_InvalidateAll(t *testing.T) {
	mock := &mockRuntimeForCache{statusValue: StatusRunning}
	cr := NewCachedRuntime(mock, 10*time.Second)
	defer func() { _ = cr.Close() }()

	ctx := context.Background()

	// Cache multiple containers
	_, _ = cr.Status(ctx, "container-1")
	_, _ = cr.Status(ctx, "container-2")
	if mock.statusCalls.Load() != 2 {
		t.Errorf("statusCalls = %v, want 2", mock.statusCalls.Load())
	}

	// Invalidate all
	cr.InvalidateAll()

	// Both should hit underlying runtime again
	_, _ = cr.Status(ctx, "container-1")
	_, _ = cr.Status(ctx, "container-2")
	if mock.statusCalls.Load() != 4 {
		t.Errorf("statusCalls = %v, want 4 (after invalidate all)", mock.statusCalls.Load())
	}
}

func TestCachedRuntime_StartInvalidatesCache(t *testing.T) {
	mock := &mockRuntimeForCache{statusValue: StatusStopped}
	cr := NewCachedRuntime(mock, 10*time.Second)
	defer func() { _ = cr.Close() }()

	ctx := context.Background()

	// Cache status
	_, _ = cr.Status(ctx, "container-1")
	if mock.statusCalls.Load() != 1 {
		t.Errorf("statusCalls = %v, want 1", mock.statusCalls.Load())
	}

	// Start should invalidate cache
	mock.statusValue = StatusRunning
	_ = cr.Start(ctx, "container-1")

	// Next status call should hit runtime
	_, _ = cr.Status(ctx, "container-1")
	if mock.statusCalls.Load() != 2 {
		t.Errorf("statusCalls = %v, want 2 (after Start)", mock.statusCalls.Load())
	}
}

func TestCachedRuntime_StopInvalidatesCache(t *testing.T) {
	mock := &mockRuntimeForCache{statusValue: StatusRunning}
	cr := NewCachedRuntime(mock, 10*time.Second)
	defer func() { _ = cr.Close() }()

	ctx := context.Background()

	// Cache status
	_, _ = cr.Status(ctx, "container-1")

	// Stop should invalidate cache
	mock.statusValue = StatusStopped
	_ = cr.Stop(ctx, "container-1")

	// Next status call should hit runtime
	status, _ := cr.Status(ctx, "container-1")
	if status != StatusStopped {
		t.Errorf("Status() = %v, want Stopped", status)
	}
}

func TestCachedRuntime_RemoveInvalidatesCache(t *testing.T) {
	mock := &mockRuntimeForCache{statusValue: StatusRunning}
	cr := NewCachedRuntime(mock, 10*time.Second)
	defer func() { _ = cr.Close() }()

	ctx := context.Background()

	// Cache status
	_, _ = cr.Status(ctx, "container-1")

	// Remove should invalidate cache
	_ = cr.Remove(ctx, "container-1", false)

	// Next status call should hit runtime
	_, _ = cr.Status(ctx, "container-1")
	if mock.statusCalls.Load() != 2 {
		t.Errorf("statusCalls = %v, want 2 (after Remove)", mock.statusCalls.Load())
	}
}

func TestCachedRuntime_CreateCachesStatus(t *testing.T) {
	mock := &mockRuntimeForCache{createID: "new-container"}
	cr := NewCachedRuntime(mock, 10*time.Second)
	defer func() { _ = cr.Close() }()

	ctx := context.Background()

	// Create should cache "created" status
	id, err := cr.Create(ctx, CreateConfig{Name: "test"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if id != "new-container" {
		t.Errorf("Create() = %v, want new-container", id)
	}

	// Status should be cached as "created"
	status, _ := cr.Status(ctx, "new-container")
	if status != StatusCreated {
		t.Errorf("Status() = %v, want Created", status)
	}
	if mock.statusCalls.Load() != 0 {
		t.Errorf("statusCalls = %v, want 0 (cached from Create)", mock.statusCalls.Load())
	}
}

func TestCachedRuntime_StatusError(t *testing.T) {
	expectedErr := errors.New("container not found")
	mock := &mockRuntimeForCache{statusError: expectedErr}
	cr := NewCachedRuntime(mock, 100*time.Millisecond)
	defer func() { _ = cr.Close() }()

	ctx := context.Background()

	_, err := cr.Status(ctx, "container-1")
	if err != expectedErr {
		t.Errorf("Status() error = %v, want %v", err, expectedErr)
	}
}

func TestCachedRuntime_ConcurrentAccess(t *testing.T) {
	mock := &mockRuntimeForCache{statusValue: StatusRunning}
	cr := NewCachedRuntime(mock, 100*time.Millisecond)
	defer func() { _ = cr.Close() }()

	ctx := context.Background()
	var wg sync.WaitGroup

	// Concurrent status calls
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			containerID := "container-" + string(rune('0'+i%10))
			_, _ = cr.Status(ctx, containerID)
		}(i)
	}

	// Concurrent invalidations
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			containerID := "container-" + string(rune('0'+i%10))
			cr.InvalidateStatus(containerID)
		}(i)
	}

	wg.Wait()
	// Test passes if no race detected
}

func TestCachedRuntime_DefaultTTL(t *testing.T) {
	mock := &mockRuntimeForCache{}
	cr := NewCachedRuntime(mock, 0) // Zero TTL should use default
	defer func() { _ = cr.Close() }()

	_, ttl := cr.CacheStats()
	if ttl != 5*time.Second {
		t.Errorf("Default TTL = %v, want 5s", ttl)
	}
}

func TestCachedRuntime_CacheStats(t *testing.T) {
	mock := &mockRuntimeForCache{statusValue: StatusRunning}
	cr := NewCachedRuntime(mock, 10*time.Second)
	defer func() { _ = cr.Close() }()

	ctx := context.Background()

	// Initially empty
	size, ttl := cr.CacheStats()
	if size != 0 {
		t.Errorf("CacheStats size = %v, want 0", size)
	}
	if ttl != 10*time.Second {
		t.Errorf("CacheStats ttl = %v, want 10s", ttl)
	}

	// After caching
	_, _ = cr.Status(ctx, "container-1")
	_, _ = cr.Status(ctx, "container-2")

	size, _ = cr.CacheStats()
	if size != 2 {
		t.Errorf("CacheStats size = %v, want 2", size)
	}
}
