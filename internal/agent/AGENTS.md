# Agent Runtime Development Guide

## Overview

Oubliette uses OpenCode as its agent runtime. The `agent.Runtime` and `agent.StreamingExecutor` interfaces provide a thin abstraction layer.

## Runtime Interface

```go
type Runtime interface {
    ExecuteStreaming(ctx context.Context, request *ExecuteRequest) (StreamingExecutor, error)
    Execute(ctx context.Context, request *ExecuteRequest) (*ExecuteResponse, error)
    Ping(ctx context.Context) error
    Close() error
}
```

| Method | Purpose |
|--------|---------|
| `Execute` | Single-turn execution (blocking) |
| `ExecuteStreaming` | Start bidirectional streaming session |
| `Ping` | Health check |
| `Close` | Release resources |

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

## OpenCode Package

```
internal/agent/opencode/
├── runtime.go      # Runtime implementation (server lifecycle per container)
├── executor.go     # StreamingExecutor with SSE event stream
├── protocol.go     # HTTP client, SSE reader
├── server.go       # Server lifecycle (Start, Stop, health checks)
└── events.go       # SSE event type constants
```

**Communication**: HTTP REST + SSE (server on port 4096)

```
┌─────────────┐     POST /session/:id/prompt_async     ┌──────────────┐
│  Executor   │ ─────────────────────────────────────▶ │  opencode    │
│             │                                        │  serve       │
│             │ ◀───────────────────────────────────── │  (HTTP)      │
└─────────────┘     GET /event (SSE stream)            └──────────────┘
```

## Event Normalization

All events are converted to `agent.StreamEvent`:

```go
type StreamEvent struct {
    Type      StreamEventType  // system, message, tool_call, tool_result, completion, error
    Role      string           // user, assistant
    Text      string           // Message text
    ToolName  string           // For tool calls
    ToolID    string           // Tool invocation ID
    FinalText string           // Final response text (completion)
}
```

## Reasoning via Variant

Reasoning level is passed per-message as OpenCode's `variant` parameter in `prompt_async`. OpenCode's `ProviderTransform.variants()` handles provider-specific translation:
- **Anthropic**: `high` → `{thinking: {type: "enabled", budgetTokens: 16000}}`
- **OpenAI**: `high` → `{reasoningEffort: "high"}`
- **Google**: `high` → `{thinkingLevel: "high"}`

## Session Abort

`Cancel()` calls OpenCode's `POST /session/:id/abort` endpoint.
