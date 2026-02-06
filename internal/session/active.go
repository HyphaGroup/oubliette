package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/HyphaGroup/oubliette/internal/agent"
	"github.com/HyphaGroup/oubliette/internal/logger"
	"github.com/HyphaGroup/oubliette/internal/metrics"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ActiveStatus represents the status of an active streaming session
type ActiveStatus string

const (
	ActiveStatusIdle      ActiveStatus = "idle"      // Waiting for a message
	ActiveStatusRunning   ActiveStatus = "running"   // Actively processing
	ActiveStatusPaused    ActiveStatus = "paused"    // Paused (not currently used)
	ActiveStatusCompleted ActiveStatus = "completed" // Session process exited
	ActiveStatusFailed    ActiveStatus = "failed"    // Session failed with error
	ActiveStatusTimedOut  ActiveStatus = "timed_out" // Session timed out
)

// TaskContext tracks what OpenSpec change/task a session is working on
type TaskContext struct {
	ChangeID string `json:"change_id,omitempty"` // The OpenSpec change being built
	Mode     string `json:"mode,omitempty"`      // "plan", "build", or "interactive"
	BuildAll bool   `json:"build_all,omitempty"` // Whether building all changes
}

// CallerToolDefinition defines a tool that can be called on the external caller
type CallerToolDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	InputSchema any    `json:"inputSchema,omitempty"` // JSON Schema for tool arguments
}

// CallerToolResponse holds the response from a caller tool execution
type CallerToolResponse struct {
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

// ActiveSession represents a running streaming session with its executor and buffer
type ActiveSession struct {
	SessionID    string
	ProjectID    string
	WorkspaceID  string // For workspace-based lookup
	ContainerID  string
	Executor     agent.StreamingExecutor
	EventBuffer  *EventBuffer
	StartedAt    time.Time
	LastActivity time.Time
	Status       ActiveStatus
	Error        error // Set when Status is Failed
	TaskCtx      *TaskContext
	mcpSession   *mcp.ServerSession // MCP session for SSE event push

	// Caller tool relay fields
	callerID              string                              // ID of the caller (e.g., "myapp")
	callerTools           []CallerToolDefinition              // Tools declared by the caller
	pendingCallerRequests map[string]chan *CallerToolResponse // request_id -> response channel
	mu                    sync.RWMutex
	executorMu            sync.RWMutex // Protects Executor field access
	mcpMu                 sync.RWMutex // Protects mcpSession field access
	callerMu              sync.RWMutex // Protects callerID and callerTools fields
}

// NewActiveSession creates a new active session
func NewActiveSession(sessionID, projectID, workspaceID, containerID string, executor agent.StreamingExecutor) *ActiveSession {
	now := time.Now()
	return &ActiveSession{
		SessionID:    sessionID,
		ProjectID:    projectID,
		WorkspaceID:  workspaceID,
		ContainerID:  containerID,
		Executor:     executor,
		EventBuffer:  NewEventBuffer(sessionID, DefaultEventBufferSize),
		StartedAt:    now,
		LastActivity: now,
		Status:       ActiveStatusRunning,
	}
}

// SendMessage sends a message to the session and updates activity time
func (a *ActiveSession) SendMessage(message string) error {
	a.mu.Lock()
	a.LastActivity = time.Now()
	a.Status = ActiveStatusRunning // Message sent means we're processing
	a.mu.Unlock()

	a.executorMu.RLock()
	executor := a.Executor
	a.executorMu.RUnlock()

	if executor == nil {
		return fmt.Errorf("executor not initialized")
	}
	return executor.SendMessage(message)
}

// GetEvents returns buffered events after the given index
func (a *ActiveSession) GetEvents(sinceIndex int) ([]*BufferedEvent, error) {
	return a.EventBuffer.After(sinceIndex)
}

// GetExecutor returns the executor with read lock protection
func (a *ActiveSession) GetExecutor() agent.StreamingExecutor {
	a.executorMu.RLock()
	defer a.executorMu.RUnlock()
	return a.Executor
}

// SetExecutor replaces the executor (used when resuming a session with a new connection)
func (a *ActiveSession) SetExecutor(executor agent.StreamingExecutor) {
	a.executorMu.Lock()
	defer a.executorMu.Unlock()
	a.Executor = executor
}

// CloseExecutor safely closes the executor with write lock protection
func (a *ActiveSession) CloseExecutor() {
	a.executorMu.Lock()
	executor := a.Executor
	a.Executor = nil
	a.executorMu.Unlock()

	if executor != nil {
		_ = executor.Close()
	}
}

// IsRunning returns true if the session can receive messages (idle or running)
func (a *ActiveSession) IsRunning() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.Status == ActiveStatusRunning || a.Status == ActiveStatusIdle
}

