# Proposal: Optimize Base Container Image

## Why

The current base container image includes redundant packages. Strip it to the minimum required for OpenCode and Factory Droid to work.

## What Changes

Rewrite `containers/base/Dockerfile`:

- **debian:bookworm-slim** base (~75MB, already includes bash, tar, coreutils)
- **apt packages**: ca-certificates, curl, git, locales, openssh-client, procps
- **Pre-install ripgrep** globally (avoids OpenCode's 5-10s auto-download on first use)
- **No sudo** - all root operations happen before `USER gogol`
- **Build-time verification** - `opencode --version && droid --version && rg --version`

**Removed:** wget, gnupg, sudo, jq, bash (redundant with bookworm-slim)

### Runtime Dependencies

| Dependency | Why | Source |
|------------|-----|--------|
| bash, tar, sha256sum | Shell exec, install scripts, checksums | bookworm-slim built-in |
| git | VCS operations | apt |
| curl | HTTP client, agent install scripts | apt |
| ca-certificates | TLS | apt |
| openssh-client | git SSH auth | apt |
| locales | UTF-8 text handling | apt |
| procps | ps, pkill | apt |
| ripgrep 14.1.1 | Code search (OpenCode + Droid) | .deb from GitHub releases |

## Impact

- `containers/base/Dockerfile` - rewrite
- `containers/base/metadata.json` - updated

## Out of Scope

- Dev/osint containers (separate work)
- Alpine base (musl compat issues with agent binaries)
