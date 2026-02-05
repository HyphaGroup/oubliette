# Tasks

- [x] 1. Add `BackupDefaults` struct to `internal/config/loader.go` with fields: `Enabled`, `Directory`, `Retention`, `IntervalHours`
- [x] 2. Add `Backup BackupDefaults` field to `ConfigDefaultsConfig` struct
- [x] 3. Add `DefaultBackupDefaults()` function returning enabled=false, directory="data/backups", retention=7, interval_hours=24
- [x] 4. Update `LoadConfigDefaults()` to apply backup defaults for zero/missing values
- [x] 5. Update `config/config-defaults.json` to include backup section with enabled=false
- [x] 6. Update `cmd/server/main.go` to read backup config from `cfg.ConfigDefaults.Backup` instead of hardcoded values
- [x] 7. Update `config.yaml.example` to document `data/backups` as the default directory
- [x] 8. Build and verify: `./build.sh`
