# Oubliette Architecture

This document describes the core architectural concepts of Oubliette.

## Overview

Oubliette is a containerized autonomous agent execution system with recursive task decomposition. It provides isolated execution environments for AI agents (Factory Droid, OpenCode) with support for bidirectional streaming and session management.

## Three-Layer Architecture

```
┌─────────────────────────────────────────┐
│ MCP Layer (Protocol Integration)        │
│  - Tool handlers                         │
│  - Context extraction from headers       │
│  - Request routing (prime vs child)      │
│  - ActiveSessionManager for streaming    │
│  - Workspace resolution                  │
└────────────────┬────────────────────────┘
                 ↓
┌─────────────────────────────────────────┐
│ Manager Layer (Business Logic)          │
│  - ProjectManager: CRUD, limits         │
│  - SessionManager: Lifecycle, hierarchy  │
│  - DockerManager: Container operations  │
│  - DroidManager: Agent execution        │
└────────────────┬────────────────────────┘
                 ↓
┌─────────────────────────────────────────┐
│ Resource Layer                           │
│  - Container Runtime (Docker/Apple)      │
│  - Filesystem (projects/workspaces/)     │
│  - Agent runtime CLI (droid/opencode)    │
└─────────────────────────────────────────┘
```

## Container Runtime Support

Oubliette has **equal first-class support for both Docker and Apple Container**:

| Runtime | Platform | Description |
|---------|----------|-------------|
| Docker | Cross-platform | Production-ready, widely supported |
| Apple Container | macOS ARM64 | Better performance via Virtualization.framework |

**Auto-detection**: Prefers Apple Container on macOS ARM64, falls back to Docker.

Both runtimes provide identical functionality through the unified `container.Runtime` interface:
- Container lifecycle (create, start, stop, remove)
- Interactive execution with stdin/stdout/stderr
- Volume mounts and environment variables
- Image building from Dockerfile

See: `internal/container/runtime.go`

## Container-Host Communication

Agents running inside containers need to communicate with the Oubliette server on the host for recursive session spawning, caller tool relay, and access to Oubliette tools. This is achieved through a socket relay architecture.

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                          HOST                                    │
│  ┌──────────────────┐     ┌───────────────────────────────────┐ │
│  │  Oubliette       │     │  /tmp/oubliette-sockets/proj/     │ │
│  │  Server          │◄────┤  relay.sock (host side)           │ │
│  │  (SocketHandler) │     │                                   │ │
│  └──────────────────┘     └───────────────────────────────────┘ │
│                                      ▲                          │
│                                      │ published socket         │
└──────────────────────────────────────┼──────────────────────────┘
                                       │
┌──────────────────────────────────────┼──────────────────────────┐
│                       CONTAINER      │                          │
│  ┌──────────────────┐               ▼                          │
│  │  Agent           │     ┌───────────────────────────────────┐ │
│  │  (Droid/OpenCode)│     │  /mcp/relay.sock (container side) │ │
│  └────────┬─────────┘     └─────────────┬─────────────────────┘ │
│           │ stdio                       │                       │
│           ▼                             │                       │
│  ┌──────────────────┐                   │                       │
│  │  oubliette-client│───────────────────┘                       │
│  │  (MCP server)    │  connects as "downstream"                 │
│  └──────────────────┘                                           │
│                       ┌───────────────────────────────────────┐ │
│                       │  oubliette-relay                      │ │
│                       │  - Listens on /mcp/relay.sock         │ │
│                       │  - Pairs upstream/downstream conns    │ │
│                       │  - Pipes bytes bidirectionally        │ │
│                       └───────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### Components

**oubliette-relay** (runs in container, started by container-init.sh)
- Creates `/mcp/relay.sock` unix socket inside the container
- Accepts "DOWNSTREAM" connections from oubliette-client
- Accepts "UPSTREAM" connections from host (via published socket)
- Pairs connections FIFO and pipes bytes bidirectionally
- Protocol: `OUBLIETTE-DOWNSTREAM {project_id}\n` or `OUBLIETTE-UPSTREAM {session_id} {project_id} {depth}\n`

**oubliette-client** (runs in container, MCP server for agents)
- Communicates with agents (Droid/OpenCode) via stdio
- Connects to relay socket as "downstream"
- Registers caller tools received from parent session
- Provides `session_message` tool for recursive session spawning
- Proxies Oubliette tools when `OUBLIETTE_API_KEY` is set

**SocketHandler** (runs on host, `internal/mcp/socket_handler.go`)
- Connects to the published socket as "upstream" when a session starts
- Handles JSON-RPC requests from oubliette-client
- Sends `caller_tools_config` notification with tools from parent caller
- Processes recursive session spawning, caller tool relay, and Oubliette tool calls

### Socket Publishing

The socket is exposed from container to host differently per runtime:

| Runtime | Mechanism | How It Works |
|---------|-----------|--------------|
| Apple Container | `--publish-socket` flag | Native socket forwarding from container to host |
| Docker | Bind mount | Socket directory mounted into container; relay creates socket visible on host |

### Communication Flow

1. Container starts with oubliette-relay listening on `/mcp/relay.sock`
2. Session spawns, SocketHandler connects to published socket as "upstream"
3. Agent starts, oubliette-client connects to relay as "downstream"
4. Relay pairs the connections and pipes bytes bidirectionally
5. oubliette-client sends JSON-RPC requests (e.g., `session_message`, `caller_tool`)
6. SocketHandler processes requests and returns responses

### Supported Operations

