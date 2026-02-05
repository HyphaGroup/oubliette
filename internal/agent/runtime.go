// Package agent provides the agent runtime abstraction layer.
//
// runtime.go - Runtime interface definition
//
// This file contains:
// - Runtime interface for agent execution backends
// - RuntimeConfig for runtime initialization
// - ExecuteResponse for single-turn execution results
//
// See AGENTS.md for documentation on implementing new runtimes.

package agent

import "context"

// Runtime is the interface for agent execution backends (Droid, OpenCode, etc.)
type Runtime interface {
	// Initialize prepares the runtime with configuration
	Initialize(ctx context.Context, config *RuntimeConfig) error

	// ExecuteStreaming starts a bidirectional streaming session
	ExecuteStreaming(ctx context.Context, request *ExecuteRequest) (StreamingExecutor, error)

	// Execute runs a single-turn execution (blocking)
	Execute(ctx context.Context, request *ExecuteRequest) (*ExecuteResponse, error)

	// Ping checks if the runtime is available and responsive
	Ping(ctx context.Context) error

	// Close releases any resources held by the runtime
	Close() error

	// Name returns the runtime identifier (e.g., "droid", "opencode")
	Name() string

	// IsAvailable returns whether the runtime can be used
	IsAvailable() bool
}

// RuntimeConfig holds configuration for initializing a runtime
type RuntimeConfig struct {
	// DefaultModel is the model to use when not specified
	DefaultModel string

	// DefaultAutonomy is the autonomy level when not specified
	DefaultAutonomy string

	// APIKey is the API key for the runtime (e.g., Factory API key for Droid)
	APIKey string

	// Additional runtime-specific config
	Extra map[string]interface{}
}

// ExecuteResponse contains output from single-turn execution
type ExecuteResponse struct {
	SessionID    string
	Result       string
	InputTokens  int
	OutputTokens int
	DurationMs   int
	NumTurns     int
}
