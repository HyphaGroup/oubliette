package mcp

import (
	"context"
	"testing"

	"github.com/HyphaGroup/oubliette/internal/auth"
)

func TestRequireAuth(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		wantErr bool
	}{
		{
			name:    "no auth context",
			ctx:     context.Background(),
			wantErr: true,
		},
		{
			name: "with auth context",
			ctx: auth.WithContext(context.Background(), &auth.AuthContext{
				Type:  auth.AuthTypeToken,
				Token: &auth.Token{ID: "test", Scope: auth.ScopeAdmin},
			}),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := requireAuth(tt.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("requireAuth() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRequireProjectAccess(t *testing.T) {
	tests := []struct {
		name      string
		ctx       context.Context
		projectID string
		wantErr   bool
	}{
		{
			name:      "no auth context",
			ctx:       context.Background(),
			projectID: "proj-1",
			wantErr:   true,
		},
		{
			name: "admin can access any project",
			ctx: auth.WithContext(context.Background(), &auth.AuthContext{
				Type:  auth.AuthTypeToken,
				Token: &auth.Token{ID: "test", Scope: auth.ScopeAdmin},
			}),
			projectID: "proj-1",
			wantErr:   false,
		},
		{
			name: "project scope can access matching project",
			ctx: auth.WithContext(context.Background(), &auth.AuthContext{
				Type:  auth.AuthTypeToken,
				Token: &auth.Token{ID: "test", Scope: "project:proj-1"},
			}),
			projectID: "proj-1",
			wantErr:   false,
		},
		{
			name: "project scope cannot access different project",
			ctx: auth.WithContext(context.Background(), &auth.AuthContext{
				Type:  auth.AuthTypeToken,
				Token: &auth.Token{ID: "test", Scope: "project:proj-1"},
			}),
			projectID: "proj-2",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := requireProjectAccess(tt.ctx, tt.projectID)
			if (err != nil) != tt.wantErr {
				t.Errorf("requireProjectAccess() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRequireWriteAccess(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		wantErr bool
	}{
		{
			name:    "no auth context",
			ctx:     context.Background(),
			wantErr: true,
		},
		{
			name: "admin can write",
			ctx: auth.WithContext(context.Background(), &auth.AuthContext{
				Type:  auth.AuthTypeToken,
				Token: &auth.Token{ID: "test", Scope: auth.ScopeAdmin},
			}),
			wantErr: false,
		},
		{
			name: "read-only cannot write",
			ctx: auth.WithContext(context.Background(), &auth.AuthContext{
				Type:  auth.AuthTypeToken,
				Token: &auth.Token{ID: "test", Scope: auth.ScopeReadOnly},
			}),
			wantErr: true,
		},
		{
			name: "project scope can write",
			ctx: auth.WithContext(context.Background(), &auth.AuthContext{
				Type:  auth.AuthTypeToken,
				Token: &auth.Token{ID: "test", Scope: "project:proj-1"},
			}),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := requireWriteAccess(tt.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("requireWriteAccess() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRequireAdmin(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		wantErr bool
	}{
		{
			name:    "no auth context",
			ctx:     context.Background(),
			wantErr: true,
		},
		{
			name: "admin scope passes",
			ctx: auth.WithContext(context.Background(), &auth.AuthContext{
				Type:  auth.AuthTypeToken,
				Token: &auth.Token{ID: "test", Scope: auth.ScopeAdmin},
			}),
			wantErr: false,
		},
		{
			name: "read-only scope fails",
			ctx: auth.WithContext(context.Background(), &auth.AuthContext{
				Type:  auth.AuthTypeToken,
				Token: &auth.Token{ID: "test", Scope: auth.ScopeReadOnly},
			}),
			wantErr: true,
		},
		{
			name: "project scope fails",
			ctx: auth.WithContext(context.Background(), &auth.AuthContext{
				Type:  auth.AuthTypeToken,
				Token: &auth.Token{ID: "test", Scope: "project:proj-1"},
			}),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := requireAdmin(tt.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("requireAdmin() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetTokenInfo(t *testing.T) {
	tests := []struct {
		name      string
		authCtx   *auth.AuthContext
		wantID    string
		wantScope string
	}{
		{
			name:      "nil auth context",
			authCtx:   nil,
			wantID:    "",
			wantScope: "",
		},
		{
			name:      "nil token",
			authCtx:   &auth.AuthContext{Type: auth.AuthTypeToken, Token: nil},
			wantID:    "",
			wantScope: "",
		},
		{
			name: "valid token",
			authCtx: &auth.AuthContext{
				Type:  auth.AuthTypeToken,
				Token: &auth.Token{ID: "token-123", Scope: auth.ScopeAdmin},
			},
			wantID:    "token-123",
			wantScope: auth.ScopeAdmin,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, scope := getTokenInfo(tt.authCtx)
			if id != tt.wantID {
				t.Errorf("getTokenInfo() id = %v, want %v", id, tt.wantID)
			}
			if scope != tt.wantScope {
				t.Errorf("getTokenInfo() scope = %v, want %v", scope, tt.wantScope)
			}
		})
	}
}
