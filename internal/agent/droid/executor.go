// Package droid provides the Factory Droid agent runtime.
//
// executor.go - StreamingExecutor implementation
//
// This file contains:
// - StreamingExecutor struct implementing agent.StreamingExecutor
// - Bidirectional message handling (SendMessage, Cancel)
// - Event stream processing from Droid stdout (readEvents)
// - Permission auto-approval for tool calls
//
// The executor manages a long-lived interactive session with the Droid CLI,
// communicating via JSON-RPC over stdin/stdout.

package droid

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/HyphaGroup/oubliette/internal/agent"
	"github.com/HyphaGroup/oubliette/internal/container"
	"github.com/HyphaGroup/oubliette/internal/logger"
)

// StreamingExecutor manages a bidirectional streaming droid execution
// It implements agent.StreamingExecutor
type StreamingExecutor struct {
	exec              *container.InteractiveExec
	eventCh           chan *agent.StreamEvent
	errCh             chan error
	doneCh            chan struct{}
	initDoneCh        chan error
	droidSessionID    string
	lastAssistantText string
	requestID         atomic.Int64
	mu                sync.RWMutex
	closed            bool
	ctx               context.Context
	cancel            context.CancelFunc
}

// Ensure StreamingExecutor implements agent.StreamingExecutor
var _ agent.StreamingExecutor = (*StreamingExecutor)(nil)

// NewStreamingExecutor creates a new streaming executor from an interactive exec
func NewStreamingExecutor(ctx context.Context, exec *container.InteractiveExec) *StreamingExecutor {
	ctx, cancel := context.WithCancel(ctx)
	e := &StreamingExecutor{
		exec:       exec,
		eventCh:    make(chan *agent.StreamEvent, 100),
		errCh:      make(chan error, 1),
		doneCh:     make(chan struct{}),
		initDoneCh: make(chan error, 1),
		ctx:        ctx,
		cancel:     cancel,
	}
	go e.readEvents()
	return e
}

// SendMessage sends a user message to the droid session
func (e *StreamingExecutor) SendMessage(message string) error {
	e.mu.RLock()
	if e.closed {
		e.mu.RUnlock()
		return fmt.Errorf("executor is closed")
	}
	e.mu.RUnlock()

	id := e.requestID.Add(1)
	req := NewUserMessageRequest(message, id)
	return e.sendRequest(req)
}

// Cancel sends a cancel request to terminate the session
func (e *StreamingExecutor) Cancel() error {
	e.mu.RLock()
	if e.closed {
		e.mu.RUnlock()
		return fmt.Errorf("executor is closed")
	}
	e.mu.RUnlock()

	id := e.requestID.Add(1)
	req := NewCancelRequest(id)
	return e.sendRequest(req)
}

// sendRequest sends a JSON-RPC request to the droid stdin
func (e *StreamingExecutor) sendRequest(req *JSONRPCRequest) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	data = append(data, '\n')

	_, err = e.exec.Stdin.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to stdin: %w", err)
	}
	return nil
}

// Events returns the channel for receiving stream events
func (e *StreamingExecutor) Events() <-chan *agent.StreamEvent {
	return e.eventCh
}

// Errors returns the channel for receiving errors
func (e *StreamingExecutor) Errors() <-chan error {
	return e.errCh
}

// Done returns a channel that closes when the executor finishes
func (e *StreamingExecutor) Done() <-chan struct{} {
	return e.doneCh
}

// Close gracefully shuts down the executor by sending interrupt_session
func (e *StreamingExecutor) Close() error {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return nil
	}
	e.closed = true
	e.mu.Unlock()

	// Send interrupt_session for graceful shutdown
	id := e.requestID.Add(1)
	interruptReq := NewCancelRequest(id)
	if err := e.sendRequest(interruptReq); err == nil {
		time.Sleep(100 * time.Millisecond)
	}

	e.cancel()
	_ = e.exec.Close()

	return nil
}

// Wait waits for the execution to complete and returns the exit code
func (e *StreamingExecutor) Wait() (int, error) {
	return e.exec.Wait()
}

// IsClosed returns whether the executor has been closed
func (e *StreamingExecutor) IsClosed() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.closed
}

// RuntimeSessionID returns Factory's session ID (available after WaitForInit)
func (e *StreamingExecutor) RuntimeSessionID() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.droidSessionID
}

// setDroidSessionID stores the session ID from init response
func (e *StreamingExecutor) setDroidSessionID(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.droidSessionID = id
}

// WaitForInit waits for the initialize_session response with a timeout
func (e *StreamingExecutor) WaitForInit(ctx context.Context) error {
	select {
	case err := <-e.initDoneCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(30 * time.Second):
		return fmt.Errorf("timeout waiting for initialize_session response")
	}
}

