# Add Install Script and MCP Setup CLI

## Problem

Currently there's no easy way to:
1. Install Oubliette from scratch
2. Configure MCP integration with AI tools (Droid, Claude, Claude Code)
3. Set up auth tokens for MCP access

Users must manually build from source, configure files, and create tokens.

## Solution

### 1. Install Script

Hosted at GitHub repo, runnable via:
```bash
curl -fsSL https://raw.githubusercontent.com/HyphaGroup/oubliette/main/install.sh | bash
```

Does:
- Detect OS/arch (darwin/linux, arm64/amd64)
- Download latest release binary from GitHub Releases
- Prompt for install location (default: `~/.oubliette/bin`)
- Create data directory (`~/.oubliette/data/`)
- Add to PATH or suggest adding
- Prompt to run `oubliette init`

### 2. `oubliette init` Command

Interactive setup:
- Create `~/.oubliette/config/` with defaults
- Prompt for Factory API key
- Create data directories (`projects/`, `logs/`, `backups/`)
- Generate initial admin token

### 3. `oubliette mcp --setup <tool>` Command

Configure MCP integration:
```bash
oubliette mcp --setup droid       # ~/.factory/mcp.json
oubliette mcp --setup claude      # ~/Library/Application Support/Claude/claude_desktop_config.json
oubliette mcp --setup claude-code # VS Code settings
```

Does:
- Create auth token if none exists
- Detect tool's config file location
- Add/update oubliette MCP server entry
- Show what was changed

### 4. Release Artifacts

Build and publish to GitHub Releases:
- `oubliette-darwin-arm64`
- `oubliette-darwin-amd64`
- `oubliette-linux-arm64`
- `oubliette-linux-amd64`
- Checksums file

Container images deferred to first `oubliette start`.

## User Flow

```bash
# Install (fast, ~5 seconds)
curl -fsSL https://raw.githubusercontent.com/HyphaGroup/oubliette/main/install.sh | bash

# Setup config (interactive)
oubliette init

# Configure MCP for your AI tool
oubliette mcp --setup droid

# Start server (pulls containers on first run)
oubliette start
```

## Out of Scope

- Windows support (future)
- Homebrew formula (future)
- Auto-update mechanism (future)
- Systemd/launchd service installation (future)
