package opencode

import (
	"testing"

	"github.com/HyphaGroup/oubliette/internal/agent"
)

func TestParseSSEEvent_MessageUpdated(t *testing.T) {
	// OpenCode SSE format nests data under "properties"
	data := `{"type":"message.updated","properties":{"info":{"sessionID":"ses_123","id":"msg_456","role":"assistant"}}}`

	event, err := parseSSEEvent(data)
	if err != nil {
		t.Fatalf("parseSSEEvent() returned error: %v", err)
	}

	if event.Type != agent.StreamEventMessage {
		t.Errorf("Type = %q, want %q", event.Type, agent.StreamEventMessage)
	}
	if event.SessionID != "ses_123" {
		t.Errorf("SessionID = %q, want 'ses_123'", event.SessionID)
	}
	if event.ID != "msg_456" {
		t.Errorf("ID = %q, want 'msg_456'", event.ID)
	}
	if event.Role != "assistant" {
		t.Errorf("Role = %q, want 'assistant'", event.Role)
	}
}

func TestParseSSEEvent_TextPartUpdated(t *testing.T) {
	// OpenCode SSE format nests data under "properties"
	data := `{"type":"message.part.updated","properties":{"part":{"type":"text","text":"Hello world"},"delta":"Hello"}}`

	event, err := parseSSEEvent(data)
	if err != nil {
		t.Fatalf("parseSSEEvent() returned error: %v", err)
	}

	if event.Type != agent.StreamEventMessage {
		t.Errorf("Type = %q, want %q", event.Type, agent.StreamEventMessage)
	}
	// Should use delta when available
	if event.Text != "Hello" {
		t.Errorf("Text = %q, want 'Hello'", event.Text)
	}
}

func TestParseSSEEvent_ToolInvocation(t *testing.T) {
	// OpenCode SSE format nests data under "properties"
	data := `{"type":"message.part.updated","properties":{"part":{"type":"tool-invocation","id":"tool_123","toolName":"read","args":{"path":"/test"}}}}`

	event, err := parseSSEEvent(data)
	if err != nil {
		t.Fatalf("parseSSEEvent() returned error: %v", err)
	}

	if event.Type != agent.StreamEventToolCall {
		t.Errorf("Type = %q, want %q", event.Type, agent.StreamEventToolCall)
	}
	if event.ToolID != "tool_123" {
		t.Errorf("ToolID = %q, want 'tool_123'", event.ToolID)
	}
	if event.ToolName != "read" {
		t.Errorf("ToolName = %q, want 'read'", event.ToolName)
	}
}

func TestParseSSEEvent_ToolResult(t *testing.T) {
	// OpenCode SSE format nests data under "properties"
	data := `{"type":"message.part.updated","properties":{"part":{"type":"tool-result","id":"tool_123","result":"file contents","isError":false}}}`

	event, err := parseSSEEvent(data)
	if err != nil {
		t.Fatalf("parseSSEEvent() returned error: %v", err)
	}

	if event.Type != agent.StreamEventToolResult {
		t.Errorf("Type = %q, want %q", event.Type, agent.StreamEventToolResult)
	}
	if event.ToolID != "tool_123" {
		t.Errorf("ToolID = %q, want 'tool_123'", event.ToolID)
	}
	if event.Value != "file contents" {
		t.Errorf("Value = %q, want 'file contents'", event.Value)
	}
	if event.IsError {
		t.Error("IsError should be false")
	}
}

func TestParseSSEEvent_SessionIdle(t *testing.T) {
	data := `{"type":"session.idle"}`

	event, err := parseSSEEvent(data)
	if err != nil {
		t.Fatalf("parseSSEEvent() returned error: %v", err)
	}

	if event.Type != agent.StreamEventCompletion {
		t.Errorf("Type = %q, want %q", event.Type, agent.StreamEventCompletion)
	}
}

func TestParseSSEEvent_ServerConnected(t *testing.T) {
	data := `{"type":"server.connected","properties":{}}`

	event, err := parseSSEEvent(data)
	if err != nil {
		t.Fatalf("parseSSEEvent() returned error: %v", err)
	}

	if event.Type != agent.StreamEventSystem {
		t.Errorf("Type = %q, want %q", event.Type, agent.StreamEventSystem)
	}
	if event.Subtype != "server.connected" {
		t.Errorf("Subtype = %q, want 'server.connected'", event.Subtype)
	}
}

func TestParseSSEEvent_InvalidJSON(t *testing.T) {
	data := `not json`

	_, err := parseSSEEvent(data)
	if err == nil {
		t.Error("parseSSEEvent() should return error for invalid JSON")
	}
}

func TestEventTypeConstants(t *testing.T) {
	// Verify constants match expected values
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"EventSessionCreated", EventSessionCreated, "session.created"},
		{"EventSessionIdle", EventSessionIdle, "session.idle"},
		{"EventMessageUpdated", EventMessageUpdated, "message.updated"},
		{"EventMessagePartUpdated", EventMessagePartUpdated, "message.part.updated"},
		{"EventServerConnected", EventServerConnected, "server.connected"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.constant, tt.expected)
			}
		})
	}
}
