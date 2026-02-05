package container

import (
	"context"
	"io"
	"time"
)

// Runtime defines the container runtime abstraction
type Runtime interface {
	// Lifecycle
	Create(ctx context.Context, config CreateConfig) (string, error)
	Start(ctx context.Context, containerID string) error
	Stop(ctx context.Context, containerID string) error
	Remove(ctx context.Context, containerID string, force bool) error

	// Execution
	Exec(ctx context.Context, containerID string, config ExecConfig) (*ExecResult, error)
	ExecInteractive(ctx context.Context, containerID string, config ExecConfig) (*InteractiveExec, error)

	// Inspection
	Inspect(ctx context.Context, containerID string) (*ContainerInfo, error)
	Logs(ctx context.Context, containerID string, opts LogsOptions) (string, error)
	Status(ctx context.Context, containerID string) (ContainerStatus, error)

	// Images
	Build(ctx context.Context, config BuildConfig) error
	ImageExists(ctx context.Context, imageName string) (bool, error)
	Pull(ctx context.Context, imageName string) error

	// Health
	Ping(ctx context.Context) error
	Close() error

	// Metadata
	Name() string
	IsAvailable() bool
}

// InteractiveExec represents an interactive command execution with I/O pipes
type InteractiveExec struct {
	Stdin  io.WriteCloser
	Stdout io.ReadCloser
	Stderr io.ReadCloser
	done   chan struct{}
	wait   func() (int, error)
}

// NewInteractiveExec creates a new InteractiveExec
func NewInteractiveExec(stdin io.WriteCloser, stdout, stderr io.ReadCloser, wait func() (int, error)) *InteractiveExec {
	return &InteractiveExec{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		done:   make(chan struct{}),
		wait:   wait,
	}
}

// Done returns a channel that is closed when the process exits
func (e *InteractiveExec) Done() <-chan struct{} {
	return e.done
}

// Wait waits for the process to exit and returns the exit code
func (e *InteractiveExec) Wait() (int, error) {
	code, err := e.wait()
	select {
	case <-e.done:
	default:
		close(e.done)
	}
	return code, err
}

// Close closes all I/O streams
func (e *InteractiveExec) Close() error {
	if e.Stdin != nil {
		_ = e.Stdin.Close()
	}
	if e.Stdout != nil {
		_ = e.Stdout.Close()
	}
	if e.Stderr != nil {
		_ = e.Stderr.Close()
	}
	return nil
}

// CreateConfig for container creation
type CreateConfig struct {
	Name        string
	Image       string
	Cmd         []string
	Entrypoint  []string
	Env         []string
	WorkingDir  string
	Mounts      []Mount
	Labels      map[string]string
	Init        bool
	AutoRemove  bool
	NetworkMode string
	Memory      string // Memory limit (e.g., "4G", "2048M")
	CPUs        int    // Number of CPUs

	// PublishedSockets exposes container sockets to the host
	// For Apple Container: uses --publish-socket (container->host forwarding)
	// For Docker: uses bind mount of socket directory
	PublishedSockets []PublishedSocket
}

// PublishedSocket represents a socket to expose from container to host
type PublishedSocket struct {
	HostPath      string // Path on host where socket will appear
	ContainerPath string // Path inside container where socket is created
}

// MountType represents the type of mount
type MountType string

const (
	MountTypeBind   MountType = "bind"
	MountTypeVolume MountType = "volume"
	MountTypeTmpfs  MountType = "tmpfs"
)

// Mount represents a bind mount or volume
type Mount struct {
	Type     MountType
	Source   string
	Target   string
	ReadOnly bool
}

// ExecConfig for command execution
type ExecConfig struct {
	Cmd          []string
	Env          []string
	WorkingDir   string
	AttachStdout bool
	AttachStderr bool
	AttachStdin  bool
	TTY          bool
	User         string
}

// ExecResult contains execution output
type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// LogsOptions for log retrieval
type LogsOptions struct {
	Tail       string
	Timestamps bool
}

// ContainerInfo contains inspection data
type ContainerInfo struct {
	ID        string
	Name      string
	Image     string
	Status    ContainerStatus
	IPAddress string
	Mounts    []Mount
	Env       []string
	CreatedAt time.Time
	StartedAt time.Time
}

// ContainerStatus enum
type ContainerStatus string

const (
	StatusCreated ContainerStatus = "created"
	StatusRunning ContainerStatus = "running"
	StatusPaused  ContainerStatus = "paused"
	StatusStopped ContainerStatus = "stopped"
	StatusExited  ContainerStatus = "exited"
	StatusDead    ContainerStatus = "dead"
	StatusUnknown ContainerStatus = "unknown"
)

// BuildConfig for image building
type BuildConfig struct {
	ImageName      string
	DockerfilePath string
	ContextPath    string
	BuildArgs      map[string]string
}
