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
- `~/.oubliette/config/` - Configuration files
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

This adds Oubliette as an MCP server to your tool's configuration.

## Add API Keys

Edit `~/.oubliette/config/credentials.json` to add your API keys:

```json
{
  "factory": {
    "default": "main",
    "keys": {
      "main": "your-factory-api-key"
    }
  },
  "github": {
    "default": "main",
    "tokens": {
      "main": "your-github-token"
    }
  },
  "providers": {
    "default": "anthropic",
    "keys": {
      "anthropic": "your-anthropic-key"
    }
  }
}
```

## Start the Server

```bash
oubliette --config-dir ~/.oubliette/config
```

Or run from a project directory with its own config:

```bash
cd /path/to/project
oubliette
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

### "Configuration error"

Run `oubliette init` to create default configuration files.
