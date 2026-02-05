package validation

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// UUIDRegex matches standard UUID format
	uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

	// SafePathRegex matches safe path components (alphanumeric, dash, underscore, dot)
	safePathRegex = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)
)

// ValidateUUID checks if the string is a valid UUID
func ValidateUUID(id string) error {
	if id == "" {
		return fmt.Errorf("ID cannot be empty")
	}
	if !uuidRegex.MatchString(id) {
		return fmt.Errorf("invalid UUID format: %s", id)
	}
	return nil
}

// ValidateProjectID validates a project ID
func ValidateProjectID(id string) error {
	return ValidateUUID(id)
}

// gogolSessionRegex matches gogol_YYYYMMDD_HHMMSS_RANDOMHEX format
var gogolSessionRegex = regexp.MustCompile(`^gogol_\d{8}_\d{6}_[0-9a-fA-F]+$`)

// ValidateSessionID validates a session ID (can be UUID, child_*, or gogol_* format)
func ValidateSessionID(id string) error {
	if id == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	// Child session IDs have format child_<parent>_<counter>
	if strings.HasPrefix(id, "child_") {
		parts := strings.Split(id, "_")
		if len(parts) < 3 {
			return fmt.Errorf("invalid child session ID format: %s", id)
		}
		return nil
	}

	// Gogol session IDs have format gogol_YYYYMMDD_HHMMSS_RANDOMHEX
	if strings.HasPrefix(id, "gogol_") {
		if !gogolSessionRegex.MatchString(id) {
			return fmt.Errorf("invalid gogol session ID format: %s", id)
		}
		return nil
	}

	// Regular session IDs are UUIDs
	return ValidateUUID(id)
}

// ValidateWorkspaceID validates a workspace ID
func ValidateWorkspaceID(id string) error {
	return ValidateUUID(id)
}

// SanitizePath removes path traversal attempts and validates path components
func SanitizePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// Reject obvious traversal attempts
	if strings.Contains(path, "..") {
		return "", fmt.Errorf("path traversal detected: %s", path)
	}

	// Reject absolute paths when relative expected
	if strings.HasPrefix(path, "/") {
		return "", fmt.Errorf("absolute paths not allowed: %s", path)
	}

	// Split and validate each component
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if part == "" {
			continue // Allow trailing/leading slashes
		}
		if !safePathRegex.MatchString(part) {
			return "", fmt.Errorf("unsafe path component: %s", part)
		}
	}

	return path, nil
}

// ValidateContainerID validates a container ID (hex string)
func ValidateContainerID(id string) error {
	if id == "" {
		return fmt.Errorf("container ID cannot be empty")
	}

	// Container IDs are hex strings, typically 64 chars but can be shorter for short IDs
	if len(id) < 12 || len(id) > 64 {
		return fmt.Errorf("invalid container ID length: %s", id)
	}

	for _, c := range id {
		isDigit := c >= '0' && c <= '9'
		isLowerHex := c >= 'a' && c <= 'f'
		isUpperHex := c >= 'A' && c <= 'F'
		if !isDigit && !isLowerHex && !isUpperHex {
			return fmt.Errorf("invalid container ID format: %s", id)
		}
	}

	return nil
}
