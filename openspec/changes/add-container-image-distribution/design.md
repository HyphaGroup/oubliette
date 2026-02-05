# Design: Container Image Distribution

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      GitHub Actions (Release)                    │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────┐  │
│  │ Build base  │───▶│ Build dev   │───▶│ Push to ghcr.io     │  │
│  │ image       │    │ image       │    │ - oubliette-base    │  │
│  └─────────────┘    └─────────────┘    │ - oubliette-dev     │  │
│                                         └─────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                    GitHub Container Registry                     │
│         ghcr.io/hyphagroup/oubliette-base:latest                │
│         ghcr.io/hyphagroup/oubliette-dev:latest                 │
└─────────────────────────────────────────────────────────────────┘
                                │
                    ┌───────────┴───────────┐
                    ▼                       ▼
┌─────────────────────────┐   ┌─────────────────────────┐
│   Docker Runtime        │   │  Apple Container Runtime │
│   docker pull ghcr...   │   │  container image pull... │
└─────────────────────────┘   └─────────────────────────┘
```

## Config-Driven Container Types

### oubliette.jsonc Structure

```jsonc
{
  "containers": {
    // Built-in types (shipped by default)
    "base": "ghcr.io/hyphagroup/oubliette-base:latest",
    "dev": "ghcr.io/hyphagroup/oubliette-dev:latest",
    
    // User-defined custom images
    "python-ml": "myregistry.com/ml-agent:v2",
    "node-frontend": "my-local-image:latest"
  }
}
```

### Config Types

```go
// ContainersConfig maps container type names to image references
type ContainersConfig map[string]string

// In UnifiedConfig
type UnifiedConfig struct {
    // ... existing fields ...
    Containers ContainersConfig `json:"containers"`
}
```

### Default Config (from oubliette init)

```jsonc
"containers": {
  "base": "ghcr.io/hyphagroup/oubliette-base:latest",
  "dev": "ghcr.io/hyphagroup/oubliette-dev:latest"
}
```

## ImageManager

Replace hardcoded `ImageNameForType()` and `ContainerTypeBuilder` with `ImageManager`:

```go
type ImageManager struct {
    runtime    Runtime
    containers map[string]string  // type name -> image reference
}

func NewImageManager(runtime Runtime, containers map[string]string) *ImageManager {
    return &ImageManager{
        runtime:    runtime,
        containers: containers,
    }
}

// GetImageName returns the image reference for a container type
func (m *ImageManager) GetImageName(typeName string) (string, error) {
    image, ok := m.containers[typeName]
    if !ok {
        return "", fmt.Errorf("unknown container type: %s", typeName)
    }
    return image, nil
}

// ValidTypes returns all configured container type names
func (m *ImageManager) ValidTypes() []string {
    types := make([]string, 0, len(m.containers))
    for t := range m.containers {
        types = append(types, t)
    }
    sort.Strings(types)
    return types
}

// IsValidType checks if a container type is configured
func (m *ImageManager) IsValidType(typeName string) bool {
    _, ok := m.containers[typeName]
    return ok
}

