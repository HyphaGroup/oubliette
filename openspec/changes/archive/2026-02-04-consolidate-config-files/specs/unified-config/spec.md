# Unified Config Specification

## ADDED Requirements

### Requirement: System loads configuration from single oubliette.jsonc file

The system SHALL load all configuration from a single `oubliette.jsonc` file in JSONC format (JSON with Comments), containing server, credentials, defaults, and models sections.

#### Scenario: Load unified config from project directory

- Given ./config/oubliette.jsonc exists with valid JSONC content
- When the server starts
- Then configuration is loaded from ./config/oubliette.jsonc
- And all sections (server, credentials, defaults, models) are parsed

#### Scenario: Load unified config from user directory

- Given ./config/oubliette.jsonc does not exist
- And ~/.oubliette/config/oubliette.jsonc exists
- When the server starts
- Then configuration is loaded from ~/.oubliette/config/oubliette.jsonc

#### Scenario: JSONC comments are ignored

- Given oubliette.jsonc contains // and /* */ comments
- When the config is parsed
- Then comments are stripped
- And valid JSON content is parsed successfully

#### Scenario: Error when no config found

- Given no oubliette.jsonc exists in any location
- When the server starts
- Then server exits with error
- And message indicates config file not found

### Requirement: Config discovery follows precedence order

The system SHALL discover configuration with the following precedence (first found wins):
1. Path specified by `--config-dir` flag + `/oubliette.jsonc`
2. `./config/oubliette.jsonc` (project-local)
3. `~/.oubliette/config/oubliette.jsonc` (user global)

#### Scenario: Explicit config-dir flag takes precedence

- Given --config-dir /custom/path is specified
- And /custom/path/oubliette.jsonc exists
- When the server starts
- Then configuration is loaded from /custom/path/oubliette.jsonc

#### Scenario: Project-local config takes precedence over user config

- Given ./config/oubliette.jsonc exists
- And ~/.oubliette/config/oubliette.jsonc exists
- When the server starts without --config-dir
- Then configuration is loaded from ./config/oubliette.jsonc

### Requirement: Init command creates unified config

The `oubliette init` command SHALL create a single `oubliette.jsonc` file with inline JSONC comments documenting each configuration option.

#### Scenario: Init creates oubliette.jsonc

- Given oubliette init is run
- When initialization completes
- Then ~/.oubliette/config/oubliette.jsonc is created
- And the file contains all config sections with default values
- And the file contains JSONC comments explaining options

## REMOVED Requirements

### Requirement: Individual JSON config files

The system SHALL NOT support individual JSON config files (server.json, credentials.json, config-defaults.json, models.json). Only oubliette.jsonc is supported.

**Reason**: Simplify configuration to single file with comments.

**Migration**: Manually merge existing JSON files into oubliette.jsonc format.

#### Scenario: Old config files are not loaded

- Given ./config/server.json exists
- And ./config/oubliette.jsonc does not exist
- When the server starts
- Then server exits with error
- And message indicates oubliette.jsonc not found
