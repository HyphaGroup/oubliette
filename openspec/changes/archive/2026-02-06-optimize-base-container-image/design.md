# Design: Optimize Base Container Image

## Context

The base container runs two agent runtimes (OpenCode, Factory Droid) plus our relay/client binaries. Both agents are self-contained binaries installed via curl scripts. The image needs to be as small as possible while keeping both agents functional.

## Decisions

### debian:bookworm-slim as base

Both agents ship glibc binaries. Alpine uses musl. Not worth the compat risk.

### Pre-install ripgrep globally

OpenCode checks `Bun.which("rg")` first; if missing, downloads 14.1.1 from GitHub (5-10s delay). Droid's install script bundles its own to `~/.factory/bin/rg`. Global install at `/usr/bin/rg` ensures both find it immediately.

```dockerfile
RUN ARCH=$(dpkg --print-architecture) && \
    curl -LO "https://github.com/BurntSushi/ripgrep/releases/download/14.1.1/ripgrep_14.1.1-1_${ARCH}.deb" && \
    dpkg -i "ripgrep_14.1.1-1_${ARCH}.deb" && \
    rm "ripgrep_14.1.1-1_${ARCH}.deb"
```

### No sudo

Current Dockerfile uses sudo post-USER for:
1. `sudo chmod +x /usr/local/bin/container-init.sh`
2. `sudo mkdir -p /mcp && sudo chown gogol:gogol /mcp`

Fix: do both as root before `USER gogol`. Use `COPY --chmod=755` for init script.

### Removed packages

- **wget**: curl sufficient
- **gnupg**: not needed at runtime
- **jq**: agents parse JSON internally
- **bash**: already in bookworm-slim
- **sudo**: security risk, not needed

### Multi-stage build

Already the case. Go builder stage produces static relay/client binaries, runtime stage is slim.

### Build-time verification

```dockerfile
RUN opencode --version && droid --version && rg --version
```