// SetStatus updates the session status
func (a *ActiveSession) SetStatus(status ActiveStatus, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.Status = status
	a.Error = err
}

// GetStatus returns the current status
func (a *ActiveSession) GetStatus() ActiveStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.Status
}

// SetTaskContext updates the task context for this session
func (a *ActiveSession) SetTaskContext(ctx *TaskContext) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.TaskCtx = ctx
	a.LastActivity = time.Now()
}

// GetTaskContext returns the current task context
func (a *ActiveSession) GetTaskContext() *TaskContext {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.TaskCtx
}

// LastActivityTime returns the last activity time
func (a *ActiveSession) LastActivityTime() time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.LastActivity
}

// SetCallerTools sets the caller ID and tools for this session
func (a *ActiveSession) SetCallerTools(callerID string, tools []CallerToolDefinition) {
	a.callerMu.Lock()
	defer a.callerMu.Unlock()
	a.callerID = callerID
	a.callerTools = tools
}

// GetCallerTools returns the caller ID and tools for this session
func (a *ActiveSession) GetCallerTools() (string, []CallerToolDefinition) {
	a.callerMu.RLock()
	defer a.callerMu.RUnlock()
	return a.callerID, a.callerTools
}

// HasCallerTools returns true if the session has caller tools configured
func (a *ActiveSession) HasCallerTools() bool {
	a.callerMu.RLock()
	defer a.callerMu.RUnlock()
	return a.callerID != "" && len(a.callerTools) > 0
}

// RegisterCallerRequest creates a response channel for a pending caller tool request.
// Returns the channel that will receive the response.
func (a *ActiveSession) RegisterCallerRequest(requestID string) chan *CallerToolResponse {
	a.callerMu.Lock()
	defer a.callerMu.Unlock()

	if a.pendingCallerRequests == nil {
		a.pendingCallerRequests = make(map[string]chan *CallerToolResponse)
	}

	ch := make(chan *CallerToolResponse, 1)
	a.pendingCallerRequests[requestID] = ch
	return ch
}

// ResolveCallerRequest sends a response to a pending caller tool request.
// Returns true if the request was found and resolved.
func (a *ActiveSession) ResolveCallerRequest(requestID string, response *CallerToolResponse) bool {
	a.callerMu.Lock()
	defer a.callerMu.Unlock()

	ch, ok := a.pendingCallerRequests[requestID]
	if !ok {
		return false
	}

	// Send response (non-blocking since channel is buffered)
	select {
	case ch <- response:
	default:
		// Channel already has a response (shouldn't happen with buffer of 1)
	}

	delete(a.pendingCallerRequests, requestID)
	return true
}

// CancelCallerRequest cancels a pending request (used on timeout or disconnect)
func (a *ActiveSession) CancelCallerRequest(requestID string) {
	a.callerMu.Lock()
	defer a.callerMu.Unlock()

	if ch, ok := a.pendingCallerRequests[requestID]; ok {
		close(ch)
		delete(a.pendingCallerRequests, requestID)
	}
}

// SetMCPSession sets the MCP ServerSession for SSE event push
func (a *ActiveSession) SetMCPSession(session *mcp.ServerSession) {
	a.mcpMu.Lock()
	defer a.mcpMu.Unlock()
	a.mcpSession = session
}

// GetMCPSession returns the MCP ServerSession
func (a *ActiveSession) GetMCPSession() *mcp.ServerSession {
	a.mcpMu.RLock()
	defer a.mcpMu.RUnlock()
	return a.mcpSession
}

