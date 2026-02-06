// Package agent provides the agent runtime abstraction layer.
//
// runtime.go - Runtime interface definition
//
// This file contains:
// - Runtime interface for agent execution backends
// - ExecuteResponse for single-turn execution results

package agent

import "context"

// Runtime is the interface for agent execution backends
type Runtime interface {
	// ExecuteStreaming starts a bidirectional streaming session
	ExecuteStreaming(ctx context.Context, request *ExecuteRequest) (StreamingExecutor, error)

	// Execute runs a single-turn execution (blocking)
	Execute(ctx context.Context, request *ExecuteRequest) (*ExecuteResponse, error)

	// Ping checks if the runtime is available and responsive
	Ping(ctx context.Context) error

	// Close releases any resources held by the runtime
	Close() error
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
