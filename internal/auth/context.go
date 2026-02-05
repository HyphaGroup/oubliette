package auth

import (
	"context"
)

type contextKey string

const authContextKey contextKey = "auth"

// WithContext adds an AuthContext to the context
func WithContext(ctx context.Context, auth *AuthContext) context.Context {
	return context.WithValue(ctx, authContextKey, auth)
}

// FromContext retrieves the AuthContext from the context
func FromContext(ctx context.Context) *AuthContext {
	auth, ok := ctx.Value(authContextKey).(*AuthContext)
	if !ok {
		return nil
	}
	return auth
}
