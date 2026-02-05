package suites

import (
	"fmt"
	"strings"
	"time"

	testpkg "github.com/HyphaGroup/oubliette/test/pkg/testing"
)

// GetSessionTests returns session management test suite
func GetSessionTests() []*testpkg.TestCase {
	return []*testpkg.TestCase{
		{
			Name:        "test_start_task",
			Description: "Test starting a task (basic session spawn)",
			Tags:        []string{"session", "task"},
			Timeout:     90 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := fmt.Sprintf("test-session-start-%d", time.Now().UnixNano())

				// Pre-cleanup
				ctx.PreTestCleanup(projName)

				// Create project (container auto-starts)
				projID, err := ctx.CreateProject(projName, "Test project for session")
				ctx.Assertions.AssertNoError(err, "Should create project")

				if err != nil {
					return err
				}

				// Start task
				ctx.Log("Starting session task")
				params := map[string]interface{}{
					"project_id":  projID,
					"prompt":      "Say hello",
					"new_session": true, // Force new session
				}

				result, err := ctx.Client.InvokeTool("session", params)
				ctx.Assertions.AssertNoError(err, "Should start task")

				if err == nil && !result.IsError {
					ctx.Log("Task started successfully")
				}

				return nil
			},
		},

		{
			Name:        "test_session_resume",
			Description: "Test that session_spawn resumes existing session by default",
			Tags:        []string{"session", "resume"},
			Timeout:     120 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := fmt.Sprintf("test-session-resume-%d", time.Now().UnixNano())

				// Pre-cleanup
				ctx.PreTestCleanup(projName)

				// Create project
				projID, err := ctx.CreateProject(projName, "Test project for session resume")
				ctx.Assertions.AssertNoError(err, "Should create project")
				if err != nil {
					return err
				}

				// Step 1: Create initial session with new_session=true
				ctx.Log("Step 1: Creating initial session")
				spawnParams := map[string]interface{}{
					"project_id":  projID,
					"prompt":      "Remember the number 42",
					"new_session": true,
				}

				result1, err := ctx.Client.InvokeTool("session", spawnParams)
				ctx.Assertions.AssertNoError(err, "Should create initial session")
				if err != nil {
					return err
				}

				content1 := result1.GetToolContent()
				ctx.Assertions.AssertContains(content1, "New session created", "Should be a new session")

				// Extract session ID from first spawn
				sessionID := testpkg.ExtractSessionID(content1)
				ctx.Log("Initial session: %s", sessionID)

				// Wait for session to process
				time.Sleep(3 * time.Second)

				// End the active session (so it's not running anymore)
				ctx.Log("Ending initial session")
				endParams := map[string]interface{}{
					"action":     "end",
					"session_id": sessionID,
				}
				_, err = ctx.Client.InvokeTool("session", endParams)
				ctx.Assertions.AssertNoError(err, "Should end session")

				// Wait a moment
				time.Sleep(1 * time.Second)

				// Step 2: Spawn again WITHOUT new_session - should resume
				ctx.Log("Step 2: Spawning again (should resume)")
				resumeParams := map[string]interface{}{
					"project_id": projID,
					"prompt":     "What number did I ask you to remember?",
					// new_session not set - should resume
				}

				result2, err := ctx.Client.InvokeTool("session", resumeParams)
				ctx.Assertions.AssertNoError(err, "Should resume session")
				if err != nil {
					return err
				}

				content2 := result2.GetToolContent()
				ctx.Log("Resume result: %s", content2)

				// Verify it resumed (should say "resumed" not "created")
				ctx.Assertions.AssertContains(content2, "resumed", "Should resume existing session")

				// Verify it's the same session
				resumedSessionID := testpkg.ExtractSessionID(content2)
				ctx.Assertions.AssertEqual(sessionID, resumedSessionID, "Should be same session ID")

				// Step 3: Spawn with new_session=true - should create new
				ctx.Log("Step 3: Spawning with new_session=true")
				newParams := map[string]interface{}{
					"project_id":  projID,
					"prompt":      "This is a fresh start",
					"new_session": true,
				}

				result3, err := ctx.Client.InvokeTool("session", newParams)
				ctx.Assertions.AssertNoError(err, "Should create new session")
				if err != nil {
					return err
				}

				content3 := result3.GetToolContent()
				ctx.Log("New session result: %s", content3)

				ctx.Assertions.AssertContains(content3, "New session created", "Should be a new session")

				newSessionID := testpkg.ExtractSessionID(content3)
				if newSessionID == sessionID {
					return fmt.Errorf("new_session=true should create different session ID, got same: %s", newSessionID)
				}
				ctx.Log("New session ID differs: %s vs %s", sessionID, newSessionID)

				ctx.Log("✅ Session resume test passed")
				return nil
			},
		},

		{
			Name:        "test_session_droid_id_captured",
			Description: "Test that droid's session ID is captured and persisted",
			Tags:        []string{"session", "droid"},
			Timeout:     90 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := fmt.Sprintf("test-droid-id-%d", time.Now().UnixNano())

				// Pre-cleanup
				ctx.PreTestCleanup(projName)

				// Create project
				projID, err := ctx.CreateProject(projName, "Test project for droid ID")
				ctx.Assertions.AssertNoError(err, "Should create project")
				if err != nil {
					return err
				}

				// Spawn session
				ctx.Log("Spawning session")
				params := map[string]interface{}{
					"project_id":  projID,
					"prompt":      "Say hello",
					"new_session": true,
				}

				result, err := ctx.Client.InvokeTool("session", params)
				ctx.Assertions.AssertNoError(err, "Should spawn session")
				if err != nil {
					return err
				}

				content := result.GetToolContent()
				ctx.Log("Spawn result: %s", content)

				// Check that Droid Session ID is shown
				ctx.Assertions.AssertContains(content, "Droid Session:", "Should show droid session ID")

				// Extract and verify droid session ID is not empty/placeholder
				lines := strings.Split(content, "\n")
				for _, line := range lines {
					if strings.HasPrefix(line, "Droid Session:") {
						droidID := strings.TrimSpace(strings.TrimPrefix(line, "Droid Session:"))
						if droidID == "" || droidID == "gogol_" {
							return fmt.Errorf("droid session ID should not be empty: %s", droidID)
						}
						ctx.Log("Droid session ID: %s", droidID)
						break
					}
				}

				ctx.Log("✅ Droid session ID captured")
				return nil
			},
		},

		{
			Name:        "test_session_cleanup",
			Description: "Test session_cleanup tool for removing old session metadata",
			Tags:        []string{"session", "cleanup", "management"},
			Timeout:     30 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				// Test cleanup with no old sessions (just verify the tool works)
				// Since we can't easily create old sessions in this test, we just verify
				// the tool runs without error and returns expected output format

				// Test 1: Cleanup all projects (no project_id specified)
				ctx.Log("Testing cleanup all projects")
				cleanupParams := map[string]interface{}{
					"action":        "cleanup",
					"max_age_hours": 24,
				}
				result, err := ctx.Client.InvokeTool("session", cleanupParams)
				ctx.Assertions.AssertNoError(err, "Should run cleanup for all projects")

				if err == nil {
					content := result.GetToolContent()
					// Should either show "No sessions older than" or "Cleaned up X session(s)"
					hasExpectedOutput := strings.Contains(content, "older than") ||
						strings.Contains(content, "session(s)")
					ctx.Assertions.AssertTrue(hasExpectedOutput, "Should return cleanup summary")
					ctx.Log("Cleanup all result: %s", content[:min(200, len(content))])
				}

				// Test 2: Create a project and cleanup specific project
				projName := fmt.Sprintf("test-cleanup-%d", time.Now().UnixNano())
				ctx.PreTestCleanup(projName)

				projID, err := ctx.CreateProject(projName, "Test project for session cleanup")
				ctx.Assertions.AssertNoError(err, "Should create project")
				if err != nil {
					return err
				}

				ctx.Log("Testing cleanup specific project: %s", projID)
				cleanupParams2 := map[string]interface{}{
					"action":        "cleanup",
					"project_id":    projID,
					"max_age_hours": 1,
				}
				result2, err := ctx.Client.InvokeTool("session", cleanupParams2)
				ctx.Assertions.AssertNoError(err, "Should run cleanup for specific project")

				if err == nil {
					content := result2.GetToolContent()
					ctx.Assertions.AssertContains(content, projID, "Should reference the project ID")
					ctx.Log("Cleanup project result: %s", content)
				}

				ctx.Log("✅ Session cleanup tool verified")
				return nil
			},
		},
	}
}

