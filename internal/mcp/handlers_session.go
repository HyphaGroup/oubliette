package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/HyphaGroup/oubliette/internal/container"
	"github.com/HyphaGroup/oubliette/internal/logger"
	"github.com/HyphaGroup/oubliette/internal/project"
	"github.com/HyphaGroup/oubliette/internal/session"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

/*
SESSION MANAGEMENT AND DEPTH TRACKING

This file handles session lifecycle operations including spawning, messaging, and
event streaming. The key complexity is recursive session spawning with depth tracking.

DEPTH TRACKING ALGORITHM:

    Prime Session (depth 0)
          │
          ├── calls session_message internally
          │         │
          │         └── spawns Child Session (depth 1)
          │                   │
          │                   ├── calls session_message internally
          │                   │         │
          │                   │         └── spawns Grandchild (depth 2)
          │                   │                   ...
          │                   └── (constrained by max_depth)
          │
          └── receives results from children

    Depth is tracked as follows:
    1. Prime sessions start at depth 0 (spawned by external MCP client)
    2. Child sessions increment depth: childDepth = parentSession.Depth + 1
    3. Before spawning, check: childDepth <= project.max_depth
    4. If exceeded, return error with suggestions instead of spawning

EXPLORATION ID:

    Related sessions are grouped by explorationID for tracing:
    - Prime session generates explorationID on first child spawn
    - All descendants inherit the same explorationID
    - Enables tracing entire "exploration tree" in logs/metrics

WORKSPACE RESOLUTION:

    See resolveWorkspaceGeneric() - handles 6 cases based on:
    - workspace_id present/absent
    - create_workspace true/false
    - workspace exists/missing

    Key principle: workspace_id is NEVER inferred except for default.
    Callers must explicitly provide workspace_id or set create_workspace=true.

REVERSE SOCKET RELAY:

    Child sessions communicate with parent via oubliette-relay:
    1. Parent opens upstream connection to relay socket before spawning
    2. Child runs with oubliette-client configured in MCP config
    3. Client connects to relay socket as downstream
    4. Relay pairs upstream/downstream via FIFO queue
    5. Child has full MCP tool access through relay tunnel

    Files: cmd/oubliette-relay/, cmd/oubliette-client/
*/

// Session Management Handlers

// sessionEnv holds pre-validated environment for session operations
type sessionEnv struct {
	project       *project.Project
	workspaceID   string
	containerName string
	created       bool
}

// prepareSessionEnvironment validates and prepares the environment for session operations.
// Handles: auth check, project load, workspace resolution, container startup.
func (s *Server) prepareSessionEnvironment(ctx context.Context, projectID, workspaceID string, createWorkspace bool, externalID, source string) (*sessionEnv, error) {
	authCtx, err := requireProjectAccess(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if !authCtx.CanWrite() {
		return nil, fmt.Errorf("read-only access, cannot spawn sessions")
	}

	proj, err := s.projectMgr.Get(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to load project: %w", err)
	}

	// Resolve workspace
	resolvedWorkspaceID, created, err := s.resolveWorkspaceGeneric(projectID, proj, workspaceID, createWorkspace, externalID, source)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve workspace: %w", err)
	}

	// Update workspace timestamp
	if err := s.projectMgr.UpdateWorkspaceLastSession(projectID, resolvedWorkspaceID); err != nil {
		logger.Error("Failed to update workspace last_session_at: %v", err)
	}

	// Ensure container is running
	containerName := fmt.Sprintf("oubliette-%s", projectID[:8])
	status, err := s.runtime.Status(ctx, containerName)
	if err != nil || status != container.StatusRunning {
		logger.Info("Container not running for project %s, starting automatically", projectID)
		_, err = s.createAndStartContainer(ctx, containerName, proj.ImageName, projectID)
		if err != nil {
			return nil, fmt.Errorf("failed to auto-start container: %w", err)
		}
		logger.Info("Container started automatically for project %s", projectID)
	}

	return &sessionEnv{
		project:       proj,
		workspaceID:   resolvedWorkspaceID,
		containerName: containerName,
		created:       created,
	}, nil
}

// resolveWorkspaceGeneric handles workspace resolution for both spawn and message handlers.
func (s *Server) resolveWorkspaceGeneric(projectID string, proj *project.Project, workspaceID string, createWorkspace bool, externalID, source string) (string, bool, error) {
	if workspaceID == "" && !createWorkspace {
		return proj.DefaultWorkspaceID, false, nil
	}

	if workspaceID == "" && createWorkspace {
		metadata, err := s.projectMgr.CreateWorkspace(projectID, "", externalID, source)
		if err != nil {
			return "", false, err
		}
		return metadata.ID, true, nil
	}

	if !s.projectMgr.WorkspaceExists(projectID, workspaceID) {
		if !createWorkspace {
			return "", false, fmt.Errorf("workspace %s not found", workspaceID)
		}
		_, err := s.projectMgr.CreateWorkspace(projectID, workspaceID, externalID, source)
		if err != nil {
			return "", false, err
		}
		return workspaceID, true, nil
	}

	return workspaceID, false, nil
}

