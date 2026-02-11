package auth

import (
	"testing"
)

func TestAuthContext_CanAccessProject(t *testing.T) {
	tests := []struct {
		name      string
		authCtx   *AuthContext
		projectID string
		want      bool
	}{
		{
			name:      "nil token",
			authCtx:   &AuthContext{Type: AuthTypeToken, Token: nil},
			projectID: "proj-1",
			want:      false,
		},
		{
			name:      "admin scope can access any project",
			authCtx:   &AuthContext{Type: AuthTypeToken, Token: &Token{Scope: ScopeAdmin}},
			projectID: "proj-1",
			want:      true,
		},
		{
			name:      "admin:ro scope can access any project",
			authCtx:   &AuthContext{Type: AuthTypeToken, Token: &Token{Scope: ScopeAdminRO}},
			projectID: "proj-1",
			want:      true,
		},
		{
			name:      "project scope can access matching project",
			authCtx:   &AuthContext{Type: AuthTypeToken, Token: &Token{Scope: "project:proj-1"}},
			projectID: "proj-1",
			want:      true,
		},
		{
			name:      "project scope cannot access different project",
			authCtx:   &AuthContext{Type: AuthTypeToken, Token: &Token{Scope: "project:proj-1"}},
			projectID: "proj-2",
			want:      false,
		},
		{
			name:      "unknown scope cannot access project",
			authCtx:   &AuthContext{Type: AuthTypeToken, Token: &Token{Scope: "invalid"}},
			projectID: "proj-1",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.authCtx.CanAccessProject(tt.projectID); got != tt.want {
				t.Errorf("CanAccessProject() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthContext_CanWrite(t *testing.T) {
	tests := []struct {
		name    string
		authCtx *AuthContext
		want    bool
	}{
		{
			name:    "nil token",
			authCtx: &AuthContext{Type: AuthTypeToken, Token: nil},
			want:    false,
		},
		{
			name:    "admin scope can write",
			authCtx: &AuthContext{Type: AuthTypeToken, Token: &Token{Scope: ScopeAdmin}},
			want:    true,
		},
		{
			name:    "admin:ro scope cannot write",
			authCtx: &AuthContext{Type: AuthTypeToken, Token: &Token{Scope: ScopeAdminRO}},
			want:    false,
		},
		{
			name:    "project scope can write",
			authCtx: &AuthContext{Type: AuthTypeToken, Token: &Token{Scope: "project:proj-1"}},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.authCtx.CanWrite(); got != tt.want {
				t.Errorf("CanWrite() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthContext_IsAdmin(t *testing.T) {
	tests := []struct {
		name    string
		authCtx *AuthContext
		want    bool
	}{
		{
			name:    "nil token",
			authCtx: &AuthContext{Type: AuthTypeToken, Token: nil},
			want:    false,
		},
		{
			name:    "admin scope is admin",
			authCtx: &AuthContext{Type: AuthTypeToken, Token: &Token{Scope: ScopeAdmin}},
			want:    true,
		},
		{
			name:    "admin:ro scope is not admin",
			authCtx: &AuthContext{Type: AuthTypeToken, Token: &Token{Scope: ScopeAdminRO}},
			want:    false,
		},
		{
			name:    "project scope is not admin",
			authCtx: &AuthContext{Type: AuthTypeToken, Token: &Token{Scope: "project:proj-1"}},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.authCtx.IsAdmin(); got != tt.want {
				t.Errorf("IsAdmin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScopeProject(t *testing.T) {
	scope := ScopeProject("my-project-id")
	if scope != "project:my-project-id" {
		t.Errorf("ScopeProject() = %v, want project:my-project-id", scope)
	}
}

func TestScopeProjectRO(t *testing.T) {
	scope := ScopeProjectRO("my-project-id")
	if scope != "project:my-project-id:ro" {
		t.Errorf("ScopeProjectRO() = %v, want project:my-project-id:ro", scope)
	}
}

func TestIsAdminScope(t *testing.T) {
	tests := []struct {
		scope string
		want  bool
	}{
		{ScopeAdmin, true},
		{ScopeAdminRO, true},
		{"project:abc", false},
		{"project:abc:ro", false},
		{"invalid", false},
	}
	for _, tt := range tests {
		if got := IsAdminScope(tt.scope); got != tt.want {
			t.Errorf("IsAdminScope(%q) = %v, want %v", tt.scope, got, tt.want)
		}
	}
}

func TestIsProjectScope(t *testing.T) {
	tests := []struct {
		scope string
		want  bool
	}{
		{"project:abc", true},
		{"project:abc:ro", true},
		{"project:", true}, // edge case: prefix match
		{ScopeAdmin, false},
		{ScopeAdminRO, false},
		{"invalid", false},
	}
	for _, tt := range tests {
		if got := IsProjectScope(tt.scope); got != tt.want {
			t.Errorf("IsProjectScope(%q) = %v, want %v", tt.scope, got, tt.want)
		}
	}
}

func TestIsReadOnlyScope(t *testing.T) {
	tests := []struct {
		scope string
		want  bool
	}{
		{ScopeAdmin, false},
		{ScopeAdminRO, true},
		{"project:abc", false},
		{"project:abc:ro", true},
		{"invalid", false},
		{"invalid:ro", true}, // ends with :ro
	}
	for _, tt := range tests {
		if got := IsReadOnlyScope(tt.scope); got != tt.want {
			t.Errorf("IsReadOnlyScope(%q) = %v, want %v", tt.scope, got, tt.want)
		}
	}
}

func TestExtractProjectID(t *testing.T) {
	tests := []struct {
		scope string
		want  string
	}{
		{"project:abc-123", "abc-123"},
		{"project:abc-123:ro", "abc-123"},
		{"project:", ""},
		{"project::ro", ""}, // empty project ID
		{ScopeAdmin, ""},
		{"invalid", ""},
	}
	for _, tt := range tests {
		if got := ExtractProjectID(tt.scope); got != tt.want {
			t.Errorf("ExtractProjectID(%q) = %q, want %q", tt.scope, got, tt.want)
		}
	}
}

func TestAuthContext_CanAccessProject_NewScopes(t *testing.T) {
	tests := []struct {
		name      string
		scope     string
		projectID string
		want      bool
	}{
		{"admin:ro can access any project", ScopeAdminRO, "proj-1", true},
		{"project:ro can access own project", "project:proj-1:ro", "proj-1", true},
		{"project:ro cannot access other project", "project:proj-1:ro", "proj-2", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authCtx := &AuthContext{Type: AuthTypeToken, Token: &Token{Scope: tt.scope}}
			if got := authCtx.CanAccessProject(tt.projectID); got != tt.want {
				t.Errorf("CanAccessProject() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthContext_CanWrite_NewScopes(t *testing.T) {
	tests := []struct {
		name  string
		scope string
		want  bool
	}{
		{"admin:ro cannot write", ScopeAdminRO, false},
		{"project:ro cannot write", "project:proj-1:ro", false},
		{"project can write", "project:proj-1", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authCtx := &AuthContext{Type: AuthTypeToken, Token: &Token{Scope: tt.scope}}
			if got := authCtx.CanWrite(); got != tt.want {
				t.Errorf("CanWrite() = %v, want %v", got, tt.want)
			}
		})
	}
}
