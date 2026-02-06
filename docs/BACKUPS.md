# Backup System

Automated backup for project data, configured in `oubliette.jsonc`:

```jsonc
"defaults": {
  "backup": {
    "enabled": false,
    "directory": "data/backups",
    "retention": 7,
    "interval_hours": 24
  }
}
```

## What Gets Backed Up

Per-project: metadata, configuration, session data. Workspace file contents are excluded.

## Backup Format

```
{project_id}_{YYYYMMDD}_{HHMMSS}.tar.gz
```

## Retention

Old backups exceeding the retention limit are automatically removed per project.

## Manual Operations

```bash
ls -la data/backups/                                    # List
tar -tzf data/backups/<file>.tar.gz                     # Verify
tar -xzf data/backups/<file>.tar.gz -C data/projects/   # Restore (stop server first)
```

## Implementation

`internal/backup/backup.go` — runs as background goroutine at the configured interval.
