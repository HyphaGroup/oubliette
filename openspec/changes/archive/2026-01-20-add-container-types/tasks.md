# Tasks: Add Container Types

## Container Definitions

- [x] 1. Create `containers/base/Dockerfile` and `metadata.json`
  - Dockerfile: Debian bookworm-slim
  - Essential tools: bash, curl, wget, git, jq, ca-certificates
  - Non-root user (gogol) with sudo
  - Factory Droid CLI installation
  - Build stage for oubliette-relay and oubliette-client
  - Container init script entrypoint
  - metadata.json: `{"description": "Minimal runtime environment"}`

- [x] 2. Create `containers/dev/Dockerfile` and `metadata.json`
  - Dockerfile: FROM oubliette-base:latest
  - Node.js via nvm
  - Python via uv
  - Java via SDKMAN
  - Build tools: gcc, make, cmake, pkg-config
  - CLI tools: gh, glab, ripgrep, fd, jq, yq
  - OpenSpec CLI
  - metadata.json: `{"description": "Full development environment"}`

- [x] 3. Create `containers/osint/Dockerfile` and `metadata.json`
  - Dockerfile: FROM oubliette-base:latest
  - Python with pip
  - theHarvester
  - Shodan CLI
  - Amass, subfinder, httpx, nuclei (Go tools)
  - Common Python OSINT libraries (requests, beautifulsoup4, dnspython)
  - metadata.json: `{"description": "OSINT/reconnaissance tools"}`

- [x] 4. Remove/migrate `internal/container/Dockerfile`
  - Update references to use containers/dev/
  - Keep as symlink or remove entirely

## Project Model Updates

- [x] 5. Add `ContainerType` field to Project struct in `types.go`
  - Type: string
  - Default value: "dev"
  - Valid values: "base", "dev", "osint"

- [x] 6. Add `ContainerType` to CreateProjectRequest

- [x] 7. Update `project_create` MCP tool schema to accept `container_type` parameter

- [x] 8. Update project creation logic to set ImageName from container type
  - Map: `oubliette-{container_type}:latest`

## Build Infrastructure

- [x] 9. Add helper function to check if container image exists
  - Query runtime for image by name

- [x] 10. Add `BuildContainerType(ctx, typeName)` function
  - Build specific container type image
  - Handle base dependency (build base first if building dev/osint)

- [x] 11. Update `createAndStartContainer` for on-demand builds
  - Check if project's image exists
  - If missing, build it (and base if needed)
  - Then start container

- [x] 12. Update `image_rebuild` MCP tool
  - Add optional `container_type` parameter
  - If project_id provided, rebuild that project's type
  - If container_type provided, rebuild that specific type

- [x] 13. Add container types to `project_options` MCP tool
  - Add `container_types` section to response
  - List available types with descriptions
  - Indicate default type ("dev")
  - Coordinate with add-github-account-registry

## Manager Script Updates

- [x] 14. Add `rebuild-images` command to manager.sh
  - Build all container types in dependency order (base first)
  - Option `--type base|dev|osint` to build specific type

- [x] 15. Update `create` command to build images during instance setup
  - Build all container types before starting instance

## Testing

- [x] 16. Add unit tests for container type functions
  - Image existence check
  - Build dependency resolution
  - ImageName from container type

- [x] 17. Add integration test for project creation with container_type parameter

## Documentation

- [x] 18. Update AGENTS.md with container types section
  - Available types and their contents
  - How to specify on project creation

- [x] 19. Create `docs/CONTAINER_TYPES.md`
  - Full documentation of each type's contents
  - Build commands
  - How to add new types in future

- [x] 20. Update docs/INSTANCE_MANAGER.md
  - Document rebuild-images command
  - Document container image building during instance creation
