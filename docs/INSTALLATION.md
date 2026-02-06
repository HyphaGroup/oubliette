# Installation Guide

## Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/HyphaGroup/oubliette/main/install.sh | bash
```

This script:
1. Detects your OS and architecture
2. Downloads the latest release from GitHub
3. Verifies the checksum
4. Installs to `~/.oubliette/bin/` (or custom location)

## Manual Install

Download the appropriate binary from [GitHub Releases](https://github.com/HyphaGroup/oubliette/releases):

- `oubliette-darwin-arm64` - macOS Apple Silicon
- `oubliette-darwin-amd64` - macOS Intel
- `oubliette-linux-arm64` - Linux ARM64
- `oubliette-linux-amd64` - Linux x86_64

```bash
# Example for macOS Apple Silicon
curl -LO https://github.com/HyphaGroup/oubliette/releases/latest/download/oubliette-darwin-arm64
chmod +x oubliette-darwin-arm64
mv oubliette-darwin-arm64 ~/.oubliette/bin/oubliette
```

## Initial Setup

After installation, initialize Oubliette:

```bash
oubliette init
```

This creates:
- `~/.oubliette/config/` - Configuration files (including `oubliette.jsonc`)
- `~/.oubliette/data/` - Runtime data (projects, logs, backups)
- An admin token for API access

## Configure Your AI Tool

Set up MCP integration with your AI tool:

```bash
# For Factory Droid
oubliette mcp --setup droid

# For Claude Desktop
oubliette mcp --setup claude

# For Claude Code (VS Code)
oubliette mcp --setup claude-code
```

This adds Oubliette as an MCP server to your tool's configuration, creating an auth token automatically.

## Add API Keys

Edit `~/.oubliette/config/oubliette.jsonc` to add your API keys:

```jsonc
{
  "credentials": {
    "factory": {
      "credentials": {
        "default": {
          "api_key": "fk-your-factory-api-key"
        }
      },
      "default": "default"
    },
    "github": {
      "credentials": {
        "default": {
          "token": "ghp_your-github-token"
        }
      },
      "default": "default"
    },
    "providers": {
      "credentials": {
        "anthropic": {
          "provider": "anthropic",
          "api_key": "sk-ant-your-key"
        }
      },
      "default": "anthropic"
    }
  }
}
```

**Note:** Either a Factory API key OR a provider API key (e.g., Anthropic) is required. If no Factory key is provided, the system uses OpenCode runtime with provider keys.

## Start the Server

```bash
# Start in foreground
oubliette

# Start in background (daemon mode)
oubliette --daemon
```

The server auto-detects config location:
1. `--dir` flag if specified
2. `OUBLIETTE_HOME` environment variable
3. `./.oubliette` if present in current directory
4. `~/.oubliette` (default)

### Using --dir for Project-Specific Instances

Run a separate Oubliette instance from a project directory:

```bash
# Initialize in project directory
oubliette init --dir /path/to/project

# Configure MCP (creates .factory/mcp.json in project)
oubliette mcp --setup droid --dir /path/to/project

# Start server with project config
oubliette --dir /path/to/project --daemon
```

## Upgrading

Check for updates:

```bash
oubliette upgrade --check
```

Upgrade to the latest version:

```bash
oubliette upgrade
```

## Requirements

- **Docker** or **Apple Container** (macOS) for running agent workloads
- Container images are pulled automatically on first use

## Verifying Installation

```bash
oubliette --version
```

## Troubleshooting

### "Command not found"

Add `~/.oubliette/bin` to your PATH:

```bash
echo 'export PATH="$HOME/.oubliette/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

### "No container runtime available"

Install Docker Desktop or ensure the Docker daemon is running:

```bash
docker ps
```

On macOS, you can also use Apple Container:

```bash
brew install apple/apple/container
container system start
```

### "Configuration error"

Run `oubliette init` to create default configuration files.

### "Token validation failed"

If you regenerate tokens with `mcp --setup`, restart the server to pick up the new token:

```bash
pkill oubliette
oubliette --daemon
```

### "No API credentials configured"

Add either a Factory API key or a provider API key (Anthropic, OpenAI, etc.) to `oubliette.jsonc`. See [CONFIGURATION.md](CONFIGURATION.md) for details.
