# Change: Add SSE Event Streaming for Session Events

## Why

Currently, MCP clients (like Ant) must poll `session_events` to receive updates from running Factory Droid sessions. This creates latency and unnecessary network traffic. The MCP Go SDK supports server-initiated notifications via SSE (Server-Sent Events), enabling real-time event push without polling.

Key pain points solved:
- **Latency**: Events arrive immediately instead of on next poll interval
- **UX**: Clients can show "thinking/working" indicators in real-time (e.g., Signal typing status)
- **Efficiency**: No wasted requests when nothing has changed
- **Resilience**: SDK's EventStore provides automatic stream resumption on disconnect

## What Changes

- Enable `EventStore` on `StreamableHTTPHandler` for SSE stream resumption
- Capture MCP `ServerSession` when handling `session_message` requests
- Push Factory Droid events to clients via `ServerSession.NotifyProgress()` as they arrive
- Add session-to-ServerSession mapping for event routing
- **BREAKING**: None - `session_events` polling continues to work unchanged

## Impact

- **Affected specs**: New capability `session-streaming`
- **Affected code**:
  - `internal/mcp/server.go` - Enable EventStore, track ServerSessions
  - `internal/session/active.go` - Add ServerSession reference for push notifications
  - `internal/mcp/handlers_session.go` - Capture ServerSession from request context
