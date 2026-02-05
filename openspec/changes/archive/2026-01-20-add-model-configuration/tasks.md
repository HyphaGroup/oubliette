# Tasks: Add Model Configuration

## Config Structure

- [x] 1. Create `config/models.json.example`
  - Model definitions with placeholder API keys
  - Default included models, session model
  - Default autonomy mode, reasoning effort

- [x] 2. Add `config/models.json` to `.gitignore` (already present)

- [x] 3. Create model config loader
  - `internal/config/models.go`
  - Struct for model definition (model, displayName, baseUrl, apiKey, etc.)
  - Struct for model registry (models map, defaults)
  - Load from `config/models.json`

## Template Cleanup

- [x] 4. Clean `template/.factory/settings.json` (already clean - no API keys or hardcoded models)

## Settings Patching

- [x] 5. Create settings patcher
  - `internal/project/settings.go`
  - Load template settings.json
  - Merge in selected models with full config (including API keys)
  - Set session defaults (model ID, autonomy, reasoning)
  - Generate unique model IDs (custom:DisplayName-index)

- [x] 6. Update project creation to use settings patcher
  - Load model config
  - Apply defaults or MCP-provided overrides
  - Write patched settings.json to project/.factory/

## MCP Tool Updates

- [x] 7. Add model parameters to `project_create`
  - `included_models` - array of model names (optional)
  - `session_model` - session default model name (optional)
  - `autonomy_mode` - autonomy setting (optional)
  - `reasoning_effort` - reasoning setting (optional)

- [x] 8. Add models section to `project_options` response
  - List available models (name, displayName, provider - no API keys)
  - Include defaults from config

- [x] 9. Validate model parameters in project creation
  - Check included_models exist in registry
  - Check session_model is in included_models
  - Validate autonomy_mode and reasoning_effort values

## Manager Script Updates

- [x] 10. Update `manager.sh init-config` for models
  - Prompt for model configurations
  - Create models.json with API keys

- [x] 11. ~~Add model management helpers~~ (not needed - edit JSON directly)

## Testing

- [x] 12. Add unit tests for model config loading
  - Valid JSON parsing
  - Missing file handling
  - Invalid model reference

- [x] 13. Add unit tests for settings patcher
  - Model injection
  - Session defaults
  - ID generation

- [x] 14. ~~Add integration test for project creation with models~~ (covered by existing tests)

- [x] 15. ~~Test manager.sh model commands~~ (not applicable - no model commands added)

## Documentation

- [x] 16. Update AGENTS.md with model configuration
- [x] 17. Update README.md with model setup (config section already present)
- [x] 18. Update docs/INSTANCE_MANAGER.md with model commands (init-config already updated)
