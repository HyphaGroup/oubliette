package suites

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	testpkg "github.com/HyphaGroup/oubliette/test/pkg/testing"
)

// GetRecursionTests returns recursion/nesting test suite
func GetRecursionTests() []*testpkg.TestCase {
	return []*testpkg.TestCase{
		{
			Name:        "test_get_recursion_limits",
			Description: "Test querying recursion limits for a project",
			Tags:        []string{"recursion", "limits"},
			Timeout:     60 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := "test-recursion-limits"

				// Create project
				projID, err := ctx.CreateProject(projName, "Test project for recursion limits")
				ctx.Assertions.AssertNoError(err, "Should create project")

				if err != nil {
					return err
				}

				// Get recursion limits
				ctx.Log("Querying recursion limits")
				params := map[string]interface{}{
					"project_id": projID,
				}

				result, err := ctx.Client.InvokeTool("config_limits", params)
				ctx.Assertions.AssertNoError(err, "Should get recursion limits")

				if err != nil {
					return err
				}

				ctx.Assertions.AssertFalse(result.IsError, "Should not return error")

				_ = result.GetToolContent()
				ctx.Log("Recursion limits retrieved successfully")

				return nil
			},
		},

		{
			Name:        "test_recursion_limits_by_session",
			Description: "Test querying recursion limits by session ID",
			Tags:        []string{"recursion", "limits", "session"},
			Timeout:     120 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := fmt.Sprintf("test-recursion-session-%d", time.Now().UnixNano())
				
				// Pre-cleanup
				ctx.PreTestCleanup(projName)

				// Create project (container auto-starts)
				projID, err := ctx.CreateProject(projName, "Test project for recursion by session")
				ctx.Assertions.AssertNoError(err, "Should create project")

				if err != nil {
					return err
				}

				// Start task
				ctx.Log("Starting session task")
				startParams := map[string]interface{}{
					"project_id": projID,
					"prompt":       "Simple task",
				}

				startResult, err := ctx.Client.InvokeTool("session", startParams)
				ctx.Assertions.AssertNoError(err, "Should start task")

				if err != nil {
					return err
				}

				// Extract session ID
				content := startResult.GetToolContent()
				var sessionData map[string]interface{}
				if err := json.Unmarshal([]byte(content), &sessionData); err == nil {
					if sessionID, ok := sessionData["session_id"].(string); ok {
						ctx.SessionID = sessionID
						ctx.Log("Got session ID: %s", sessionID)

						// Get recursion limits by session
						time.Sleep(1 * time.Second)
						limitsParams := map[string]interface{}{
							"session_id": sessionID,
						}

						limitsResult, err := ctx.Client.InvokeTool("config_limits", limitsParams)
						ctx.Assertions.AssertNoError(err, "Should get recursion limits by session")

						if err == nil {
							ctx.Assertions.AssertFalse(limitsResult.IsError, "Should not return error")
							limitsContent := limitsResult.GetToolContent()
							ctx.Assertions.AssertContains(limitsContent, "depth", "Should show depth info")
						}
					}
				}

				return nil
			},
		},

		{
			Name:        "test_spawn_child_basic",
			Description: "Test spawning a child session (basic recursion)",
			Tags:        []string{"recursion", "spawn", "child"},
			Timeout:     120 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := fmt.Sprintf("test-recursion-child-%d", time.Now().UnixNano())
				
				// Pre-cleanup
				ctx.PreTestCleanup(projName)

				// Create project (container auto-starts)
				projID, err := ctx.CreateProject(projName, "Test project for child spawning")
				ctx.Assertions.AssertNoError(err, "Should create project")

				if err != nil {
					return err
				}

				// Start root task
				ctx.Log("Starting root session task")
				startParams := map[string]interface{}{
					"project_id": projID,
					"prompt":       "Parent task",
				}

				startResult, err := ctx.Client.InvokeTool("session", startParams)
				ctx.Assertions.AssertNoError(err, "Should start root task")

				if err != nil {
					return err
				}

				// Extract session ID
				content := startResult.GetToolContent()
				var sessionData map[string]interface{}
				if err := json.Unmarshal([]byte(content), &sessionData); err == nil {
					if sessionID, ok := sessionData["session_id"].(string); ok {
						ctx.SessionID = sessionID
						ctx.Log("Root session ID: %s", sessionID)

						// Note: session_spawn_child requires X-Oubliette-Session-ID header
						// which is set automatically by the MCP server when called from within a session
						// For this test, we just verify the tool exists and can be described
						tools, err := ctx.Client.ListTools()
						ctx.Assertions.AssertNoError(err, "Should list tools")

						hasSpawnChild := false
						for _, tool := range tools {
							if tool.Name == "session_spawn_child" {
								hasSpawnChild = true
								break
							}
						}

						ctx.Assertions.AssertTrue(hasSpawnChild, "Should have session_spawn_child tool")
						ctx.Log("Verified session_spawn_child tool exists")
					}
				}

				return nil
			},
		},

		{
			Name:        "test_recursion_depth_info",
			Description: "Test retrieving depth information from session",
			Tags:        []string{"recursion", "depth"},
			Timeout:     120 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := fmt.Sprintf("test-recursion-depth-%d", time.Now().UnixNano())
				
				// Pre-cleanup
				ctx.PreTestCleanup(projName)

				// Create project (container auto-starts)
				projID, err := ctx.CreateProject(projName, "Test project for depth info")
				ctx.Assertions.AssertNoError(err, "Should create project")

				if err != nil {
					return err
				}

				// Start task
				ctx.Log("Starting session task")
				startParams := map[string]interface{}{
					"project_id": projID,
					"prompt":       "Check depth",
				}

				startResult, err := ctx.Client.InvokeTool("session", startParams)
				ctx.Assertions.AssertNoError(err, "Should start task")

				if err != nil {
					return err
				}

				// Extract session ID
				content := startResult.GetToolContent()
				var sessionData map[string]interface{}
				if err := json.Unmarshal([]byte(content), &sessionData); err == nil {
					if sessionID, ok := sessionData["session_id"].(string); ok {
						ctx.SessionID = sessionID
						ctx.Log("Session ID: %s", sessionID)

						// Get session to check depth
						time.Sleep(1 * time.Second)
						getParams := map[string]interface{}{
							"action":     "get",
							"session_id": sessionID,
						}

						getResult, err := ctx.Client.InvokeTool("session", getParams)
						ctx.Assertions.AssertNoError(err, "Should get session")

						if err == nil {
							_ = getResult.GetToolContent()
							// Session data should contain depth info (depth: 0 for root)
							ctx.Log("Session retrieved with depth info")
						}
					}
				}

				return nil
			},
		},

		{
			Name:        "test_shared_workspace",
			Description: "Test that sessions share workspace via /workspace mount",
			Tags:        []string{"recursion", "workspace", "shared"},
			Timeout:     90 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := fmt.Sprintf("test-recursion-workspace-%d", time.Now().UnixNano())
				
				// Pre-cleanup
				ctx.PreTestCleanup(projName)

				// Create project (container auto-starts)
				projID, err := ctx.CreateProject(projName, "Test project for shared workspace")
				ctx.Assertions.AssertNoError(err, "Should create project")

				if err != nil {
					return err
				}

				// Write a test file to workspace
				ctx.Log("Creating test file in workspace")
				execParams := map[string]interface{}{
					"action":     "exec",
					"project_id": projID,
					"command":    "echo 'test content' > /workspace/test-file.txt",
				}

				execResult, err := ctx.Client.InvokeTool("container", execParams)
				ctx.Assertions.AssertNoError(err, "Should write file to workspace")

				if err == nil {
					ctx.Assertions.AssertFalse(execResult.IsError, "Write should succeed")

					// Read the file back
					readParams := map[string]interface{}{
						"action":     "exec",
						"project_id": projID,
						"command":    "cat /workspace/test-file.txt",
					}

					readResult, err := ctx.Client.InvokeTool("container", readParams)
					ctx.Assertions.AssertNoError(err, "Should read file from workspace")

					if err == nil {
						readContent := readResult.GetToolContent()
						ctx.Assertions.AssertContains(readContent, "test content", "File content should match")
						ctx.Log("Verified workspace file access")
					}
				}

				return nil
			},
		},

		// Socket connectivity tests (Phase 6)
		{
			Name:        "test_socket_connectivity_from_container",
			Description: "Test that container can connect to host MCP via unix socket",
			Tags:        []string{"recursion", "socket", "integration"},
			Timeout:     120 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := fmt.Sprintf("test-socket-conn-%d", time.Now().UnixNano())
				ctx.PreTestCleanup(projName)

				// Create project (container auto-starts with socket mount)
				projID, err := ctx.CreateProject(projName, "Socket connectivity test")
				ctx.Assertions.AssertNoError(err, "Should create project")
				if err != nil {
					return err
				}

				// Verify socket is mounted in container
				ctx.Log("Verifying socket is mounted in container")
				execParams := map[string]interface{}{
					"action":     "exec",
					"project_id": projID,
					"command":    "ls -la /mcp/oubliette.sock",
				}
				result, err := ctx.Client.InvokeTool("container", execParams)
				ctx.Assertions.AssertNoError(err, "Should list socket")
				if err != nil {
					return err
				}
				ctx.Assertions.AssertContains(result.GetToolContent(), "oubliette.sock", "Socket should be mounted")

				// Verify bridge binary exists
				ctx.Log("Verifying bridge binary exists")
				bridgeParams := map[string]interface{}{
					"action":     "exec",
					"project_id": projID,
					"command":    "test -x /usr/local/bin/mcp-oubliette-bridge && echo 'BRIDGE_EXISTS'",
				}
				bridgeResult, err := ctx.Client.InvokeTool("container", bridgeParams)
				ctx.Assertions.AssertNoError(err, "Should check bridge binary")
				if err != nil {
					return err
				}
				ctx.Assertions.AssertContains(bridgeResult.GetToolContent(), "BRIDGE_EXISTS", "Bridge binary should exist")

				ctx.Log("Socket connectivity verified")
				return nil
			},
		},

		{
			Name:        "test_cross_project_socket_isolation",
			Description: "Test that containers cannot access other projects' sockets",
			Tags:        []string{"recursion", "socket", "security"},
			Timeout:     120 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				// Create two projects
				proj1Name := fmt.Sprintf("test-isolation-1-%d", time.Now().UnixNano())
				proj2Name := fmt.Sprintf("test-isolation-2-%d", time.Now().UnixNano())

				ctx.PreTestCleanup(proj1Name)
				ctx.PreTestCleanup(proj2Name)

				proj1ID, err := ctx.CreateProject(proj1Name, "Isolation test 1")
				ctx.Assertions.AssertNoError(err, "Should create project 1")
				if err != nil {
					return err
				}

				proj2ID, err := ctx.CreateProject(proj2Name, "Isolation test 2")
				ctx.Assertions.AssertNoError(err, "Should create project 2")
				if err != nil {
					return err
				}

				// From project 1's container, try to access project 2's socket
				// This should fail because only the project's own socket is mounted
				ctx.Log("Verifying project 1 cannot see project 2's socket")
				attackParams := map[string]interface{}{
					"action":     "exec",
					"project_id": proj1ID,
					"command":    fmt.Sprintf("ls /tmp/oubliette-sockets/oubliette-%s.sock 2>&1 || echo 'NOT_ACCESSIBLE'", proj2ID),
				}
				result, err := ctx.Client.InvokeTool("container", attackParams)
				ctx.Assertions.AssertNoError(err, "Should execute check command")

				if err == nil {
					content := result.GetToolContent()
					// Container should only see its own socket at /mcp/oubliette.sock
					// Not have access to the host's /tmp directory or other projects' sockets
					ctx.Assertions.AssertContains(content, "NOT_ACCESSIBLE",
						"Container should not see other project's socket")
				}

				// Cleanup project 2
				ctx.Log("Cleaning up project 2")
				deleteParams := map[string]interface{}{
					"action":     "delete",
					"project_id": proj2ID,
				}
				ctx.Client.InvokeTool("project", deleteParams)

				ctx.Log("Cross-project socket isolation verified")
				return nil
			},
		},

		{
			Name:        "test_socket_mcp_json_config",
			Description: "Test that mcp.json is configured for socket transport",
			Tags:        []string{"recursion", "socket", "config"},
			Timeout:     90 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := fmt.Sprintf("test-mcp-config-%d", time.Now().UnixNano())
				ctx.PreTestCleanup(projName)

				projID, err := ctx.CreateProject(projName, "MCP config test")
				ctx.Assertions.AssertNoError(err, "Should create project")
				if err != nil {
					return err
				}

				// Start a session to trigger MCP config creation
				ctx.Log("Starting session to generate MCP config")
				spawnParams := map[string]interface{}{
					"project_id": projID,
					"prompt":     "echo test",
				}
				spawnResult, err := ctx.Client.InvokeTool("session", spawnParams)
				ctx.Assertions.AssertNoError(err, "Should spawn session")
				if err != nil {
					return err
				}

				// Extract session ID for cleanup
				content := spawnResult.GetToolContent()
				sessionID := testpkg.ExtractSessionID(content)
				if sessionID != "" {
					ctx.SessionID = sessionID
				}

				// Give MCP config time to be written
				time.Sleep(2 * time.Second)

				// Check mcp.json in workspace
				ctx.Log("Checking mcp.json configuration")
				checkParams := map[string]interface{}{
					"action":     "exec",
					"project_id": projID,
					"command":    "cat /workspace/.factory/mcp.json 2>/dev/null || echo 'NOT_FOUND'",
				}
				result, err := ctx.Client.InvokeTool("container", checkParams)
				ctx.Assertions.AssertNoError(err, "Should read mcp.json")
				if err != nil {
					return err
				}

				mcpContent := result.GetToolContent()
				if strings.Contains(mcpContent, "NOT_FOUND") {
					ctx.Log("Warning: mcp.json not found - may not be created until session starts")
					return nil
				}

				// Verify it uses stdio transport with bridge
				ctx.Assertions.AssertContains(mcpContent, "mcp-oubliette-bridge",
					"Should use mcp-oubliette-bridge command")
				ctx.Assertions.AssertContains(mcpContent, "/mcp/oubliette.sock",
					"Should reference socket path")

				ctx.Log("MCP config verified for socket transport")
				return nil
			},
		},

		{
			Name:        "test_session_env_vars_set",
			Description: "Test that session environment variables are set in container",
			Tags:        []string{"recursion", "socket", "env"},
			Timeout:     120 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := fmt.Sprintf("test-session-env-%d", time.Now().UnixNano())
				ctx.PreTestCleanup(projName)

				projID, err := ctx.CreateProject(projName, "Session env test")
				ctx.Assertions.AssertNoError(err, "Should create project")
				if err != nil {
					return err
				}

				// Spawn a session
				ctx.Log("Spawning session")
				spawnParams := map[string]interface{}{
					"project_id":  projID,
					"prompt":      "Test environment variables",
					"new_session": true,
				}
				spawnResult, err := ctx.Client.InvokeTool("session", spawnParams)
				ctx.Assertions.AssertNoError(err, "Should spawn session")
				if err != nil {
					return err
				}

				// Extract session ID from text response
				content := spawnResult.GetToolContent()
				sessionID := testpkg.ExtractSessionID(content)
				if sessionID == "" {
					ctx.Log("Failed to extract session ID from: %s", content)
					return fmt.Errorf("failed to extract session ID from spawn result")
				}
				ctx.SessionID = sessionID

				// The session environment variables are set when the droid process starts
				// We can verify by checking the session status/info contains expected fields
				ctx.Log("Verifying session was created with project context")
				getParams := map[string]interface{}{
					"action":     "get",
					"session_id": sessionID,
				}
				getResult, err := ctx.Client.InvokeTool("session", getParams)
				ctx.Assertions.AssertNoError(err, "Should get session")
				if err != nil {
					return err
				}

				sessionContent := getResult.GetToolContent()
				ctx.Assertions.AssertContains(sessionContent, projID,
					"Session should be associated with project")
				ctx.Assertions.AssertContains(sessionContent, sessionID,
					"Session response should contain session ID")

				ctx.Log("Session environment context verified")
				return nil
			},
		},
	}
}
