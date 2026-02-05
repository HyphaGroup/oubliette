## 1. Add Registry Infrastructure

- [x] 1.1 Create `internal/mcp/registry.go` with `Registry` struct, `ToolDef`, `Register[P]()`, `GenerateSchema[P]()`
- [x] 1.2 Implement `GetAllTools()`, `GetToolsForScope()`, `CallTool()`, `RegisterWithMCPServer()`
- [x] 1.3 Add unit tests for schema generation

## 2. Replace Tool Registration

- [x] 2.1 Add `Registry` to `Server`, delete `registerTools()` and `toolMetadata`
- [x] 2.2 Register all 25 tools via `mcp.Register[P]()` calls grouped by category
- [x] 2.3 Delete `dispatchToolCall()` switch, use `registry.CallTool()`
- [x] 2.4 Delete manual `getToolsForScope()`, use `registry.GetToolsForScope()`

## 3. Validation

- [x] 3.1 Run `./build.sh` and integration tests
- [x] 3.2 Verify `--coverage-report` works
- [x] 3.3 Spot-check generated schemas match expected structure

**Implementation Notes**:
- Auto-schema generation from struct tags now drives tool validation
- Fixed `session_message` handler to make `workspace_id` optional (defaults to project's default workspace)
- Added test helper `GetDefaultWorkspaceID()` for tests that need explicit workspace ID
- All 12 openspec tests and 14/15 container/project tests pass
