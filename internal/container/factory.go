package container

import (
	"os"
)

// GetRuntimePreference returns the configured runtime preference from environment
func GetRuntimePreference() string {
	pref := os.Getenv("CONTAINER_RUNTIME")
	if pref == "" {
		return "auto"
	}
	return pref
}
