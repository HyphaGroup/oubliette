# Proposal: Add Container Image Distribution

## Problem

When users install Oubliette via the install script and run `oubliette init`, the container images (`oubliette-base`, `oubliette-dev`) don't exist on their machine. Currently, images are only built locally via `./build.sh` during development.

**Current state:**
- `build.sh` builds images locally using `docker build` and syncs to Apple Container
- Server assumes images exist (no pull/build-on-demand logic)
- Container types are hardcoded in `internal/container/types.go`
- `ImageNameForType()` returns local names like `oubliette-dev:latest`
- Users cannot run Oubliette without first cloning repo and running `build.sh`

**Impact:** The entire install flow (`curl | bash` → `oubliette init` → `oubliette`) is broken for end users.

## Solution

1. **Move container definitions to config** - Define available containers in `oubliette.jsonc`
2. **Use GitHub Container Registry** - Push pre-built images to `ghcr.io/hyphagroup/`
3. **Pull on demand** - Server pulls missing images automatically
4. **Support custom images** - Users can add their own images to config

### Config-Driven Container Types

```jsonc
// oubliette.jsonc
{
  "containers": {
    "base": "ghcr.io/hyphagroup/oubliette-base:latest",
    "dev": "ghcr.io/hyphagroup/oubliette-dev:latest",
    
    // Users can add custom images
    "python-ml": "myregistry.com/ml-agent:v2",
    "my-custom": "my-local-image:latest"
  }
}
```

## Scope

| Component | Change |
|-----------|--------|
| `internal/config/unified.go` | Add `Containers` section to UnifiedConfig |
| `internal/config/loader.go` | Add ContainersConfig to LoadedConfig |
| `internal/container/types.go` | Remove hardcoded types, add ImageManager |
| `internal/container/runtime.go` | Add `Pull(ctx, imageName)` to Runtime interface |
| `internal/container/docker/runtime.go` | Implement Docker pull |
| `internal/container/applecontainer/runtime.go` | Implement Apple Container pull |
| `internal/project/manager.go` | Use ImageManager instead of ImageNameForType() |
| `internal/mcp/handlers_project.go` | Get container types from config |
| `internal/mcp/handlers_container.go` | Replace `image_rebuild` with `container_refresh` |
| `internal/mcp/tools_registry.go` | Register `container_refresh` instead of `image_rebuild` |
| `cmd/server/main.go` | Initialize ImageManager, remove ContainerTypeBuilder |
| `.github/workflows/release.yml` | Add container build + push to ghcr.io |
| `docs/CONTAINER_TYPES.md` | Document custom container creation |

## Out of Scope

- Building images on user machines (requires Go toolchain + source)
- Multi-architecture images (ARM64 Linux) - defer to future work
- Offline/air-gapped usage - defer to future work
- Private registry authentication - images are public
- `osint` container type - defer to separate proposal

## Alternatives Considered

1. **Tarball distribution** - Export images as `.tar.gz`, attach to GitHub Release
   - Rejected: Large downloads (~400MB), no incremental updates, more complex

2. **Build on first run** - Ship Dockerfiles, build when image missing
   - Rejected: Requires Go toolchain on user machine, slow first-run experience

3. **Keep hardcoded types** - Only support base/dev/osint
   - Rejected: Users need flexibility for custom tooling/environments
