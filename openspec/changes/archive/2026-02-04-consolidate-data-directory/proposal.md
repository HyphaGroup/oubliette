# Consolidate Data Directory

## Problem

Runtime state is scattered across multiple top-level directories:
- `data/` - databases (auth.db, schedules.db), backups
- `projects/` - project state
- `logs/` - server logs

## Solution

Move everything into `data/`:

```
data/
├── projects/
├── logs/
├── backups/
├── auth.db
└── schedules.db
```

## Benefits

- Single directory for all runtime state
- Simpler backups (one directory)
- Cleaner repo root
- Easier volume mounting in production

## Changes Required

1. `cmd/server/main.go` - Update default paths
2. `config.yaml.example` - Update documented paths
3. Documentation updates
