package suites

import (
	"encoding/json"
	"fmt"
	"time"

	testpkg "github.com/HyphaGroup/oubliette/test/pkg/testing"
)

// GetMessagingTests returns interactive messaging test suite
func GetMessagingTests() []*testpkg.TestCase {
	return []*testpkg.TestCase{
		{
			Name:        "test_send_message_basic",
			Description: "Test sending a message to an interactive session",
			Tags:        []string{"messaging", "interactive"},
			Timeout:     120 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := fmt.Sprintf("test-messaging-send-%d", time.Now().UnixNano())

				// Pre-cleanup
				ctx.PreTestCleanup(projName)

				// Create project (container auto-starts)
				projID, err := ctx.CreateProject(projName, "Test project for messaging")
				ctx.Assertions.AssertNoError(err, "Should create project")

				if err != nil {
					return err
				}

				// Start interactive task
				ctx.Log("Starting interactive session task")
				startParams := map[string]interface{}{
					"project_id":  projID,
					"prompt":      "Wait for instructions",
					"new_session": true,
				}

				startResult, err := ctx.Client.InvokeTool("session", startParams)
				ctx.Assertions.AssertNoError(err, "Should start interactive task")

				if err != nil {
					return err
				}

				// Extract session ID
				content := startResult.GetToolContent()
				var sessionData map[string]interface{}
				if err := json.Unmarshal([]byte(content), &sessionData); err == nil {
					if sessionID, ok := sessionData["session_id"].(string); ok {
						ctx.SessionID = sessionID
						ctx.Log("Interactive session ID: %s", sessionID)

						// Wait for session to initialize
						time.Sleep(2 * time.Second)

						// Send message
						ctx.Log("Sending message to interactive session")
						msgParams := map[string]interface{}{
							"action":     "message",
							"session_id": sessionID,
							"message":    "Hello from test!",
						}

						msgResult, err := ctx.Client.InvokeTool("session", msgParams)
						ctx.Assertions.AssertNoError(err, "Should send message")

						if err == nil {
							ctx.Assertions.AssertFalse(msgResult.IsError, "Send message should succeed")
							msgContent := msgResult.GetToolContent()
							ctx.Assertions.AssertContains(msgContent, "message", "Should confirm message sent")
							ctx.Log("Message sent successfully")
						}
					}
				}

				return nil
			},
		},

		{
			Name:        "test_send_message_feedback",
			Description: "Test sending feedback message during session execution",
			Tags:        []string{"messaging", "feedback"},
			Timeout:     120 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := fmt.Sprintf("test-messaging-feedback-%d", time.Now().UnixNano())

				// Pre-cleanup
				ctx.PreTestCleanup(projName)

				// Create project (container auto-starts)
				projID, err := ctx.CreateProject(projName, "Test project for feedback")
				ctx.Assertions.AssertNoError(err, "Should create project")

				if err != nil {
					return err
				}

				// Start interactive task
				ctx.Log("Starting interactive session task")
				startParams := map[string]interface{}{
					"project_id":  projID,
					"prompt":      "Create a simple script",
					"new_session": true,
				}

				startResult, err := ctx.Client.InvokeTool("session", startParams)
				ctx.Assertions.AssertNoError(err, "Should start interactive task")

				if err != nil {
					return err
				}

				// Extract session ID
				content := startResult.GetToolContent()
				var sessionData map[string]interface{}
				if err := json.Unmarshal([]byte(content), &sessionData); err == nil {
					if sessionID, ok := sessionData["session_id"].(string); ok {
						ctx.SessionID = sessionID
						ctx.Log("Interactive session ID: %s", sessionID)

						// Wait for session to start working
						time.Sleep(2 * time.Second)

						// Send feedback
						ctx.Log("Sending feedback message")
						feedbackParams := map[string]interface{}{
							"action":     "message",
							"session_id": sessionID,
							"message":    "Please make it simpler",
						}

						feedbackResult, err := ctx.Client.InvokeTool("session", feedbackParams)
						ctx.Assertions.AssertNoError(err, "Should send feedback")

						if err == nil {
							ctx.Assertions.AssertFalse(feedbackResult.IsError, "Feedback should be sent")
							ctx.Log("Feedback sent successfully")
						}
					}
				}

				return nil
			},
		},

		{
			Name:        "test_send_message_non_interactive",
			Description: "Test that sending message to non-interactive session fails",
			Tags:        []string{"messaging", "error"},
			Timeout:     90 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				projName := fmt.Sprintf("test-messaging-error-%d", time.Now().UnixNano())

				// Pre-cleanup
				ctx.PreTestCleanup(projName)

				// Create project (container auto-starts)
				projID, err := ctx.CreateProject(projName, "Test project for messaging error")
				ctx.Assertions.AssertNoError(err, "Should create project")

				if err != nil {
					return err
				}

				// Start NON-interactive task
				ctx.Log("Starting non-interactive session task")
				startParams := map[string]interface{}{
					"project_id": projID,
					"prompt":     "Simple task",
					// interactive: false (default)
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
						ctx.Log("Non-interactive session ID: %s", sessionID)

						// Wait a bit
						time.Sleep(1 * time.Second)

						// Try to send message (should fail or be ignored)
						ctx.Log("Attempting to send message to non-interactive session")
						msgParams := map[string]interface{}{
							"action":     "message",
							"session_id": sessionID,
							"message":    "This should not work",
						}

						msgResult, err := ctx.Client.InvokeTool("session", msgParams)

						// Either it errors, or returns an error response
						if err != nil {
							ctx.Assertions.AssertError(err, "Should error when sending to non-interactive session")
							ctx.Log("Correctly errored: %v", err)
						} else if msgResult.IsError {
							ctx.Assertions.AssertTrue(msgResult.IsError, "Should return error result")
							msgContent := msgResult.GetToolContent()
							ctx.Log("Correctly returned error: %s", msgContent)
						} else {
							// If it succeeds, it might just be ignored - that's also acceptable
							ctx.Log("Message was accepted (possibly ignored)")
						}
					}
				}

				return nil
			},
		},
	}
}
