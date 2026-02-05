# Spec: Unified Credentials

Centralized credential management for Factory, GitHub, and provider API keys.

## ADDED Requirements

### Requirement: Credential File Loading

The system SHALL load credentials from `config/credentials.json` at startup.

#### Scenario: Valid credentials file loaded
- **GIVEN** a valid `config/credentials.json` exists
- **WHEN** the server starts
- **THEN** all credential types (factory, github, providers) are loaded into memory
- **AND** default references are validated to exist

#### Scenario: Missing credentials file
- **GIVEN** `config/credentials.json` does not exist
- **WHEN** the server starts
- **THEN** server fails to start with error: "credentials.json not found, see credentials.json.example"

#### Scenario: Invalid default reference
- **GIVEN** credentials.json has factory.default = "nonexistent"
- **AND** no credential named "nonexistent" exists in factory.credentials
- **WHEN** the server starts
- **THEN** server fails with error: "invalid factory default: credential 'nonexistent' not found"

### Requirement: Factory Credentials

The system SHALL support multiple named Factory API credentials with a default.

#### Scenario: Default factory credential used
- **GIVEN** credentials.json has factory.default = "main"
- **AND** project has no credential_refs.factory
- **WHEN** Droid runtime needs Factory API key
- **THEN** the "main" factory credential is used

#### Scenario: Project-specific factory credential
- **GIVEN** project has credential_refs.factory = "backup"
- **WHEN** Droid runtime needs Factory API key
- **THEN** the "backup" factory credential is used

#### Scenario: Invalid factory credential reference
- **GIVEN** project has credential_refs.factory = "nonexistent"
- **WHEN** project_create is called
- **THEN** error returned: "unknown factory credential: nonexistent"

### Requirement: GitHub Credentials

The system SHALL support multiple named GitHub tokens with a default.

#### Scenario: Default github credential used
- **GIVEN** credentials.json has github.default = "personal"
- **AND** project_create has no credential_refs.github specified
- **WHEN** project is created
- **THEN** the "personal" github token is used for GITHUB_TOKEN env

#### Scenario: Project-specific github credential
- **GIVEN** project_create has credential_refs.github = "orgbot"
- **WHEN** project is created
- **THEN** the "orgbot" github token is used

#### Scenario: Invalid github credential reference
- **GIVEN** project_create has credential_refs.github = "nonexistent"
- **WHEN** project_create is called
- **THEN** error returned: "unknown github credential: nonexistent"

### Requirement: Provider Credentials

The system SHALL support multiple named provider API credentials (Anthropic, OpenAI, etc.) with a default.

#### Scenario: Default provider credential injected to container
- **GIVEN** credentials.json has providers.default = "anthropic-main"
- **AND** "anthropic-main" has provider = "anthropic"
- **AND** project has no credential_refs.provider
- **WHEN** container is created for project
- **THEN** ANTHROPIC_API_KEY env var is set from "anthropic-main" credential

#### Scenario: Project-specific provider credential
- **GIVEN** project has credential_refs.provider = "openai-client-a"
- **AND** "openai-client-a" has provider = "openai"
- **WHEN** container is created for project
- **THEN** OPENAI_API_KEY env var is set from "openai-client-a" credential

#### Scenario: Provider credential with model mismatch
- **GIVEN** project uses model "anthropic/claude-sonnet-4-5"
- **AND** project has credential_refs.provider = "openai-main" (wrong provider)
- **WHEN** container is created
- **THEN** warning logged about provider mismatch
- **AND** credential still injected (user may know what they're doing)

### Requirement: Credential References in Project

Projects SHALL store credential references (names), not raw credentials.

#### Scenario: Credential refs stored in project metadata
- **GIVEN** project_create with credential_refs.provider = "anthropic-main"
- **WHEN** project is created
- **THEN** metadata.json contains credential_refs.provider = "anthropic-main"
- **AND** no API keys are stored in project directory

#### Scenario: Credential rotation affects all projects
- **GIVEN** projects A, B, C all have credential_refs.provider = "anthropic-main"
- **WHEN** "anthropic-main" api_key is updated in credentials.json
- **AND** server is restarted (or config reloaded)
- **THEN** all three projects use the new API key

### Requirement: project_options Shows Available Credentials

The project_options tool SHALL list available credentials without exposing keys.

#### Scenario: List available credentials
- **WHEN** project_options is called
- **THEN** response includes available credential names per type
- **AND** descriptions are included
- **AND** API keys/tokens are NOT included
- **AND** defaults are indicated

## REMOVED Requirements

### Requirement: Separate factory.json File
**Reason**: Consolidated into credentials.json. File deleted.

### Requirement: Separate github-accounts.json File
**Reason**: Consolidated into credentials.json. File deleted.

### Requirement: API Keys in models.json
**Reason**: Keys must be in credentials.json only. apiKey field removed from model definitions.

### Requirement: github_account Parameter
**Reason**: Replaced by credential_refs.github. Old parameter removed entirely.
