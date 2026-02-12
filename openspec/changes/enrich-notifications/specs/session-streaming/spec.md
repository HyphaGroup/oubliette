## ADDED Requirements

### Requirement: Status Transition Notifications

The system SHALL push status transition events to MCP clients when a session's active status changes.

#### Scenario: Session starts processing
- **GIVEN** a session is in `idle` status
- **WHEN** the first work event (delta, tool_call, tool_result, or assistant message) arrives
- **THEN** a `status` event with text `"running"` is buffered in the EventBuffer
- **AND** a notification is pushed to the connected MCP client

#### Scenario: Session completes turn
- **GIVEN** a session is in `running` status
- **WHEN** a completion event arrives
- **THEN** a `status` event with text `"idle"` is buffered in the EventBuffer
- **AND** a notification is pushed to the connected MCP client

#### Scenario: Session fails
- **GIVEN** a session is in `running` status
- **WHEN** an executor error occurs
- **THEN** a `status` event with text `"failed"` is buffered in the EventBuffer
- **AND** a notification is pushed to the connected MCP client

#### Scenario: Status events available via polling
- **GIVEN** a session has transitioned from idle to running to idle
- **WHEN** a client calls `session_events` with `since_index`
- **THEN** the returned events include `status` type events with the transition values

### Requirement: Consolidated Message Notifications

The system SHALL push consolidated assistant message text to MCP clients during text generation, with deduplication.

#### Scenario: New text content pushed
- **GIVEN** a session is actively generating text
- **WHEN** a `message` event arrives with text different from the last notified text
- **THEN** a notification containing the consolidated text is pushed to the MCP client

#### Scenario: Duplicate text suppressed
- **GIVEN** a session has already pushed a message notification with text "Hello world"
- **WHEN** another `message` event arrives with the same text "Hello world"
- **THEN** no notification is pushed

#### Scenario: Message events serve as activity heartbeat
- **GIVEN** a client is showing a typing indicator
- **AND** the session is generating a long text response with no tool calls
- **WHEN** consolidated message events arrive periodically during generation
- **THEN** the client can use these events to confirm the session is still active

### Requirement: Unified Event Shape

The system SHALL use a single event type for both streaming notifications and polling responses.

#### Scenario: Streaming notification shape
- **GIVEN** a session pushes a notification to the MCP client via `logging/message`
- **THEN** the `Data` field contains a JSON object with keys: `index`, `type`, `text`, `tool_name`, `role`, `session_id`

#### Scenario: Polling response shape
- **GIVEN** a client calls `session_events`
- **THEN** each event in the `events` array contains the same keys: `index`, `type`, `text`, `tool_name`, `role`, `session_id`

#### Scenario: Same event type produces same output
- **GIVEN** a `tool_call` event is emitted
- **WHEN** it is serialized for streaming notification
- **AND** it is serialized for polling response
- **THEN** both serializations produce the same JSON structure (differing only in `index` which is 0 for streaming)

## MODIFIED Requirements

### Requirement: SSE Event Delivery

The system SHALL push session events to connected MCP clients via log notifications as events occur.

#### Scenario: Client receives status transition event
- **GIVEN** a client has an active session with an MCP connection
- **AND** the client has set logging level via MCP
- **WHEN** the session transitions from idle to running
- **THEN** the client receives a log notification with type `status` and text `running`

#### Scenario: Client receives tool usage event
- **GIVEN** a client has an active session with an MCP connection
- **AND** the client has set logging level via MCP
- **WHEN** the agent invokes a tool
- **THEN** the client receives a log notification with the tool name in `tool_name`

#### Scenario: Client receives consolidated message event
- **GIVEN** a client has an active session with an MCP connection
- **AND** the client has set logging level via MCP
- **WHEN** the agent produces text output and the text differs from the last notification
- **THEN** the client receives a log notification with type `message` and the accumulated text

#### Scenario: Client receives completion event
- **GIVEN** a client has an active session with an MCP connection
- **AND** the client has set logging level via MCP
- **WHEN** the agent finishes a turn
- **THEN** the client receives a log notification with type `completion` and the final response text

#### Scenario: No MCP connection
- **GIVEN** a session has no connected MCP client
- **WHEN** events occur
- **THEN** events are buffered in the EventBuffer for polling via `session_events`
- **AND** no errors occur

### Requirement: Backwards Compatibility

The system SHALL maintain full compatibility with the `session_events` polling mechanism.

#### Scenario: Polling returns all event types
- **GIVEN** a session has emitted status, message, tool_call, tool_result, and completion events
- **WHEN** the client polls `session_events` with `since_index: -1`
- **THEN** all event types are returned in the response

#### Scenario: Mixed mode operation
- **GIVEN** a client has an MCP connection for streaming
- **AND** the client also polls `session_events`
- **WHEN** events occur
- **THEN** events are delivered via both streaming notifications and available for polling
- **AND** no events are lost or duplicated

## REMOVED Requirements

### Requirement: Stream Resumption
**Reason**: This described SSE stream reconnection with `Last-Event-ID`, which is handled by the MCP transport layer (StreamableHTTP), not by the application event pipeline. The EventBuffer resumption protocol (`since_index`) covers the polling case.

### Requirement: EventStore Configuration
**Reason**: This described memory-backed EventStore for SSE persistence, which is an MCP transport concern. The application uses its own ring buffer (`EventBuffer`) for event storage. Transport-level SSE resumption is handled by the MCP go-sdk's `StreamableHTTPOptions.EventStore`.
