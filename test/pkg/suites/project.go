package suites

import (
	"fmt"
	"time"

	testpkg "github.com/HyphaGroup/oubliette/test/pkg/testing"
)

// GetProjectTests returns project management test suite
func GetProjectTests() []*testpkg.TestCase {
	return []*testpkg.TestCase{
		{
			Name:        "test_create_project",
			Description: "Test creating a new project with metadata",
			Tags:        []string{"project", "management"},
			Timeout:     30 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := "test-proj-create"
				
				// Pre-cleanup
				ctx.PreTestCleanup(projName)

				// Create project with description
				projID, err := ctx.CreateProject(projName, "Test project for creation validation")
				ctx.Assertions.AssertNoError(err, "Should create project successfully")

				if err != nil {
					return err
				}

				// Get project details
				params := map[string]interface{}{
					"action":     "get",
					"project_id": projID,
				}
				result, err := ctx.Client.InvokeTool("project", params)
				ctx.Assertions.AssertNoError(err, "Should get project details")

				if err == nil {
					content := result.GetToolContent()
					ctx.Assertions.AssertContains(content, projName, "Project details should contain name")
					ctx.Assertions.AssertContains(content, "Test project for creation validation", "Project details should contain description")
					ctx.Assertions.AssertContains(content, "oubliette-dev:latest", "Project should use default image")
				}

				return nil
			},
		},

		{
			Name:        "test_project_with_git",
			Description: "Test creating a project with git initialization",
			Tags:        []string{"project", "git"},
			Timeout:     30 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := "test-proj-git"

				// Pre-cleanup
				ctx.PreTestCleanup(projName)

				// Create project with git
				params := map[string]interface{}{
					"name":        projName,
					"description": "Test project with git",
					"init_git":    true,
				}

				ctx.Log("Creating project with git initialization")
				params["action"] = "create"
				result, err := ctx.Client.InvokeTool("project", params)
				ctx.Assertions.AssertNoError(err, "Should create project with git")

				if err != nil {
					return err
				}

				ctx.Assertions.AssertFalse(result.IsError, "Should not return error")

				// Extract project ID
				projID := testpkg.ExtractProjectID(result.GetToolContent())
				ctx.Assertions.AssertNotEmpty(projID, "Should have project ID")
				ctx.CreatedProjs = append(ctx.CreatedProjs, projID)

				// Get project details
				getParams := map[string]interface{}{
					"action":     "get",
					"project_id": projID,
				}
				getResult, err := ctx.Client.InvokeTool("project", getParams)
				ctx.Assertions.AssertNoError(err, "Should get project details")

				if err == nil {
					content := getResult.GetToolContent()
					ctx.Assertions.AssertContains(content, projName, "Should contain project name")
				}

				return nil
			},
		},

		{
			Name:        "test_list_multiple_projects",
			Description: "Test listing multiple projects",
			Tags:        []string{"project", "list"},
			Timeout:     45 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				// Create 3 projects
				projects := []string{
					"test-proj-multi-1",
					"test-proj-multi-2",
					"test-proj-multi-3",
				}
				
				// Pre-cleanup all
				for _, projName := range projects {
					ctx.PreTestCleanup(projName)
				}

				for _, projName := range projects {
					_, err := ctx.CreateProject(projName, fmt.Sprintf("Project %s", projName))
					ctx.Assertions.AssertNoError(err, fmt.Sprintf("Should create %s", projName))
					if err != nil {
						return err
					}
				}

				// List projects
				result, err := ctx.Client.InvokeTool("project", map[string]interface{}{"action": "list"})
				ctx.Assertions.AssertNoError(err, "Should list projects")

				if err != nil {
					return err
				}

				// Verify all projects are in the list
				content := result.GetToolContent()
				for _, projName := range projects {
					ctx.Assertions.AssertContains(content, projName, fmt.Sprintf("List should contain %s", projName))
				}

				return nil
			},
		},

		{
			Name:        "test_get_project",
			Description: "Test getting detailed project information",
			Tags:        []string{"project", "get"},
			Timeout:     30 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := "test-proj-get"
				description := "Test project for get validation"
				
				// Pre-cleanup
				ctx.PreTestCleanup(projName)

				// Create project
				projID, err := ctx.CreateProject(projName, description)
				ctx.Assertions.AssertNoError(err, "Should create project")

				if err != nil {
					return err
				}

				// Get project details
				params := map[string]interface{}{
					"action":     "get",
					"project_id": projID,
				}
				result, err := ctx.Client.InvokeTool("project", params)
				ctx.Assertions.AssertNoError(err, "Should get project")

				if err != nil {
					return err
				}

				ctx.Assertions.AssertFalse(result.IsError, "Should not return error")

				// Verify details
				content := result.GetToolContent()
				ctx.Assertions.AssertContains(content, projName, "Should contain project name")
				ctx.Assertions.AssertContains(content, description, "Should contain description")
				ctx.Assertions.AssertContains(content, "Image:", "Should show image info")
				ctx.Assertions.AssertContains(content, "Workspace:", "Should show workspace path")

				return nil
			},
		},

		{
			Name:        "test_delete_project",
			Description: "Test deleting a project",
			Tags:        []string{"project", "delete"},
			Timeout:     30 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := "test-proj-delete"
				
				// Pre-cleanup
				ctx.PreTestCleanup(projName)

				// Create project
				projID, err := ctx.CreateProject(projName, "Project to delete")
				ctx.Assertions.AssertNoError(err, "Should create project")

				if err != nil {
					return err
				}

				// Delete project
				ctx.Log("Deleting project: %s", projName)
				params := map[string]interface{}{
					"action":     "delete",
					"project_id": projID,
				}
				result, err := ctx.Client.InvokeTool("project", params)
				ctx.Assertions.AssertNoError(err, "Should delete project")

				if err == nil {
					ctx.Assertions.AssertFalse(result.IsError, "Should not return error")
					content := result.GetToolContent()
					ctx.Assertions.AssertContains(content, "deleted", "Should confirm deletion")
				}

				// Remove from cleanup list since we already deleted it
				ctx.CreatedProjs = []string{}

				// Verify project is gone
				listResult, err := ctx.Client.InvokeTool("project", map[string]interface{}{"action": "list"})
				if err == nil {
					listContent := listResult.GetToolContent()
					ctx.Assertions.AssertNotContains(listContent, projName, "Deleted project should not be in list")
				}

				return nil
			},
		},

		{
			Name:        "test_project_with_container_type",
			Description: "Test creating a project with specific container type",
			Tags:        []string{"project", "container-types"},
			Timeout:     30 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := "test-proj-container-type"

				// Pre-cleanup
				ctx.PreTestCleanup(projName)

				// Create project with osint container type
				params := map[string]interface{}{
					"action":         "create",
					"name":           projName,
					"description":    "Test project with osint container type",
					"container_type": "osint",
				}

				ctx.Log("Creating project with osint container type")
				result, err := ctx.Client.InvokeTool("project", params)
				ctx.Assertions.AssertNoError(err, "Should create project with container_type")

				if err != nil {
					return err
				}

				ctx.Assertions.AssertFalse(result.IsError, "Should not return error")

				// Extract project ID
				projID := testpkg.ExtractProjectID(result.GetToolContent())
				ctx.Assertions.AssertNotEmpty(projID, "Should have project ID")
				ctx.CreatedProjs = append(ctx.CreatedProjs, projID)

				// Get project details
				getParams := map[string]interface{}{
					"action":     "get",
					"project_id": projID,
				}
				getResult, err := ctx.Client.InvokeTool("project", getParams)
				ctx.Assertions.AssertNoError(err, "Should get project details")

				if err == nil {
					content := getResult.GetToolContent()
					ctx.Assertions.AssertContains(content, "oubliette-osint", "Project should use osint image")
					ctx.Assertions.AssertContains(content, "osint", "Project should have osint container type")
				}

				return nil
			},
		},

		{
			Name:        "test_project_with_agent_runtime",
			Description: "Test creating a project with agent_runtime parameter",
			Tags:        []string{"project", "runtime"},
			Timeout:     30 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := "test-proj-runtime"

				// Pre-cleanup
				ctx.PreTestCleanup(projName)

				// Create project with agent_runtime set
				// Note: We use "droid" since it's the only implemented runtime
				// When opencode is implemented, this test verifies the parameter is persisted
				params := map[string]interface{}{
					"action":        "create",
					"name":          projName,
					"description":   "Test project with agent runtime",
					"agent_runtime": "droid",
				}

				ctx.Log("Creating project with agent_runtime parameter")
				result, err := ctx.Client.InvokeTool("project", params)
				ctx.Assertions.AssertNoError(err, "Should create project with agent_runtime")

				if err != nil {
					return err
				}

				ctx.Assertions.AssertFalse(result.IsError, "Should not return error")
				content := result.GetToolContent()
				ctx.Log("project_create result: %s", content)

				// Extract project ID
				projID := testpkg.ExtractProjectID(content)
				ctx.Assertions.AssertNotEmpty(projID, "Should have project ID")
				ctx.CreatedProjs = append(ctx.CreatedProjs, projID)

				// Verify project_options shows available runtimes
				optResult, err := ctx.Client.InvokeTool("project", map[string]interface{}{"action": "options"})
				ctx.Assertions.AssertNoError(err, "Should get project options")

				if err == nil {
					optContent := optResult.GetToolContent()
					ctx.Assertions.AssertContains(optContent, "agent_runtimes", "Should list agent runtimes")
					ctx.Assertions.AssertContains(optContent, "droid", "Should list droid runtime")
				}

				return nil
			},
		},
	}
}
