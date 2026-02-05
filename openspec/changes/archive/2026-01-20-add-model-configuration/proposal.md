# Proposal: Add Model Configuration

## Summary

Add configurable custom models and session defaults that can be specified at project creation time. Models are defined in `config/models.json` and applied to `template/.factory/settings.json` when creating projects.

## Motivation

Current `template/.factory/settings.json` has:
- Hardcoded API keys (security issue - shouldn't be in template)
- Fixed custom models that can't vary per deployment
- No way to specify different defaults for different project types

Need:
- Centralized model configuration with API keys (gitignored)
- Ability to select which models a project gets at creation time
- Session defaults (model, autonomy, reasoning) configurable per project

## Proposed Solution

### Config Structure

**config/models.json** (gitignored - contains API keys):
```json
{
  "models": {
    "sonnet": {
      "model": "claude-sonnet-4-5",
      "displayName": "Sonnet 4.5",
      "baseUrl": "https://api.anthropic.com",
      "apiKey": "sk-ant-xxx",
      "maxOutputTokens": 64000,
      "provider": "anthropic"
    },
    "opus": {
      "model": "claude-opus-4-5",
      "displayName": "Opus 4.5",
      "baseUrl": "https://api.anthropic.com",
      "apiKey": "sk-ant-xxx",
      "maxOutputTokens": 64000,
      "provider": "anthropic"
    },
    "gpt5": {
      "model": "gpt-5.1",
      "displayName": "GPT 5.1",
      "baseUrl": "https://api.openai.com/v1",
      "apiKey": "sk-xxx",
      "maxOutputTokens": 32000,
      "provider": "openai"
    }
  },
  "defaults": {
    "included_models": ["sonnet", "opus"],
    "session_model": "opus",
    "autonomy_mode": "auto-high",
    "reasoning_effort": "medium"
  }
}
```

**template/.factory/settings.json** (tracked, no secrets):
```json
{
  "sessionDefaultSettings": {
    "autonomyMode": "{{autonomy_mode}}",
    "model": "{{session_model_id}}",
    "reasoningEffort": "{{reasoning_effort}}"
  },
  "customModels": [],
  "logoAnimation": "off",
  "includeCoAuthoredByDroid": false,
  "allowBackgroundProcesses": true,
  "cloudSessionSync": false,
  "hooks": {}
}
```

### Project Creation Flow

1. Load `config/models.json`
2. Load `template/.factory/settings.json`
3. Apply overrides from MCP parameters or defaults:
   - `included_models` → populate `customModels` array with full model configs
   - `session_model` → set `sessionDefaultSettings.model` to model's generated ID
   - `autonomy_mode` → set `sessionDefaultSettings.autonomyMode`
   - `reasoning_effort` → set `sessionDefaultSettings.reasoningEffort`
4. Write patched `settings.json` to project `.factory/`

### MCP Tool Parameters

**project_create** additions:
- `included_models` - array of model names from registry (default: from config)
- `session_model` - which model is session default (default: from config)
- `autonomy_mode` - "auto-high", "auto-low", "manual" (default: from config)
- `reasoning_effort` - "low", "medium", "high" (default: from config)

**project_options** additions:
```json
{
  "models": {
    "available": [
      {"name": "sonnet", "displayName": "Sonnet 4.5", "provider": "anthropic"},
      {"name": "opus", "displayName": "Opus 4.5", "provider": "anthropic"},
      {"name": "gpt5", "displayName": "GPT 5.1", "provider": "openai"}
    ],
    "defaults": {
      "included_models": ["sonnet", "opus"],
      "session_model": "opus",
      "autonomy_mode": "auto-high",
      "reasoning_effort": "medium"
    }
  }
}
```

## Scope

### In Scope
- `config/models.json` schema and loading
- Clean `template/.factory/settings.json` (no secrets)
- Patch settings.json during project creation
- Model/session parameters on `project_create`
- Models section in `project_options` response

### Out of Scope
- Runtime model switching (already supported by Factory)
- Per-session model overrides via MCP (use session_message params)
- Model validation/testing

## Related Changes

- **migrate-config-to-files**: Creates config/ structure
- **add-github-account-registry**: Creates project_options tool