// EnsureImageExists pulls the image if it doesn't exist locally
func (m *ImageManager) EnsureImageExists(ctx context.Context, typeName string) error {
    imageName, err := m.GetImageName(typeName)
    if err != nil {
        return err
    }
    
    exists, err := m.runtime.ImageExists(ctx, imageName)
    if err != nil {
        return fmt.Errorf("checking image %s: %w", imageName, err)
    }
    
    if exists {
        return nil
    }
    
    log.Printf("Pulling image %s...", imageName)
    if err := m.runtime.Pull(ctx, imageName); err != nil {
        return fmt.Errorf("pulling image %s: %w", imageName, err)
    }
    
    return nil
}
```

## Runtime Interface Change

Add `Pull` method to the `Runtime` interface:

```go
type Runtime interface {
    // ... existing methods ...
    
    // Pull downloads an image from a registry
    Pull(ctx context.Context, imageName string) error
}
```

### Docker Implementation

```go
func (r *Runtime) Pull(ctx context.Context, imageName string) error {
    out, err := r.client.ImagePull(ctx, imageName, image.PullOptions{})
    if err != nil {
        return fmt.Errorf("failed to pull image: %w", err)
    }
    defer out.Close()
    io.Copy(io.Discard, out)
    return nil
}
```

### Apple Container Implementation

```go
func (r *Runtime) Pull(ctx context.Context, imageName string) error {
    cmd := exec.CommandContext(ctx, r.binaryPath, "image", "pull", imageName)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to pull image: %w", err)
    }
    return nil
}
```

## Migration: Removing Hardcoded Types

### Files to Change

1. **`internal/container/types.go`**
   - Remove `ContainerType` constants (Base, Dev, Osint)
   - Remove `ValidContainerTypes()`, `IsValidContainerType()`
   - Remove `ImageNameForType()`
   - Remove `ContainerTypeBuilder`
   - Add `ImageManager` struct

2. **`internal/mcp/handlers_project.go`**
   - Change `handleCreateProject` to validate against config, not hardcoded list
   - Change `handleProjectOptions` to return types from config

3. **`internal/mcp/handlers_container.go`**
   - Replace `image_rebuild` with `container_refresh`
   - `container_refresh` pulls image and restarts container
   - Fail if active sessions exist (user must end them first)

4. **`internal/project/manager.go`**
   - Use `ImageManager.GetImageName()` instead of hardcoded format string
   - Pass ImageManager from server startup

## container_refresh Tool

Replaces `image_rebuild`. Pulls latest image and restarts the project's container.

```json
{
  "name": "container_refresh",
  "arguments": {
    "project_id": "proj_xxx"
  }
}
```

### Implementation Flow

```go
func (s *Server) handleContainerRefresh(ctx context.Context, params *ContainerRefreshParams) error {
    // 1. Get project and its container type
    proj, err := s.projectMgr.Get(params.ProjectID)
    if err != nil {
        return err
    }
    
    // 2. Check for active sessions - fail if any exist
    activeSessions := s.activeSessionMgr.ListByProject(params.ProjectID)
    if len(activeSessions) > 0 {
        return fmt.Errorf("cannot refresh container: %d active session(s) exist - end them first", len(activeSessions))
    }
    
    // 3. Get image name from config
    imageName, err := s.imageMgr.GetImageName(proj.ContainerType)
    if err != nil {
        return err
    }
    
    // 4. Pull latest image
    log.Printf("Pulling %s...", imageName)
    if err := s.runtime.Pull(ctx, imageName); err != nil {
        return fmt.Errorf("failed to pull image: %w", err)
    }
    
    // 5. Stop and remove current container
    containerName := containerNameForProject(params.ProjectID)
    if err := s.runtime.Stop(ctx, containerName); err != nil {
        // Ignore if not running
    }
    if err := s.runtime.Remove(ctx, containerName, true); err != nil {
        // Ignore if doesn't exist
    }
    
    // 6. Start fresh container
    _, err = s.createAndStartContainer(ctx, containerName, imageName, params.ProjectID)
    if err != nil {
        return fmt.Errorf("failed to start container: %w", err)
    }
    
    return nil
}
```

### Use Cases

- **After oubliette release**: Pull new container with updated tools
- **Custom image update**: User rebuilt their custom image, refresh to use it
- **Troubleshooting**: "Have you tried turning it off and on again?"

## CLI Container Subcommand

Mirror of MCP functionality for direct user access:

```bash
oubliette container list                 # list running containers
oubliette container refresh              # refresh all running containers  
oubliette container refresh <project_id> # refresh specific container
oubliette container stop <project_id>    # stop specific container
oubliette container stop --all           # stop all containers
```

### Implementation

```go
func cmdContainer(args []string) {
    if len(args) < 1 {
        printContainerUsage()
        os.Exit(1)
    }

    // Initialize runtime
    runtime := initRuntime()
    
    switch args[0] {
    case "list":
        containerList(runtime)
    case "refresh":
        if len(args) > 1 {
            containerRefresh(runtime, args[1])  // specific project
        } else {
            containerRefreshAll(runtime)        // all running
        }
    case "stop":
        if len(args) > 1 && args[1] == "--all" {
            containerStopAll(runtime)
        } else if len(args) > 1 {
            containerStop(runtime, args[1])
        } else {
            fmt.Fprintln(os.Stderr, "Error: project_id or --all required")
            os.Exit(1)
        }
    default:
        printContainerUsage()
        os.Exit(1)
    }
}
```

### Example Output

```
$ oubliette container list
PROJECT ID                            IMAGE                                      STATUS
proj_abc123                           ghcr.io/hyphagroup/oubliette-dev:latest   running
proj_def456                           myregistry.com/custom:v1                   running

$ oubliette container refresh
Refreshing proj_abc123...
  Pulling ghcr.io/hyphagroup/oubliette-dev:latest...
  Restarting container...
  Done.
Refreshing proj_def456...
  Pulling myregistry.com/custom:v1...
  Restarting container...
  Done.
All containers refreshed.
```

## GitHub Actions Workflow

Add container build job to `release.yml`:

```yaml
container:
  runs-on: ubuntu-latest
  permissions:
    contents: read
    packages: write
  steps:
    - uses: actions/checkout@v4
    
    - name: Extract version from tag
      id: version
      run: echo "VERSION=${GITHUB_REF#refs/tags/}" >> $GITHUB_OUTPUT
    
    - name: Log in to GitHub Container Registry
      uses: docker/login-action@v3
      with:
        registry: ghcr.io
        username: ${{ github.actor }}
        password: ${{ secrets.GITHUB_TOKEN }}
    
    - name: Build and push base image
      uses: docker/build-push-action@v5
      with:
        context: .
        file: containers/base/Dockerfile
        push: true
        tags: |
          ghcr.io/hyphagroup/oubliette-base:latest
          ghcr.io/hyphagroup/oubliette-base:${{ steps.version.outputs.VERSION }}
    
    - name: Build and push dev image
      uses: docker/build-push-action@v5
      with:
        context: .
        file: containers/dev/Dockerfile
        push: true
        tags: |
          ghcr.io/hyphagroup/oubliette-dev:latest
          ghcr.io/hyphagroup/oubliette-dev:${{ steps.version.outputs.VERSION }}
