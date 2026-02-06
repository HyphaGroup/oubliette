package mcp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/HyphaGroup/oubliette/internal/agent"
	"github.com/HyphaGroup/oubliette/internal/auth"
	"github.com/HyphaGroup/oubliette/internal/config"
	"github.com/HyphaGroup/oubliette/internal/container"
	"github.com/HyphaGroup/oubliette/internal/logger"
	"github.com/HyphaGroup/oubliette/internal/metrics"
	"github.com/HyphaGroup/oubliette/internal/project"
	"github.com/HyphaGroup/oubliette/internal/schedule"
	"github.com/HyphaGroup/oubliette/internal/session"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// generateRequestID creates a unique request identifier
func generateRequestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Server wraps the MCP server with our managers
type Server struct {
	projectMgr      *project.Manager
	runtime         container.Runtime
	imageManager    *container.ImageManager // Manages container images from config
	agentRuntime    agent.Runtime
	sessionMgr      *session.Manager
	activeSessions  *session.ActiveSessionManager
	authStore       *auth.Store
	socketHandler   *SocketHandler
	mcpServer       *mcp.Server                // The underlying MCP server for handling requests
	registry        *Registry                  // Tool registry for unified tool management
	containerMemory string                     // Container memory limit (e.g., "4G")
	containerCPUs   int                        // Container CPU limit
	credentials     *config.CredentialRegistry // Unified credential registry
	modelRegistry   *config.ModelRegistry      // Model configuration registry
	scheduleStore   *schedule.Store            // Schedule persistence
	scheduleRunner  *schedule.Runner           // Schedule execution runner
}

// ServerConfig holds container resource configuration
type ServerConfig struct {
	ContainerMemory string
	ContainerCPUs   int
	Credentials     *config.CredentialRegistry
	ModelRegistry   *config.ModelRegistry
	ImageManager    *container.ImageManager
	AgentRuntime    agent.Runtime
	ScheduleStore   *schedule.Store
}

// NewServer creates a new MCP server instance
func NewServer(projectMgr *project.Manager, runtime container.Runtime, sessionMgr *session.Manager, authStore *auth.Store, socketsDir string, cfg *ServerConfig) *Server {
	memory := "4G"
	cpus := 4
	var credentials *config.CredentialRegistry
	var modelRegistry *config.ModelRegistry
	var imageMgr *container.ImageManager
	var agentRt agent.Runtime
	var schedStore *schedule.Store
	if cfg != nil {
		if cfg.ContainerMemory != "" {
			memory = cfg.ContainerMemory
		}
		if cfg.ContainerCPUs > 0 {
			cpus = cfg.ContainerCPUs
		}
		credentials = cfg.Credentials
		modelRegistry = cfg.ModelRegistry
		imageMgr = cfg.ImageManager
		agentRt = cfg.AgentRuntime
		schedStore = cfg.ScheduleStore
	}

	s := &Server{
		projectMgr:      projectMgr,
		runtime:         runtime,
		imageManager:    imageMgr,
		agentRuntime:    agentRt,
		sessionMgr:      sessionMgr,
		activeSessions:  session.NewActiveSessionManager(session.DefaultMaxActiveSessions, session.DefaultSessionIdleTimeout),
		authStore:       authStore,
		registry:        NewRegistry(),
		containerMemory: memory,
		containerCPUs:   cpus,
		credentials:     credentials,
		modelRegistry:   modelRegistry,
		scheduleStore:   schedStore,
	}

	// Initialize schedule runner if store is provided
	if schedStore != nil {
		s.scheduleRunner = schedule.NewRunner(schedStore, s.executeScheduleTarget)
	}
	s.socketHandler = NewSocketHandler(s)

	// Register all tools with the registry
	s.registerAllTools(s.registry)

	// Register as session checker for safe deletion
	projectMgr.SetSessionChecker(s)

	return s
}

// HasActiveSessionsForProject checks if a project has any active sessions
// Implements project.ActiveSessionChecker
func (s *Server) HasActiveSessionsForProject(projectID string) bool {
	return s.activeSessions.CountByProject(projectID) > 0
}

// HasActiveSessionsForWorkspace checks if a workspace has an active session
// Implements project.ActiveSessionChecker
func (s *Server) HasActiveSessionsForWorkspace(projectID, workspaceID string) bool {
	_, ok := s.activeSessions.GetByWorkspace(projectID, workspaceID)
	return ok
}

// GetSocketHandler returns the socket handler for JSON-RPC communication
func (s *Server) GetSocketHandler() *SocketHandler {
	return s.socketHandler
}

// HasAPICredentials returns true if any API credentials are configured
// (provider credentials for Anthropic, OpenAI, etc.)
func (s *Server) HasAPICredentials() bool {
	if s.credentials == nil {
		return false
	}
	if cred, ok := s.credentials.GetDefaultProviderCredential(); ok && cred.APIKey != "" {
		return true
	}
	return false
}

