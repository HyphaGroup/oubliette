# session-streaming Specification

## Purpose
TBD - created by archiving change add-sse-event-streaming. Update Purpose after archive.
## Requirements
### Requirement: SSE Event Delivery

The system SHALL push Factory Droid session events to connected MCP clients via SSE notifications as events occur.

#### Scenario: Client receives working state event
- **GIVEN** a client has called `session_message` to start or continue a session
- **AND** the client has an open SSE stream to the MCP endpoint
- **AND** the client has set logging level via MCP
- **WHEN** Factory Droid emits a working state change event
- **THEN** the client receives a log notification containing the event type and details within 100ms

#### Scenario: Client receives tool usage event
- **GIVEN** a client has an active session with an open SSE stream
- **WHEN** Factory Droid invokes a tool
- **THEN** the client receives a log notification with tool name and status in the Data field

#### Scenario: Client receives message event
- **GIVEN** a client has an active session with an open SSE stream
- **WHEN** Factory Droid produces text output
- **THEN** the client receives a log notification with the message content in the Data field

#### Scenario: No SSE stream open
- **GIVEN** a client has called `session_message` but has not opened an SSE stream
- **WHEN** Factory Droid emits events
- **THEN** events are buffered in the EventBuffer for polling via `session_events`
- **AND** no errors occur due to missing SSE stream

### Requirement: Stream Resumption

The system SHALL support automatic stream resumption when a client reconnects after network interruption.

#### Scenario: Client reconnects with Last-Event-ID
- **GIVEN** a client was receiving SSE events
- **AND** the network connection was interrupted
- **WHEN** the client reconnects with the `Last-Event-ID` header set to the last received event ID
- **THEN** the server replays all events after that ID
- **AND** streaming continues from the current position

#### Scenario: Events purged before reconnect
- **GIVEN** a client was disconnected for an extended period
- **AND** the EventStore has purged old events
- **WHEN** the client reconnects with an old `Last-Event-ID`
- **THEN** the server returns an error indicating events were purged
- **AND** the client can fall back to `session_events` polling

### Requirement: EventStore Configuration

The system SHALL use a memory-backed EventStore for SSE stream persistence.

#### Scenario: EventStore enabled
- **GIVEN** the MCP server is configured with StreamableHTTPOptions
- **WHEN** a client connects via SSE
- **THEN** events are persisted to the MemoryEventStore for replay

#### Scenario: EventStore memory limit
- **GIVEN** the EventStore has reached its memory limit (default 10MB)
- **WHEN** new events arrive
- **THEN** oldest events are purged to stay within the limit
- **AND** new events are stored successfully

### Requirement: Backwards Compatibility

The system SHALL maintain full compatibility with the `session_events` polling mechanism.

#### Scenario: Polling continues to work
- **GIVEN** a client calls `session_message`
- **WHEN** the client polls `session_events` with `since_index`
- **THEN** buffered events are returned as before
- **AND** SSE streaming does not affect polling behavior

#### Scenario: Mixed mode operation
- **GIVEN** a client has an open SSE stream
- **AND** the client also polls `session_events`
- **WHEN** events occur
- **THEN** events are delivered via both SSE and available for polling
- **AND** no events are lost or duplicated

