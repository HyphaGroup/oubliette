# Backup System

Oubliette includes automated backup functionality for project data.

## Configuration

Backups are configured in `config/config-defaults.json`:

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

| Setting | Default | Description |
|---------|---------|-------------|
| `enabled` | `false` | Enable/disable automatic backups |
| `directory` | `data/backups` | Backup storage location (relative or absolute) |
| `retention` | `7` | Number of backups to keep per project |
| `interval_hours` | `24` | Hours between automatic backups |

## What Gets Backed Up

Each project backup includes:
- Project metadata (`metadata.json`)
- Project configuration files
- Session data and indexes

**Excluded**: Workspace directories are skipped (only metadata is backed up, not full workspace contents).

## Backup Format

Backups are stored as gzip-compressed tar archives:

```
{project_id}_{YYYYMMDD}_{HHMMSS}.tar.gz
```

Example: `1f5ef945-91c3-483f-8d61-427389bb1291_20260131_155522.tar.gz`

## Retention Policy

When a new backup is created, the system automatically removes old backups exceeding the retention limit. Only the most recent N backups (per project) are kept, where N is the `retention` setting.

## Manual Operations

### List Backups

```bash
ls -la data/backups/
```

### Verify Backup Integrity

```bash
tar -tzf data/backups/<project>_<date>.tar.gz
```

### Manual Restore

```bash
# Stop server first
tar -xzf data/backups/<project>_<date>.tar.gz -C data/projects/
```

## Enabling Backups

To enable automatic backups, edit `config/config-defaults.json`:

```json
{
  "backup": {
    "enabled": true
  }
}
```

Then restart the server. Backups will run at the configured interval starting from server startup time.

## Implementation

The backup system is implemented in `internal/backup/backup.go`. Key components:

- `Manager` - Handles backup scheduling and execution
- `BackupProject()` - Creates a single project backup
- `BackupAll()` - Backs up all valid projects
- `RestoreProject()` - Restores from a backup file
- `ListSnapshots()` - Lists available backups

The manager runs as a background goroutine when enabled, using a ticker at the configured interval.
