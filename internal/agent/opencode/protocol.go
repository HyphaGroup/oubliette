// Package opencode provides the OpenCode agent runtime.
//
// protocol.go - HTTP communication layer
//
// This file contains:
// - HTTP client methods for OpenCode REST API (doRequest)
// - Message sending (SendMessage, SendMessageAsync)
// - SSE event subscription (SubscribeEvents)
// - SSE stream reader (sseReader)
//
// OpenCode uses HTTP REST for commands and SSE for event streaming.
// All HTTP requests are executed via curl inside the container.

package opencode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/HyphaGroup/oubliette/internal/container"
)

// SendMessage sends a message to a session and returns the response (synchronous)
func (s *Server) SendMessage(ctx context.Context, sessionID, message string) (string, error) {
	body := map[string]interface{}{
		"parts": []map[string]string{
			{"type": "text", "text": message},
		},
	}
	jsonBody, _ := json.Marshal(body)

	resp, err := s.doRequest(ctx, "POST", fmt.Sprintf("/session/%s/message", sessionID), bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("send message failed: %s", string(respBody))
	}

	// Read streaming response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response - extract text from parts
	var result struct {
		Info struct {
			ID string `json:"id"`
		} `json:"info"`
		Parts []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"parts"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return string(respBody), nil // Return raw response if parse fails
	}

	var texts []string
	for _, part := range result.Parts {
		if part.Type == "text" && part.Text != "" {
			texts = append(texts, part.Text)
		}
	}

	return strings.Join(texts, "\n"), nil
}

// SendMessageAsync sends a message asynchronously (returns immediately, events via SSE)
// Uses the /session/:id/prompt_async endpoint
// model format: "providerID/modelID" (e.g., "anthropic/claude-sonnet-4-5")
// variant maps to OpenCode's reasoning variant ("low", "medium", "high", or "" for none)
func (s *Server) SendMessageAsync(ctx context.Context, sessionID, message, model, variant string) error {
	body := map[string]interface{}{
		"parts": []map[string]string{
			{"type": "text", "text": message},
		},
	}

	// Include model if specified (format: "providerID/modelID")
	if model != "" {
		parts := strings.SplitN(model, "/", 2)
		if len(parts) == 2 {
			body["model"] = map[string]string{
				"providerID": parts[0],
				"modelID":    parts[1],
			}
		}
	}

	// Include variant for reasoning level
	if variant != "" && variant != "off" {
		body["variant"] = variant
	}

	jsonBody, _ := json.Marshal(body)

	resp, err := s.doRequest(ctx, "POST", fmt.Sprintf("/session/%s/prompt_async", sessionID), bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	// prompt_async returns 204 No Content on success
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("send message async failed: %s", string(respBody))
	}

	return nil
}

// AbortSession sends an abort request to stop the current operation
func (s *Server) AbortSession(ctx context.Context, sessionID string) error {
	resp, err := s.doRequest(ctx, "POST", fmt.Sprintf("/session/%s/abort", sessionID), nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	return nil
}

// SubscribeEvents connects to the SSE event stream using interactive exec
// Returns a reader that streams SSE events incrementally
func (s *Server) SubscribeEvents(ctx context.Context) (io.ReadCloser, error) {
	url := fmt.Sprintf("http://127.0.0.1:%d/event", serverPort)
	curlCmd := fmt.Sprintf("curl -sN '%s'", url) // -N disables buffering

	execConfig := container.ExecConfig{
		Cmd:          []string{"sh", "-c", curlCmd},
		WorkingDir:   s.workingDir,
		AttachStdout: true,
		AttachStderr: true,
	}

	// Use ExecInteractive for streaming - this returns immediately with pipes
	interactive, err := s.containerRuntime.ExecInteractive(ctx, s.containerID, execConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to start SSE stream: %w", err)
	}

	// Close stdin since we don't need to send anything
	_ = interactive.Stdin.Close()

	// Return stdout as the SSE reader (wrap to also close the interactive exec on close)
	return &sseReader{
		reader:      interactive.Stdout,
		interactive: interactive,
	}, nil
}

// sseReader wraps the interactive exec stdout and handles cleanup
type sseReader struct {
	reader      io.Reader
	interactive *container.InteractiveExec
}

func (r *sseReader) Read(p []byte) (n int, err error) {
	return r.reader.Read(p)
}

func (r *sseReader) Close() error {
	// Signal to close and cleanup
	if r.interactive != nil {
		_ = r.interactive.Stdin.Close()
	}
	return nil
}

// doRequest executes an HTTP request via exec curl in the container
func (s *Server) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := fmt.Sprintf("http://127.0.0.1:%d%s", serverPort, path)

	var curlCmd string
	if body != nil {
		bodyBytes, err := io.ReadAll(body)
		if err != nil {
			return nil, err
		}
		// Escape single quotes in body for shell
		escapedBody := strings.ReplaceAll(string(bodyBytes), "'", "'\\''")
		curlCmd = fmt.Sprintf("curl -s -X %s -H 'Content-Type: application/json' -d '%s' '%s'", method, escapedBody, url)
	} else {
		curlCmd = fmt.Sprintf("curl -s -X %s '%s'", method, url)
	}

	execConfig := container.ExecConfig{
		Cmd:          []string{"sh", "-c", curlCmd},
		WorkingDir:   s.workingDir,
		AttachStdout: true,
		AttachStderr: true,
	}

	result, err := s.containerRuntime.Exec(ctx, s.containerID, execConfig)
	if err != nil {
		return nil, fmt.Errorf("curl exec failed: %w", err)
	}

	// Create a fake http.Response from curl output
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(result.Stdout)),
	}

	// Check for error in output
	if strings.Contains(result.Stdout, `"name":`) && strings.Contains(result.Stdout, `"data":`) {
		// Might be an error response
		var errResp struct {
			Name string `json:"name"`
		}
		if json.Unmarshal([]byte(result.Stdout), &errResp) == nil && errResp.Name != "" {
			resp.StatusCode = http.StatusBadRequest
		}
	}

	return resp, nil
}
