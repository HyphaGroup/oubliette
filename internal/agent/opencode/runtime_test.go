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

func TestRuntimePing(t *testing.T) {
	r := NewRuntime(nil)
	if err := r.Ping(context.Background()); err != nil {
		t.Errorf("Ping() returned error: %v", err)
	}
}

func TestRuntimeClose(t *testing.T) {
	r := NewRuntime(nil)
	if err := r.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

var _ agent.Runtime = (*Runtime)(nil)
