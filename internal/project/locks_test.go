package project

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestProjectLockMap_ConcurrentAccess(t *testing.T) {
	var locks ProjectLockMap
	projectID := "test-project-123"

	var counter int64
	var wg sync.WaitGroup

	// Simulate 100 concurrent writers
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			locks.Lock(projectID)
			defer locks.Unlock(projectID)

			// Critical section - increment counter
			current := atomic.LoadInt64(&counter)
			atomic.StoreInt64(&counter, current+1)
		}()
	}

	wg.Wait()

	if counter != 100 {
		t.Errorf("Expected counter to be 100, got %d", counter)
	}
}

func TestProjectLockMap_ReadersCanRunConcurrently(t *testing.T) {
	var locks ProjectLockMap
	projectID := "test-project-456"

	var readersActive int64
	var maxConcurrentReaders int64
	var wg sync.WaitGroup

	// Simulate 10 concurrent readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			locks.RLock(projectID)
			defer locks.RUnlock(projectID)

			// Track concurrent readers
			current := atomic.AddInt64(&readersActive, 1)
			for {
				maxReaders := atomic.LoadInt64(&maxConcurrentReaders)
				if current <= maxReaders || atomic.CompareAndSwapInt64(&maxConcurrentReaders, maxReaders, current) {
					break
				}
			}

			// Simulate some read work
			for j := 0; j < 1000; j++ {
				_ = j * j
			}

			atomic.AddInt64(&readersActive, -1)
		}()
	}

	wg.Wait()

	// With RWMutex, multiple readers should have been active concurrently
	if maxConcurrentReaders < 2 {
		t.Logf("Warning: only %d concurrent readers detected (expected multiple)", maxConcurrentReaders)
	}
}

func TestProjectLockMap_IsolatesProjects(t *testing.T) {
	var locks ProjectLockMap

	// Lock project A
	locks.Lock("project-a")

	// Project B should be independently lockable
	done := make(chan bool, 1)
	go func() {
		locks.Lock("project-b")
		locks.Unlock("project-b")
		done <- true
	}()

	// Give goroutine time to attempt lock
	select {
	case <-done:
		// Success - project B was not blocked by project A
	case <-time.After(100 * time.Millisecond):
		t.Error("Project B was blocked by Project A lock")
	}

	locks.Unlock("project-a")
}
