package mcp

import (
	"testing"

	"github.com/HyphaGroup/oubliette/internal/auth"
)

func TestIsValidScope(t *testing.T) {
	tests := []struct {
		name  string
		scope string
		want  bool
	}{
		{"admin scope", auth.ScopeAdmin, true},
		{"admin:ro scope", auth.ScopeAdminRO, true},
		{"project scope", "project:550e8400-e29b-41d4-a716-446655440000", true},
		{"short project scope", "project:a", true}, // Just needs prefix
		{"empty", "", false},
		{"random string", "invalid", false},
		{"project without id", "project:", false}, // Too short after prefix
		{"partial project", "projec", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidScope(tt.scope)
			if got != tt.want {
				t.Errorf("isValidScope(%q) = %v, want %v", tt.scope, got, tt.want)
			}
		})
	}
}

func TestMaskToken(t *testing.T) {
	tests := []struct {
		name    string
		tokenID string
		want    string
	}{
		{"empty", "", "***"},
		{"short token", "abc", "***"},
		{"12 char token", "123456789012", "***"},
		{"13 char token", "1234567890123", "12345678...0123"},
		{"long token", "abc123def456ghi789", "abc123de...i789"},
		{"typical UUID-like", "oubliette_1234567890abcdef", "oubliett...cdef"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskToken(tt.tokenID)
			if got != tt.want {
				t.Errorf("maskToken(%q) = %q, want %q", tt.tokenID, got, tt.want)
			}
		})
	}
}

func TestGetTokenInfo_EdgeCases(t *testing.T) {
	t.Run("nil context", func(t *testing.T) {
		id, scope := getTokenInfo(nil)
		if id != "" || scope != "" {
			t.Errorf("getTokenInfo(nil) = (%q, %q), want (\"\", \"\")", id, scope)
		}
	})

	t.Run("nil token", func(t *testing.T) {
		authCtx := &auth.AuthContext{Token: nil}
		id, scope := getTokenInfo(authCtx)
		if id != "" || scope != "" {
			t.Errorf("getTokenInfo(nil token) = (%q, %q), want (\"\", \"\")", id, scope)
		}
	})

	t.Run("valid token", func(t *testing.T) {
		authCtx := &auth.AuthContext{
			Token: &auth.Token{
				ID:    "token-123",
				Scope: "admin",
			},
		}
		id, scope := getTokenInfo(authCtx)
		if id != "token-123" {
			t.Errorf("getTokenInfo().id = %q, want %q", id, "token-123")
		}
		if scope != "admin" {
			t.Errorf("getTokenInfo().scope = %q, want %q", scope, "admin")
		}
	})
}

func TestTokenCreateParams(t *testing.T) {
	params := TokenCreateParams{
		Name:  "test-token",
		Scope: "admin",
	}

	if params.Name != "test-token" {
		t.Errorf("Name = %q, want %q", params.Name, "test-token")
	}
	if params.Scope != "admin" {
		t.Errorf("Scope = %q, want %q", params.Scope, "admin")
	}
}

func TestTokenRevokeParams(t *testing.T) {
	params := TokenRevokeParams{
		TokenID: "token-to-revoke",
	}

	if params.TokenID != "token-to-revoke" {
		t.Errorf("TokenID = %q, want %q", params.TokenID, "token-to-revoke")
	}
}
