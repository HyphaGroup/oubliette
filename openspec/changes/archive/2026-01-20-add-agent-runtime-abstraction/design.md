# Design: Agent Runtime Abstraction

## Context

Oubliette currently has a single AI backend (Factory Droid) hardcoded throughout the codebase. The existing `container.Runtime` abstraction successfully supports Docker and Apple Container interchangeably. We need a similar pattern for AI agent runtimes to support both Droid and OpenCode.

**Reference Implementation**: `reference/opencode/` contains the full OpenCode source code for implementation reference. Key files:
- `packages/opencode/src/server/server.ts` - HTTP API with SSE streaming
- `packages/opencode/src/session/` - Session management, prompts, messages
- `packages/opencode/src/cli/` - CLI commands including `serve`

**Stakeholders**: Developers wanting local development without Factory API key, teams preferring OpenCode ecosystem.

**Constraints**:
- Must not break existing Droid functionality
- Must support bidirectional streaming (JSON-RPC for Droid, SSE for OpenCode)
- Must handle different session management paradigms
- Container-based execution model must be preserved

## Goals / Non-Goals

**Goals**:
- Abstract agent execution behind a unified interface
- Support Droid and OpenCode as interchangeable backends
- Normalize streaming events across runtimes
- Enable auto-detection based on available credentials
- Maintain existing performance characteristics

**Non-Goals**:
- Cross-runtime session migration (sessions bound to runtime)
- Hybrid execution (different runtimes at different depths)
- Supporting non-streaming execution as primary mode

## Decisions

### D1: OpenCode Server Lifecycle

**Decision**: Long-running server per project

**Rationale**:
- Matches container lifecycle (start/stop together)
- Fast session creation (no cold boot per session)
- OpenCode's `serve` command designed for this pattern

**Implementation**:
```
Container Start → opencode serve --port 4096 --hostname 127.0.0.1
Session Operations → HTTP API at http://127.0.0.1:4096
Container Stop → Server terminates with container
```

**OpenCode API Endpoints** (from `reference/opencode/packages/opencode/src/server/server.ts`):
- `GET /global/health` - Health check
- `GET /event` - SSE stream for all events
- `POST /session` - Create session (operationId: `session.create`)
- `GET /session/:sessionID` - Get session details
- `POST /session/:sessionID/message` - Send message (streaming response)
- `POST /session/:sessionID/abort` - Abort active session
- `DELETE /session/:sessionID` - Delete session

### D2: Event Normalization

**Decision**: Normalize at executor level, not in consumers

**Rationale**:
- Single point of translation
- Consumers work with unified `StreamEvent` type
- Runtime-specific details hidden behind interface

### D3: Permission Handling

**Decision**: Auto-approve all permissions in both runtimes

**Rationale**:
- Oubliette containers are trusted execution environments
- Droid: `--skip-permissions-unsafe`
- OpenCode: `permission: { edit: "allow", bash: { "*": "allow" } }`

### D4: Session ID Mapping

**Decision**: Oubliette manages own session IDs, stores runtime session ID separately

**Rationale**:
- Droid returns `droidSessionID` from init response
- OpenCode returns `sessionID` from `session.create()`
- Both stored in `Session.RuntimeSessionID` for continuation

### D5: Health Monitoring

**Decision**: Detect failures on next operation, return error to caller

**Rationale**:
- Keeps abstraction simple
- Container restart already recovers everything
- Add optional `Runtime.Ping(ctx)` for proactive checks

### D6: Model Configuration Integration

**Decision**: Use `config/models.json` for Droid models, OpenCode uses its own config

**Rationale**:
- Droid models configured via `add-model-configuration` change
- API keys stored in `config/models.json` (gitignored)
- OpenCode has its own `~/.opencode/config.json` inside container
- No cross-runtime model configuration needed

**Integration**:
- Droid: Models from `config/models.json` injected into project `.factory/settings.json`
- OpenCode: Uses container's local OpenCode config

### D7: OpenCode Agent Selection

**Decision**: Use `build` agent by default, `plan` agent for UseSpec mode

**Rationale**:
- `build` matches Droid's default behavior
- `plan` matches Droid's `--use-spec` flag

**Mapping**:
- `UseSpec=false` → `agent: "build"`
- `UseSpec=true` → `agent: "plan"`

## Package Structure

```
internal/agent/
├── runtime.go              # Interface definitions
├── types.go                # StreamEvent, ExecuteRequest, etc.
├── factory.go              # NewRuntime() with auto-detection
├── droid/
│   ├── runtime.go          # Runtime implementation
│   ├── executor.go         # StreamingExecutor implementation
│   ├── command.go          # CLI command building
│   ├── jsonrpc.go          # JSON-RPC protocol
│   └── parser.go           # Output parsing
└── opencode/
    ├── runtime.go          # Runtime implementation
    ├── executor.go         # StreamingExecutor with SSE
    ├── server.go           # Server lifecycle management
    └── events.go           # SSE event parsing
```

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| Protocol differences cause data loss | Medium | Comprehensive event mapping tests |
| OpenCode server startup overhead | Low | Connection pooling, long-running servers |
| Session state incompatibility | Medium | Abstract session storage |
| Permission model mismatch | Low | Auto-approve mode for OpenCode |

## Migration Plan

1. **Phase 1**: Define interfaces, no behavior change
2. **Phase 2**: Move Droid code, implement interface wrapper
3. **Phase 3**: Update session package to use interfaces
4. **Phase 4**: Implement OpenCode backend
5. **Phase 5**: Integration and configuration
6. **Phase 6**: Documentation

**Rollback**: Each phase is independently revertible. Interface can be removed if abstraction proves problematic.

## Configuration Dependencies

This change depends on and integrates with:

1. **migrate-config-to-files**: Provides `config/server.json` where `agent_runtime` is configured
2. **add-model-configuration**: Provides `config/models.json` for Droid API keys

**Config file integration**:
```
config/
├── server.json           # agent_runtime: "auto"|"droid"|"opencode"
├── factory.json          # Factory API key (for auto-detection)
└── models.json           # Droid model configs with API keys
```

**Auto-detection flow**:
```
server.json.agent_runtime == "auto"
    ↓
Check factory.json exists and has valid api_key
    ↓
Yes → Droid runtime
No  → OpenCode runtime
```

**Per-project override**:
- Stored in `projects/<id>/metadata.json` as `agent_runtime`
- Overrides server-level default
- Exposed via `project_options` tool

## Open Questions

- None remaining (all resolved in prior research)
