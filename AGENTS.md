<coding_guidelines>
<!-- OPENSPEC:START -->
# OpenSpec Instructions

These instructions are for AI assistants working in this project.

Always open `@/openspec/AGENTS.md` when the request:
- Mentions planning or proposals (words like proposal, spec, change, plan)
- Introduces new capabilities, breaking changes, architecture shifts
- Sounds ambiguous and you need the authoritative spec before coding

<!-- OPENSPEC:END -->

# Oubliette Development Guide

**Project**: Containerized autonomous agent execution system
**Runtime**: OpenCode (sole runtime)

## Critical Philosophy

> **NO BACKWARDS COMPATIBILITY. NO LEGACY CODE.**
>
> This codebase operates on a **rip-and-replace** model:
> - Remove old code completely - don't wrap in feature flags
> - Update all callers immediately when changing interfaces
> - Delete dead code the moment it becomes unused

**After EVERY implementation task:**
1. **Hunt for dead code** - Search for unused functions, unreachable branches
2. **Remove it immediately** - Don't leave TODOs, delete it
3. **Build and verify** - `./build.sh`
4. **Run integration tests** - `cd test/cmd && go run . --test`

## Quick Start

```bash
./build.sh                                  # Build
./dev.sh                                    # Hot reload dev
go test ./... -short                        # Unit tests
cd test/cmd && go run . --test              # Integration tests
cd test/cmd && go run . --coverage-report   # Coverage (must be 100%)
```

## Documentation

| Document | Purpose |
|----------|---------|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Sessions, containers, streaming, relay |
| [docs/CONFIGURATION.md](docs/CONFIGURATION.md) | oubliette.jsonc, models, credentials |
| [docs/PATTERNS.md](docs/PATTERNS.md) | Manager, Handler, Locking, Ring Buffer patterns |
| [docs/MCP_TOOLS.md](docs/MCP_TOOLS.md) | Tool development, caller relay, socket protocol |
| [docs/TESTING.md](docs/TESTING.md) | Testing strategy, coverage requirements |
| [docs/INSTALLATION.md](docs/INSTALLATION.md) | Install, init, MCP setup |
| [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) | Production deployment, TLS, systemd |
| [docs/OPERATIONS.md](docs/OPERATIONS.md) | Runbook, troubleshooting, monitoring |
| [docs/BACKUPS.md](docs/BACKUPS.md) | Backup configuration and restore |

## Core Concepts

- **Gogol** = An executing agent session (OpenCode) running in a container
- **Workspace** = Isolated execution environment within a project
- **Container Runtime** = Docker or Apple Container (auto-detected)
- **Schedule** = Cron-based recurring task with session pinning

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for details.

## Code Organization

```
oubliette/
├── cmd/
│   ├── server/              # Main MCP server + CLI commands
│   ├── oubliette-client/    # In-container MCP proxy
│   └── oubliette-relay/     # Socket relay for nested sessions
├── internal/
│   ├── agent/               # Agent runtime abstraction
│   │   ├── config/          # Config translation to opencode.json
│   │   └── opencode/        # OpenCode runtime implementation
│   ├── container/           # Container runtime (Docker/Apple Container)
│   ├── mcp/                 # MCP server + unified tool handlers
│   ├── project/             # Project/workspace management
│   ├── session/             # Session management + event buffer
│   ├── schedule/            # Cron scheduling with session pinning
│   └── config/              # Server configuration (oubliette.jsonc)
├── containers/              # Container image definitions (base, dev)
├── test/                    # Integration tests
└── docs/                    # Documentation
```

## Key Patterns

See [docs/PATTERNS.md](docs/PATTERNS.md) for full details.

- **Manager Pattern**: CRUD via Create/Get/List/Delete
- **Handler Pattern**: MCP handlers validate, delegate to managers, format response
- **Error Wrapping**: Always `fmt.Errorf("context: %w", err)`

## Commit Format

```
<type>: <description>

Types: feat, fix, docs, refactor, test, chore
```

## Before Committing

```bash
gofmt -w .
golangci-lint run --enable gocritic ./cmd/... ./internal/...
go test ./... -short
cd test/cmd && go run . --test
```

## Landing the Plane

1. Run quality gates (tests, lints, builds)
2. **Push to remote** - work is NOT complete until pushed

---

**Last Updated**: 2026-02-06
</coding_guidelines>
