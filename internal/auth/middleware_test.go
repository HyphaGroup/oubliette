package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddleware_ValidToken(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create a token
	_, tokenID, _ := store.CreateToken("test-token", ScopeAdmin, nil)

	// Create handler that checks for auth context
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authCtx := FromContext(r.Context())
		if authCtx == nil {
			t.Error("Expected auth context to be set")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if authCtx.Token.Scope != ScopeAdmin {
			t.Errorf("Expected admin scope, got %v", authCtx.Token.Scope)
		}
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with middleware
	wrapped := Middleware(store)(handler)

	// Make request with valid token
	req := httptest.NewRequest("GET", "/", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+tokenID)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %v, want 200", rec.Code)
	}
}

func TestMiddleware_MissingToken(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called without auth")
	})

	wrapped := Middleware(store)(handler)

	// Make request without token
	req := httptest.NewRequest("GET", "/", http.NoBody)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Status = %v, want 401", rec.Code)
	}

	// Should be JSON-RPC error
	var resp map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["error"] == nil {
		t.Error("Response should contain error field")
	}
}

func TestMiddleware_InvalidToken(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called with invalid token")
	})

	wrapped := Middleware(store)(handler)

	// Make request with invalid token
	req := httptest.NewRequest("GET", "/", http.NoBody)
	req.Header.Set("Authorization", "Bearer oub_invalid")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Status = %v, want 401", rec.Code)
	}
}

func TestMiddleware_MalformedAuthHeader(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called with malformed auth")
	})

	wrapped := Middleware(store)(handler)

	tests := []struct {
		name   string
		header string
	}{
		{"Basic auth", "Basic dXNlcjpwYXNz"},
		{"No bearer prefix", "token123"},
		{"Empty bearer", "Bearer "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", http.NoBody)
			req.Header.Set("Authorization", tt.header)
			rec := httptest.NewRecorder()

			wrapped.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("Status = %v, want 401", rec.Code)
			}
		})
	}
}

func TestRateLimitMiddleware_AllowsRequests(t *testing.T) {
	limiter := NewRateLimiter(100, 10)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := RateLimitMiddleware(limiter)(handler)

	// Make request
	req := httptest.NewRequest("GET", "/", http.NoBody)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %v, want 200", rec.Code)
	}
}

func TestRateLimitMiddleware_BlocksOverLimit(t *testing.T) {
	limiter := NewRateLimiter(0.01, 1) // Very low limit

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := RateLimitMiddleware(limiter)(handler)

	// First request should succeed
	req1 := httptest.NewRequest("GET", "/", http.NoBody)
	req1.RemoteAddr = "192.168.1.1:12345"
	rec1 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Errorf("First request status = %v, want 200", rec1.Code)
	}

	// Second request should be rate limited
	req2 := httptest.NewRequest("GET", "/", http.NoBody)
	req2.RemoteAddr = "192.168.1.1:12345"
	rec2 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusTooManyRequests {
		t.Errorf("Second request status = %v, want 429", rec2.Code)
	}

	// Check Retry-After header
	if rec2.Header().Get("Retry-After") == "" {
		t.Error("Missing Retry-After header")
	}
}

func TestRateLimitMiddleware_UsesTokenID(t *testing.T) {
	limiter := NewRateLimiter(0.01, 1)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := RateLimitMiddleware(limiter)(handler)

	// Request with auth context
	req := httptest.NewRequest("GET", "/", http.NoBody)
	authCtx := &AuthContext{Type: AuthTypeToken, Token: &Token{ID: "token-1"}}
	req = req.WithContext(WithContext(req.Context(), authCtx))
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %v, want 200", rec.Code)
	}
}

func Test_maskToken(t *testing.T) {
	tests := []struct {
		name    string
		tokenID string
		want    string
	}{
		{"short token", "abc", "***"},
		{"normal token", "oub_1234567890abcdefghij", "oub_1234...ghij"},
		{"exact 12 chars", "123456789012", "***"},
		{"13 chars", "1234567890123", "12345678...0123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := maskToken(tt.tokenID); got != tt.want {
				t.Errorf("maskToken() = %v, want %v", got, tt.want)
			}
		})
	}
}