```

Images are tagged with both `:latest` and the version tag (e.g., `:v1.0.0`) for reproducibility.

## Development Mode

For local development, use `OUBLIETTE_DEV=1`:

```go
func DefaultContainers() map[string]string {
    if os.Getenv("OUBLIETTE_DEV") == "1" {
        return map[string]string{
            "base": "oubliette-base:latest",
            "dev":  "oubliette-dev:latest",
        }
    }
    return map[string]string{
        "base": "ghcr.io/hyphagroup/oubliette-base:latest",
        "dev":  "ghcr.io/hyphagroup/oubliette-dev:latest",
    }
}

// In dev mode, skip pull and require local images
func (m *ImageManager) EnsureImageExists(ctx context.Context, typeName string) error {
    imageName, err := m.GetImageName(typeName)
    if err != nil {
        return err
    }
    
    exists, err := m.runtime.ImageExists(ctx, imageName)
    if err != nil {
        return fmt.Errorf("checking image %s: %w", imageName, err)
    }
    
    if exists {
        return nil
    }
    
    // In dev mode, don't pull - local images must be built with build.sh
    if os.Getenv("OUBLIETTE_DEV") == "1" {
        return fmt.Errorf("image %s not found locally - run ./build.sh to build it", imageName)
    }
    
    log.Printf("Pulling image %s...", imageName)
    if err := m.runtime.Pull(ctx, imageName); err != nil {
        return fmt.Errorf("pulling image %s: %w", imageName, err)
    }
    
    return nil
}
```

## Custom Container Documentation

Update `docs/CONTAINER_TYPES.md` to explain:

1. **Using custom images** - Add to `containers` in config
2. **Extending base images** - `FROM ghcr.io/hyphagroup/oubliette-base:latest`
3. **Required components** - What must be in custom images (oubliette-client, oubliette-relay, etc.)
4. **Building locally** - For development/testing
5. **Publishing** - Push to your own registry

## Pull Progress

Image pulls can take several minutes for large images (~2GB for dev). The `Pull` method SHALL stream progress to stdout so users see download status.

### Docker Implementation

```go
func (r *Runtime) Pull(ctx context.Context, imageName string) error {
    out, err := r.client.ImagePull(ctx, imageName, image.PullOptions{})
    if err != nil {
        return fmt.Errorf("failed to pull image: %w", err)
    }
    defer out.Close()
    
    // Stream progress to stdout
    decoder := json.NewDecoder(out)
    for {
        var event struct {
            Status   string `json:"status"`
            Progress string `json:"progress"`
        }
        if err := decoder.Decode(&event); err == io.EOF {
            break
        } else if err != nil {
            return err
        }
        if event.Progress != "" {
            fmt.Printf("\r%s: %s", event.Status, event.Progress)
        }
    }
    fmt.Println()
    return nil
}
```

### Apple Container Implementation

Apple Container's `image pull` command outputs progress by default when stdout is a TTY.

## Init Pre-Pull

`oubliette init` SHALL pre-pull all configured container images so the first session spawn is fast.

```go
func cmdInit() {
    // ... create config, directories, token ...
    
    // Pre-pull all configured images
    fmt.Println("Pulling container images...")
    runtime := detectRuntime()
    images := []string{
        "ghcr.io/hyphagroup/oubliette-base:latest",
        "ghcr.io/hyphagroup/oubliette-dev:latest",
    }
    for _, imageName := range images {
        fmt.Printf("Pulling %s...\n", imageName)
        if err := runtime.Pull(ctx, imageName); err != nil {
            fmt.Printf("Warning: failed to pull image %s: %v\n", imageName, err)
            fmt.Println("You can pull manually later with:")
            fmt.Printf("  docker pull %s\n", imageName)
        }
    }
}
```

## Error Handling

If pull fails:
1. Log clear error message with image name
2. Suggest checking network connectivity
3. Suggest running `docker pull` / `container image pull` manually for debugging
4. Exit with non-zero status (unless during init, where it's a warning)

## Security Considerations

- ghcr.io images are public (open source project)
- No credentials needed to pull public images
- Images are built in GitHub Actions with auditable workflow
- GITHUB_TOKEN used for push has minimal scope (packages:write)
- Custom images are user's responsibility
