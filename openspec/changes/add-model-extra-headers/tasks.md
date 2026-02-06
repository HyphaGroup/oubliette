# Tasks: Add Model Extra Headers

## 1. Schema

- [x] 1.1 Add `ExtraHeaders map[string]string` to `ModelDefinition` in `internal/config/models.go`
- [x] 1.2 Add `ExtraHeaders map[string]string` to `DroidCustomModel` in `internal/agent/config/droid.go`
- [x] 1.3 Add `Headers map[string]string` to `ModelOptions` in `internal/agent/config/opencode.go`

## 2. Translation

- [x] 2.1 Update `ToDroidSettings` to emit `customModels` entries when extra headers are present
- [x] 2.2 Update `ToOpenCodeConfig` to populate `provider.<id>.models.<id>.headers` from `ExtraHeaders`
- [x] 2.3 Add model metadata fields to `AgentConfig` (`ModelProvider`, `ModelDisplay`, `ModelBaseURL`, `ModelMaxOut`, `ExtraHeaders`) and populate from model registry in project manager

## 3. Default Model

- [x] 3.1 Remove hardcoded model fallback from `applyUnifiedDefaults()` -- config must specify the model
- [x] 3.2 Update `opus` model in `config/oubliette.jsonc`: model `claude-opus-4-6`, maxOutputTokens 128000, extraHeaders with `anthropic-beta: context-1m-2025-08-07`, default model to `opus`
- [x] 3.3 Update `config/oubliette.jsonc.example` to match
- [x] 3.4 Update `models.defaults.session_model` to `"opus"` in both config files

## 4. Testing

- [x] 4.1 Add unit test: `ModelDefinition` with `ExtraHeaders` round-trips through JSON
- [x] 4.2 Add unit test: `ToDroidSettings` with extra headers emits `customModels` entry
- [x] 4.3 Add unit test: `ToOpenCodeConfig` with extra headers populates provider model headers
- [x] 4.4 Add unit test: extra headers merged with reasoning config in OpenCode
- [x] 4.5 Build and verify: `go build ./...` passes, all new tests pass
