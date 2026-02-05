# Design: Agent Runtime Structure

## Context

Oubliette supports multiple agent runtimes through a pluggable architecture:

- **Droid**: Factory AI's CLI-based runtime (`droid exec`), communicates via stdin/stdout JSON-RPC
- **OpenCode**: Self-hosted runtime (`opencode serve`), communicates via HTTP REST + SSE

Both implement `agent.Runtime` and `agent.StreamingExecutor` interfaces but organize code differently.

### Current State

```
internal/agent/
├── droid/
│   ├── command.go     # CLI command building
│   ├── executor.go    # StreamingExecutor (408 lines) - HAS DEAD FALLBACK CODE
│   ├── jsonrpc.go     # JSON-RPC types (95 lines)
│   ├── parser.go      # Output parsing (40 lines)
│   ├── runtime.go     # Runtime impl (205 lines)
│   └── types.go       # DEAD CODE - unused duplicate types
│
└── opencode/
    ├── events.go      # Event constants (44 lines)
    ├── executor.go    # StreamingExecutor (273 lines)
    ├── runtime.go     # Runtime impl (181 lines)
    └── server.go      # Server + HTTP client (385 lines) - HAS DEAD CODE
```

**Dead Code Found:**

| File | Dead Code | Reason |
|------|-----------|--------|
| `droid/types.go` | Entire file | `droid.StreamEvent`, `droid.ExecuteResponse` duplicate `agent.*` types and are never used |
| `droid/executor.go` | Lines ~363-390 | Fallback for `-o stream-json` format which is never requested |
| `opencode/server.go` | `httpProxy` struct | Created but never used (HTTP done via curl) |
| `opencode/server.go` | `SendMessageStreaming()` | Defined but never called |

**Other Issues:**
1. `opencode/server.go` mixes server lifecycle with HTTP protocol
2. No guidance for contributors on where to add code
3. Hard to compare how runtimes handle the same capability

## Goals

1. **Remove dead code**: Delete all unused types, functions, and fallback paths
2. **Parallel structure**: Same file names for same responsibilities
3. **Single responsibility**: Each file has one clear purpose
4. **Discoverability**: Easy to find where to make changes
5. **Documentation**: Clear guidance for contributors

## Non-Goals

1. Unifying implementation details (they're fundamentally different)
2. Creating abstractions between runtimes
3. Changing external interfaces

## Decisions

### Decision 1: Normalize to responsibility-based file structure

**Target structure for both runtimes:**

| File | Responsibility |
|------|----------------|
| `runtime.go` | `agent.Runtime` interface implementation |
| `executor.go` | `agent.StreamingExecutor` interface implementation |
| `protocol.go` | Communication layer (message sending/receiving) |
| `events.go` | Event type constants and parsing functions |
| `types.go` | Runtime-specific types (if needed) |

**Droid-specific additions:**
- `command.go` - CLI command building (unchanged)
- `parser.go` - Single-turn output parsing (unchanged)

**OpenCode-specific additions:**
- `server.go` - Server lifecycle management (trimmed)

**Rationale**: This aligns with the project's pattern of single-responsibility files while acknowledging that runtimes have different needs.

### Decision 2: Create protocol.go for communication layer

**Droid `protocol.go`** (extracted from jsonrpc.go + executor.go):
- JSON-RPC request/response types
- Request ID generation
- Message serialization/deserialization
- Permission handling responses

**OpenCode `protocol.go`** (extracted from server.go):
- HTTP client methods (doRequest, doRequestRaw)
- Request body formatting
- Response parsing
- SSE connection management

**Rationale**: Separating protocol handling makes it clear where to add new message types or API endpoints.

### Decision 3: Create events.go for event handling

**Both runtimes:**
- Event type constants specific to the runtime
- Event parsing functions
- Event normalization to `agent.StreamEvent`

**Rationale**: Event handling is a key capability that should be easy to compare between runtimes.

### Decision 4: Add internal/agent/AGENTS.md

Document:
- Runtime interface contracts
- File structure guide
- How to add capabilities to a runtime
- How to add a new runtime
- Communication pattern diagrams

**Rationale**: Follows project convention of AGENTS.md files for AI assistant guidance.

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| Churn in imports | One-time change, all internal |
| Merge conflicts | Complete in single PR |
| Test coverage gaps | Run full test suite after each file move |

## Migration Plan

1. Create new files with extracted code
2. Update imports in each package
3. Remove old code from original files
4. Verify tests pass
5. Add AGENTS.md documentation
6. Add file header comments

## Open Questions

None - this is a straightforward refactoring with clear target state.
