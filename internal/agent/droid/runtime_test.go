package droid

import (
	"context"
	"testing"

	"github.com/HyphaGroup/oubliette/internal/agent"
)

func TestNewRuntime(t *testing.T) {
	r := NewRuntime(nil)
	if r == nil {
		t.Fatal("NewRuntime returned nil")
	}
}

func TestRuntimeName(t *testing.T) {
	r := NewRuntime(nil)
	if r.Name() != "droid" {
		t.Errorf("Name() = %q, want 'droid'", r.Name())
	}
}

func TestRuntimeInitialize(t *testing.T) {
	r := NewRuntime(nil)

	// Initialize with nil config
	err := r.Initialize(context.Background(), nil)
	if err != nil {
		t.Errorf("Initialize(nil) returned error: %v", err)
	}

	// Verify defaults are set
	if r.defaultModel != "claude-opus-4-5-20251101" {
		t.Errorf("defaultModel = %q, want 'claude-opus-4-5-20251101'", r.defaultModel)
	}
	if r.defaultAutonomy != "skip-permissions-unsafe" {
		t.Errorf("defaultAutonomy = %q, want 'skip-permissions-unsafe'", r.defaultAutonomy)
	}
}

func TestRuntimeInitializeWithConfig(t *testing.T) {
	r := NewRuntime(nil)

	cfg := &agent.RuntimeConfig{
		DefaultModel:    "claude-sonnet-4-5",
		DefaultAutonomy: "auto-high",
		APIKey:          "test-api-key",
	}

	err := r.Initialize(context.Background(), cfg)
	if err != nil {
		t.Errorf("Initialize(cfg) returned error: %v", err)
	}

	// Verify config values are used
	if r.defaultModel != "claude-sonnet-4-5" {
		t.Errorf("defaultModel = %q, want 'claude-sonnet-4-5'", r.defaultModel)
	}
	if r.defaultAutonomy != "auto-high" {
		t.Errorf("defaultAutonomy = %q, want 'auto-high'", r.defaultAutonomy)
	}
	if r.apiKey != "test-api-key" {
		t.Errorf("apiKey = %q, want 'test-api-key'", r.apiKey)
	}
}

func TestRuntimeExecuteRequiresInitialization(t *testing.T) {
	r := NewRuntime(nil)

	// Execute without initialization should fail
	_, err := r.Execute(context.Background(), &agent.ExecuteRequest{})
	if err == nil {
		t.Error("Execute() without Initialize() should return error")
	}
	if err.Error() != "runtime not initialized" {
		t.Errorf("error = %q, want 'runtime not initialized'", err.Error())
	}
}

func TestRuntimeExecuteRequiresAPIKey(t *testing.T) {
	r := NewRuntime(nil)

	// Initialize without API key
	_ = r.Initialize(context.Background(), &agent.RuntimeConfig{})

	// Execute without API key should fail
	_, err := r.Execute(context.Background(), &agent.ExecuteRequest{})
	if err == nil {
		t.Error("Execute() without API key should return error")
	}
	if err.Error() != "FACTORY_API_KEY not configured" {
		t.Errorf("error = %q, want 'FACTORY_API_KEY not configured'", err.Error())
	}
}

// Verify Runtime implements agent.Runtime interface
var _ agent.Runtime = (*Runtime)(nil)
