package mcp

import "github.com/HyphaGroup/oubliette/internal/auth"

// IsToolAllowed checks if a tool is allowed for a given token scope and project
// Uses new Target/Access model if set, falls back to legacy Scope if not
func IsToolAllowed(tool *ToolDef, tokenScope, projectID string) bool {
	// If new model is set, use it
	if tool.Target != "" && tool.Access != "" {
		return isToolAllowedNew(tool, tokenScope, projectID)
	}
	// Fall back to legacy scope check
	return isToolAllowedForTokenScope(tool.Scope, tokenScope)
}

// isToolAllowedNew implements the new permission model
func isToolAllowedNew(tool *ToolDef, tokenScope, projectID string) bool {
	isAdmin := tokenScope == auth.ScopeAdmin
	isAdminRO := auth.IsAdminScope(tokenScope) && auth.IsReadOnlyScope(tokenScope)
	isProjectScope := auth.IsProjectScope(tokenScope)
	isReadOnly := auth.IsReadOnlyScope(tokenScope)
	scopeProjectID := auth.ExtractProjectID(tokenScope)

	// Admin-only tools (token management) require full admin
	if tool.Access == AccessAdmin {
		return isAdmin
	}

	// Write access check - read-only tokens can't write
	if tool.Access == AccessWrite && isReadOnly {
		return false
	}

	// Global tools
	if tool.Target == TargetGlobal {
		// Admin scopes can access all global tools (respecting read/write)
		if isAdmin || isAdminRO {
			return true
		}
		// Project scopes can access global read tools only
		if isProjectScope && tool.Access == AccessRead {
			return true
		}
		return false
	}

	// Project-targeted tools
	if tool.Target == TargetProject {
		// Admin scopes can access any project
		if isAdmin || isAdminRO {
			return true
		}
		// Project scopes can only access their project
		if isProjectScope {
			// Empty projectID means we can't verify - deny for safety
			if projectID == "" {
				return false
			}
			return scopeProjectID == projectID
		}
	}

	return false
}

// ExtractProjectIDFromArgs extracts project ID from tool arguments
func ExtractProjectIDFromArgs(args map[string]any) string {
	if pid, ok := args["project_id"].(string); ok && pid != "" {
		return pid
	}
	return ""
}
