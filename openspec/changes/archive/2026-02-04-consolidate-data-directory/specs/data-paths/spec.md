# data-paths Specification

All runtime state consolidated under the `data/` directory.

## ADDED Requirements

### Requirement: Consolidated Data Directory

The system SHALL store all runtime state under `data/`.

#### Scenario: Default directory structure
- **GIVEN** the server starts with default configuration
- **WHEN** runtime directories are created
- **THEN** projects are stored in `data/projects/`
- **AND** logs are stored in `data/logs/`
- **AND** backups are stored in `data/backups/`
- **AND** databases are stored in `data/` (auth.db, schedules.db)

#### Scenario: Single data directory for state
- **GIVEN** default configuration
- **WHEN** checking runtime state location
- **THEN** all mutable state is under `data/`
- **AND** repo root contains only code, config, and docs
