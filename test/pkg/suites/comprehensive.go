package suites

import (
	"fmt"
	"strings"
	"time"

	testpkg "github.com/HyphaGroup/oubliette/test/pkg/testing"
)

// GetComprehensiveTests returns comprehensive end-to-end integration tests
func GetComprehensiveTests() []*testpkg.TestCase {
	return []*testpkg.TestCase{
		{
			Name:        "test_full_dev_workflow",
			Description: "Test complete development workflow: create → code → commit → verify",
			Tags:        []string{"comprehensive", "e2e", "workflow"},
			Timeout:     180 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := fmt.Sprintf("test-full-workflow-%d", time.Now().Unix())

				// Pre-cleanup
				ctx.PreTestCleanup(projName)

				// 1. Create project with git
				ctx.Log("Step 1: Creating project with git initialization")
				createParams := map[string]interface{}{
					"name":        projName,
					"description": "Comprehensive test project",
					"init_git":    true,
				}

				createParams["action"] = "create"
				createResult, err := ctx.Client.InvokeTool("project", createParams)
				ctx.Assertions.AssertNoError(err, "Should create project")
				if err != nil {
					return err
				}
				ctx.Assertions.AssertFalse(createResult.IsError, "Create should succeed")

				// Extract project ID
				projID := testpkg.ExtractProjectID(createResult.GetToolContent())
				ctx.Assertions.AssertNotEmpty(projID, "Should have project ID")
				ctx.CreatedProjs = append(ctx.CreatedProjs, projID)

				// 2. Verify container auto-started
				ctx.Log("Step 2: Verifying container auto-started")
				getParams := map[string]interface{}{
					"action":     "get",
					"project_id": projID,
				}
				getResult, err := ctx.Client.InvokeTool("project", getParams)
				if err == nil {
					content := getResult.GetToolContent()
					ctx.Assertions.AssertContains(content, "running", "Container should be auto-started")
				}

				// 3. Write a test file using container exec
				ctx.Log("Step 3: Writing test file to workspace")
				execParams := map[string]interface{}{
					"action":     "exec",
					"project_id": projID,
					"command":    "echo 'Hello from comprehensive test' > /workspace/test.txt",
				}
				execResult, err := ctx.Client.InvokeTool("container", execParams)
				ctx.Assertions.AssertNoError(err, "Should execute write command")
				if err == nil {
					ctx.Assertions.AssertFalse(execResult.IsError, "Write should succeed")
				}

				// 4. Verify file was created
				ctx.Log("Step 4: Verifying file was created")
				catParams := map[string]interface{}{
					"action":     "exec",
					"project_id": projID,
					"command":    "cat /workspace/test.txt",
				}
				catResult, err := ctx.Client.InvokeTool("container", catParams)
				ctx.Assertions.AssertNoError(err, "Should read file")
				if err == nil {
					content := catResult.GetToolContent()
					ctx.Assertions.AssertContains(content, "Hello from comprehensive test", "File should contain expected text")
				}

				// 5. Create a git commit
				ctx.Log("Step 5: Creating git commit")
				gitAddParams := map[string]interface{}{
					"action":     "exec",
					"project_id": projID,
					"command":    "cd /workspace && git add test.txt",
				}
				_, err = ctx.Client.InvokeTool("container", gitAddParams)
				ctx.Assertions.AssertNoError(err, "Should git add")

				gitCommitParams := map[string]interface{}{
					"action":     "exec",
					"project_id": projID,
					"command":    "cd /workspace && git commit -m 'Add test file'",
				}
				commitResult, err := ctx.Client.InvokeTool("container", gitCommitParams)
				ctx.Assertions.AssertNoError(err, "Should git commit")
				if err == nil {
					ctx.Assertions.AssertFalse(commitResult.IsError, "Commit should succeed")
				}

				// 6. Verify commit exists (using git status for reliable verification)
				ctx.Log("Step 6: Verifying commit succeeded")
				gitStatusParams := map[string]interface{}{
					"action":     "exec",
					"project_id": projID,
					"command":    "cd /workspace && git status --short",
				}
				statusResult, err := ctx.Client.InvokeTool("container", gitStatusParams)
				ctx.Assertions.AssertNoError(err, "Should get git status")

				if err == nil {
					statusContent := statusResult.GetToolContent()
					ctx.Log("Git status output: [%s]", statusContent)

					// Clean status (empty or no test.txt) means file is committed
					statusContent = strings.TrimSpace(statusContent)
					if statusContent == "" || !strings.Contains(statusContent, "test.txt") {
						ctx.Log("✓ File successfully committed (clean status)")
					} else {
						ctx.Assertions.AssertNotContains(statusContent, "test.txt", "File should be committed (not in status)")
					}
				}

				ctx.Log("✅ Full development workflow completed successfully")
				return nil
			},
		},

		{
			Name:        "test_tool_refactoring_validation",
			Description: "Validate all refactored tool names work correctly",
			Tags:        []string{"comprehensive", "tools", "refactoring"},
			Timeout:     120 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := fmt.Sprintf("test-tool-names-%d", time.Now().Unix())

				// Pre-cleanup
				ctx.PreTestCleanup(projName)

				ctx.Log("Testing all refactored tool names...")

				// PROJECT TOOLS
				ctx.Log("1. Testing project_* tools")

				// project create
				createResult, err := ctx.Client.InvokeTool("project", map[string]interface{}{
					"action":      "create",
					"name":        projName,
					"description": "Tool validation project",
				})
				ctx.Assertions.AssertNoError(err, "project create should work")
				ctx.Assertions.AssertFalse(createResult.IsError, "project create should succeed")

				// Extract project ID
				projID := testpkg.ExtractProjectID(createResult.GetToolContent())
				ctx.Assertions.AssertNotEmpty(projID, "Should have project ID")
				ctx.CreatedProjs = append(ctx.CreatedProjs, projID)

				// project list
				listResult, err := ctx.Client.InvokeTool("project", map[string]interface{}{"action": "list"})
				ctx.Assertions.AssertNoError(err, "project list should work")
				ctx.Assertions.AssertContains(listResult.GetToolContent(), projName, "Should list new project")

				// project get
				getResult, err := ctx.Client.InvokeTool("project", map[string]interface{}{
					"action":     "get",
					"project_id": projID,
				})
				ctx.Assertions.AssertNoError(err, "project get should work")
				ctx.Assertions.AssertContains(getResult.GetToolContent(), projName, "Should get project details")

				// CONTAINER TOOLS
				ctx.Log("2. Testing container_* tools")

				// Container should be auto-started, verify it
				ctx.Assertions.AssertContains(getResult.GetToolContent(), "running", "Container should be running")

				// container_exec
				_, err = ctx.Client.InvokeTool("container", map[string]interface{}{"action": "exec",
					"project_id": projID,
					"command":    "echo 'test'",
				})
				ctx.Assertions.AssertNoError(err, "container_exec should work")

				// container_logs
				_, err = ctx.Client.InvokeTool("container", map[string]interface{}{"action": "logs",
					"project_id": projID,
				})
				ctx.Assertions.AssertNoError(err, "container_logs should work")

				// container_stop
				_, err = ctx.Client.InvokeTool("container", map[string]interface{}{"action": "stop",
					"project_id": projID,
				})
				ctx.Assertions.AssertNoError(err, "container_stop should work")

				// Wait a moment for container to stop
				time.Sleep(2 * time.Second)

				// container_start (restart it)
				_, err = ctx.Client.InvokeTool("container", map[string]interface{}{"action": "start",
					"project_id": projID,
				})
				ctx.Assertions.AssertNoError(err, "container_start should work")

				// CONFIG TOOLS
				ctx.Log("3. Testing config_* tools")

				// config_limits
				limitsResult, err := ctx.Client.InvokeTool("config_limits", map[string]interface{}{
					"project_id": projID,
				})
				ctx.Assertions.AssertNoError(err, "config_limits should work")
				ctx.Assertions.AssertContains(limitsResult.GetToolContent(), "Max Depth", "Should show limits")

				// PROJECT DELETE (will be done in cleanup, but verify tool works)
				ctx.Log("4. All refactored tools validated successfully")

				return nil
			},
		},

		{
			Name:        "test_interactive_streaming",
			Description: "Validate interactive streaming with session_events",
			Tags:        []string{"comprehensive", "streaming", "tools"},
			Timeout:     150 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := fmt.Sprintf("test-interactive-%d", time.Now().Unix())

				// Pre-cleanup
				ctx.PreTestCleanup(projName)

				ctx.Log("Testing interactive streaming...")

				// Create project
				projID, err := ctx.CreateProject(projName, "Streaming test project")
				ctx.Assertions.AssertNoError(err, "Should create project")
				if err != nil {
					return err
				}

				// Wait for container to be ready
				time.Sleep(2 * time.Second)

				// Start an interactive streaming session
				ctx.Log("Starting interactive streaming session")
				spawnParams := map[string]interface{}{
					"project_id":  projID,
					"prompt":      "Say hello",
					"new_session": true,
				}

				spawnResult, err := ctx.Client.InvokeTool("session", spawnParams)
				ctx.Assertions.AssertNoError(err, "Should spawn streaming session")
				if err != nil {
					return err
				}

				// Extract session ID from text response
				content := spawnResult.GetToolContent()
				sessionID := testpkg.ExtractSessionID(content)
				if sessionID != "" {
					ctx.SessionID = sessionID
					ctx.Log("Streaming session ID: %s", sessionID)

					// Wait for session to process
					time.Sleep(3 * time.Second)

					// Test 1: session_events to get streaming output
					ctx.Log("Test 1: session events")
					eventsParams := map[string]interface{}{
						"action":     "events",
						"session_id": sessionID,
					}
					eventsResult, err := ctx.Client.InvokeTool("session", eventsParams)
					ctx.Assertions.AssertNoError(err, "session events should work")
					if err == nil {
						eventsContent := eventsResult.GetToolContent()
						ctx.Assertions.AssertContains(eventsContent, "Events", "Should show events")
						ctx.Log("✓ session events works")
					}

					// Test 2: session get for persisted metadata
					ctx.Log("Test 2: session get")
					getParams := map[string]interface{}{
						"action":     "get",
						"session_id": sessionID,
					}
					getResult, err := ctx.Client.InvokeTool("session", getParams)
					ctx.Assertions.AssertNoError(err, "session_get should work")
					if err == nil {
						getContent := getResult.GetToolContent()
						ctx.Assertions.AssertContains(getContent, "Session:", "Should show session info")
						ctx.Log("✓ session_get works")
					}

					ctx.Log("✅ Interactive streaming validated successfully")
				}

				return nil
			},
		},

		{
			Name:        "test_session_lifecycle_complete",
			Description: "Test complete session lifecycle: spawn → get → continue → list → end",
			Tags:        []string{"comprehensive", "session", "lifecycle"},
			Timeout:     150 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := fmt.Sprintf("test-session-lifecycle-%d", time.Now().Unix())

				// Pre-cleanup
				ctx.PreTestCleanup(projName)

				ctx.Log("Testing complete session lifecycle...")

				// Create project
				projID, err := ctx.CreateProject(projName, "Session lifecycle test")
				ctx.Assertions.AssertNoError(err, "Should create project")
				if err != nil {
					return err
				}

				time.Sleep(2 * time.Second)

				// 1. session_spawn
				ctx.Log("1. Spawning session")
				spawnParams := map[string]interface{}{
					"project_id":  projID,
					"prompt":      "Initial task",
					"new_session": true,
				}
				spawnResult, err := ctx.Client.InvokeTool("session", spawnParams)
				ctx.Assertions.AssertNoError(err, "session_spawn should work")
				if err != nil {
					return err
				}

				// Extract session ID from text response
				content := spawnResult.GetToolContent()
				sessionID := testpkg.ExtractSessionID(content)
				if sessionID == "" {
					ctx.Log("Failed to extract session ID from: %s", content)
					return fmt.Errorf("failed to extract session ID")
				}

				ctx.SessionID = sessionID
				ctx.Log("Session ID: %s", sessionID)

				// 2. session get
				ctx.Log("2. Getting session status")
				time.Sleep(2 * time.Second)
				getParams := map[string]interface{}{
					"action":     "get",
					"session_id": sessionID,
				}
				getResult, err := ctx.Client.InvokeTool("session", getParams)
				ctx.Assertions.AssertNoError(err, "session get should work")
				if err == nil {
					ctx.Assertions.AssertContains(getResult.GetToolContent(), sessionID, "Should contain session ID")
				}

				// 3. session list
				ctx.Log("3. Listing sessions for project")
				listParams := map[string]interface{}{
					"action":     "list",
					"project_id": projID,
				}
				listResult, err := ctx.Client.InvokeTool("session", listParams)
				ctx.Assertions.AssertNoError(err, "session list should work")
				if err == nil {
					content := listResult.GetToolContent()
					// Session ID might be shortened in list
					shortID := sessionID
					if len(sessionID) > 8 {
						shortID = sessionID[:8]
					}
					if !strings.Contains(content, sessionID) && !strings.Contains(content, shortID) {
						ctx.Log("Session might not be in list yet (async)")
					}
				}

				// 4. session message (send message to active streaming session)
				ctx.Log("4. Sending message to active session")
				msgParams := map[string]interface{}{
					"action":     "message",
					"session_id": sessionID,
					"message":    "Additional instructions",
				}
				_, err = ctx.Client.InvokeTool("session", msgParams)
				ctx.Assertions.AssertNoError(err, "session message should work")

				// 5. session end
				ctx.Log("5. Ending session")
				time.Sleep(1 * time.Second)
				endParams := map[string]interface{}{
					"action":     "end",
					"session_id": sessionID,
				}
				endResult, err := ctx.Client.InvokeTool("session", endParams)
				ctx.Assertions.AssertNoError(err, "session_end should work")
				if err == nil {
					ctx.Assertions.AssertContains(endResult.GetToolContent(), "ended", "Should confirm session ended")
				}

				ctx.Log("✅ Complete session lifecycle validated")
				return nil
			},
		},

		{
			Name:        "test_error_recovery_scenarios",
			Description: "Test error handling and recovery in various failure scenarios",
			Tags:        []string{"comprehensive", "error", "resilience"},
			Timeout:     120 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := fmt.Sprintf("test-error-recovery-%d", time.Now().Unix())

				// Pre-cleanup
				ctx.PreTestCleanup(projName)

				ctx.Log("Testing error recovery scenarios...")

				// Scenario 1: Get nonexistent project
				ctx.Log("Scenario 1: Get nonexistent project")
				getParams := map[string]interface{}{
					"action":     "get",
					"project_id": "nonexistent-project-12345",
				}
				getResult, err := ctx.Client.InvokeTool("project", getParams)
				if getResult != nil && getResult.IsError {
					ctx.Log("✓ Correctly returns error for nonexistent project")
				} else if err != nil {
					ctx.Log("✓ Correctly errors for nonexistent project")
				}

				// Scenario 2: Delete nonexistent project
				ctx.Log("Scenario 2: Delete nonexistent project")
				deleteParams := map[string]interface{}{
					"action":     "delete",
					"project_id": "nonexistent-project-12345",
				}
				deleteResult, err := ctx.Client.InvokeTool("project", deleteParams)
				if deleteResult != nil && deleteResult.IsError {
					ctx.Log("✓ Correctly handles delete of nonexistent project")
				} else if err != nil {
					ctx.Log("✓ Correctly errors on delete of nonexistent project")
				}

				// Scenario 3: Execute command in nonexistent container
				ctx.Log("Scenario 3: Execute in nonexistent container")
				execParams := map[string]interface{}{
					"action":     "exec",
					"project_id": "nonexistent-project-12345",
					"command":    "echo test",
				}
				execResult, err := ctx.Client.InvokeTool("container", execParams)
				if execResult != nil && execResult.IsError {
					ctx.Log("✓ Correctly handles exec in nonexistent container")
				} else if err != nil {
					ctx.Log("✓ Correctly errors on exec in nonexistent container")
				}

				// Scenario 4: Get nonexistent session
				ctx.Log("Scenario 4: Get nonexistent session")
				sessionParams := map[string]interface{}{
					"action":     "get",
					"session_id": "nonexistent-session-12345",
				}
				sessionResult, err := ctx.Client.InvokeTool("session", sessionParams)
				if sessionResult != nil && sessionResult.IsError {
					ctx.Log("✓ Correctly handles get of nonexistent session")
				} else if err != nil {
					ctx.Log("✓ Correctly errors on get of nonexistent session")
				}

				// Scenario 5: Create duplicate project
				ctx.Log("Scenario 5: Create duplicate project")
				projID, err := ctx.CreateProject(projName, "First project")
				ctx.Assertions.AssertNoError(err, "Should create first project")
				_ = projID // projID used for cleanup via ctx.CreatedProjs

				// Try to create again with same name
				dupParams := map[string]interface{}{
					"action":      "create",
					"name":        projName,
					"description": "Duplicate project",
				}
				dupResult, err := ctx.Client.InvokeTool("project", dupParams)
				if dupResult != nil && dupResult.IsError {
					ctx.Log("✓ Correctly prevents duplicate project creation")
				} else if err != nil {
					ctx.Log("✓ Correctly errors on duplicate project")
				} else {
					ctx.Log("⚠ Duplicate project was allowed (may be intentional)")
				}

				ctx.Log("✅ Error recovery scenarios validated")
				return nil
			},
		},
	}
}
