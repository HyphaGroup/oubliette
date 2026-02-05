## Context

Oubliette exposes 25 MCP tools. Currently, adding a new tool requires:
1. Define param struct in `handlers_*.go`
2. Add manual JSON schema in `tools.go` `toolMetadata`
3. Add `mcp.AddTool()` call in `server.go` `registerTools()`
4. Add case in `tools.go` `dispatchToolCall()` switch
5. Add auth check in handler

Ant's `Registry` pattern reduces this to:
1. Define param struct with json tags
2. Call `mcp.Register[P](registry, def, handler)`

## Goals

- Single registration point for each tool
- Auto-generated JSON schemas from Go param structs
- Declarative scope/auth metadata
- Preserve existing handler signatures
- Enable `GetAllTools()` for coverage tooling

## Non-Goals

- Changing handler function signatures
- Automatic auth enforcement (phase 2, optional)
- Changing MCP SDK dependency

## Decisions

### Decision: Port Ant's Registry with Oubliette Adaptations

Copy `internal/mcp/registry.go` from Ant with these changes:
- Keep `Scope` field (read/write/admin) instead of Ant's `Role` field
- Remove `Capability` field (Oubliette doesn't have capability grouping)
- Keep `GenerateSchema[P]()` reflection-based schema generation
- Add `GetToolsForScope(scope string)` for socket tool filtering

**Alternatives considered:**
- Write from scratch: More work, same result
- Use Ant as dependency: Unnecessary coupling

### Decision: Clean Replacement (No Phasing)

Per project philosophy: rip-and-replace in one change.
- Add registry infrastructure
- Migrate all tools in one pass
- Delete old code immediately

### Decision: Keep Handler Signatures

Handlers keep existing signature:
```go
func (s *Server) handleFoo(ctx context.Context, req *mcp.CallToolRequest, params *FooParams) (*mcp.CallToolResult, any, error)
```

The registry wraps this to match MCP SDK expectations internally.

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| Reflection-based schema may miss edge cases | Unit tests for schema generation |
| Large diff (touches many files) | Split into handler groups if needed |
| Registry adds abstraction layer | Well worth it for reduced boilerplate |

## Migration Plan

1. Add `registry.go` with `Registry`, `ToolDef`, `Register[P]()`, `GenerateSchema[P]()`
2. Create registry instance in `Server`
3. Migrate one handler group (e.g., project_*) to validate pattern
4. Migrate remaining handler groups
5. Remove old `toolMetadata` array and `dispatchToolCall()` switch
6. Update coverage analyzer to use `registry.GetAllTools()`

Rollback: Revert commit (no data migration needed)
