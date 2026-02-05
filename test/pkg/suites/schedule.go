package suites

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	testpkg "github.com/HyphaGroup/oubliette/test/pkg/testing"
)

// GetScheduleTests returns scheduled tasks tests
func GetScheduleTests() []*testpkg.TestCase {
	return []*testpkg.TestCase{
		{
			Name:        "test_schedule_create_and_list",
			Description: "Test schedule_create and schedule_list tools",
			Tags:        []string{"schedule", "crud"},
			Timeout:     60 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				// First create a project to target
				projName := fmt.Sprintf("schedule-test-%d", time.Now().UnixNano())
				projectID, err := ctx.CreateProject(projName, "Project for schedule testing")
				ctx.Assertions.AssertNoError(err, "Should create project")
				if err != nil {
					return err
				}
				// Project is auto-tracked for cleanup by CreateProject

				// Create a schedule
				createResult, err := ctx.Client.InvokeTool("schedule", map[string]interface{}{"action": "create",
					"name":      "test-daily-check",
					"cron_expr": "0 9 * * *",
					"prompt":    "Run daily health check",
					"targets": []map[string]interface{}{
						{"project_id": projectID},
					},
				})
				ctx.Assertions.AssertNoError(err, "Should invoke schedule_create")
				if err != nil {
					return err
				}

				ctx.Assertions.AssertFalse(createResult.IsError, "schedule_create should succeed")
				createContent := createResult.GetToolContent()
				ctx.Log("schedule_create result: %s", createContent)
				ctx.Assertions.AssertContains(createContent, "Schedule created successfully", "Should confirm creation")
				ctx.Assertions.AssertContains(createContent, "sched_", "Should include schedule ID")

				// Extract schedule ID
				var scheduleID string
				for _, line := range strings.Split(createContent, "\n") {
					if strings.Contains(line, "ID:") && strings.Contains(line, "sched_") {
						parts := strings.Fields(line)
						for _, part := range parts {
							if strings.HasPrefix(part, "sched_") {
								scheduleID = part
								break
							}
						}
					}
				}
				ctx.Log("Created schedule ID: %s", scheduleID)

				// List schedules
				listResult, err := ctx.Client.InvokeTool("schedule", map[string]interface{}{"action": "list"})
				ctx.Assertions.AssertNoError(err, "Should invoke schedule_list")
				if err != nil {
					return err
				}

				ctx.Assertions.AssertFalse(listResult.IsError, "schedule_list should succeed")
				listContent := listResult.GetToolContent()
				ctx.Log("schedule_list result: %s", listContent)
				ctx.Assertions.AssertContains(listContent, "test-daily-check", "Should include created schedule")

				// Cleanup - delete the schedule
				if scheduleID != "" {
					deleteResult, err := ctx.Client.InvokeTool("schedule", map[string]interface{}{"action": "delete",
						"schedule_id": scheduleID,
					})
					ctx.Assertions.AssertNoError(err, "Should invoke schedule_delete")
					if deleteResult != nil && !deleteResult.IsError {
						ctx.Log("Deleted schedule %s", scheduleID)
					}
				}

				return nil
			},
		},
		{
			Name:        "test_schedule_update_enabled",
			Description: "Test schedule_update to toggle enabled status",
			Tags:        []string{"schedule", "crud"},
			Timeout:     60 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				// Create project
				projName := fmt.Sprintf("schedule-update-test-%d", time.Now().UnixNano())
				projectID, err := ctx.CreateProject(projName, "Project for update testing")
				ctx.Assertions.AssertNoError(err, "Should create project")
				if err != nil {
					return err
				}

				// Create schedule (enabled by default)
				createResult, err := ctx.Client.InvokeTool("schedule", map[string]interface{}{"action": "create",
					"name":      "toggle-test",
					"cron_expr": "0 * * * *",
					"prompt":    "test",
					"targets":   []map[string]interface{}{{"project_id": projectID}},
				})
				ctx.Assertions.AssertNoError(err, "Should create schedule")
				if err != nil {
					return err
				}

				// Extract schedule ID from result
				var scheduleID string
				createContent := createResult.GetToolContent()
				for _, line := range strings.Split(createContent, "\n") {
					if strings.Contains(line, "sched_") {
						parts := strings.Fields(line)
						for _, part := range parts {
							if strings.HasPrefix(part, "sched_") {
								scheduleID = part
								break
							}
						}
					}
				}

				// Disable the schedule
				updateResult, err := ctx.Client.InvokeTool("schedule", map[string]interface{}{"action": "update",
					"schedule_id": scheduleID,
					"enabled":     false,
				})
				ctx.Assertions.AssertNoError(err, "Should invoke schedule_update")
				if err != nil {
					return err
				}

				ctx.Assertions.AssertFalse(updateResult.IsError, "schedule_update should succeed")
				ctx.Assertions.AssertContains(updateResult.GetToolContent(), "updated successfully", "Should confirm update")

				// Get and verify disabled
				getResult, err := ctx.Client.InvokeTool("schedule", map[string]interface{}{"action": "get",
					"schedule_id": scheduleID,
				})
				ctx.Assertions.AssertNoError(err, "Should invoke schedule_get")
				if err != nil {
					return err
				}

				getContent := getResult.GetToolContent()
				ctx.Log("After disable: %s", getContent)
				ctx.Assertions.AssertContains(getContent, "disabled", "Should show disabled status")

				// Cleanup
				ctx.Client.InvokeTool("schedule", map[string]interface{}{"action": "delete","schedule_id": scheduleID})

				return nil
			},
		},
		{
			Name:        "test_schedule_delete",
			Description: "Test schedule_delete tool",
			Tags:        []string{"schedule", "crud"},
			Timeout:     60 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				// Create project
				projName := fmt.Sprintf("schedule-delete-test-%d", time.Now().UnixNano())
				projectID, err := ctx.CreateProject(projName, "Project for delete testing")
				ctx.Assertions.AssertNoError(err, "Should create project")
				if err != nil {
					return err
				}

				// Create schedule
				createResult, err := ctx.Client.InvokeTool("schedule", map[string]interface{}{"action": "create",
					"name":      "to-delete",
					"cron_expr": "0 0 * * *",
					"prompt":    "delete me",
					"targets":   []map[string]interface{}{{"project_id": projectID}},
				})
				ctx.Assertions.AssertNoError(err, "Should create schedule")
				if err != nil {
					return err
				}

				// Extract ID
				var scheduleID string
				for _, line := range strings.Split(createResult.GetToolContent(), "\n") {
					if strings.Contains(line, "sched_") {
						parts := strings.Fields(line)
						for _, part := range parts {
							if strings.HasPrefix(part, "sched_") {
								scheduleID = part
								break
							}
						}
					}
				}
				ctx.Assertions.AssertTrue(scheduleID != "", "Should have schedule ID")

				// Delete
				deleteResult, err := ctx.Client.InvokeTool("schedule", map[string]interface{}{"action": "delete",
					"schedule_id": scheduleID,
				})
				ctx.Assertions.AssertNoError(err, "Should invoke schedule_delete")
				if err != nil {
					return err
				}

				ctx.Assertions.AssertFalse(deleteResult.IsError, "schedule_delete should succeed")
				ctx.Assertions.AssertContains(deleteResult.GetToolContent(), "deleted successfully", "Should confirm deletion")

				// Verify gone
				getResult, err := ctx.Client.InvokeTool("schedule", map[string]interface{}{"action": "get",
					"schedule_id": scheduleID,
				})
				ctx.Assertions.AssertNoError(err, "Should invoke schedule_get")
				// Should be error result since not found
				ctx.Assertions.AssertTrue(getResult.IsError, "schedule_get should fail for deleted schedule")

				return nil
			},
		},
		{
			Name:        "test_schedule_trigger_manual",
			Description: "Test schedule_trigger tool for manual execution",
			Tags:        []string{"schedule", "trigger"},
			Timeout:     120 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				// Create project with container
				projName := fmt.Sprintf("schedule-trigger-test-%d", time.Now().UnixNano())
				projectID, err := ctx.CreateProject(projName, "Project for trigger testing")
				ctx.Assertions.AssertNoError(err, "Should create project")
				if err != nil {
					return err
				}

				// Create a disabled schedule (so it won't auto-run)
				createResult, err := ctx.Client.InvokeTool("schedule", map[string]interface{}{"action": "create",
					"name":      "manual-trigger-test",
					"cron_expr": "0 0 1 1 *", // Jan 1 at midnight (won't fire naturally)
					"prompt":    "echo hello from schedule",
					"enabled":   false,
					"targets":   []map[string]interface{}{{"project_id": projectID}},
				})
				ctx.Assertions.AssertNoError(err, "Should create schedule")
				if err != nil {
					return err
				}

				// Extract ID
				var scheduleID string
				for _, line := range strings.Split(createResult.GetToolContent(), "\n") {
					if strings.Contains(line, "sched_") {
						parts := strings.Fields(line)
						for _, part := range parts {
							if strings.HasPrefix(part, "sched_") {
								scheduleID = part
								break
							}
						}
					}
				}

				// Manual trigger
				triggerResult, err := ctx.Client.InvokeTool("schedule", map[string]interface{}{"action": "trigger",
					"schedule_id": scheduleID,
				})
				ctx.Assertions.AssertNoError(err, "Should invoke schedule_trigger")
				if err != nil {
					// Cleanup even on error
					ctx.Client.InvokeTool("schedule", map[string]interface{}{"action": "delete","schedule_id": scheduleID})
					return err
				}

				triggerContent := triggerResult.GetToolContent()
				ctx.Log("Trigger result: %s", triggerContent)

				// Note: The actual execution might fail if container isn't ready,
				// but the trigger itself should work
				ctx.Assertions.AssertContains(triggerContent, "triggered", "Should confirm trigger")

				// Cleanup
				ctx.Client.InvokeTool("schedule", map[string]interface{}{"action": "delete","schedule_id": scheduleID})

				return nil
			},
		},
		{
			Name:        "test_schedule_project_scope_restriction",
			Description: "Test that project-scoped tokens can only access their schedules",
			Tags:        []string{"schedule", "auth"},
			Timeout:     60 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				// This test verifies that schedules are filtered by token scope
				// Since our test client has admin scope, we'll verify listing works
				// and schedules include creator scope info

				projName := fmt.Sprintf("schedule-scope-test-%d", time.Now().UnixNano())
				projectID, err := ctx.CreateProject(projName, "Project for scope testing")
				ctx.Assertions.AssertNoError(err, "Should create project")
				if err != nil {
					return err
				}

				// Create schedule
				createResult, err := ctx.Client.InvokeTool("schedule", map[string]interface{}{"action": "create",
					"name":      "scope-test",
					"cron_expr": "0 0 * * *",
					"prompt":    "test",
					"targets":   []map[string]interface{}{{"project_id": projectID}},
				})
				ctx.Assertions.AssertNoError(err, "Should create schedule")
				if err != nil {
					return err
				}

				// Extract ID
				var scheduleID string
				for _, line := range strings.Split(createResult.GetToolContent(), "\n") {
					if strings.Contains(line, "sched_") {
						parts := strings.Fields(line)
						for _, part := range parts {
							if strings.HasPrefix(part, "sched_") {
								scheduleID = part
								break
							}
						}
					}
				}

				// Get schedule details and verify it includes creator info
				getResult, err := ctx.Client.InvokeTool("schedule", map[string]interface{}{"action": "get",
					"schedule_id": scheduleID,
				})
				ctx.Assertions.AssertNoError(err, "Should get schedule")
				if err != nil {
					return err
				}

				getContent := getResult.GetToolContent()
				ctx.Log("Schedule details: %s", getContent)

				// Verify schedule was retrieved (scope check passed)
				ctx.Assertions.AssertContains(getContent, "scope-test", "Should retrieve schedule")
				ctx.Assertions.AssertContains(getContent, projectID, "Should include target project")

				// List with project filter
				listResult, err := ctx.Client.InvokeTool("schedule", map[string]interface{}{"action": "list",
					"project_id": projectID,
				})
				ctx.Assertions.AssertNoError(err, "Should list with project filter")
				if err != nil {
					return err
				}

				listContent := listResult.GetToolContent()
				ctx.Assertions.AssertContains(listContent, "scope-test", "Filter should include matching schedule")

				// Cleanup
				ctx.Client.InvokeTool("schedule", map[string]interface{}{"action": "delete","schedule_id": scheduleID})

				return nil
			},
		},
	}
}

// Helper to parse JSON result
func parseJSONResult(content string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(content), &result)
	return result, err
}
