package agent

import (
	"testing"
)

func TestStreamEventTypeConstants(t *testing.T) {
	// Verify event type constants have expected string values
	tests := []struct {
		eventType StreamEventType
		expected  string
	}{
		{StreamEventSystem, "system"},
		{StreamEventMessage, "message"},
		{StreamEventToolCall, "tool_call"},
		{StreamEventToolResult, "tool_result"},
		{StreamEventCompletion, "completion"},
		{StreamEventError, "error"},
	}

	for _, tt := range tests {
		t.Run(string(tt.eventType), func(t *testing.T) {
			if string(tt.eventType) != tt.expected {
				t.Errorf("StreamEventType = %q, want %q", tt.eventType, tt.expected)
			}
		})
	}
}

func TestStreamEvent(t *testing.T) {
	// Test StreamEvent struct creation
	event := &StreamEvent{
		Type: StreamEventMessage,
		Role: "assistant",
		Text: "Hello, world!",
	}

	if event.Type != StreamEventMessage {
		t.Errorf("event.Type = %q, want %q", event.Type, StreamEventMessage)
	}
	if event.Role != "assistant" {
		t.Errorf("event.Role = %q, want 'assistant'", event.Role)
	}
	if event.Text != "Hello, world!" {
		t.Errorf("event.Text = %q, want 'Hello, world!'", event.Text)
	}
}

func TestExecuteRequest(t *testing.T) {
	// Test ExecuteRequest struct creation
	req := &ExecuteRequest{
		ContainerID: "container-123",
		Prompt:      "Test prompt",
		WorkingDir:  "/workspace",
		Model:       "claude-sonnet-4-5",
	}

	if req.ContainerID != "container-123" {
		t.Errorf("req.ContainerID = %q, want 'container-123'", req.ContainerID)
	}
	if req.Prompt != "Test prompt" {
		t.Errorf("req.Prompt = %q, want 'Test prompt'", req.Prompt)
	}
	if req.WorkingDir != "/workspace" {
		t.Errorf("req.WorkingDir = %q, want '/workspace'", req.WorkingDir)
	}
	if req.Model != "claude-sonnet-4-5" {
		t.Errorf("req.Model = %q, want 'claude-sonnet-4-5'", req.Model)
	}
}
