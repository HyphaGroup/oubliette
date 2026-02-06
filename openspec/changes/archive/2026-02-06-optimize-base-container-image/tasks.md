# Tasks: Optimize Base Container Image

## 1. Dockerfile Rewrite

- [x] 1.1 Rewrite `containers/base/Dockerfile`
  - apt: ca-certificates, curl, git, locales, openssh-client, procps
  - ripgrep 14.1.1 via arch-aware tarball download (no arm64 .deb available)
  - All root operations (create /mcp, copy init script) before USER switch
  - No sudo, no sudoers config
  - Build-time verification: `opencode --version && droid --version && rg --version`
  - PATH includes agent bin dirs in ENV
- [x] 1.2 Update `containers/base/metadata.json`
- [x] 1.3 Add `gnupg` to `containers/dev/Dockerfile` (was inherited from base, needed for GitHub CLI)

## 2. Build and Verify

- [x] 2.1 Build: `docker build -f containers/base/Dockerfile -t oubliette-base:latest .`
- [x] 2.2 Verify in container:
  - `opencode --version` = 1.1.53
  - `droid --version` = 0.57.5
  - `rg --version` = 14.1.1 (at /usr/local/bin)
  - `git --version` = 2.39.5
  - `whoami` = gogol
  - `which sudo` = not found
  - `which wget` = not found
  - Image size: 735MB (old: 740MB)

## 3. Integration

- [x] 3.1 Spawned session against optimized base image (project container-test)
- [x] 3.2 Agent ran bash commands, created files, searched with ripgrep -- all passed