// eventNotification is the structured payload sent as MCP log notifications.
type eventNotification struct {
	SessionID     string `json:"session_id"`
	Type          string `json:"type"`
	Text          string `json:"text,omitempty"`
	ToolName      string `json:"tool_name,omitempty"`
	FinalResponse string `json:"final_response,omitempty"`
}

// NotifyEvent sends a session event to the connected MCP client via Log.
// Returns nil if no MCP session is connected (graceful degradation).
func (a *ActiveSession) NotifyEvent(ctx context.Context, event *agent.StreamEvent) error {
	a.mcpMu.RLock()
	session := a.mcpSession
	a.mcpMu.RUnlock()

	if session == nil {
		return nil
	}

	data := eventNotification{
		SessionID: a.SessionID,
		Type:      string(event.Type),
		Text:      event.Text,
		ToolName:  event.ToolName,
	}
	if event.Type == agent.StreamEventCompletion && event.FinalText != "" {
		data.FinalResponse = event.FinalText
	}

	if err := session.Log(ctx, &mcp.LoggingMessageParams{
		Logger: "oubliette.session",
		Level:  "info",
		Data:   data,
	}); err != nil {
		logger.Error("Failed to push event to MCP client: %v", err)
		return err
	}
	return nil
}

// ActiveSessionManager manages active streaming sessions
type ActiveSessionManager struct {
	sessions    map[string]*ActiveSession    // by session ID
	byProject   map[string][]string          // project ID -> session IDs
	byWorkspace map[string]map[string]string // project ID -> workspace ID -> session ID
	maxPerProj  int
	idleTimeout time.Duration
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewActiveSessionManager creates a new active session manager
func NewActiveSessionManager(maxPerProject int, idleTimeout time.Duration) *ActiveSessionManager {
	if maxPerProject <= 0 {
		maxPerProject = DefaultMaxActiveSessions
	}
	if idleTimeout <= 0 {
		idleTimeout = DefaultSessionIdleTimeout
	}

	ctx, cancel := context.WithCancel(context.Background())
	m := &ActiveSessionManager{
		sessions:    make(map[string]*ActiveSession),
		byProject:   make(map[string][]string),
		byWorkspace: make(map[string]map[string]string),
		maxPerProj:  maxPerProject,
		idleTimeout: idleTimeout,
		ctx:         ctx,
		cancel:      cancel,
	}

	// Start background cleanup goroutine
	go m.cleanupLoop()

	return m
}

// Register adds an active session to the manager
func (m *ActiveSessionManager) Register(sess *ActiveSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check per-project limit
	if len(m.byProject[sess.ProjectID]) >= m.maxPerProj {
		logger.Error("Session registration rejected: max sessions (%d) reached for project %s", m.maxPerProj, sess.ProjectID)
		return fmt.Errorf("maximum active sessions (%d) reached for project %s", m.maxPerProj, sess.ProjectID)
	}

	m.sessions[sess.SessionID] = sess
	m.byProject[sess.ProjectID] = append(m.byProject[sess.ProjectID], sess.SessionID)

	// Add to workspace index
	if sess.WorkspaceID != "" {
		if m.byWorkspace[sess.ProjectID] == nil {
			m.byWorkspace[sess.ProjectID] = make(map[string]string)
		}
		m.byWorkspace[sess.ProjectID][sess.WorkspaceID] = sess.SessionID
		logger.Info("Session registered: %s (project: %s, workspace: %s)", sess.SessionID, sess.ProjectID, sess.WorkspaceID)
	} else {
		logger.Info("Session registered: %s (project: %s, no workspace)", sess.SessionID, sess.ProjectID)
	}

	// Record metrics for session start
	metrics.RecordSessionStart(sess.ProjectID)

	// Start event collection goroutine
	go m.collectEvents(sess)

	return nil
}

// RestartEventCollection starts a new event collection goroutine for a session
// whose executor has been replaced (e.g., after resume).
func (m *ActiveSessionManager) RestartEventCollection(sess *ActiveSession) {
	go m.collectEvents(sess)
}

// Get returns an active session by ID
func (m *ActiveSessionManager) Get(sessionID string) (*ActiveSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sess, ok := m.sessions[sessionID]
	return sess, ok
}

// GetByWorkspace returns an active session for a project+workspace combination
func (m *ActiveSessionManager) GetByWorkspace(projectID, workspaceID string) (*ActiveSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if projectWorkspaces, ok := m.byWorkspace[projectID]; ok {
		if sessionID, ok := projectWorkspaces[workspaceID]; ok {
			if sess, ok := m.sessions[sessionID]; ok {
				logger.Info("GetByWorkspace found session %s for project %s, workspace %s", sessionID, projectID, workspaceID)
				return sess, true
			}
		}
	}
	logger.Info("GetByWorkspace: no active session for project %s, workspace %s", projectID, workspaceID)
	return nil, false
}

// Remove removes an active session from the manager
func (m *ActiveSessionManager) Remove(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sess, ok := m.sessions[sessionID]
	if !ok {
		logger.Info("Remove: session %s not found", sessionID)
		return
	}

	status := sess.GetStatus()
	logger.Info("Removing session: %s (project: %s, workspace: %s, status: %s)", sessionID, sess.ProjectID, sess.WorkspaceID, status)

	// Record metrics for session end
	durationSeconds := time.Since(sess.StartedAt).Seconds()
	metrics.RecordSessionEnd(sess.ProjectID, string(status), durationSeconds)

	// Close the executor safely
	sess.CloseExecutor()

	delete(m.sessions, sessionID)

	// Remove from project index
	projectSessions := m.byProject[sess.ProjectID]
	for i, id := range projectSessions {
		if id == sessionID {
			m.byProject[sess.ProjectID] = append(projectSessions[:i], projectSessions[i+1:]...)
			break
		}
	}

	// Remove from workspace index
	if sess.WorkspaceID != "" {
		if projectWorkspaces, ok := m.byWorkspace[sess.ProjectID]; ok {
			delete(projectWorkspaces, sess.WorkspaceID)
		}
	}
}

// SendMessage sends a message to an active session
func (m *ActiveSessionManager) SendMessage(sessionID, message string) error {
	sess, ok := m.Get(sessionID)
	if !ok {
		return fmt.Errorf("session %s not found or not active", sessionID)
	}

	if !sess.IsRunning() {
		return fmt.Errorf("session %s is not running (status: %s)", sessionID, sess.GetStatus())
	}

	return sess.SendMessage(message)
}

// GetEvents returns buffered events for a session
func (m *ActiveSessionManager) GetEvents(sessionID string, sinceIndex int) ([]*BufferedEvent, error) {
	sess, ok := m.Get(sessionID)
	if !ok {
		return nil, fmt.Errorf("session %s not found or not active", sessionID)
	}
	return sess.GetEvents(sinceIndex)
}

// GetLastEventIndex returns the last event index for a session
func (m *ActiveSessionManager) GetLastEventIndex(sessionID string) (int, error) {
	sess, ok := m.Get(sessionID)
	if !ok {
		return -1, fmt.Errorf("session %s not found or not active", sessionID)
	}
	return sess.EventBuffer.LastIndex(), nil
}

// ListByProject returns all active sessions for a project
func (m *ActiveSessionManager) ListByProject(projectID string) []*ActiveSession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*ActiveSession
	for _, sessionID := range m.byProject[projectID] {
		if sess, ok := m.sessions[sessionID]; ok {
			result = append(result, sess)
		}
	}
	return result
}

