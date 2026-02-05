# backup-config Specification

Configurable backup automation settings loaded from config-defaults.json.

## ADDED Requirements

### Requirement: Backup Configuration Loading

The system SHALL load backup configuration from `config/config-defaults.json` with code-level defaults when not specified.

#### Scenario: Backup section present in config
- **GIVEN** `config/config-defaults.json` contains a `backup` section
- **WHEN** the server starts
- **THEN** backup settings are loaded from the config file
- **AND** missing fields use code-level defaults

#### Scenario: Backup section missing from config
- **GIVEN** `config/config-defaults.json` does not contain a `backup` section
- **WHEN** the server starts
- **THEN** all backup settings use code-level defaults

#### Scenario: Code-level defaults applied
- **GIVEN** backup configuration is not specified
- **WHEN** defaults are applied
- **THEN** `enabled` defaults to `false`
- **AND** `directory` defaults to `"data/backups"`
- **AND** `retention` defaults to `7`
- **AND** `interval_hours` defaults to `24`

### Requirement: Backup Enabled Toggle

The system SHALL only start backup automation when `backup.enabled` is `true`.

#### Scenario: Backups disabled by default
- **GIVEN** `backup.enabled` is not set or is `false`
- **WHEN** the server starts
- **THEN** backup automation does NOT start
- **AND** no backup files are created

#### Scenario: Backups enabled explicitly
- **GIVEN** `backup.enabled` is `true` in config
- **WHEN** the server starts
- **THEN** backup automation starts
- **AND** backups are created at the configured interval

### Requirement: Backup Directory Configuration

The system SHALL use `data/backups` as the default backup directory.

#### Scenario: Default backup directory
- **GIVEN** `backup.directory` is not set
- **WHEN** backup automation runs
- **THEN** backups are stored in `data/backups` relative to cwd

#### Scenario: Custom backup directory
- **GIVEN** `backup.directory` is set to `"/var/backups/oubliette"`
- **WHEN** backup automation runs
- **THEN** backups are stored in the specified absolute path