// Close shuts down the server and cleans up resources
func (s *Server) Close() {
	// Stop schedule runner first (waits for in-flight)
	if s.scheduleRunner != nil {
		s.scheduleRunner.Stop()
	}

	// Close all active sessions
	s.activeSessions.Close()

	// Close socket handler
	s.socketHandler.Close()
}

// Serve starts the MCP HTTP server
func (s *Server) Serve(addr string) error {
	// Start schedule runner if configured
	if s.scheduleRunner != nil {
		s.scheduleRunner.Start()
	}

	// Create MCP server (store for socket connections too)
	s.mcpServer = mcp.NewServer(&mcp.Implementation{
		Name:    "oubliette",
		Version: "0.1.0",
	}, nil)

	// Register tools from registry
	s.registry.RegisterWithMCPServer(s.mcpServer)

	// Cleanup stale socket directories from previous runs
	// (New architecture: relay creates sockets inside containers)

	// Create HTTP handler with streamable transport
	// Enable EventStore for SSE stream resumption support
	mcpHandler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return s.mcpServer
	}, &mcp.StreamableHTTPOptions{
		EventStore: mcp.NewMemoryEventStore(nil),
	})

	// Wrap with request ID and logging middleware
	loggingHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate or extract request ID
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		// Add request ID to response headers
		w.Header().Set("X-Request-ID", requestID)

		// Add to context for downstream handlers
		ctx := context.WithValue(r.Context(), logger.ContextKeyRequestID, requestID)
		ctx = WithRemoteAddr(ctx, r.RemoteAddr)
		r = r.WithContext(ctx)

		logger.Info("HTTP %s %s from %s [request_id=%s]", r.Method, r.URL.Path, r.RemoteAddr, requestID)
		mcpHandler.ServeHTTP(w, r)
	})

	// Wrap with auth middleware (Bearer token only, socket auth is separate)
	authedHandler := auth.Middleware(s.authStore)(loggingHandler)

	// Wrap with rate limiting (after auth, so we can rate limit per-token)
	rateLimiter := auth.DefaultRateLimiter() // 10 req/s, burst 20
	rateLimitedHandler := auth.RateLimitMiddleware(rateLimiter)(authedHandler)

	// Create main mux with health endpoints (no auth required) and MCP endpoints
	mainMux := http.NewServeMux()

	// Health endpoints - no authentication required
	mainMux.HandleFunc("/health", s.handleHealthCheck)
	mainMux.HandleFunc("/ready", s.handleReadinessCheck)

	// Metrics endpoint - no authentication required (Prometheus scraping)
	mainMux.Handle("/metrics", metrics.Handler())

	// MCP endpoints - require authentication, rate limiting, wrapped with metrics middleware
	mainMux.Handle("/mcp", metrics.Middleware(rateLimitedHandler))
	mainMux.Handle("/mcp/", metrics.Middleware(rateLimitedHandler))

	logger.Info("🚀 Oubliette MCP server listening on %s", addr)
	logger.Info("🔌 Relay sockets directory: %s", SocketsBaseDir)
	logger.Info("💚 Health check: http://localhost%s/health", addr)
	logger.Info("💚 Readiness check: http://localhost%s/ready", addr)
	logger.Info("📊 Metrics: http://localhost%s/metrics", addr)
	return http.ListenAndServe(addr, mainMux)
}

// handleHealthCheck is a basic liveness check
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// handleReadinessCheck verifies the server can serve requests
func (s *Server) handleReadinessCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check container runtime availability
	if err := s.runtime.Ping(r.Context()); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"not ready","reason":"container runtime unavailable"}`))
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ready"}`))
}

// GetRegistry returns the tool registry for external access (e.g., coverage tooling)
func (s *Server) GetRegistry() *Registry {
	return s.registry
}

// ScheduleExecutionResult contains the result of executing a schedule target
type ScheduleExecutionResult struct {
	SessionID string
	Output    string
}

// executeScheduleTarget is called by the schedule runner to execute a single target
// It sends a message to the target project/workspace using pinned session logic.
func (s *Server) executeScheduleTarget(ctx context.Context, sched *schedule.Schedule, target *schedule.ScheduleTarget) ([]string, error) {
	startTime := time.Now()

	result, err := s.doExecuteScheduleTarget(ctx, sched, target)

	// Record execution in history
	exec := &schedule.Execution{
		ScheduleID: sched.ID,
		TargetID:   target.ID,
		ExecutedAt: startTime,
		DurationMs: time.Since(startTime).Milliseconds(),
	}

	if err != nil {
		exec.Status = schedule.ExecutionFailed
		exec.Error = err.Error()
	} else {
		exec.Status = schedule.ExecutionSuccess
		exec.SessionID = result.SessionID
		exec.Output = result.Output

		// Update target with session ID and last output
		if updateErr := s.scheduleStore.UpdateTargetExecution(target.ID, result.SessionID, result.Output); updateErr != nil {
			logger.Error("Failed to update target execution: %v", updateErr)
		}
	}

	if recordErr := s.scheduleStore.RecordExecution(exec); recordErr != nil {
		logger.Error("Failed to record execution: %v", recordErr)
	}

	if err != nil {
		return nil, err
	}
	return []string{result.SessionID}, nil
}

