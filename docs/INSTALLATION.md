# Installation

## Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/HyphaGroup/oubliette/main/install.sh | bash
```

## Initial Setup

```bash
# Create config and auth token
oubliette init

# Add API keys
# Edit ~/.oubliette/config/oubliette.jsonc

# Configure MCP for your AI tool
oubliette mcp --setup claude       # or: claude-code

# Start (foreground)
oubliette

# Or background
oubliette --daemon
```

## Add API Keys

Edit `~/.oubliette/config/oubliette.jsonc`:

```jsonc
{
  "credentials": {
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

At least one provider API key is required. See [CONFIGURATION.md](CONFIGURATION.md) for all options.

## Config Location

The server auto-detects config in this order:
1. `--dir` flag
2. `OUBLIETTE_HOME` environment variable
3. `./.oubliette` in current directory
4. `~/.oubliette`

### Project-Specific Instances

```bash
oubliette init --dir /path/to/project
oubliette mcp --setup claude --dir /path/to/project
oubliette --dir /path/to/project --daemon
```

## Container Runtime

Docker or Apple Container required. Auto-detected.

```bash
# Docker
docker ps

# Apple Container (macOS)
brew install apple/apple/container
container system start
```

## Upgrading

```bash
oubliette upgrade --check   # Check for updates
oubliette upgrade           # Install latest
```

## Troubleshooting

| Problem | Solution |
|---------|----------|
| "Command not found" | Add `~/.oubliette/bin` to PATH |
| "No container runtime" | Install Docker or Apple Container |
| "No API credentials" | Add provider key to `oubliette.jsonc` |
| "Token validation failed" | Restart server after `mcp --setup` |
