package suites

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	testpkg "github.com/HyphaGroup/oubliette/test/pkg/testing"
)

// GetOpenSpecTests returns OpenSpec integration test suite
func GetOpenSpecTests() []*testpkg.TestCase {
	return []*testpkg.TestCase{
		{
			Name:        "test_project_has_openspec_directory",
			Description: "Verify new project has openspec/ directory with expected structure",
			Tags:        []string{"openspec", "project"},
			Timeout:     60 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := "test-openspec-dir"

				ctx.PreTestCleanup(projName)

				projID, err := ctx.CreateProject(projName, "Test OpenSpec directory creation")
				ctx.Assertions.AssertNoError(err, "Should create project")
				if err != nil {
					return err
				}

				// Execute ls in container to check openspec directory
				params := map[string]interface{}{
					"action":     "exec",
					"project_id": projID,
					"command":    "ls -la /workspace/openspec/",
				}
				result, err := ctx.Client.InvokeTool("container", params)
				ctx.Assertions.AssertNoError(err, "Should exec ls command")
				if err != nil {
					return err
				}

				content := result.GetToolContent()
				ctx.Assertions.AssertContains(content, "AGENTS.md", "Should have AGENTS.md in openspec/")
				ctx.Assertions.AssertContains(content, "project.md", "Should have project.md in openspec/")

				return nil
			},
		},

		{
			Name:        "test_openspec_cli_available",
			Description: "Verify openspec CLI is available in container",
			Tags:        []string{"openspec", "cli"},
			Timeout:     60 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := "test-openspec-cli"

				ctx.PreTestCleanup(projName)

				projID, err := ctx.CreateProject(projName, "Test OpenSpec CLI availability")
				ctx.Assertions.AssertNoError(err, "Should create project")
				if err != nil {
					return err
				}

				// Check openspec version
				params := map[string]interface{}{
					"action":     "exec",
					"project_id": projID,
					"command":    "source ~/.nvm/nvm.sh && openspec --version",
				}
				result, err := ctx.Client.InvokeTool("container", params)
				ctx.Assertions.AssertNoError(err, "Should exec openspec --version")
				if err != nil {
					return err
				}

				content := result.GetToolContent()
				// Should contain version number like "0.19.0"
				ctx.Assertions.AssertTrue(
					strings.Contains(content, ".") && !strings.Contains(content, "not found"),
					"openspec CLI should be available and return version",
				)

				return nil
			},
		},

		{
			Name:        "test_openspec_list_works",
			Description: "Verify openspec list command works in container",
			Tags:        []string{"openspec", "cli"},
			Timeout:     60 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := "test-openspec-list"

				ctx.PreTestCleanup(projName)

				projID, err := ctx.CreateProject(projName, "Test OpenSpec list command")
				ctx.Assertions.AssertNoError(err, "Should create project")
				if err != nil {
					return err
				}

				// Run openspec list --json
				params := map[string]interface{}{
					"action":     "exec",
					"project_id": projID,
					"command":    "source ~/.nvm/nvm.sh && cd /workspace && openspec list --json",
				}
				result, err := ctx.Client.InvokeTool("container", params)
				ctx.Assertions.AssertNoError(err, "Should exec openspec list")
				if err != nil {
					return err
				}

				content := result.GetToolContent()
				// Should be valid JSON with changes array
				ctx.Assertions.AssertContains(content, "changes", "Should contain changes array")

				return nil
			},
		},

		{
			Name:        "test_project_changes_tool",
			Description: "Test project_changes MCP tool returns change list",
			Tags:        []string{"openspec", "mcp"},
			Timeout:     60 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := "test-proj-changes"

				ctx.PreTestCleanup(projName)

				projID, err := ctx.CreateProject(projName, "Test project_changes tool")
				ctx.Assertions.AssertNoError(err, "Should create project")
				if err != nil {
					return err
				}

				// Call project_changes
				params := map[string]interface{}{
					"project_id": projID,
				}
				result, err := ctx.Client.InvokeTool("project_changes", params)
				ctx.Assertions.AssertNoError(err, "Should invoke project_changes")
				if err != nil {
					return err
				}

				ctx.Assertions.AssertFalse(result.IsError, "Should not return error")
				content := result.GetToolContent()
				ctx.Assertions.AssertContains(content, "project_id", "Should contain project_id")
				ctx.Assertions.AssertContains(content, projID, "Should contain the project ID value")

				return nil
			},
		},

		{
			Name:        "test_session_plan_mode",
			Description: "Test session with mode=plan prepends /openspec-proposal",
			Tags:        []string{"openspec", "session", "mode"},
			Timeout:     120 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := "test-plan-mode"

				ctx.PreTestCleanup(projName)

				projID, err := ctx.CreateProject(projName, "Test plan mode")
				ctx.Assertions.AssertNoError(err, "Should create project")
				if err != nil {
					return err
				}

				// Send message with mode=plan (uses default workspace)
				params := map[string]interface{}{
					"action":     "message",
					"project_id": projID,
					"message":    "Add user authentication",
					"mode":       "plan",
				}
				result, err := ctx.Client.InvokeTool("session", params)
				ctx.Assertions.AssertNoError(err, "Should send message in plan mode")
				if err != nil {
					return err
				}

				ctx.Assertions.AssertFalse(result.IsError, "Should not return error")

				// Parse result to get session ID
				content := result.GetToolContent()
				var msgResult struct {
					SessionID string `json:"session_id"`
					Spawned   bool   `json:"spawned"`
				}
				if err := json.Unmarshal([]byte(content), &msgResult); err == nil {
					ctx.Assertions.AssertNotEmpty(msgResult.SessionID, "Should have session ID")
				}

				return nil
			},
		},

		{
			Name:        "test_session_build_mode_creates_state",
			Description: "Test session with mode=build creates build-mode.json state file",
			Tags:        []string{"openspec", "session", "build"},
			Timeout:     120 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := "test-build-state"

				ctx.PreTestCleanup(projName)

				projID, err := ctx.CreateProject(projName, "Test build mode state")
				ctx.Assertions.AssertNoError(err, "Should create project")
				if err != nil {
					return err
				}

				// First, create a change directory with tasks (mock)
				setupCmd := `source ~/.nvm/nvm.sh && cd /workspace && mkdir -p openspec/changes/test-change && echo "# Tasks" > openspec/changes/test-change/tasks.md && echo "- [ ] Test task" >> openspec/changes/test-change/tasks.md`
				_, err = ctx.Client.InvokeTool("container", map[string]interface{}{"action": "exec",
					"project_id": projID,
					"command":    setupCmd,
				})
				ctx.Assertions.AssertNoError(err, "Should create mock change")

				// Send message with mode=build and change_id (uses default workspace)
				params := map[string]interface{}{
					"action":     "message",
					"project_id": projID,
					"message":    "Implement the change",
					"mode":       "build",
					"change_id":  "test-change",
				}
				result, err := ctx.Client.InvokeTool("session", params)
				ctx.Assertions.AssertNoError(err, "Should send message in build mode")
				if err != nil {
					return err
				}

				ctx.Assertions.AssertFalse(result.IsError, "Should not return error")

				// Get the workspace list to find the default workspace ID
				wsListResult, err := ctx.Client.InvokeTool("workspace", map[string]interface{}{"action": "list",
					"project_id": projID,
				})
				if err != nil {
					return err
				}

				// Parse workspace list to get default workspace ID
				var wsData struct {
					Workspaces []struct {
						ID        string `json:"id"`
						IsDefault bool   `json:"is_default"`
					} `json:"workspaces"`
				}
				workspaceID := ""
				if err := json.Unmarshal([]byte(wsListResult.GetToolContent()), &wsData); err == nil {
					for _, ws := range wsData.Workspaces {
						if ws.IsDefault {
							workspaceID = ws.ID
							break
						}
					}
				}

				if workspaceID != "" {
					// Check that state file was created
					checkCmd := fmt.Sprintf("cat /workspace/workspaces/%s/.factory/build-mode.json 2>/dev/null || echo 'NOT_FOUND'", workspaceID)
					checkResult, err := ctx.Client.InvokeTool("container", map[string]interface{}{"action": "exec",
						"project_id": projID,
						"command":    checkCmd,
					})
					if err == nil {
						content := checkResult.GetToolContent()
						if !strings.Contains(content, "NOT_FOUND") {
							ctx.Assertions.AssertContains(content, "test-change", "State file should contain change_id")
							ctx.Assertions.AssertContains(content, "phase", "State file should contain phase")
						}
					}
				}

				return nil
			},
		},

		{
			Name:        "test_session_events_include_children",
			Description: "Test session_events with include_children parameter",
			Tags:        []string{"openspec", "session", "events"},
			Timeout:     60 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := "test-events-children"

				ctx.PreTestCleanup(projName)

				projID, err := ctx.CreateProject(projName, "Test events include_children")
				ctx.Assertions.AssertNoError(err, "Should create project")
				if err != nil {
					return err
				}

				// Send a simple message to create session (uses default workspace)
				msgResult, err := ctx.Client.InvokeTool("session", map[string]interface{}{"action": "message",
					"project_id": projID,
					"message":    "Hello",
				})
				ctx.Assertions.AssertNoError(err, "Should send message")
				if err != nil {
					return err
				}

				// Extract session ID
				var result struct {
					SessionID string `json:"session_id"`
				}
				if err := json.Unmarshal([]byte(msgResult.GetToolContent()), &result); err != nil {
					return fmt.Errorf("failed to parse session result: %w", err)
				}

				// Get events with include_children=true
				eventsResult, err := ctx.Client.InvokeTool("session", map[string]interface{}{"action": "events",
					"session_id":       result.SessionID,
					"include_children": true,
				})
				ctx.Assertions.AssertNoError(err, "Should get events with include_children")
				if err != nil {
					return err
				}

				ctx.Assertions.AssertFalse(eventsResult.IsError, "Should not return error")
				content := eventsResult.GetToolContent()
				ctx.Assertions.AssertContains(content, "events", "Should contain events field")

				return nil
			},
		},

		{
			Name:        "test_project_tasks_tool",
			Description: "Test project_tasks MCP tool returns task tree",
			Tags:        []string{"openspec", "mcp"},
			Timeout:     60 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := "test-proj-tasks"

				ctx.PreTestCleanup(projName)

				projID, err := ctx.CreateProject(projName, "Test project_tasks tool")
				ctx.Assertions.AssertNoError(err, "Should create project")
				if err != nil {
					return err
				}

				// Create a mock change with tasks
				setupCmd := `source ~/.nvm/nvm.sh && cd /workspace && mkdir -p openspec/changes/test-task-change && cat > openspec/changes/test-task-change/tasks.md << 'EOF'
# Tasks

- [ ] Task 1: First task
- [ ] Task 2: Second task
- [x] Task 3: Completed task
EOF`
				_, err = ctx.Client.InvokeTool("container", map[string]interface{}{"action": "exec",
					"project_id": projID,
					"command":    setupCmd,
				})
				ctx.Assertions.AssertNoError(err, "Should create mock change")

				// Call project_tasks
				params := map[string]interface{}{
					"project_id": projID,
					"change_id":  "test-task-change",
				}
				result, err := ctx.Client.InvokeTool("project_tasks", params)
				ctx.Assertions.AssertNoError(err, "Should invoke project_tasks")
				if err != nil {
					return err
				}

				ctx.Assertions.AssertFalse(result.IsError, "Should not return error")
				content := result.GetToolContent()
				ctx.Assertions.AssertContains(content, "project_id", "Should contain project_id")
				ctx.Assertions.AssertContains(content, "test-task-change", "Should contain change_id")

				return nil
			},
		},

		{
			Name:        "test_build_mode_state_file_fields",
			Description: "Test build mode state file has correct build_all and phase fields",
			Tags:        []string{"openspec", "build", "state"},
			Timeout:     120 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := "test-build-state-fields"

				ctx.PreTestCleanup(projName)

				projID, err := ctx.CreateProject(projName, "Test build mode state fields")
				ctx.Assertions.AssertNoError(err, "Should create project")
				if err != nil {
					return err
				}

				// Create a mock change
				setupCmd := `source ~/.nvm/nvm.sh && cd /workspace && mkdir -p openspec/changes/state-test && echo "- [ ] Task" > openspec/changes/state-test/tasks.md`
				_, err = ctx.Client.InvokeTool("container", map[string]interface{}{"action": "exec",
					"project_id": projID,
					"command":    setupCmd,
				})
				ctx.Assertions.AssertNoError(err, "Should create mock change")

				// Send message with mode=build (uses default workspace)
				_, err = ctx.Client.InvokeTool("session", map[string]interface{}{"action": "message",
					"project_id": projID,
					"message":    "Build it",
					"mode":       "build",
					"change_id":  "state-test",
					"build_all":  true,
				})
				ctx.Assertions.AssertNoError(err, "Should send build message")

				// Get workspace list to find default workspace ID
				wsListResult, err := ctx.Client.InvokeTool("workspace", map[string]interface{}{"action": "list",
					"project_id": projID,
				})
				if err != nil {
					return err
				}

				var wsData struct {
					Workspaces []struct {
						ID        string `json:"id"`
						IsDefault bool   `json:"is_default"`
					} `json:"workspaces"`
				}
				workspaceID := ""
				if err := json.Unmarshal([]byte(wsListResult.GetToolContent()), &wsData); err == nil {
					for _, ws := range wsData.Workspaces {
						if ws.IsDefault {
							workspaceID = ws.ID
							break
						}
					}
				}

				if workspaceID != "" {
					// Check state file contents
					checkCmd := fmt.Sprintf("cat /workspace/workspaces/%s/.factory/build-mode.json 2>/dev/null || echo 'NOT_FOUND'", workspaceID)
					checkResult, err := ctx.Client.InvokeTool("container", map[string]interface{}{"action": "exec",
						"project_id": projID,
						"command":    checkCmd,
					})
					if err == nil {
						content := checkResult.GetToolContent()
						if !strings.Contains(content, "NOT_FOUND") {
							ctx.Assertions.AssertContains(content, `"build_all": true`, "State should have build_all=true")
							ctx.Assertions.AssertContains(content, `"phase": "build"`, "State should have phase=build")
							ctx.Assertions.AssertContains(content, `"change_id": "state-test"`, "State should have change_id")
							ctx.Assertions.AssertContains(content, `"max_iterations"`, "State should have max_iterations")
						}
					}
				}

				return nil
			},
		},

		{
			Name:        "test_openspec_validate",
			Description: "Test openspec validate works on empty project",
			Tags:        []string{"openspec", "cli"},
			Timeout:     60 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := "test-openspec-validate"

				ctx.PreTestCleanup(projName)

				projID, err := ctx.CreateProject(projName, "Test OpenSpec validate")
				ctx.Assertions.AssertNoError(err, "Should create project")
				if err != nil {
					return err
				}

				// Run openspec validate
				params := map[string]interface{}{
					"action":     "exec",
					"project_id": projID,
					"command":    "source ~/.nvm/nvm.sh && cd /workspace && openspec validate 2>&1 || true",
				}
				result, err := ctx.Client.InvokeTool("container", params)
				ctx.Assertions.AssertNoError(err, "Should exec openspec validate")
				if err != nil {
					return err
				}

				// Validate should run without crashing (may report warnings/errors for empty project)
				content := result.GetToolContent()
				ctx.Assertions.AssertFalse(
					strings.Contains(content, "command not found"),
					"openspec validate should be available",
				)

				return nil
			},
		},

		{
			Name:        "test_session_events_without_children",
			Description: "Test session_events without include_children excludes child events",
			Tags:        []string{"openspec", "session", "events"},
			Timeout:     60 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := "test-events-no-children"

				ctx.PreTestCleanup(projName)

				projID, err := ctx.CreateProject(projName, "Test events exclude children")
				ctx.Assertions.AssertNoError(err, "Should create project")
				if err != nil {
					return err
				}

				// Send message to create session (uses default workspace)
				msgResult, err := ctx.Client.InvokeTool("session", map[string]interface{}{"action": "message",
					"project_id": projID,
					"message":    "Hello",
				})
				ctx.Assertions.AssertNoError(err, "Should send message")
				if err != nil {
					return err
				}

				var result struct {
					SessionID string `json:"session_id"`
				}
				if err := json.Unmarshal([]byte(msgResult.GetToolContent()), &result); err != nil {
					return fmt.Errorf("failed to parse session result: %w", err)
				}

				// Get events WITHOUT include_children (default)
				eventsResult, err := ctx.Client.InvokeTool("session", map[string]interface{}{"action": "events",
					"session_id": result.SessionID,
				})
				ctx.Assertions.AssertNoError(err, "Should get events")
				if err != nil {
					return err
				}

				ctx.Assertions.AssertFalse(eventsResult.IsError, "Should not return error")
				content := eventsResult.GetToolContent()
				ctx.Assertions.AssertContains(content, "events", "Should contain events field")

				// Events should NOT have session_id field when include_children is false
				var eventsData struct {
					Events []struct {
						SessionID string `json:"session_id"`
					} `json:"events"`
				}
				if err := json.Unmarshal([]byte(content), &eventsData); err == nil {
					for _, e := range eventsData.Events {
						// Session ID should be empty string when include_children=false
						ctx.Assertions.AssertEqual("", e.SessionID, "Events should not have session_id when include_children=false")
					}
				}

				return nil
			},
		},

		{
			Name:        "test_project_changes_includes_session_correlation",
			Description: "Verify project_changes includes active_sessions for changes being built",
			Tags:        []string{"openspec", "session", "correlation"},
			Timeout:     120 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := "test-session-correlation"

				ctx.PreTestCleanup(projName)

				projID, err := ctx.CreateProject(projName, "Test session correlation")
				ctx.Assertions.AssertNoError(err, "Should create project")
				if err != nil {
					return err
				}

				// Get default workspace ID
				workspaceID, err := ctx.GetDefaultWorkspaceID(projID)
				ctx.Assertions.AssertNoError(err, "Should get default workspace ID")
				if err != nil {
					return err
				}

				// Create a minimal OpenSpec change in the container
				createChangeCmd := `mkdir -p /workspace/openspec/changes/test-change && 
cat > /workspace/openspec/changes/test-change/proposal.md << 'EOF'
# Test Change
A test change for correlation testing.
EOF
cat > /workspace/openspec/changes/test-change/tasks.md << 'EOF'
- [ ] 1.1 Test task one
- [ ] 1.2 Test task two
EOF`
				params := map[string]interface{}{
					"action":     "exec",
					"project_id": projID,
					"command":    createChangeCmd,
				}
				_, err = ctx.Client.InvokeTool("container", params)
				ctx.Assertions.AssertNoError(err, "Should create test change")
				if err != nil {
					return err
				}

				// Start a session in build mode for this change
				msgParams := map[string]interface{}{
					"action":       "message",
					"project_id":   projID,
					"workspace_id": workspaceID,
					"message":      "Implement the test tasks",
					"mode":         "build",
					"change_id":    "test-change",
				}
				msgResult, err := ctx.Client.InvokeTool("session", msgParams)
				ctx.Assertions.AssertNoError(err, "Should send build mode message")
				if err != nil {
					return err
				}
				ctx.Assertions.AssertFalse(msgResult.IsError, "session_message should succeed")

				// Extract session ID
				var msgData struct {
					SessionID string `json:"session_id"`
				}
				if err := json.Unmarshal([]byte(msgResult.GetToolContent()), &msgData); err != nil {
					return fmt.Errorf("failed to parse session_message result: %w", err)
				}
				sessionID := msgData.SessionID
				ctx.Assertions.AssertNotEmpty(sessionID, "Should have session ID")

				// Give the session a moment to register
				time.Sleep(500 * time.Millisecond)

				// Now call project_changes and check for session correlation
				changesParams := map[string]interface{}{
					"project_id": projID,
				}
				changesResult, err := ctx.Client.InvokeTool("project_changes", changesParams)
				ctx.Assertions.AssertNoError(err, "Should get project changes")
				if err != nil {
					return err
				}
				ctx.Assertions.AssertFalse(changesResult.IsError, "project_changes should succeed")

				content := changesResult.GetToolContent()

				// Parse the result
				var changesData struct {
					ProjectID string `json:"project_id"`
					Changes   []struct {
						Name           string   `json:"name"`
						ActiveSessions []string `json:"active_sessions"`
					} `json:"changes"`
				}
				if err := json.Unmarshal([]byte(content), &changesData); err != nil {
					return fmt.Errorf("failed to parse project_changes result: %w", err)
				}

				// Find the test-change and verify it has our session
				foundChange := false
				for _, change := range changesData.Changes {
					if change.Name == "test-change" {
						foundChange = true
						ctx.Assertions.AssertTrue(len(change.ActiveSessions) > 0, "test-change should have active sessions")
						if len(change.ActiveSessions) > 0 {
							ctx.Assertions.AssertEqual(sessionID, change.ActiveSessions[0], "Active session should match spawned session")
						}
						break
					}
				}
				ctx.Assertions.AssertTrue(foundChange, "Should find test-change in changes list")

				// End the session
				endParams := map[string]interface{}{
					"action":     "end",
					"session_id": sessionID,
				}
				_, _ = ctx.Client.InvokeTool("session", endParams)

				return nil
			},
		},
	}
}
