## ADDED Requirements

### Requirement: Unified Tool Registration

All MCP tools SHALL be registered through a central Registry that captures tool metadata, generates schemas, and handles dispatch.

#### Scenario: Tool registered with auto-generated schema

- **GIVEN** a handler with typed params struct `CreateProjectParams`
- **WHEN** the tool is registered via `mcp.Register[CreateProjectParams](registry, def, handler)`
- **THEN** the registry generates JSON schema from struct field tags
- **AND** the schema includes `type`, `properties`, and `required` fields
- **AND** `json:"omitempty"` fields are not marked required

#### Scenario: Tool registered with explicit schema

- **GIVEN** a tool definition with `InputSchema` already set
- **WHEN** the tool is registered
- **THEN** the explicit schema is used instead of auto-generation

#### Scenario: All tools enumerable

- **GIVEN** 25 tools are registered
- **WHEN** `registry.GetAllTools()` is called
- **THEN** all 25 tool definitions are returned in registration order

### Requirement: Schema Generation from Go Types

The registry SHALL generate JSON Schema from Go struct types using reflection.

#### Scenario: String field generates string schema

- **GIVEN** a struct field `Name string json:"name"`
- **WHEN** schema is generated
- **THEN** property schema is `{"type": "string"}`

#### Scenario: Integer field generates integer schema

- **GIVEN** a struct field `Limit int json:"limit"`
- **WHEN** schema is generated
- **THEN** property schema is `{"type": "integer"}`

#### Scenario: Boolean field generates boolean schema

- **GIVEN** a struct field `Force bool json:"force"`
- **WHEN** schema is generated
- **THEN** property schema is `{"type": "boolean"}`

#### Scenario: Array field generates array schema

- **GIVEN** a struct field `Tags []string json:"tags"`
- **WHEN** schema is generated
- **THEN** property schema is `{"type": "array", "items": {"type": "string"}}`

#### Scenario: Nested struct generates object schema

- **GIVEN** a struct field `Config ConfigParams json:"config"`
- **WHEN** schema is generated
- **THEN** property schema is `{"type": "object", "properties": {...}}`

#### Scenario: Omitempty field is optional

- **GIVEN** a struct field `Description string json:"description,omitempty"`
- **WHEN** schema is generated
- **THEN** `description` is NOT in the `required` array

#### Scenario: Non-omitempty field is required

- **GIVEN** a struct field `Name string json:"name"`
- **WHEN** schema is generated
- **THEN** `name` IS in the `required` array

### Requirement: Scope-Based Tool Filtering

The registry SHALL filter tools based on token scope for socket connections.

#### Scenario: Admin scope sees all tools

- **GIVEN** a token with admin scope
- **WHEN** `registry.GetToolsForScope("admin")` is called
- **THEN** all 25 tools are returned

#### Scenario: Read-only scope sees only read tools

- **GIVEN** a token with read-only scope
- **WHEN** `registry.GetToolsForScope("read-only")` is called
- **THEN** only tools with `Scope: "read"` are returned
- **AND** write and admin tools are excluded

#### Scenario: Write scope sees read and write tools

- **GIVEN** a token with write scope (non-admin, non-read-only)
- **WHEN** `registry.GetToolsForScope("write")` is called
- **THEN** tools with `Scope: "read"` or `Scope: "write"` are returned
- **AND** admin-only tools are excluded

### Requirement: MCP SDK Integration

The registry SHALL integrate with the MCP Go SDK for tool registration.

#### Scenario: Tools registered with MCP server

- **GIVEN** a registry with tools registered
- **WHEN** `registry.RegisterWithMCPServer(server)` is called
- **THEN** each tool is added to the MCP server via `mcp.AddTool()`
- **AND** handlers are wrapped to match SDK signature

#### Scenario: Tool invocation via registry

- **GIVEN** a tool "project_create" is registered
- **WHEN** `registry.CallTool(ctx, "project_create", args)` is called
- **THEN** the registered handler is invoked with parsed params
- **AND** the result is returned