// SpawnSessionConfig holds optional configuration for spawning a session
type SpawnSessionConfig struct {
	CallerID    string
	CallerTools []session.CallerToolDefinition
}

// spawnAndRegisterSession creates a new session and registers it as active.
// IMPORTANT: All session config (http proxies, caller tools) must be passed in
// because the socket handler goroutine needs access to them immediately.
func (s *Server) spawnAndRegisterSession(ctx context.Context, projectID, containerName, workspaceID, prompt string, opts session.StartOptions, config *SpawnSessionConfig) (*session.Session, *session.ActiveSession, error) {
	sess, executor, err := s.sessionMgr.CreateBidirectionalSession(ctx, projectID, containerName, prompt, opts)
	if err != nil {
		return nil, nil, err
	}

	// Register as active session FIRST (before socket handler goroutine)
	activeSess := session.NewActiveSession(sess.SessionID, projectID, workspaceID, containerName, executor)

	// Set caller tools if provided - MUST happen before socket handler goroutine
	if config != nil && config.CallerID != "" && len(config.CallerTools) > 0 {
		activeSess.SetCallerTools(config.CallerID, config.CallerTools)
		logger.Info("Session %s configured with caller tools from %s: %d tools", sess.SessionID, config.CallerID, len(config.CallerTools))
	}

	// Register session BEFORE starting socket handler
	// (socket handler needs to find it via activeSessions.Get)
	if err := s.activeSessions.Register(activeSess); err != nil {
		_ = executor.Close()
		return nil, nil, fmt.Errorf("failed to register active session: %w", err)
	}

	// NOW connect to relay in background (after session is registered with all config)
	go func() {
		if err := s.socketHandler.ConnectSession(context.Background(), projectID, sess.SessionID, 0); err != nil {
			logger.Error("Failed to connect to relay for session %s: %v", sess.SessionID, err)
		}
		// Connection closed - mark session as completed if still active
		if activeSess, ok := s.activeSessions.Get(sess.SessionID); ok && activeSess.IsRunning() {
			logger.Info("Session %s relay connection closed, marking as completed", sess.SessionID)
			activeSess.SetStatus(session.ActiveStatusCompleted, nil)
		}
	}()

	return sess, activeSess, nil
}

// SpawnParams unifies parameters for both prime and child gogol spawning
func (s *Server) handleSpawn(ctx context.Context, request *mcp.CallToolRequest, params *SessionParams) (*mcp.CallToolResult, any, error) {
	if params.Message == "" {
		return nil, nil, fmt.Errorf("message is required")
	}

	mcpCtx := ExtractMCPContext(ctx)
	isPrime := mcpCtx.SessionID == ""

	if isPrime {
		return s.handleSpawnPrime(ctx, request, params)
	}
	return s.handleSpawnChild(ctx, mcpCtx, params)
}

