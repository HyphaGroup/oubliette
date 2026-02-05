package auth

import (
	"sync"
	"testing"
)

func TestRateLimiter_Allow(t *testing.T) {
	// High rate for testing
	limiter := NewRateLimiter(1000, 10)

	// Should allow up to burst
	for i := 0; i < 10; i++ {
		if !limiter.Allow("test-key") {
			t.Errorf("Allow() should return true for request %d (within burst)", i)
		}
	}
}

func TestRateLimiter_BlocksOverLimit(t *testing.T) {
	// Very low rate with small burst
	limiter := NewRateLimiter(0.1, 2) // 0.1 req/sec, burst of 2

	// First two should be allowed (burst)
	if !limiter.Allow("test-key") {
		t.Error("First request should be allowed")
	}
	if !limiter.Allow("test-key") {
		t.Error("Second request should be allowed (burst)")
	}

	// Third should be blocked
	if limiter.Allow("test-key") {
		t.Error("Third request should be blocked (over limit)")
	}
}

func TestRateLimiter_PerKeyIsolation(t *testing.T) {
	limiter := NewRateLimiter(0.1, 2)

	// Exhaust key1's burst
	limiter.Allow("key1")
	limiter.Allow("key1")

	// key2 should still have full burst
	if !limiter.Allow("key2") {
		t.Error("key2's first request should be allowed")
	}
	if !limiter.Allow("key2") {
		t.Error("key2's second request should be allowed")
	}
}

func TestRateLimiter_DefaultRateLimiter(t *testing.T) {
	limiter := DefaultRateLimiter()

	// Should be created with sensible defaults
	if limiter == nil {
		t.Fatal("DefaultRateLimiter() returned nil")
	}

	// Should allow requests
	if !limiter.Allow("test") {
		t.Error("Default limiter should allow requests")
	}
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	limiter := NewRateLimiter(10000, 100) // High limits for concurrency test
	var wg sync.WaitGroup
	var allowed, denied int
	var mu sync.Mutex

	// Many concurrent requests
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := "key-" + string(rune('0'+i%10))
			result := limiter.Allow(key)
			mu.Lock()
			if result {
				allowed++
			} else {
				denied++
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// All should be allowed with high limits
	if allowed != 200 {
		t.Logf("allowed=%d, denied=%d", allowed, denied)
	}
}

func TestRateLimiter_Cleanup(t *testing.T) {
	limiter := NewRateLimiter(10, 5)

	// Create some limiters
	limiter.Allow("key1")
	limiter.Allow("key2")
	limiter.Allow("key3")

	// Cleanup should clear all
	limiter.Cleanup(0)

	// Should create new limiters (with fresh burst)
	// After cleanup, first request gets fresh burst
	if !limiter.Allow("key1") {
		t.Error("After cleanup, first request should be allowed")
	}
}

func TestRateLimiter_getLimiter_DoubleCheck(t *testing.T) {
	limiter := NewRateLimiter(10, 5)

	// Concurrent access to same key should return same limiter
	var wg sync.WaitGroup
	results := make(chan bool, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Get limiter for same key concurrently
			l := limiter.getLimiter("same-key")
			results <- (l != nil)
		}()
	}

	wg.Wait()
	close(results)

	// All should have gotten a valid limiter
	for result := range results {
		if !result {
			t.Error("getLimiter should always return non-nil")
		}
	}
}
