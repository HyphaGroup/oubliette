# Tasks: Migrate Config to Files

**Status**: COMPLETE - All 21 tasks done

## Config File Structure

- [x] 1. Create `config/` directory and gitignore entries
  - Add `config/factory.json` to `.gitignore`
  - `config/server.json` and `config/project-defaults.json` tracked

- [x] 2. Create `config/server.json.example`
  - `address`: server bind address
  - `droid.default_model`: default model for sessions

- [x] 3. Create `config/factory.json.example`
  - `api_key`: placeholder for Factory API key

- [x] 4. Create `config/project-defaults.json.example`
  - `max_recursion_depth`: default 3
  - `max_agents_per_session`: default 50
  - `max_cost_usd`: default 10.00
  - `container_type`: default "dev"

## Config Loading

- [x] 5. Create `internal/config/loader.go`
  - `LoadServerConfig(path)` - load server.json
  - `LoadFactoryConfig(path)` - load factory.json
  - `LoadProjectDefaults(path)` - load project-defaults.json
  - Handle missing files gracefully with sensible defaults

- [x] 6. Update server startup to use new config loading
  - Load all config files from `config/` directory
  - Remove viper/env var usage
  - Pass loaded config to managers

## Remove .env Support

- [x] 7. Remove env var bindings from `internal/config/config.go`
  - Remove `FACTORY_API_KEY` binding
  - Remove `SERVER_ADDR` binding
  - Remove `DROID_DEFAULT_MODEL` binding
  - Remove `DEFAULT_MAX_*` bindings
  - Remove `DEFAULT_GITHUB_TOKEN` binding (if not already done)

- [x] 8. Delete `.env.example` file

- [x] 9. Update `cmd/server/main.go`
  - Use new config loader instead of viper
  - Pass config to MCP server and managers

## Project Creation Updates

- [x] 10. Add project limit parameters to `project_create` MCP tool
  - `max_recursion_depth` (optional)
  - `max_agents_per_session` (optional)
  - `max_cost_usd` (optional)

- [x] 11. Update project creation to use defaults
  - Load defaults from project-defaults.json
  - Override with explicit parameters if provided

- [x] 12. Add `defaults` section to `project_options` response
  - Include all project default values
  - Callers can see what will be used if not specified

## Manager Script Updates

- [x] 13. Update `manager.sh create` to handle new config structure
  - Prompt for Factory API key and create `config/factory.json`
  - Copy `config/*.example` to instance config directory
  - Remove .env file creation

- [x] 14. Update `manager.sh` config handling
  - Read server config from `config/server.json` (port, etc.)
  - Instance-specific overrides in `instances/<name>/config/`

- [x] 15. Add `manager.sh init-config` command
  - Interactive setup for config files
  - Create factory.json with prompted API key
  - Create github-accounts.json with prompted accounts

## Testing

- [x] 16. Add unit tests for config loading
  - Valid JSON parsing
  - Missing file handling (defaults)
  - Invalid JSON error handling

- [x] 17. Update integration tests
  - Remove .env dependencies
  - Use config files instead

- [x] 18. Test manager.sh with new config structure
  - Test create command
  - Test init-config command

## Documentation

- [x] 19. Update AGENTS.md with new config structure
- [x] 20. Update README.md
  - Remove .env instructions
  - Add config/ setup instructions
- [x] 21. Update docs/INSTANCE_MANAGER.md
  - Document config file handling
  - Document init-config command