func (s *Server) handleSpawnPrime(ctx context.Context, request *mcp.CallToolRequest, params *SessionParams) (*mcp.CallToolResult, any, error) {
	if params.ProjectID == "" {
		return nil, nil, fmt.Errorf("project_id is required for prime gogol")
	}

	// Check for API credentials before attempting to spawn a session
	if !s.HasAPICredentials() {
		return nil, nil, fmt.Errorf("no API credentials configured - add credentials.providers in oubliette.jsonc")
	}

	// Prepare environment using shared helper
	env, err := s.prepareSessionEnvironment(ctx, params.ProjectID, params.WorkspaceID, params.CreateWorkspace, params.ExternalID, params.Source)
	if err != nil {
		return nil, nil, SanitizeError(err, "prepare session environment")
	}

	// Use project model as default if not specified in params
	model := params.Model
	if model == "" {
		model = env.project.Model
	}

	opts := session.StartOptions{
		Model:          model,
		AutonomyLevel:  params.AutonomyLevel,
		ReasoningLevel: params.ReasoningLevel,

		WorkspaceID:        env.workspaceID,
		ToolsAllowed:       params.ToolsAllowed,
		ToolsDisallowed:    params.ToolsDisallowed,
		WorkspaceIsolation: env.project.WorkspaceIsolation,
		RuntimeOverride:    s.agentRuntime,
	}

	var sess *session.Session
	var activeSess *session.ActiveSession
	var isResume bool

	// Try to resume existing session if not forcing new
	if !params.NewSession {
		existingSession, err := s.sessionMgr.GetLatestSession(params.ProjectID)
		if err == nil && existingSession != nil && existingSession.RuntimeSessionID != "" {
			logger.Info("Resuming existing session %s for project %s", existingSession.SessionID, params.ProjectID)
			resumedSess, executor, resumeErr := s.sessionMgr.ResumeBidirectionalSession(ctx, existingSession, env.containerName, params.Message, opts)
			if resumeErr != nil {
				logger.Error("Failed to resume session, creating new: %v", resumeErr)
			} else {
				isResume = true
				sess = resumedSess

				// Reuse existing active session if available (avoids duplicate event goroutines)
				if existing, ok := s.activeSessions.Get(sess.SessionID); ok {
					activeSess = existing
					activeSess.SetExecutor(executor)
					s.activeSessions.RestartEventCollection(activeSess)
				} else {
					activeSess = session.NewActiveSession(sess.SessionID, params.ProjectID, env.workspaceID, env.containerName, executor)
					if err := s.activeSessions.Register(activeSess); err != nil {
						_ = executor.Close()
						return nil, nil, fmt.Errorf("failed to register active session: %w", err)
					}
				}

			}
		}
	}

	// Create new session if resume failed or not attempted
	if activeSess == nil {
		logger.Info("Creating new session for project %s", params.ProjectID)
		var err error
		sess, activeSess, err = s.spawnAndRegisterSession(ctx, params.ProjectID, env.containerName, env.workspaceID, params.Message, opts, nil)
		if err != nil {
			logger.Error("Failed to spawn gogol for %s: %v", params.ProjectID, err)
			return nil, nil, err
		}
	}

	// Set MCP session for SSE event push
	activeSess.SetMCPSession(request.Session)

	// Build result message
	var result string
	if isResume {
		logger.Info("Session resumed: %s", sess.SessionID)
		result = fmt.Sprintf("✅ Session resumed: %s\n\n", sess.SessionID)
		result += fmt.Sprintf("Project: %s\n", params.ProjectID)
		result += fmt.Sprintf("Workspace: %s\n", env.workspaceID)
		result += fmt.Sprintf("Runtime Session: %s\n", sess.RuntimeSessionID)
		result += fmt.Sprintf("Turns: %d\n", len(sess.Turns))
	} else {
		logger.Info("New session created: %s", sess.SessionID)
		result = fmt.Sprintf("✅ New session created: %s\n\n", sess.SessionID)
		result += fmt.Sprintf("Project: %s\n", params.ProjectID)
		result += fmt.Sprintf("Workspace: %s", env.workspaceID)
		if env.created {
			result += " (created)\n"
		} else {
			result += "\n"
		}
		result += fmt.Sprintf("Runtime Session: %s\n", sess.RuntimeSessionID)
	}
	result += fmt.Sprintf("Status: %s\n\n", activeSess.GetStatus())
	result += "Use session_events to get streaming output\n"
	result += "Use session_message to send messages\n"

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, nil, nil
}

