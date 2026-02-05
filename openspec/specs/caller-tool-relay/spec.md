# caller-tool-relay Specification

## Purpose
Enable agents inside Oubliette containers to call tools exposed by the external MCP client that initiated the session.

## Requirements
### Requirement: REQ-CTR-001: Agent can call caller tools as native MCP tools

Agents running inside containers SHALL be able to call caller-provided tools as native MCP tools with prefixed names (e.g., `myapp_send_notification`), using the existing SSE stream for communication.

#### Scenario: Agent calls caller tool successfully

- Given an agent session is active with caller_id "myapp" and caller_tools including "send_notification"
- When the agent calls `myapp_send_notification({"message": "hello"})`
- Then oubliette-server emits a `caller_tool_request` event on the SSE stream
- And the caller executes the tool and calls `caller_tool_response` with the result
- And the agent receives the result from the tool call

#### Scenario: Caller tool request times out

- Given an agent session is active with caller_id "myapp"
- When the agent calls `myapp_slow_tool({})`
- And the caller does not respond within 60 seconds
- Then the agent receives a timeout error

#### Scenario: Caller disconnected before response

- Given an agent session is active with caller_id "myapp"
- When the agent calls `myapp_send_notification({})`
- And the caller disconnects before responding
- Then the agent receives a connection error

### Requirement: REQ-CTR-002: Dynamic tool registration from caller declaration

Oubliette SHALL dynamically register caller tools as MCP tools when the caller declares them in session context.

#### Scenario: Caller declares tools at session start

- Given an MCP client calls `session_message` with context `{"caller_id": "myapp", "caller_tools": [{"name": "send_notification", "description": "Send notification"}]}`
- When oubliette-client receives the `caller_tools_config` notification
- Then it registers `myapp_send_notification` as an MCP tool
- And the agent can discover and call `myapp_send_notification`

#### Scenario: Tools include input schema

- Given the caller declares a tool with `inputSchema: {"type": "object", "properties": {"message": {"type": "string"}}}`
- When the agent lists available tools
- Then `myapp_send_notification` includes the input schema for validation

### Requirement: REQ-CTR-003: SSE event for tool requests

Oubliette SHALL emit `caller_tool_request` events on the SSE stream when agents request tool execution.

#### Scenario: Tool request event format

- Given an agent calls `myapp_send_notification({"message": "hi"})`
- When the event is emitted on the SSE stream
- Then it contains `type: "caller_tool_request"`
- And it contains a unique `request_id`
- And it contains `tool: "send_notification"`
- And it contains `arguments: {"message": "hi"}`

### Requirement: REQ-CTR-004: MCP method for tool responses

Oubliette SHALL provide a `caller_tool_response` MCP method for callers to send tool execution results back.

#### Scenario: Successful tool response

- Given a `caller_tool_request` event was emitted with request_id "abc123"
- When the caller calls `caller_tool_response(session_id, "abc123", {"status": "sent"}, null)`
- Then the pending request is resolved
- And the result is returned to the agent

#### Scenario: Tool execution error response

- Given a `caller_tool_request` event was emitted with request_id "abc123"
- When the caller calls `caller_tool_response(session_id, "abc123", null, "recipient not found")`
- Then the pending request is resolved
- And the error is returned to the agent

### Requirement: REQ-CTR-005: Socket protocol for caller_tool

The socket protocol between oubliette-client and oubliette-server SHALL support `caller_tool` requests.

#### Scenario: Socket request and response

- Given oubliette-client is connected to oubliette-server
- When oubliette-client sends `{"method": "caller_tool", "params": {"tool": "x", "arguments": {}}}`
- Then oubliette-server processes the request
- And returns the result when the caller responds