// Count returns the total number of active sessions
func (m *ActiveSessionManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// CountByProject returns the number of active sessions for a project
func (m *ActiveSessionManager) CountByProject(projectID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.byProject[projectID])
}

// GetSessionsByChangeID returns all active sessions working on a specific change
func (m *ActiveSessionManager) GetSessionsByChangeID(projectID, changeID string) []*ActiveSession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*ActiveSession
	for _, sessionID := range m.byProject[projectID] {
		if sess, ok := m.sessions[sessionID]; ok {
			if ctx := sess.GetTaskContext(); ctx != nil && ctx.ChangeID == changeID {
				result = append(result, sess)
			}
		}
	}
	return result
}

// Close shuts down the manager and all active sessions
func (m *ActiveSessionManager) Close() {
	m.cancel()

	m.mu.Lock()
	defer m.mu.Unlock()

	for sessionID, sess := range m.sessions {
		sess.CloseExecutor()
		delete(m.sessions, sessionID)
	}
	m.byProject = make(map[string][]string)
}

// collectEvents reads events from the executor and buffers them
func (m *ActiveSessionManager) collectEvents(sess *ActiveSession) {
	defer func() {
		status := sess.GetStatus()
		if status == ActiveStatusRunning || status == ActiveStatusIdle {
			sess.SetStatus(ActiveStatusCompleted, nil)
		}
	}()

	executor := sess.GetExecutor()
	if executor == nil {
		return
	}

	var lastAssistantText string
	var completionNotified bool

	for {
		select {
		case <-m.ctx.Done():
			return

		case event, ok := <-executor.Events():
			if !ok {
				return
			}
			// If executor was replaced (session resumed), exit so the new goroutine takes over
			if sess.GetExecutor() != executor {
				return
			}

			// Track status transitions
			if event.Type == agent.StreamEventCompletion {
				sess.SetStatus(ActiveStatusIdle, nil)
			} else if sess.GetStatus() == ActiveStatusIdle && isWorkEvent(event) {
				sess.SetStatus(ActiveStatusRunning, nil)
				completionNotified = false
				lastAssistantText = ""
			}

			// Track last consolidated assistant message for attaching to completion
			if event.Type == agent.StreamEventMessage && event.Text != "" {
				lastAssistantText = event.Text
			}

			sess.EventBuffer.Append(event)

			if !isNotifiableEvent(event) {
				continue
			}

			// Deduplicate completions and attach final response text
			if event.Type == agent.StreamEventCompletion {
				if completionNotified {
					continue
				}
				completionNotified = true
				if event.Text == "" && lastAssistantText != "" {
					event.FinalText = lastAssistantText
					event.Text = lastAssistantText
				}
			}

			if err := sess.NotifyEvent(context.Background(), event); err != nil {
				logger.Error("Failed to push SSE event for session %s: %v", sess.SessionID, err)
			}

		case err := <-executor.Errors():
			if err != nil {
				sess.SetStatus(ActiveStatusFailed, err)
				return
			}
		}
	}
}

