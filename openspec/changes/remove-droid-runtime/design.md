# Design: Remove Droid Runtime

## Current Architecture

```
cmd/server/main.go
  ├── droidRuntime = agentdroid.NewRuntime()
  ├── opencodeRuntime = agentopencode.NewRuntime()
  ├── runtimeFactory = func(type) → droid | opencode
  └── defaultRuntime = droid if FACTORY_API_KEY, else opencode

mcp/server.go
  ├── agentRuntime    agent.Runtime       (default)
  ├── runtimeFactory  RuntimeFactoryFunc  (per-project switching)
  └── GetRuntimeForProject(proj) → resolve runtime

project/manager.go → generateRuntimeConfigs()
  ├── .factory/mcp.json      (Droid MCP config)
  ├── .factory/settings.json  (Droid settings)
  └── opencode.json           (OpenCode config)

config/unified.go
  ├── ServerSection.AgentRuntime ("auto"|"droid"|"opencode")
  ├── ServerSection.Droid.DefaultModel
  └── CredentialsSection.Factory (FACTORY_API_KEY)

agent/config/opencode.go
  └── translateReasoningToOpenCode() → 80 lines of provider-specific reasoning baked into opencode.json
```

## Target Architecture

```
cmd/server/main.go
  └── agentRuntime = opencode.NewRuntime(containerRuntime)

mcp/server.go
  └── agentRuntime agent.Runtime (always OpenCode, no factory/dispatch)

project/manager.go → generateRuntimeConfigs()
  └── opencode.json (OpenCode config -- no reasoning, that's per-message)

config/unified.go
  └── ServerSection.Address (no runtime/droid fields)

agent/opencode/protocol.go
  └── SendMessageAsync() now passes variant for reasoning
```

## Reasoning: Static Config → Per-Message Variant

Currently oubliette translates reasoning level into provider-specific config in `opencode.json`:
```json
{"provider": {"anthropic": {"models": {"claude-opus-4-6": {"options": {"thinking": {"type": "enabled", "budgetTokens": 16000}}}}}}}
```

This is wrong for two reasons:
1. It's baked at project creation time -- can't change between messages
2. It duplicates 80+ lines of provider-specific logic that OpenCode already handles

OpenCode's `prompt_async` accepts `variant: "high"` which maps through `ProviderTransform.variants()` to the correct provider-specific options. The mapping is:
- **Anthropic**: `high` → `{thinking: {type: "enabled", budgetTokens: 16000}}`
- **OpenAI**: `high` → `{reasoningEffort: "high"}`
- **Google**: `high` → `{thinkingLevel: "high"}`

Oubliette's `ReasoningLevel: "low|medium|high"` maps directly to OpenCode's `variant` parameter.

## What stays

- `agent.Runtime` interface (minus `Initialize`, `Name`, `IsAvailable`)
- `agent.StreamingExecutor` interface
- `agent.StreamEvent` types
- `internal/agent/opencode/` (modified: variant support, abort)
- `internal/agent/config/opencode.go` (simplified: no reasoning translation)
- `internal/agent/config/types.go` (AgentConfig, minus Factory fields)

## What goes

| Component | Lines | Reason |
|-----------|-------|--------|
| `internal/agent/droid/` (6 files) | ~925 | Droid CLI protocol |
| `internal/agent/config/droid.go` + test | ~399 | Droid settings/MCP translation |
| `internal/agent/factory.go` + test | ~153 | Multi-runtime factory |
| `ServerSection.Droid` + `AgentRuntime` | ~15 | Droid server config |
| `credentials.factory` (types + methods) | ~60 | Factory API key management |
| `RuntimeFactoryFunc` + dispatch | ~30 | Per-project runtime switching |
| `.factory/` scaffolding in project manager | ~50 | Droid project setup |
| `template/.factory/` (19 files) | ~500 | Droid templates |
| `translateReasoningToOpenCode()` | ~80 | Replaced by per-message variant |
| `initializeFactoryConfig()` | ~25 | Droid minimal config |
| Droid install in Dockerfile | ~5 | Container image |

## Credential simplification

Currently three credential types: `factory`, `github`, `providers`. With Droid gone:
- `factory` is deleted entirely (types, methods, config section)
- `github` stays (for git operations)
- `providers` stays (for API keys to Anthropic/OpenAI/etc)

## Session abort

Currently `Cancel()` is a no-op TODO. OpenCode exposes `POST /:sessionID/abort`. Implementation:
```go
func (s *Server) AbortSession(ctx context.Context, sessionID string) error {
    _, err := s.doRequest(ctx, "POST", fmt.Sprintf("/session/%s/abort", sessionID), nil)
    return err
}
```
