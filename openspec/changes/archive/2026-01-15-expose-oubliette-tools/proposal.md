# Proposal: Expose Oubliette Tools to Container Droids

## Summary

Enable Droids running inside Oubliette containers to call Oubliette's own MCP tools (project_create, token_create, etc.) when configured with an API key. This allows building admin agents that can manage Oubliette on behalf of users.

## Motivation

Currently, Droids inside containers can only:
1. Call tools on the parent caller via caller_tool relay (e.g., Ant's tools)
2. Call the limited tools exposed by oubliette-client (session_message, project_list)

**Use Case:** An admin agent that:
- Receives requests from users via Signal (through Ant)
- Creates/manages Oubliette projects on their behalf
- Provisions API tokens for new users
- Manages workspaces and sessions

Without this feature, the Droid cannot perform admin operations directly - it would need to route everything through the caller, adding complexity.

## Design

### Configuration

When a project/workspace has `OUBLIETTE_API_KEY` environment variable set:
1. oubliette-client detects this on startup
2. Requests the available tools list from oubliette-server via the existing socket
3. Registers those tools (prefixed with `oubliette_`) for the Droid to call
4. Tool calls are forwarded through the socket to the server

### Tool Discovery

oubliette-client sends a new socket request:
```json
{"jsonrpc": "2.0", "id": 1, "method": "oubliette_tools", "params": {"api_key": "..."}}
```

Server responds with the list of tools available for that key's scope:
```json
{"jsonrpc": "2.0", "id": 1, "result": {"tools": [{"name": "project_create", ...}, ...]}}
```

### Tool Execution

When Droid calls `oubliette_project_create`:
1. oubliette-client intercepts the call
2. Sends via socket: `{"method": "oubliette_call_tool", "params": {"tool": "project_create", "arguments": {...}, "api_key": "..."}}`
3. Server validates the API key and executes the tool
4. Response flows back through socket

### Security

- API key is validated on every tool call (not just discovery)
- Token scopes respected (read-only tokens can't call write tools)
- Key stored only in environment variable, never logged
- Server-side auth handlers already exist and are reused

## Architecture

```
Droid (inside container)
    ↓ stdio (MCP)
oubliette-client
    ↓ unix socket (existing relay)
oubliette-server
    ↓ internal tool handlers
Tool execution with API key auth
```

No new network paths - uses existing socket relay that already bypasses Apple Container's network isolation.

## Alternatives Considered

### A. HTTP Client Inside Container
oubliette-client makes HTTP calls directly to host's :8080.

**Rejected:** Doesn't work in Apple Container (no host network access). Would require new network plumbing.

### B. Caller Proxies Oubliette Tools
Ant includes Oubliette tools in caller_tools, proxies calls.

**Rejected:** Adds latency and complexity. Ant shouldn't need to know about Oubliette internals.

### C. Special Admin Project Flag
Mark certain projects as "admin" with full tool access.

**Rejected:** Less flexible than per-key scoping. API keys already have scope system.

## Scope

**In Scope:**
- Socket protocol for tool discovery (`oubliette_tools`)
- Socket protocol for tool execution (`oubliette_call_tool`)
- oubliette-client changes to request/register tools
- API key validation on server side

**Out of Scope:**
- New MCP tools (uses existing ones)
- Changes to auth system (reuses existing token scopes)
- UI for managing admin Droids

## Questions

1. Should all tools be exposed, or a curated subset? (Proposal: all tools, scoped by token permissions)
2. Tool prefix: `oubliette_` or something shorter like `ou_`? (Proposal: `oubliette_` for clarity)
3. Should tool discovery happen once at startup, or be refreshable? (Proposal: once at startup, simplest)
