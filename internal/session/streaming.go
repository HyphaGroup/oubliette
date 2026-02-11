package session

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/HyphaGroup/oubliette/internal/agent"
)

/*
STREAMING SESSION ARCHITECTURE

This file implements bidirectional streaming sessions between MCP clients and OpenCode.

Flow Overview:
┌─────────────┐     ┌─────────────┐     ┌─────────────────┐     ┌──────────┐
│ MCP Client  │────>│ MCP Server  │────>│ StreamingExecutor│────>│ OpenCode │
│             │<────│ (handlers)  │<────│ (HTTP+SSE)       │<────│ Server   │
└─────────────┘     └─────────────┘     └─────────────────┘     └──────────┘

Key Design Decisions:

1. EXECUTOR OWNERSHIP: The StreamingExecutor owns the HTTP/SSE connection.
   - Executor manages HTTP requests and SSE event stream
   - Events are read from SSE via processEvents() goroutine
   - Messages sent via SendMessage() post to prompt_async endpoint

2. CONTEXT PROPAGATION: The provided ctx is passed through to the executor.
   - Canceling ctx will cancel the execution
   - This enables proper cleanup when sessions are terminated externally

3. SESSION RESUMPTION: Sessions can be resumed using the runtime session ID.
   - RuntimeSessionID is captured from session creation and stored in metadata
   - ResumeBidirectionalSession() passes this ID to resume conversation

4. ERROR HANDLING: Errors during setup cause immediate cleanup.
   - If executor fails to start, session is marked failed
   - If metadata save fails, executor is closed to prevent orphan processes

5. WORKSPACE ISOLATION: Each session runs in its own workspace directory.
   - workingDir is /workspace/workspaces/<uuid>
   - This ensures sessions don't interfere with each other
*/

// CreateBidirectionalSession creates a new session and returns both session and executor
// for registration with ActiveSessionManager.
func (m *Manager) CreateBidirectionalSession(ctx context.Context, projectID, containerID, prompt string, opts StartOptions) (*Session, agent.StreamingExecutor, error) {
	sessionID := generateSessionID()

	// Workspace ID is required - handlers must resolve it before calling
	workspaceID := opts.WorkspaceID
	if workspaceID == "" {
		return nil, nil, fmt.Errorf("workspace_id is required - caller must resolve workspace before creating session")
	}

	// Working directory depends on workspace isolation setting
	var workingDir string
	if opts.WorkspaceIsolation {
		// Isolated mode: /workspace is mounted to workspaces/, so workingDir is /workspace/<uuid>
		workingDir = fmt.Sprintf("/workspace/%s", workspaceID)
	} else {
		// Non-isolated mode: /workspace is mounted to project root
		workingDir = fmt.Sprintf("/workspace/workspaces/%s", workspaceID)
	}

	session := &Session{
		SessionID:      sessionID,
		ProjectID:      projectID,
		WorkspaceID:    workspaceID,
		ContainerID:    containerID,
		Status:         StatusActive,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Turns:          []Turn{},
		TotalCost:      Cost{},
		Model:          opts.Model,
		AutonomyLevel:  opts.AutonomyLevel,
		ReasoningLevel: opts.ReasoningLevel,
	}

	// Create agent request (new session, no -s flag)
	req := &agent.ExecuteRequest{
		Prompt:         prompt,
		ContainerID:    containerID,
		WorkingDir:     workingDir,
		ProjectID:      projectID,
		Depth:          0,
		Model:          opts.Model,
		AutonomyLevel:  opts.AutonomyLevel,
		ReasoningLevel: opts.ReasoningLevel,
		EnabledTools:   opts.ToolsAllowed,
		DisabledTools:  opts.ToolsDisallowed,
		SystemPrompt:   opts.AppendSystemPrompt,
		StreamJSONRPC:  true,
	}

	// Determine which runtime to use (override or manager's default)
	runtime := m.agentRuntime
	if opts.RuntimeOverride != nil {
		if rt, ok := opts.RuntimeOverride.(agent.Runtime); ok {
			runtime = rt
		}
	}

	// Start bidirectional streaming - use background context so executor survives request completion
	// The executor has its own lifecycle managed by ActiveSessionManager
	executor, err := runtime.ExecuteStreaming(context.Background(), req)
	if err != nil {
		session.Status = StatusFailed
		return nil, nil, fmt.Errorf("failed to start streaming session: %w", err)
	}

	// Capture runtime's session ID (available after init)
	runtimeSessionID := executor.RuntimeSessionID()
	if runtimeSessionID != "" {
		session.RuntimeSessionID = runtimeSessionID
	} else {
		session.RuntimeSessionID = sessionID
	}

	// Record the turn
	turn := Turn{
		TurnNumber: 1,
		Prompt:     prompt,
		StartedAt:  time.Now(),
		Output: TurnOutput{
			Text:     "Streaming session active",
			ExitCode: 0,
		},
	}
	session.Turns = append(session.Turns, turn)

	// Save session metadata
	sessionsDir := fmt.Sprintf("%s/%s/sessions", m.sessionsBaseDir, projectID)
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		_ = executor.Close()
		return nil, nil, fmt.Errorf("failed to create sessions directory: %w", err)
	}

	if err := m.saveSession(session); err != nil {
		_ = executor.Close()
		return nil, nil, err
	}

	return session, executor, nil
}

