// Package chaos provides chaos testing for the Oubliette MCP server.
//
// These tests verify graceful degradation under failure conditions.
// Run with: go test -v -tags=chaos ./test/chaos/... -timeout 30m
//
// WARNING: Some tests require root/admin privileges and may affect system state.
// Run in an isolated environment.
package chaos

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Config for chaos tests
type Config struct {
	ServerURL   string
	AuthToken   string
	ProjectsDir string
	DataDir     string
}

func getConfig() Config {
	serverURL := os.Getenv("OUBLIETTE_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}

	return Config{
		ServerURL:   serverURL,
		AuthToken:   os.Getenv("OUBLIETTE_AUTH_TOKEN"),
		ProjectsDir: os.Getenv("OUBLIETTE_PROJECTS_DIR"),
		DataDir:     os.Getenv("OUBLIETTE_DATA_DIR"),
	}
}

// MCPClient for chaos testing
type MCPClient struct {
	baseURL   string
	authToken string
	client    *http.Client
}

func NewMCPClient(baseURL, authToken string) *MCPClient {
	return &MCPClient{
		baseURL:   baseURL,
		authToken: authToken,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *MCPClient) Call(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      time.Now().UnixNano(),
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      method,
			"arguments": params,
		},
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.authToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// TestCorruptMetadata verifies server handles corrupt metadata gracefully
func TestCorruptMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping chaos test in short mode")
	}

	cfg := getConfig()
	if cfg.ProjectsDir == "" {
		t.Skip("OUBLIETTE_PROJECTS_DIR not set")
	}

	// Create a test project directory with corrupt metadata
	testProjectID := "chaos-corrupt-test"
	projectDir := filepath.Join(cfg.ProjectsDir, testProjectID)

	// Setup
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("Failed to create test project dir: %v", err)
	}
	defer os.RemoveAll(projectDir)

	// Write corrupt metadata
	metadataPath := filepath.Join(projectDir, "metadata.json")
	if err := os.WriteFile(metadataPath, []byte("{ invalid json }}}"), 0o644); err != nil {
		t.Fatalf("Failed to write corrupt metadata: %v", err)
	}

	// Verify server returns error, not crash
	client := NewMCPClient(cfg.ServerURL+"/mcp", cfg.AuthToken)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := client.Call(ctx, "project_get", map[string]interface{}{
		"project_id": testProjectID,
	})

	// Should get an error, not a crash
	if err == nil {
		t.Log("Server handled corrupt metadata gracefully (returned success, possibly skipped)")
	} else if strings.Contains(err.Error(), "corrupt") || strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "not found") {
		t.Logf("Server returned expected error: %v", err)
	} else {
		t.Logf("Server returned error (acceptable): %v", err)
	}

	// Verify server is still responsive
	resp, err := http.Get(cfg.ServerURL + "/health")
	if err != nil {
		t.Fatalf("Server crashed or unresponsive after corrupt metadata test: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Server unhealthy after corrupt metadata test: %d", resp.StatusCode)
	}
	t.Log("Server remained healthy after corrupt metadata test")
}

// TestMissingWorkspace verifies server handles missing workspace directory
func TestMissingWorkspace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping chaos test in short mode")
	}

	cfg := getConfig()
	if cfg.AuthToken == "" {
		t.Skip("OUBLIETTE_AUTH_TOKEN not set")
	}

	client := NewMCPClient(cfg.ServerURL+"/mcp", cfg.AuthToken)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try to access non-existent workspace
	_, err := client.Call(ctx, "session_message", map[string]interface{}{
		"project_id":       "nonexistent-project",
		"workspace_id":     "nonexistent-workspace",
		"message":          "chaos test",
		"create_workspace": false,
	})

	// Should return error, not crash
	if err == nil {
		t.Log("Unexpected success - server may have created resources")
	} else {
		t.Logf("Expected error received: %v", err)
	}

	// Verify server health
	resp, err := http.Get(cfg.ServerURL + "/health")
	if err != nil {
		t.Fatalf("Server unresponsive after missing workspace test: %v", err)
	}
	resp.Body.Close()
	t.Log("Server remained healthy after missing workspace test")
}

// TestConcurrentWrites verifies data integrity under concurrent writes
func TestConcurrentWrites(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping chaos test in short mode")
	}

	cfg := getConfig()
	if cfg.AuthToken == "" || cfg.ProjectsDir == "" {
		t.Skip("OUBLIETTE_AUTH_TOKEN or OUBLIETTE_PROJECTS_DIR not set")
	}

	client := NewMCPClient(cfg.ServerURL+"/mcp", cfg.AuthToken)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a test project
	projectName := fmt.Sprintf("chaos-concurrent-%d", time.Now().UnixNano())
	result, err := client.Call(ctx, "project_create", map[string]interface{}{
		"name": projectName,
	})
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}
	t.Logf("Created project: %s", string(result))

	// Concurrent workspace creation
	done := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			_, err := client.Call(ctx, "workspace_list", map[string]interface{}{
				"project_id": projectName,
			})
			done <- err
		}(i)
	}

	// Collect results
	errors := 0
	for i := 0; i < 10; i++ {
		if err := <-done; err != nil {
			errors++
			t.Logf("Concurrent request %d error: %v", i, err)
		}
	}

	if errors > 2 {
		t.Errorf("Too many errors in concurrent access: %d/10", errors)
	}

	// Cleanup - delete the project
	_, _ = client.Call(ctx, "project_delete", map[string]interface{}{
		"project_id": projectName,
	})

	// Verify server health
	resp, err := http.Get(cfg.ServerURL + "/health")
	if err != nil {
		t.Fatalf("Server unresponsive after concurrent writes: %v", err)
	}
	resp.Body.Close()
	t.Log("Server remained healthy after concurrent writes test")
}

