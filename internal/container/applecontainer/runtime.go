package applecontainer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/HyphaGroup/oubliette/internal/container"
)

// Runtime implements container.Runtime using Apple Container CLI
type Runtime struct {
	binaryPath string
}

// NewRuntime creates a new Apple Container runtime
func NewRuntime() (*Runtime, error) {
	binaryPath := os.Getenv("APPLE_CONTAINER_BINARY")
	if binaryPath == "" {
		binaryPath = findContainerBinary()
	}

	return &Runtime{binaryPath: binaryPath}, nil
}

// findContainerBinary searches common locations for the container binary
func findContainerBinary() string {
	candidates := []string{
		"/opt/homebrew/bin/container", // Homebrew on Apple Silicon
		"/usr/local/bin/container",    // Standard install / Homebrew on Intel
		"/usr/bin/container",          // System install
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Fall back to PATH lookup
	if path, err := exec.LookPath("container"); err == nil {
		return path
	}

	// Default to standard location
	return "/usr/local/bin/container"
}

// Name returns the runtime name
func (r *Runtime) Name() string {
	return "apple-container"
}

// IsAvailable checks if Apple Container is available and running
func (r *Runtime) IsAvailable() bool {
	if _, err := exec.LookPath(r.binaryPath); err != nil {
		return false
	}

	cmd := exec.Command(r.binaryPath, "system", "status")
	return cmd.Run() == nil
}

// Ping verifies Apple Container system is running
func (r *Runtime) Ping(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, r.binaryPath, "system", "status")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("apple container system not running: %w", err)
	}
	return nil
}

// Close is a no-op for CLI-based runtime
func (r *Runtime) Close() error {
	return nil
}

// Create creates a new container
func (r *Runtime) Create(ctx context.Context, cfg container.CreateConfig) (string, error) {
	args := []string{"create"}

	if cfg.Name != "" {
		args = append(args, "--name", cfg.Name)
	}

	for _, env := range cfg.Env {
		args = append(args, "-e", env)
	}

	if cfg.WorkingDir != "" {
		args = append(args, "-w", cfg.WorkingDir)
	}

	for _, m := range cfg.Mounts {
		mountStr := fmt.Sprintf("%s:%s", m.Source, m.Target)
		if m.ReadOnly {
			mountStr += ":ro"
		}
		args = append(args, "-v", mountStr)
	}

	// Note: Apple Container doesn't support --init flag
	// The container runs with a proper init system by default

	if cfg.AutoRemove {
		args = append(args, "--rm")
	}

	if cfg.NetworkMode != "" && cfg.NetworkMode != "bridge" {
		args = append(args, "--network", cfg.NetworkMode)
	}

	// Resource limits
	if cfg.Memory != "" {
		args = append(args, "-m", cfg.Memory)
	}
	if cfg.CPUs > 0 {
		args = append(args, "-c", fmt.Sprintf("%d", cfg.CPUs))
	}

	// Socket publishing (container -> host direction)
	for _, ps := range cfg.PublishedSockets {
		args = append(args, "--publish-socket", fmt.Sprintf("%s:%s", ps.HostPath, ps.ContainerPath))
	}

	args = append(args, cfg.Image)
	args = append(args, cfg.Cmd...)

	cmd := exec.CommandContext(ctx, r.binaryPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w, output: %s", err, string(output))
	}

	containerID := strings.TrimSpace(string(output))
	return containerID, nil
}

// Start starts a container
func (r *Runtime) Start(ctx context.Context, containerID string) error {
	cmd := exec.CommandContext(ctx, r.binaryPath, "start", containerID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start container: %w, output: %s", err, string(output))
	}
	return nil
}

// Stop stops a container
func (r *Runtime) Stop(ctx context.Context, containerID string) error {
	cmd := exec.CommandContext(ctx, r.binaryPath, "stop", containerID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop container: %w, output: %s", err, string(output))
	}
	return nil
}

// Remove removes a container
func (r *Runtime) Remove(ctx context.Context, containerID string, force bool) error {
	args := []string{"rm"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, containerID)

	cmd := exec.CommandContext(ctx, r.binaryPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove container: %w, output: %s", err, string(output))
	}
	return nil
}

// Exec executes a command in a running container
func (r *Runtime) Exec(ctx context.Context, containerID string, cfg container.ExecConfig) (*container.ExecResult, error) {
	args := []string{"exec"}

	if cfg.TTY {
		args = append(args, "-t")
	}

	if cfg.AttachStdin {
		args = append(args, "-i")
	}

	for _, env := range cfg.Env {
		args = append(args, "-e", env)
	}

	if cfg.WorkingDir != "" {
		args = append(args, "-w", cfg.WorkingDir)
	}

	if cfg.User != "" {
		args = append(args, "-u", cfg.User)
	}

	args = append(args, containerID)
	args = append(args, cfg.Cmd...)

	cmd := exec.CommandContext(ctx, r.binaryPath, args...)
	output, err := cmd.CombinedOutput()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("failed to exec in container: %w", err)
		}
	}

	return &container.ExecResult{
		Stdout:   string(output),
		Stderr:   "",
		ExitCode: exitCode,
	}, nil
}