| Method | Description |
|--------|-------------|
| `session_message` | Spawn or message a child session (recursive) |
| `session_events` | Poll for child session completion |
| `caller_tool` | Execute a tool on the external caller |
| `oubliette_tools` | Discover available Oubliette tools (requires API key) |
| `oubliette_call_tool` | Execute an Oubliette tool (requires API key) |

See [MCP_TOOLS.md](MCP_TOOLS.md) for detailed documentation on caller tool relay and Oubliette tool exposure.

## Agent Sessions

**Agent Session** = A Factory Droid or OpenCode session executing in an isolated container.

| Term | Description |
|------|-------------|
| Prime Session | Root-level session (no parent) |
| Child Session | Spawned by another session for recursive task decomposition |
| Exploration | Group of related sessions sharing an exploration ID |

**Session Features:**
- Bidirectional streaming via `stream-jsonrpc` protocol
- Automatic resume by default when spawning for a project
- Event buffering for disconnect recovery (1000 events)
- Graceful shutdown with `interrupt_session`

**Session Lifecycle:**
```
Created → Running → Idle → Completed/Failed
              ↑       │
              └───────┘ (on new message)
```

## Workspace Architecture

**Workspaces** provide isolated execution environments within a project.

```
projects/<project-id>/
├── metadata.json           # Project settings
├── config.json             # Canonical agent config
├── opencode.json           # OpenCode-specific config (generated)
├── .factory/               # Droid-specific config (generated)
└── workspaces/
    └── <workspace-uuid>/   # Isolated workspace
        ├── .factory/       # Workspace MCP config
        └── (user files)
```

**Key Concepts:**
- **Default Workspace**: Created automatically with each project
- **User Workspaces**: Created on-demand via `session_spawn` with `create_workspace=true`
- **External ID**: Caller-provided identifier (e.g., user UUID from external system)

**Workspace Resolution on Spawn:**

| `workspace_id` | `create_workspace` | Behavior |
|----------------|-------------------|----------|
| omitted | `false` (default) | Uses project's default workspace |
| `"<uuid>"` | `false` | Uses specified workspace (error if not found) |
| `"<uuid>"` | `true` | Creates workspace if missing, then uses it |
| omitted | `true` | Generates new UUID, creates workspace |

**MCP Config Hierarchy (Three Levels):**
```
template/.factory/           → Shipped with oubliette
    ↓ (copied on project_create)
projects/<id>/.factory/      → Project template
    ↓ (copied on workspace_create)
workspaces/<uuid>/.factory/  → Workspace-specific (user tokens merged here)
```

**Workspace Isolation Mode** (optional):
Projects can enable `workspace_isolation: true` to restrict agent access:
- **When disabled** (default): Full project directory mounted at `/workspace`
- **When enabled**: Only workspaces directory mounted, agents cannot access project root

## Bidirectional Streaming

All sessions use the `stream-jsonrpc` protocol for real-time streaming:

```
┌──────────────────────────────────────────────────────────────┐
│                    MCP Server                                 │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │              ActiveSessionManager                        │ │
│  │  - Tracks running streaming sessions                     │ │
│  │  - Per-project limits (default: 10 sessions)            │ │
│  │  - Idle timeout cleanup (default: 30 min)               │ │
│  └─────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────┐
│                    ActiveSession                              │
│  ┌────────────────────┐    ┌─────────────────────────────┐  │
│  │  StreamingExecutor │    │      EventBuffer            │  │
│  │  - SendMessage()   │    │  - Ring buffer (1000 events)│  │
│  │  - Events() chan   │    │  - Index-based resumption   │  │
│  │  - Close()         │    │  - After(index) retrieval   │  │
│  └────────────────────┘    └─────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

**Configuration Defaults:**
```go
DefaultEventBufferSize    = 1000      // Events kept for resumption
DefaultSessionIdleTimeout = 30 * time.Minute
DefaultMaxActiveSessions  = 10        // Per project
```

## Recursive Depth Tracking

Oubliette automatically tracks recursion depth via MCP headers—no manual plumbing required:

```
Parent Session        Child Session
     ↓                     ↓
Generate MCP config → X-Oubliette-Depth: 1
     ↓                     ↓
Child reads headers ← Depth incremented automatically
```

**Shared Workspace Map-Reduce:**
- Children write to `/workspace/.rlm-context/<session_id>_<descriptor>.json`
- Parents aggregate from `.rlm-context/`
- Atomic writes via temp file + rename

## Agent Runtime Abstraction

Oubliette uses a pluggable agent runtime architecture:

| Runtime | Status | Description |
|---------|--------|-------------|
| `droid` | Active | Factory AI Droid CLI |
| `opencode` | Active | OpenCode local models |
| `auto` | Default | Auto-detect based on config |

All runtimes implement `internal/agent.Runtime`:

```go
type Runtime interface {
    Initialize(ctx context.Context, config *RuntimeConfig) error
    Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error)
    ExecuteStreaming(ctx context.Context, req *ExecuteRequest) (StreamingExecutor, error)
    Name() string
    IsAvailable() bool
    Close() error
}
```

## Concurrent Sessions

Multiple sessions can run in the same workspace simultaneously. This is intentional for parallel workflows.

**Session Isolation:**
- Factory Droid stores session state in `~/.factory/sessions/<path-encoded-cwd>/`
- Each session gets its own UUID-named files
- No file conflicts for droid internals even when sharing workspace

**What's Shared:**
- Project source files in the workspace
- Git repository state
- Build artifacts and caches

**Limits:**
- Default: 10 active sessions per project
- No per-workspace limit (multiple sessions per workspace allowed)
