# Proposal: Remove Droid Runtime, OpenCode-Only

## Why

Oubliette ships dual-runtime support (Factory Droid + OpenCode) but only needs OpenCode. The Droid runtime adds ~1,300 lines of code, forces a `Runtime`/`StreamingExecutor` abstraction shaped by Droid's JSON-RPC protocol, and scatters Factory-specific concepts (`FACTORY_API_KEY`, `.factory/` directories, `settings.json` generation) throughout the codebase. OpenCode has a rich HTTP API with its own session management, model config, reasoning variants, agent types, and event streaming -- the abstraction layer duplicates rather than leverages these capabilities.

## What Changes

### Phase 1: Delete Droid Runtime Code

- Delete `internal/agent/droid/` (6 files, ~925 lines)
- Delete `internal/agent/config/droid.go` and `droid_test.go` (~399 lines)
- Delete `internal/agent/factory.go` and `factory_test.go` (multi-runtime factory)

### Phase 2: Simplify Runtime Interface

The current interface has methods shaped by multi-runtime needs:
- `Initialize()` -- no-op for OpenCode (just sets `initialized = true`)
- `Name()` -- only needed for runtime dispatch/logging
- `IsAvailable()` -- always true for OpenCode
- `RuntimeConfig.APIKey` -- Droid-only field

Simplify: remove `Initialize()`, `Name()`, `IsAvailable()`, `RuntimeConfig.APIKey`. OpenCode runtime construction does everything needed.

Remove `RuntimeType` constants, `DetectRuntimeType`, `RuntimeFactoryFunc`, `GetRuntimeForProject`.

### Phase 3: Remove Droid from Config/Credentials

- Remove `ServerSection.Droid` and `DroidSection` from `UnifiedConfig`
- Remove `ServerSection.AgentRuntime` (always OpenCode)
- Remove `credentials.factory` section from `CredentialRegistry`
- Remove `GetDefaultFactoryKey`, `GetFactoryKey`, `HasFactoryCredential`, `HasFactoryAPIKey`
- Remove `FactoryCredentials`, `FactoryCredential` types
- Remove `CredentialRefs.Factory` from project types and agent config
- Remove `LoadedConfig.HasFactoryAPIKey()`
- Remove `DroidJSONConfig`, `ServerJSONConfig.Droid`
- Remove `FACTORY_API_KEY` from `sensitivePatterns` in `errors.go`

### Phase 4: Simplify Project Manager

- Delete `initializeFactoryConfig()` (creates `.factory/{mcp.json, settings.json}`)
- Remove all `.factory/` directory scaffolding from `generateRuntimeConfigs()`
- Remove Droid MCP and settings generation from `generateRuntimeConfigs()`
- Keep only `opencode.json` generation
- Remove `AgentRuntime` from `Project`, `CreateProjectRequest`
- Remove `AgentRuntime` validation in `handleProjectCreate`

### Phase 5: Delete Droid Templates

- Delete `template/.factory/` (19 files: droids, skills, commands, hooks, config)
- Remove `.factory/` copy logic from `CreateWorkspace`
- Remove `.factory/` copy from project creation

### Phase 6: Pass Reasoning as Variant

Currently oubliette bakes reasoning config into `opencode.json` at project creation time (80+ lines of `translateReasoningToOpenCode()` per-provider logic). OpenCode supports per-message `variant` parameter which maps to the model's reasoning variants natively.

- Add `variant` field to `SendMessageAsync` call in `protocol.go`
- Map `ExecuteRequest.ReasoningLevel` to OpenCode's `variant` parameter
- Delete `translateReasoningToOpenCode()` (80+ lines)
- Remove reasoning from `opencode.json` provider config generation

### Phase 7: Implement Session Abort

OpenCode has `POST /:sessionID/abort`. Currently `executor.Cancel()` is a TODO no-op.

- Add `AbortSession` to `protocol.go`
- Implement `Cancel()` on `StreamingExecutor` to call abort endpoint

### Phase 8: Rename DroidSessionID

- Rename `session.DroidSessionID` to `session.RuntimeSessionID` across all files
- Update all references and JSON tags

### Phase 9: Simplify Server Layer

- Remove `RuntimeFactoryFunc` type and `runtimeFactory` field from `Server`
- Remove `GetRuntimeForProject` -- all callers use `s.agentRuntime` directly
- Remove runtime factory initialization from `cmd/server/main.go`
- Remove Droid runtime initialization from `cmd/server/main.go`
- Remove `agentdroid` import
- Remove `droid` from project options response
- Remove Droid MCP setup option (`oubliette mcp --setup droid`)

### Phase 10: Container Image

- Remove Droid CLI install from `containers/base/Dockerfile`
- Remove `~/.factory/bin` from PATH
- Update build-time verification to `opencode --version && rg --version`
- Remove `droid` from `containers/base/metadata.json`

### Phase 11: Verify

- `go build ./...`
- `go test ./... -short` -- all pass
- Build container image
- Spawn session and verify agent works

## Impact

~40 files touched. ~2,800 lines deleted, ~200 lines added. Net reduction ~2,600 lines.

## Out of Scope

- Per-message agent selection (`agent: "plan"|"explore"|"general"`) -- useful but separate change
- Session summarization/compaction API -- separate change
- Todo API surfacing -- separate change
- Removing the `agent.Runtime` interface entirely -- kept as thin abstraction
