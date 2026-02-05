package auth

import (
	"context"
	"testing"
)

func TestWithContext_FromContext(t *testing.T) {
	authCtx := &AuthContext{
		Type:  AuthTypeToken,
		Token: &Token{ID: "test-id", Name: "test", Scope: ScopeAdmin},
	}

	ctx := WithContext(context.Background(), authCtx)

	got := FromContext(ctx)
	if got == nil {
		t.Fatal("FromContext() returned nil")
	}

	if got.Token.ID != "test-id" {
		t.Errorf("FromContext().Token.ID = %v, want test-id", got.Token.ID)
	}
}

func TestFromContext_NoAuth(t *testing.T) {
	ctx := context.Background()

	got := FromContext(ctx)
	if got != nil {
		t.Error("FromContext() should return nil for context without auth")
	}
}

func TestFromContext_WrongType(t *testing.T) {
	// Store something other than AuthContext at the key
	ctx := context.WithValue(context.Background(), authContextKey, "not-auth-context")

	got := FromContext(ctx)
	if got != nil {
		t.Error("FromContext() should return nil for wrong type")
	}
}
