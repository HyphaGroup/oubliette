# Tasks: Add SSE Event Streaming

## 1. Enable EventStore on StreamableHTTPHandler

- [x] 1.1 Add `StreamableHTTPOptions` with `EventStore: mcp.NewMemoryEventStore(nil)` in `server.go`
- [x] 1.2 Verify SSE stream works by connecting with MCP client and checking `text/event-stream` response

## 2. Add ServerSession tracking to ActiveSession

- [x] 2.1 Add `mcpSession *mcp.ServerSession` field to `ActiveSession` struct
- [x] 2.2 Add `SetMCPSession()` and `GetMCPSession()` methods with mutex protection
- [x] 2.3 Add `NotifyEvent()` method that calls `mcpSession.Log()` if session exists

## 3. Capture ServerSession in session_message handler

- [x] 3.1 Extract `*mcp.ServerSession` from `request.Session` in `handleSendMessage()`
- [x] 3.2 Store ServerSession in ActiveSession via `SetMCPSession()`
- [x] 3.3 Handle case where session already exists (update ServerSession reference)

## 4. Push events via Log

- [x] 4.1 Modify `collectEvents()` to call `NotifyEvent()` for each received event
- [x] 4.2 Format event as `LoggingMessageParams` with event JSON in `Data` field
- [x] 4.3 Handle errors from Log gracefully (log locally, don't fail)
- [x] 4.4 Skip notification if mcpSession is nil (no client watching)

## 5. Testing and validation

- [x] 5.1 Add integration test: client receives events via SSE without polling
- [x] 5.2 Add integration test: stream resumption after disconnect
- [x] 5.3 Verify `session_events` polling still works unchanged
- [x] 5.4 Test with Ant client end-to-end

## Dependencies

- Tasks 1.x can be done in parallel with 2.x
- Task 3.x depends on 2.x (needs ActiveSession methods)
- Task 4.x depends on 2.x and 3.x
- Task 5.x depends on all above