func (s *Server) handleSpawnChild(ctx context.Context, mcpCtx MCPContext, params *SessionParams) (*mcp.CallToolResult, any, error) {
	logger.Info("Spawning child session from parent: %s", mcpCtx.SessionID)

	parentSession, err := s.sessionMgr.Load(mcpCtx.SessionID)
	if err != nil {
		logger.Error("Failed to load parent session %s: %v", mcpCtx.SessionID, err)
		return nil, nil, fmt.Errorf("failed to load parent session: %w", err)
	}

	proj, err := s.projectMgr.Get(parentSession.ProjectID)
	if err != nil {
		logger.Error("Failed to load project %s: %v", parentSession.ProjectID, err)
		return nil, nil, fmt.Errorf("failed to load project: %w", err)
	}

	childDepth := parentSession.Depth + 1
	maxDepth := s.projectMgr.GetMaxDepth(proj)

	if childDepth > maxDepth {
		errMsg := fmt.Sprintf("❌ Recursion depth limit exceeded: %d > %d\n\n", childDepth, maxDepth)
		errMsg += fmt.Sprintf("Project: %s\n", parentSession.ProjectID)
		errMsg += fmt.Sprintf("Parent session: %s (depth %d)\n\n", mcpCtx.SessionID, parentSession.Depth)
		errMsg += "Suggestion: Use direct tools (Read, Grep, Bash) instead of spawning more sessions.\n"
		errMsg += "Or increase the limit in project metadata.json with:\n"
		errMsg += `  "recursion_config": {"max_depth": 5}`

		logger.Info("Recursion limit exceeded for session %s: depth %d > max %d", mcpCtx.SessionID, childDepth, maxDepth)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: errMsg},
			},
		}, nil, nil
	}

	containerName := fmt.Sprintf("oubliette-%s", parentSession.ProjectID[:8])
	status, err := s.runtime.Status(ctx, containerName)
	if err != nil || status != container.StatusRunning {
		return nil, nil, fmt.Errorf("container for project '%s' is not running. Use container_start first", parentSession.ProjectID)
	}

	explorationID := parentSession.ExplorationID
	if explorationID == "" {
		explorationID = session.GenerateExplorationID()
		parentSession.ExplorationID = explorationID
		if err := s.sessionMgr.SaveSession(parentSession); err != nil {
			logger.Error("Failed to update parent session with exploration ID: %v", err)
			return nil, nil, fmt.Errorf("failed to update parent session: %w", err)
		}
	}

	// Build the full prompt with context preamble inlined
	prompt := fmt.Sprintf(`You are a child session at depth %d/%d.
Exploration ID: %s
Parent session: %s

When complete, write results to: /workspace/.rlm-context/{{SESSION_ID}}_results.json
The .rlm-context/ directory is shared with your parent and siblings for result aggregation.

---

%s`,
		childDepth, maxDepth,
		explorationID,
		mcpCtx.SessionID,
		params.Message,
	)

	// Use project model as default if not specified in params
	model := params.Model
	if model == "" {
		model = proj.Model
	}

	opts := session.StartOptions{
		Model:              model,
		AutonomyLevel:      params.AutonomyLevel,
		ReasoningLevel:     params.ReasoningLevel,
		ToolsAllowed:       params.ToolsAllowed,
		ToolsDisallowed:    params.ToolsDisallowed,
		WorkspaceID:        parentSession.WorkspaceID,
		WorkspaceIsolation: proj.WorkspaceIsolation,
	}

	childSession, err := s.sessionMgr.Create(ctx, parentSession.ProjectID, containerName, prompt, opts)
	if err != nil {
		logger.Error("Failed to create child session: %v", err)
		return nil, nil, fmt.Errorf("failed to create child session: %w", err)
	}

	childSession.ParentSessionID = &mcpCtx.SessionID
	childSession.Depth = childDepth
	childSession.ExplorationID = explorationID
	childSession.TaskContext = params.Context
	childSession.ToolsAllowed = params.ToolsAllowed

	workspaceDir := s.projectMgr.GetWorkspacePath(parentSession.ProjectID, parentSession.WorkspaceID)
	rlmContextDir := filepath.Join(workspaceDir, ".rlm-context")
	if err := os.MkdirAll(rlmContextDir, 0o755); err != nil {
		logger.Error("Failed to create .rlm-context directory: %v", err)
		return nil, nil, fmt.Errorf("failed to create .rlm-context directory: %w", err)
	}

	if err := s.sessionMgr.SaveSession(childSession); err != nil {
		logger.Error("Failed to save child session metadata: %v", err)
		return nil, nil, fmt.Errorf("failed to save child session: %w", err)
	}

	if err := s.sessionMgr.AddChildSession(mcpCtx.SessionID, childSession.SessionID); err != nil {
		logger.Error("Failed to add child to parent session: %v", err)
		return nil, nil, fmt.Errorf("failed to add child to parent session: %w", err)
	}

	logger.Info("Child gogol spawned successfully: %s (depth %d/%d)", childSession.SessionID, childDepth, maxDepth)

	result := fmt.Sprintf("✅ Child gogol spawned: %s\n\n", childSession.SessionID)
	result += fmt.Sprintf("Depth: %d/%d\n", childDepth, maxDepth)
	result += fmt.Sprintf("Parent: %s\n", mcpCtx.SessionID)
	result += fmt.Sprintf("Exploration: %s\n", explorationID)
	result += fmt.Sprintf("Project: %s\n\n", parentSession.ProjectID)

	if len(childSession.Turns) > 0 {
		lastTurn := childSession.Turns[len(childSession.Turns)-1]
		result += fmt.Sprintf("Output:\n%s\n\n", lastTurn.Output.Text)
		result += fmt.Sprintf("Cost: %d input tokens, %d output tokens\n", lastTurn.Cost.InputTokens, lastTurn.Cost.OutputTokens)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: result},
		},
	}, nil, nil
}

