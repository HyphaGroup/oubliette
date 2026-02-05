// Package agent provides the agent runtime abstraction layer.
//
// factory.go - Runtime factory and auto-detection
//
// This file contains:
// - RuntimeType constants (droid, opencode, auto)
// - FactoryConfig for runtime creation parameters
// - RuntimeFactory for creating runtime instances
// - Auto-detection logic (Factory API key -> Droid, otherwise OpenCode)
//
// Note: The factory methods are placeholders; main.go creates runtimes
// directly to avoid circular imports between agent and runtime packages.

package agent

import (
	"context"
	"fmt"

	"github.com/HyphaGroup/oubliette/internal/container"
)

// RuntimeType identifies the agent runtime backend
type RuntimeType string

const (
	RuntimeTypeDroid    RuntimeType = "droid"
	RuntimeTypeOpenCode RuntimeType = "opencode"
	RuntimeTypeAuto     RuntimeType = "auto"
)

// FactoryConfig holds configuration for runtime creation
type FactoryConfig struct {
	// RuntimeType specifies which runtime to use
	RuntimeType RuntimeType

	// ContainerRuntime for container operations
	ContainerRuntime container.Runtime

	// Droid-specific config
	DroidConfig *RuntimeConfig

	// OpenCode-specific config
	OpenCodeConfig *RuntimeConfig

	// FactoryAPIKey for auto-detection
	FactoryAPIKey string
}

// RuntimeFactory creates agent runtimes based on configuration
type RuntimeFactory struct {
	containerRuntime container.Runtime
}

// NewFactory creates a new runtime factory
func NewFactory(containerRuntime container.Runtime) *RuntimeFactory {
	return &RuntimeFactory{
		containerRuntime: containerRuntime,
	}
}

// CreateRuntime creates a new agent runtime based on configuration
func (f *RuntimeFactory) CreateRuntime(ctx context.Context, config *FactoryConfig) (Runtime, error) {
	runtimeType := config.RuntimeType

	// Auto-detection: use Droid if Factory API key is available
	if runtimeType == RuntimeTypeAuto || runtimeType == "" {
		if config.FactoryAPIKey != "" {
			runtimeType = RuntimeTypeDroid
		} else {
			runtimeType = RuntimeTypeOpenCode
		}
	}

	switch runtimeType {
	case RuntimeTypeDroid:
		return f.createDroidRuntime(ctx, config)
	case RuntimeTypeOpenCode:
		return f.createOpenCodeRuntime(ctx, config)
	default:
		return nil, fmt.Errorf("unknown runtime type: %s", runtimeType)
	}
}

// createDroidRuntime creates and initializes a Droid runtime
// Note: We import dynamically to avoid circular imports
func (f *RuntimeFactory) createDroidRuntime(ctx context.Context, config *FactoryConfig) (Runtime, error) {
	// This is a placeholder - the actual implementation needs to import droid package
	// For now, main.go creates the Droid runtime directly
	return nil, fmt.Errorf("use agentdroid.NewRuntime() directly - factory method pending")
}

// createOpenCodeRuntime creates and initializes an OpenCode runtime
// Note: We import dynamically to avoid circular imports
func (f *RuntimeFactory) createOpenCodeRuntime(ctx context.Context, config *FactoryConfig) (Runtime, error) {
	// This is a placeholder - the actual implementation needs to import opencode package
	// For now, main.go would create the OpenCode runtime directly
	return nil, fmt.Errorf("use agentopencode.NewRuntime() directly - factory method pending")
}

// DetectRuntimeType determines the best runtime based on available configuration
func DetectRuntimeType(factoryAPIKey string) RuntimeType {
	if factoryAPIKey != "" {
		return RuntimeTypeDroid
	}
	return RuntimeTypeOpenCode
}
