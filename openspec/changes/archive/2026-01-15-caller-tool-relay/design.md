# Design: Caller Tool Relay

## MCP SDK Alignment (go-sdk v1.2.0)

This design leverages the official MCP Go SDK capabilities:

| Feature | SDK Capability | Our Implementation | Notes |
|---------|---------------|-------------------|-------|
| Event push | `ServerSession.Log()` | `caller_tool_request` events | `LoggingMessageParams.Data` accepts any JSON |
| Dynamic tools | `Server.AddTool()` | oubliette-client registers `{caller}_{tool}` | SDK validates inputSchema type="object" |
| Tool schema | `Tool.InputSchema` | Pass-through from caller's `inputSchema` | Direct mapping |
| Response | `AddTool()` | `caller_tool_response` as MCP tool | Cleanest SDK-compatible approach |

### Key SDK Types Used

```go
// Event push via logging (server.go)
type LoggingMessageParams struct {
    Level  LoggingLevel `json:"level"`
    Logger string       `json:"logger,omitempty"`
    Data   any          `json:"data"`  // Our event payload goes here
}

// Dynamic tool registration (server.go)
type Tool struct {
    Name        string `json:"name"`
    Description string `json:"description,omitempty"`
    InputSchema any    `json:"inputSchema"`  // Must have type="object"
}

// Already using correctly in ActiveSession.PushEvent():
params := &mcp.LoggingMessageParams{
    Logger: "oubliette.session",
    Level:  "info",
    Data:   eventData,  // Our caller_tool_request structure
}
session.Log(ctx, params)
```

### Why `caller_tool_response` is an MCP Tool (not custom method)

The SDK doesn't support adding custom JSON-RPC methods outside the MCP spec. Options considered:

1. **Custom method via middleware** - Requires low-level JSON-RPC handling, fragile
2. **MCP Tool** ✅ - Fully supported, clean API, validates schema

Implementation:
```go
mcp.AddTool(server, &mcp.Tool{
    Name:        "caller_tool_response",
    Description: "Respond to a caller tool request from a session",
    InputSchema: callerToolResponseSchema,
}, s.handleCallerToolResponse)
```

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              Container                                   │
│  ┌─────────┐    ┌──────────────────┐    ┌─────────────────────────────┐ │
│  │  Droid  │───▶│ oubliette-client │───▶│ Unix Socket (/mcp/relay)    │ │
│  │         │    │                  │    │                             │ │
│  │ calls:  │    │ caller_tool()    │    │                             │ │
│  │ caller_ │    │ blocks waiting   │    │                             │ │
│  │ tool()  │    │ for response     │    │                             │ │
│  └─────────┘    └──────────────────┘    └──────────────┬──────────────┘ │
└─────────────────────────────────────────────────────────┼───────────────┘
                                                          │
                                                          ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          oubliette-server                                │
