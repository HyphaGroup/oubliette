package auth

import (
	"testing"
	"time"
)

func TestStore_CreateAndValidateToken(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create token
	token, tokenID, err := store.CreateToken("test-token", ScopeAdmin, nil)
	if err != nil {
		t.Fatalf("CreateToken() error = %v", err)
	}

	if token.Name != "test-token" {
		t.Errorf("Token.Name = %v, want test-token", token.Name)
	}
	if token.Scope != ScopeAdmin {
		t.Errorf("Token.Scope = %v, want admin", token.Scope)
	}
	if !hasPrefix(tokenID, "oub_") {
		t.Errorf("Token ID should have prefix 'oub_', got %v", tokenID[:minInt(8, len(tokenID))])
	}

	// Validate token
	validated, err := store.ValidateToken(tokenID)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	if validated.ID != tokenID {
		t.Errorf("Validated token ID = %v, want %v", validated.ID, tokenID)
	}
}

func TestStore_ValidateToken_NotFound(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	_, err = store.ValidateToken("oub_nonexistent")
	if err != ErrTokenNotFound {
		t.Errorf("ValidateToken() error = %v, want ErrTokenNotFound", err)
	}
}

func TestStore_ValidateToken_InvalidFormat(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	_, err = store.ValidateToken("invalid-token")
	if err != ErrInvalidToken {
		t.Errorf("ValidateToken() error = %v, want ErrInvalidToken", err)
	}
}

func TestStore_ValidateToken_Expired(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create token that already expired
	expiredAt := time.Now().Add(-time.Hour)
	_, tokenID, err := store.CreateToken("expired-token", ScopeAdmin, &expiredAt)
	if err != nil {
		t.Fatalf("CreateToken() error = %v", err)
	}

	_, err = store.ValidateToken(tokenID)
	if err != ErrTokenExpired {
		t.Errorf("ValidateToken() error = %v, want ErrTokenExpired", err)
	}
}

func TestStore_ListTokens(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create multiple tokens
	_, _, _ = store.CreateToken("token1", ScopeAdmin, nil)
	_, _, _ = store.CreateToken("token2", ScopeReadOnly, nil)

	tokens, err := store.ListTokens()
	if err != nil {
		t.Fatalf("ListTokens() error = %v", err)
	}

	if len(tokens) != 2 {
		t.Errorf("ListTokens() count = %v, want 2", len(tokens))
	}
}

func TestStore_RevokeToken(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create token
	_, tokenID, _ := store.CreateToken("to-revoke", ScopeAdmin, nil)

	// Revoke it
	err = store.RevokeToken(tokenID)
	if err != nil {
		t.Fatalf("RevokeToken() error = %v", err)
	}

	// Validate should fail
	_, err = store.ValidateToken(tokenID)
	if err != ErrTokenNotFound {
		t.Errorf("ValidateToken() after revoke error = %v, want ErrTokenNotFound", err)
	}
}

func TestStore_RevokeToken_NotFound(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	err = store.RevokeToken("oub_nonexistent")
	if err != ErrTokenNotFound {
		t.Errorf("RevokeToken() error = %v, want ErrTokenNotFound", err)
	}
}

func TestStore_GetToken(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	_, tokenID, _ := store.CreateToken("test", ScopeAdmin, nil)

	token, err := store.GetToken(tokenID)
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}

	if token.Name != "test" {
		t.Errorf("GetToken().Name = %v, want test", token.Name)
	}
}

func TestStore_TokenWithExpiry(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create token with future expiry
	futureExpiry := time.Now().Add(time.Hour)
	token, tokenID, err := store.CreateToken("future-token", ScopeAdmin, &futureExpiry)
	if err != nil {
		t.Fatalf("CreateToken() error = %v", err)
	}

	if token.ExpiresAt == nil {
		t.Error("Token.ExpiresAt should not be nil")
	}

	// Should validate successfully
	_, err = store.ValidateToken(tokenID)
	if err != nil {
		t.Errorf("ValidateToken() error = %v (token not expired yet)", err)
	}
}

func TestStore_ProjectScopedToken(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	projectScope := ScopeProject("my-project-id")
	token, _, err := store.CreateToken("project-token", projectScope, nil)
	if err != nil {
		t.Fatalf("CreateToken() error = %v", err)
	}

	if token.Scope != "project:my-project-id" {
		t.Errorf("Token.Scope = %v, want project:my-project-id", token.Scope)
	}
}

func TestStore_NewStore_InvalidPath(t *testing.T) {
	// Try to create store in a path that can't be created
	// Using a null byte in path which is invalid on most systems
	_, err := NewStore("/dev/null/invalid")
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestStore_ValidateToken_Empty(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	_, err = store.ValidateToken("")
	if err != ErrInvalidToken {
		t.Errorf("ValidateToken() error = %v, want ErrInvalidToken", err)
	}
}

func TestStore_ValidateToken_ShortPrefix(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	_, err = store.ValidateToken("ou")
	if err != ErrInvalidToken {
		t.Errorf("ValidateToken() error = %v, want ErrInvalidToken", err)
	}
}

func TestStore_ValidateToken_WrongPrefix(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	_, err = store.ValidateToken("xxx_sometoken")
	if err != ErrInvalidToken {
		t.Errorf("ValidateToken() error = %v, want ErrInvalidToken", err)
	}
}

func TestStore_ListTokens_Empty(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	tokens, err := store.ListTokens()
	if err != nil {
		t.Fatalf("ListTokens() error = %v", err)
	}

	if len(tokens) != 0 {
		t.Errorf("ListTokens() count = %v, want 0", len(tokens))
	}
}

func TestStore_TokenWithLastUsed(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	_, tokenID, _ := store.CreateToken("test", ScopeAdmin, nil)

	// Validate to trigger updateLastUsed
	_, _ = store.ValidateToken(tokenID)

	// Give goroutine time to update
	time.Sleep(50 * time.Millisecond)

	// List should show last_used_at
	tokens, err := store.ListTokens()
	if err != nil {
		t.Fatalf("ListTokens() error = %v", err)
	}

	if len(tokens) != 1 {
		t.Fatalf("expected 1 token, got %d", len(tokens))
	}

	// LastUsedAt may or may not be set depending on timing
	// This is just for coverage of the scan path
}

func TestStore_RevokeToken_InvalidFormat(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Even with invalid format, RevokeToken just tries to delete
	// and returns ErrTokenNotFound if no rows affected
	err = store.RevokeToken("invalid")
	if err != ErrTokenNotFound {
		t.Errorf("RevokeToken() error = %v, want ErrTokenNotFound", err)
	}
}

func TestStore_OperationsOnClosedDB(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	// Close the database
	_ = store.Close()

	// Operations should fail
	_, _, err = store.CreateToken("test", ScopeAdmin, nil)
	if err == nil {
		t.Error("CreateToken on closed DB should fail")
	}

	_, err = store.ValidateToken("oub_test")
	if err == nil {
		t.Error("ValidateToken on closed DB should fail")
	}

	_, err = store.ListTokens()
	if err == nil {
		t.Error("ListTokens on closed DB should fail")
	}

	err = store.RevokeToken("oub_test")
	if err == nil {
		t.Error("RevokeToken on closed DB should fail")
	}
}

// Helper functions
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
