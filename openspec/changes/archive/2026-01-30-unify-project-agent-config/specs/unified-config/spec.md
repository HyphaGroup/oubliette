# unified-config Specification

## Purpose

Define the canonical project configuration format and runtime config generation system that enables seamless switching between Droid and OpenCode agent runtimes.

## ADDED Requirements

### Requirement: Canonical Project Config File

Each project SHALL have a `config.json` file at its root containing all project and agent settings as the single source of truth.

The config file SHALL contain:
- Project identity (id, name, description, created_at, default_workspace_id)
- Container settings (type, image_name)
- Agent settings (runtime, model, autonomy, reasoning, mcp_servers, permissions)
- Resource limits (max_recursion_depth, max_agents_per_session, max_cost_usd)
- Isolation settings (workspace_isolation, protected_paths)

#### Scenario: New project has config.json
- **GIVEN** a request to create a new project
- **WHEN** the project is created successfully
- **THEN** `projects/<id>/config.json` exists
- **AND** it contains valid JSON matching the canonical schema
- **AND** all required fields are populated (id, name, created_at, default_workspace_id)

#### Scenario: Config includes agent settings
- **GIVEN** a project with `config.json`
- **WHEN** reading the config file
- **THEN** `agent.runtime` specifies the runtime (droid or opencode)
- **AND** `agent.model` specifies the model ID
- **AND** `agent.autonomy` specifies the autonomy level (off, low, medium, high)
- **AND** `agent.mcp_servers` contains MCP server definitions

### Requirement: Runtime Config Generation

The system SHALL generate runtime-specific config files for BOTH runtimes at project creation time, regardless of which runtime is selected.

Generated files:
- `.factory/mcp.json` - Droid MCP server configuration
- `.factory/settings.json` - Droid session settings
- `opencode.json` - OpenCode configuration

#### Scenario: Both runtime configs generated on project create
- **GIVEN** a request to create a project with `agent_runtime: "droid"`
- **WHEN** the project is created
- **THEN** `projects/<id>/.factory/mcp.json` exists with Droid-format MCP config
- **AND** `projects/<id>/.factory/settings.json` exists with Droid settings
- **AND** `projects/<id>/.factory/droids/` directory exists for custom droid definitions
- **AND** `projects/<id>/.factory/skills/` directory exists for Droid skill definitions
- **AND** `projects/<id>/opencode.json` exists with OpenCode-format config
- **AND** `projects/<id>/.opencode/agents/` directory exists for OpenCode agent definitions
- **AND** `projects/<id>/.opencode/skills/` directory exists for OpenCode skill definitions

#### Scenario: Runtime configs reflect canonical settings
- **GIVEN** a project with `agent.model: "claude-sonnet-4-5-20250929"` and `agent.autonomy: "high"`
- **WHEN** examining the generated runtime configs
- **THEN** `.factory/settings.json` contains the model in Droid format
- **AND** `opencode.json` contains `model: "anthropic/claude-sonnet-4-5-20250929"`
- **AND** `opencode.json` contains `permission` config equivalent to high autonomy

### Requirement: MCP Server Schema Translation

The system SHALL translate MCP server definitions between canonical format and runtime-specific formats.

| Canonical | Droid | OpenCode |
|-----------|-------|----------|
| `type: "stdio"` | `type: "stdio"` | `type: "local"` |
| `type: "http"` | `type: "http"` | `type: "remote"` |
| `command` + `args` | `command` + `args` | `command: [array]` |
| `disabled: true` | `disabled: true` | `enabled: false` |
| `env` | `env` | `environment` |

#### Scenario: Stdio server translated to Droid format
- **GIVEN** a canonical MCP server with `type: "stdio"`, `command: "/usr/bin/server"`, `args: ["--port", "8080"]`
- **WHEN** generating Droid config
- **THEN** `.factory/mcp.json` contains the server with `type: "stdio"`, `command: "/usr/bin/server"`, `args: ["--port", "8080"]`

#### Scenario: Stdio server translated to OpenCode format
- **GIVEN** a canonical MCP server with `type: "stdio"`, `command: "/usr/bin/server"`, `args: ["--port", "8080"]`
- **WHEN** generating OpenCode config
- **THEN** `opencode.json` contains the server with `type: "local"`, `command: ["/usr/bin/server", "--port", "8080"]`