// ResumeBidirectionalSession resumes an existing session using agent's session ID
func (m *Manager) ResumeBidirectionalSession(ctx context.Context, existingSession *Session, containerID, prompt string, opts StartOptions) (*Session, agent.StreamingExecutor, error) {
	// Workspace ID is required - handlers must resolve it before calling
	workspaceID := opts.WorkspaceID
	if workspaceID == "" {
		return nil, nil, fmt.Errorf("workspace_id is required - caller must resolve workspace before resuming session")
	}

	// Working directory depends on workspace isolation setting
	var workingDir string
	if opts.WorkspaceIsolation {
		// Isolated mode: /workspace is mounted to workspaces/, so workingDir is /workspace/<uuid>
		workingDir = fmt.Sprintf("/workspace/%s", workspaceID)
	} else {
		// Non-isolated mode: /workspace is mounted to project root
		workingDir = fmt.Sprintf("/workspace/workspaces/%s", workspaceID)
	}

	// Create agent request with session ID for resumption
	req := &agent.ExecuteRequest{
		Prompt:         prompt,
		ContainerID:    containerID,
		WorkingDir:     workingDir,
		SessionID:      existingSession.RuntimeSessionID, // Resume this session
		ProjectID:      existingSession.ProjectID,
		Depth:          existingSession.Depth,
		Model:          opts.Model,
		AutonomyLevel:  opts.AutonomyLevel,
		ReasoningLevel: opts.ReasoningLevel,
		EnabledTools:   opts.ToolsAllowed,
		DisabledTools:  opts.ToolsDisallowed,
		SystemPrompt:   opts.AppendSystemPrompt,
		StreamJSONRPC:  true,
	}

	// Determine which runtime to use (override or manager's default)
	runtime := m.agentRuntime
	if opts.RuntimeOverride != nil {
		if rt, ok := opts.RuntimeOverride.(agent.Runtime); ok {
			runtime = rt
		}
	}

	// Start bidirectional streaming with session resumption - use background context
	// so executor survives request completion
	executor, err := runtime.ExecuteStreaming(context.Background(), req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resume streaming session: %w", err)
	}

	// Update session metadata
	existingSession.Status = StatusActive
	existingSession.UpdatedAt = time.Now()
	existingSession.ContainerID = containerID

	// Record the new turn
	turn := Turn{
		TurnNumber: len(existingSession.Turns) + 1,
		Prompt:     prompt,
		StartedAt:  time.Now(),
		Output: TurnOutput{
			Text:     "Session resumed",
			ExitCode: 0,
		},
	}
	existingSession.Turns = append(existingSession.Turns, turn)

	if err := m.saveSession(existingSession); err != nil {
		_ = executor.Close()
		return nil, nil, err
	}

	return existingSession, executor, nil
}

// GetLatestSession returns the most recent session for a project, if any
func (m *Manager) GetLatestSession(projectID string) (*Session, error) {
	sessions, err := m.List(projectID, nil)
	if err != nil {
		return nil, err
	}

	if len(sessions) == 0 {
		return nil, nil
	}

	// Find the most recent session
	var latest *SessionSummary
	for _, s := range sessions {
		if latest == nil || s.UpdatedAt.After(latest.UpdatedAt) {
			latest = s
		}
	}

	if latest == nil {
		return nil, nil
	}

	return m.Load(latest.SessionID)
}
