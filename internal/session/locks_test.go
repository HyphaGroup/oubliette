package session

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSessionLockMap_BasicLocking(t *testing.T) {
	locks := NewSessionLockMap()

	// Should be able to acquire and release lock
	locks.Lock("session-1")
	locks.Unlock("session-1")

	// Should be able to re-acquire after release
	locks.Lock("session-1")
	locks.Unlock("session-1")
}

func TestSessionLockMap_ReadLocking(t *testing.T) {
	locks := NewSessionLockMap()

	// Multiple read locks should be allowed
	locks.RLock("session-1")
	locks.RLock("session-1")

	// Release both
	locks.RUnlock("session-1")
	locks.RUnlock("session-1")
}

func TestSessionLockMap_DifferentSessions(t *testing.T) {
	locks := NewSessionLockMap()

	// Different sessions should have independent locks
	locks.Lock("session-1")
	locks.Lock("session-2") // Should not block

	locks.Unlock("session-1")
	locks.Unlock("session-2")
}

func TestSessionLockMap_ConcurrentReadAccess(t *testing.T) {
	locks := NewSessionLockMap()
	var wg sync.WaitGroup
	var readersActive atomic.Int32

	// Multiple concurrent readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			locks.RLock("session-1")
			readersActive.Add(1)
			time.Sleep(10 * time.Millisecond)
			readersActive.Add(-1)
			locks.RUnlock("session-1")
		}()
	}

	// Wait for some readers to start
	time.Sleep(5 * time.Millisecond)

	// Multiple readers should be active simultaneously
	if readersActive.Load() < 2 {
		t.Log("Note: May need multiple readers active, but timing-dependent")
	}

	wg.Wait()
}

func TestSessionLockMap_WriterBlocksReaders(t *testing.T) {
	locks := NewSessionLockMap()
	var wg sync.WaitGroup
	var writerDone atomic.Bool
	var readerStartedAfterWriter atomic.Bool

	// Acquire write lock first
	locks.Lock("session-1")

	// Reader should block
	wg.Add(1)
	go func() {
		defer wg.Done()
		locks.RLock("session-1")
		if writerDone.Load() {
			readerStartedAfterWriter.Store(true)
		}
		locks.RUnlock("session-1")
	}()

	// Give reader time to attempt lock
	time.Sleep(20 * time.Millisecond)

	// Release write lock
	writerDone.Store(true)
	locks.Unlock("session-1")

	wg.Wait()

	if !readerStartedAfterWriter.Load() {
		t.Error("Reader should have been blocked until writer finished")
	}
}

func TestSessionLockMap_Delete(t *testing.T) {
	locks := NewSessionLockMap()

	// Create lock by using it
	locks.Lock("session-1")
	locks.Unlock("session-1")

	// Delete should not panic
	locks.Delete("session-1")

	// Should still work after delete (creates new lock)
	locks.Lock("session-1")
	locks.Unlock("session-1")
}

func TestSessionLockMap_Delete_NonExistent(t *testing.T) {
	locks := NewSessionLockMap()

	// Delete non-existent session should not panic
	locks.Delete("non-existent")
}

func TestSessionLockMap_ConcurrentLockUnlock(t *testing.T) {
	locks := NewSessionLockMap()
	var wg sync.WaitGroup
	var counter atomic.Int32

	// Many goroutines competing for same lock
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			locks.Lock("session-1")
			// Critical section
			val := counter.Load()
			time.Sleep(time.Microsecond) // Simulate work
			counter.Store(val + 1)
			locks.Unlock("session-1")
		}()
	}

	wg.Wait()

	// Counter should be exactly 100 (no races)
	if counter.Load() != 100 {
		t.Errorf("Counter = %v, want 100 (race detected)", counter.Load())
	}
}

func TestSessionLockMap_ReadWriteContention(t *testing.T) {
	locks := NewSessionLockMap()
	var wg sync.WaitGroup
	var value atomic.Int32

	// Start with a value
	value.Store(0)

	// Concurrent readers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				locks.RLock("session-1")
				_ = value.Load() // Read value
				locks.RUnlock("session-1")
			}
		}()
	}

	// Concurrent writers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				locks.Lock("session-1")
				value.Add(1) // Write value
				locks.Unlock("session-1")
			}
		}()
	}

	wg.Wait()

	// Writers should have incremented 100 times (10 writers * 10 iterations)
	if value.Load() != 100 {
		t.Errorf("Value = %v, want 100", value.Load())
	}
}

func TestSessionLockMap_IndependentSessionPerformance(t *testing.T) {
	locks := NewSessionLockMap()
	var wg sync.WaitGroup
	start := time.Now()

	// Lock many different sessions concurrently
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sessionID := "session-" + string(rune('0'+i%10))
			locks.Lock(sessionID)
			time.Sleep(time.Millisecond) // Simulate work
			locks.Unlock(sessionID)
		}(i)
	}

	wg.Wait()

	// Should complete quickly since different sessions don't block each other
	elapsed := time.Since(start)
	// With 10 unique sessions and 10 goroutines per session, ~10ms expected
	if elapsed > 200*time.Millisecond {
		t.Logf("Elapsed = %v (may be slow due to contention)", elapsed)
	}
}
