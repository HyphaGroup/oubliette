package suites

import (
	"time"

	testpkg "github.com/HyphaGroup/oubliette/test/pkg/testing"
)

// GetBasicTests returns basic smoke tests
func GetBasicTests() []*testpkg.TestCase {
	return []*testpkg.TestCase{
		{
			Name:        "test_connection",
			Description: "Verify MCP server connection and tool listing",
			Tags:        []string{"basic", "smoke"},
			Timeout:     10 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				// List tools
				tools, err := ctx.Client.ListTools()
				ctx.Assertions.AssertNoError(err, "Should list tools without error")

				if err != nil {
					return err
				}

				// Assert we have tools
				ctx.Assertions.AssertGreaterThan(len(tools), 0, "Should have at least 1 tool")

				// Check for expected tools
				hasListProjects := false
				hasProjectTool := false
				for _, tool := range tools {
					if tool.Name == "project" {
						hasProjectTool = true
						hasListProjects = true // project tool handles list action
					}
				}

				ctx.Assertions.AssertTrue(hasListProjects, "Should have project tool (handles list action)")
				ctx.Assertions.AssertTrue(hasProjectTool, "Should have project tool")

				return nil
			},
		},

		{
			Name:        "test_list_projects",
			Description: "Test listing projects",
			Tags:        []string{"basic", "project"},
			Timeout:     10 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				// Invoke project list
				result, err := ctx.Client.InvokeTool("project", map[string]interface{}{"action": "list"})
				ctx.Assertions.AssertNoError(err, "Should invoke project list without error")

				if err != nil {
					return err
				}

				ctx.Assertions.AssertFalse(result.IsError, "Should not return error result")

				// Check content
				content := result.GetToolContent()
				ctx.Assertions.AssertContains(content, "project", "Result should mention projects")

				return nil
			},
		},

		// NOTE: test_create_delete_project removed - covered by test_full_dev_workflow
		// NOTE: test_get_streaming_output removed - covered by test_streaming_merged_tool

		{
			Name:        "test_project_options",
			Description: "Test project_options returns configuration defaults",
			Tags:        []string{"basic", "project"},
			Timeout:     10 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				result, err := ctx.Client.InvokeTool("project", map[string]interface{}{"action": "options"})
				ctx.Assertions.AssertNoError(err, "Should invoke project_options without error")

				if err != nil {
					return err
				}

				ctx.Assertions.AssertFalse(result.IsError, "Should not return error result")

				content := result.GetToolContent()
				ctx.Log("project_options result: %s", content)

				// Should contain defaults section
				ctx.Assertions.AssertContains(content, "max_recursion_depth", "Should include max_recursion_depth default")
				ctx.Assertions.AssertContains(content, "max_agents_per_session", "Should include max_agents_per_session default")

				return nil
			},
		},

		{
			Name:        "test_caller_tool_response_invalid",
			Description: "Test caller_tool_response with invalid session/request IDs",
			Tags:        []string{"basic", "session"},
			Timeout:     10 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				// Call with non-existent session - should fail gracefully
				result, err := ctx.Client.InvokeTool("caller_tool_response", map[string]interface{}{
					"session_id": "nonexistent_session",
					"request_id": "nonexistent_request",
					"result":     map[string]string{"status": "ok"},
				})
				ctx.Assertions.AssertNoError(err, "Should invoke caller_tool_response without error")

				if err != nil {
					return err
				}

				// Should return an error since session doesn't exist
				ctx.Assertions.AssertTrue(result.IsError, "Should return error for non-existent session")
				content := result.GetToolContent()
				ctx.Log("caller_tool_response error: %s", content)
				ctx.Assertions.AssertContains(content, "not found", "Error should mention session not found")

				return nil
			},
		},
	}
}
