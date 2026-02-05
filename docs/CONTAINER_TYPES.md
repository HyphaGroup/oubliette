# Container Types

Oubliette uses a config-driven container system. Container types are defined in `oubliette.jsonc` and images are pulled from ghcr.io (or built locally in dev mode).

## Default Containers

| Type | Image | Description |
|------|-------|-------------|
| `base` | `ghcr.io/hyphagroup/oubliette-base:latest` | Minimal runtime for lightweight tasks |
| `dev` | `ghcr.io/hyphagroup/oubliette-dev:latest` | Full development environment (default) |

## Configuration

Container types are defined in `~/.oubliette/config/oubliette.jsonc`:

```jsonc
{
  "containers": {
    "base": "ghcr.io/hyphagroup/oubliette-base:latest",
    "dev": "ghcr.io/hyphagroup/oubliette-dev:latest"
  }
}
```

You can add custom container types:

```jsonc
{
  "containers": {
    "base": "ghcr.io/hyphagroup/oubliette-base:latest",
    "dev": "ghcr.io/hyphagroup/oubliette-dev:latest",
    "custom": "my-registry.io/my-oubliette-image:v1.0"
  }
}
```

## Image Pull Behavior

1. **On `oubliette init`**: All configured images are pre-pulled
2. **On container start**: If image is missing, it's pulled automatically
3. **On `container_refresh`**: Latest image is pulled and container restarted

## Container Contents

### base

Minimal runtime environment:
- Debian bookworm-slim
- Essential tools: bash, curl, wget, git, jq
- Network: openssh-client
- Factory Droid CLI
- oubliette-relay, oubliette-client

### dev

Full development environment (extends base):
- Node.js via nvm (LTS version)
- Python via uv package manager
- Java via SDKMAN
- GitHub CLI (gh), GitLab CLI (glab)
- Build tools: gcc, g++, make, cmake
- Search tools: ripgrep, fd
- OpenSpec CLI

## Development Mode

Set `OUBLIETTE_DEV=1` to use locally-built images instead of ghcr.io:

```bash
# Build local images
./build.sh

# Run with local images
OUBLIETTE_DEV=1 ./bin/oubliette
```

In dev mode:
- Image names default to `oubliette-<type>:latest` (local)
- Images are NOT pulled - they must exist locally
- Missing images cause errors with instructions to run `./build.sh`

## Creating Custom Images

Custom images must include these components:
- oubliette-client (`/usr/local/bin/oubliette-client`)
- oubliette-relay (`/usr/local/bin/oubliette-relay`)
- container-init.sh (`/usr/local/bin/container-init.sh`)

Easiest approach: extend the base image:

```dockerfile
FROM ghcr.io/hyphagroup/oubliette-base:latest

# Add your tools
RUN apt-get update && apt-get install -y mypackage

USER gogol
WORKDIR /workspace
```

Build and push to your registry:

```bash
docker build -t my-registry.io/my-oubliette:v1.0 .
docker push my-registry.io/my-oubliette:v1.0
```

Then add to your config:

```jsonc
{
  "containers": {
    "base": "ghcr.io/hyphagroup/oubliette-base:latest",
    "dev": "ghcr.io/hyphagroup/oubliette-dev:latest",
    "custom": "my-registry.io/my-oubliette:v1.0"
  }
}
```

## MCP Tools

### container_refresh

Pull latest image and restart container:

```json
{
  "name": "container_refresh",
  "arguments": {
    "project_id": "abc123"
  }
}
```

Or refresh a container type without a project:

```json
{
  "name": "container_refresh",
  "arguments": {
    "container_type": "dev"
  }
}
```

Note: Refreshing a project container fails if there are active sessions.

### project_options

Lists available container types:

```json
{
  "name": "project_options",
  "arguments": {}
}
```

## Architecture

```
ghcr.io/hyphagroup/oubliette-base:latest
├── Debian bookworm-slim
├── Essential tools
├── Factory Droid CLI
└── oubliette-relay, oubliette-client

ghcr.io/hyphagroup/oubliette-dev:latest (extends base)
├── Node.js (nvm)
├── Python (uv)
├── Java (SDKMAN)
└── Development tools
```
