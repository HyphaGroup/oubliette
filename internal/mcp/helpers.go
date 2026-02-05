package mcp

import (
	mcp_sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewTextResult creates a CallToolResult with text content
func NewTextResult(text string) *mcp_sdk.CallToolResult {
	return &mcp_sdk.CallToolResult{
		Content: []mcp_sdk.Content{
			&mcp_sdk.TextContent{Text: text},
		},
	}
}

// NewErrorResult creates a CallToolResult indicating an error
func NewErrorResult(msg string) *mcp_sdk.CallToolResult {
	return &mcp_sdk.CallToolResult{
		IsError: true,
		Content: []mcp_sdk.Content{
			&mcp_sdk.TextContent{Text: msg},
		},
	}
}