│  ┌─────────────────────────────────────────────────────────────────────┐│
│  │                      SocketHandler                                   ││
│  │  handleCallerTool(request_id, tool, args)                           ││
│  │    → stores pending request in ActiveSession                        ││
│  │    → pushes caller_tool_request event to SSE stream                 ││
│  │    → waits for response (with timeout)                              ││
│  └─────────────────────────────────────────────────────────────────────┘│
│                                    │                                     │
│                                    ▼                                     │
│  ┌─────────────────────────────────────────────────────────────────────┐│
│  │                      ActiveSession                                   ││
│  │  pendingCallerRequests map[string]chan CallerToolResponse           ││
│  │  SetCallerToolResponse(request_id, result)                          ││
│  └─────────────────────────────────────────────────────────────────────┘│
│                                    │                                     │
│                                    ▼                                     │
│  ┌─────────────────────────────────────────────────────────────────────┐│
│  │                      SSE Event Push                                  ││
│  │  ServerSession.Log() emits:                                         ││
│  │  {"type":"caller_tool_request","request_id":"...","tool":"...",     ││
│  │   "arguments":{...}}                                                ││
│  └─────────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼ SSE Stream
┌─────────────────────────────────────────────────────────────────────────┐
│                              Caller (Ant)                                │
│  ┌─────────────────────────────────────────────────────────────────────┐│
│  │                      SSE Event Handler                               ││
│  │  on("caller_tool_request"):                                         ││
│  │    → execute tool locally                                           ││
│  │    → POST /mcp caller_tool_response(request_id, result)             ││
│  └─────────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼ MCP Call
┌─────────────────────────────────────────────────────────────────────────┐
│                          oubliette-server                                │
│  ┌─────────────────────────────────────────────────────────────────────┐│
│  │                 handleCallerToolResponse()                           ││
│  │  → looks up pending request by request_id                           ││
│  │  → sends result to waiting channel                                  ││
│  │  → SocketHandler returns result to oubliette-client                 ││
│  └─────────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────────┘
```

## Data Structures

### Context: caller_tools declaration

Passed in `session_message` context:
```json
{
  "context": {
    "caller_id": "ant",
    "caller_tools": [
      {
        "name": "send_response",
        "description": "Send a message via Signal",
        "inputSchema": {
          "type": "object",
          "properties": {
            "message": {"type": "string", "description": "Message to send"},
            "recipients": {"type": "array", "items": {"type": "string"}}
          },
          "required": ["message", "recipients"]
        }
      },
      {
        "name": "get_memory",
        "description": "Retrieve stored memories for context"
      }
    ]
  }
}
```

Stored in `ActiveSession` and sent to oubliette-client via `caller_tools_config` notification.

### Socket Notification: caller_tools_config

Sent from oubliette-server to oubliette-client when caller tools are declared:
```json
{
  "type": "caller_tools_config",
  "params": {
    "caller_id": "ant",
    "tools": [
      {"name": "send_response", "description": "...", "inputSchema": {...}},
      {"name": "get_memory", "description": "..."}
    ]
  }
}
```

oubliette-client dynamically registers these as MCP tools:
- `ant_send_response`
- `ant_get_memory`

### SSE Event: caller_tool_request

```json
{
  "type": "caller_tool_request",
  "request_id": "uuid-v4",
  "tool": "send_response",
  "arguments": {
    "message": "Hello from Droid",
    "recipients": ["+1234567890"]
  }
}
```

### MCP Tool: caller_tool_response

Implemented as a standard MCP tool (not a custom method) for SDK compatibility.

```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "caller_tool_response",
    "arguments": {
      "session_id": "gogol_xxx",
      "request_id": "uuid-v4",
      "result": { ... },
      "error": null
    }
  }
}
```

Tool registration (oubliette-server):
```go
type CallerToolResponseParams struct {
    SessionID string          `json:"session_id"`
    RequestID string          `json:"request_id"`
    Result    json.RawMessage `json:"result,omitempty"`
    Error     *string         `json:"error,omitempty"`
}

mcp.AddTool(server, &mcp.Tool{
    Name:        "caller_tool_response",
    Description: "Respond to a caller_tool_request event with the tool execution result",
    InputSchema: buildSchema(CallerToolResponseParams{}),
}, s.handleCallerToolResponse)
```

### Socket Protocol: caller_tool

Request (oubliette-client → oubliette-server):
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "caller_tool",
  "params": {
    "tool": "send_response",
    "arguments": { ... }
  }
}
```

Response (oubliette-server → oubliette-client):
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": { ... }
}
```

## Timeout Handling

- Socket request timeout: 60 seconds (configurable)
- If timeout expires before caller responds, return error to Droid
- Caller is responsible for executing promptly

## Security

- Only the caller that created the session can respond to tool requests
- Request ID prevents replay attacks
- Session-scoped: requests only valid for the active session

## Error Cases

1. **Caller disconnected**: Return error immediately
2. **Timeout**: Return timeout error after 60s
3. **Tool execution failed**: Caller sends error in response
4. **Invalid request_id**: Log warning, ignore response
