## Context

Oubliette pushes session events to MCP clients via two channels:
1. **Streaming**: `notifications/logging/message` pushed via `ServerSession.Log()` as events occur
2. **Polling**: `session_events` tool returns buffered events from the ring buffer

Both derive from the same `agent.StreamEvent` → `EventBuffer` pipeline, but produce different JSON shapes:

- Streaming uses `eventNotification`: `{session_id, type, text, tool_name, final_response}`
- Polling uses `SessionEventItem`: `{index, type, text, tool_name, role, session_id}`

This divergence forces clients to maintain two parsers for the same semantic data.

## Goals

- Clients can reliably show typing indicators during all phases of agent work
- Intermediate response text is available before completion
- One event shape for both streaming and polling channels
- No increase in notification noise (dedup where needed)

## Non-Goals

- Per-token streaming to MCP clients (too noisy, deltas stay buffer-only)
- MCP `notifications/progress` (requires ProgressToken tied to a request; our events are session-scoped, not request-scoped)
- Configurable log levels for event filtering (adds complexity; whitelist approach is simpler)

## Decisions

### Decision 1: Use `notifications/logging/message` (not `notifications/progress`)

MCP `notifications/progress` requires a `ProgressToken` that ties progress to a specific request. Our session events are session-scoped (they fire continuously across multiple tool calls), so progress tokens don't map cleanly. `logging/message` with structured `Data` is the right fit — it's fire-and-forget, session-scoped, and already works.

**Important caveat**: The MCP go-sdk's `ServerSession.Log()` only sends notifications if the client has called `logging/setLevel`. If the client never sets a level, all log notifications are silently dropped. Clients MUST set logging level to receive events. This is existing behavior — no change needed, but must be documented.

### Decision 2: Add `status` as a new StreamEventType

Create `StreamEventStatus` = `"status"` in `agent.StreamEventType`. This is emitted by `collectEvents` on status transitions (not by the OpenCode SSE parser). The `Text` field carries the status value (`"running"`, `"idle"`, `"completed"`, `"failed"`).

Transitions that fire status events:
- `idle → running` (first work event after idle)
- `running → idle` (completion)
- `running → completed` (executor channel closed)
- `running → failed` (executor error)

Status events are buffered in the ring buffer (available via polling) AND pushed as notifications (available via streaming). This means both polling and streaming clients see the same status transitions.

### Decision 3: Unified `SessionEvent` type

Replace both `eventNotification` (in `session/active.go`) and `SessionEventItem` (in `mcp/handlers_session.go`) with a single `SessionEvent` type.

Fields:
```go
type SessionEvent struct {
    Index     int    `json:"index"`              // Buffer position (0 for streaming)
    Type      string `json:"type"`               // Event type
    Text      string `json:"text,omitempty"`      // Message text / status value
    ToolName  string `json:"tool_name,omitempty"` // Tool name for tool_call/tool_result
    Role      string `json:"role,omitempty"`       // "assistant" for messages
    SessionID string `json:"session_id,omitempty"` // Set for child session events
}
```

- Streaming notifications: `SessionEvent` serialized as `LoggingMessageParams.Data`
- Polling response: `[]SessionEvent` in `SessionEventsResult.Events`

The `FinalResponse` field from `eventNotification` is dropped. Instead, the `completion` event's `Text` field carries the final response text directly (it already does via the `lastAssistantText` attachment logic).

### Decision 4: StreamEventMessage dedup

Only notify on `StreamEventMessage` when `event.Text != lastAssistantText`. OpenCode can emit multiple `message.part.updated` events with the same accumulated text (e.g., after tool results). Without dedup, clients would see redundant notifications.

The dedup variable `lastAssistantText` already exists in `collectEvents` — we just need to check against it before notifying.

### Decision 5: Remove dead StreamEvent fields

`StreamEvent.Raw` (map of raw SSE data) and `StreamEvent.Subtype` (sub-type string) are populated by `parseSSEEvent` but never read by any consumer in `session/` or `mcp/`. They waste memory for every buffered event. Remove both fields and their assignments.

## Risks / Trade-offs

- **Risk**: Adding `message` events to notifications increases notification volume during text generation.
  **Mitigation**: Dedup against `lastAssistantText` ensures only genuinely new text triggers a notification. In practice, consolidated text events fire ~5-10 times per response (not per token).

- **Risk**: Clients that parse `eventNotification` will see a shape change.
  **Mitigation**: The Ant client is the primary consumer and will be updated alongside. The unified shape is a superset (adds `index`, `role`; keeps `type`, `text`, `tool_name`, `session_id`). The removed `final_response` field was redundant with `text` on completion events.

## Open Questions

None — all decisions are grounded in current code and MCP SDK behavior.
