package mcp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"

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
	agentRuntime    agent.Runtime           // Agent runtime (droid or opencode)
	runtimeFactory  RuntimeFactoryFunc      // Factory to create runtimes by name
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

// RuntimeFactoryFunc creates an agent runtime by name (e.g., "droid", "opencode")
// Returns nil if the runtime type is not supported
type RuntimeFactoryFunc func(runtimeType string) agent.Runtime

// ServerConfig holds container resource configuration
type ServerConfig struct {
	ContainerMemory string                     // e.g., "4G"
	ContainerCPUs   int                        // e.g., 4
	Credentials     *config.CredentialRegistry // Unified credential registry
	ModelRegistry   *config.ModelRegistry      // Model configuration registry
	ImageManager    *container.ImageManager    // Manages container images from config
	AgentRuntime    agent.Runtime              // Agent runtime (droid or opencode)
	RuntimeFactory  RuntimeFactoryFunc         // Factory to create runtimes by name for per-project overrides
	ScheduleStore   *schedule.Store            // Schedule persistence store
}

// NewServer creates a new MCP server instance
func NewServer(projectMgr *project.Manager, runtime container.Runtime, sessionMgr *session.Manager, authStore *auth.Store, socketsDir string, cfg *ServerConfig) *Server {
	memory := "4G"
	cpus := 4
	var credentials *config.CredentialRegistry
	var modelRegistry *config.ModelRegistry
	var imageMgr *container.ImageManager
	var agentRt agent.Runtime
	var runtimeFactory RuntimeFactoryFunc
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
		runtimeFactory = cfg.RuntimeFactory
		schedStore = cfg.ScheduleStore
	}

	s := &Server{
		projectMgr:      projectMgr,
		runtime:         runtime,
		imageManager:    imageMgr,
		agentRuntime:    agentRt,
		runtimeFactory:  runtimeFactory,
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

// GetRuntimeForProject returns the appropriate agent runtime for a project.
// If the project has an agent_runtime override and a factory is configured,
// uses the factory to create the appropriate runtime. Otherwise returns server default.
func (s *Server) GetRuntimeForProject(proj *project.Project) agent.Runtime {
	// If project has a runtime override and we have a factory
	if proj.AgentRuntime != "" && s.runtimeFactory != nil {
		if rt := s.runtimeFactory(proj.AgentRuntime); rt != nil {
			return rt
		}
		// Factory returned nil (runtime type not supported), fall through to default
	}
	// Return server default runtime
	return s.agentRuntime
}

// GetSocketHandler returns the socket handler for JSON-RPC communication
func (s *Server) GetSocketHandler() *SocketHandler {
	return s.socketHandler
}

// HasAPICredentials returns true if any API credentials are configured
// (either Factory API key or provider credentials)
func (s *Server) HasAPICredentials() bool {
	if s.credentials == nil {
		return false
	}
	// Check for Factory API key
	if key, ok := s.credentials.GetDefaultFactoryKey(); ok && key != "" {
		return true
	}
	// Check for provider credentials
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

// executeScheduleTarget is called by the schedule runner to execute a single target
// It sends a message to the target project/workspace, reusing existing session logic.
func (s *Server) executeScheduleTarget(ctx context.Context, sched *schedule.Schedule, target *schedule.ScheduleTarget) ([]string, error) {
	// Determine workspace ID (use default if not specified)
	workspaceID := target.WorkspaceID
	if workspaceID == "" {
		proj, err := s.projectMgr.Get(target.ProjectID)
		if err != nil {
			return nil, err
		}
		workspaceID = proj.DefaultWorkspaceID
	}

	// For new session behavior, remove any existing active session first
	if sched.SessionBehavior == schedule.SessionNew {
		if activeSess, ok := s.activeSessions.GetByWorkspace(target.ProjectID, workspaceID); ok {
			s.activeSessions.Remove(activeSess.SessionID)
		}
	}

	// Check for existing active session (resume behavior)
	if activeSess, ok := s.activeSessions.GetByWorkspace(target.ProjectID, workspaceID); ok && activeSess.IsRunning() {
		// Send message to existing session
		if err := activeSess.SendMessage(sched.Prompt); err != nil {
			return nil, err
		}
		return []string{activeSess.SessionID}, nil
	}

	// Need to spawn new session - use prepareSessionEnvironment + spawnAndRegisterSession
	env, err := s.prepareSessionEnvironment(ctx, target.ProjectID, workspaceID, false, "", "schedule", nil)
	if err != nil {
		return nil, err
	}

	projectRuntime := s.GetRuntimeForProject(env.project)
	opts := session.StartOptions{
		WorkspaceID:        env.workspaceID,
		WorkspaceIsolation: env.project.WorkspaceIsolation,
		RuntimeOverride:    projectRuntime,
	}

	sess, _, err := s.spawnAndRegisterSession(ctx, target.ProjectID, env.containerName, env.workspaceID, sched.Prompt, opts, nil)
	if err != nil {
		return nil, err
	}

	return []string{sess.SessionID}, nil
}
