package testutil

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/HyphaGroup/oubliette/internal/container"
)

// MockRuntime is a test double for container.Runtime.
// It records calls and allows configuring responses for testing.
type MockRuntime struct {
	mu sync.Mutex

	// Configurable responses
	CreateResponse  string
	CreateError     error
	StartError      error
	StopError       error
	RemoveError     error
	ExecResponse    *container.ExecResult
	ExecError       error
	InspectResponse *container.ContainerInfo
	InspectError    error
	LogsResponse    string
	LogsError       error
	StatusResponse  container.ContainerStatus
	StatusError     error
	BuildError      error
	PingError       error
	ImageExistsFunc func(imageName string) (bool, error)

	// Call tracking
	CreateCalls  []container.CreateConfig
	StartCalls   []string
	StopCalls    []string
	RemoveCalls  []RemoveCall
	ExecCalls    []ExecCall
	InspectCalls []string
	LogsCalls    []LogsCall
	StatusCalls  []string
	BuildCalls   []container.BuildConfig

	// Container state (for stateful mocking)
	Containers map[string]*container.ContainerInfo
}

// RemoveCall records a Remove call.
type RemoveCall struct {
	ContainerID string
	Force       bool
}

// ExecCall records an Exec call.
type ExecCall struct {
	ContainerID string
	Config      container.ExecConfig
}

// LogsCall records a Logs call.
type LogsCall struct {
	ContainerID string
	Opts        container.LogsOptions
}

// NewMockRuntime creates a new mock runtime with sensible defaults.
func NewMockRuntime(t *testing.T) *MockRuntime {
	t.Helper()
	return &MockRuntime{
		CreateResponse: "mock-container-id",
		ExecResponse: &container.ExecResult{
			Stdout:   "",
			Stderr:   "",
			ExitCode: 0,
		},
		StatusResponse: container.StatusRunning,
		Containers:     make(map[string]*container.ContainerInfo),
	}
}

// Create implements container.Runtime.
func (m *MockRuntime) Create(ctx context.Context, config container.CreateConfig) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CreateCalls = append(m.CreateCalls, config)
	if m.CreateError != nil {
		return "", m.CreateError
	}

	id := m.CreateResponse
	if id == "" {
		id = "mock-" + config.Name
	}

	m.Containers[id] = &container.ContainerInfo{
		ID:        id,
		Name:      config.Name,
		Image:     config.Image,
		Status:    container.StatusCreated,
		CreatedAt: time.Now(),
	}

	return id, nil
}

// Start implements container.Runtime.
func (m *MockRuntime) Start(ctx context.Context, containerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.StartCalls = append(m.StartCalls, containerID)
	if m.StartError != nil {
		return m.StartError
	}

	if info, ok := m.Containers[containerID]; ok {
		info.Status = container.StatusRunning
		info.StartedAt = time.Now()
	}

	return nil
}

// Stop implements container.Runtime.
func (m *MockRuntime) Stop(ctx context.Context, containerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.StopCalls = append(m.StopCalls, containerID)
	if m.StopError != nil {
		return m.StopError
	}

	if info, ok := m.Containers[containerID]; ok {
		info.Status = container.StatusStopped
	}

	return nil
}

// Remove implements container.Runtime.
func (m *MockRuntime) Remove(ctx context.Context, containerID string, force bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RemoveCalls = append(m.RemoveCalls, RemoveCall{containerID, force})
	if m.RemoveError != nil {
		return m.RemoveError
	}

	delete(m.Containers, containerID)
	return nil
}

// Exec implements container.Runtime.
func (m *MockRuntime) Exec(ctx context.Context, containerID string, config container.ExecConfig) (*container.ExecResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ExecCalls = append(m.ExecCalls, ExecCall{containerID, config})
	if m.ExecError != nil {
		return nil, m.ExecError
	}

	return m.ExecResponse, nil
}

// ExecInteractive implements container.Runtime.
func (m *MockRuntime) ExecInteractive(ctx context.Context, containerID string, config container.ExecConfig) (*container.InteractiveExec, error) {
	// For interactive exec, create a mock that returns immediately
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ExecCalls = append(m.ExecCalls, ExecCall{containerID, config})
	if m.ExecError != nil {
		return nil, m.ExecError
	}

	// Create mock pipes
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()

	// Close the write ends to simulate process exit
	go func() {
		time.Sleep(10 * time.Millisecond)
		_ = stdinR.Close()
		_ = stdoutW.Close()
		_ = stderrW.Close()
	}()

	return container.NewInteractiveExec(stdinW, stdoutR, stderrR, func() (int, error) {
		return 0, nil
	}), nil
}

