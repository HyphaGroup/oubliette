# Proposal: Add Model Extra Headers and Default to Opus 4.6 1M

## Why

Models like `claude-opus-4-6` require extra HTTP headers (`anthropic-beta: context-1m-2025-08-07`) to enable extended context windows. There's no way to configure these in `oubliette.jsonc` today, and neither runtime translation layer passes them through.

Additionally, the default shipped model should be `claude-opus-4-6` with the 1M context header.

## What Changes

### 1. Add `extraHeaders` to model definitions

In `oubliette.jsonc`:

```jsonc
"models": {
  "models": {
    "opus": {
      "model": "claude-opus-4-6",
      "displayName": "Opus 4.6 1M",
      "baseUrl": "https://api.anthropic.com",
      "maxOutputTokens": 128000,
      "provider": "anthropic",
      "extraHeaders": {
        "anthropic-beta": "context-1m-2025-08-07"
      }
    }
  }
}
```

### 2. Thread extra headers through both runtime config translators

**Droid** (`settings.json` `customModels`): Already supports `extraHeaders` natively. Just pass it through from `ModelDefinition`.

**OpenCode** (`opencode.json` `provider.<id>.models.<id>.headers`): Already supports `headers` on model definitions. Translate `extraHeaders` to OpenCode's `headers` field.

### 3. Change default model to `opus`

Update `applyUnifiedDefaults()` to default `model` to `"opus"` instead of `"sonnet"`, and ship the Opus 4.6 1M definition in the default config.

## Impact

| File | Change |
|------|--------|
| `internal/config/models.go` | Add `ExtraHeaders map[string]string` to `ModelDefinition` |
| `internal/agent/config/droid.go` | Add `ExtraHeaders` to `DroidCustomModel`, populate in `ToDroidSettings` |
| `internal/agent/config/opencode.go` | Populate `headers` in provider model config from `ExtraHeaders` |
| `internal/config/unified.go` | Change default model from `"sonnet"` to `"opus"` |
| `config/oubliette.jsonc` | Update `opus` model definition, change default |
| `config/oubliette.jsonc.example` | Same |

## Out of Scope

- `extraArgs` (provider-specific request body params) -- separate concern
- Per-provider API key in model definitions (already supported via credential refs)
