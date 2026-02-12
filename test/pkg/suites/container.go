package suites

import (
	"fmt"
	"time"

	testpkg "github.com/HyphaGroup/oubliette/test/pkg/testing"
)

// GetContainerTests returns container operations test suite
func GetContainerTests() []*testpkg.TestCase {
	return []*testpkg.TestCase{
		{
			Name:        "test_spawn_container",
			Description: "Test spawning a container for a project",
			Tags:        []string{"container", "spawn"},
			Timeout:     60 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				// Use unique name to prevent container conflicts across test runs
				projName := fmt.Sprintf("test-container-spawn-%d", time.Now().UnixNano())

				// Pre-cleanup
				ctx.PreTestCleanup(projName)

				// Create project (container auto-starts)
				projID, err := ctx.CreateProject(projName, "Test project for container spawn")
				ctx.Assertions.AssertNoError(err, "Should create project")

				if err != nil {
					return err
				}

				// Verify container is running (auto-started by CreateProject)
				params := map[string]interface{}{
					"action":     "get",
					"project_id": projID,
				}
				result, err := ctx.Client.InvokeTool("project", params)
				ctx.Assertions.AssertNoError(err, "Should get project status")

				if err == nil {
					content := result.GetToolContent()
					ctx.Assertions.AssertContains(content, "running", "Container should be running")
				}

				return nil
			},
		},

		{
			Name:        "test_exec_command",
			Description: "Test executing a command in a running container",
			Tags:        []string{"container", "exec"},
			Timeout:     60 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := fmt.Sprintf("test-container-exec-%d", time.Now().UnixNano())

				// Pre-cleanup
				ctx.PreTestCleanup(projName)

				// Create project (container auto-starts)
				projID, err := ctx.CreateProject(projName, "Test project for exec")
				ctx.Assertions.AssertNoError(err, "Should create project")

				if err != nil {
					return err
				}

				// Execute command
				ctx.Log("Executing command in container")
				params := map[string]interface{}{
					"action":     "exec",
					"project_id": projID,
					"command":    "echo 'Hello from container'",
				}

				result, err := ctx.Client.InvokeTool("container", params)
				ctx.Assertions.AssertNoError(err, "Should execute command")

				if err == nil {
					ctx.Assertions.AssertFalse(result.IsError, "Command should succeed")
					content := result.GetToolContent()
					ctx.Assertions.AssertContains(content, "Hello from container", "Should see command output")
				}

				return nil
			},
		},

		{
			Name:        "test_container_logs",
			Description: "Test retrieving container logs",
			Tags:        []string{"container", "logs"},
			Timeout:     60 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := fmt.Sprintf("test-container-logs-%d", time.Now().UnixNano())

				// Pre-cleanup
				ctx.PreTestCleanup(projName)

				// Create project (container auto-starts)
				projID, err := ctx.CreateProject(projName, "Test project for logs")
				ctx.Assertions.AssertNoError(err, "Should create project")

				if err != nil {
					return err
				}

				// Get logs
				ctx.Log("Retrieving container logs")
				params := map[string]interface{}{
					"action":     "logs",
					"project_id": projID,
				}

				result, err := ctx.Client.InvokeTool("container", params)
				ctx.Assertions.AssertNoError(err, "Should get logs")

				if err == nil {
					ctx.Assertions.AssertFalse(result.IsError, "Should not return error")
					// Logs might be empty or contain initialization messages
					ctx.Log("Retrieved logs successfully")
				}

				return nil
			},
		},

		{
			Name:        "test_stop_container",
			Description: "Test stopping a running container",
			Tags:        []string{"container", "stop"},
			Timeout:     60 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := fmt.Sprintf("test-container-stop-%d", time.Now().UnixNano())

				// Pre-cleanup
				ctx.PreTestCleanup(projName)

				// Create project (container auto-starts)
				projID, err := ctx.CreateProject(projName, "Test project for stop")
				ctx.Assertions.AssertNoError(err, "Should create project")

				if err != nil {
					return err
				}

				// Verify container is running
				getParams := map[string]interface{}{
					"action":     "get",
					"project_id": projID,
				}
				result, err := ctx.Client.InvokeTool("project", getParams)
				if err == nil {
					content := result.GetToolContent()
					ctx.Assertions.AssertContains(content, "running", "Container should be running before stop")
				}

				// Stop container
				ctx.Log("Stopping container")
				stopParams := map[string]interface{}{
					"action":     "stop",
					"project_id": projID,
				}

				stopResult, err := ctx.Client.InvokeTool("container", stopParams)
				ctx.Assertions.AssertNoError(err, "Should stop container")

				if err == nil {
					ctx.Assertions.AssertFalse(stopResult.IsError, "Stop should succeed")
				}

				// Verify container is stopped
				result2, err := ctx.Client.InvokeTool("project", getParams)
				if err == nil {
					content := result2.GetToolContent()
					// Check that container is stopped (might be "stopped" or "not running")
					ctx.Assertions.AssertNotContains(content, "Container: running", "Container should be stopped after stop")
				}

				return nil
			},
		},

		{
			Name:        "test_refresh_container",
			Description: "Test refreshing container image for a project",
			Tags:        []string{"container", "refresh"},
			Timeout:     120 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := fmt.Sprintf("test-container-refresh-%d", time.Now().UnixNano())

				// Pre-cleanup
				ctx.PreTestCleanup(projName)

				// Create project
				projID, err := ctx.CreateProject(projName, "Test project for refresh")
				ctx.Assertions.AssertNoError(err, "Should create project")

				if err != nil {
					return err
				}

				// Refresh container (pulls image and restarts)
				ctx.Log("Refreshing container image")
				params := map[string]interface{}{
					"project_id": projID,
				}

				result, err := ctx.Client.InvokeTool("container_refresh", params)
				ctx.Assertions.AssertNoError(err, "Should refresh container")

				if err == nil {
					ctx.Assertions.AssertFalse(result.IsError, "Refresh should succeed")
					ctx.Log("Container refreshed successfully")
				}

				return nil
			},
		},

		{
			Name:        "test_start_container",
			Description: "Test stopping and restarting a container",
			Tags:        []string{"container", "start"},
			Timeout:     90 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := fmt.Sprintf("test-container-start-%d", time.Now().UnixNano())

				// Pre-cleanup
				ctx.PreTestCleanup(projName)

				// Create project (container auto-starts)
				projID, err := ctx.CreateProject(projName, "Test project for start")
				ctx.Assertions.AssertNoError(err, "Should create project")

				if err != nil {
					return err
				}

				// Verify container is running after create
				getParams := map[string]interface{}{
					"action":     "get",
					"project_id": projID,
				}
				getResult, err := ctx.Client.InvokeTool("project", getParams)
				if err == nil {
					content := getResult.GetToolContent()
					ctx.Assertions.AssertContains(content, "running", "Container should be running after create")
				}

				// Stop container
				ctx.Log("Stopping container")
				stopParams := map[string]interface{}{
					"action":     "stop",
					"project_id": projID,
				}
				_, err = ctx.Client.InvokeTool("container", stopParams)
				ctx.Assertions.AssertNoError(err, "Should stop container")

				// Start container again
				ctx.Log("Starting container again")
				startParams := map[string]interface{}{
					"action":     "start",
					"project_id": projID,
				}

				result, err := ctx.Client.InvokeTool("container", startParams)
				ctx.Assertions.AssertNoError(err, "Should start container")

				if err == nil {
					ctx.Assertions.AssertFalse(result.IsError, "Start should succeed")
					ctx.Log("Container restarted successfully")

					// Verify container is running
					getResult2, err := ctx.Client.InvokeTool("project", getParams)
					if err == nil {
						content := getResult2.GetToolContent()
						ctx.Assertions.AssertContains(content, "running", "Container should be running after restart")
					}
				}

				return nil
			},
		},
	}
}
