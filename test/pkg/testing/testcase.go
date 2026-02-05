package testing

import (
	"fmt"
	"strings"
	"time"

	"github.com/HyphaGroup/oubliette/test/pkg/client"
)

// TestCase represents a single test scenario
type TestCase struct {
	Name        string
	Description string
	Tags        []string
	Covers      []string // Coverage annotations like "manager:create", "cli:oubliette-client"
	Setup       func(*TestContext) error
	Execute     func(*TestContext) error
	Teardown    func(*TestContext) error
	Timeout     time.Duration
}

// TestContext provides state and utilities for test execution
type TestContext struct {
	Client       *client.MCPClient
	Assertions   *Assertions
	ProjectID  string
	SessionID    string
	CreatedProjs []string // Track projects for cleanup
	Logs         []string
	Failed       bool
}

// NewTestContext creates a new test context with the given MCP client
func NewTestContext(mcpClient *client.MCPClient) *TestContext {
	ctx := &TestContext{
		Client:       mcpClient,
		CreatedProjs: []string{},
		Logs:         []string{},
		Failed:       false,
	}
	ctx.Assertions = NewAssertions(ctx)
	return ctx
}

// Log adds a log message to the test context
func (tc *TestContext) Log(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	tc.Logs = append(tc.Logs, msg)
}

// MarkFailed marks the test as failed
func (tc *TestContext) MarkFailed() {
	tc.Failed = true
}

// PreTestCleanup performs cleanup before test starts to ensure clean state
// NOTE: With UUID-based project IDs, pre-test cleanup by name is no longer possible.
// This function now just waits to ensure any previous test's cleanup has completed.
func (tc *TestContext) PreTestCleanup(projectName string) error {
	tc.Log("Pre-test cleanup for: %s (waiting for any previous cleanup to complete)", projectName)

	// Wait for any previous container cleanup to complete
	// Container removal can take a few seconds
	time.Sleep(1000 * time.Millisecond)

	tc.Log("Pre-test cleanup complete")
	return nil
}

// Cleanup performs automatic cleanup of created resources
func (tc *TestContext) Cleanup() error {
	tc.Log("Starting cleanup...")

	// Delete all created projects (CreatedProjs now stores project IDs, not names)
	for _, projID := range tc.CreatedProjs {
		tc.Log("Deleting project: %s", projID)
		params := map[string]interface{}{
			"project_id": projID,
		}

		// Try to stop container first (with retry)
		for i := 0; i < 3; i++ {
			stopResult, _ := tc.Client.InvokeTool("container_stop", params)
			if stopResult != nil && !stopResult.IsError {
				break
			}
			time.Sleep(time.Second)
		}

		// Delete project (with retry)
		for i := 0; i < 3; i++ {
			result, err := tc.Client.InvokeTool("project_delete", params)
			if err != nil {
				if i == 2 { // Last attempt
					tc.Log("Warning: Failed to delete project %s: %v", projID, err)
				}
				time.Sleep(time.Second)
				continue
			}
			if result.IsError {
				if i == 2 {
					tc.Log("Warning: Error deleting project %s: %s", projID, result.GetToolContent())
				}
				time.Sleep(time.Second)
				continue
			}
			break // Success
		}
	}

	tc.Log("Cleanup complete")
	return nil
}

// CreateProject is a helper to create a project and track it for cleanup
// Returns the project ID (UUID) on success
func (tc *TestContext) CreateProject(name, description string) (string, error) {
	tc.Log("Creating project: %s", name)
	params := map[string]interface{}{
		"action":      "create",
		"name":        name,
		"description": description,
	}

	result, err := tc.Client.InvokeTool("project", params)
	if err != nil {
		return "", fmt.Errorf("failed to create project: %w", err)
	}

	if result.IsError {
		return "", fmt.Errorf("create project returned error: %s", result.GetToolContent())
	}

	// Extract project ID from response
	content := result.GetToolContent()
	projectID := ExtractProjectID(content)
	if projectID == "" {
		return "", fmt.Errorf("failed to extract project ID from response: %s", content)
	}

	// Track for cleanup
	tc.CreatedProjs = append(tc.CreatedProjs, projectID)
	tc.ProjectID = projectID

	tc.Log("Project created: %s (ID: %s)", name, projectID)
	return projectID, nil
}

// ExtractProjectID parses project ID from create response text
// Response format: "✅ Project 'name' created successfully!\n\nID: UUID\n..."
func ExtractProjectID(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "ID: ") {
			return strings.TrimPrefix(line, "ID: ")
		}
	}
	return ""
}

// GetDefaultWorkspaceID fetches the default workspace ID for a project
func (tc *TestContext) GetDefaultWorkspaceID(projectID string) (string, error) {
	result, err := tc.Client.InvokeTool("project", map[string]interface{}{
		"action":     "get",
		"project_id": projectID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get project: %w", err)
	}
	if result.IsError {
		return "", fmt.Errorf("project_get returned error: %s", result.GetToolContent())
	}

	// Extract default_workspace_id from response text
	// Response format: "Workspace: /path/to/projects/<id>/workspaces/<workspace_id>"
	content := result.GetToolContent()
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Workspace: ") {
			path := strings.TrimPrefix(line, "Workspace: ")
			// Extract the UUID from the end of the path
			parts := strings.Split(path, "/")
			if len(parts) > 0 {
				return parts[len(parts)-1], nil
			}
		}
	}
	return "", fmt.Errorf("could not find default workspace ID in response")
}