// Inspect implements container.Runtime.
func (m *MockRuntime) Inspect(ctx context.Context, containerID string) (*container.ContainerInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.InspectCalls = append(m.InspectCalls, containerID)
	if m.InspectError != nil {
		return nil, m.InspectError
	}

	if m.InspectResponse != nil {
		return m.InspectResponse, nil
	}

	if info, ok := m.Containers[containerID]; ok {
		return info, nil
	}

	return nil, errors.New("container not found")
}

// Logs implements container.Runtime.
func (m *MockRuntime) Logs(ctx context.Context, containerID string, opts container.LogsOptions) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.LogsCalls = append(m.LogsCalls, LogsCall{containerID, opts})
	if m.LogsError != nil {
		return "", m.LogsError
	}

	return m.LogsResponse, nil
}

// Status implements container.Runtime.
func (m *MockRuntime) Status(ctx context.Context, containerID string) (container.ContainerStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.StatusCalls = append(m.StatusCalls, containerID)
	if m.StatusError != nil {
		return container.StatusUnknown, m.StatusError
	}

	if info, ok := m.Containers[containerID]; ok {
		return info.Status, nil
	}

	return m.StatusResponse, nil
}

// Build implements container.Runtime.
func (m *MockRuntime) Build(ctx context.Context, config container.BuildConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.BuildCalls = append(m.BuildCalls, config)
	return m.BuildError
}

// ImageExists implements container.Runtime.
func (m *MockRuntime) ImageExists(ctx context.Context, imageName string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ImageExistsFunc != nil {
		return m.ImageExistsFunc(imageName)
	}
	return true, nil
}

// Pull implements container.Runtime.
func (m *MockRuntime) Pull(ctx context.Context, imageName string) error {
	return nil
}

// Ping implements container.Runtime.
func (m *MockRuntime) Ping(ctx context.Context) error {
	return m.PingError
}

// Close implements container.Runtime.
func (m *MockRuntime) Close() error {
	return nil
}

// Name implements container.Runtime.
func (m *MockRuntime) Name() string {
	return "mock"
}

// IsAvailable implements container.Runtime.
func (m *MockRuntime) IsAvailable() bool {
	return true
}

// Reset clears all recorded calls and containers.
func (m *MockRuntime) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CreateCalls = nil
	m.StartCalls = nil
	m.StopCalls = nil
	m.RemoveCalls = nil
	m.ExecCalls = nil
	m.InspectCalls = nil
	m.LogsCalls = nil
	m.StatusCalls = nil
	m.BuildCalls = nil
	m.Containers = make(map[string]*container.ContainerInfo)
}

// SetContainerStatus sets the status for a specific container.
func (m *MockRuntime) SetContainerStatus(containerID string, status container.ContainerStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Containers[containerID] == nil {
		m.Containers[containerID] = &container.ContainerInfo{ID: containerID}
	}
	m.Containers[containerID].Status = status
}

// AssertCreateCalled asserts Create was called with expected image.
func (m *MockRuntime) AssertCreateCalled(t *testing.T, expectedImage string) {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.CreateCalls {
		if m.CreateCalls[i].Image == expectedImage {
			return
		}
	}
	t.Errorf("Create not called with image %q, calls: %v", expectedImage, m.CreateCalls)
}

// AssertStartCalled asserts Start was called with the given container ID.
func (m *MockRuntime) AssertStartCalled(t *testing.T, containerID string) {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, id := range m.StartCalls {
		if id == containerID {
			return
		}
	}
	t.Errorf("Start not called with container %q, calls: %v", containerID, m.StartCalls)
}

// AssertExecCalled asserts Exec was called with expected command prefix.
func (m *MockRuntime) AssertExecCalled(t *testing.T, cmdPrefix string) {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, call := range m.ExecCalls {
		if len(call.Config.Cmd) > 0 && strings.HasPrefix(call.Config.Cmd[0], cmdPrefix) {
			return
		}
	}
	t.Errorf("Exec not called with command prefix %q, calls: %v", cmdPrefix, m.ExecCalls)
}

// Verify MockRuntime implements Runtime interface
var _ container.Runtime = (*MockRuntime)(nil)
