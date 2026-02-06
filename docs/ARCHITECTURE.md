# Architecture

## Overview

Oubliette is a containerized agent execution system. Agents (OpenCode) run inside containers with MCP tool access, bidirectional streaming, and recursive session spawning via a socket relay.

## Three-Layer Architecture

```
┌──────────────────────────────────────────┐
│ MCP Layer (Protocol)                      │
│  - Unified tool handlers (action param)   │
│  - Auth context from Bearer tokens        │
│  - ActiveSessionManager for streaming     │
│  - SSE push notifications via ServerSession│
└────────────────┬─────────────────────────┘
                 ↓
┌──────────────────────────────────────────┐
│ Manager Layer (Business Logic)            │
│  - ProjectManager: CRUD, limits           │
│  - SessionManager: Lifecycle, streaming   │
│  - ScheduleStore: Cron, session pinning   │
└────────────────┬─────────────────────────┘
                 ↓
┌──────────────────────────────────────────┐
│ Resource Layer                            │
│  - Container Runtime (Docker/Apple)       │
│  - Filesystem (projects/workspaces/)      │
│  - OpenCode agent (HTTP+SSE on port 4096) │
│  - SQLite (auth.db, schedules.db)         │
└──────────────────────────────────────────┘
```

## Container Runtime

Both Docker and Apple Container are supported through a unified `container.Runtime` interface. Auto-detection prefers Apple Container on macOS ARM64.

See `internal/container/runtime.go` for the interface.

## Agent Runtime

OpenCode is the sole agent runtime. The server initializes a single `opencode.NewRuntime(containerRuntime)` — no factory, no dispatch.

```
Executor ──POST /session/:id/prompt_async──▶ OpenCode (port 4096)
         ◀──GET /event (SSE stream)─────────
```

See [internal/agent/AGENTS.md](../internal/agent/AGENTS.md) for interface details.

## Reverse Socket Relay

Nested sessions communicate via a reverse socket architecture:

```
┌─── HOST ──────────────────────────────────────────────────┐
│  Oubliette Server ◄── published socket ── Container       │
│  (SocketHandler)                          ┌────────────┐  │
│                                           │ Agent      │  │
│                                           │   ↕ stdio  │  │
│                                           │ oub-client │  │
│                                           │   ↕ socket │  │
│                                           │ oub-relay  │  │
│                                           └────────────┘  │
└───────────────────────────────────────────────────────────┘
```

1. Container starts with `oubliette-relay` listening on `/mcp/relay.sock`
2. Session spawns, SocketHandler connects as "upstream"
3. Agent starts, `oubliette-client` connects as "downstream"
4. Relay pairs connections for bidirectional MCP
5. Child can recursively spawn grandchildren

Socket methods: `session_message`, `session_events`, `caller_tool`, `oubliette_tools`, `oubliette_call_tool`

## Session Lifecycle

```
Created → Running ⇄ Idle → Completed/Failed
```

- **Running**: Actively processing a prompt
- **Idle**: Turn complete, waiting for next message
- **Completed**: Executor exited or session ended
- Sessions auto-resume by default when spawning for a project

## Streaming Events

Events flow from OpenCode SSE → `parseSSEEvent` (noise filtered) → `EventBuffer` (ring buffer) → optional SSE push notification to MCP client.

**Event types**: `system`, `message`, `delta`, `tool_call`, `tool_result`, `completion`, `error`

**Notification filtering**: Only `completion`, `tool_call`, `tool_result`, and `error` events are pushed as MCP `notifications/message`. Deltas, message updates, and system metadata are buffered for polling only.

**Polling pattern**:
```
session events {since_index: -1}  → all events, returns last_index
session events {since_index: 42}  → events after 42
```

**Event buffer**: Ring buffer (1000 events). Clients that fall behind lose old events.

## Workspace Architecture

```
projects/<project-id>/
├── metadata.json       # Project settings, recursion limits
├── opencode.json       # Generated OpenCode config
└── workspaces/
    └── <uuid>/         # Isolated workspace directory
```

- **Default workspace**: Created with each project
- **User workspaces**: Created on-demand via `create_workspace=true`
- **Config inheritance**: Project `opencode.json` generated from `oubliette.jsonc` credentials + model config
- **Workspace isolation mode**: Optional — restricts agent to workspace directory only

## Schedule Pinning

Each schedule target maintains a pinned `session_id` that persists across runs. On execution:
1. Find active session → resume with new message
2. No active session → spawn new, pin the session ID
3. Failed resume → spawn new, update pinned ID

Execution history stored in `schedule_executions` table with status, output, duration.

## Config Precedence

```
--dir flag → OUBLIETTE_HOME env → ./.oubliette → ~/.oubliette
```

## Concurrent Sessions

Multiple sessions can run in the same workspace. Default limit: 10 active sessions per project.
