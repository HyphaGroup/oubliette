# Change: Enrich session notification pipeline

## Why

Clients (like Ant) have no signal during pure text generation — they get `tool_call`/`tool_result` events during tool use, but silence during long text responses until `completion`. This makes typing indicators unreliable and prevents clients from showing intermediate response text.

Additionally, the streaming notification payload (`eventNotification`) and the polling payload (`SessionEventItem`) have divergent shapes despite being derived from the same underlying `StreamEvent`. This makes client integration harder — consumers must handle two different schemas for the same semantic data.

## What Changes

1. **Status notifications**: Push `status` events on `ActiveStatus` transitions (idle → running, running → idle/completed/failed). Fires once per transition, not per event. Gives clients clean start/stop signals.

2. **StreamEventMessage notifications**: Add consolidated assistant text (`message` type) to `isNotifiableEvent`. These fire when OpenCode flushes accumulated text (less frequent than per-token deltas, more frequent than completion). Dedup against `lastAssistantText` to avoid pushing identical snapshots. This serves as both intermediate text delivery and a natural activity heartbeat.

3. **Unified event shape**: Replace both `eventNotification` and `SessionEventItem` with a single `SessionEvent` type used by both the streaming notification pipeline and the polling `session_events` response. Same fields, same JSON keys, same semantics.

4. **Documentation**: Update `docs/MCP_TOOLS.md` with the full event type taxonomy — when each fires, what fields are populated, and how clients should use them for typing indicators.

5. **Dead code sweep**: Remove `ActiveStatusPaused` (never set), `EventBuffer.Since()` (never called outside tests), `StreamEvent.Raw` field (populated but never consumed), and `StreamEvent.Subtype` field (same).

## Impact

- Affected specs: `session-streaming`
- Affected code: `internal/session/active.go`, `internal/agent/types.go`, `internal/agent/opencode/executor.go`, `internal/mcp/handlers_session.go`, `docs/MCP_TOOLS.md`
- **Not breaking**: Polling clients get richer events (additive fields). Streaming clients get more event types (additive).