// doExecuteScheduleTarget performs the actual execution logic
func (s *Server) doExecuteScheduleTarget(ctx context.Context, sched *schedule.Schedule, target *schedule.ScheduleTarget) (*ScheduleExecutionResult, error) {
	// Determine workspace ID (use default if not specified)
	workspaceID := target.WorkspaceID
	if workspaceID == "" {
		proj, err := s.projectMgr.Get(target.ProjectID)
		if err != nil {
			return nil, err
		}
		workspaceID = proj.DefaultWorkspaceID
	}

	// For new session behavior, clear pinned session
	if sched.SessionBehavior == schedule.SessionNew {
		if activeSess, ok := s.activeSessions.GetByWorkspace(target.ProjectID, workspaceID); ok {
			s.activeSessions.Remove(activeSess.SessionID)
		}
		// Clear pinned session - will spawn fresh
		target.SessionID = ""
	}

	// Try to use pinned session first
	if target.SessionID != "" {
		// Check if session is active
		if activeSess, ok := s.activeSessions.Get(target.SessionID); ok && activeSess.IsRunning() {
			if err := activeSess.SendMessage(sched.Prompt); err != nil {
				return nil, err
			}
			output := s.waitForSessionOutput(activeSess, target.SessionID)
			return &ScheduleExecutionResult{SessionID: target.SessionID, Output: output}, nil
		}

		// Session not active - try to resume from disk
		existingSession, err := s.sessionMgr.Load(target.SessionID)
		if err == nil && existingSession != nil && existingSession.RuntimeSessionID != "" {
			env, err := s.prepareSessionEnvironment(ctx, target.ProjectID, workspaceID, false, "", "schedule")
			if err != nil {
				logger.Info("Failed to prepare environment for session resume: %v", err)
			} else {
				opts := session.StartOptions{
					WorkspaceID:        workspaceID,
					WorkspaceIsolation: env.project.WorkspaceIsolation,
					RuntimeOverride:    s.agentRuntime,
				}

				resumedSess, executor, resumeErr := s.sessionMgr.ResumeBidirectionalSession(ctx, existingSession, env.containerName, sched.Prompt, opts)
				if resumeErr == nil {
					activeSess := session.NewActiveSession(resumedSess.SessionID, target.ProjectID, workspaceID, env.containerName, executor)
					if regErr := s.activeSessions.Register(activeSess); regErr != nil {
						_ = executor.Close()
						logger.Info("Failed to register resumed session: %v", regErr)
					} else {
						// Connect to relay in background
						go func() {
							if err := s.socketHandler.ConnectSession(context.Background(), target.ProjectID, resumedSess.SessionID, 0); err != nil {
								logger.Error("Failed to connect to relay for resumed session %s: %v", resumedSess.SessionID, err)
							}
							if activeSess, ok := s.activeSessions.Get(resumedSess.SessionID); ok && activeSess.IsRunning() {
								logger.Info("Session %s relay connection closed, marking as completed", resumedSess.SessionID)
								activeSess.SetStatus(session.ActiveStatusCompleted, nil)
							}
						}()

						output := s.waitForSessionOutput(activeSess, resumedSess.SessionID)
						return &ScheduleExecutionResult{SessionID: resumedSess.SessionID, Output: output}, nil
					}
				} else {
					logger.Info("Failed to resume pinned session %s, will spawn new: %v", target.SessionID, resumeErr)
				}
			}
		}
	}

	// No pinned session or resume failed - spawn new
	env, err := s.prepareSessionEnvironment(ctx, target.ProjectID, workspaceID, false, "", "schedule")
	if err != nil {
		return nil, err
	}

	opts := session.StartOptions{
		WorkspaceID:        env.workspaceID,
		WorkspaceIsolation: env.project.WorkspaceIsolation,
		RuntimeOverride:    s.agentRuntime,
	}

	sess, activeSess, err := s.spawnAndRegisterSession(ctx, target.ProjectID, env.containerName, env.workspaceID, sched.Prompt, opts, nil)
	if err != nil {
		return nil, err
	}

	output := s.waitForSessionOutput(activeSess, sess.SessionID)
	return &ScheduleExecutionResult{SessionID: sess.SessionID, Output: output}, nil
}

// waitForSessionOutput waits for the session to complete its current task and returns the output
func (s *Server) waitForSessionOutput(activeSess *session.ActiveSession, sessionID string) string {
	// Wait for session to become idle (up to 5 minutes)
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			logger.Info("Timeout waiting for session %s output", sessionID)
			return ""
		case <-ticker.C:
			status := activeSess.GetStatus()
			if status == session.ActiveStatusIdle || status == session.ActiveStatusCompleted || status == session.ActiveStatusFailed {
				// Session finished - get the last turn output
				sess, err := s.sessionMgr.Load(sessionID)
				if err != nil || sess == nil || len(sess.Turns) == 0 {
					return ""
				}
				return sess.Turns[len(sess.Turns)-1].Output.Text
			}
		}
	}
}
