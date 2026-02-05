package mcp

import (
	"fmt"
	"strings"

	"github.com/HyphaGroup/oubliette/internal/logger"
)

// sensitivePatterns contains substrings that indicate sensitive error details
var sensitivePatterns = []string{
	"FACTORY_API_KEY",
	"API_KEY",
	"api_key",
	"token",
	"password",
	"secret",
	"credential",
	"auth",
}

// internalErrorPatterns contains substrings that indicate internal errors
var internalErrorPatterns = []string{
	"failed to exec",
	"failed to start",
	"connection refused",
	"no such file",
	"permission denied",
	"timeout",
	"context canceled",
	"EOF",
}

// SanitizeError returns a client-safe error message.
// Internal details are logged but not exposed to clients.
func SanitizeError(err error, operation string) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// Check for sensitive information
	for _, pattern := range sensitivePatterns {
		if strings.Contains(strings.ToLower(errStr), strings.ToLower(pattern)) {
			logger.Error("%s failed (sensitive): %v", operation, err)
			return fmt.Errorf("%s failed: internal configuration error", operation)
		}
	}

	// Check for internal error patterns
	for _, pattern := range internalErrorPatterns {
		if strings.Contains(strings.ToLower(errStr), strings.ToLower(pattern)) {
			logger.Error("%s failed (internal): %v", operation, err)
			return fmt.Errorf("%s failed: internal error", operation)
		}
	}

	// For other errors, still log the full error but return a generic message
	// unless it looks like a user-facing error (validation, not found, etc.)
	if isUserFacingError(errStr) {
		return err
	}

	logger.Error("%s failed: %v", operation, err)
	return fmt.Errorf("%s failed: %s", operation, genericErrorMessage(errStr))
}

// isUserFacingError returns true if the error message is safe to show to users
func isUserFacingError(errStr string) bool {
	userFacingPatterns := []string{
		"not found",
		"already exists",
		"invalid",
		"required",
		"must be",
		"cannot be",
		"is not",
		"exceeded",
		"limit",
	}

	lower := strings.ToLower(errStr)
	for _, pattern := range userFacingPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// genericErrorMessage extracts a safe portion of the error or returns generic text
func genericErrorMessage(errStr string) string {
	// If it's short and doesn't contain sensitive info, it's probably safe
	if len(errStr) < 50 {
		return errStr
	}
	return "an unexpected error occurred"
}
