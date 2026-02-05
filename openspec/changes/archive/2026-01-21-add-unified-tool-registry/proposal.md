# Change: Add Unified Tool Registry

## Why

Oubliette's MCP tool registration is split across multiple locations with significant duplication:

1. **`tools.go`**: Manual `toolMetadata` array with ~400 lines of hand-written JSON schemas
2. **`server.go`**: `registerTools()` with 25 separate `mcp.AddTool()` calls
3. **`tools.go`**: `dispatchToolCall()` with 25-case switch statement for socket routing
4. **Each handler**: Manual `requireAuth`/`requireProjectAccess` calls

This leads to:
- **Schema drift**: Param structs and JSON schemas can get out of sync
- **Boilerplate**: Adding a tool requires changes in 4 places
- **Inconsistent auth**: Each handler implements auth checks differently
- **Hard to enumerate**: Coverage tooling must parse multiple files

Ant solved this with a unified `Registry` pattern that auto-generates schemas from Go types and centralizes metadata.

## What Changes

- **ADDED** `internal/mcp/registry.go` - Unified tool registry with auto-schema generation
- **MODIFIED** Tool registration to use `mcp.Register[P](registry, def, handler)` pattern
- **REMOVED** Manual JSON schemas from `tools.go` (generated from param structs)
- **REMOVED** `dispatchToolCall()` switch statement (registry handles dispatch)
- **MODIFIED** Auth to be declarative in tool definition, not imperative in handlers

**NOT changing:**
- Handler function signatures (still `func(ctx, req, params) (result, data, error)`)
- MCP SDK integration (registry wraps it)
- Existing param struct definitions

## Impact

- **Affected specs**: New `mcp-tools` capability spec
- **Affected code**:
  - `internal/mcp/registry.go` - New file (copied/adapted from Ant)
  - `internal/mcp/tools.go` - Remove manual schemas, keep scope constants
  - `internal/mcp/server.go` - Use registry for tool registration
  - `internal/mcp/handlers_*.go` - Remove manual auth checks (optional, phase 2)
- **Breaking changes**: None (internal refactor)
- **Benefits**:
  - Adding new tool: 1 place instead of 4
  - Schema always matches param struct
  - `registry.GetAllTools()` for coverage tooling
  - `registry.GetToolsForScope()` replaces `getToolsForScope()`
