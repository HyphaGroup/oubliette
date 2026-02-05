package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/HyphaGroup/oubliette/internal/container"
	"github.com/docker/docker/api/types"
	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// Runtime implements container.Runtime using Docker SDK
type Runtime struct {
	client *client.Client
}

// NewRuntime creates a new Docker runtime
func NewRuntime() (*Runtime, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}
	return &Runtime{client: cli}, nil
}

// Name returns the runtime name
func (r *Runtime) Name() string {
	return "docker"
}

// IsAvailable checks if Docker is available
func (r *Runtime) IsAvailable() bool {
	ctx := context.Background()
	_, err := r.client.Ping(ctx)
	return err == nil
}

// Ping verifies connectivity to Docker daemon
func (r *Runtime) Ping(ctx context.Context) error {
	_, err := r.client.Ping(ctx)
	return err
}

// Close closes the Docker client connection
func (r *Runtime) Close() error {
	return r.client.Close()
}

// GetClient returns the underlying Docker client for advanced operations
func (r *Runtime) GetClient() *client.Client {
	return r.client
}

// Create creates a new container
func (r *Runtime) Create(ctx context.Context, cfg container.CreateConfig) (string, error) {
	containerConfig := &dockercontainer.Config{
		Image:      cfg.Image,
		Cmd:        cfg.Cmd,
		Entrypoint: cfg.Entrypoint,
		Env:        cfg.Env,
		WorkingDir: cfg.WorkingDir,
		Labels:     cfg.Labels,
		Tty:        false,
	}

	var mounts []mount.Mount
	for _, m := range cfg.Mounts {
		mounts = append(mounts, mount.Mount{
			Type:     mount.Type(m.Type),
			Source:   m.Source,
			Target:   m.Target,
			ReadOnly: m.ReadOnly,
		})
	}

	// For published sockets, Docker uses bind mounts of the socket's parent directory
	// The relay inside the container creates the socket, and it becomes visible on host
	for _, ps := range cfg.PublishedSockets {
		// Extract directory from host path (e.g., /tmp/oubliette-sockets/proj-id/relay.sock -> /tmp/oubliette-sockets/proj-id)
		hostDir := filepath.Dir(ps.HostPath)
		containerDir := filepath.Dir(ps.ContainerPath)

		// Create host directory if it doesn't exist
		_ = os.MkdirAll(hostDir, 0o755)

		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: hostDir,
			Target: containerDir,
		})
	}

	hostConfig := &dockercontainer.HostConfig{
		Mounts:      mounts,
		AutoRemove:  cfg.AutoRemove,
		NetworkMode: dockercontainer.NetworkMode(cfg.NetworkMode),
		Init:        boolPtr(cfg.Init),
		Resources:   buildResourceConstraints(cfg.Memory, cfg.CPUs),
	}

	resp, err := r.client.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, cfg.Name)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	return resp.ID, nil
}

// Start starts a container
func (r *Runtime) Start(ctx context.Context, containerID string) error {
	if err := r.client.ContainerStart(ctx, containerID, dockercontainer.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}
	return nil
}

// Stop stops a container
func (r *Runtime) Stop(ctx context.Context, containerID string) error {
	return r.client.ContainerStop(ctx, containerID, dockercontainer.StopOptions{})
}

// Remove removes a container
func (r *Runtime) Remove(ctx context.Context, containerID string, force bool) error {
	return r.client.ContainerRemove(ctx, containerID, dockercontainer.RemoveOptions{Force: force})
}

// Exec executes a command in a running container
func (r *Runtime) Exec(ctx context.Context, containerID string, cfg container.ExecConfig) (*container.ExecResult, error) {
	execConfig := dockercontainer.ExecOptions{
		Cmd:          cfg.Cmd,
		Env:          cfg.Env,
		WorkingDir:   cfg.WorkingDir,
		AttachStdout: cfg.AttachStdout,
		AttachStderr: cfg.AttachStderr,
		AttachStdin:  cfg.AttachStdin,
		Tty:          cfg.TTY,
		User:         cfg.User,
	}

	execResp, err := r.client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create exec: %w", err)
	}

	attachResp, err := r.client.ContainerExecAttach(ctx, execResp.ID, dockercontainer.ExecStartOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to attach to exec: %w", err)
	}
	defer attachResp.Close()

	var outBuf, errBuf bytes.Buffer
	if _, err := stdcopy.StdCopy(&outBuf, &errBuf, attachResp.Reader); err != nil {
		return nil, fmt.Errorf("failed to read exec output: %w", err)
	}

	inspectResp, err := r.client.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect exec: %w", err)
	}

	return &container.ExecResult{
		Stdout:   outBuf.String(),
		Stderr:   errBuf.String(),
		ExitCode: inspectResp.ExitCode,
	}, nil
}

