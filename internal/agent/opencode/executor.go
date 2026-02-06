// Package opencode provides the OpenCode agent runtime.
//
// executor.go - StreamingExecutor implementation
//
// This file contains:
// - StreamingExecutor struct implementing agent.StreamingExecutor
// - Message sending via async HTTP (SendMessage)
// - SSE event stream processing (processEvents)
// - Event parsing and normalization to agent.StreamEvent
//
// The executor subscribes to the OpenCode SSE event stream and converts
// events to the normalized agent.StreamEvent format. Messages are sent
// via the async HTTP endpoint, with responses arriving via SSE.

package opencode

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/HyphaGroup/oubliette/internal/agent"
)

// StreamingExecutor implements agent.StreamingExecutor for OpenCode
type StreamingExecutor struct {
	server         *Server
	sessionID      string
	model          string // Model in "providerID/modelID" format
	reasoningLevel string // Reasoning level passed as variant to OpenCode

	ctx      context.Context
	cancel   context.CancelFunc
	eventsCh chan *agent.StreamEvent
	errorsCh chan error
	doneCh   chan struct{}

	mu        sync.RWMutex
	closed    bool
	eventConn io.ReadCloser
	exitCode  int
}

// Ensure StreamingExecutor implements agent.StreamingExecutor
var _ agent.StreamingExecutor = (*StreamingExecutor)(nil)

// NewStreamingExecutor creates a new streaming executor
// model should be in "providerID/modelID" format (e.g., "anthropic/claude-sonnet-4-5")
// reasoningLevel maps to OpenCode's variant parameter ("low", "medium", "high", or "")
func NewStreamingExecutor(ctx context.Context, server *Server, sessionID, model, reasoningLevel string) (*StreamingExecutor, error) {
	ctx, cancel := context.WithCancel(ctx)

	e := &StreamingExecutor{
		server:         server,
		sessionID:      sessionID,
		model:          model,
		reasoningLevel: reasoningLevel,
		ctx:            ctx,
		cancel:         cancel,
		eventsCh:       make(chan *agent.StreamEvent, 100),
		errorsCh:       make(chan error, 10),
		doneCh:         make(chan struct{}),
	}

	// Subscribe to event stream
	eventConn, err := server.SubscribeEvents(ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to subscribe to events: %w", err)
	}
	e.eventConn = eventConn

	// Start event processing goroutine
	go e.processEvents()

	return e, nil
}

// SendMessage sends a user message to the session
func (e *StreamingExecutor) SendMessage(message string) error {
	e.mu.RLock()
	if e.closed {
		e.mu.RUnlock()
		return fmt.Errorf("executor is closed")
	}
	e.mu.RUnlock()

	// Send message via async endpoint (returns immediately, events come via SSE)
	return e.server.SendMessageAsync(e.ctx, e.sessionID, message, e.model, e.reasoningLevel)
}

// Cancel requests termination of the current operation via OpenCode's abort endpoint
func (e *StreamingExecutor) Cancel() error {
	return e.server.AbortSession(e.ctx, e.sessionID)
}

// Events returns a channel for receiving stream events
func (e *StreamingExecutor) Events() <-chan *agent.StreamEvent {
	return e.eventsCh
}

// Errors returns a channel for receiving errors
func (e *StreamingExecutor) Errors() <-chan error {
	return e.errorsCh
}

// Done returns a channel that closes when execution finishes
func (e *StreamingExecutor) Done() <-chan struct{} {
	return e.doneCh
}

// Wait blocks until execution completes and returns exit code
func (e *StreamingExecutor) Wait() (int, error) {
	<-e.doneCh
	return e.exitCode, nil
}

// Close gracefully shuts down the executor
func (e *StreamingExecutor) Close() error {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return nil
	}
	e.closed = true
	e.mu.Unlock()

	e.cancel()

	if e.eventConn != nil {
		_ = e.eventConn.Close()
	}

	return nil
}

// RuntimeSessionID returns the OpenCode session ID
func (e *StreamingExecutor) RuntimeSessionID() string {
	return e.sessionID
}

