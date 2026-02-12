package session

import (
	"testing"
	"time"
)

func TestSessionToSummary(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		session  *Session
		wantTurn int
		wantLast string
	}{
		{
			name: "session with turns",
			session: &Session{
				SessionID:   "session-1",
				ProjectID:   "project-1",
				WorkspaceID: "workspace-1",
				Status:      StatusActive,
				CreatedAt:   now,
				UpdatedAt:   now,
				Turns: []Turn{
					{TurnNumber: 1, Prompt: "First"},
					{TurnNumber: 2, Prompt: "Second"},
					{TurnNumber: 3, Prompt: "Third"},
				},
			},
			wantTurn: 3,
			wantLast: "Third",
		},
		{
			name: "session with no turns",
			session: &Session{
				SessionID:   "session-2",
				ProjectID:   "project-2",
				WorkspaceID: "workspace-2",
				Status:      StatusCompleted,
				CreatedAt:   now,
				UpdatedAt:   now,
				Turns:       []Turn{},
			},
			wantTurn: 0,
			wantLast: "",
		},
		{
			name: "session with single turn",
			session: &Session{
				SessionID:   "session-3",
				ProjectID:   "project-3",
				WorkspaceID: "workspace-3",
				Status:      StatusFailed,
				CreatedAt:   now,
				UpdatedAt:   now,
				Turns: []Turn{
					{TurnNumber: 1, Prompt: "Only prompt"},
				},
			},
			wantTurn: 1,
			wantLast: "Only prompt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := tt.session.ToSummary()

			if summary.SessionID != tt.session.SessionID {
				t.Errorf("SessionID = %q, want %q", summary.SessionID, tt.session.SessionID)
			}
			if summary.ProjectID != tt.session.ProjectID {
				t.Errorf("ProjectID = %q, want %q", summary.ProjectID, tt.session.ProjectID)
			}
			if summary.WorkspaceID != tt.session.WorkspaceID {
				t.Errorf("WorkspaceID = %q, want %q", summary.WorkspaceID, tt.session.WorkspaceID)
			}
			if summary.Status != tt.session.Status {
				t.Errorf("Status = %q, want %q", summary.Status, tt.session.Status)
			}
			if !summary.CreatedAt.Equal(tt.session.CreatedAt) {
				t.Errorf("CreatedAt = %v, want %v", summary.CreatedAt, tt.session.CreatedAt)
			}
			if !summary.UpdatedAt.Equal(tt.session.UpdatedAt) {
				t.Errorf("UpdatedAt = %v, want %v", summary.UpdatedAt, tt.session.UpdatedAt)
			}
			if summary.TurnCount != tt.wantTurn {
				t.Errorf("TurnCount = %d, want %d", summary.TurnCount, tt.wantTurn)
			}
			if summary.LastPrompt != tt.wantLast {
				t.Errorf("LastPrompt = %q, want %q", summary.LastPrompt, tt.wantLast)
			}
		})
	}
}

func TestStatusConstants(t *testing.T) {
	// Verify status constants are distinct
	statuses := []Status{StatusActive, StatusCompleted, StatusFailed}
	seen := make(map[Status]bool)

	for _, s := range statuses {
		if seen[s] {
			t.Errorf("duplicate status: %q", s)
		}
		seen[s] = true
	}

	// Verify expected string values
	if StatusActive != "active" {
		t.Errorf("StatusActive = %q, want %q", StatusActive, "active")
	}
	if StatusCompleted != "completed" {
		t.Errorf("StatusCompleted = %q, want %q", StatusCompleted, "completed")
	}
	if StatusFailed != "failed" {
		t.Errorf("StatusFailed = %q, want %q", StatusFailed, "failed")
	}
}

func TestTurnFields(t *testing.T) {
	now := time.Now()
	turn := Turn{
		TurnNumber:  1,
		Prompt:      "Test prompt",
		StartedAt:   now.Add(-time.Second),
		CompletedAt: now,
		Output: TurnOutput{
			Text:     "Test output",
			ExitCode: 0,
		},
		Cost: Cost{
			InputTokens:  100,
			OutputTokens: 200,
		},
	}

	if turn.TurnNumber != 1 {
		t.Errorf("TurnNumber = %d, want 1", turn.TurnNumber)
	}
	if turn.Prompt != "Test prompt" {
		t.Errorf("Prompt = %q, want %q", turn.Prompt, "Test prompt")
	}
	if turn.Output.Text != "Test output" {
		t.Errorf("Output.Text = %q, want %q", turn.Output.Text, "Test output")
	}
	if turn.Output.ExitCode != 0 {
		t.Errorf("Output.ExitCode = %d, want 0", turn.Output.ExitCode)
	}
	if turn.Cost.InputTokens != 100 {
		t.Errorf("Cost.InputTokens = %d, want 100", turn.Cost.InputTokens)
	}
	if turn.Cost.OutputTokens != 200 {
		t.Errorf("Cost.OutputTokens = %d, want 200", turn.Cost.OutputTokens)
	}
}

