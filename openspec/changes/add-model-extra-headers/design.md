# Design: Add Model Extra Headers

## Data Flow

```
oubliette.jsonc ModelDefinition.ExtraHeaders
    ↓
ModelRegistry (unchanged lookup interface)
    ↓
project manager writes per-runtime configs
    ├── Droid:    .factory/settings.json → customModels[].extraHeaders
    └── OpenCode: opencode.json → provider.<id>.models.<id>.headers
```

## Droid Translation

Droid's `customModels` already supports `extraHeaders` natively (see Factory BYOK docs). The `DroidCustomModel` struct just needs the field added, and `ToDroidSettings` needs to populate `customModels` from the model registry when extra headers are present.

Currently `ToDroidSettings` only sets `SessionDefaultSettings.Model` -- it doesn't write `customModels` at all. For extra headers to work, it needs to emit at least the active model as a `customModels` entry when that model has extra headers.

```go
type DroidCustomModel struct {
    // ... existing fields
    ExtraHeaders map[string]string `json:"extraHeaders,omitempty"`
}
```

## OpenCode Translation

OpenCode uses `provider.<providerID>.models.<modelID>.headers` (see anomalyco/opencode#7719). The `ToOpenCodeConfig` function already builds a `Provider` map for reasoning config. Extra headers slot into the same structure:

```go
// In provider config for the model
config.Provider["anthropic"] = ProviderConfig{
    Models: map[string]ModelOptions{
        "claude-opus-4-6": {
            Options: map[string]any{...},  // reasoning
            Headers: map[string]string{    // new
                "anthropic-beta": "context-1m-2025-08-07",
            },
        },
    },
}
```

This requires adding a `Headers` field to `ModelOptions`.

## Default Model Change

`applyUnifiedDefaults()` changes:
- `cfg.Server.Droid.DefaultModel`: `"sonnet"` → `"opus"`
- `cfg.Defaults.Agent.Model`: `"sonnet"` → `"opus"`

The shipped `oubliette.jsonc` gets an updated `opus` entry pointing at `claude-opus-4-6` with the 1M context header, and `session_model` defaults to `"opus"`.
