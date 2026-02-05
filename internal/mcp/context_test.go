package mcp

import (
	"context"
	"testing"
)

func TestExtractMCPContext_Empty(t *testing.T) {
	ctx := context.Background()
	mcpCtx := ExtractMCPContext(ctx)

	if mcpCtx.SessionID != "" {
		t.Errorf("SessionID = %q, want empty", mcpCtx.SessionID)
	}
	if mcpCtx.ProjectID != "" {
		t.Errorf("ProjectID = %q, want empty", mcpCtx.ProjectID)
	}
	if mcpCtx.Depth != 0 {
		t.Errorf("Depth = %d, want 0", mcpCtx.Depth)
	}
}

func TestWithMCPHeaders(t *testing.T) {
	ctx := context.Background()
	ctx = WithMCPHeaders(ctx, "session-123", "project-456", 3)

	mcpCtx := ExtractMCPContext(ctx)

	if mcpCtx.SessionID != "session-123" {
		t.Errorf("SessionID = %q, want %q", mcpCtx.SessionID, "session-123")
	}
	if mcpCtx.ProjectID != "project-456" {
		t.Errorf("ProjectID = %q, want %q", mcpCtx.ProjectID, "project-456")
	}
	if mcpCtx.Depth != 3 {
		t.Errorf("Depth = %d, want 3", mcpCtx.Depth)
	}
}

func TestGenerateMCPHeaders(t *testing.T) {
	headers := GenerateMCPHeaders("sess-001", "proj-001", 5)

	if headers["X-Oubliette-Session-ID"] != "sess-001" {
		t.Errorf("X-Oubliette-Session-ID = %q, want %q", headers["X-Oubliette-Session-ID"], "sess-001")
	}
	if headers["X-Oubliette-Project"] != "proj-001" {
		t.Errorf("X-Oubliette-Project = %q, want %q", headers["X-Oubliette-Project"], "proj-001")
	}
	if headers["X-Oubliette-Depth"] != "5" {
		t.Errorf("X-Oubliette-Depth = %q, want %q", headers["X-Oubliette-Depth"], "5")
	}
}

func TestGenerateMCPHeaders_Zero(t *testing.T) {
	headers := GenerateMCPHeaders("", "", 0)

	if headers["X-Oubliette-Session-ID"] != "" {
		t.Errorf("X-Oubliette-Session-ID = %q, want empty", headers["X-Oubliette-Session-ID"])
	}
	if headers["X-Oubliette-Depth"] != "0" {
		t.Errorf("X-Oubliette-Depth = %q, want %q", headers["X-Oubliette-Depth"], "0")
	}
}

func TestMCPContext_Struct(t *testing.T) {
	mcpCtx := MCPContext{
		SessionID: "test-session",
		ProjectID: "test-project",
		Depth:     2,
	}

	if mcpCtx.SessionID != "test-session" {
		t.Errorf("SessionID = %q, want %q", mcpCtx.SessionID, "test-session")
	}
	if mcpCtx.ProjectID != "test-project" {
		t.Errorf("ProjectID = %q, want %q", mcpCtx.ProjectID, "test-project")
	}
	if mcpCtx.Depth != 2 {
		t.Errorf("Depth = %d, want 2", mcpCtx.Depth)
	}
}

func TestGetStringFromContext(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  string
	}{
		{"string value", "test-value", "test-value"},
		{"empty string", "", ""},
		{"nil value", nil, ""},
		{"int value", 123, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.value != nil {
				ctx = context.WithValue(ctx, contextKeySessionID, tt.value)
			}

			got := getStringFromContext(ctx, contextKeySessionID)
			if got != tt.want {
				t.Errorf("getStringFromContext() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetIntFromContext(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  int
	}{
		{"string number", "42", 42},
		{"int value", 100, 100},
		{"zero string", "0", 0},
		{"empty string", "", 0},
		{"invalid string", "abc", 0},
		{"nil value", nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.value != nil {
				ctx = context.WithValue(ctx, contextKeyDepth, tt.value)
			}

			got := getIntFromContext(ctx, contextKeyDepth)
			if got != tt.want {
				t.Errorf("getIntFromContext() = %d, want %d", got, tt.want)
			}
		})
	}
}
