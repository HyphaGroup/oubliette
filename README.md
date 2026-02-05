# Oubliette

> _"Oubliettes held uploaded copies of the mind... The city remembered every one of its citizens, every minute of their existence."_ — The Quantum Thief

Headless agentic coding tool orchestration system with session management and recursive task decomposition. Create persistent containerized environments where coding runtimes can execute tasks autonomously with full MCP tool access at all recursion levels safely within containers.

## Features

- **🤖 Multi-Runtime Agents**: Supports **Factory Droid** and **OpenCode** runtimes with unified configuration
  - Factory Droid: Factory AI's managed agent runtime
  - OpenCode: Open-source local model execution
- **🐳 Dual Container Support**: Works with both **Docker** and **Apple Container** (auto-detected)
  - Docker: Cross-platform, production-ready
  - Apple Container: macOS-native, better performance via Virtualization.framework
- **🤖 Autonomous Sessions**: Headless agent execution with streaming multi-turn conversations
- **🔄 Recursive Task Decomposition**: Sessions spawn child sessions via reverse socket relay architecture
- **💬 Interactive Messaging**: Send real-time messages to active sessions with bidirectional communication
- **📊 Session Hierarchy**: Automatic depth tracking across parent → child → grandchild chains
- **🔧 Configurable Limits**: Per-project recursion depth limits enforced automatically
- **📁 Workspace Isolation**: UUID-based workspaces with inherited runtime configuration
- **🎯 Streaming Events**: Ring buffer with index-based event polling for live session monitoring
- **⏰ Cron Scheduling**: Schedule recurring agent tasks with standard cron expressions and overlap handling

## Quick Start

### Prerequisites

