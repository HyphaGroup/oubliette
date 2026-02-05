# HTTP Proxy Relay Specification

## ADDED Requirements

### Requirement: HTTP Proxy Configuration

The system SHALL accept HTTP proxy configuration in the `session_message` context parameter to enable agents to reach external MCP servers on the host.

#### Scenario: Configure proxy for Ant MCP server
- **GIVEN** a client calling `session_message`
- **WHEN** the context includes `http_proxies.ant` with URL and headers
- **THEN** the session stores the proxy configuration for use during execution

#### Scenario: Multiple proxy targets
- **GIVEN** a client calling `session_message`
- **WHEN** the context includes multiple entries in `http_proxies`
- **THEN** all proxy targets are available to the agent

### Requirement: HTTP Proxy Request Forwarding

The system SHALL forward HTTP requests from oubliette-client to configured proxy targets via the socket connection.

#### Scenario: Successful proxy request
- **GIVEN** an active session with proxy config for target "ant"
- **WHEN** oubliette-client sends an `http_proxy` request with target "ant"
- **THEN** the server forwards the request to the configured URL with configured headers
- **AND** returns the response status, headers, and body to the client

#### Scenario: Unknown proxy target
- **GIVEN** an active session with proxy config for target "ant"
- **WHEN** oubliette-client sends an `http_proxy` request with target "unknown"
- **THEN** the server returns an error indicating the target is not configured

#### Scenario: Proxy request timeout
- **GIVEN** an active session with proxy config
- **WHEN** the upstream server does not respond within 30 seconds
- **THEN** the server returns a timeout error to the client

#### Scenario: Response size limit exceeded
- **GIVEN** an active session with proxy config
- **WHEN** the upstream server returns a response larger than 10MB
- **THEN** the server returns an error indicating the response was too large

### Requirement: HTTP Proxy Server in Container

The oubliette-client SHALL start an HTTP proxy server inside the container when proxy configuration is provided.

#### Scenario: Proxy server starts on proxy config
- **GIVEN** oubliette-client connected to relay
- **WHEN** it receives a `proxy_config` message with one or more targets
- **THEN** it starts an HTTP server on `127.0.0.1:19999`

#### Scenario: No proxy server without config
- **GIVEN** oubliette-client connected to relay
- **WHEN** it does not receive a `proxy_config` message
- **THEN** no HTTP server is started

#### Scenario: Path-based routing
- **GIVEN** the HTTP proxy server is running with target "ant" configured
- **WHEN** Droid makes a request to `http://127.0.0.1:19999/ant/mcp`
- **THEN** the request is forwarded with path `/mcp` to the "ant" target

### Requirement: Proxy Security

The system SHALL protect proxy configurations and authentication tokens.

#### Scenario: Tokens not persisted to disk
- **GIVEN** a session with proxy config containing auth tokens
- **WHEN** the session is active
- **THEN** tokens are stored only in server memory, not written to workspace files

#### Scenario: Tokens discarded on session end
- **GIVEN** a session with proxy config containing auth tokens
- **WHEN** the session ends
- **THEN** the proxy configuration including tokens is discarded

#### Scenario: Only configured targets accessible
- **GIVEN** an active session with proxy config for specific targets
- **WHEN** oubliette-client attempts to proxy to an arbitrary URL
- **THEN** the request is rejected (only pre-configured targets allowed)
