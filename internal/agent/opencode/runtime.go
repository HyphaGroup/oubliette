package opencode

import (
	"context"
	"fmt"
	"sync"

	"github.com/HyphaGroup/oubliette/internal/agent"
	"github.com/HyphaGroup/oubliette/internal/container"
)

// Runtime implements agent.Runtime for OpenCode
type Runtime struct {
	containerRuntime container.Runtime
	initialized      bool

	// Server management per container
	serversMu sync.RWMutex
	servers   map[string]*Server // containerID -> server
}

// Ensure Runtime implements agent.Runtime
var _ agent.Runtime = (*Runtime)(nil)

// NewRuntime creates a new OpenCode runtime
func NewRuntime(containerRuntime container.Runtime) *Runtime {
	return &Runtime{
		containerRuntime: containerRuntime,
		servers:          make(map[string]*Server),
	}
}

// Initialize prepares the runtime with configuration
func (r *Runtime) Initialize(ctx context.Context, config *agent.RuntimeConfig) error {
	r.initialized = true
	return nil
}

// Execute runs a single-turn OpenCode session
func (r *Runtime) Execute(ctx context.Context, req *agent.ExecuteRequest) (*agent.ExecuteResponse, error) {
	if !r.initialized {
		return nil, fmt.Errorf("runtime not initialized")
	}

	// Ensure server is running in container
	server, err := r.ensureServer(ctx, req.ContainerID, req.WorkingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to start OpenCode server: %w", err)
	}

	// Create session
	sessionID, err := server.CreateSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Combine prompts
	prompt := req.Prompt
	if req.SystemPrompt != "" {
		prompt = fmt.Sprintf("%s\n\n---\n\n%s", req.SystemPrompt, req.Prompt)
	}

	// Send message and wait for completion
	result, err := server.SendMessage(ctx, sessionID, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	return &agent.ExecuteResponse{
		SessionID: sessionID,
		Result:    result,
	}, nil
}

// ExecuteStreaming starts a bidirectional streaming OpenCode session
func (r *Runtime) ExecuteStreaming(ctx context.Context, req *agent.ExecuteRequest) (agent.StreamingExecutor, error) {
	if !r.initialized {
		return nil, fmt.Errorf("runtime not initialized")
	}

	// Ensure server is running in container
	server, err := r.ensureServer(ctx, req.ContainerID, req.WorkingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to start OpenCode server: %w", err)
	}

	// Create or resume session
	var sessionID string
	if req.SessionID != "" {
		sessionID = req.SessionID
	} else {
		sessionID, err = server.CreateSession(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create session: %w", err)
		}
	}

	// Create streaming executor with model
	// Model should be in "providerID/modelID" format (e.g., "anthropic/claude-sonnet-4-5")
	executor, err := NewStreamingExecutor(ctx, server, sessionID, req.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}

	// Send initial prompt if provided
	prompt := req.Prompt
	if req.SystemPrompt != "" {
		prompt = fmt.Sprintf("%s\n\n---\n\n%s", req.SystemPrompt, req.Prompt)
	}
	if prompt != "" {
		if err := executor.SendMessage(prompt); err != nil {
			_ = executor.Close()
			return nil, fmt.Errorf("failed to send initial message: %w", err)
		}
	}

	return executor, nil
}

// Ping checks if the runtime is available
func (r *Runtime) Ping(ctx context.Context) error {
	// OpenCode is always available (no external API key needed)
	return nil
}

// Close releases runtime resources
func (r *Runtime) Close() error {
	r.serversMu.Lock()
	defer r.serversMu.Unlock()

	for _, server := range r.servers {
		server.Stop()
	}
	r.servers = make(map[string]*Server)
	r.initialized = false
	return nil
}

// Name returns the runtime identifier
func (r *Runtime) Name() string {
	return "opencode"
}

// IsAvailable returns whether the runtime can be used
func (r *Runtime) IsAvailable() bool {
	// OpenCode is available if runtime is initialized
	return r.initialized
}

// ensureServer ensures an OpenCode server is running in the container
func (r *Runtime) ensureServer(ctx context.Context, containerID, workingDir string) (*Server, error) {
	r.serversMu.Lock()
	defer r.serversMu.Unlock()

	if server, ok := r.servers[containerID]; ok {
		if server.IsRunning() {
			return server, nil
		}
		// Server died, remove it
		delete(r.servers, containerID)
	}

	// Start new server
	server := NewServer(r.containerRuntime, containerID, workingDir)
	if err := server.Start(ctx); err != nil {
		return nil, err
	}

	r.servers[containerID] = server
	return server, nil
}

// StopServer stops the OpenCode server for a container
func (r *Runtime) StopServer(containerID string) {
	r.serversMu.Lock()
	defer r.serversMu.Unlock()

	if server, ok := r.servers[containerID]; ok {
		server.Stop()
		delete(r.servers, containerID)
	}
}