- Docker or Apple Container (auto-detected, see [Container Runtime](#container-runtime))
- Factory API key (or provider API key for OpenCode runtime)

### Install

```bash
# One-line install
curl -fsSL https://raw.githubusercontent.com/HyphaGroup/oubliette/main/install.sh | bash

# Initialize (creates config and auth token)
oubliette init

# Configure MCP for your AI tool
oubliette mcp --setup droid    # or: claude, claude-code

# Add your API keys
# Edit ~/.oubliette/config/oubliette.jsonc

# Start the server
oubliette --config-dir ~/.oubliette/config
```

See [docs/INSTALLATION.md](docs/INSTALLATION.md) for detailed setup instructions.

### Development Setup

For development from source:

```bash
# 1. Clone repository
git clone https://github.com/HyphaGroup/oubliette.git
cd oubliette

# 2. Install dependencies
# - Go 1.24+ (https://go.dev/dl/)
# - Docker Desktop or Apple Container (see Container Runtime below)

# 3. Configure
cp config/oubliette.jsonc.example config/oubliette.jsonc
# Edit config/oubliette.jsonc with your API keys

# 4. Build (creates agent container image + server binary)
./build.sh

# 5. Run
./bin/oubliette
```

### Configuration

Configuration uses a single `config/oubliette.jsonc` file:

```bash
cp config/oubliette.jsonc.example config/oubliette.jsonc
# Edit with your API keys and settings
```

**Example configuration:**
```jsonc
{
  "server": {
    "address": ":8080",
    "agent_runtime": "auto"  // "auto", "droid", or "opencode"
  },

  "credentials": {
    "factory": {
      "credentials": {
        "default": {
          "api_key": "fk-your_key_here",
          "description": "Primary Factory account"
        }
      },
      "default": "default"
    },
    "github": {
      "credentials": {
        "default": {
          "token": "github_pat_xxx",
          "description": "Default GitHub account"
        }
      },
      "default": "default"
    },
    "providers": {
      "credentials": {
        "anthropic-main": {
          "provider": "anthropic",
          "api_key": "sk-ant-xxx",
          "description": "For OpenCode runtime"
        }
      },
      "default": "anthropic-main"
    }
  },

  "defaults": {
    "limits": {
      "max_recursion_depth": 3,
      "max_agents_per_session": 50,
      "max_cost_usd": 10.00
    },
    "agent": {
      "runtime": "opencode",   // "opencode" or "droid"
      "model": "sonnet",       // Model alias from models section
      "autonomy": "off",       // off, low, medium, high
      "reasoning": "medium"    // off, low, medium, high
    },
    "container": {
      "type": "dev"
    }
  },

  "models": {
    "models": {
      "sonnet": {
        "model": "claude-sonnet-4-5",
        "displayName": "Sonnet 4.5",
        "provider": "anthropic"
      },
      "opus": {
        "model": "claude-opus-4-5",
        "displayName": "Opus 4.5",
        "provider": "anthropic"
      }
    },
    "defaults": {
      "session_model": "sonnet"
    }
  }
}
```

**Agent autonomy levels:**
- `off` - No permission prompts (default for headless execution)
- `low` - Ask for most operations
- `medium` - Allow read/edit, ask for bash
- `high` - Allow most operations, ask for external access

**Reasoning levels:**
- `off` - No extended thinking
- `low` - Minimal thinking budget
- `medium` - Balanced thinking (default)
- `high` - Maximum thinking budget



### Container Runtime

Oubliette has **full support for both Docker and Apple Container** with automatic runtime detection.

#### Docker

- **Platforms**: Linux, macOS, Windows
- **Use cases**: Production deployments, CI/CD, cross-platform development
- **Install**: [Docker Engine](https://docs.docker.com/engine/install/) or [Docker Desktop](https://www.docker.com/products/docker-desktop/)

#### Apple Container

- **Platforms**: macOS only (Apple Silicon and Intel)
- **Use cases**: macOS development, faster I/O performance, native VM isolation
- **Advantages**: Uses macOS Virtualization.framework, no daemon overhead, better file mounting performance
- **Install**: 
  ```bash
  brew install apple/apple/container
  container system start
  ```

#### Runtime Selection

**Auto-detection** (default behavior):
- On macOS ARM64: Prefers Apple Container if available, falls back to Docker
- On other platforms: Uses Docker

**Manual override** via environment variable:
```bash
# Auto-detect (recommended)
export CONTAINER_RUNTIME=auto

# Force Docker
export CONTAINER_RUNTIME=docker

# Force Apple Container
export CONTAINER_RUNTIME=apple-container
```

**Verify runtime in use**:
```bash
# Check metrics endpoint
curl http://localhost:8080/metrics | grep oubliette_runtime_info
```

### Start Server

```bash
./bin/oubliette

# Or with hot reload for development:
./dev.sh
```

The server runs as an HTTP MCP service on port 8080.

### Create Auth Token

```bash
# Create admin token for MCP access
./bin/oubliette token create --name "My Token" --scope admin

# Create read-only admin token (can view but not modify)
./bin/oubliette token create --name "Monitor" --scope admin:ro

# Create token scoped to a specific project
./bin/oubliette token create --name "Project Alpha" --scope project:proj_abc123

# Create read-only project-scoped token
./bin/oubliette token create --name "Project Alpha RO" --scope project:proj_abc123:ro

# Save the token - it cannot be retrieved later
```

**Token Scope Formats:**
| Scope | Access |
|-------|--------|
| `admin` | Full access to all tools and projects |
| `admin:ro` | Read-only access to all tools and projects |
| `project:<uuid>` | Full access to one project only |
| `project:<uuid>:ro` | Read-only access to one project only |

### MCP Client Configuration

Configure your MCP client (e.g., Claude Desktop, Factory Droid):

```json
{
  "mcpServers": {
    "oubliette": {
      "type": "http",
      "url": "http://localhost:8080/mcp",
      "headers": {
        "Authorization": "Bearer oub_your_token_here"
      }
    }
  }
}
```

## Core Concepts

**Project**: Containerized environment with persistent volumes and git repository. Uses either Docker or Apple Container based on runtime configuration.

**Workspace**: Isolated execution environment within a project, identified by UUID. Each workspace has its own configuration directory.

**Gogol**: An executing agent session instance. Named after the uploaded mind-states in *The Quantum Thief*, a gogol is a running agent that can autonomously execute tasks, spawn child gogols for recursive decomposition, and communicate via the reverse socket relay. Gogols are either:
  - **Prime gogols**: Started externally via MCP (depth 0)
  - **Child gogols**: Spawned by parent gogols via `session_message` tool (depth N+1)

**Agent Runtimes**: The execution backend powering each gogol:
  - **Factory Droid**: Factory AI's managed agent with streaming support
  - **OpenCode**: Open-source runtime for local/custom model execution
  - **Auto**: Automatically selects based on available credentials

**Streaming Architecture**: Sessions run in streaming mode with bidirectional communication:
  - **Event buffer**: Ring buffer (1000 events) stores all session events
  - **Index-based polling**: Clients poll with `since_index` for event resumption
  - **Interactive messaging**: Send messages to running sessions via `session_message`

**Reverse Socket Relay**: Nested sessions communicate via reverse socket architecture:
  - Parent opens upstream connection to relay before spawning child
  - Child's oubliette-client connects as downstream
  - Relay pairs connections via FIFO for bidirectional MCP communication
  - Each child can recursively spawn grandchildren with full MCP tool access

**Depth Tracking**: Recursion depth enforced automatically:
  - Project sets `max_depth` in metadata.json (default: 3)
  - System tracks depth via session hierarchy
  - Spawn requests exceeding limit are rejected

## MCP Tools

### Project Management

- **project_create** - Create new project with workspace and container (auto-starts)
  - Parameters: `name`, `description`, `github_token`, `remote_url`, `init_git`, `languages`
- **project_list** - List all projects with status
  - Parameters: `limit`, `name_contains`
- **project_get** - Get project details and configuration
  - Parameters: `project_id`
- **project_delete** - Delete project (⚠️ irreversible)
  - Parameters: `project_id`
- **project_changes** - List OpenSpec changes for a project
  - Parameters: `project_id`
  - Returns: JSON from `openspec list --json` with project_id wrapper
- **project_tasks** - Get task details for an OpenSpec change
  - Parameters: `project_id`, `change_id`
  - Returns: JSON from `openspec instructions apply --json` with project_id wrapper

### Container Operations

- **container_start** - Manually start container (rarely needed due to auto-start)
  - Parameters: `project_id`
- **container_stop** - Stop container (preserves state)
  - Parameters: `project_id`
- **container_exec** - Execute command in running container
  - Parameters: `project_id`, `command`, `working_dir`
- **container_logs** - Get container logs
  - Parameters: `project_id`
- **image_rebuild** - Rebuild Docker image for project
  - Parameters: `project_id`, `instructions`

### Session Management

- **session_message** - Send message to a workspace. Finds existing active session or spawns a new one. Returns session_id for event polling.
  - Parameters: `project_id`, `workspace_id`, `message`, `create_workspace`, `source`, `external_id`, `model`, `reasoning_level`, `autonomy_level`, `context`, `append_system_prompt`, `tools_allowed`, `tools_disallowed`, `mode`, `change_id`, `build_all`
  - **Behavior**: Checks for existing active session in workspace, creates new if none found
  - **Context propagation**: `context` map merged into workspace MCP config (e.g., auth tokens)
  - **Session Modes** (`mode` parameter):
    - `interactive` (default): Message sent as-is
    - `plan`: Prepends `/openspec-proposal` to message for planning workflow
    - `build`: Prepends `/openspec-apply <change_id>` for implementation; creates build-mode.json state file
  - **Build Mode Options**:
    - `change_id`: Specific change to implement (required for build mode unless build_all=true)
    - `build_all`: If true, auto-selects first incomplete change and advances through all changes
- **session_get** - Get session status and output. For live streaming sessions, use session_events instead.
  - Parameters: `session_id`
- **session_events** - Get streaming events from an active session with resumption support. Use since_index to resume from a specific event.
  - Parameters: `session_id`, `since_index`, `max_events`, `include_children`
  - Returns: `session_id`, `status`, `last_index`, `events[]`, `completed`, `failed`
  - **include_children**: When true, includes events from child sessions with `session_id` field on each event
- **session_list** - List sessions for project (filter by status)
  - Parameters: `project_id`, `status`
- **session_end** - End session and mark as completed
  - Parameters: `session_id`

### Workspace Management

- **workspace_list** - List all workspaces for a project with metadata (created_at, last_session_at, external_id, source)
  - Parameters: `project_id`
- **workspace_delete** - Delete a workspace and all its data. Cannot delete the default workspace.
  - Parameters: `project_id`, `workspace_id`

### Configuration

- **config_limits** - Get recursion limits and depth information for project or session
  - Parameters: `project_id`, `session_id`

### Schedule Management

- **schedule_create** - Create a new scheduled task with cron expression
  - Parameters: `name`, `cron`, `targets[]` (project_id, workspace_id, message, mode), `enabled`, `overlap_behavior`
  - Returns: schedule_id
- **schedule_list** - List all schedules with optional filtering
  - Parameters: `project_id` (optional), `enabled` (optional)
- **schedule_get** - Get schedule details by ID
  - Parameters: `schedule_id`
- **schedule_update** - Update schedule properties
  - Parameters: `schedule_id`, `name`, `cron`, `enabled`, `targets[]`, `overlap_behavior`
- **schedule_delete** - Delete a schedule
  - Parameters: `schedule_id`
- **schedule_trigger** - Manually trigger a schedule immediately
  - Parameters: `schedule_id`
  - Note: Bypasses cron timing, respects overlap_behavior

### Token Management (Admin Only)

- **token_create** - Create new API token for MCP access. Requires admin scope.
  - Parameters: `name`, `scope` (admin, admin:ro, project:\<uuid\>, project:\<uuid\>:ro)
- **token_list** - List all API tokens. Requires admin scope.
- **token_revoke** - Revoke an API token. Requires admin scope.
  - Parameters: `token_id`

See [Create Auth Token](#create-auth-token) for scope format details.

## Example Workflows

### Simple Task

```json
// 1. Create project (auto-starts container)
project_create({
  "name": "auth-api",
  "remote_url": "https://github.com/org/auth-api.git",
  "init_git": true
})
// Returns: project_id

// 2. Send message to workspace (spawns session)
session_message({
  "project_id": "proj_abc123",
  "workspace_id": "user-uuid-here",
  "message": "Create Express API with JWT auth and tests",
  "create_workspace": true
})
// Returns: session_id

// 3. Poll for events
session_events({
  "session_id": "session_20251113_abc123",
  "since_index": 0
})
// Returns: events[], last_index, completed status

// 4. Continue polling with resumption
session_events({
  "session_id": "session_20251113_abc123",
  "since_index": 15  // Resume from last_index + 1
})
```

### Interactive Messaging

```json
// 1. Start session via workspace message
session_message({
  "project_id": "proj_abc123",
  "workspace_id": "user-uuid",
  "message": "Implement user authentication",
  "create_workspace": true
})
// Returns: session_id

// 2. Poll for progress
session_events({
  "session_id": "session_xyz789",
  "since_index": 0,
  "max_events": 100
})

// 3. Send feedback mid-execution
session_message({
  "project_id": "proj_abc123",
  "workspace_id": "user-uuid",
  "message": "Use bcrypt instead of SHA256 for passwords"
})
// Finds active session in workspace, routes message there

// 4. Continue polling to see adaptation
session_events({
  "session_id": "session_xyz789",
  "since_index": 42
})
```

### Nested Sessions (Child Spawning)

Sessions can spawn child sessions via the `session_message` tool from within their execution:

```
Parent Session (depth 0)
  └─ Calls session_message internally →
       Child Session (depth 1)
         └─ Calls session_message internally →
              Grandchild Session (depth 2)
```

The system automatically:
- Opens upstream relay connection before spawning child
- Pairs child's oubliette-client as downstream
- Tracks depth and enforces project limits
- Provides full MCP tool access at all levels
- Routes messages through reverse socket relay

## Documentation

- **[AGENTS.md](AGENTS.md)** - Development guide for AI agents
- **[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)** - System architecture and design
- **[docs/PATTERNS.md](docs/PATTERNS.md)** - Design patterns (Manager, Handler, Configuration, Locking)
- **[docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)** - Production deployment guide (Docker Compose, Kubernetes, TLS)
- **[docs/OPERATIONS.md](docs/OPERATIONS.md)** - Production runbook and troubleshooting
- **[docs/SECURITY.md](docs/SECURITY.md)** - Security policies and reporting
- **[docs/CONTRIBUTING.md](docs/CONTRIBUTING.md)** - Contribution guidelines
- **[docs/DEVELOPMENT.md](docs/DEVELOPMENT.md)** - Development setup and workflows

## Project Structure

### Source Repository
```
oubliette/
├── build.sh                # One-command build script
├── dev.sh                  # Hot reload development mode
├── containers/             # Container image definitions
│   ├── base/               # Base image with common tools
│   ├── dev/                # Development environment
│   └── osint/              # OSINT-specific tools
├── cmd/
│   ├── server/             # Main MCP server (includes token commands)
│   ├── oubliette-client/   # MCP client for nested gogols
│   └── oubliette-relay/    # Reverse socket relay
├── internal/
│   ├── agent/              # Agent runtime abstraction
│   │   ├── config/         # Unified config translation
│   │   ├── droid/          # Factory Droid implementation
│   │   └── opencode/       # OpenCode implementation
│   ├── auth/               # Token authentication
│   ├── container/          # Container runtime (Docker/Apple Container)
│   ├── mcp/                # MCP server + tool handlers
│   ├── project/            # Project + workspace CRUD
│   ├── session/            # Session management + event buffer
│   └── schedule/           # Cron-based scheduling
├── template/               # Default config copied to new projects
└── docs/                   # Documentation
```

### Runtime Data (~/.oubliette/)
```
~/.oubliette/
├── config/
│   └── oubliette.jsonc     # Server + credentials + defaults
└── data/
    ├── projects/
    │   └── <project-id>/   # UUID-based project directory
    │       ├── metadata.json
    │       ├── config.json     # Canonical agent config
    │       ├── .factory/       # Droid runtime config
    │       ├── opencode.json   # OpenCode runtime config
    │       └── workspaces/
    │           └── <uuid>/     # Isolated workspace
    │               └── .factory/
    ├── logs/               # Server logs
    └── backups/            # Automated backups (if enabled)
```

## Per-Project Configuration

**Recursion Limits** (`~/.oubliette/data/projects/<id>/metadata.json`):
```json
{
  "name": "osint-analysis",
  "description": "OSINT coordination detection",
  "recursion_config": {
    "max_depth": 4,
    "max_agents": 50,
    "max_cost_usd": 25.00
  }
}
```

**Custom MCP Servers**: Edit project's `.factory/mcp-servers.json`

**Custom Dockerfile**: Add `Dockerfile` to project and run `image_rebuild`

## OpenSpec Integration

Oubliette integrates with [OpenSpec](https://github.com/fission-ai/openspec) for spec-driven development workflows.

### Session Modes

Use the `mode` parameter on `session_message` to control agent behavior:

```json
// Plan mode - create a proposal for a new feature
session_message({
  "project_id": "proj_abc",
  "workspace_id": "ws_123",
  "message": "Add user authentication with OAuth2",
  "mode": "plan"
})
// Agent receives: "/openspec-proposal Add user authentication with OAuth2"

// Build mode - implement a specific change
session_message({
  "project_id": "proj_abc",
  "workspace_id": "ws_123",
  "message": "Start building",
  "mode": "build",
  "change_id": "add-oauth2-auth"
})
// Agent receives: "/openspec-apply add-oauth2-auth"

// Build all mode - implement all incomplete changes
session_message({
  "project_id": "proj_abc",
  "workspace_id": "ws_123",
  "message": "Build everything",
  "mode": "build",
  "build_all": true
})
// Auto-selects first incomplete change, advances through all
```

### Build Mode Phases

Build mode uses a stop hook to ensure task completion:

1. **Build Phase**: Agent implements tasks from `openspec/changes/<id>/tasks.md`
   - Stop hook blocks exit while tasks remain incomplete
   - Re-prompts agent with remaining task count
2. **Verify Phase**: Agent runs build/tests
   - Triggered when all tasks marked complete
   - Agent outputs `VERIFIED` when build passes and tests are green
3. **Archive Phase**: Change is archived
   - Runs `openspec archive <change_id>`
   - Creates git commit
   - In `build_all` mode, advances to next change

### Task Reminders

The stop hook checks if `tasks.md` has been updated since build mode started. If the file appears stale (not modified), it sends a reminder to update task completion status.

## Architecture Details

### Reverse Socket Relay Architecture

Nested sessions use a reverse socket relay for MCP communication:

**Components:**
- **oubliette-relay**: UNIX socket relay that pairs upstream/downstream connections via FIFO
- **oubliette-client**: MCP client that connects to relay as downstream
- **SocketHandler**: Opens upstream connections before spawning child sessions

**Flow:**
```
Parent Session (depth 0)
  1. Opens upstream connection to relay socket
  2. Spawns child session with streaming executor
  3. Child's oubliette-client connects as downstream
  4. Relay pairs connections for bidirectional MCP
  5. Child has full MCP tool access via relay
  6. Child can recursively spawn grandchildren (same process)
```

**Key files:**
- `cmd/oubliette-relay/main.go` - Relay server with FIFO pairing
- `cmd/oubliette-client/main.go` - MCP client for nested sessions
- `internal/mcp/socket_handler.go` - Upstream connection management

### Streaming Event Architecture

Sessions use ring buffer with index-based polling:

**Event Buffer (1000 events):**
- Stores all session events (message, tool_call, tool_result, completion)
- Supports resumption via `since_index` parameter
- Events older than buffer capacity are lost (clients must keep up)

**Polling Pattern:**
```go
// Initial poll
session_events({"session_id": "sess_123", "since_index": 0})
// Returns: {events: [...], last_index: 15}

// Resume from last event
session_events({"session_id": "sess_123", "since_index": 16})
// Returns: {events: [...], last_index: 42}
```

**Event Types:**
- `message`: Assistant or user text
- `tool_call`: Tool invocation with parameters
- `tool_result`: Tool result or error
- `completion`: Session finished (success/failure)

### Workspace Isolation

UUID-based workspaces provide isolation within projects:

**Canonical Config Architecture:**
Each project has a single source of truth (`config.json`) that generates runtime-specific configs:
```
projects/<id>/
├── config.json        → Canonical config (single source of truth)
├── .factory/          → Generated Droid runtime config
│   ├── mcp.json
│   └── settings.json
├── opencode.json      → Generated OpenCode runtime config
└── workspaces/
    └── <uuid>/        → Workspace with copied runtime configs
```

**Config Inheritance:**
1. Project `config.json` defines agent settings, MCP servers, and limits
2. Runtime configs (`.factory/`, `opencode.json`) are generated from canonical config
3. Workspaces inherit project config, with context merged at session start

**Context Propagation:**
`session_message` accepts `context` map that gets merged into workspace config:
```json
{
  "context": {
    "mcp_servers": {
      "my-service": {
        "url": "http://my-service:8080/mcp",
        "auth_token": "jwt_token_here"
      }
    }
  }
}
```

This enables dynamic token injection for user-specific MCP servers.

## Development

### Hot Reload (Recommended)

```bash
./dev.sh
```

Air watches for file changes and automatically rebuilds/restarts. MCP clients reconnect automatically.

### Manual Development

```bash
# Run tests
go test ./...

# Build
go build -o oubliette ./cmd/server

# Run
./bin/oubliette

# Run with logging
./bin/oubliette 2>debug.log
```

## Related Projects

- [Factory AI](https://factory.ai) - Factory Droid CLI and API
- [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk) - Official MCP implementation

## License

MIT
