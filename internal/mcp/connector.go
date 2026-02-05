// connector.go provides socket path utilities for relay communication
package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// SocketsBaseDir is the base directory for relay sockets on the host
	SocketsBaseDir = "/tmp/oubliette-sockets"

	// SocketConnectTimeout is how long to wait for socket to appear
	SocketConnectTimeout = 30 * time.Second
)

// SocketPath returns the standardized socket path for a project
func SocketPath(projectID string) string {
	return filepath.Join(SocketsBaseDir, projectID, "relay.sock")
}

// SocketDir returns the socket directory for a project
func SocketDir(projectID string) string {
	return filepath.Join(SocketsBaseDir, projectID)
}

// EnsureSocketDir creates the socket directory for a project.
// It first removes any existing socket directory to clean up stale sockets
// from previous runs that didn't shut down cleanly.
func EnsureSocketDir(projectID string) error {
	dir := SocketDir(projectID)
	// Remove any stale socket directory first
	_ = os.RemoveAll(dir)
	return os.MkdirAll(dir, 0o755)
}

// CleanupSocketDir removes the socket directory for a project
func CleanupSocketDir(projectID string) error {
	dir := SocketDir(projectID)
	return os.RemoveAll(dir)
}

// waitForSocket waits for a unix socket to appear at the given path
func waitForSocket(socketPath string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	checkInterval := 100 * time.Millisecond

	for time.Now().Before(deadline) {
		info, err := os.Stat(socketPath)
		if err == nil && info.Mode()&os.ModeSocket != 0 {
			return nil
		}
		time.Sleep(checkInterval)
	}

	return fmt.Errorf("socket %s did not appear within %v", socketPath, timeout)
}
