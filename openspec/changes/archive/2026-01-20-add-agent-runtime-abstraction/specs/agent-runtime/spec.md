# Capability: Agent Runtime

Agent runtime abstraction providing a unified interface for AI execution backends (Factory Droid, OpenCode).

## ADDED Requirements

### Requirement: Runtime Interface

The system SHALL provide an `agent.Runtime` interface that abstracts AI execution backends.

The interface SHALL support:
- Initialization with container runtime and project context
- Streaming execution returning a `StreamingExecutor`
- Single-turn execution for simpler use cases
- Health check via `Ping()` method
- Graceful shutdown via `Close()`
- Metadata: `Name()`, `IsAvailable()`

#### Scenario: Runtime initialization succeeds
- **GIVEN** a valid container runtime and project configuration
- **WHEN** `runtime.Initialize(ctx, config)` is called
- **THEN** the runtime is ready for execution

#### Scenario: Runtime initialization fails without credentials
- **GIVEN** Droid runtime selected but `FACTORY_API_KEY` not set
- **WHEN** `runtime.Initialize(ctx, config)` is called
- **THEN** an error is returned indicating missing credentials

### Requirement: Streaming Executor Interface

The system SHALL provide an `agent.StreamingExecutor` interface for bidirectional streaming communication.

The interface SHALL support:
- `SendMessage(message string) error` - Send user messages
- `Cancel() error` - Interrupt execution
- `Events() <-chan *StreamEvent` - Receive normalized events
- `Errors() <-chan error` - Receive errors
- `Done() <-chan struct{}` - Signal completion
- `Wait() (int, error)` - Block until completion
- `Close() error` - Cleanup resources
- `RuntimeSessionID() string` - Get runtime's session ID
- `IsClosed() bool` - Check closure state

#### Scenario: Streaming session lifecycle
- **GIVEN** an initialized runtime and running container
- **WHEN** `ExecuteStreaming(ctx, request)` is called with a prompt
- **THEN** a `StreamingExecutor` is returned
- **AND** `Events()` channel receives message events
- **AND** completion event is received when done

#### Scenario: Session cancellation
- **GIVEN** an active streaming session
- **WHEN** `executor.Cancel()` is called
- **THEN** the runtime receives interrupt signal
- **AND** `Done()` channel closes

### Requirement: Normalized Stream Events

The system SHALL provide a normalized `StreamEvent` type that works across all runtimes.

Event types SHALL include:
- `EventTypeSystem` - System messages
- `EventTypeMessage` - AI assistant messages
- `EventTypeToolCall` - Tool invocations
- `EventTypeToolResult` - Tool outputs
- `EventTypeCompletion` - Session completion
- `EventTypeError` - Error events

#### Scenario: Droid event normalization
- **GIVEN** Droid runtime receiving `create_message` notification
- **WHEN** the event is processed by the executor
- **THEN** a `StreamEvent` with `Type=EventTypeMessage` is emitted
- **AND** `Role="assistant"` and `Text` are populated

#### Scenario: OpenCode event normalization
- **GIVEN** OpenCode runtime receiving `message.part.updated` SSE event
- **WHEN** the event is processed by the executor
- **THEN** a `StreamEvent` with `Type=EventTypeMessage` is emitted
- **AND** `Text` is populated from the part content

### Requirement: Runtime Factory with Auto-Detection

The system SHALL provide a factory function that auto-detects the appropriate runtime.

Detection logic:
- If `config/server.json` has `agent_runtime: "droid"` → use Droid (error if no API key)
- If `config/server.json` has `agent_runtime: "opencode"` → use OpenCode
- If `config/server.json` has `agent_runtime: "auto"` (default):
  - If `config/factory.json` exists with valid API key → use Droid
  - Otherwise → use OpenCode

#### Scenario: Auto-detection prefers Droid with API key
- **GIVEN** `config/server.json` has `agent_runtime: "auto"`
- **AND** `config/factory.json` exists with valid API key
- **WHEN** `agent.NewRuntime(config)` is called
- **THEN** a Droid runtime is returned

#### Scenario: Auto-detection falls back to OpenCode
- **GIVEN** `config/server.json` has `agent_runtime: "auto"`
- **AND** `config/factory.json` does not exist
- **WHEN** `agent.NewRuntime(config)` is called
- **THEN** an OpenCode runtime is returned

