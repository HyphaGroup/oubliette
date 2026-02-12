## 1. Add StreamEventStatus type and status event emission

- [ ] 1.1 Add `StreamEventStatus StreamEventType = "status"` to `internal/agent/types.go`
- [ ] 1.2 In `collectEvents` (`internal/session/active.go`), emit a `StreamEvent{Type: StreamEventStatus, Text: "running"}` into the buffer when transitioning idle → running
- [ ] 1.3 Emit `StreamEvent{Type: StreamEventStatus, Text: "idle"}` on completion transition
- [ ] 1.4 Emit `StreamEvent{Type: StreamEventStatus, Text: "failed"}` on executor error
- [ ] 1.5 Add `StreamEventStatus` to `isNotifiableEvent` whitelist

## 2. Add StreamEventMessage to notification pipeline

- [ ] 2.1 Add `StreamEventMessage` to `isNotifiableEvent` whitelist (with `Role == "assistant"` guard)
- [ ] 2.2 Add dedup check: only notify when `event.Text != lastAssistantText` and `event.Text != ""`
- [ ] 2.3 Update `lastAssistantText` tracking after successful notification (not just on receipt)

## 3. Unify event shape

- [ ] 3.1 Create `SessionEvent` type in `internal/session/active.go` with fields: `Index`, `Type`, `Text`, `ToolName`, `Role`, `SessionID`
- [ ] 3.2 Replace `eventNotification` in `NotifyEvent` with `SessionEvent` (set Index=0 for streaming)
- [ ] 3.3 Remove `FinalResponse` field — completion events carry final text in `Text` field directly
- [ ] 3.4 Replace `SessionEventItem` in `handlers_session.go` with `SessionEvent` from session package
- [ ] 3.5 Update `handleSessionEvents` to construct `SessionEvent` instances instead of `SessionEventItem`

## 4. Remove dead code

- [ ] 4.1 Remove `StreamEvent.Raw` field from `internal/agent/types.go` and all assignments in `parseSSEEvent`
- [ ] 4.2 Remove `StreamEvent.Subtype` field from `internal/agent/types.go` and all assignments in `parseSSEEvent`
- [ ] 4.3 Remove `ActiveStatusPaused` constant (comment says "not currently used", no code sets it)
- [ ] 4.4 Remove `EventBuffer.Since()` method (only called from tests, not from production code)
- [ ] 4.5 Update tests that reference removed fields/methods

## 5. Update documentation

- [ ] 5.1 Add "Session Events" section to `docs/MCP_TOOLS.md` documenting all event types, when they fire, what fields are populated, and client typing indicator pattern
- [ ] 5.2 Document the `logging/setLevel` prerequisite for receiving streaming notifications

## 6. Verify

- [ ] 6.1 `./build.sh` passes
- [ ] 6.2 `go test ./... -short` passes
- [ ] 6.3 `gofmt -l .` produces no output
- [ ] 6.4 `golangci-lint run --enable gocritic ./cmd/... ./internal/...` passes
- [ ] 6.5 `cd test/cmd && go run . --test` integration tests pass
