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
**Runtime**: Factory AI Droid / OpenCode

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
# Build
./build.sh

# Development with hot reload
./dev.sh

# Run tests
cd test/cmd && go run . --test              # Integration tests
cd test/cmd && go run . --coverage-report   # Must be 100%
go test ./... -short                        # Unit tests

# Pattern check
./tools/check-patterns.sh
```

## Documentation Map

| Document | Purpose |
|----------|---------|
| [docs/INSTALLATION.md](docs/INSTALLATION.md) | Quick install, init, MCP setup, upgrading |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Sessions, workspaces, container runtime, streaming |
| [docs/PATTERNS.md](docs/PATTERNS.md) | Design patterns (Manager, Handler, Config, Locking) |
| [docs/TESTING.md](docs/TESTING.md) | Testing strategy, coverage requirements |
| [docs/MCP_TOOLS.md](docs/MCP_TOOLS.md) | Tool development, caller relay, container tools |
| [docs/CONFIGURATION.md](docs/CONFIGURATION.md) | Config files, models, environment variables |
| [docs/BACKUPS.md](docs/BACKUPS.md) | Backup automation, configuration, restore procedures |
| [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) | Production deployment guide |
| [docs/OPERATIONS.md](docs/OPERATIONS.md) | Runbook, troubleshooting, monitoring |

## Core Concepts

**Agent Session** = AI agent (Droid/OpenCode) executing in an isolated container.

**Workspace** = Isolated execution environment within a project. Each has its own `.factory/` config.

**Container Runtime** = Docker or Apple Container (auto-detected). Both provide identical functionality.

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for details.

## Code Organization

```
oubliette/
├── cmd/                     # CLI binaries
│   ├── server/              # Main MCP server
│   ├── oubliette-client/    # In-container MCP proxy
│   ├── oubliette-relay/     # Socket relay
│   └── token/               # Token management
├── internal/                # Private packages
│   ├── agent/               # Agent runtime abstraction
│   │   ├── config/          # Unified config translation
│   │   ├── droid/           # Factory Droid implementation
│   │   └── opencode/        # OpenCode implementation
│   ├── container/           # Container runtime abstraction
│   ├── mcp/                 # MCP protocol handlers
│   ├── project/             # Project/workspace management
│   ├── session/             # Session management
│   └── ...
├── test/                    # Integration tests
│   ├── cmd/                 # Test runner
│   └── pkg/suites/          # Test suites
├── docs/                    # Documentation
└── config/                  # Configuration files
```

## Key Patterns

**1. Manager Pattern** - CRUD operations for resources
```go
func (m *Manager) Create(ctx context.Context, req CreateRequest) (*Resource, error)
func (m *Manager) Get(resourceID string) (*Resource, error)
func (m *Manager) List(filter *ListFilter) ([]*Resource, error)
func (m *Manager) Delete(resourceID string) error
```

**2. Handler Pattern** - MCP tool handlers delegate to managers
```go
func (s *Server) handleTool(ctx context.Context, req *mcp.CallToolRequest, params *Params) (*mcp.CallToolResult, any, error) {
    // 1. Validate
    // 2. Delegate to manager
    // 3. Format response
}
```

**3. Error Wrapping** - Always use `%w`
```go
return fmt.Errorf("failed to create project %s: %w", name, err)
```

See [docs/PATTERNS.md](docs/PATTERNS.md) for full pattern documentation.

## Testing Strategy

**Spec-driven development** via OpenSpec:
- **Integration tests** are primary (100% MCP tool coverage required)
- **Unit tests** only for pure logic with complex edge cases
- **Smoke tests** for post-deploy verification

```bash
# Integration tests (primary)
cd test/cmd && go run . --test

# Coverage report (must pass)
cd test/cmd && go run . --coverage-report
```

See [docs/TESTING.md](docs/TESTING.md) for full testing guidelines.

## Development Workflow

### Before Starting
```bash
grep -A 20 "Pattern" docs/PATTERNS.md  # Review patterns
git diff main..HEAD                      # Check what you're changing
```

### Before Committing
```bash
gofmt -w .                              # Format
./tools/check-patterns.sh               # Pattern check
cd test/cmd && go run . --test          # Tests
cd test/cmd && go run . --coverage-report  # Coverage
```

### Commit Format
```
<type>: <description>

Types: feat, fix, docs, refactor, test, chore
```

## Landing the Plane

**When ending a session**, complete ALL steps:

1. **File issues** for remaining work
2. **Run quality gates** (tests, lints, builds)
3. **PUSH TO REMOTE** - Work is NOT complete until pushed:
   ```bash
   git pull --rebase
   git push
   git status  # MUST show "up to date with origin"
   ```

## Getting Help

- **Patterns**: [docs/PATTERNS.md](docs/PATTERNS.md)
- **Architecture**: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- **Testing**: [docs/TESTING.md](docs/TESTING.md)
- **Bug Reports**: Include steps to reproduce, logs, environment

---

**Last Updated**: 2025-01-29
</coding_guidelines>
