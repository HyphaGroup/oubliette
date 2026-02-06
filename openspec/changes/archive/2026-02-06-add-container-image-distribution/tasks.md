# Tasks: Add Container Image Distribution

## Phase 1: Config Schema

- [x] Add `Containers map[string]string` to `UnifiedConfig` in `internal/config/unified.go`
- [x] Add `Containers` field to `LoadedConfig` in `internal/config/loader.go`
- [x] Set default containers in `applyUnifiedDefaults()` with ghcr.io paths
- [x] Add `OUBLIETTE_DEV` check to use local image names in dev mode
- [x] In dev mode, skip Pull and error if local image missing
- [x] Update `oubliette init` template to include `containers` section
- [x] Update `config/oubliette.jsonc.example` with containers section
- [x] Add pre-pull of all configured images (`base`, `dev`) during `oubliette init`

## Phase 2: Runtime Interface

- [x] Add `Pull(ctx context.Context, imageName string) error` to `Runtime` interface in `internal/container/runtime.go`
- [x] Implement `Pull` for Docker runtime in `internal/container/docker/runtime.go` with progress streaming
- [x] Implement `Pull` for Apple Container runtime in `internal/container/applecontainer/runtime.go`
- [x] Add Pull to MockRuntime in testutil
- [x] Add Pull to mockRuntimeForCache in cache_test.go

## Phase 3: ImageManager

- [x] Create `ImageManager` struct in `internal/container/images.go`
- [x] Implement `GetImageName(typeName string) (string, error)`
- [x] Implement `ValidTypes() []string`
- [x] Implement `IsValidType(typeName string) bool`
- [x] Implement `EnsureImageExists(ctx, typeName)` with pull logic
- [x] Implement `EnsureAllImages(ctx)` for pre-pulling all images

## Phase 4: Remove Hardcoded Types

- [x] Remove `ContainerType` constants from `internal/container/types.go`
- [x] Remove `ValidContainerTypes()` function
- [x] Remove `IsValidContainerType()` function
- [x] Remove `ImageNameForType()` function
- [x] Remove `ContainerTypeBuilder` struct and methods
- [x] Delete `internal/container/types_test.go`

## Phase 5: Server Integration

- [x] Add `ImageManager` to `Server` struct in `internal/mcp/server.go`
- [x] Add `ImageManager` to `ServerConfig`
- [x] Initialize `ImageManager` in `cmd/server/main.go` with config containers
- [x] Pass `ImageManager` to Server via ServerConfig
- [x] Add `SetContainers()` to project manager for image name resolution
- [x] Update project manager `Create()` to use `GetImageNameForType()`
- [x] Update `internal/mcp/handlers_project.go` to validate types via ImageManager
- [x] Update `internal/mcp/handlers_project.go` `handleProjectOptions` to return types from config
- [x] Replace `image_rebuild` with `container_refresh` in `internal/mcp/handlers_container.go`
- [x] Implement `container_refresh`: pull image, check no active sessions, restart container
- [x] Update `internal/mcp/tools_registry.go` to register `container_refresh` instead of `image_rebuild`
- [x] Update `createAndStartContainer` to pull image if missing

## Phase 6: CI/CD Pipeline

- [x] Add `packages: write` permission to `.github/workflows/release.yml`
- [x] Add `containers` job to `.github/workflows/release.yml`
- [x] Configure ghcr.io login with GITHUB_TOKEN
- [x] Build and push `ghcr.io/hyphagroup/oubliette-base:latest` and `:$VERSION`
- [x] Build and push `ghcr.io/hyphagroup/oubliette-dev:latest` and `:$VERSION`
- [x] Add `BASE_IMAGE` build arg to dev Dockerfile for CI builds

## Phase 7: Development Workflow

- [x] `OUBLIETTE_DEV=1` support in `applyUnifiedDefaults()` uses local image names
- [x] In dev mode, ImageManager returns error with build instructions if image missing
- [x] `build.sh` already builds local images without changes needed

## Phase 8: Documentation

- [x] Rewrite `docs/CONTAINER_TYPES.md` with config-driven architecture
- [x] Update `docs/CONFIGURATION.md` with containers section and unified config format

## Phase 9: CLI Container Subcommand

- [x] Add `container` subcommand routing in `cmd/server/main.go`
- [x] Implement `oubliette container list` - show running containers
- [x] Implement `oubliette container refresh <project_id>` - pull and restart
- [x] Implement `oubliette container stop <project_id> | --all` - stop containers
- [x] Add `printContainerUsage()` help text

## Phase 10: Testing

- [x] Update `test/pkg/suites/project.go` for config-driven container types (osint -> base)
- [x] Update `test/pkg/suites/container.go` to use `container_refresh` instead of `image_rebuild`
