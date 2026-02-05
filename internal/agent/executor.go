// Package agent provides the agent runtime abstraction layer.
//
// executor.go - StreamingExecutor interface definition
//
// This file contains:
// - StreamingExecutor interface for bidirectional streaming sessions
//
// StreamingExecutor enables real-time communication with agent backends,
// supporting message sending, event streaming, and graceful shutdown.
// See AGENTS.md for documentation on implementing executors.

package agent

// StreamingExecutor manages a bidirectional streaming agent execution
type StreamingExecutor interface {
	// SendMessage sends a user message to the agent session
	SendMessage(message string) error

	// Cancel requests termination of the current operation
	Cancel() error

	// Events returns a channel for receiving stream events
	Events() <-chan *StreamEvent

	// Errors returns a channel for receiving errors
	Errors() <-chan error

	// Done returns a channel that closes when execution finishes
	Done() <-chan struct{}

	// Wait blocks until execution completes and returns exit code
	Wait() (int, error)

	// Close gracefully shuts down the executor
	Close() error

	// RuntimeSessionID returns the backend's session identifier
	// (e.g., Factory's session ID for Droid)
	RuntimeSessionID() string

	// IsClosed returns whether the executor has been closed
	IsClosed() bool
}
