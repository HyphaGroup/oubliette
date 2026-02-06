# Agent Runtime

Oubliette uses OpenCode as its sole agent runtime.

## Interface

```go
type Runtime interface {
    ExecuteStreaming(ctx context.Context, req *ExecuteRequest) (StreamingExecutor, error)
    Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error)
    Ping(ctx context.Context) error
    Close() error
}
```

## StreamingExecutor

```go
type StreamingExecutor interface {
    SendMessage(message string) error
    Cancel() error              // POST /session/:id/abort
    Events() <-chan *StreamEvent
    Errors() <-chan error
    Done() <-chan struct{}
    Wait() (int, error)
    Close() error
    RuntimeSessionID() string
    IsClosed() bool
}
```

## Event Types

Events are normalized into `StreamEvent` with these types:

| Type | Description |
|------|-------------|
| `system` | Step boundaries, status changes |
| `message` | Consolidated assistant/user text |
| `delta` | Streaming token chunks |
| `tool_call` | Tool invocation with parameters |
| `tool_result` | Tool result or error |
| `completion` | Turn complete (includes final response text) |
| `error` | Error event |

Noise is filtered at the source (`parseSSEEvent`): `message.updated`, `server.connected`, `server.heartbeat`, and duplicate `session.idle` events are dropped before entering the event channel.

## Reasoning

Reasoning level is passed per-message as OpenCode's `variant` parameter. OpenCode handles provider-specific translation natively.

## Package Layout

```
internal/agent/
├── runtime.go          # Runtime + StreamingExecutor interfaces
├── types.go            # StreamEvent, ExecuteRequest
├── executor.go         # Executor helpers
└── opencode/
    ├── runtime.go      # Runtime impl (server lifecycle per container)
    ├── executor.go     # StreamingExecutor with SSE events
    ├── protocol.go     # HTTP client (prompt_async, abort, events SSE)
    ├── server.go       # Server lifecycle (start, stop, health)
    └── events.go       # SSE event type constants
```
