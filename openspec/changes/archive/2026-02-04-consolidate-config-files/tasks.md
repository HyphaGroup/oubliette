# Tasks: Consolidate Config Files

**COMPLETED** - Commit `b16d814`

## Phase 1: JSONC Parser

- [x] Add JSONC parsing support to `internal/config/`
  - Strip `//` and `/* */` comments before JSON parsing

## Phase 2: Unified Config Loader

- [x] Create `UnifiedConfig` struct combining all sections (server, credentials, defaults, models)
- [x] Replace `LoadAll()` to only load `oubliette.jsonc`
- [x] Delete old loaders: `LoadServerConfig`, `LoadCredentials`, `LoadConfigDefaults`, `LoadModels`
- [x] Delete old structs that are now embedded in UnifiedConfig

## Phase 3: Config Path Discovery

- [x] Add `FindConfigPath()` function with precedence:
  1. `--config-dir` flag + `/oubliette.jsonc`
  2. `./config/oubliette.jsonc` (project-local)
  3. `~/.oubliette/config/oubliette.jsonc` (user global)
- [x] Update `cmd/server/main.go` to use `FindConfigPath()`
- [x] Error if no config found (no silent defaults)

## Phase 4: Update Init Command

- [x] Update `oubliette init` to create single `oubliette.jsonc`
- [x] Add JSONC template with comments explaining each section

## Phase 5: Delete Old Config Files

- [x] Delete `config/server.json` and `config/server.json.example`
- [x] Delete `config/credentials.json` and `config/credentials.json.example`
- [x] Delete `config/config-defaults.json` and `config/config-defaults.json.example`
- [x] Delete `config/models.json` and `config/models.json.example`
- [x] Create `config/oubliette.jsonc.example` with documented template

## Phase 6: Update Documentation

- [x] Update `docs/CONFIGURATION.md` with new format
- [x] Update `docs/INSTALLATION.md` with new config structure
- [x] Update `README.md` config references

## Phase 7: Update Tests

- [x] Update `internal/config/*_test.go` for new loader
- [x] Remove tests for deleted loaders
