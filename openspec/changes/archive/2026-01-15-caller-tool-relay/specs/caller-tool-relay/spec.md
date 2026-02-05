# Caller Tool Relay Specification

## ADDED Requirements

### Requirement: REQ-CTR-001: Droid can call caller tools as native MCP tools

Droids running inside containers SHALL be able to call caller-provided tools as native MCP tools with prefixed names (e.g., `ant_send_response`), using the existing SSE stream for communication.

#### Scenario: Droid calls caller tool successfully

- Given a Droid session is active with caller_id "ant" and caller_tools including "send_response"
- When Droid calls `ant_send_response({"message": "hello"})`
- Then oubliette-server emits a `caller_tool_request` event on the SSE stream
- And the caller executes the tool and calls `caller_tool_response` with the result
- And Droid receives the result from the tool call

#### Scenario: Caller tool request times out

- Given a Droid session is active with caller_id "ant"
- When Droid calls `ant_slow_tool({})`
- And the caller does not respond within 60 seconds
- Then Droid receives a timeout error

#### Scenario: Caller disconnected before response

- Given a Droid session is active with caller_id "ant"
- When Droid calls `ant_send_response({})`
- And the caller disconnects before responding
- Then Droid receives a connection error

### Requirement: REQ-CTR-002: Dynamic tool registration from caller declaration

Oubliette SHALL dynamically register caller tools as MCP tools when the caller declares them in session context.

#### Scenario: Caller declares tools at session start

- Given Ant calls `session_message` with context `{"caller_id": "ant", "caller_tools": [{"name": "send_response", "description": "Send message"}]}`
- When oubliette-client receives the `caller_tools_config` notification
- Then it registers `ant_send_response` as an MCP tool
- And Droid can discover and call `ant_send_response`

#### Scenario: Tools include input schema

- Given Ant declares a tool with `inputSchema: {"type": "object", "properties": {"message": {"type": "string"}}}`
- When Droid lists available tools
- Then `ant_send_response` includes the input schema for validation

### Requirement: REQ-CTR-003: SSE event for tool requests

Oubliette SHALL emit `caller_tool_request` events on the SSE stream when Droids request tool execution.

#### Scenario: Tool request event format

- Given a Droid calls `ant_send_response({"message": "hi"})`
- When the event is emitted on the SSE stream
- Then it contains `type: "caller_tool_request"`
- And it contains a unique `request_id`
- And it contains `tool: "send_response"`
- And it contains `arguments: {"message": "hi"}`

### Requirement: REQ-CTR-004: MCP method for tool responses

Oubliette SHALL provide a `caller_tool_response` MCP method for callers to send tool execution results back.

#### Scenario: Successful tool response

- Given a `caller_tool_request` event was emitted with request_id "abc123"
- When the caller calls `caller_tool_response(session_id, "abc123", {"status": "sent"}, null)`
- Then the pending request is resolved
- And the result is returned to the Droid

#### Scenario: Tool execution error response

- Given a `caller_tool_request` event was emitted with request_id "abc123"
- When the caller calls `caller_tool_response(session_id, "abc123", null, "recipient not found")`
- Then the pending request is resolved
- And the error is returned to the Droid

### Requirement: REQ-CTR-005: Socket protocol for caller_tool

The socket protocol between oubliette-client and oubliette-server SHALL support `caller_tool` requests.

#### Scenario: Socket request and response

- Given oubliette-client is connected to oubliette-server
- When oubliette-client sends `{"method": "caller_tool", "params": {"tool": "x", "arguments": {}}}`
- Then oubliette-server processes the request
- And returns the result when the caller responds
