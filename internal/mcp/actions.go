package mcp

import (
	"fmt"
	"strings"
)

// actionError returns a formatted error for invalid actions
func actionError(tool, action string, valid []string) error {
	return fmt.Errorf("unknown action '%s' for %s tool; valid actions: %s", action, tool, strings.Join(valid, ", "))
}

// missingActionError returns an error for missing action parameter
func missingActionError(tool string, valid []string) error {
	return fmt.Errorf("action parameter is required for %s tool; valid actions: %s", tool, strings.Join(valid, ", "))
}
