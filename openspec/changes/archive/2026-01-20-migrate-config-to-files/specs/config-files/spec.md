# Config Files

JSON-based configuration files replacing .env.

## ADDED Requirements

### Requirement: Server config MUST be loaded from config/server.json

The system SHALL load server runtime configuration from `config/server.json`.

#### Scenario: Load server config

- Given a valid `config/server.json` exists
- When the server starts
- Then the server binds to the configured address
- And uses the configured default model

#### Scenario: Missing server config uses defaults

- Given no `config/server.json` exists
- When the server starts
- Then the server uses default address ":8080"
- And uses default model "claude-sonnet-4-5-20250929"

### Requirement: Factory API key MUST be loaded from config/factory.json

The system SHALL load the Factory API key from `config/factory.json`.

#### Scenario: Load Factory API key

- Given a valid `config/factory.json` with api_key
- When the server starts
- Then the Factory API key is available for droid sessions

#### Scenario: Missing Factory config fails startup

- Given no `config/factory.json` exists
- When the server starts
- Then the server fails with clear error message
- And indicates factory.json is required

### Requirement: Project defaults MUST be loaded from config/project-defaults.json

The system SHALL load default project settings from `config/project-defaults.json`.

#### Scenario: Load project defaults

- Given a valid `config/project-defaults.json`
- When creating a project without explicit limits
- Then the project uses limits from project-defaults.json

#### Scenario: Missing project defaults uses hardcoded defaults

- Given no `config/project-defaults.json` exists
- When creating a project without explicit limits
- Then the project uses hardcoded defaults (depth=3, agents=50, cost=10.00)

### Requirement: Project limits MUST be settable at project creation

The `project_create` tool SHALL accept optional limit parameters.

#### Scenario: Create project with custom limits

- Given project-defaults.json has max_recursion_depth=3
- When calling `project_create` with `max_recursion_depth=5`
- Then the project is created with max_recursion_depth=5
- And other limits use defaults

#### Scenario: Create project with default limits

- Given project-defaults.json has max_cost_usd=10.00
- When calling `project_create` without max_cost_usd
- Then the project is created with max_cost_usd=10.00

### Requirement: Project defaults MUST be included in project_options

The `project_options` tool SHALL include a `defaults` section.

#### Scenario: Get project defaults from project_options

- Given project-defaults.json has max_recursion_depth=3
- When calling `project_options`
- Then response includes `defaults` section
- And `defaults.max_recursion_depth` is 3

## REMOVED Requirements

### Requirement: .env file support MUST be removed

The system SHALL no longer load configuration from .env files or environment variables.

#### Scenario: Env vars are ignored

- Given `FACTORY_API_KEY` is set in environment
- And no `config/factory.json` exists
- When the server starts
- Then the server fails (missing factory.json)
- And the env var is not used
