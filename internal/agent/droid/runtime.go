package droid

import (
	"context"
	"fmt"
	"os"

	"github.com/HyphaGroup/oubliette/internal/agent"
	"github.com/HyphaGroup/oubliette/internal/container"
)

// Runtime implements agent.Runtime for Factory Droid
type Runtime struct {
	containerRuntime container.Runtime
	defaultModel     string
	defaultAutonomy  string
	apiKey           string
	initialized      bool
}

// Ensure Runtime implements agent.Runtime
var _ agent.Runtime = (*Runtime)(nil)

// NewRuntime creates a new Droid runtime
func NewRuntime(containerRuntime container.Runtime) *Runtime {
	return &Runtime{
		containerRuntime: containerRuntime,
	}
}

// Initialize prepares the runtime with configuration
func (r *Runtime) Initialize(ctx context.Context, config *agent.RuntimeConfig) error {
	if config == nil {
		config = &agent.RuntimeConfig{}
	}

	r.defaultModel = config.DefaultModel
	if r.defaultModel == "" {
		r.defaultModel = "claude-opus-4-5-20251101"
	}

	r.defaultAutonomy = config.DefaultAutonomy
	if r.defaultAutonomy == "" {
		r.defaultAutonomy = "skip-permissions-unsafe"
	}

	// Try config API key first, then fall back to environment
	r.apiKey = config.APIKey
	if r.apiKey == "" {
		r.apiKey = os.Getenv("FACTORY_API_KEY")
	}

	r.initialized = true
	return nil
}

// Execute runs a single-turn Droid session
func (r *Runtime) Execute(ctx context.Context, req *agent.ExecuteRequest) (*agent.ExecuteResponse, error) {
	if !r.initialized {
		return nil, fmt.Errorf("runtime not initialized")
	}

	// Build droid exec command
	cmd := r.buildCommand(req)

	if r.apiKey == "" {
		return nil, fmt.Errorf("FACTORY_API_KEY not configured")
	}

	// Build env vars including session identity for MCP bridge
	env := []string{
		fmt.Sprintf("FACTORY_API_KEY=%s", r.apiKey),
	}
	if req.SessionID != "" {
		env = append(env, fmt.Sprintf("OUBLIETTE_SESSION_ID=%s", req.SessionID))
	}
	if req.ProjectID != "" {
		env = append(env, fmt.Sprintf("OUBLIETTE_PROJECT_ID=%s", req.ProjectID))
	}
	env = append(env, fmt.Sprintf("OUBLIETTE_DEPTH=%d", req.Depth))

	// Create exec configuration using container abstraction
	execConfig := container.ExecConfig{
		Cmd:          []string{"sh", "-c", cmd},
		WorkingDir:   req.WorkingDir,
		AttachStdout: true,
		AttachStderr: true,
		Env:          env,
	}

	// Execute command in container
	result, err := r.containerRuntime.Exec(ctx, req.ContainerID, execConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to exec in container: %w", err)
	}

	if result.ExitCode != 0 {
		return nil, fmt.Errorf("droid exec failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}

	// Parse JSON output
	return r.parseOutput([]byte(result.Stdout))
}

// ExecuteStreaming starts a bidirectional streaming Droid session
func (r *Runtime) ExecuteStreaming(ctx context.Context, req *agent.ExecuteRequest) (agent.StreamingExecutor, error) {
	if !r.initialized {
		return nil, fmt.Errorf("runtime not initialized")
	}

	// Force stream-jsonrpc mode
	req.StreamJSONRPC = true

	// Build droid exec command
	cmd := r.buildCommand(req)

	if r.apiKey == "" {
		return nil, fmt.Errorf("FACTORY_API_KEY not configured")
	}

	// Build env vars including session identity for MCP bridge
	env := []string{
		fmt.Sprintf("FACTORY_API_KEY=%s", r.apiKey),
	}
	if req.SessionID != "" {
		env = append(env, fmt.Sprintf("OUBLIETTE_SESSION_ID=%s", req.SessionID))
	}
	if req.ProjectID != "" {
		env = append(env, fmt.Sprintf("OUBLIETTE_PROJECT_ID=%s", req.ProjectID))
	}
	env = append(env, fmt.Sprintf("OUBLIETTE_DEPTH=%d", req.Depth))

	// Create exec configuration for interactive execution
	execConfig := container.ExecConfig{
		Cmd:          []string{"sh", "-c", cmd},
		WorkingDir:   req.WorkingDir,
		AttachStdout: true,
		AttachStderr: true,
		AttachStdin:  true,
		Env:          env,
	}

	// Start interactive exec
	interactiveExec, err := r.containerRuntime.ExecInteractive(ctx, req.ContainerID, execConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to start interactive exec: %w", err)
	}

	// Create streaming executor
	executor := NewStreamingExecutor(ctx, interactiveExec)

	// Combine system and user prompts
	prompt := req.Prompt
	if req.SystemPrompt != "" {
		prompt = fmt.Sprintf("%s\n\n---\n\n%s", req.SystemPrompt, req.Prompt)
	}

	// Send initialize_session request (without prompt - that comes via add_user_message)
	initReq := NewInitializeSessionRequest("", req.WorkingDir, req.ContainerID)
	if err := executor.sendRequest(initReq); err != nil {
		_ = executor.Close()
		return nil, fmt.Errorf("failed to send initialize_session request: %w", err)
	}

	// Wait for initialize_session response before sending user message
	if err := executor.WaitForInit(ctx); err != nil {
		_ = executor.Close()
		return nil, fmt.Errorf("failed to initialize session: %w", err)
	}

	// Now send the user message with the prompt
	if prompt != "" {
		if err := executor.SendMessage(prompt); err != nil {
			_ = executor.Close()
			return nil, fmt.Errorf("failed to send user message: %w", err)
		}
	}

	return executor, nil
}

// Ping checks if the runtime is available
func (r *Runtime) Ping(ctx context.Context) error {
	if r.apiKey == "" {
		return fmt.Errorf("FACTORY_API_KEY not configured")
	}
	// Could add a health check to Factory API here
	return nil
}

// Close releases runtime resources
func (r *Runtime) Close() error {
	r.initialized = false
	return nil
}

// Name returns the runtime identifier
func (r *Runtime) Name() string {
	return "droid"
}

// IsAvailable returns whether the runtime can be used
func (r *Runtime) IsAvailable() bool {
	return r.apiKey != ""
}