#### Scenario: HTTP server translated to OpenCode format
- **GIVEN** a canonical MCP server with `type: "http"`, `url: "https://mcp.example.com"`
- **WHEN** generating OpenCode config
- **THEN** `opencode.json` contains the server with `type: "remote"`, `url: "https://mcp.example.com"`

### Requirement: Model Format Translation

The system SHALL translate model identifiers between canonical format (bare model ID) and OpenCode format (provider/model).

#### Scenario: Claude model translated to OpenCode format
- **GIVEN** `agent.model: "claude-sonnet-4-5-20250929"`
- **WHEN** generating OpenCode config
- **THEN** `opencode.json` contains `model: "anthropic/claude-sonnet-4-5-20250929"`

#### Scenario: GPT model translated to OpenCode format
- **GIVEN** `agent.model: "gpt-5.1"`
- **WHEN** generating OpenCode config
- **THEN** `opencode.json` contains `model: "openai/gpt-5.1"`

#### Scenario: Gemini model translated to OpenCode format
- **GIVEN** `agent.model: "gemini-3-pro-preview"`
- **WHEN** generating OpenCode config
- **THEN** `opencode.json` contains `model: "google/gemini-3-pro-preview"`

### Requirement: Autonomy to Permission Translation

The system SHALL translate autonomy levels to runtime-specific permission configurations.

| Autonomy | Droid CLI | OpenCode Permission |
|----------|-----------|---------------------|
| `off` | `--skip-permissions-unsafe` | `"permission": "allow"` |
| `high` | `--auto high` | `{"*": "allow", "external_directory": "ask", "doom_loop": "ask"}` |
| `medium` | `--auto medium` | `{"read": "allow", "edit": "allow", "bash": {"*": "ask", "git *": "allow"}}` |
| `low` | `--auto low` | `{"read": "allow", "edit": "ask", "bash": "ask"}` |

#### Scenario: Off autonomy translated to unrestricted

### Requirement: Reasoning Effort Translation

The system SHALL translate reasoning effort levels to runtime-specific configurations.

| Reasoning | Droid CLI | OpenCode (Anthropic) | OpenCode (OpenAI) | OpenCode (Google) |
|-----------|-----------|---------------------|-------------------|-------------------|
| `off` | `-r off` | No thinking block | `reasoningEffort: "none"` | `variant: "low"` |
| `low` | `-r low` | `thinking.budgetTokens: 4000` | `reasoningEffort: "low"` | `variant: "low"` |
| `medium` | `-r medium` | `thinking.budgetTokens: 16000` | `reasoningEffort: "medium"` | `variant: "high"` |
| `high` | `-r high` | `thinking.budgetTokens: 32000` | `reasoningEffort: "high"` | `variant: "high"` |

Default: `medium`

#### Scenario: Medium reasoning translated to Droid
- **GIVEN** `agent.reasoning: "medium"`
- **WHEN** generating Droid command
- **THEN** the command includes `-r medium`

#### Scenario: Medium reasoning translated to OpenCode (Anthropic)
- **GIVEN** `agent.reasoning: "medium"` and `agent.model: "claude-sonnet-4-5-20250929"`
- **WHEN** generating OpenCode config
- **THEN** model options include `thinking.type: "enabled"` and `thinking.budgetTokens: 16000`

#### Scenario: High reasoning translated to OpenCode (OpenAI)
- **GIVEN** `agent.reasoning: "high"` and `agent.model: "gpt-5.1"`
- **WHEN** generating OpenCode config
- **THEN** model options include `reasoningEffort: "high"`

#### Scenario: Off reasoning disables thinking
- **GIVEN** `agent.reasoning: "off"`
- **WHEN** generating Droid command
- **THEN** the command includes `-r off`
- **WHEN** generating OpenCode config for Anthropic model
- **THEN** no thinking block is included in model options
- **GIVEN** `agent.autonomy: "off"`
- **WHEN** generating Droid command
- **THEN** the command includes `--skip-permissions-unsafe`
- **WHEN** generating OpenCode config
- **THEN** `opencode.json` contains `"permission": "allow"` (string, not object)