func TestTurnOutputWithError(t *testing.T) {
	output := TurnOutput{
		Text:     "",
		ExitCode: 1,
		Error:    "Something went wrong",
	}

	if output.Error != "Something went wrong" {
		t.Errorf("Error = %q, want %q", output.Error, "Something went wrong")
	}
	if output.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", output.ExitCode)
	}
}

func TestSessionWithRecursionFields(t *testing.T) {
	parentID := "parent-session"
	session := Session{
		SessionID:       "child-session",
		ProjectID:       "project-1",
		WorkspaceID:     "workspace-1",
		Status:          StatusActive,
		ParentSessionID: &parentID,
		ChildSessions:   []string{"grandchild-1", "grandchild-2"},
		Depth:           1,
		ExplorationID:   "exp_20250101_abc123",
		TaskContext: map[string]interface{}{
			"task": "implement feature X",
		},
	}

	if session.ParentSessionID == nil {
		t.Error("ParentSessionID should not be nil")
	}
	if *session.ParentSessionID != parentID {
		t.Errorf("ParentSessionID = %q, want %q", *session.ParentSessionID, parentID)
	}
	if len(session.ChildSessions) != 2 {
		t.Errorf("ChildSessions count = %d, want 2", len(session.ChildSessions))
	}
	if session.Depth != 1 {
		t.Errorf("Depth = %d, want 1", session.Depth)
	}
	if session.ExplorationID != "exp_20250101_abc123" {
		t.Errorf("ExplorationID = %q, want %q", session.ExplorationID, "exp_20250101_abc123")
	}
	if session.TaskContext["task"] != "implement feature X" {
		t.Errorf("TaskContext[task] = %v, want %q", session.TaskContext["task"], "implement feature X")
	}
}

func TestStartOptions(t *testing.T) {
	opts := StartOptions{
		Model:           "claude-opus-4-5-20251101",
		AutonomyLevel:   "high",
		ReasoningLevel:  "medium",
		WorkspaceID:     "workspace-1",
		ToolsAllowed:    []string{"read", "write"},
		ToolsDisallowed: []string{"execute"},
	}

	if opts.Model != "claude-opus-4-5-20251101" {
		t.Errorf("Model = %q, want %q", opts.Model, "claude-opus-4-5-20251101")
	}
	if opts.AutonomyLevel != "high" {
		t.Errorf("AutonomyLevel = %q, want %q", opts.AutonomyLevel, "high")
	}
	if opts.ReasoningLevel != "medium" {
		t.Errorf("ReasoningLevel = %q, want %q", opts.ReasoningLevel, "medium")
	}
	if len(opts.ToolsAllowed) != 2 {
		t.Errorf("ToolsAllowed count = %d, want 2", len(opts.ToolsAllowed))
	}
	if len(opts.ToolsDisallowed) != 1 {
		t.Errorf("ToolsDisallowed count = %d, want 1", len(opts.ToolsDisallowed))
	}
}

func TestCostAccumulation(t *testing.T) {
	session := &Session{
		SessionID: "test-session",
		TotalCost: Cost{
			InputTokens:  0,
			OutputTokens: 0,
		},
	}

	// Simulate adding turns with costs
	turns := []Cost{
		{InputTokens: 100, OutputTokens: 50},
		{InputTokens: 200, OutputTokens: 100},
		{InputTokens: 150, OutputTokens: 75},
	}

	for _, c := range turns {
		session.TotalCost.InputTokens += c.InputTokens
		session.TotalCost.OutputTokens += c.OutputTokens
	}

	if session.TotalCost.InputTokens != 450 {
		t.Errorf("TotalCost.InputTokens = %d, want 450", session.TotalCost.InputTokens)
	}
	if session.TotalCost.OutputTokens != 225 {
		t.Errorf("TotalCost.OutputTokens = %d, want 225", session.TotalCost.OutputTokens)
	}
}
