package mcp

import (
	"context"
	"testing"

	"github.com/HyphaGroup/oubliette/internal/agent"
	"github.com/HyphaGroup/oubliette/internal/project"
)

// mockRuntime implements agent.Runtime for testing
type mockRuntime struct {
	name string
}

func (m *mockRuntime) Initialize(ctx context.Context, config *agent.RuntimeConfig) error { return nil }
func (m *mockRuntime) ExecuteStreaming(ctx context.Context, request *agent.ExecuteRequest) (agent.StreamingExecutor, error) {
	return nil, nil
}
func (m *mockRuntime) Execute(ctx context.Context, request *agent.ExecuteRequest) (*agent.ExecuteResponse, error) {
	return nil, nil
}
func (m *mockRuntime) Ping(ctx context.Context) error { return nil }
func (m *mockRuntime) Close() error                   { return nil }
func (m *mockRuntime) Name() string                   { return m.name }
func (m *mockRuntime) IsAvailable() bool              { return true }

func TestGetRuntimeForProject_WithOverride(t *testing.T) {
	defaultRuntime := &mockRuntime{name: "droid"}
	opencodeRuntime := &mockRuntime{name: "opencode"}

	factory := func(runtimeType string) agent.Runtime {
		if runtimeType == "opencode" {
			return opencodeRuntime
		}
		return nil
	}

	s := &Server{
		agentRuntime:   defaultRuntime,
		runtimeFactory: factory,
	}

	proj := &project.Project{
		AgentRuntime: "opencode",
	}

	got := s.GetRuntimeForProject(proj)
	if got != opencodeRuntime {
		t.Errorf("GetRuntimeForProject() = %v, want opencode runtime", got.Name())
	}
}

func TestGetRuntimeForProject_WithNoOverride(t *testing.T) {
	defaultRuntime := &mockRuntime{name: "droid"}

	factory := func(runtimeType string) agent.Runtime {
		return nil
	}

	s := &Server{
		agentRuntime:   defaultRuntime,
		runtimeFactory: factory,
	}

	proj := &project.Project{
		AgentRuntime: "", // No override
	}

	got := s.GetRuntimeForProject(proj)
	if got != defaultRuntime {
		t.Errorf("GetRuntimeForProject() = %v, want droid runtime", got.Name())
	}
}

func TestGetRuntimeForProject_FactoryReturnsNil(t *testing.T) {
	defaultRuntime := &mockRuntime{name: "droid"}

	factory := func(runtimeType string) agent.Runtime {
		return nil // Factory doesn't support this runtime type
	}

	s := &Server{
		agentRuntime:   defaultRuntime,
		runtimeFactory: factory,
	}

	proj := &project.Project{
		AgentRuntime: "unsupported-runtime",
	}

	got := s.GetRuntimeForProject(proj)
	if got != defaultRuntime {
		t.Errorf("GetRuntimeForProject() = %v, want droid runtime (fallback)", got.Name())
	}
}

func TestGetRuntimeForProject_NoFactory(t *testing.T) {
	defaultRuntime := &mockRuntime{name: "droid"}

	s := &Server{
		agentRuntime:   defaultRuntime,
		runtimeFactory: nil, // No factory configured
	}

	proj := &project.Project{
		AgentRuntime: "opencode",
	}

	got := s.GetRuntimeForProject(proj)
	if got != defaultRuntime {
		t.Errorf("GetRuntimeForProject() = %v, want droid runtime (no factory)", got.Name())
	}
}