#### Scenario: High autonomy translated to permissions
- **GIVEN** `agent.autonomy: "high"`
- **WHEN** generating Droid command
- **THEN** the command includes `--auto high`
- **WHEN** generating OpenCode config
- **THEN** `opencode.json` contains `permission` with `"*": "allow"`
- **AND** `permission` includes `"external_directory": "ask"`

#### Scenario: Low autonomy translated to permissions
- **GIVEN** `agent.autonomy: "low"`
- **WHEN** generating Droid command
- **THEN** the command includes `--auto low`
- **WHEN** generating OpenCode config
- **THEN** `opencode.json` contains `permission` with `"read": "allow"`
- **AND** `permission` includes `"edit": "ask"`
- **AND** `permission` includes `"bash": "ask"`

### Requirement: Server Defaults for Agent Config

The server SHALL provide default agent configuration values in `config/config-defaults.json`.

Defaults SHALL include:
- `agent.runtime` - Default runtime (droid)
- `agent.model` - Default model
- `agent.autonomy` - Default autonomy level (off - unrestricted, since agents run in isolated containers)
- `agent.reasoning` - Default reasoning level (medium)
- `agent.mcp_servers` - Default MCP servers including oubliette-parent

#### Scenario: Project uses server defaults
- **GIVEN** a `project_create` request with no agent settings specified
- **WHEN** the project is created
- **THEN** `config.json` contains agent settings from `config/config-defaults.json`

#### Scenario: Project overrides server defaults
- **GIVEN** a `project_create` request with `autonomy: "low"`
- **WHEN** the project is created
- **THEN** `config.json` contains `agent.autonomy: "low"`
- **AND** other agent settings use server defaults

### Requirement: Read-Only Config Mounts in Containers

Config files SHALL be mounted read-only in containers to prevent agents from modifying their own configuration.

Read-only mounts:
- `config.json` - Canonical project config
- `opencode.json` - OpenCode runtime config
- `.factory/mcp.json` - Droid MCP server config
- `.factory/settings.json` - Droid session settings

Writable mounts (agent can modify):
- `.factory/commands/` - Custom slash commands
- `.factory/hooks/` - Lifecycle hooks
- `workspaces/<uuid>/` - Agent working directory

#### Scenario: Agent cannot modify config.json
- **GIVEN** a running container with an active agent session
- **WHEN** the agent attempts to write to `/workspace/config.json`
- **THEN** the write operation fails with a read-only filesystem error

#### Scenario: Agent cannot modify opencode.json
- **GIVEN** a running container with an active agent session
- **WHEN** the agent attempts to write to `/workspace/opencode.json`
- **THEN** the write operation fails with a read-only filesystem error

#### Scenario: Agent cannot modify mcp.json
- **GIVEN** a running container with an active agent session
- **WHEN** the agent attempts to write to `/workspace/.factory/mcp.json`
- **THEN** the write operation fails with a read-only filesystem error

#### Scenario: Agent can modify commands
- **GIVEN** a running container with an active agent session
- **WHEN** the agent creates a file in `/workspace/.factory/commands/`
- **THEN** the file is created successfully

## MODIFIED Requirements

### Requirement: Project Creation Includes Agent Config

**Modifies**: Existing project_create behavior

The `project_create` MCP tool SHALL accept agent configuration parameters:
- `agent_runtime` - Runtime selection (droid, opencode)
- `model` - Model identifier
- `autonomy` - Autonomy level (low, medium, high)
- `reasoning` - Reasoning effort (off, low, medium, high)
- `mcp_servers` - Additional MCP servers to include
- `permissions` - Custom permission overrides (OpenCode format)
- `disabled_tools` - List of tools to disable

#### Scenario: Create project with custom agent config
- **GIVEN** a `project_create` request with `model: "claude-opus-4-5-20251101"` and `autonomy: "low"`
- **WHEN** the project is created
- **THEN** `config.json` contains the specified model and autonomy
- **AND** generated runtime configs reflect these settings

## REMOVED Requirements

### Requirement: IncludedModels and SessionModel Parameters

**Removes**: The `included_models` and `session_model` parameters from `project_create`

These are replaced by the simpler `model` parameter. Model variants and multiple model support can be added in a future enhancement if needed.

#### Scenario: Legacy parameters rejected
- **GIVEN** a `project_create` request with `included_models: ["sonnet", "opus"]`
- **WHEN** the request is processed
- **THEN** the parameter is ignored or returns an error indicating deprecation
