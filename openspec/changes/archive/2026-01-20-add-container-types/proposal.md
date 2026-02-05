# Proposal: Add Container Types

## Summary

Add a container type system with base, dev, and osint images that projects can select at creation time. Container definitions live in `containers/` directory and are built on-demand or during instance setup.

## Motivation

Currently all projects use the same monolithic container image (`internal/container/Dockerfile`) which includes:
- Development tools (Node.js, Python, Java, build tools)
- CLI tools (gh, glab, ripgrep)
- Factory Droid CLI

**Problems:**
- One-size-fits-all approach wastes resources for simple tasks
- No way to add specialized tooling (OSINT, data science, etc.)
- Large image size (~2GB) even for chat-only use cases

## Proposed Solution

### Container Type Hierarchy

Container types are discovered by scanning the `containers/` directory. Each subdirectory with a `Dockerfile` is a type. Metadata (description) comes from `metadata.json` in each type's directory.

```
containers/
├── base/
│   ├── Dockerfile        # Minimal: bash, curl, git, droid CLI
│   └── metadata.json     # {"description": "Minimal runtime environment"}
├── dev/
│   ├── Dockerfile        # FROM base + Node, Python, Java, build tools
│   └── metadata.json     # {"description": "Full development environment"}
└── osint/
    ├── Dockerfile        # FROM base + OSINT tools
    └── metadata.json     # {"description": "OSINT/reconnaissance tools"}
```

### Image Naming

- `oubliette-base:latest`
- `oubliette-dev:latest`
- `oubliette-osint:latest`

### Project Assignment

```bash
# Create project with specific container type
project_create name="my-osint-project" container_type="osint"

# Default to "dev" if not specified (backwards compatible)
project_create name="my-project"  # uses dev
```

### Build Triggers

1. **On-demand**: When project requests a type that isn't built yet
2. **Instance setup**: `manager.sh create` builds all container types
3. **Manual rebuild**: `manager.sh rebuild-images` or `image_rebuild` MCP tool

## Scope

### In Scope
- `containers/` directory with base, dev, osint Dockerfiles
- `container_type` field on Project and CreateProjectRequest
- Update `project_create` MCP tool to accept `container_type` parameter
- Build images on-demand if missing
- `manager.sh rebuild-images` command to build all types
- Update `image_rebuild` to support specific types

### Out of Scope
- Remote registry (DockerHub/GHCR) - images stay local
- Custom user-defined container types
- Per-project Dockerfile overrides (existing feature, unchanged)

## Container Contents

### base
Minimal environment for lightweight tasks:
- Debian bookworm-slim
- bash, curl, wget, git, jq
- ca-certificates
- Factory Droid CLI
- oubliette-relay, oubliette-client

### dev (extends base)
Full development environment (current Dockerfile contents):
- Node.js (via nvm), Python (via uv), Java (via SDKMAN)
- Build tools: gcc, make, cmake
- CLI tools: gh, glab, ripgrep, fd, jq, yq
- OpenSpec CLI

### osint (extends base)
OSINT/reconnaissance tools:
- theHarvester
- Shodan CLI
- Amass
- subfinder
- httpx
- nuclei
- Python with common OSINT libraries

## Related Changes

- **add-github-account-registry**: Creates `project_options` tool; this change adds `container_types` section to it