// ExecInteractive starts an interactive command execution with I/O pipes
func (r *Runtime) ExecInteractive(ctx context.Context, containerID string, cfg container.ExecConfig) (*container.InteractiveExec, error) {
	args := []string{"exec", "-i"} // -i for interactive stdin

	for _, env := range cfg.Env {
		args = append(args, "-e", env)
	}

	if cfg.WorkingDir != "" {
		args = append(args, "-w", cfg.WorkingDir)
	}

	if cfg.User != "" {
		args = append(args, "-u", cfg.User)
	}

	args = append(args, containerID)
	args = append(args, cfg.Cmd...)

	cmd := exec.CommandContext(ctx, r.binaryPath, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		_ = stderr.Close()
		return nil, fmt.Errorf("failed to start interactive exec: %w", err)
	}

	wait := func() (int, error) {
		err := cmd.Wait()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				return exitErr.ExitCode(), nil
			}
			return -1, err
		}
		return 0, nil
	}

	return container.NewInteractiveExec(stdin, stdout, stderr, wait), nil
}

// appleContainerInspect represents the JSON output from `container inspect`
type appleContainerInspect struct {
	Status   string `json:"status"`
	Networks []struct {
		Address  string `json:"address"`
		Gateway  string `json:"gateway"`
		Hostname string `json:"hostname"`
		Network  string `json:"network"`
	} `json:"networks"`
	Configuration struct {
		ID       string `json:"id"`
		Hostname string `json:"hostname"`
		Mounts   []struct {
			Source      string `json:"source"`
			Destination string `json:"destination"`
			ReadOnly    bool   `json:"readonly"`
		} `json:"mounts"`
		Resources struct {
			CPUs          int   `json:"cpus"`
			MemoryInBytes int64 `json:"memoryInBytes"`
		} `json:"resources"`
	} `json:"configuration"`
}

// Inspect returns container information
func (r *Runtime) Inspect(ctx context.Context, containerID string) (*container.ContainerInfo, error) {
	cmd := exec.CommandContext(ctx, r.binaryPath, "inspect", containerID)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	var inspects []appleContainerInspect
	if err := json.Unmarshal(output, &inspects); err != nil {
		return nil, fmt.Errorf("failed to parse inspect output: %w", err)
	}

	if len(inspects) == 0 {
		return nil, fmt.Errorf("container not found: %s", containerID)
	}

	inspect := inspects[0]

	var mounts []container.Mount
	for _, m := range inspect.Configuration.Mounts {
		mounts = append(mounts, container.Mount{
			Type:     container.MountTypeBind,
			Source:   m.Source,
			Target:   m.Destination,
			ReadOnly: m.ReadOnly,
		})
	}

	status := container.StatusUnknown
	switch inspect.Status {
	case "created":
		status = container.StatusCreated
	case "running":
		status = container.StatusRunning
	case "stopped", "exited":
		status = container.StatusExited
	}

	var ipAddress string
	if len(inspect.Networks) > 0 {
		ipAddress = strings.Split(inspect.Networks[0].Address, "/")[0]
	}

	return &container.ContainerInfo{
		ID:        inspect.Configuration.ID,
		Name:      inspect.Configuration.Hostname,
		Status:    status,
		IPAddress: ipAddress,
		Mounts:    mounts,
	}, nil
}

// Logs retrieves container logs
func (r *Runtime) Logs(ctx context.Context, containerID string, opts container.LogsOptions) (string, error) {
	args := []string{"logs"}

	if opts.Tail != "" {
		args = append(args, "-n", opts.Tail)
	}

	args = append(args, containerID)

	cmd := exec.CommandContext(ctx, r.binaryPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get container logs: %w, output: %s", err, string(output))
	}

	return string(output), nil
}

// Status returns the container status
func (r *Runtime) Status(ctx context.Context, containerID string) (container.ContainerStatus, error) {
	info, err := r.Inspect(ctx, containerID)
	if err != nil {
		return container.StatusUnknown, err
	}
	return info.Status, nil
}

// Build builds an image using Apple Container
func (r *Runtime) Build(ctx context.Context, cfg container.BuildConfig) error {
	args := []string{"build"}

	if cfg.ImageName != "" {
		args = append(args, "-t", cfg.ImageName)
	}

	if cfg.DockerfilePath != "" {
		args = append(args, "-f", cfg.DockerfilePath)
	}

	for k, v := range cfg.BuildArgs {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", k, v))
	}

	args = append(args, cfg.ContextPath)

	cmd := exec.CommandContext(ctx, r.binaryPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}

	return nil
}

// ImageExists checks if an image exists in Apple Container
func (r *Runtime) ImageExists(ctx context.Context, imageName string) (bool, error) {
	cmd := exec.CommandContext(ctx, r.binaryPath, "image", "inspect", imageName)
	if err := cmd.Run(); err != nil {
		// Check if it's a "not found" error vs actual error
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, fmt.Errorf("failed to inspect image: %w", err)
	}
	return true, nil
}

// Pull pulls an image from a registry
func (r *Runtime) Pull(ctx context.Context, imageName string) error {
	cmd := exec.CommandContext(ctx, r.binaryPath, "image", "pull", imageName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	return nil
}

// EnsureSystemRunning starts the Apple Container system if not running
func (r *Runtime) EnsureSystemRunning(ctx context.Context) error {
	if r.IsAvailable() {
		return nil
	}

	cmd := exec.CommandContext(ctx, r.binaryPath, "system", "start")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start apple container system: %w, output: %s", err, string(output))
	}

	// Wait for system to be ready
	for i := 0; i < 30; i++ {
		if r.IsAvailable() {
			return nil
		}
		time.Sleep(time.Second)
	}

	return fmt.Errorf("apple container system did not become ready")
}
