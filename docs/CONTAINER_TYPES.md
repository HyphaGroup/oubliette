# Container Types

Container types are defined in `oubliette.jsonc` and resolved to images from ghcr.io (or locally in dev mode).

## Default Containers

| Type | Image | Description |
|------|-------|-------------|
| `base` | `ghcr.io/hyphagroup/oubliette-base:latest` | Minimal runtime |
| `dev` | `ghcr.io/hyphagroup/oubliette-dev:latest` | Full dev environment (default) |

## Configuration

```jsonc
{
  "containers": {
    "base": "ghcr.io/hyphagroup/oubliette-base:latest",
    "dev": "ghcr.io/hyphagroup/oubliette-dev:latest",
    "custom": "my-registry.io/my-image:v1.0"
  }
}
```

## Container Contents

### base

- Debian bookworm-slim
- curl, git, openssh-client, procps
- OpenCode CLI
- ripgrep 14.1.1 (pre-installed)
- oubliette-relay, oubliette-client

### dev (extends base)

- Node.js via nvm
- Python via uv
- Java via SDKMAN
- GitHub CLI (gh), GitLab CLI (glab)
- Build tools: gcc, g++, make, cmake

## Development Mode

```bash
./build.sh                        # Build local images
OUBLIETTE_DEV=1 ./bin/oubliette   # Use local images
```

## Custom Images

Extend the base image with required components (`oubliette-client`, `oubliette-relay`, `container-init.sh`):

```dockerfile
FROM ghcr.io/hyphagroup/oubliette-base:latest
RUN apt-get update && apt-get install -y mypackage
USER gogol
WORKDIR /workspace
```
