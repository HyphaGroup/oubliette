package opencode

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
	if r.servers == nil {
		t.Error("servers map not initialized")
	}
}

func TestRuntimeName(t *testing.T) {
	r := NewRuntime(nil)
	if r.Name() != "opencode" {
		t.Errorf("Name() = %q, want 'opencode'", r.Name())
	}
}

func TestRuntimeInitialize(t *testing.T) {
	r := NewRuntime(nil)

	err := r.Initialize(context.Background(), nil)
	if err != nil {
		t.Errorf("Initialize(nil) returned error: %v", err)
	}

	if !r.initialized {
		t.Error("runtime not marked as initialized")
	}
}

func TestRuntimeIsAvailable(t *testing.T) {
	r := NewRuntime(nil)

	// Not available before init
	if r.IsAvailable() {
		t.Error("IsAvailable() should return false before Initialize()")
	}

	// Available after init
	_ = r.Initialize(context.Background(), nil)
	if !r.IsAvailable() {
		t.Error("IsAvailable() should return true after Initialize()")
	}
}

func TestRuntimeExecuteRequiresInitialization(t *testing.T) {
	r := NewRuntime(nil)

	_, err := r.Execute(context.Background(), &agent.ExecuteRequest{})
	if err == nil {
		t.Error("Execute() without Initialize() should return error")
	}
	if err.Error() != "runtime not initialized" {
		t.Errorf("error = %q, want 'runtime not initialized'", err.Error())
	}
}

func TestRuntimeExecuteStreamingRequiresInitialization(t *testing.T) {
	r := NewRuntime(nil)

	_, err := r.ExecuteStreaming(context.Background(), &agent.ExecuteRequest{})
	if err == nil {
		t.Error("ExecuteStreaming() without Initialize() should return error")
	}
	if err.Error() != "runtime not initialized" {
		t.Errorf("error = %q, want 'runtime not initialized'", err.Error())
	}
}

func TestRuntimePing(t *testing.T) {
	r := NewRuntime(nil)
	_ = r.Initialize(context.Background(), nil)

	err := r.Ping(context.Background())
	if err != nil {
		t.Errorf("Ping() returned error: %v", err)
	}
}

func TestRuntimeClose(t *testing.T) {
	r := NewRuntime(nil)
	_ = r.Initialize(context.Background(), nil)

	err := r.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	if r.initialized {
		t.Error("runtime still marked as initialized after Close()")
	}
}

// Verify Runtime implements agent.Runtime interface
var _ agent.Runtime = (*Runtime)(nil)
