# agent-runtime Spec Delta

## ADDED Requirements

### Requirement: Model Extra Headers

Model definitions in `oubliette.jsonc` SHALL support an `extraHeaders` field containing HTTP headers to send with API requests for that model.

#### Scenario: Droid receives extra headers via customModels

- **GIVEN** a model definition with `extraHeaders: {"anthropic-beta": "context-1m-2025-08-07"}`
- **WHEN** the project config is written to `.factory/settings.json`
- **THEN** the model appears in `customModels` with an `extraHeaders` field containing the same headers

#### Scenario: OpenCode receives extra headers via provider model config

- **GIVEN** a model definition with `extraHeaders: {"anthropic-beta": "context-1m-2025-08-07"}`
- **WHEN** the project config is written to `opencode.json`
- **THEN** the model appears under `provider.<providerID>.models.<modelID>` with a `headers` field containing the same headers

#### Scenario: Models without extra headers are unaffected

- **GIVEN** a model definition with no `extraHeaders` field
- **WHEN** the project config is written for either runtime
- **THEN** no headers are added and the config matches prior behavior

## MODIFIED Requirements

### Requirement: Runtime Package File Structure

Each agent runtime config translator SHALL accept a model registry to resolve extra headers and other model-level configuration when generating runtime-specific config files.

#### Scenario: Config translators receive model registry

- **GIVEN** a project with a model that has `extraHeaders`
- **WHEN** `writeProjectConfigs` generates runtime-specific configs
- **THEN** it passes the model registry to both `ToDroidSettings` and `ToOpenCodeConfig`
- **AND** the generated configs include the extra headers for the active model
