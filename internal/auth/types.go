package auth

import (
	"strings"
	"time"
)

// Token represents an API token for MCP access
type Token struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Scope      string     `json:"scope"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

// Scope constants
const (
	ScopeAdmin    = "admin"
	ScopeAdminRO  = "admin:ro"
	ScopeReadOnly = "read-only" // Deprecated: use ScopeAdminRO
)

// ScopeProject returns a project-scoped scope string
func ScopeProject(projectID string) string {
	return "project:" + projectID
}

// ScopeProjectRO returns a read-only project-scoped scope string
func ScopeProjectRO(projectID string) string {
	return "project:" + projectID + ":ro"
}

// IsAdminScope returns true if scope is admin or admin:ro
func IsAdminScope(scope string) bool {
	return scope == ScopeAdmin || scope == ScopeAdminRO || scope == ScopeReadOnly
}

// IsProjectScope returns true if scope is project:<uuid> or project:<uuid>:ro
func IsProjectScope(scope string) bool {
	return strings.HasPrefix(scope, "project:")
}

// IsReadOnlyScope returns true if scope is read-only (admin:ro, project:*:ro, or legacy read-only)
func IsReadOnlyScope(scope string) bool {
	return scope == ScopeAdminRO || scope == ScopeReadOnly || strings.HasSuffix(scope, ":ro")
}

// ExtractProjectID extracts project ID from a project scope, returns empty if not a project scope
func ExtractProjectID(scope string) string {
	if !strings.HasPrefix(scope, "project:") {
		return ""
	}
	// Remove "project:" prefix
	rest := scope[8:]
	// Remove ":ro" suffix if present
	if strings.HasSuffix(rest, ":ro") {
		return rest[:len(rest)-3]
	}
	return rest
}

// AuthType represents the type of authentication used
type AuthType int

const (
	AuthTypeToken AuthType = iota
)

// AuthContext holds authentication information for a request
type AuthContext struct {
	Type  AuthType
	Token *Token
}

// CanAccessProject checks if the auth context allows access to a project
func (a *AuthContext) CanAccessProject(projectID string) bool {
	if a.Token == nil {
		return false
	}
	// Admin scopes (admin, admin:ro, read-only) can access any project
	if IsAdminScope(a.Token.Scope) {
		return true
	}
	// Project scopes can only access their specific project
	if IsProjectScope(a.Token.Scope) {
		return ExtractProjectID(a.Token.Scope) == projectID
	}
	return false
}

// CanWrite checks if the auth context allows write operations
func (a *AuthContext) CanWrite() bool {
	if a.Token == nil {
		return false
	}
	return !IsReadOnlyScope(a.Token.Scope)
}

// IsAdmin checks if the auth context has admin scope (full admin, not read-only)
func (a *AuthContext) IsAdmin() bool {
	if a.Type != AuthTypeToken || a.Token == nil {
		return false
	}
	return a.Token.Scope == ScopeAdmin
}