// ExecInteractive starts an interactive command execution with I/O pipes
func (r *Runtime) ExecInteractive(ctx context.Context, containerID string, cfg container.ExecConfig) (*container.InteractiveExec, error) {
	execConfig := dockercontainer.ExecOptions{
		Cmd:          cfg.Cmd,
		Env:          cfg.Env,
		WorkingDir:   cfg.WorkingDir,
		AttachStdout: true,
		AttachStderr: true,
		AttachStdin:  true,
		Tty:          false,
		User:         cfg.User,
	}

	execResp, err := r.client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create exec: %w", err)
	}

	attachResp, err := r.client.ContainerExecAttach(ctx, execResp.ID, dockercontainer.ExecStartOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to attach to exec: %w", err)
	}

	// Create pipes for stdout/stderr demuxing
	stdoutReader, stdoutWriter := io.Pipe()
	stderrReader, stderrWriter := io.Pipe()

	// Demux stdout/stderr in background
	go func() {
		defer func() { _ = stdoutWriter.Close() }()
		defer func() { _ = stderrWriter.Close() }()
		_, _ = stdcopy.StdCopy(stdoutWriter, stderrWriter, attachResp.Reader)
	}()

	execID := execResp.ID
	wait := func() (int, error) {
		// Wait for the exec to complete by polling
		for {
			inspectResp, err := r.client.ContainerExecInspect(ctx, execID)
			if err != nil {
				return -1, fmt.Errorf("failed to inspect exec: %w", err)
			}
			if !inspectResp.Running {
				return inspectResp.ExitCode, nil
			}
			select {
			case <-ctx.Done():
				return -1, ctx.Err()
			case <-time.After(100 * time.Millisecond):
			}
		}
	}

	// Wrap the hijacked connection's writer as stdin
	stdin := &hijackedWriteCloser{conn: attachResp}

	return container.NewInteractiveExec(stdin, stdoutReader, stderrReader, wait), nil
}

// hijackedWriteCloser wraps a HijackedResponse to implement io.WriteCloser
type hijackedWriteCloser struct {
	conn types.HijackedResponse
}

func (h *hijackedWriteCloser) Write(p []byte) (n int, err error) {
	return h.conn.Conn.Write(p)
}

func (h *hijackedWriteCloser) Close() error {
	h.conn.Close()
	return nil
}

// Inspect returns container information
func (r *Runtime) Inspect(ctx context.Context, containerID string) (*container.ContainerInfo, error) {
	inspect, err := r.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, err
	}

	var mounts []container.Mount
	for _, m := range inspect.Mounts {
		mounts = append(mounts, container.Mount{
			Type:     container.MountType(m.Type),
			Source:   m.Source,
			Target:   m.Destination,
			ReadOnly: !m.RW,
		})
	}

	status := container.StatusUnknown
	if inspect.State != nil {
		switch inspect.State.Status {
		case "created":
			status = container.StatusCreated
		case "running":
			status = container.StatusRunning
		case "paused":
			status = container.StatusPaused
		case "exited":
			status = container.StatusExited
		case "dead":
			status = container.StatusDead
		}
	}

	var ipAddress string
	if inspect.NetworkSettings != nil && inspect.NetworkSettings.IPAddress != "" {
		ipAddress = inspect.NetworkSettings.IPAddress
	}

	createdAt, _ := time.Parse(time.RFC3339, inspect.Created)

	return &container.ContainerInfo{
		ID:        inspect.ID,
		Name:      inspect.Name,
		Image:     inspect.Image,
		Status:    status,
		IPAddress: ipAddress,
		Mounts:    mounts,
		Env:       inspect.Config.Env,
		CreatedAt: createdAt,
	}, nil
}

// Logs retrieves container logs
func (r *Runtime) Logs(ctx context.Context, containerID string, opts container.LogsOptions) (string, error) {
	options := dockercontainer.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: opts.Timestamps,
		Tail:       opts.Tail,
	}

	if options.Tail == "" {
		options.Tail = "1000"
	}

	logs, err := r.client.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return "", fmt.Errorf("failed to get container logs: %w", err)
	}
	defer func() { _ = logs.Close() }()

	var buf bytes.Buffer
	if _, err := stdcopy.StdCopy(&buf, &buf, logs); err != nil {
		return "", fmt.Errorf("failed to read logs: %w", err)
	}

	return buf.String(), nil
}

