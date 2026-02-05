# Design: SSE Event Streaming

## Context

The MCP Go SDK provides built-in SSE streaming support via `StreamableHTTPHandler`. When a client connects, it can open a long-lived GET request to receive server-initiated notifications. The SDK handles:
- Session management and ID tracking
- SSE event formatting and delivery
- Stream resumption via `Last-Event-ID` header and `EventStore`
- Automatic reconnection with event replay

Currently, Oubliette captures Factory Droid events in an internal `EventBuffer` and exposes them via the `session_events` tool. Clients must poll this endpoint.

## Goals

- Push Factory Droid events to MCP clients in real-time via SSE
- Enable Ant to show typing/working indicators without polling
- Maintain backwards compatibility with `session_events` polling
- Support stream resumption on network interruption

## Non-Goals

- Replacing `session_events` tool (kept for backwards compatibility)
- Custom SSE implementation (use SDK's built-in support)
- WebSocket support (SSE is sufficient for unidirectional server→client)

## Architecture

```
┌─────────────────┐     SSE Stream (GET)     ┌─────────────────┐
│   Ant (Client)  │ ◄────────────────────────│ Oubliette (MCP) │
└─────────────────┘                          └────────┬────────┘
        │                                            │
        │ POST session_message                       │
        │────────────────────────────────────────────►
        │                                            │
        │         {"session_id": "..."}              │
        │◄────────────────────────────────────────────
        │                                            │
        │                               ┌────────────▼────────────┐
        │                               │    ActiveSession        │
        │                               │  + ServerSession ref    │
        │                               └────────────┬────────────┘
        │                                            │
        │                               ┌────────────▼────────────┐
        │                               │   Factory Droid         │
        │                               │   (in container)        │
        │                               └────────────┬────────────┘
        │                                            │
        │                                   event: working
        │◄──────── NotifyProgress(working) ──────────┤
        │                                   event: tool_use
        │◄──────── NotifyProgress(tool) ─────────────┤
        │                                   event: message
        │◄──────── NotifyProgress(message) ──────────┤
```

## Decisions

### Decision 1: Use Log for event delivery

**What**: Use `ServerSession.Log()` to push events as MCP logging notifications with structured JSON in the `Data` field.

**Why**: 
- `Data` field accepts arbitrary JSON (perfect for our event structure)
- Semantically appropriate: we're logging session events
- Standard MCP mechanism, clients already support it
- Level filtering available if needed

**Alternatives considered**:
- `ServerSession.NotifyProgress()`: Designed for "50% complete" style updates, `Message` is just a string
- Custom notification type: Requires client changes, not standard MCP

### Decision 2: Store ServerSession reference in ActiveSession

**What**: Add `MCPSession *mcp.ServerSession` field to `ActiveSession` struct.

**Why**:
- Events arrive asynchronously in `collectEvents()` goroutine
- Need ServerSession reference to call `Log()`
- One ServerSession per MCP client connection

**Alternatives considered**:
- Global session→ServerSession map: More complex, same result
- Pass ServerSession through channels: Adds complexity to event flow

### Decision 3: Enable MemoryEventStore

**What**: Configure `StreamableHTTPOptions{EventStore: mcp.NewMemoryEventStore(nil)}`.

**Why**:
- Enables automatic stream resumption on disconnect
- SDK handles `Last-Event-ID` parsing and event replay
- No external dependencies (Redis, etc.)

**Risks**:
- Memory usage: Default 10MB limit, adjustable via `SetMaxBytes()`
- Old events purged: Acceptable for real-time streaming use case

### Decision 4: Map MCP sessions to Oubliette sessions

**What**: Track which MCP ServerSession is interested in which Oubliette session.

**Why**:
- Multiple MCP clients could connect
- Need to route events to the right client
- Client may reconnect with new ServerSession

**Implementation**:
- Store ServerSession in ActiveSession when `session_message` creates/finds session
- Update ServerSession if client reconnects
- Clear on session completion

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| Client doesn't open SSE stream | `session_events` polling still works |
| Memory pressure from EventStore | Default 10MB limit, monitor metrics |
| ServerSession closed mid-stream | Check session validity before push |
| Multiple clients for same session | Last client wins (single ServerSession ref) |

## Migration Plan

1. Enable EventStore on StreamableHTTPHandler (no behavior change)
2. Add ServerSession field to ActiveSession (backwards compatible)
3. Capture ServerSession in session_message handler
4. Push events via NotifyProgress in collectEvents goroutine
5. Monitor metrics, adjust EventStore size if needed

No breaking changes. Clients continue to work without modification. SSE benefits are automatic for clients that open the GET stream.

## Open Questions

- Should we support multiple MCP clients watching the same session? (Current: last client wins)
- Should NotifyProgress include structured event data or text summary? (Proposed: structured JSON)
