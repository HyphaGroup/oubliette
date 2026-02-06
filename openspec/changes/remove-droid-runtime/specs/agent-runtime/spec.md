# agent-runtime Spec Delta

## REMOVED Requirements

### Requirement: Multi-Runtime Factory

Removed -- single runtime, no factory pattern needed.

### Requirement: Factory API Key Credentials

Removed -- no Factory Droid, no FACTORY_API_KEY.

### Requirement: Droid Config Generation

Removed -- no `.factory/` scaffolding, no `settings.json`/`mcp.json` generation.

### Requirement: Static Reasoning Config

Removed -- reasoning is passed per-message via variant, not baked into `opencode.json`.

## MODIFIED Requirements

### Requirement: Runtime Interface

The `agent.Runtime` interface SHALL be simplified to only execution and lifecycle methods. With a single runtime that requires no initialization or availability checking, the interface focuses on what matters.

#### Scenario: Simplified interface
- **GIVEN** the `agent.Runtime` interface
- **WHEN** examining its methods
- **THEN** it has `ExecuteStreaming`, `Execute`, `Ping`, `Close`
- **AND** it does NOT have `Initialize`, `Name`, `IsAvailable`

### Requirement: Session Continuation

Sessions SHALL use `RuntimeSessionID` (not `DroidSessionID`) for continuation across turns.

#### Scenario: Session stores runtime session ID
- **GIVEN** a new streaming session is started
- **WHEN** the executor returns a session ID
- **THEN** it is stored as `RuntimeSessionID`
- **AND** subsequent messages to the same session use this ID

### Requirement: OpenCode Config Generation

Project creation SHALL generate `opencode.json` with model, permissions, tools, and MCP config. It SHALL NOT include provider-specific reasoning configuration.

#### Scenario: Generated config has no reasoning
- **GIVEN** a project with `reasoning: "high"`
- **WHEN** `opencode.json` is generated
- **THEN** it does NOT contain a `provider` section with reasoning options
- **AND** reasoning is handled per-message via variant parameter

## ADDED Requirements

### Requirement: Single Runtime Architecture

Oubliette SHALL use OpenCode as its sole agent runtime. No runtime selection, factory, or auto-detection logic SHALL exist.

#### Scenario: Server initializes only OpenCode runtime
- **GIVEN** the oubliette server starts
- **WHEN** initializing the agent runtime
- **THEN** only the OpenCode runtime is created
- **AND** no runtime factory or selection logic exists
- **AND** no Factory API key check occurs

#### Scenario: No Droid code in codebase
- **GIVEN** the oubliette codebase
- **WHEN** searching for Droid-specific code
- **THEN** no `internal/agent/droid/` directory exists
- **AND** no `FACTORY_API_KEY` references exist in runtime code
- **AND** no `.factory/` directory scaffolding exists in project creation
- **AND** no `template/.factory/` directory exists

### Requirement: Per-Message Reasoning via Variant

Reasoning level SHALL be passed per-message as OpenCode's `variant` parameter in `prompt_async`, not baked into static config.

#### Scenario: Reasoning passed as variant
- **GIVEN** an `ExecuteRequest` with `ReasoningLevel: "high"`
- **WHEN** `SendMessageAsync` is called
- **THEN** the HTTP body includes `"variant": "high"`
- **AND** OpenCode's `ProviderTransform.variants()` handles provider-specific translation

#### Scenario: No reasoning means no variant
- **GIVEN** an `ExecuteRequest` with `ReasoningLevel: "off"` or empty
- **WHEN** `SendMessageAsync` is called
- **THEN** the HTTP body does NOT include a `variant` field

### Requirement: Session Abort

The streaming executor SHALL support canceling the current operation via OpenCode's abort endpoint.

#### Scenario: Cancel calls abort
- **GIVEN** an active streaming session
- **WHEN** `Cancel()` is called on the executor
- **THEN** `POST /session/:id/abort` is sent to the OpenCode server

### Requirement: No Factory Credentials

The credential system SHALL NOT include Factory-specific credential types.

#### Scenario: No factory credential section
- **GIVEN** the `oubliette.jsonc` configuration
- **WHEN** parsing credentials
- **THEN** no `credentials.factory` section exists
- **AND** no `GetDefaultFactoryKey` method exists
- **AND** `CredentialRefs` has no `Factory` field
