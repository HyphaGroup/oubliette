# Change: Add HTTP Proxy Relay for External MCP Servers

## Why

Agents running inside Apple Container VMs cannot reach services on the host's `127.0.0.1` due to VM network isolation (no `host.docker.internal` equivalent - see [apple/container#346](https://github.com/apple/container/issues/346)). This blocks agents from calling back to MCP servers like Ant that only listen on localhost.

## What Changes

- **oubliette-client**: Add HTTP proxy server that listens on `127.0.0.1:<port>` inside container
- **oubliette-server**: Add `http_proxy` message type to socket protocol for forwarding HTTP requests
- **session_message**: Accept `context.http_proxies` config specifying external endpoints to proxy
- **MCP config**: Template includes proxy endpoint with env var substitution (e.g., `${ANT_TOKEN}`)

## Impact

- Affected specs: None (new capability)
- Affected code:
  - `cmd/oubliette-client/main.go` - HTTP proxy server
  - `internal/mcp/socket_handler.go` - Handle `http_proxy` requests
  - `internal/mcp/handlers_session.go` - Pass proxy config to client
  - `template/.factory/mcp.json` - Add proxy endpoint template
