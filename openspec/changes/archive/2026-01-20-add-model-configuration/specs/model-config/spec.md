# Model Configuration

Configurable custom models and session defaults for projects.

## ADDED Requirements

### Requirement: Models MUST be configurable in config/models.json

The system SHALL load model definitions from `config/models.json`.

#### Scenario: Load model configuration

- Given a valid `config/models.json` with models "sonnet" and "opus"
- When the server starts
- Then both models are available for project creation
- And API keys are loaded but never exposed via MCP

#### Scenario: Missing model config handled gracefully

- Given no `config/models.json` exists
- When creating a project
- Then project is created with empty customModels
- And a warning is logged

### Requirement: Template settings.json MUST NOT contain secrets

The `template/.factory/settings.json` SHALL not contain API keys or other secrets.

#### Scenario: Template is safe to track in git

- Given `template/.factory/settings.json`
- When inspecting its contents
- Then no API keys are present
- And no sensitive credentials are present
- And customModels array is empty

### Requirement: Settings MUST be patched during project creation

The system SHALL merge model configurations into settings.json when creating projects.

#### Scenario: Models injected into project settings

- Given models "sonnet" and "opus" configured
- And project created with `included_models=["sonnet", "opus"]`
- When project creation completes
- Then project's settings.json has both models in customModels
- And each model has full config including API key

#### Scenario: Session default model set

- Given project created with `session_model="opus"`
- When project creation completes
- Then settings.json sessionDefaultSettings.model is opus's generated ID

### Requirement: Project creation MUST accept model parameters

The `project_create` tool SHALL accept optional model configuration parameters.

#### Scenario: Create project with specific models

- Given models "sonnet", "opus", "gpt5" available
- When calling `project_create` with `included_models=["sonnet"]`
- Then project has only sonnet in customModels

#### Scenario: Create project with default models

- Given defaults include_models=["sonnet", "opus"]
- When calling `project_create` without included_models
- Then project has sonnet and opus in customModels

#### Scenario: Create project with custom session settings

- Given calling `project_create` with `autonomy_mode="manual"` and `reasoning_effort="high"`
- When project creation completes
- Then settings.json has autonomyMode="manual"
- And settings.json has reasoningEffort="high"

#### Scenario: Invalid model name rejected

- Given model "invalid" does not exist
- When calling `project_create` with `included_models=["invalid"]`
- Then request fails with validation error
- And error lists available model names

### Requirement: Models MUST be included in project_options

The `project_options` tool SHALL include available models and defaults.

#### Scenario: Get available models

- Given models "sonnet" and "opus" configured
- When calling `project_options`
- Then response includes `models` section
- And `models.available` lists both models with name, displayName, provider
- And no API keys are included in response

#### Scenario: Get model defaults

- Given defaults session_model="opus"
- When calling `project_options`
- Then `models.defaults.session_model` is "opus"
- And `models.defaults.included_models` lists default models
