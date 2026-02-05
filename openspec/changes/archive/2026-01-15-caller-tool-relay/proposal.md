# Caller Tool Relay

## Problem

Agents running inside Oubliette containers (Droids) need to call tools on the external caller (e.g., Ant) that initiated the session. Current approaches have issues:

1. **HTTP Proxy**: Requires running a proxy server, complex timing (proxy must start before Droid), and managing separate auth tokens
2. **Network Isolation**: Apple Container VMs can't reach host's `127.0.0.1` directly

## Solution

Leverage the existing SSE stream between caller and Oubliette for bidirectional tool execution:

```
Droid → oubliette-client → socket → oubliette-server → SSE event → Caller
                                                                      ↓
Droid ← oubliette-client ← socket ← oubliette-server ← MCP call ← Caller executes tool
```

This makes the SSE stream bidirectional:
- **Caller → Oubliette**: Existing flow (session_message, session_events)
- **Oubliette → Caller**: New flow (tool requests via SSE, responses via MCP)

## Benefits

- Uses existing authenticated connection (no new auth tokens needed)
- No HTTP proxy or additional network configuration
- Works with any container runtime (Docker, Apple Container)
- Token lifetime naturally matches session lifetime
- Simple Droid-side API: `oubliette-parent.caller_tool(tool, args)`

## Components

1. **Tool declaration**: Caller passes `caller_tools` and `caller_id` in session context
2. **Dynamic tool registration**: oubliette-client registers caller tools with prefix (e.g., `ant_send_response`)
3. **New SSE event**: `caller_tool_request` - Oubliette asks caller to execute a tool
4. **New MCP method**: `caller_tool_response` - Caller returns the result

Droid sees tools like:
- `oubliette_session_message` (existing)
- `oubliette_project_list` (existing)
- `ant_send_response` (from caller)
- `ant_get_memory` (from caller)

## Out of Scope

- Timeout/retry logic on caller side (caller's responsibility)
- Multiple concurrent callers per session (single caller assumed)
