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

// Execute runs a single-turn OpenCode session
func (r *Runtime) Execute(ctx context.Context, req *agent.ExecuteRequest) (*agent.ExecuteResponse, error) {
	server, err := r.ensureServer(ctx, req.ContainerID, req.WorkingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to start OpenCode server: %w", err)
	}

	sessionID, err := server.CreateSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	prompt := req.Prompt
	if req.SystemPrompt != "" {
		prompt = fmt.Sprintf("%s\n\n---\n\n%s", req.SystemPrompt, req.Prompt)
	}

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
	server, err := r.ensureServer(ctx, req.ContainerID, req.WorkingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to start OpenCode server: %w", err)
	}

	var sessionID string
	if req.SessionID != "" {
		sessionID = req.SessionID
	} else {
		sessionID, err = server.CreateSession(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create session: %w", err)
		}
	}

	executor, err := NewStreamingExecutor(ctx, server, sessionID, req.Model, req.ReasoningLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}

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
	return nil
}

// ensureServer ensures an OpenCode server is running in the container
func (r *Runtime) ensureServer(ctx context.Context, containerID, workingDir string) (*Server, error) {
	r.serversMu.Lock()
	defer r.serversMu.Unlock()

	if server, ok := r.servers[containerID]; ok {
		if server.IsRunning() {
			return server, nil
		}
		delete(r.servers, containerID)
	}

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
