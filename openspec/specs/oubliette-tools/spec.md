# oubliette-tools Specification

## Purpose
TBD - created by archiving change expose-oubliette-tools. Update Purpose after archive.
## Requirements
### Requirement: Tool Discovery via Socket

When oubliette-client starts with `OUBLIETTE_API_KEY` environment variable set, it SHALL request available tools from the server and register them for Droid access.

#### Scenario: Valid admin API key discovers all tools

- **Given** OUBLIETTE_API_KEY is set to a valid admin-scoped token
- **When** oubliette-client starts and sends `oubliette_tools` request
- **Then** server returns all Oubliette tools (project_*, session_*, token_*, etc.)
- **And** tools are registered with `oubliette_` prefix

#### Scenario: Valid read-only API key discovers limited tools

- **Given** OUBLIETTE_API_KEY is set to a valid read-scoped token
- **When** oubliette-client starts and sends `oubliette_tools` request
- **Then** server returns only read-allowed tools (project_list, project_get, session_list, etc.)
- **And** write/admin tools are not included

#### Scenario: Invalid API key fails discovery

- **Given** OUBLIETTE_API_KEY is set to an invalid token
- **When** oubliette-client starts and sends `oubliette_tools` request
- **Then** server returns an authentication error
- **And** no oubliette tools are registered
- **And** client logs a warning but continues startup

### Requirement: Tool Execution via Socket

Droids SHALL be able to call discovered Oubliette tools, with each call validated against the API key's scope.

#### Scenario: Droid calls project_list with valid key

- **Given** oubliette-client has registered `oubliette_project_list`
- **When** Droid calls `oubliette_project_list`
- **Then** oubliette-client sends `oubliette_call_tool` request with stored API key
- **And** server validates the key and executes project_list
- **And** result is returned to Droid

#### Scenario: Droid calls project_create with write key

- **Given** oubliette-client has registered `oubliette_project_create`
- **And** the API key has write scope
- **When** Droid calls `oubliette_project_create` with name "test-project"
- **Then** server validates the key and executes project_create
- **And** new project is created
- **And** project details are returned to Droid

#### Scenario: Droid calls token_create with insufficient scope

- **Given** oubliette-client has registered `oubliette_token_create`
- **And** the API key has write scope (not admin)
- **When** Droid calls `oubliette_token_create`
- **Then** server returns scope error
- **And** Droid receives error indicating insufficient permissions

#### Scenario: Tool call with expired key fails

- **Given** oubliette-client has registered oubliette tools
- **And** the API key has expired since startup
- **When** Droid calls any oubliette tool
- **Then** server validates the key and returns authentication error
- **And** Droid receives error indicating invalid key

### Requirement: Tool Naming Convention

All Oubliette tools exposed to Droids SHALL be prefixed with `oubliette_` to avoid collision with other tool sources.

#### Scenario: Tools use oubliette_ prefix

- **Given** server returns tool "project_create" in discovery
- **When** oubliette-client registers the tool
- **Then** it is registered as "oubliette_project_create"
- **And** Droid sees "oubliette_project_create" in tools/list

### Requirement: No API Key, No Tools

If OUBLIETTE_API_KEY is not set, the system SHALL NOT expose any Oubliette tools.

#### Scenario: Missing API key skips tool registration

- **Given** OUBLIETTE_API_KEY environment variable is not set
- **When** oubliette-client starts
- **Then** no `oubliette_tools` request is sent
- **And** no oubliette_* tools are registered
- **And** only caller tools and base tools are available

