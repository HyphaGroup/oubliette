package suites

import (
	"strings"
	"time"

	testpkg "github.com/HyphaGroup/oubliette/test/pkg/testing"
)

// GetAuthTests returns authentication-related tests
func GetAuthTests() []*testpkg.TestCase {
	return []*testpkg.TestCase{
		{
			Name:        "test_auth_token_tools",
			Description: "Test token_list, token_create, and token_revoke tools (requires admin scope)",
			Tags:        []string{"auth", "admin"},
			Timeout:     30 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				// List tokens (should succeed if we have admin scope)
				listResult, err := ctx.Client.InvokeTool("token", map[string]interface{}{"action": "list"})
				ctx.Assertions.AssertNoError(err, "Should invoke token_list without error")

				if err != nil {
					return err
				}

				ctx.Assertions.AssertFalse(listResult.IsError, "token_list should not return error")
				listContent := listResult.GetToolContent()
				ctx.Log("token_list result: %s", listContent)

				// Create a test token
				createResult, err := ctx.Client.InvokeTool("token", map[string]interface{}{"action": "create",
					"name":  "test-token-integration",
					"scope": "read-only",
				})
				ctx.Assertions.AssertNoError(err, "Should invoke token_create without error")

				if err != nil {
					return err
				}

				ctx.Assertions.AssertFalse(createResult.IsError, "token_create should not return error")
				createContent := createResult.GetToolContent()
				ctx.Log("token_create result: %s", createContent)
				ctx.Assertions.AssertContains(createContent, "Token created successfully", "Should confirm token creation")
				ctx.Assertions.AssertContains(createContent, "oub_", "Should include token ID with prefix")

				// Extract token ID from result (look for oub_ prefix)
				var tokenID string
				for _, line := range strings.Split(createContent, "\n") {
					if strings.Contains(line, "oub_") {
						// Find the token ID
						parts := strings.Fields(line)
						for _, part := range parts {
							if strings.HasPrefix(part, "oub_") {
								tokenID = part
								break
							}
						}
					}
				}
				ctx.Log("Extracted token ID: %s", tokenID)
				ctx.Assertions.AssertTrue(tokenID != "", "Should extract token ID from response")

				// Verify the token appears in list
				listResult2, err := ctx.Client.InvokeTool("token", map[string]interface{}{"action": "list"})
				ctx.Assertions.AssertNoError(err, "Should invoke token_list again without error")

				if err != nil {
					return err
				}

				listContent2 := listResult2.GetToolContent()
				ctx.Assertions.AssertContains(listContent2, "test-token-integration", "Token should appear in list")

				// Revoke the token
				revokeResult, err := ctx.Client.InvokeTool("token", map[string]interface{}{"action": "revoke",
					"token_id": tokenID,
				})
				ctx.Assertions.AssertNoError(err, "Should invoke token_revoke without error")

				if err != nil {
					return err
				}

				ctx.Assertions.AssertFalse(revokeResult.IsError, "token_revoke should not return error")
				revokeContent := revokeResult.GetToolContent()
				ctx.Log("token_revoke result: %s", revokeContent)
				ctx.Assertions.AssertContains(revokeContent, "revoked successfully", "Should confirm token revocation")

				// Verify the token no longer appears in list
				listResult3, err := ctx.Client.InvokeTool("token", map[string]interface{}{"action": "list"})
				ctx.Assertions.AssertNoError(err, "Should invoke token_list after revocation without error")

				if err != nil {
					return err
				}

				listContent3 := listResult3.GetToolContent()
				ctx.Assertions.AssertNotContains(listContent3, "test-token-integration", "Revoked token should not appear in list")

				return nil
			},
		},

		{
			Name:        "test_auth_token_create_invalid_scope",
			Description: "Test that token_create rejects invalid scopes",
			Tags:        []string{"auth", "admin", "validation"},
			Timeout:     10 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				// Try to create token with invalid scope
				result, err := ctx.Client.InvokeTool("token", map[string]interface{}{"action": "create",
					"name":  "bad-scope-token",
					"scope": "invalid-scope",
				})

				// The call may return an error or a tool error result
				if err != nil {
					ctx.Log("Got expected error: %v", err)
					ctx.Assertions.AssertContains(err.Error(), "invalid scope", "Error should mention invalid scope")
					return nil
				}

				// Check if tool returned error in result
				if result.IsError {
					content := result.GetToolContent()
					ctx.Log("Got expected tool error: %s", content)
					ctx.Assertions.AssertContains(content, "invalid scope", "Tool error should mention invalid scope")
					return nil
				}

				// If we get here, something is wrong
				ctx.Assertions.Fail("Should have rejected invalid scope")
				return nil
			},
		},

		{
			Name:        "test_auth_token_create_missing_params",
			Description: "Test that token_create rejects missing parameters",
			Tags:        []string{"auth", "admin", "validation"},
			Timeout:     10 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				// Try to create token without name
				result, err := ctx.Client.InvokeTool("token", map[string]interface{}{"action": "create",
					"scope": "read-only",
				})

				if err != nil {
					ctx.Log("Got expected error: %v", err)
					return nil
				}

				if result.IsError {
					content := result.GetToolContent()
					ctx.Log("Got expected tool error: %s", content)
					return nil
				}

				ctx.Assertions.Fail("Should have rejected missing name parameter")
				return nil
			},
		},

		{
			Name:        "test_auth_token_create_admin_ro_scope",
			Description: "Test creating a token with admin:ro scope",
			Tags:        []string{"auth", "admin", "scopes"},
			Timeout:     15 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				// Create a token with admin:ro scope
				createResult, err := ctx.Client.InvokeTool("token", map[string]interface{}{"action": "create",
					"name":  "test-admin-ro-token",
					"scope": "admin:ro",
				})
				ctx.Assertions.AssertNoError(err, "Should invoke token_create without error")
				if err != nil {
					return err
				}

				ctx.Assertions.AssertFalse(createResult.IsError, "token_create should not return error")
				createContent := createResult.GetToolContent()
				ctx.Log("token_create result: %s", createContent)
				ctx.Assertions.AssertContains(createContent, "Token created successfully", "Should confirm token creation")
				ctx.Assertions.AssertContains(createContent, "admin:ro", "Should show admin:ro scope")

				// Extract token ID and clean up
				var tokenID string
				for _, line := range strings.Split(createContent, "\n") {
					if strings.Contains(line, "oub_") {
						parts := strings.Fields(line)
						for _, part := range parts {
							if strings.HasPrefix(part, "oub_") {
								tokenID = part
								break
							}
						}
					}
				}

				if tokenID != "" {
					ctx.Client.InvokeTool("token", map[string]interface{}{"action": "revoke", "token_id": tokenID})
				}

				return nil
			},
		},

		{
			Name:        "test_auth_token_create_project_scope",
			Description: "Test creating a token with project scope",
			Tags:        []string{"auth", "admin", "scopes"},
			Timeout:     15 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				// Create a token with project scope
				createResult, err := ctx.Client.InvokeTool("token", map[string]interface{}{"action": "create",
					"name":  "test-project-scope-token",
					"scope": "project:test-project-123",
				})
				ctx.Assertions.AssertNoError(err, "Should invoke token_create without error")
				if err != nil {
					return err
				}

				ctx.Assertions.AssertFalse(createResult.IsError, "token_create should not return error")
				createContent := createResult.GetToolContent()
				ctx.Log("token_create result: %s", createContent)
				ctx.Assertions.AssertContains(createContent, "Token created successfully", "Should confirm token creation")
				ctx.Assertions.AssertContains(createContent, "project:test-project-123", "Should show project scope")

				// Extract token ID and clean up
				var tokenID string
				for _, line := range strings.Split(createContent, "\n") {
					if strings.Contains(line, "oub_") {
						parts := strings.Fields(line)
						for _, part := range parts {
							if strings.HasPrefix(part, "oub_") {
								tokenID = part
								break
							}
						}
					}
				}

				if tokenID != "" {
					ctx.Client.InvokeTool("token", map[string]interface{}{"action": "revoke", "token_id": tokenID})
				}

				return nil
			},
		},

		{
			Name:        "test_auth_token_create_project_ro_scope",
			Description: "Test creating a token with project:ro scope",
			Tags:        []string{"auth", "admin", "scopes"},
			Timeout:     15 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				// Create a token with project:ro scope
				createResult, err := ctx.Client.InvokeTool("token", map[string]interface{}{"action": "create",
					"name":  "test-project-ro-token",
					"scope": "project:test-project-456:ro",
				})
				ctx.Assertions.AssertNoError(err, "Should invoke token_create without error")
				if err != nil {
					return err
				}

				ctx.Assertions.AssertFalse(createResult.IsError, "token_create should not return error")
				createContent := createResult.GetToolContent()
				ctx.Log("token_create result: %s", createContent)
				ctx.Assertions.AssertContains(createContent, "Token created successfully", "Should confirm token creation")
				ctx.Assertions.AssertContains(createContent, "project:test-project-456:ro", "Should show project:ro scope")

				// Extract token ID and clean up
				var tokenID string
				for _, line := range strings.Split(createContent, "\n") {
					if strings.Contains(line, "oub_") {
						parts := strings.Fields(line)
						for _, part := range parts {
							if strings.HasPrefix(part, "oub_") {
								tokenID = part
								break
							}
						}
					}
				}

				if tokenID != "" {
					ctx.Client.InvokeTool("token", map[string]interface{}{"action": "revoke", "token_id": tokenID})
				}

				return nil
			},
		},

		{
			Name:        "test_auth_project_options_shows_token_scopes",
			Description: "Test that project_options includes token scope documentation",
			Tags:        []string{"auth", "scopes"},
			Timeout:     10 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				result, err := ctx.Client.InvokeTool("project", map[string]interface{}{"action": "options"})
				ctx.Assertions.AssertNoError(err, "Should invoke project_options without error")
				if err != nil {
					return err
				}

				ctx.Assertions.AssertFalse(result.IsError, "project_options should not return error")
				content := result.GetToolContent()
				ctx.Log("project_options result: %s", content)

				// Check that token scopes are documented
				ctx.Assertions.AssertContains(content, "token_scopes", "Should include token_scopes section")
				ctx.Assertions.AssertContains(content, "admin", "Should document admin scope")
				ctx.Assertions.AssertContains(content, "admin:ro", "Should document admin:ro scope")
				ctx.Assertions.AssertContains(content, "project:<uuid>", "Should document project scope format")
				ctx.Assertions.AssertContains(content, "project:<uuid>:ro", "Should document project:ro scope format")

				return nil
			},
		},
	}
}
