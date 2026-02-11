package mcp

import (
	"context"
	"fmt"

	"github.com/HyphaGroup/oubliette/internal/audit"
	"github.com/HyphaGroup/oubliette/internal/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Token Management Handlers

type TokenCreateParams struct {
	Name  string `json:"name"`
	Scope string `json:"scope"`
}

func (s *Server) handleTokenCreate(ctx context.Context, request *mcp.CallToolRequest, params *TokenCreateParams) (*mcp.CallToolResult, any, error) {
	authCtx, err := requireAdmin(ctx)
	if err != nil {
		return nil, nil, err
	}

	if params.Name == "" {
		return nil, nil, fmt.Errorf("name is required")
	}
	if params.Scope == "" {
		return nil, nil, fmt.Errorf("scope is required")
	}

	if !isValidScope(params.Scope) {
		return nil, nil, fmt.Errorf("invalid scope '%s'. Valid scopes: admin, read-only, project:<uuid>", params.Scope)
	}

	callerTokenID, callerScope := getTokenInfo(authCtx)
	token, tokenID, err := s.authStore.CreateToken(params.Name, params.Scope, nil)
	if err != nil {
		audit.LogFailure(audit.OpTokenCreate, callerTokenID, callerScope, "", err)
		return nil, nil, fmt.Errorf("failed to create token: %w", err)
	}

	audit.Log(&audit.Event{
		Operation:  audit.OpTokenCreate,
		TokenID:    callerTokenID,
		TokenScope: callerScope,
		Success:    true,
		Details:    map[string]interface{}{"new_token_name": params.Name, "new_token_scope": params.Scope},
	})

	result := "✅ Token created successfully!\n\n"
	result += fmt.Sprintf("Token ID: %s\n", tokenID)
	result += fmt.Sprintf("Name:     %s\n", token.Name)
	result += fmt.Sprintf("Scope:    %s\n", token.Scope)
	result += "\n⚠️  IMPORTANT: Save this token now. It cannot be retrieved later."

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: result},
		},
	}, nil, nil
}

type TokenListParams struct{}

func (s *Server) handleTokenList(ctx context.Context, request *mcp.CallToolRequest, params *TokenListParams) (*mcp.CallToolResult, any, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, nil, err
	}

	tokens, err := s.authStore.ListTokens()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list tokens: %w", err)
	}

	if len(tokens) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "No tokens found."},
			},
		}, nil, nil
	}

	result := fmt.Sprintf("Found %d token(s):\n\n", len(tokens))
	for _, t := range tokens {
		lastUsed := "never"
		if t.LastUsedAt != nil {
			lastUsed = t.LastUsedAt.Format("2006-01-02 15:04")
		}
		result += fmt.Sprintf("• %s\n", maskToken(t.ID))
		result += fmt.Sprintf("  Name:      %s\n", t.Name)
		result += fmt.Sprintf("  Scope:     %s\n", t.Scope)
		result += fmt.Sprintf("  Created:   %s\n", t.CreatedAt.Format("2006-01-02 15:04"))
		result += fmt.Sprintf("  Last Used: %s\n\n", lastUsed)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: result},
		},
	}, nil, nil
}

type TokenRevokeParams struct {
	TokenID string `json:"token_id"`
}

func (s *Server) handleTokenRevoke(ctx context.Context, request *mcp.CallToolRequest, params *TokenRevokeParams) (*mcp.CallToolResult, any, error) {
	authCtx, err := requireAdmin(ctx)
	if err != nil {
		return nil, nil, err
	}

	if params.TokenID == "" {
		return nil, nil, fmt.Errorf("token_id is required")
	}

	callerTokenID, callerScope := getTokenInfo(authCtx)
	if err := s.authStore.RevokeToken(params.TokenID); err != nil {
		audit.LogFailure(audit.OpTokenRevoke, callerTokenID, callerScope, "", err)
		return nil, nil, fmt.Errorf("failed to revoke token: %w", err)
	}

	audit.Log(&audit.Event{
		Operation:  audit.OpTokenRevoke,
		TokenID:    callerTokenID,
		TokenScope: callerScope,
		Success:    true,
		Details:    map[string]interface{}{"revoked_token_id": maskToken(params.TokenID)},
	})

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("✅ Token %s revoked successfully.", maskToken(params.TokenID))},
		},
	}, nil, nil
}

// Helper functions for token handlers

func isValidScope(scope string) bool {
	if scope == auth.ScopeAdmin || scope == auth.ScopeAdminRO {
		return true
	}
	if len(scope) > 8 && scope[:8] == "project:" {
		return true
	}
	return false
}

func maskToken(tokenID string) string {
	if len(tokenID) <= 12 {
		return "***"
	}
	return tokenID[:8] + "..." + tokenID[len(tokenID)-4:]
}
