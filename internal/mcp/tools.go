package mcp

import (
	"context"

	"github.com/HyphaGroup/oubliette/internal/auth"
)

// ToolDefinition represents an Oubliette tool that can be exposed via socket
type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// ToolTarget defines what a tool operates on
type ToolTarget string

const (
	// TargetGlobal - tool operates system-wide (e.g., project_list, token_create)
	TargetGlobal ToolTarget = "global"
	// TargetProject - tool operates on a specific project
	TargetProject ToolTarget = "project"
)

// ToolAccess defines the access level required for a tool
type ToolAccess string

const (
	// AccessRead - read-only operation
	AccessRead ToolAccess = "read"
	// AccessWrite - modifies data
	AccessWrite ToolAccess = "write"
	// AccessAdmin - admin-only (token management)
	AccessAdmin ToolAccess = "admin"
)

// getToolsForScope returns tool definitions available for the given token scope
func (s *Server) getToolsForScope(scope string) []ToolDefinition {
	tools := s.registry.GetToolsForScope(scope)
	result := make([]ToolDefinition, len(tools))
	for i, t := range tools {
		result[i] = ToolDefinition{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		}
	}
	return result
}

// isToolAllowedWithProject checks if a specific tool name is allowed for a token scope and project
func (s *Server) isToolAllowedWithProject(toolName, tokenScope, projectID string) bool {
	return s.registry.IsToolAllowedWithProject(toolName, tokenScope, projectID)
}

// dispatchToolCall routes a tool call to the appropriate handler with auth context
func (s *Server) dispatchToolCall(ctx context.Context, toolName string, arguments map[string]any, token *auth.Token) (map[string]any, error) {
	// Inject auth context
	authCtx := &auth.AuthContext{Type: auth.AuthTypeToken, Token: token}
	ctx = auth.WithContext(ctx, authCtx)

	// Use registry to dispatch - CallToolWithMap returns map already formatted
	return s.registry.CallToolWithMap(ctx, toolName, arguments)
}