#### Scenario: Explicit runtime selection
- **GIVEN** `config/server.json` has `agent_runtime: "opencode"`
- **WHEN** `agent.NewRuntime(config)` is called
- **THEN** an OpenCode runtime is returned regardless of API key presence

### Requirement: Droid Runtime Implementation

The system SHALL provide a Droid implementation of `agent.Runtime`.

The implementation SHALL:
- Execute via CLI with JSON-RPC over stdin/stdout
- Support session continuation via `-s` flag
- Use `--skip-permissions-unsafe` for autonomy
- Capture `droidSessionID` from init response

#### Scenario: Droid session continuation
- **GIVEN** an existing session with `RuntimeSessionID` set
- **WHEN** `ExecuteStreaming()` is called with the session ID
- **THEN** Droid resumes the session using `-s <session_id>` flag

#### Scenario: Droid graceful shutdown
- **GIVEN** an active Droid streaming session
- **WHEN** `executor.Close()` is called
- **THEN** `interrupt_session` JSON-RPC request is sent
- **AND** resources are cleaned up

### Requirement: OpenCode Runtime Implementation

The system SHALL provide an OpenCode implementation of `agent.Runtime`.

The implementation SHALL:
- Manage OpenCode server lifecycle (start with container, stop with container)
- Connect via SDK to `http://127.0.0.1:4096`
- Parse SSE events and normalize to `StreamEvent`
- Use `build` agent by default, `plan` agent for UseSpec mode
- Pre-configure permissive permissions in opencode.json

#### Scenario: OpenCode server lifecycle
- **GIVEN** OpenCode runtime and container starting
- **WHEN** `Initialize()` is called
- **THEN** `opencode serve --port 4096` is started in container
- **AND** health check passes
- **AND** SDK can connect

#### Scenario: OpenCode session continuation
- **GIVEN** an existing session with `RuntimeSessionID` set
- **WHEN** `ExecuteStreaming()` is called with the session ID
- **THEN** OpenCode resumes the session using `sessionID` parameter

#### Scenario: OpenCode agent selection
- **GIVEN** an execution request with `UseSpec=true`
- **WHEN** `ExecuteStreaming()` is called
- **THEN** the `plan` agent is used instead of `build`

### Requirement: Runtime Health Check

The system SHALL support proactive health checking via `Ping()` method.

#### Scenario: Droid health check
- **GIVEN** Droid runtime initialized
- **WHEN** `runtime.Ping(ctx)` is called
- **THEN** nil is returned (no-op, container health determines availability)

#### Scenario: OpenCode health check succeeds
- **GIVEN** OpenCode runtime with server running
- **WHEN** `runtime.Ping(ctx)` is called
- **THEN** nil is returned after successful `/health` response

#### Scenario: OpenCode health check fails
- **GIVEN** OpenCode runtime with crashed server
- **WHEN** `runtime.Ping(ctx)` is called
- **THEN** connection error is returned

### Requirement: Concurrent Session Support

The system SHALL support multiple concurrent sessions per project.

#### Scenario: Concurrent sessions same project
- **GIVEN** OpenCode runtime with running server
- **WHEN** two `ExecuteStreaming()` calls are made for different workspaces
- **THEN** both sessions operate independently
- **AND** events are correctly routed to respective executors

### Requirement: Per-Project Runtime Override

The system SHALL support per-project runtime configuration that overrides the server default.

#### Scenario: Project with explicit runtime
- **GIVEN** server configured with `agent_runtime: "auto"` (selects Droid)
- **AND** project created with `agent_runtime: "opencode"`
- **WHEN** spawning a session for that project
- **THEN** OpenCode runtime is used

#### Scenario: Project using server default
- **GIVEN** server configured with `agent_runtime: "droid"`
- **AND** project created without `agent_runtime` parameter
- **WHEN** spawning a session for that project
- **THEN** Droid runtime is used (server default)

### Requirement: Runtime Options in project_options

The system SHALL include available runtimes in the `project_options` response.

#### Scenario: Get available runtimes
- **GIVEN** both Droid and OpenCode runtimes available
- **WHEN** calling `project_options`
- **THEN** response includes `agent_runtimes` section
- **AND** `agent_runtimes.available` lists `["droid", "opencode"]`
- **AND** `agent_runtimes.default` shows the server's configured default
