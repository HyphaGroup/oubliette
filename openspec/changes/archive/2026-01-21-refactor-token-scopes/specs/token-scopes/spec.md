# Token Scopes

Token-based authentication with granular access control for MCP tools.

## MODIFIED Requirements

### Requirement: Token Scope Formats

The system SHALL support the following token scope formats:
- `admin` - Full access to all tools and projects
- `admin:ro` - Read-only access to all tools and projects
- `project:<uuid>` - Full access to a single project
- `project:<uuid>:ro` - Read-only access to a single project

The system SHALL validate scope format on token creation.

#### Scenario: Admin scope has full access
- Given: A token with scope "admin"
- When: Token attempts to call any MCP tool
- Then: The call is permitted

#### Scenario: Admin read-only scope can only read
- Given: A token with scope "admin:ro"
- When: Token attempts to call a write tool like "project_delete"
- Then: The call is denied
- When: Token attempts to call a read tool like "project_get"
- Then: The call is permitted

#### Scenario: Project scope can access own project
- Given: A token with scope "project:proj-123"
- When: Token attempts to call "project_get" with project_id "proj-123"
- Then: The call is permitted

#### Scenario: Project scope cannot access other projects
- Given: A token with scope "project:proj-123"
- When: Token attempts to call "project_get" with project_id "proj-456"
- Then: The call is denied

#### Scenario: Project read-only scope cannot write
- Given: A token with scope "project:proj-123:ro"
- When: Token attempts to call "session_spawn" with project_id "proj-123"
- Then: The call is denied
- When: Token attempts to call "session_list" with project_id "proj-123"
- Then: The call is permitted

### Requirement: Tool Permission Model

Each tool SHALL declare its target and required access level:
- **Target**: `global` (system-wide) or `project` (operates on a project)
- **Access**: `read`, `write`, or `admin` (admin-only tools like token management)

The system SHALL enforce these permissions on every tool call.

#### Scenario: Admin-only tools require admin scope
- Given: A token with scope "admin:ro"
- When: Token attempts to call "token_create"
- Then: The call is denied because token tools require full admin access

#### Scenario: Global tools require admin scope
- Given: A token with scope "project:proj-123"
- When: Token attempts to call "project_list" (global tool)
- Then: The call is denied because global tools require admin scope

#### Scenario: Project tools check project match
- Given: A token with scope "project:proj-123"
- When: Token attempts to call "container_logs" with project_id "proj-123"
- Then: The call is permitted
- When: Token attempts to call "container_logs" with project_id "proj-456"
- Then: The call is denied
