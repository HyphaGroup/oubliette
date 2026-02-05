# Project Context

## Purpose

Oubliette is a containerized autonomous agent execution system with recursive task decomposition. It provides isolated container environments where AI agents (Factory AI Droid) can execute tasks, spawn child agents for subtasks, and coordinate work through a shared workspace model.

**Key Goals:**
- Enable recursive task decomposition with automatic depth tracking
- Provide isolated, reproducible execution environments via Docker or Apple Container
- Support bidirectional streaming for real-time agent interaction
- Maintain session state for disconnect recovery and resumption

## Quick Start

```bash
# Build all binaries
./build.sh

# Development with hot reload
./dev.sh

# Run integration tests
cd test/cmd && go run . --test

# Check MCP tool coverage
cd test/cmd && go run . --coverage-report

# Pattern enforcement
./tools/check-patterns.sh
```

## Tech Stack

- **Language**: Go 1.24
- **Container Runtimes**: Docker, Apple Container (macOS ARM64)
- **Protocol**: MCP (Model Context Protocol) for tool integration
- **Database**: SQLite (for session metadata), file-based storage
- **Configuration**: Viper (YAML + environment variables)
- **Metrics**: Prometheus
- **Tracing**: OpenTelemetry

**Key Dependencies:**
- `github.com/docker/docker` - Docker API client
- `github.com/modelcontextprotocol/go-sdk` - MCP server SDK
- `github.com/spf13/viper` - Configuration management
- `github.com/prometheus/client_golang` - Metrics

## Code Organization

```
oubliette/
├── cmd/server/              # Main entry point
├── internal/                # Private packages
│   ├── container/          # Runtime abstraction (Docker + Apple Container)
│   ├── droid/              # Factory Droid CLI integration
│   ├── project/            # Project and workspace management
│   ├── session/            # Agent session lifecycle
│   ├── mcp/                # MCP protocol handlers
│   └── logger/             # Centralized logging
├── docs/                   # Architecture docs, patterns, specs
├── test/                   # Integration test suite
│   ├── cmd/                # Test runner
│   └── pkg/suites/         # Test cases by category
├── tools/                  # Development tools (pattern checker)
└── template/               # New project template (.factory/ config)
```

## MCP Tools

**Session Management:**
- `session_spawn` - Spawn or resume session (resumes by default)
- `session_message` - Send message to active streaming session
- `session_events` - Retrieve buffered events with `since_index`
- `session_end` - End session gracefully

**Workspace Management:**
- `workspace_list` - List workspaces with metadata
- `workspace_delete` - Delete workspace (not default)

**Project & Container:**
- `project_create`, `project_get`, `project_list`, `project_delete`
- `container_start`, `container_stop`, `container_logs`, `container_exec`

## Project Conventions

### Code Style

- **Formatting**: `gofmt` (standard Go formatting)
- **Naming**: 
  - Managers: `*Manager` suffix (e.g., `ProjectManager`, `SessionManager`)
  - Mutexes: `*Mu` suffix (e.g., `sessionsMu`, `cacheMu`)
  - Interfaces: Define at point of use, not declaration
- **Error handling**: Always wrap with `%w` and context: `fmt.Errorf("context: %w", err)`
- **Context**: First parameter in all manager methods

### Architecture Patterns

**Three-Layer Architecture:**
1. **MCP Layer** - Protocol handlers, context extraction, request routing
2. **Manager Layer** - Business logic (ProjectManager, SessionManager, DroidManager)
3. **Resource Layer** - Container runtime, filesystem, agent CLI

**Required Patterns:**
- **Manager Pattern** - One manager per resource type with CRUD operations
- **Handler Pattern** - MCP handlers delegate to managers, never contain business logic
- **Configuration Hierarchy** - Environment defaults + project overrides via pointer types
- **Per-Entity Locking** - Fine-grained `sync.RWMutex` for concurrent access
- **Ring Buffer** - Event streaming with disconnect recovery (1000 events)

**Critical Philosophy:**
> **NO BACKWARDS COMPATIBILITY. NO LEGACY CODE. NO DEPRECATION PATHS.**
> Rip-and-replace model: remove old code completely, update all callers immediately.

**Anti-Patterns to Avoid:**
- **Critical**: No auth on MCP server, missing input validation
- **Moderate**: Manager >500 lines, `context.Background()` in non-main code, no interfaces
- **Minor**: Deprecated `os.IsNotExist`, magic numbers, custom shell escaping

### Testing Strategy

- **Integration tests only** - Located in `test/pkg/suites/`
- **100% MCP tool coverage** - Every tool must have at least one test
- **TDD workflow** - Issue → Spec (`docs/specs/`) → Draft PR → TDD cycle → Merge
- **Test runner**: `cd test/cmd && go run . --test`
- **Coverage check**: `go run . --coverage-report`

### Git Workflow

- **Branching**: Feature branches off `main`, squash merge
- **Commit format**: `<type>: <description>` (types: feat, fix, docs, refactor, test, chore)
- **PR template**: Pattern adherence checklist required
- **Pre-commit**: Run `./tools/check-patterns.sh` and full test suite

## Domain Context

**Key Terminology:**
- **Gogol** - An autonomous agent session (literary reference to Nikolai Gogol)
- **Prime Session** - Root-level session with no parent
- **Child Session** - Spawned by another session for subtask decomposition
- **Workspace** - Isolated execution environment within a project (UUID-based)
- **Exploration** - Group of related sessions sharing an exploration ID

**Session Lifecycle:**
- Sessions automatically resume by default via `session_spawn`
- Droid session ID captured from init response for proper resumption
- Graceful shutdown sends `interrupt_session` before closing
- Event buffer (ring buffer) enables disconnect recovery

**Workspace Hierarchy:**
```
template/.factory/           → Shipped with oubliette
    ↓ (copied on project_create)
projects/<id>/.factory/      → Project template
    ↓ (copied on workspace_create)
workspaces/<uuid>/.factory/  → User-specific MCP configs
```

**Workspace Resolution on Spawn:**
| `workspace_id` | `create_workspace` | Behavior |
|----------------|-------------------|----------|
| omitted | `false` (default) | Uses project's default workspace |
| `"<uuid>"` | `false` | Uses specified workspace (error if not found) |
| `"<uuid>"` | `true` | Creates workspace if missing, then uses it |
| omitted | `true` | Generates new UUID, creates workspace |

## Important Constraints

**Security (Development Only):**
- No authentication/authorization currently implemented
- Server binds to localhost only - do NOT expose to untrusted networks
- Secrets stored in `.env` (gitignored)

**Scalability Limits:**
- ~10-20 concurrent sessions per project (empirical)
- File-based session storage works up to ~1000 sessions
- One long-lived container per project

**Configuration Limits:**
- Default max recursion depth: 3
- Default max agents per session: 50
- Default max cost: $10 USD
- Event buffer size: 1000 events
- Session idle timeout: 30 minutes

## External Dependencies

**Required Services:**
- **Factory AI API** - Agent runtime (`FACTORY_API_KEY` required)
- **Container Runtime** - Docker daemon or Apple Container (`container` CLI)
- **GitHub** (optional) - For project source cloning (`DEFAULT_GITHUB_TOKEN`)

**AI Models Supported:**
- Claude Sonnet 4.5 (recommended)
- Claude Opus 4.5 (highest capability)
- Claude Haiku 4.5 (fastest)
- GPT-5.1
- Gemini 3 Pro Preview