// Status returns the container status
func (r *Runtime) Status(ctx context.Context, containerID string) (container.ContainerStatus, error) {
	inspect, err := r.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return container.StatusUnknown, err
	}

	switch inspect.State.Status {
	case "created":
		return container.StatusCreated, nil
	case "running":
		return container.StatusRunning, nil
	case "paused":
		return container.StatusPaused, nil
	case "exited":
		return container.StatusExited, nil
	case "dead":
		return container.StatusDead, nil
	default:
		return container.StatusUnknown, nil
	}
}

// Build builds a Docker image
func (r *Runtime) Build(ctx context.Context, cfg container.BuildConfig) error {
	tarBuf := new(bytes.Buffer)
	tw := tar.NewWriter(tarBuf)
	defer func() { _ = tw.Close() }()

	err := filepath.Walk(cfg.ContextPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(cfg.ContextPath, path)
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer func() { _ = file.Close() }()

			if _, err := io.Copy(tw, file); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create build context: %w", err)
	}

	buildArgs := make(map[string]*string)
	for k, v := range cfg.BuildArgs {
		val := v
		buildArgs[k] = &val
	}

	buildOptions := types.ImageBuildOptions{
		Tags:       []string{cfg.ImageName},
		Dockerfile: cfg.DockerfilePath,
		Remove:     true,
		BuildArgs:  buildArgs,
		Version:    types.BuilderBuildKit,
	}

	resp, err := r.client.ImageBuild(ctx, bytes.NewReader(tarBuf.Bytes()), buildOptions)
	if err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	type buildMessage struct {
		Stream string `json:"stream"`
		Error  string `json:"error"`
	}

	decoder := json.NewDecoder(resp.Body)
	for {
		var msg buildMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode build output: %w", err)
		}

		if msg.Error != "" {
			return fmt.Errorf("build error: %s", msg.Error)
		}

		if msg.Stream != "" {
			fmt.Print(msg.Stream)
		}
	}

	return nil
}

// ImageExists checks if a Docker image exists locally
func (r *Runtime) ImageExists(ctx context.Context, imageName string) (bool, error) {
	_, err := r.client.ImageInspect(ctx, imageName)
	if err != nil {
		if client.IsErrNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to inspect image: %w", err)
	}
	return true, nil
}

// Pull pulls an image from a registry
func (r *Runtime) Pull(ctx context.Context, imageName string) error {
	reader, err := r.client.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	defer func() { _ = reader.Close() }()

	// Stream progress to stdout
	type pullProgress struct {
		Status   string `json:"status"`
		Progress string `json:"progress"`
		ID       string `json:"id"`
		Error    string `json:"error"`
	}

	decoder := json.NewDecoder(reader)
	for {
		var msg pullProgress
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode pull output: %w", err)
		}

		if msg.Error != "" {
			return fmt.Errorf("pull error: %s", msg.Error)
		}

		// Print progress
		if msg.ID != "" {
			fmt.Printf("   %s: %s %s\n", msg.ID, msg.Status, msg.Progress)
		} else if msg.Status != "" {
			fmt.Printf("   %s\n", msg.Status)
		}
	}

	return nil
}

func boolPtr(b bool) *bool {
	return &b
}

// buildResourceConstraints creates Docker resource constraints from config
func buildResourceConstraints(memory string, cpus int) dockercontainer.Resources {
	resources := dockercontainer.Resources{}

	// Parse memory limit (e.g., "4G", "2048M", "1073741824")
	if memory != "" {
		memBytes := parseMemoryString(memory)
		if memBytes > 0 {
			resources.Memory = memBytes
		}
	}

	// Set CPU limit using NanoCPUs (1 CPU = 1e9 NanoCPUs)
	if cpus > 0 {
		resources.NanoCPUs = int64(cpus) * 1e9
	}

	return resources
}

// parseMemoryString converts memory strings like "4G", "2048M" to bytes
func parseMemoryString(mem string) int64 {
	if mem == "" {
		return 0
	}

	var multiplier int64 = 1
	numStr := mem

	// Check for suffix
	if len(mem) > 1 {
		suffix := mem[len(mem)-1]
		switch suffix {
		case 'K', 'k':
			multiplier = 1024
			numStr = mem[:len(mem)-1]
		case 'M', 'm':
			multiplier = 1024 * 1024
			numStr = mem[:len(mem)-1]
		case 'G', 'g':
			multiplier = 1024 * 1024 * 1024
			numStr = mem[:len(mem)-1]
		case 'T', 't':
			multiplier = 1024 * 1024 * 1024 * 1024
			numStr = mem[:len(mem)-1]
		}
	}

	var value int64
	_, _ = fmt.Sscanf(numStr, "%d", &value)
	return value * multiplier
}