// SpawnContainer is a helper to spawn a container
func (tc *TestContext) SpawnContainer(projectName string) error {
	tc.Log("Starting container for: %s", projectName)
	params := map[string]interface{}{
		"project_id": projectName,
	}

	result, err := tc.Client.InvokeTool("container_start", params)
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	if result.IsError {
		return fmt.Errorf("start container returned error: %s", result.GetToolContent())
	}

	tc.Log("Container started for: %s", projectName)
	return nil
}

// StartTask is a helper to start a session task
func (tc *TestContext) StartTask(projectName, prompt string) (string, error) {
	tc.Log("Starting session task for project: %s", projectName)
	params := map[string]interface{}{
		"project_id": projectName,
		"prompt":       prompt,
	}

	result, err := tc.Client.InvokeTool("session_spawn", params)
	if err != nil {
		return "", fmt.Errorf("failed to spawn session: %w", err)
	}

	if result.IsError {
		return "", fmt.Errorf("spawn session returned error: %s", result.GetToolContent())
	}

	// Extract session ID from content
	content := result.GetToolContent()
	sessionID := ExtractSessionID(content)
	if sessionID == "" {
		return "", fmt.Errorf("failed to extract session ID from response: %s", content)
	}
	
	tc.SessionID = sessionID
	tc.Log("Session spawned: %s", tc.SessionID)

	return tc.SessionID, nil
}

// ExtractSessionID parses session ID from spawn response text
// Response formats:
// - "✅ New session created: SESSION_ID\n\n..."
// - "✅ Session resumed: SESSION_ID\n\n..."
// - Legacy: "✅ Prime gogol spawned: SESSION_ID\n\n..."
func ExtractSessionID(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) > 0 {
		firstLine := lines[0]
		// Look for pattern "created: SESSION_ID" or "resumed: SESSION_ID" or "spawned: SESSION_ID"
		for _, marker := range []string{"created: ", "resumed: ", "spawned: "} {
			if strings.Contains(firstLine, marker) {
				parts := strings.Split(firstLine, marker)
				if len(parts) >= 2 {
					return strings.TrimSpace(parts[1])
				}
			}
		}
	}
	return ""
}

// TestResult represents the outcome of a test execution
type TestResult struct {
	TestName    string
	Passed      bool
	Duration    time.Duration
	Error       error
	Logs        []string
	Assertions  int
	FailedAt    string // Which phase failed: "setup", "execute", "teardown"
}

// Run executes the test case and returns the result
func (t *TestCase) Run(mcpClient *client.MCPClient) *TestResult {
	start := time.Now()
	ctx := NewTestContext(mcpClient)
	result := &TestResult{
		TestName:   t.Name,
		Passed:     true,
		Assertions: 0,
	}

	// Ensure cleanup always runs
	defer func() {
		if err := ctx.Cleanup(); err != nil {
			ctx.Log("Cleanup error: %v", err)
		}
		result.Logs = ctx.Logs
		result.Duration = time.Since(start)
		result.Assertions = ctx.Assertions.Count
	}()

	// Apply timeout if specified
	if t.Timeout > 0 {
		done := make(chan bool, 1)
		go func() {
			// Run test phases
			if err := t.runPhases(ctx, result); err != nil {
				result.Passed = false
				result.Error = err
			}
			done <- true
		}()

		select {
		case <-done:
			// Test completed
		case <-time.After(t.Timeout):
			result.Passed = false
			result.Error = fmt.Errorf("test timeout after %v", t.Timeout)
			result.FailedAt = "timeout"
		}
	} else {
		// Run without timeout
		if err := t.runPhases(ctx, result); err != nil {
			result.Passed = false
			result.Error = err
		}
	}

	return result
}

// runPhases executes setup, execute, and teardown phases
func (t *TestCase) runPhases(ctx *TestContext, result *TestResult) error {
	// Setup phase
	if t.Setup != nil {
		ctx.Log("Running setup...")
		if err := t.Setup(ctx); err != nil {
			result.FailedAt = "setup"
			return fmt.Errorf("setup failed: %w", err)
		}
	}

	// Execute phase
	ctx.Log("Running test...")
	if err := t.Execute(ctx); err != nil {
		result.FailedAt = "execute"
		return fmt.Errorf("test failed: %w", err)
	}

	// Check if any assertions failed
	if ctx.Failed {
		result.FailedAt = "execute"
		return fmt.Errorf("test assertions failed")
	}

	// Teardown phase
	if t.Teardown != nil {
		ctx.Log("Running teardown...")
		if err := t.Teardown(ctx); err != nil {
			result.FailedAt = "teardown"
			return fmt.Errorf("teardown failed: %w", err)
		}
	}

	return nil
}