// isNotifiableEvent returns true for events worth pushing as MCP notifications.
// Only completion (with final response), tool calls, and tool results are pushed.
// Message updates, system metadata, and token deltas are too noisy.
func isNotifiableEvent(event *agent.StreamEvent) bool {
	switch event.Type {
	case agent.StreamEventCompletion:
		return true
	case agent.StreamEventToolCall, agent.StreamEventToolResult:
		return true
	case agent.StreamEventError:
		return true
	default:
		return false
	}
}

// isWorkEvent returns true if the event indicates actual processing work
func isWorkEvent(event *agent.StreamEvent) bool {
	switch event.Type {
	case agent.StreamEventDelta, agent.StreamEventToolCall, agent.StreamEventToolResult:
		return true
	case agent.StreamEventMessage:
		return event.Role == "assistant"
	default:
		return false
	}
}

// cleanupLoop periodically checks for idle sessions
func (m *ActiveSessionManager) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.cleanupIdleSessions()
		}
	}
}

// cleanupIdleSessions removes sessions that have been idle too long
func (m *ActiveSessionManager) cleanupIdleSessions() {
	m.mu.RLock()
	var toRemove []string
	now := time.Now()

	for sessionID, sess := range m.sessions {
		if sess.IsRunning() && now.Sub(sess.LastActivityTime()) > m.idleTimeout {
			toRemove = append(toRemove, sessionID)
		}
	}
	m.mu.RUnlock()

	if len(toRemove) > 0 {
		logger.Info("Cleaning up %d idle sessions", len(toRemove))
	}

	// Remove idle sessions
	for _, sessionID := range toRemove {
		if sess, ok := m.Get(sessionID); ok {
			logger.Info("Session %s timed out after %v idle (project: %s, workspace: %s)",
				sessionID, now.Sub(sess.LastActivityTime()), sess.ProjectID, sess.WorkspaceID)
			sess.SetStatus(ActiveStatusTimedOut, fmt.Errorf("session timed out after %v of inactivity", m.idleTimeout))
			sess.CloseExecutor()
		}
		m.Remove(sessionID)
	}
}
