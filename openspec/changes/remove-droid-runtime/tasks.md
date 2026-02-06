# Tasks: Remove Droid Runtime

## 1. Delete Droid Runtime Code

- [x] 1.1 Delete `internal/agent/droid/` directory (6 files, ~925 lines)
- [x] 1.2 Delete `internal/agent/config/droid.go` and `internal/agent/config/droid_test.go` (~399 lines)
- [x] 1.3 Delete `internal/agent/factory.go` and `internal/agent/factory_test.go` (~153 lines)

## 2. Simplify Runtime Interface

- [x] 2.1 Remove `Initialize()` from `agent.Runtime` interface and OpenCode implementation
- [x] 2.2 Remove `Name()` from `agent.Runtime` interface and OpenCode implementation
- [x] 2.3 Remove `IsAvailable()` from `agent.Runtime` interface and OpenCode implementation
- [x] 2.4 Remove `RuntimeConfig` type entirely (was only used for Initialize)
- [x] 2.5 Remove `RuntimeType` constants and `DetectRuntimeType` (deleted with factory.go)
- [x] 2.6 Update `internal/agent/AGENTS.md` to reflect single-runtime, simplified interface

## 3. Remove Droid from Config/Credentials

- [x] 3.1 Remove `DroidSection` and `Droid` field from `ServerSection` in `internal/config/unified.go`
- [x] 3.2 Remove `AgentRuntime` field from `ServerSection`
- [x] 3.3 Remove `FactoryCredentials`, `FactoryCredential` types and `GetFactoryKey`, `GetDefaultFactoryKey`, `HasFactoryCredential` methods from `credentials.go`
- [x] 3.4 Remove `credentials.factory` from `CredentialsSection`
- [x] 3.5 Remove `DroidJSONConfig` and `ServerJSONConfig.Droid` from `loader.go`
- [x] 3.6 Remove `HasFactoryAPIKey()` from `LoadedConfig`
- [x] 3.7 Remove Factory credential initialization in `LoadUnifiedConfig`
- [x] 3.8 Remove Factory credential tests from `loader_test.go`
- [x] 3.9 Remove `FACTORY_API_KEY` from `sensitivePatterns` in `errors.go`
- [x] 3.10 Remove `CredentialRefs.Factory` from `internal/agent/config/types.go` and `internal/project/types.go`
- [x] 3.11 Update `config/oubliette.jsonc` and `config/oubliette.jsonc.example` to remove Factory/Droid fields

## 4. Simplify Project Manager

- [x] 4.1 Delete `initializeFactoryConfig()` method
- [x] 4.2 Remove `.factory/` directory scaffolding from `generateRuntimeConfigs()`
- [x] 4.3 Remove Droid MCP and settings generation from `generateRuntimeConfigs()`
- [x] 4.4 Remove `AgentRuntime` from `Project` struct and `CreateProjectRequest`
- [x] 4.5 Remove `AgentRuntime` validation in `handlers_project.go`
- [x] 4.6 Remove `droid` from `AgentRuntimesResponse` in project options
- [x] 4.7 Remove `AgentRuntimesResponse` entirely (single runtime, not needed)
- [x] 4.8 Remove `.factory/` template copy from project creation and workspace creation

## 5. Delete Droid Templates

- [x] 5.1 Delete `template/.factory/` directory (19 files)

## 6. Pass Reasoning as Variant

- [x] 6.1 Add `variant` field to `SendMessageAsync()` in `internal/agent/opencode/protocol.go`
- [x] 6.2 Pass `ExecuteRequest.ReasoningLevel` as `variant` in `opencode/executor.go` (via runtime.go)
- [x] 6.3 Delete `translateReasoningToOpenCode()` from `internal/agent/config/opencode.go` (~80 lines)
- [x] 6.4 Remove reasoning from `ToOpenCodeConfig()` provider config generation
- [x] 6.5 Update opencode config tests to remove reasoning assertions

## 7. Implement Session Abort

- [x] 7.1 Add `AbortSession()` to `internal/agent/opencode/protocol.go`
- [x] 7.2 Implement `Cancel()` on `StreamingExecutor` to call `AbortSession()`

## 8. Rename DroidSessionID

- [x] 8.1 Rename `Session.DroidSessionID` to `RuntimeSessionID` in `session/types.go`
- [x] 8.2 Update JSON tag from `droid_session_id` to `runtime_session_id`
- [x] 8.3 Update all references in `session/streaming.go`, `session/manager.go`, `session/active.go`
- [x] 8.4 Update streaming.go comments to remove Droid references

## 9. Simplify Server Layer

- [x] 9.1 Remove `RuntimeFactoryFunc` type and `runtimeFactory` field from `mcp/server.go`
- [x] 9.2 Remove `GetRuntimeForProject()` -- replace callers with `s.agentRuntime`
- [x] 9.3 Remove `RuntimeFactory` from `ServerConfig`
- [x] 9.4 Simplify `HasAPICredentials()` to only check provider credentials
- [x] 9.5 Remove runtime factory initialization from `cmd/server/main.go`
- [x] 9.6 Remove Droid runtime initialization from `cmd/server/main.go`
- [x] 9.7 Remove `agentdroid` import from `cmd/server/main.go`
- [x] 9.8 Simplify default runtime selection (always OpenCode, no Factory key check)
- [x] 9.9 Remove Droid MCP setup option from `cmd/server/main.go` (`mcp --setup droid`)
- [x] 9.10 Remove `server_test.go` runtime factory tests (replaced with minimal test)
- [x] 9.11 Remove Droid references from socket_handler.go comments

## 10. Container Image

- [x] 10.1 Remove `curl -fsSL https://app.factory.ai/cli | sh` from `containers/base/Dockerfile`
- [x] 10.2 Remove `~/.factory/bin` from PATH
- [x] 10.3 Update verification to `opencode --version && rg --version`
- [x] 10.4 Update `containers/base/metadata.json`

## 11. Verify

- [x] 11.1 `go build ./...` passes
- [x] 11.2 `go test ./... -short` -- all pass
- [ ] 11.3 Build container image (requires docker)
- [ ] 11.4 Spawn session and verify agent works (requires running server)
