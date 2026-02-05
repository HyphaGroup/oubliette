package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TokenParams is the unified params struct for the token tool
type TokenParams struct {
	Action string `json:"action"` // Required: create, list, revoke

	// For create
	Name  string `json:"name,omitempty"`
	Scope string `json:"scope,omitempty"`

	// For revoke
	TokenID string `json:"token_id,omitempty"`
}

var tokenActions = []string{"create", "list", "revoke"}

// handleToken is the unified handler for the token tool
func (s *Server) handleToken(ctx context.Context, request *mcp.CallToolRequest, params *TokenParams) (*mcp.CallToolResult, any, error) {
	if params.Action == "" {
		return nil, nil, missingActionError("token", tokenActions)
	}

	switch params.Action {
	case "create":
		return s.tokenCreate(ctx, request, params)
	case "list":
		return s.tokenList(ctx, request, params)
	case "revoke":
		return s.tokenRevoke(ctx, request, params)
	default:
		return nil, nil, actionError("token", params.Action, tokenActions)
	}
}

func (s *Server) tokenCreate(ctx context.Context, request *mcp.CallToolRequest, params *TokenParams) (*mcp.CallToolResult, any, error) {
	return s.handleTokenCreate(ctx, request, &TokenCreateParams{
		Name:  params.Name,
		Scope: params.Scope,
	})
}

func (s *Server) tokenList(ctx context.Context, request *mcp.CallToolRequest, params *TokenParams) (*mcp.CallToolResult, any, error) {
	return s.handleTokenList(ctx, request, &TokenListParams{})
}

func (s *Server) tokenRevoke(ctx context.Context, request *mcp.CallToolRequest, params *TokenParams) (*mcp.CallToolResult, any, error) {
	return s.handleTokenRevoke(ctx, request, &TokenRevokeParams{TokenID: params.TokenID})
}