func (s *Server) handleGetSession(ctx context.Context, request *mcp.CallToolRequest, params *SessionParams) (*mcp.CallToolResult, any, error) {
	if params.SessionID == "" {
		return nil, nil, fmt.Errorf("session_id is required")
	}

	sess, err := s.sessionMgr.Load(params.SessionID)
	if err != nil {
		return nil, nil, err
	}

	result := fmt.Sprintf("Session: %s\n\n", sess.SessionID)
	result += fmt.Sprintf("Project: %s\n", sess.ProjectID)
	result += fmt.Sprintf("Status: %s\n", sess.Status)
	result += fmt.Sprintf("Created: %s\n", sess.CreatedAt.Format("2006-01-02 15:04:05"))
	result += fmt.Sprintf("Updated: %s\n", sess.UpdatedAt.Format("2006-01-02 15:04:05"))
	result += fmt.Sprintf("Turns: %d\n", len(sess.Turns))
	result += fmt.Sprintf("Total Cost: %d input tokens, %d output tokens\n\n", sess.TotalCost.InputTokens, sess.TotalCost.OutputTokens)

	if len(sess.Turns) > 0 {
		lastTurn := sess.Turns[len(sess.Turns)-1]
		result += "Last Turn:\n"
		result += fmt.Sprintf("  Prompt: %s\n", lastTurn.Prompt)
		result += fmt.Sprintf("  Output: %s\n", lastTurn.Output.Text[:minInt(200, len(lastTurn.Output.Text))])
		if len(lastTurn.Output.Text) > 200 {
			result += "  ...(truncated)\n"
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: result},
		},
	}, nil, nil
}

func (s *Server) handleListSessions(ctx context.Context, request *mcp.CallToolRequest, params *SessionParams) (*mcp.CallToolResult, any, error) {
	if params.ProjectID == "" {
		return nil, nil, fmt.Errorf("project_id is required")
	}

	var statusFilter *session.Status
	if params.Status != "" {
		status := session.Status(params.Status)
		statusFilter = &status
	}

	sessions, err := s.sessionMgr.List(params.ProjectID, statusFilter)
	if err != nil {
		return nil, nil, err
	}

	if len(sessions) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("No sessions found for project '%s'", params.ProjectID)},
			},
		}, nil, nil
	}

	result := fmt.Sprintf("Found %d session(s) for project '%s':\n\n", len(sessions), params.ProjectID)
	for _, sess := range sessions {
		result += fmt.Sprintf("• %s\n", sess.SessionID)
		result += fmt.Sprintf("  Status: %s\n", sess.Status)
		result += fmt.Sprintf("  Turns: %d\n", sess.TurnCount)
		result += fmt.Sprintf("  Created: %s\n", sess.CreatedAt.Format("2006-01-02 15:04:05"))
		if sess.LastPrompt != "" {
			result += fmt.Sprintf("  Last: %s\n", sess.LastPrompt[:minInt(80, len(sess.LastPrompt))])
		}
		result += "\n"
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: result},
		},
	}, nil, nil
}

func (s *Server) handleEndSession(ctx context.Context, request *mcp.CallToolRequest, params *SessionParams) (*mcp.CallToolResult, any, error) {
	if params.SessionID == "" {
		return nil, nil, fmt.Errorf("session_id is required")
	}

	logger.Info("Ending session: %s", params.SessionID)

	if err := s.sessionMgr.End(params.SessionID); err != nil {
		logger.Error("Failed to end session %s: %v", params.SessionID, err)
		return nil, nil, err
	}

	logger.Info("Session ended successfully: %s", params.SessionID)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Session '%s' ended successfully", params.SessionID)},
		},
	}, nil, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// SendMessageParams for the unified session_message tool
