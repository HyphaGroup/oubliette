# Tasks: Install Script and MCP Setup CLI

## Phase 1: Version Infrastructure

- [x] Add `Version` variable to `cmd/server/main.go` (set via ldflags)
- [x] Add `--version` / `-v` flag to print version
- [x] Update `build.sh` to pass `-ldflags "-X main.Version=dev"` for local builds

## Phase 2: Release Infrastructure

- [x] Update `build.sh` to add `release` mode that builds all platform binaries
  - `./build.sh release v1.0.0` produces:
    - `bin/oubliette-darwin-arm64`
    - `bin/oubliette-darwin-amd64`
    - `bin/oubliette-linux-arm64`
    - `bin/oubliette-linux-amd64`
    - `bin/checksums.txt`
- [x] Create `.github/workflows/release.yml`
  - Trigger on tag push `v*`
  - Build all platform binaries with version from tag
  - Generate checksums
  - Create GitHub Release with artifacts

## Phase 3: Install Script

- [x] Create `install.sh` in repo root
  - Detect OS (`uname -s`) and arch (`uname -m`)
  - Map to release artifact names (darwin-arm64, darwin-amd64, linux-arm64, linux-amd64)
  - Prompt for install location (default: `~/.oubliette/bin`)
  - Download binary and checksums from GitHub Releases
  - Verify SHA256 checksum
  - Make executable and suggest PATH addition
  - Prompt to run `oubliette init`

## Phase 4: Init Command

- [x] Add `init` subcommand to `cmd/server/main.go`
  - Check if `~/.oubliette/` already exists (warn if so)
  - Create directory structure:
    - `~/.oubliette/config/`
    - `~/.oubliette/data/projects/`
    - `~/.oubliette/data/logs/`
    - `~/.oubliette/data/backups/`
  - Create config files with defaults:
    - `server.json` (address: ":8080")
    - `credentials.json` (empty template)
    - `config-defaults.json` (copy from repo)
  - Initialize auth store and create admin token
  - Print token and next steps

## Phase 5: Upgrade Command

- [x] Add `upgrade` subcommand to `cmd/server/main.go`
  - Query `https://api.github.com/repos/HyphaGroup/oubliette/releases/latest`
  - Compare `tag_name` with embedded `Version`
  - `--check` flag: just print status, don't download
  - Download platform-specific binary and checksums
  - Verify checksum
  - Replace running binary (requires write permission to binary location)
  - Print old/new version

## Phase 6: MCP Setup Command

- [x] Add `mcp` subcommand with `--setup <tool>` flag
  - Detect oubliette data dir (from binary location or `~/.oubliette/data`)
  - Load/create auth token using `internal/auth.Store`
  - Tool config paths:
    - `droid`: `~/.factory/mcp.json`
    - `claude`: `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS)
    - `claude-code`: `~/.config/Code/User/globalStorage/anthropic.claude-code/settings.json`
  - Read existing config (or create empty `{}`)
  - Add/update `mcpServers.oubliette` entry
  - Write back config, preserving other entries
  - Print what was changed

## Phase 7: Documentation

- [x] Create `docs/INSTALLATION.md` with curl install, init, mcp setup
- [x] Update `README.md` with new install instructions
- [x] Update `docs/DEPLOYMENT.md` to reference new install flow
