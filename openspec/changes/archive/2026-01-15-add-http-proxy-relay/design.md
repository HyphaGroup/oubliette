# Design: HTTP Proxy Relay

## Context

Apple Container uses a lightweight VM, so `127.0.0.1` inside the container refers to the VM's localhost, not the host machine. There's no `host.docker.internal` equivalent. External MCP servers (like Ant) listening on the host's localhost are unreachable from the container.

We already have a bidirectional socket connection between oubliette-client (inside container) and oubliette-server (on host) via the relay. We can leverage this to proxy HTTP requests.

## Goals

- Enable agents to call external MCP servers (Ant) from inside containers
- Only enable proxy for sessions that need it (configured per-session)
- Minimal changes to existing architecture
- Support authentication tokens passed at session start

## Non-Goals

- General-purpose HTTP proxy (only for configured endpoints)
- WebSocket proxying
- Streaming responses (request/response only)

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│ Container (Apple Container VM)                                   │
│                                                                  │
│  ┌─────────────┐     ┌──────────────────┐                       │
│  │    Droid    │────▶│ oubliette-client │                       │
│  │             │     │  (MCP stdio)     │                       │
│  └─────────────┘     │                  │                       │
│         │            │  HTTP Proxy      │                       │
│         │            │  127.0.0.1:19999 │                       │
│         ▼            └────────┬─────────┘                       │
│  ┌─────────────┐              │                                 │
│  │ MCP config  │              │ unix socket                     │
│  │ ant server: │              │ /mcp/relay.sock                 │
│  │ http://     │              ▼                                 │
│  │ 127.0.0.1:  │     ┌──────────────────┐                       │
│  │ 19999/ant   │     │ oubliette-relay  │                       │
│  └─────────────┘     └────────┬─────────┘                       │
│                               │                                 │
└───────────────────────────────┼─────────────────────────────────┘
                                │ published socket
                                ▼
┌───────────────────────────────────────────────────────────────────┐
│ Host                                                              │
│                                                                   │
│  ┌──────────────────┐         ┌─────────────────┐                │
│  │ oubliette-server │────────▶│   Ant MCP       │                │
│  │ (socket_handler) │  HTTP   │ 127.0.0.1:8081  │                │
│  │                  │         └─────────────────┘                │
│  └──────────────────┘                                            │
│                                                                   │
└───────────────────────────────────────────────────────────────────┘
```

## Protocol Extension

### New message type: `http_proxy`

Request (client → server via socket):
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "http_proxy",
  "params": {
    "target": "ant",
    "method": "POST",
    "path": "/mcp",
    "headers": {"Content-Type": "application/json"},
    "body": "{\"jsonrpc\":\"2.0\",...}"
  }
}
```

Response (server → client):
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "status": 200,
    "headers": {"Content-Type": "application/json"},
    "body": "{\"jsonrpc\":\"2.0\",...}"
  }
}
```

### Proxy configuration

Passed via `session_message` context:
```json
{
  "context": {
    "http_proxies": {
      "ant": {
        "url": "http://127.0.0.1:8081/mcp",
        "headers": {
          "X-Ant-Token": "jwt-token-here"
        }
      }
    }
  }
}
```

Server stores proxy config per-session. When client sends `http_proxy` request with `target: "ant"`, server looks up the URL and headers, forwards the request, returns the response.

## Proxy Initialization Flow

1. Ant calls `session_message` with `context.http_proxies.ant` config
2. oubliette-server stores config in session state
3. On socket connection, server sends `proxy_config` message to client:
   ```json
   {"type": "proxy_config", "proxies": ["ant"]}
   ```
4. oubliette-client starts HTTP server on `127.0.0.1:19999`
5. Droid's MCP config references `http://127.0.0.1:19999/ant`

## Decisions

### Port number: 19999

Fixed port inside container. No conflicts since each container is isolated. Simple and predictable for MCP config templates.

### Path-based routing: `/ant`, `/other`

HTTP proxy uses path prefix to identify target. `GET /ant/foo` → forwards to ant's `/foo`.

### Synchronous request/response

No streaming. HTTP request blocks until response received via socket. Timeout of 30 seconds per request.

### No persistent connections

Each HTTP request creates a new socket message exchange. Simple, stateless, reliable.

## Alternatives Considered

1. **VM gateway IP (192.168.64.1)**: Undocumented, fragile, requires Ant to bind 0.0.0.0
2. **Tool relay (proxy tools not HTTP)**: More complex, requires dynamic tool registration
3. **Mount Ant's socket**: Requires Ant changes, stdio MCP only

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| Latency from socket round-trip | Acceptable for MCP calls (~10ms overhead) |
| Large response bodies | 10MB limit, error if exceeded |
| Socket disconnection mid-request | Return 502 error to Droid |

## Security Considerations

- Proxy only forwards to pre-configured targets (no arbitrary URLs)
- Auth tokens stored server-side, never exposed to container filesystem
- Tokens scoped per-session, discarded when session ends

## Migration Plan

No migration needed - new capability. Existing sessions unaffected.

## Open Questions

1. Should proxy config be stored in workspace `.factory/` or purely in-memory per session?
   - **Decision**: In-memory per session. Tokens are sensitive, shouldn't hit disk.