// readEvents reads JSONL events from stdout and sends them to the event channel
func (e *StreamingExecutor) readEvents() {
	defer close(e.eventCh)
	defer close(e.doneCh)

	scanner := bufio.NewScanner(e.exec.Stdout)
	const maxScanTokenSize = 1024 * 1024
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	initSignaled := false

	for scanner.Scan() {
		select {
		case <-e.ctx.Done():
			return
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Parse as JSON-RPC message
		var rpcMsg struct {
			JSONRPC string      `json:"jsonrpc"`
			Type    string      `json:"type"`
			ID      interface{} `json:"id,omitempty"`
			Method  string      `json:"method,omitempty"`
			Result  interface{} `json:"result,omitempty"`
			Error   *RPCError   `json:"error,omitempty"`
			Params  interface{} `json:"params,omitempty"`
		}
		if err := json.Unmarshal(line, &rpcMsg); err == nil && rpcMsg.JSONRPC == "2.0" {
			if rpcMsg.Type == "response" && !initSignaled {
				if rpcMsg.Error != nil {
					e.initDoneCh <- fmt.Errorf("init error: %s", rpcMsg.Error.Message)
				} else {
					if result, ok := rpcMsg.Result.(map[string]interface{}); ok {
						if sessionID, ok := result["sessionId"].(string); ok {
							e.setDroidSessionID(sessionID)
						}
					}
					e.initDoneCh <- nil
				}
				initSignaled = true
				continue
			}

			// Handle permission requests - auto-approve all
			if rpcMsg.Type == "request" && rpcMsg.Method == "droid.request_permission" {
				toolName := "unknown"
				if params, ok := rpcMsg.Params.(map[string]interface{}); ok {
					if tn, ok := params["toolName"].(string); ok {
						toolName = tn
					}
				}
				logger.Info("Auto-approving permission request for tool: %s (id: %v)", toolName, rpcMsg.ID)

				response := map[string]interface{}{
					"jsonrpc":           "2.0",
					"factoryApiVersion": "1.0.0",
					"type":              "response",
					"id":                rpcMsg.ID,
					"result": map[string]interface{}{
						"selectedOption": "proceed_once",
					},
				}
				respBytes, _ := json.Marshal(response)
				respBytes = append(respBytes, '\n')
				e.mu.Lock()
				_, _ = e.exec.Stdin.Write(respBytes)
				e.mu.Unlock()
				continue
			}

			// Convert to StreamEvent for consistent API
			event := &agent.StreamEvent{
				Timestamp: time.Now().UnixMilli(),
				Raw:       make(map[string]interface{}),
			}
			_ = json.Unmarshal(line, &event.Raw)

			if rpcMsg.Type == "response" {
				if rpcMsg.Result == nil {
					continue
				}
				if resultMap, ok := rpcMsg.Result.(map[string]interface{}); ok && len(resultMap) == 0 {
					continue
				}
				logger.Info("JSON-RPC response with result: %+v", rpcMsg.Result)
				event.Type = agent.StreamEventType("response")
			}

			// Handle notifications with session_notification method
			if rpcMsg.Type == "notification" && rpcMsg.Method == "droid.session_notification" {
				if params, ok := rpcMsg.Params.(map[string]interface{}); ok {
					if notification, ok := params["notification"].(map[string]interface{}); ok {
						if notifType, ok := notification["type"].(string); ok {
							// Set default event type
							event.Type = agent.StreamEventType(notifType)

							switch notifType {
							case "create_message":
								event.Type = agent.StreamEventMessage
								if msg, ok := notification["message"].(map[string]interface{}); ok {
									if role, ok := msg["role"].(string); ok {
										event.Role = role
									}
									if id, ok := msg["id"].(string); ok {
										event.ID = id
									}
									// Extract text from content array - find the text block (not thinking)
									if content, ok := msg["content"].([]interface{}); ok {
										for _, block := range content {
											if textBlock, ok := block.(map[string]interface{}); ok {
												blockType, _ := textBlock["type"].(string)
												if blockType == "text" {
													if text, ok := textBlock["text"].(string); ok {
														event.Text = text
														if event.Role == "assistant" {
															e.mu.Lock()
															e.lastAssistantText = text
															e.mu.Unlock()
														}
														break
													}
												}
											}
										}
									}
								}
							case "assistant_text_delta", "thinking_text_delta":
								// Extract textDelta for streaming text events
								if textDelta, ok := notification["textDelta"].(string); ok {
									event.Text = textDelta
								}
							case "error":
								if message, ok := notification["message"].(string); ok {
									event.Text = message
								}
							case "result", "completion":
								event.Type = agent.StreamEventCompletion
								if finalText, ok := notification["finalText"].(string); ok {
									event.FinalText = finalText
									event.Text = finalText
								}
								if numTurns, ok := notification["numTurns"].(float64); ok {
									event.NumTurns = int(numTurns)
								}
								if durationMs, ok := notification["durationMs"].(float64); ok {
									event.DurationMs = int(durationMs)
								}
							case "droid_working_state_changed":
								if newState, ok := notification["newState"].(string); ok && newState == "idle" {
									e.mu.RLock()
									finalText := e.lastAssistantText
									e.mu.RUnlock()
									if finalText != "" {
										completionEvent := &agent.StreamEvent{
											Type:      agent.StreamEventCompletion,
											Timestamp: time.Now().UnixMilli(),
											FinalText: finalText,
											Text:      finalText,
										}
										select {
										case e.eventCh <- completionEvent:
										case <-e.ctx.Done():
											return
										}
									}
								}
								continue
							}
						}
					}
				}
			}

			if event.Type == "" {
				continue
			}

			select {
			case e.eventCh <- event:
			case <-e.ctx.Done():
				return
			}
		}
	}

	if err := scanner.Err(); err != nil {
		select {
		case e.errCh <- fmt.Errorf("scanner error: %w", err):
		default:
		}
	}

	if !initSignaled {
		select {
		case e.initDoneCh <- fmt.Errorf("stream ended without init response"):
		default:
		}
	}
}
