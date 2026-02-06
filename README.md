# Oubliette

> _"Oubliettes held uploaded copies of the mind... The city remembered every one of its citizens, every minute of their existence."_ — The Quantum Thief

Headless AI agent orchestration with containerized sessions, streaming MCP, and recursive task decomposition.

## Features

- **Containerized Sessions**: Agents execute in isolated containers (Docker or Apple Container)
- **Streaming MCP**: Bidirectional communication with ring buffer event polling and SSE push notifications
- **Recursive Spawning**: Sessions spawn child sessions via reverse socket relay
- **Cron Scheduling**: Recurring tasks with session pinning and execution history
- **Workspace Isolation**: UUID-based workspaces with inherited configuration

## Quick Start

```bash
# Install
curl -fsSL https://raw.githubusercontent.com/HyphaGroup/oubliette/main/install.sh | bash

# Initialize
oubliette init

# Add API keys to ~/.oubliette/config/oubliette.jsonc

# Configure MCP for your AI tool
oubliette mcp --setup claude       # or: claude-code

# Start
oubliette --daemon
```

See [docs/INSTALLATION.md](docs/INSTALLATION.md) for detailed setup.

### From Source

```bash
git clone https://github.com/HyphaGroup/oubliette.git && cd oubliette
cp config/oubliette.jsonc.example config/oubliette.jsonc
# Edit config with your API keys
./build.sh
./bin/oubliette
```

## Configuration

Single `oubliette.jsonc` file. See [docs/CONFIGURATION.md](docs/CONFIGURATION.md) for full reference.

```jsonc
{
  "server": { "address": ":8080" },
  "credentials": {
    "providers": {
      "credentials": {
        "anthropic": {
          "provider": "anthropic",
          "api_key": "sk-ant-xxx"
        }
      },
      "default": "anthropic"
    }
  },
  "models": {
    "models": {
      "opus": {
        "model": "claude-opus-4-6",
        "provider": "anthropic",
        "extraHeaders": { "anthropic-beta": "context-1m-2025-08-07" }
      }
    }
  }
}
```

## MCP Tools

All tools use unified names with an `action` parameter:

| Tool | Actions |
|------|---------|
| `project` | `create`, `list`, `get`, `delete`, `options` |
| `container` | `start`, `stop`, `exec`, `logs` |
| `session` | `spawn`, `message`, `get`, `list`, `end`, `events`, `cleanup` |
| `workspace` | `list`, `delete` |
| `schedule` | `create`, `list`, `get`, `update`, `delete`, `trigger`, `history` |
| `token` | `create`, `list`, `revoke` |

See [docs/MCP_TOOLS.md](docs/MCP_TOOLS.md) for details.

### Example: Spawn Session

```json
// Spawn
{"action": "spawn", "project_id": "...", "prompt": "Create Express API with JWT auth"}
// Poll events
{"action": "events", "session_id": "gogol_...", "since_index": 0}
// Send follow-up
{"action": "message", "project_id": "...", "message": "Use bcrypt for passwords"}
```

## MCP Client Configuration

```json
{
  "mcpServers": {
    "oubliette": {
      "type": "http",
      "url": "http://localhost:8080/mcp",
      "headers": { "Authorization": "Bearer oub_your_token" }
    }
  }
}
```

## Container Runtime

Auto-detects Docker or Apple Container. Override with `CONTAINER_RUNTIME=docker|apple-container`.

| Runtime | Platform | Notes |
|---------|----------|-------|
| Docker | Cross-platform | Production, CI/CD |
| Apple Container | macOS | Native VM, better I/O perf |

## Architecture

```
MCP Client → Oubliette Server → Container (OpenCode agent)
                ↕ socket relay
             Child sessions (recursive)
```

- **Gogol**: An executing agent session in a container
- **Reverse Socket Relay**: Nested sessions communicate via paired UNIX sockets
- **Event Buffer**: Ring buffer (1000 events) with index-based polling + SSE push

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for full details.

## Documentation

| Document | Purpose |
|----------|---------|
| [AGENTS.md](AGENTS.md) | Development guide for AI agents |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | System architecture and design |
| [docs/CONFIGURATION.md](docs/CONFIGURATION.md) | Config files, models, credentials |
| [docs/PATTERNS.md](docs/PATTERNS.md) | Design patterns |
| [docs/MCP_TOOLS.md](docs/MCP_TOOLS.md) | Tool development and caller relay |
| [docs/TESTING.md](docs/TESTING.md) | Testing strategy |
| [docs/INSTALLATION.md](docs/INSTALLATION.md) | Installation and setup |
| [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) | Production deployment |
| [docs/OPERATIONS.md](docs/OPERATIONS.md) | Runbook and troubleshooting |

## Project Structure

```
oubliette/
├── cmd/server/              # MCP server + CLI (init, token, mcp, upgrade)
├── cmd/oubliette-client/    # In-container MCP proxy
├── cmd/oubliette-relay/     # Socket relay for nested sessions
├── internal/
│   ├── agent/opencode/      # OpenCode runtime
│   ├── container/           # Docker + Apple Container
│   ├── mcp/                 # MCP handlers
│   ├── project/             # Project + workspace CRUD
│   ├── session/             # Session lifecycle + event buffer
│   └── schedule/            # Cron scheduling
├── containers/              # Dockerfile definitions (base, dev)
├── test/                    # Integration tests
└── config/                  # Example configuration
```

## License

MIT
