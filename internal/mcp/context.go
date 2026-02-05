package mcp

import (
	"context"
	"strconv"
)

// Context keys for MCP headers
type contextKey string

const (
	contextKeySessionID  contextKey = "oubliette-session-id"
	contextKeyProject    contextKey = "oubliette-project"
	contextKeyDepth      contextKey = "oubliette-depth"
	contextKeyRemoteAddr contextKey = "oubliette-remote-addr"
)

// ExtractMCPContext extracts Oubliette-specific headers from MCP request context
// These headers are injected by the MCP client when connecting
func ExtractMCPContext(ctx context.Context) MCPContext {
	return MCPContext{
		SessionID: getStringFromContext(ctx, contextKeySessionID),
		ProjectID: getStringFromContext(ctx, contextKeyProject),
		Depth:     getIntFromContext(ctx, contextKeyDepth),
	}
}

// MCPContext holds Oubliette-specific context from MCP headers
type MCPContext struct {
	SessionID  string
	ProjectID  string
	Depth      int
	RemoteAddr string
}

// WithRemoteAddr adds the remote address to context
func WithRemoteAddr(ctx context.Context, addr string) context.Context {
	return context.WithValue(ctx, contextKeyRemoteAddr, addr)
}

// GetRemoteAddr extracts the remote address from context
func GetRemoteAddr(ctx context.Context) string {
	return getStringFromContext(ctx, contextKeyRemoteAddr)
}

// WithMCPHeaders adds Oubliette-specific headers to context
func WithMCPHeaders(ctx context.Context, sessionID, projectID string, depth int) context.Context {
	ctx = context.WithValue(ctx, contextKeySessionID, sessionID)
	ctx = context.WithValue(ctx, contextKeyProject, projectID)
	ctx = context.WithValue(ctx, contextKeyDepth, strconv.Itoa(depth))
	return ctx
}

// GenerateMCPHeaders creates MCP header map for child gogol
func GenerateMCPHeaders(sessionID, projectID string, depth int) map[string]string {
	return map[string]string{
		"X-Oubliette-Session-ID": sessionID,
		"X-Oubliette-Project":    projectID,
		"X-Oubliette-Depth":      strconv.Itoa(depth),
	}
}

// Helper functions

func getStringFromContext(ctx context.Context, key contextKey) string {
	if val := ctx.Value(key); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getIntFromContext(ctx context.Context, key contextKey) int {
	if val := ctx.Value(key); val != nil {
		if str, ok := val.(string); ok {
			if i, err := strconv.Atoi(str); err == nil {
				return i
			}
		}
		if i, ok := val.(int); ok {
			return i
		}
	}
	return 0
}
