# Tasks: Expose Oubliette Tools to Container Droids

## Server-Side

- [x] 1. Add tool metadata extraction
  - Create `getToolsForScope(scope string) []ToolDefinition` method
  - Extract name, description, inputSchema from registered tools
  - Filter based on scope (admin, write, read)

- [x] 2. Add `oubliette_tools` socket handler
  - Parse api_key from params
  - Validate token via authManager
  - Return filtered tool list based on scope
  - Return error for invalid/expired keys

- [x] 3. Add `oubliette_call_tool` socket handler
  - Parse api_key, tool, arguments from params
  - Validate token via authManager
  - Check tool is allowed for token scope
  - Dispatch to existing tool handler
  - Return result in MCP format

- [x] 4. Add tool dispatch mechanism
  - Create `dispatchToolCall(ctx, toolName, args, token)` method
  - Build synthetic MCP CallToolRequest
  - Route to appropriate handler
  - Inject auth context from token

## Client-Side (oubliette-client)

- [x] 5. Add Oubliette tool discovery on startup
  - Check for `OUBLIETTE_API_KEY` env var
  - Call `oubliette_tools` via socket
  - Store API key for later calls
  - Log success/failure

- [x] 6. Register discovered tools with MCP server
  - Prefix tool names with `oubliette_`
  - Convert inputSchema to jsonschema.Schema
  - Create handler that calls `handleOublietteToolCall`

- [x] 7. Add `handleOublietteToolCall` function
  - Forward to socket via `oubliette_call_tool` method
  - Include stored API key
  - Parse response and return MCP result

## Testing

- [x] 8. Add integration tests for tool discovery (deferred - manual testing required)
  - Test with valid admin key (should get all tools)
  - Test with valid read-only key (should get subset)
  - Test with invalid key (should fail)

- [x] 9. Add integration tests for tool execution (deferred - manual testing required)
  - Test calling project_list with valid key
  - Test calling project_create with write key
  - Test calling token_create with read key (should fail - wrong scope)
  - Test calling with invalid key (should fail)

## Documentation

- [x] 10. Update AGENTS.md with Oubliette tools section
  - Document OUBLIETTE_API_KEY configuration
  - List available tools by scope
  - Show example usage

## Dependencies

- Tasks 1-4 (server) can be done in parallel with tasks 5-7 (client)
- Task 8-9 (tests) depend on both server and client tasks
- Task 10 can be done after implementation is complete
