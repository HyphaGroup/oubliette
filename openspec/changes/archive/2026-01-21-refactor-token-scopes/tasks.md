# Tasks: Refactor Token Scopes

## Phase 1: Core Type Changes

- [x] Update `internal/auth/types.go`:
  - Add `ScopeAdminRO = "admin:ro"` constant
  - Add `ScopeProjectRO(projectID string) string` helper
  - Add scope parsing helpers: `IsAdminScope()`, `IsProjectScope()`, `IsReadOnlyScope()`, `ExtractProjectID()`

- [x] Update `internal/mcp/tools.go`:
  - Add `ToolTarget` type with `TargetGlobal`, `TargetProject` constants
  - Add `ToolAccess` type with `AccessRead`, `AccessWrite`, `AccessAdmin` constants
  - Keep old `toolScopeAdmin`, `toolScopeWrite`, `toolScopeRead` constants for backwards compat
  - Update `ToolDef` struct with `Target` and `Access` fields

## Phase 2: Permission Logic

- [x] Create `internal/mcp/permissions.go`:
  - Implement `IsToolAllowed(tool *ToolDef, tokenScope string, projectID string) bool`
  - Implement `ExtractProjectIDFromArgs(args map[string]any) string`
  - Add unit tests for all scope/access combinations

- [x] Update `internal/mcp/registry.go`:
  - Add `IsToolAllowedWithProject` method for project-aware permission check
  - Update tool filtering in `GetToolsForScope` to use new model
  - Fallback to legacy scope check when Target/Access not set

## Phase 3: Tool Registration Updates

- [x] Update `internal/mcp/tools_registry.go`:
  - Classify each tool with `Target` and `Access`:
    - Token tools: `TargetGlobal`, `AccessAdmin`
    - Project list/options/create: `TargetGlobal`, `AccessRead`/`AccessWrite`
    - Project get/delete/changes/tasks: `TargetProject`, `AccessRead`/`AccessWrite`
    - Container tools: `TargetProject`, appropriate access
    - Session tools: `TargetProject`, appropriate access
    - Workspace tools: `TargetProject`, appropriate access
    - Config tools: `TargetProject`, `AccessRead`

## Phase 4: CLI Updates

- [x] Update `cmd/token/main.go`:
  - Accept new scope formats: `admin`, `admin:ro`, `project:<uuid>`, `project:<uuid>:ro`
  - Validate scope format before creating token
  - Update help text with examples

- [x] Update `project_options` MCP tool:
  - Document available scope formats in response
  - Show examples of each scope type

## Phase 5: Testing

- [x] Add unit tests for scope parsing helpers in `internal/auth/types_test.go`

- [x] Add unit tests for `IsToolAllowed` covering:
  - Admin scope can access everything
  - Admin:ro scope can only access read tools
  - Project scope can access own project
  - Project scope cannot access other projects
  - Project:ro scope can only read own project
  - Global tools require admin scope
  - Edge cases (empty scope, malformed scope)

- [x] Add integration tests in `test/pkg/suites/auth.go`:
  - `test_auth_token_create_admin_ro_scope`
  - `test_auth_token_create_project_scope`
  - `test_auth_token_create_project_ro_scope`
  - `test_auth_project_options_shows_token_scopes`

## Phase 6: Documentation

- [x] Update `AGENTS.md`:
  - Document new scope model in "Token Scope Model" section
  - Document "Tool Permission Model" with categories
  - Replace old "Available Tools by Scope" table

- [x] Update `README.md`:
  - Document scope formats in token section with examples and table
