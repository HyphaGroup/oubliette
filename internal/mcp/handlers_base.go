package mcp

import (
	"context"
	"fmt"

	"github.com/HyphaGroup/oubliette/internal/auth"
)

// requireAuth extracts auth context and returns error if missing
func requireAuth(ctx context.Context) (*auth.AuthContext, error) {
	authCtx := auth.FromContext(ctx)
	if authCtx == nil {
		return nil, fmt.Errorf("authentication required")
	}
	return authCtx, nil
}

// requireProjectAccess checks if auth context can access the given project
func requireProjectAccess(ctx context.Context, projectID string) (*auth.AuthContext, error) {
	authCtx, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}
	if !authCtx.CanAccessProject(projectID) {
		return nil, fmt.Errorf("not authorized to access project %s", projectID)
	}
	return authCtx, nil
}

// requireWriteAccess checks if auth context can perform write operations
func requireWriteAccess(ctx context.Context) (*auth.AuthContext, error) {
	authCtx, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}
	if !authCtx.CanWrite() {
		return nil, fmt.Errorf("read-only access, write operations not permitted")
	}
	return authCtx, nil
}

// requireAdmin checks if auth context has admin scope
func requireAdmin(ctx context.Context) (*auth.AuthContext, error) {
	authCtx, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}
	if !authCtx.IsAdmin() {
		return nil, fmt.Errorf("admin access required")
	}
	return authCtx, nil
}