// Attachment represents a file attachment sent with a message
type Attachment struct {
	ID          string `json:"id,omitempty"`
	Filename    string `json:"filename,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	Size        int64  `json:"size,omitempty"`
	Data        string `json:"data,omitempty"` // Base64-encoded content
	URL         string `json:"url,omitempty"`  // URL to fetch content
}

type SendMessageResult struct {
	SessionID        string `json:"session_id"`
	Spawned          bool   `json:"spawned"`
	WorkspaceCreated bool   `json:"workspace_created"`
	LastEventIndex   int    `json:"last_event_index"`
}

func (s *Server) handleSendMessage(ctx context.Context, request *mcp.CallToolRequest, params *SessionParams) (*mcp.CallToolResult, any, error) {
	if params.ProjectID == "" {
		return nil, nil, fmt.Errorf("project_id is required")
	}
	if params.Message == "" {
		return nil, nil, fmt.Errorf("message is required")
	}

	message := params.Message

	// Check auth and resolve workspace early so we can look up active sessions
	authCtx, err := requireProjectAccess(ctx, params.ProjectID)
	if err != nil {
		return nil, nil, err
	}
	if !authCtx.CanWrite() {
		return nil, nil, fmt.Errorf("read-only access, cannot send messages")
	}

	// Resolve workspace ID (empty = use default workspace)
	workspaceID := params.WorkspaceID
	if workspaceID == "" {
		proj, err := s.projectMgr.Get(params.ProjectID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load project: %w", err)
		}
		workspaceID = proj.DefaultWorkspaceID
	}

	logger.Info("Session message for project %s, workspace %s", params.ProjectID, workspaceID)

	// Fast path: send to existing active session
	activeSess, found := s.activeSessions.GetByWorkspace(params.ProjectID, workspaceID)
	if found {
		if !activeSess.IsRunning() {
			// Session exists but is no longer running (completed/failed/timed out)
			// Remove it so we can spawn a new one
			logger.Info("Removing non-running session %s (status: %s) for workspace %s", activeSess.SessionID, activeSess.GetStatus(), workspaceID)
			s.activeSessions.Remove(activeSess.SessionID)
			found = false
		} else {
			logger.Info("Found active session %s for workspace %s", activeSess.SessionID, workspaceID)
		}
	}
	if found {

		// Update MCP session for SSE event push (may be a reconnecting client)
		activeSess.SetMCPSession(request.Session)

		// Update caller tools if provided (allows updating tools on existing session)
		if params.CallerID != "" && len(params.CallerTools) > 0 {
			activeSess.SetCallerTools(params.CallerID, params.CallerTools)
			logger.Info("Session %s configured with caller tools from %s: %d tools", activeSess.SessionID, params.CallerID, len(params.CallerTools))
		}

		if err := s.activeSessions.SendMessage(activeSess.SessionID, message); err != nil {
			return nil, nil, fmt.Errorf("failed to send message: %w", err)
		}

		result := SendMessageResult{
			SessionID:        activeSess.SessionID,
			Spawned:          false,
			WorkspaceCreated: false,
			LastEventIndex:   activeSess.EventBuffer.LastIndex(),
		}
		resultJSON, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
		}, result, nil
	}

	// Slow path: prepare environment and spawn new session
	logger.Info("No active session for workspace %s, spawning new session", workspaceID)

	env, err := s.prepareSessionEnvironment(ctx, params.ProjectID, workspaceID, params.CreateWorkspace, params.ExternalID, params.Source)
	if err != nil {
		return nil, nil, err
	}

	// Use project model as default if not specified in params
	model := params.Model
	if model == "" {
		model = env.project.Model
	}

	opts := session.StartOptions{
		Model:              model,
		AutonomyLevel:      params.AutonomyLevel,
		ReasoningLevel:     params.ReasoningLevel,
		WorkspaceID:        env.workspaceID,
		ToolsAllowed:       params.ToolsAllowed,
		ToolsDisallowed:    params.ToolsDisallowed,
		WorkspaceIsolation: env.project.WorkspaceIsolation,
		RuntimeOverride:    s.agentRuntime,
	}

	// Build spawn config with all session configuration
	// IMPORTANT: Caller tools MUST be passed here so they're set BEFORE the socket handler goroutine
	spawnConfig := &SpawnSessionConfig{
		CallerID:    params.CallerID,
		CallerTools: params.CallerTools,
	}
	sess, activeSess, err := s.spawnAndRegisterSession(ctx, params.ProjectID, env.containerName, env.workspaceID, message, opts, spawnConfig)
	if err != nil {
		return nil, nil, SanitizeError(err, "spawn session")
	}

	// Set MCP session for SSE event push
	activeSess.SetMCPSession(request.Session)

	result := SendMessageResult{
		SessionID:        sess.SessionID,
		Spawned:          true,
		WorkspaceCreated: env.created,
		LastEventIndex:   -1,
	}
	resultJSON, _ := json.Marshal(result)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
	}, result, nil
}

// SessionEventsParams for streaming events
type SessionEventsResult struct {
	SessionID     string             `json:"session_id"`
	Status        string             `json:"status"`
	LastIndex     int                `json:"last_index"`
	Events        []SessionEventItem `json:"events"`
	Completed     bool               `json:"completed"`
	Failed        bool               `json:"failed"`
	Error         string             `json:"error,omitempty"`
	DroppedEvents int64              `json:"dropped_events"`
}

type SessionEventItem struct {
	Index     int    `json:"index"`
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`
	ToolName  string `json:"tool_name,omitempty"`
	Role      string `json:"role,omitempty"`
	SessionID string `json:"session_id,omitempty"` // Set when include_children is true
}

