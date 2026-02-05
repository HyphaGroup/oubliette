# Agent Runtime Development Guide

This document provides guidance for developing and extending agent runtimes in Oubliette.

## Overview

Oubliette supports multiple AI agent backends through a pluggable runtime architecture. Each runtime implements two interfaces:

- **`agent.Runtime`** - Manages runtime lifecycle and execution
- **`agent.StreamingExecutor`** - Handles bidirectional streaming sessions

## Runtime Interface

```go
type Runtime interface {
    Initialize(ctx context.Context, config *RuntimeConfig) error
    Execute(ctx context.Context, request *ExecuteRequest) (*ExecuteResponse, error)
    ExecuteStreaming(ctx context.Context, request *ExecuteRequest) (StreamingExecutor, error)
    Ping(ctx context.Context) error
    Close() error
    Name() string
    IsAvailable() bool
}
```

| Method | Purpose |
|--------|---------|
| `Initialize` | Configure the runtime (API keys, defaults) |
| `Execute` | Single-turn execution (blocking) |
| `ExecuteStreaming` | Start bidirectional streaming session |
| `Ping` | Health check |
| `Close` | Release resources |
| `Name` | Runtime identifier ("droid", "opencode") |
| `IsAvailable` | Can the runtime be used? |

## StreamingExecutor Interface

```go
type StreamingExecutor interface {
    SendMessage(message string) error
    Cancel() error
    Events() <-chan *StreamEvent
    Errors() <-chan error
    Done() <-chan struct{}
    Wait() (int, error)
    Close() error
    RuntimeSessionID() string
    IsClosed() bool
}
```

| Method | Purpose |
|--------|---------|
| `SendMessage` | Send user message to session |
| `Cancel` | Request termination |
| `Events` | Channel of normalized events |
| `Errors` | Channel of errors |
| `Done` | Closes when execution finishes |
| `Wait` | Block until complete, return exit code |
| `Close` | Graceful shutdown |
| `RuntimeSessionID` | Backend's session ID |
| `IsClosed` | Check if executor closed |

## File Structure Convention

Each runtime package follows this structure:

```
internal/agent/<runtime>/
├── runtime.go      # agent.Runtime implementation
├── executor.go     # agent.StreamingExecutor implementation
├── protocol.go     # Communication layer (message format, transport)
├── events.go       # Event type constants and parsing (if needed)
└── <runtime-specific files>
```

### Droid Package

```
internal/agent/droid/
├── runtime.go      # Runtime implementation
├── executor.go     # StreamingExecutor with JSON-RPC over stdin/stdout
├── protocol.go     # JSON-RPC types and request builders
├── command.go      # CLI command building (droid exec ...)
└── parser.go       # Single-turn JSON output parsing
```

**Communication**: JSON-RPC 2.0 over stdin/stdout (`-o stream-jsonrpc`)

### OpenCode Package

```
internal/agent/opencode/
├── runtime.go      # Runtime implementation
├── executor.go     # StreamingExecutor with SSE event stream
├── protocol.go     # HTTP client, SSE reader
├── server.go       # Server lifecycle (Start, Stop, health checks)
└── events.go       # SSE event type constants
```

**Communication**: HTTP REST + SSE (server on port 4096)

## Communication Patterns

### Droid (JSON-RPC over stdin/stdout)

```
┌─────────────┐     stdin (JSON-RPC)      ┌─────────────┐
│  Executor   │ ───────────────────────▶  │  droid exec │
│             │                           │             │
│             │ ◀───────────────────────  │  (CLI)      │
└─────────────┘    stdout (JSON-RPC)      └─────────────┘

Request:  {"jsonrpc":"2.0","method":"droid.add_user_message","params":{"text":"..."},"id":"1"}
Response: {"jsonrpc":"2.0","type":"notification","method":"droid.session_notification",...}
```

### OpenCode (HTTP + SSE)

```
┌─────────────┐     POST /session/:id/prompt_async     ┌──────────────┐
│  Executor   │ ─────────────────────────────────────▶ │  opencode    │
│             │                                        │  serve       │
│             │ ◀───────────────────────────────────── │  (HTTP)      │
└─────────────┘     GET /event (SSE stream)            └──────────────┘

Request:  POST with {"parts":[{"type":"text","text":"..."}]}
Events:   data: {"type":"message.part.updated","part":{"type":"text","text":"..."}}
```

## Adding a New Runtime

1. **Create package**: `internal/agent/<name>/`

2. **Implement `runtime.go`**:
   ```go
   type Runtime struct { ... }
   var _ agent.Runtime = (*Runtime)(nil)
   
   func NewRuntime(...) *Runtime { ... }
   func (r *Runtime) Initialize(ctx context.Context, config *agent.RuntimeConfig) error { ... }
   func (r *Runtime) ExecuteStreaming(ctx context.Context, req *agent.ExecuteRequest) (agent.StreamingExecutor, error) { ... }
   // ... other methods
   ```

3. **Implement `executor.go`**:
   ```go
   type StreamingExecutor struct { ... }
   var _ agent.StreamingExecutor = (*StreamingExecutor)(nil)
   
   func (e *StreamingExecutor) SendMessage(message string) error { ... }
   func (e *StreamingExecutor) Events() <-chan *agent.StreamEvent { ... }
   // ... other methods
   ```

4. **Implement `protocol.go`**: Define your communication layer

5. **Register in `factory.go`**:
   ```go
   const RuntimeTypeNewRuntime RuntimeType = "newruntime"
   
   func (f *RuntimeFactory) CreateRuntime(...) (Runtime, error) {
       switch runtimeType {
       case RuntimeTypeNewRuntime:
           return f.createNewRuntime(ctx, config)
       // ...
       }
   }
   ```

6. **Add tests**: `runtime_test.go`, `protocol_test.go`

## Event Normalization

All runtimes must convert their native events to `agent.StreamEvent`:

```go
type StreamEvent struct {
    Type      StreamEventType  // system, message, tool_call, tool_result, completion, error
    Role      string           // user, assistant
    Text      string           // Message text
    ToolName  string           // For tool calls
    ToolID    string           // Tool invocation ID
    FinalText string           // Final response text (completion)
    // ... see types.go for full definition
}
```

Key event types:
- `StreamEventMessage` - User or assistant message
- `StreamEventToolCall` - Tool invocation started
- `StreamEventToolResult` - Tool returned result
- `StreamEventCompletion` - Turn complete
- `StreamEventError` - Error occurred

## Testing

Each runtime should have:

1. **Unit tests** (`*_test.go`):
   - Runtime initialization
   - Request/response formatting
   - Event parsing

2. **Integration tests** (in `test/pkg/suites/`):
   - End-to-end session spawning
   - Multi-turn conversations
   - Tool usage

## Best Practices

1. **No dead code**: Delete unused types, functions, fallback paths
2. **Single responsibility**: One file, one purpose
3. **Normalize events**: Convert to `agent.StreamEvent` consistently
4. **Handle errors**: Return meaningful errors, don't swallow them
5. **Clean shutdown**: Implement graceful `Close()` with proper cleanup
6. **Document**: Add file header comments explaining purpose
