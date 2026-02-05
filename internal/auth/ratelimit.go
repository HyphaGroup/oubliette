package auth

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter provides per-token rate limiting
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit // requests per second
	burst    int        // max burst size
}

// NewRateLimiter creates a new rate limiter
// rate: requests per second allowed
// burst: maximum burst size (requests allowed at once)
func NewRateLimiter(requestsPerSecond float64, burst int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(requestsPerSecond),
		burst:    burst,
	}
}

// DefaultRateLimiter returns a rate limiter with sensible defaults
// 10 requests/second with burst of 20
func DefaultRateLimiter() *RateLimiter {
	return NewRateLimiter(10, 20)
}

// getLimiter returns the rate limiter for a given key (token ID)
func (r *RateLimiter) getLimiter(key string) *rate.Limiter {
	r.mu.RLock()
	limiter, exists := r.limiters[key]
	r.mu.RUnlock()

	if exists {
		return limiter
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if limiter, exists = r.limiters[key]; exists {
		return limiter
	}

	limiter = rate.NewLimiter(r.rate, r.burst)
	r.limiters[key] = limiter
	return limiter
}

// Allow checks if a request should be allowed for the given key
func (r *RateLimiter) Allow(key string) bool {
	return r.getLimiter(key).Allow()
}

// Cleanup removes stale limiters that haven't been used recently
// Call this periodically to prevent memory growth
func (r *RateLimiter) Cleanup(maxAge time.Duration) {
	// For simplicity, we clear all limiters periodically
	// In production, you'd track last-used time per limiter
	r.mu.Lock()
	defer r.mu.Unlock()
	r.limiters = make(map[string]*rate.Limiter)
}

// RateLimitMiddleware creates HTTP middleware for rate limiting
// Must be applied AFTER auth middleware (needs token from context)
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get auth context to use token ID as rate limit key
			authCtx := FromContext(r.Context())

			var key string
			if authCtx != nil && authCtx.Token != nil {
				key = authCtx.Token.ID
			} else {
				// Fall back to IP address for unauthenticated requests
				key = r.RemoteAddr
			}

			if !limiter.Allow(key) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"jsonrpc": "2.0",
					"error": map[string]interface{}{
						"code":    -32029,
						"message": "Rate limit exceeded. Please slow down.",
					},
					"id": nil,
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
