# Capability: Agent Runtime

This spec defines the agent runtime abstraction layer that enables multiple AI backends (Droid, OpenCode, future runtimes) to be used interchangeably.

## ADDED Requirements

### Requirement: No Dead Code in Runtime Packages

Runtime packages SHALL NOT contain:
- Unused types or structs
- Functions that are defined but never called
- Fallback code paths for unsupported modes
- Duplicate types that mirror shared `agent.*` types

#### Scenario: Droid package has no dead code
- **GIVEN** the `internal/agent/droid/` package
- **WHEN** analyzing with `go vet` and manual review
- **THEN** all exported functions are called from outside the package or implement interfaces
- **AND** all types are used
- **AND** no fallback paths exist for output formats not used (`-o stream-json`)

#### Scenario: OpenCode package has no dead code
- **GIVEN** the `internal/agent/opencode/` package
- **WHEN** analyzing with `go vet` and manual review
- **THEN** all exported functions are called from outside the package or implement interfaces
- **AND** all types are used
- **AND** no unused HTTP proxy infrastructure exists

### Requirement: Runtime Package File Structure

Each agent runtime package SHALL follow a consistent file structure organized by responsibility:

| File | Required | Responsibility |
|------|----------|----------------|
| `runtime.go` | Yes | `agent.Runtime` interface implementation |
| `executor.go` | Yes | `agent.StreamingExecutor` interface implementation |
| `protocol.go` | Yes | Communication layer (message formatting, sending, receiving) |
| `events.go` | Yes | Event type constants and parsing to `agent.StreamEvent` |
| `types.go` | Optional | Runtime-specific types not covered by shared types |

Runtime-specific files are allowed when justified:
- Droid: `command.go` (CLI building), `parser.go` (single-turn output)
- OpenCode: `server.go` (server lifecycle)

#### Scenario: Droid package follows structure
- **GIVEN** the `internal/agent/droid/` package
- **WHEN** listing Go source files
- **THEN** the package contains: `runtime.go`, `executor.go`, `protocol.go`, `events.go`, `command.go`, `parser.go`
- **AND** each file has a header comment describing its responsibility

#### Scenario: OpenCode package follows structure
- **GIVEN** the `internal/agent/opencode/` package
- **WHEN** listing Go source files
- **THEN** the package contains: `runtime.go`, `executor.go`, `protocol.go`, `events.go`, `server.go`
- **AND** each file has a header comment describing its responsibility

### Requirement: Protocol Layer Separation

Each runtime SHALL have a `protocol.go` file that encapsulates all communication details:

- Message serialization/deserialization
- Request/response type definitions
- Connection management
- Error handling for protocol-level failures

The executor and runtime SHALL use protocol.go functions rather than implementing communication directly.

#### Scenario: Droid protocol handles JSON-RPC
- **GIVEN** the `internal/agent/droid/protocol.go` file
- **WHEN** examining its contents
- **THEN** it contains JSON-RPC request/response type definitions
- **AND** it contains request builder functions
- **AND** it contains message serialization logic

#### Scenario: OpenCode protocol handles HTTP
- **GIVEN** the `internal/agent/opencode/protocol.go` file
- **WHEN** examining its contents
- **THEN** it contains HTTP client methods
- **AND** it contains request body formatting
- **AND** it contains SSE connection management

### Requirement: Event Layer Separation

Each runtime SHALL have an `events.go` file that handles:

- Event type constants specific to the runtime's protocol
- Parsing raw events into `agent.StreamEvent`
- Any event normalization logic

#### Scenario: Droid events normalized to StreamEvent
- **GIVEN** a raw Droid notification event
- **WHEN** processed by `droid/events.go` functions
- **THEN** it produces an `agent.StreamEvent` with appropriate Type

#### Scenario: OpenCode events normalized to StreamEvent
- **GIVEN** a raw OpenCode SSE event
- **WHEN** processed by `opencode/events.go` functions
- **THEN** it produces an `agent.StreamEvent` with appropriate Type

### Requirement: Agent Package Documentation

The `internal/agent/` directory SHALL contain an `AGENTS.md` file documenting:

1. Runtime interface contracts (`Runtime`, `StreamingExecutor`)
2. File structure requirements for runtime packages
3. Communication pattern comparison between runtimes
4. Guide for adding capabilities to existing runtimes
5. Guide for adding a new runtime

#### Scenario: AGENTS.md exists and is comprehensive
- **GIVEN** the `internal/agent/AGENTS.md` file
- **WHEN** reading its contents
- **THEN** it documents the Runtime interface methods
- **AND** it documents the StreamingExecutor interface methods
- **AND** it explains the file structure convention
- **AND** it includes a section on adding new runtimes

### Requirement: File Header Comments

Each Go file in runtime packages SHALL have a header comment block that includes:

1. Package and file name
2. One-line description of responsibility
3. Key types or functions defined (if applicable)
4. Relationship to other files in the package

#### Scenario: File has proper header comment
- **GIVEN** any Go file in `internal/agent/droid/` or `internal/agent/opencode/`
- **WHEN** reading the first comment block
- **THEN** it describes the file's single responsibility
- **AND** it helps developers understand where to add new code