func (s *Server) handleSessionEvents(ctx context.Context, request *mcp.CallToolRequest, params *SessionParams) (*mcp.CallToolResult, any, error) {
	if params.SessionID == "" {
		return nil, nil, fmt.Errorf("session_id is required")
	}

	activeSess, ok := s.activeSessions.Get(params.SessionID)
	if !ok {
		return nil, nil, fmt.Errorf("session %s is not an active streaming session", params.SessionID)
	}

	sinceIndex := -1
	if params.SinceIndex != nil {
		sinceIndex = *params.SinceIndex
	}

	events, err := s.activeSessions.GetEvents(params.SessionID, sinceIndex)
	if err != nil {
		return nil, nil, err
	}

	// Collect events from child sessions if requested
	var allEvents []SessionEventItem
	if params.IncludeChildren {
		// Get child session IDs from the persisted session
		sess, err := s.sessionMgr.Load(params.SessionID)
		if err == nil && len(sess.ChildSessions) > 0 {
			// Collect events from parent first, with session_id populated
			for _, e := range events {
				allEvents = append(allEvents, SessionEventItem{
					Index:     e.Index,
					Type:      string(e.Event.Type),
					Text:      e.Event.Text,
					ToolName:  e.Event.ToolName,
					Role:      e.Event.Role,
					SessionID: params.SessionID,
				})
			}

			// Collect events from each child session
			for _, childID := range sess.ChildSessions {
				_, ok := s.activeSessions.Get(childID)
				if !ok {
					continue // Child not active, skip
				}
				childEvents, err := s.activeSessions.GetEvents(childID, sinceIndex)
				if err != nil {
					continue
				}
				for _, e := range childEvents {
					allEvents = append(allEvents, SessionEventItem{
						Index:     e.Index,
						Type:      string(e.Event.Type),
						Text:      e.Event.Text,
						ToolName:  e.Event.ToolName,
						Role:      e.Event.Role,
						SessionID: childID,
					})
				}
			}

			// Note: Events are not sorted by timestamp since event buffers
			// don't have timestamps. If needed, could add timestamp to IndexedEvent.
		}
	}

	// Apply max_events limit
	if params.MaxEvents != nil && *params.MaxEvents > 0 {
		if params.IncludeChildren && len(allEvents) > *params.MaxEvents {
			allEvents = allEvents[:*params.MaxEvents]
		} else if !params.IncludeChildren && len(events) > *params.MaxEvents {
			events = events[:*params.MaxEvents]
		}
	}

	status := activeSess.GetStatus()
	bufferStats := activeSess.EventBuffer.Stats()

	structuredResult := SessionEventsResult{
		SessionID:     params.SessionID,
		Status:        string(status),
		LastIndex:     bufferStats.LastIndex,
		Completed:     status == session.ActiveStatusCompleted,
		Failed:        status == session.ActiveStatusFailed,
		DroppedEvents: bufferStats.DroppedEvents,
	}

	if activeSess.Error != nil {
		structuredResult.Error = activeSess.Error.Error()
	}

	// Use pre-built allEvents if include_children, otherwise build from events
	if params.IncludeChildren && len(allEvents) > 0 {
		structuredResult.Events = allEvents
	} else {
		structuredResult.Events = make([]SessionEventItem, len(events))
		for i, e := range events {
			structuredResult.Events[i] = SessionEventItem{
				Index:    e.Index,
				Type:     string(e.Event.Type),
				Text:     e.Event.Text,
				ToolName: e.Event.ToolName,
				Role:     e.Event.Role,
			}
		}
	}

	resultJSON, _ := json.Marshal(structuredResult)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(resultJSON)},
		},
	}, structuredResult, nil
}

// GetRecursionLimitsParams for recursion configuration
type GetRecursionLimitsParams struct {
	ProjectID string `json:"project_id"`
	SessionID string `json:"session_id,omitempty"`
}

