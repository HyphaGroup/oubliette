# Tasks: Caller Tool Relay

## Server-Side (oubliette-server)

- [x] 1. Add caller tools storage to ActiveSession
  - Add `callerID string`
  - Add `callerTools []CallerToolDefinition`
  - Add `SetCallerTools(callerID, tools)` called from session_message handler
  - Add `GetCallerTools() (string, []CallerToolDefinition)`
  - Send `caller_tools_config` notification to oubliette-client via socket

- [x] 2. Add pending request tracking to ActiveSession
  - Add `pendingCallerRequests map[string]chan *CallerToolResponse`
  - Add `callerRequestMu sync.RWMutex`
  - Add `RegisterCallerRequest(requestID) chan *CallerToolResponse`
  - Add `ResolveCallerRequest(requestID, response)`

- [x] 3. Add `caller_tool` handler to SocketHandler
  - Generate UUID request_id
  - Register pending request with ActiveSession
  - Push `caller_tool_request` event via `ServerSession.Log()` (SDK pattern)
  - Wait for response channel (with timeout)
  - Return result or timeout error

- [x] 4. Add `caller_tool_response` MCP tool (SDK-aligned)
  - Register via `mcp.AddTool()` for SDK compatibility (not custom method)
  - Validate session_id matches caller's session
  - Look up pending request by request_id
  - Send response to waiting channel
  - Return success/error

- [x] 5. Add SSE event type for caller_tool_request
  - Define `CallerToolRequestEvent` structure
  - Use `LoggingMessageParams.Data` for event payload (SDK pattern)
  - Ensure event is included in SSE stream via existing `PushEvent()` flow

## Client-Side (oubliette-client)

- [x] 6. Handle `caller_tools_config` notification in oubliette-client
  - Parse caller_id and tools from notification
  - Use `mcp.AddTool()` to dynamically register tools with `{caller_id}_{tool_name}` naming
  - SDK validates `inputSchema` must have `type: "object"`
  - Each tool handler calls socket `caller_tool` method with original tool name

- [x] 7. Add `caller_tool` socket method handler in oubliette-client
  - Forward tool invocation to parent via socket
  - Wait for response and return to Droid

## Testing

- [x] 8. Add integration test for caller tool relay
  - Mock caller that handles caller_tool_request events
  - Verify round-trip: Droid → oubliette-client → socket → server → SSE → caller → MCP tool → server → socket → oubliette-client → Droid

## Documentation

- [x] 9. Update AGENTS.md with caller tool relay section
  - Document the flow
  - Document available tools (caller-dependent)
  - Document timeout behavior (60s default)

## Cleanup

- [x] 10. Remove HTTP proxy code (if merged)
  - Remove from oubliette-client
  - Remove from handlers_session.go
  - Remove from socket_handler.go
  - Remove from active.go