// IsClosed returns whether the executor has been closed
func (e *StreamingExecutor) IsClosed() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.closed
}

// processEvents reads SSE events and converts them to StreamEvents
func (e *StreamingExecutor) processEvents() {
	defer func() {
		close(e.eventsCh)
		close(e.errorsCh)
		close(e.doneCh)
	}()

	reader := bufio.NewReader(e.eventConn)

	for {
		select {
		case <-e.ctx.Done():
			return
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				e.errorsCh <- fmt.Errorf("error reading events: %w", err)
			}
			return
		}

		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimPrefix(line, "data:")
		data = strings.TrimSpace(data)
		if data == "" {
			continue
		}

		event, err := parseSSEEvent(data)
		if err != nil || event == nil {
			continue
		}

		// Filter for our session
		if event.SessionID != "" && event.SessionID != e.sessionID {
			continue
		}

		// Send event
		select {
		case e.eventsCh <- event:
		case <-e.ctx.Done():
			return
		}

		// Note: StreamEventCompletion means the current turn is complete,
		// NOT that the session is over. The session stays running, waiting
		// for the next message.
	}
}

// parseSSEEvent parses an SSE data payload into a StreamEvent.
// Returns nil for events that carry no useful information (transport noise,
// redundant metadata updates, duplicate completion signals).
func parseSSEEvent(data string) (*agent.StreamEvent, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(data), &raw); err != nil {
		return nil, err
	}

	eventType, _ := raw["type"].(string)
	props, _ := raw["properties"].(map[string]interface{})

	switch eventType {
	case "message.updated":
		// Metadata-only (role/id) -- fires on every token with no text content.
		// Dropped: the consolidated message already carries role info.
		return nil, nil

	case "message.part.updated":
		part, ok := props["part"].(map[string]interface{})
		if !ok {
			return nil, nil
		}
		partType, _ := part["type"].(string)
		switch partType {
		case "text":
			if delta, ok := props["delta"].(string); ok && delta != "" {
				return &agent.StreamEvent{
					Type: agent.StreamEventDelta,
					Text: delta,
					Raw:  raw,
				}, nil
			}
			// Consolidated full text (no delta prop)
			text, _ := part["text"].(string)
			return &agent.StreamEvent{
				Type: agent.StreamEventMessage,
				Text: text,
				Raw:  raw,
			}, nil
		case "tool-invocation":
			event := &agent.StreamEvent{Type: agent.StreamEventToolCall, Raw: raw}
			event.ToolID, _ = part["id"].(string)
			event.ToolName, _ = part["toolName"].(string)
			if args, ok := part["args"].(map[string]interface{}); ok {
				event.Parameters = args
			}
			return event, nil
		case "tool-result":
			event := &agent.StreamEvent{Type: agent.StreamEventToolResult, Raw: raw}
			event.ToolID, _ = part["id"].(string)
			if result, ok := part["result"].(string); ok {
				event.Value = result
			}
			event.IsError, _ = part["isError"].(bool)
			return event, nil
		case "step-start", "step-finish", "reasoning":
			return &agent.StreamEvent{
				Type:    agent.StreamEventSystem,
				Subtype: partType,
				Raw:     raw,
			}, nil
		default:
			return nil, nil
		}

	case "session.status":
		status, ok := props["status"].(map[string]interface{})
		if !ok {
			return nil, nil
		}
		statusType, _ := status["type"].(string)
		if statusType == "idle" {
			return &agent.StreamEvent{
				Type: agent.StreamEventCompletion,
				Raw:  raw,
			}, nil
		}
		return &agent.StreamEvent{
			Type:    agent.StreamEventSystem,
			Subtype: statusType,
			Raw:     raw,
		}, nil

	case "session.idle":
		// Redundant -- session.status idle already emits completion
		return nil, nil

	case "server.connected", "server.heartbeat":
		// Transport noise
		return nil, nil

	default:
		return &agent.StreamEvent{
			Type:    agent.StreamEventSystem,
			Subtype: eventType,
			Raw:     raw,
		}, nil
	}
}
