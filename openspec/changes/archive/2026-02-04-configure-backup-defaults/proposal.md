# Configure Backup Defaults

## Problem

The backup system has three issues:
1. Default backup directory is `./backups` at repo root, but should be `data/backups` to consolidate data files
2. Backup configuration (enabled, directory, retention, interval) is hardcoded in `cmd/server/main.go`
3. Backups are enabled by default, but should be opt-in (disabled by default)

## Solution

Add a `backup` section to `config/config-defaults.json` with configurable settings and code-level defaults that apply when not specified.

### Configuration Schema

Add to `config/config-defaults.json`:
```json
{
  "backup": {
    "enabled": false,
    "directory": "data/backups",
    "retention": 7,
    "interval_hours": 24
  }
}
```

### Code-Level Defaults

When `config-defaults.json` doesn't include the `backup` section:
- `enabled`: `false` (backups disabled by default)
- `directory`: `"data/backups"` (relative to cwd)
- `retention`: `7` (keep 7 backups per project)
- `interval_hours`: `24` (daily backups when enabled)

## Changes Required

1. **config/loader.go**: Add `BackupDefaults` struct and loading logic
2. **config/config-defaults.json**: Add `backup` section with defaults
3. **cmd/server/main.go**: Use loaded config instead of hardcoded values
4. **config.yaml.example**: Update documented backup directory path

## Out of Scope

- MCP tools for backup management
- Backup scheduling via cron expressions
- Remote backup storage
