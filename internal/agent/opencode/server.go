// Package opencode provides the OpenCode agent runtime.
//
// server.go - OpenCode server lifecycle management
//
// This file contains:
// - Server struct managing the OpenCode process inside a container
// - Server lifecycle (Start, Stop, IsRunning)
// - Health checking (waitForHealth, checkHealth)
// - Session creation (CreateSession)
//
// The OpenCode server runs as a background process (`opencode serve`)
// inside the container, listening on port 4096.

package opencode

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/HyphaGroup/oubliette/internal/container"
)

const (
	serverPort         = 4096
	serverStartTimeout = 30 * time.Second
	healthCheckRetries = 30
	healthCheckDelay   = time.Second
)

// Server manages the OpenCode server lifecycle inside a container
type Server struct {
	containerRuntime container.Runtime
	containerID      string
	workingDir       string

	mu      sync.RWMutex
	running bool
	stopCh  chan struct{}
}

// NewServer creates a new OpenCode server manager
func NewServer(containerRuntime container.Runtime, containerID, workingDir string) *Server {
	return &Server{
		containerRuntime: containerRuntime,
		containerID:      containerID,
		workingDir:       workingDir,
		stopCh:           make(chan struct{}),
	}
}

// Start launches the OpenCode server in the container
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	// Start opencode serve in background using nohup
	// Set XDG_CACHE_HOME to avoid permission issues with /home/gogol/.cache
	cmd := fmt.Sprintf("export XDG_CACHE_HOME=/tmp/opencode-cache && mkdir -p /tmp/opencode-cache && nohup opencode serve --port %d --hostname 127.0.0.1 > /tmp/opencode.log 2>&1 &", serverPort)
	execConfig := container.ExecConfig{
		Cmd:          []string{"sh", "-c", cmd},
		WorkingDir:   s.workingDir,
		AttachStdout: true,
		AttachStderr: true,
	}

	_, err := s.containerRuntime.Exec(ctx, s.containerID, execConfig)
	if err != nil {
		return fmt.Errorf("failed to start OpenCode server: %w", err)
	}

	// Wait for server to become healthy
	if err := s.waitForHealth(ctx); err != nil {
		return fmt.Errorf("OpenCode server failed to start: %w", err)
	}

	s.running = true
	return nil
}

// waitForHealth polls the health endpoint until server is ready
func (s *Server) waitForHealth(ctx context.Context) error {
	deadline := time.Now().Add(serverStartTimeout)

	for i := 0; i < healthCheckRetries; i++ {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for server")
		}

		// Check health via exec curl
		healthy, err := s.checkHealth(ctx)
		if err == nil && healthy {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(healthCheckDelay):
		}
	}

	return fmt.Errorf("server did not become healthy after %d retries", healthCheckRetries)
}

// checkHealth checks if the OpenCode server is responding
func (s *Server) checkHealth(ctx context.Context) (bool, error) {
	cmd := fmt.Sprintf("curl -sf http://127.0.0.1:%d/global/health", serverPort)
	execConfig := container.ExecConfig{
		Cmd:          []string{"sh", "-c", cmd},
		AttachStdout: true,
		AttachStderr: true,
	}

	result, err := s.containerRuntime.Exec(ctx, s.containerID, execConfig)
	if err != nil {
		return false, err
	}

	return result.ExitCode == 0, nil
}

// Stop terminates the OpenCode server
func (s *Server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	// Signal stop
	close(s.stopCh)

	// Kill the opencode process
	cmd := "pkill -f 'opencode serve' || true"
	execConfig := container.ExecConfig{
		Cmd:          []string{"sh", "-c", cmd},
		AttachStdout: true,
		AttachStderr: true,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = s.containerRuntime.Exec(ctx, s.containerID, execConfig)

	s.running = false
	s.stopCh = make(chan struct{})
}

// IsRunning returns whether the server is running
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// CreateSession creates a new OpenCode session
func (s *Server) CreateSession(ctx context.Context) (string, error) {
	resp, err := s.doRequest(ctx, "POST", "/session", nil)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create session failed: %s", string(body))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode session response: %w", err)
	}

	return result.ID, nil
}