func (s *Server) handleGetRecursionLimits(ctx context.Context, request *mcp.CallToolRequest, params *GetRecursionLimitsParams) (*mcp.CallToolResult, any, error) {
	if params.ProjectID == "" {
		mcpCtx := ExtractMCPContext(ctx)
		if mcpCtx.ProjectID != "" {
			params.ProjectID = mcpCtx.ProjectID
		} else {
			return nil, nil, fmt.Errorf("project_id is required")
		}
	}

	proj, err := s.projectMgr.Get(params.ProjectID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load project: %w", err)
	}

	maxDepth := s.projectMgr.GetMaxDepth(proj)
	maxAgents := s.projectMgr.GetMaxAgents(proj)
	maxCostUSD := s.projectMgr.GetMaxCostUSD(proj)

	depthSource := "environment default"
	if proj.RecursionConfig != nil && proj.RecursionConfig.MaxDepth != nil {
		depthSource = "project override"
	}

	agentsSource := "environment default"
	if proj.RecursionConfig != nil && proj.RecursionConfig.MaxAgents != nil {
		agentsSource = "project override"
	}

	costSource := "environment default"
	if proj.RecursionConfig != nil && proj.RecursionConfig.MaxCostUSD != nil {
		costSource = "project override"
	}

	result := fmt.Sprintf("Recursion Limits for Project: %s\n\n", params.ProjectID)
	result += fmt.Sprintf("Max Depth: %d (%s)\n", maxDepth, depthSource)
	result += fmt.Sprintf("Max Agents: %d (%s)\n", maxAgents, agentsSource)
	result += fmt.Sprintf("Max Cost: $%.2f (%s)\n\n", maxCostUSD, costSource)

	sessionID := params.SessionID
	if sessionID == "" {
		mcpCtx := ExtractMCPContext(ctx)
		sessionID = mcpCtx.SessionID
	}

	if sessionID != "" {
		sess, err := s.sessionMgr.Load(sessionID)
		if err == nil {
			remaining := maxDepth - sess.Depth
			result += fmt.Sprintf("Current Session: %s\n", sessionID)
			result += fmt.Sprintf("Current Depth: %d/%d\n", sess.Depth, maxDepth)
			result += fmt.Sprintf("Remaining Depth: %d\n", remaining)
			if sess.ExplorationID != "" {
				result += fmt.Sprintf("Exploration ID: %s\n", sess.ExplorationID)
			}
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: result},
		},
	}, nil, nil
}

// CallerToolResponseParams for the caller_tool_response tool
type CallerToolResponseParams struct {
	SessionID string `json:"session_id"`
	RequestID string `json:"request_id"`
	Result    any    `json:"result,omitempty"`
	Error     string `json:"error,omitempty"`
}

func (s *Server) handleCallerToolResponse(ctx context.Context, request *mcp.CallToolRequest, params *CallerToolResponseParams) (*mcp.CallToolResult, any, error) {
	if params.SessionID == "" {
		return nil, nil, fmt.Errorf("session_id is required")
	}
	if params.RequestID == "" {
		return nil, nil, fmt.Errorf("request_id is required")
	}

	// Look up the session
	activeSess, ok := s.activeSessions.Get(params.SessionID)
	if !ok {
		return nil, nil, fmt.Errorf("session %s not found or not active", params.SessionID)
	}

	// Build response
	response := &session.CallerToolResponse{
		Result: params.Result,
		Error:  params.Error,
	}

	// Resolve the pending request
	if !activeSess.ResolveCallerRequest(params.RequestID, response) {
		return nil, nil, fmt.Errorf("request %s not found or already resolved", params.RequestID)
	}

	logger.Info("Resolved caller_tool_response for session %s, request %s", params.SessionID, params.RequestID)

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "OK"}},
	}, nil, nil
}

// SessionCleanupParams for the session_cleanup tool
func (s *Server) handleSessionCleanup(ctx context.Context, request *mcp.CallToolRequest, params *SessionParams) (*mcp.CallToolResult, any, error) {
	// Require write access
	authCtx, err := requireAuth(ctx)
	if err != nil {
		return nil, nil, err
	}
	if !authCtx.CanWrite() {
		return nil, nil, fmt.Errorf("read-only access, cannot cleanup sessions")
	}

	// Default to 24 hours
	maxAgeHours := 24
	if params.MaxAgeHours != nil && *params.MaxAgeHours > 0 {
		maxAgeHours = *params.MaxAgeHours
	}
	maxAge := time.Duration(maxAgeHours) * time.Hour

	var result string
	var totalDeleted int

	if params.ProjectID != "" {
		// Clean up specific project
		deleted, err := s.sessionMgr.CleanupOldSessions(params.ProjectID, maxAge)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to cleanup sessions: %w", err)
		}
		totalDeleted = deleted
		result = fmt.Sprintf("Cleaned up %d session(s) older than %d hours from project '%s'", deleted, maxAgeHours, params.ProjectID)
	} else {
		// Clean up all projects
		results, err := s.sessionMgr.CleanupAllOldSessions(maxAge)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to cleanup sessions: %w", err)
		}

		for _, count := range results {
			totalDeleted += count
		}

		if len(results) == 0 {
			result = fmt.Sprintf("No sessions older than %d hours found across all projects", maxAgeHours)
		} else {
			result = fmt.Sprintf("Cleaned up %d session(s) older than %d hours across %d project(s):\n\n", totalDeleted, maxAgeHours, len(results))
			for projectID, count := range results {
				result += fmt.Sprintf("  • %s: %d session(s)\n", projectID, count)
			}
		}
	}

	logger.Info("Session cleanup completed: %d sessions deleted (max_age=%dh)", totalDeleted, maxAgeHours)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: result},
		},
	}, map[string]interface{}{"deleted": totalDeleted}, nil
}