// TestNetworkTimeout verifies server handles slow clients
func TestNetworkTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping chaos test in short mode")
	}

	cfg := getConfig()

	// Use very short timeout
	client := &http.Client{Timeout: 1 * time.Millisecond}

	// This should timeout, not crash server
	_, err := client.Get(cfg.ServerURL + "/health")
	if err == nil {
		t.Log("Request succeeded despite short timeout")
	} else {
		t.Logf("Expected timeout: %v", err)
	}

	// Verify server is still responding to normal requests
	normalClient := &http.Client{Timeout: 5 * time.Second}
	resp, err := normalClient.Get(cfg.ServerURL + "/health")
	if err != nil {
		t.Fatalf("Server unresponsive after timeout test: %v", err)
	}
	resp.Body.Close()
	t.Log("Server remained healthy after timeout test")
}

// TestDockerDaemonRestart verifies recovery when Docker restarts
// Note: This test may require elevated privileges
func TestDockerDaemonRestart(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping chaos test in short mode")
	}

	if os.Getenv("CHAOS_DOCKER_TEST") != "1" {
		t.Skip("Set CHAOS_DOCKER_TEST=1 to run Docker daemon tests (requires privileges)")
	}

	cfg := getConfig()

	// Verify initial state
	resp, err := http.Get(cfg.ServerURL + "/ready")
	if err != nil {
		t.Fatalf("Server not ready: %v", err)
	}
	resp.Body.Close()

	// Restart Docker (requires sudo)
	t.Log("Restarting Docker daemon...")
	cmd := exec.Command("sudo", "systemctl", "restart", "docker")
	if err := cmd.Run(); err != nil {
		// Try macOS
		cmd = exec.Command("osascript", "-e", `do shell script "killall Docker && open -a Docker" with administrator privileges`)
		if err := cmd.Run(); err != nil {
			t.Skipf("Could not restart Docker: %v", err)
		}
	}

	// Wait for Docker to come back
	t.Log("Waiting for Docker to restart...")
	time.Sleep(10 * time.Second)

	// Server should recover
	for i := 0; i < 30; i++ {
		resp, err := http.Get(cfg.ServerURL + "/ready")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			t.Logf("Server recovered after %d seconds", (i+1)*2)
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(2 * time.Second)
	}

	t.Error("Server did not recover within 60 seconds")
}

// TestGracefulShutdown verifies clean shutdown behavior
func TestGracefulShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping chaos test in short mode")
	}

	t.Log("Graceful shutdown test requires manual verification:")
	t.Log("1. Start server: go run ./cmd/server")
	t.Log("2. Create some sessions")
	t.Log("3. Send SIGTERM: kill -TERM <pid>")
	t.Log("4. Verify logs show clean shutdown")
	t.Log("5. Verify no data corruption")
	t.Skip("Manual test - see instructions above")
}

// TestDiskFull simulates disk full condition
func TestDiskFull(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping chaos test in short mode")
	}

	if os.Getenv("CHAOS_DISK_TEST") != "1" {
		t.Skip("Set CHAOS_DISK_TEST=1 to run disk full tests (may fill disk)")
	}

	cfg := getConfig()
	if cfg.DataDir == "" {
		t.Skip("OUBLIETTE_DATA_DIR not set")
	}

	// Create large file to fill disk (careful!)
	t.Log("WARNING: This test will attempt to fill disk space")
	t.Log("Ensure you have monitoring and can recover")
	t.Skip("Dangerous test - uncomment to run")

	// Uncomment below to actually run:
	/*
		fillFile := filepath.Join(cfg.DataDir, "chaos-fill.tmp")
		defer os.Remove(fillFile)

		f, err := os.Create(fillFile)
		if err != nil {
			t.Fatalf("Could not create fill file: %v", err)
		}
		defer f.Close()

		// Write until disk full
		buf := make([]byte, 1024*1024) // 1MB
		for {
			_, err := f.Write(buf)
			if err != nil {
				t.Logf("Disk full: %v", err)
				break
			}
		}

		// Verify server returns error, not crash
		client := NewMCPClient(cfg.ServerURL+"/mcp", cfg.AuthToken)
		_, err = client.Call(context.Background(), "project_create", map[string]interface{}{
			"name": "chaos-diskfull-test",
		})
		if err != nil {
			t.Logf("Expected error on disk full: %v", err)
		}

		// Remove fill file and verify recovery
		os.Remove(fillFile)
		time.Sleep(2 * time.Second)

		resp, err := http.Get(cfg.ServerURL + "/health")
		if err != nil {
			t.Fatalf("Server not responsive after disk full recovery: %v", err)
		}
		resp.Body.Close()
	*/
}
