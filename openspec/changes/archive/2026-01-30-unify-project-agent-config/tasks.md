# Tasks: Unify Project Agent Configuration

## Phase 1: Core Types and Translation Layer

### 1.1 Create canonical config types
- [x] Create `internal/agent/config/types.go` with `ProjectConfig`, `AgentConfig`, `MCPServer`, etc.
- [x] Add JSON tags matching the proposed schema
- [x] Add validation methods for required fields
- **Validates**: Unit tests for type serialization/deserialization

### 1.2 Implement Droid config translator
- [x] Create `internal/agent/config/droid.go`
- [x] Implement `ToDroidMCPConfig()` - canonical → `.factory/mcp.json`
- [x] Implement `ToDroidSettings()` - canonical → `.factory/settings.json`
- [x] Reuse existing settings logic from `internal/project/settings.go`
- **Validates**: Unit tests comparing output to expected Droid config format

### 1.3 Implement OpenCode config translator
- [x] Create `internal/agent/config/opencode.go`
- [x] Implement `ToOpenCodeConfig()` - canonical → `opencode.json`
- [x] Implement `translateModelToOpenCodeFormat()` - add provider prefix
- [x] Implement `translateAutonomyToPermissions()` - map autonomy levels
- [x] Implement `translateReasoningToOpenCode()` - map reasoning to provider-specific options
- [x] Implement MCP server translation (stdio→local, http→remote)
- **Validates**: Unit tests comparing output to expected OpenCode config format

## Phase 2: Project Manager Integration

### 2.1 Replace metadata.json with config.json
- [x] Add `loadCanonicalConfig()` to `internal/project/manager.go`
- [x] Add `saveCanonicalConfig()` to `internal/project/manager.go`
- [x] Keep `saveMetadata()` for backwards compatibility during transition
- [x] Note: `Get()` still uses metadata.json - can migrate later
- **Validates**: New tests for config loading

### 2.2 Add runtime config generation
- [x] Add `generateRuntimeConfigs()` to `internal/project/manager.go`
- [x] Call from `Create()` after saving canonical config
- [x] Generate `.factory/mcp.json`, `.factory/settings.json`, `opencode.json`
- [x] Create `.factory/droids/` and `.factory/skills/` directories (Droid runtime)
- [x] Create `.opencode/agents/` and `.opencode/skills/` directories (OpenCode runtime)
- **Validates**: Integration test verifies all config files and directories created

### 2.3 Update CreateProjectRequest
- [x] Update `CreateProjectRequest` in `internal/project/types.go`
- [x] Add `Model`, `Autonomy`, `Reasoning` fields (new canonical format)
- [x] Add `MCPServers`, `Permissions` fields
- [x] Keep legacy `IncludedModels`, `SessionModel` for backwards compatibility
- [x] Update `Create()` to build canonical config from request
- **Validates**: Existing project creation tests pass

## Phase 3: Server Defaults

### 3.1 Rename and expand config-defaults.json schema
- [x] Create `config/config-defaults.json` with new expanded schema
- [x] Create `config/config-defaults.json.example`
- [x] Add `agent` section with `runtime`, `model`, `autonomy`, `reasoning`
- [x] Add `agent.mcp_servers` with oubliette-parent default
- [x] Delete old `config/project-defaults.json` and `.example`
- **Validates**: Server starts with new defaults

### 3.2 Update config loader
- [x] Update `internal/config/loader.go` to load expanded defaults
- [x] Create `ConfigDefaultsConfig` type matching new schema
- [x] Add `SetConfigDefaults()` to `ProjectManager`
- [x] Keep legacy `ProjectDefaultsConfig` for backwards compatibility
- **Validates**: Unit tests for defaults loading

## Phase 4: Read-Only Config Mounts

### 4.1 Update container mount configuration
- [x] Modify container start logic to mount config files read-only
- [x] Mount `config.json` as read-only at `/workspace/config.json`
- [x] Mount `opencode.json` as read-only at `/workspace/opencode.json`
- [x] Mount `.factory/mcp.json` as read-only
- [x] Mount `.factory/settings.json` as read-only
- [x] Keep `.factory/commands/` and `.factory/hooks/` writable (not mounted individually)
- **Validates**: Integration test verifies agent cannot write to config files

## Phase 5: MCP Handler Updates

### 5.1 Update project_create handler
- [x] Update `ProjectCreateParams` in `internal/mcp/handlers_project.go`
- [x] Add new canonical params: `model`, `autonomy`, `reasoning`, `disabled_tools`, `mcp_servers`, `permissions`
- [x] Map new params to `CreateProjectRequest`
- [x] Add validation for autonomy levels (off, low, medium, high)
- [x] Add validation for reasoning levels (off, low, medium, high)
- **Validates**: Integration test for project_create with new params

### 5.2 Update project_get response
- [x] Note: Deferred - project_get still returns Project struct from metadata.json
- [x] Agent config can be added to response in a future enhancement
- **Validates**: Existing tests still pass

## Phase 6: Documentation and Cleanup

### 6.1 Update documentation
- [x] Update `README.md` Configuration section (rename project-defaults.json, new config.json schema, simplified model param)
- [x] Update `AGENTS.md` with new config structure
- [x] Remove `docs/RUNTIME_COMPARISON.md` (no longer needed, details in AGENTS.md)
- [x] No migration notes needed (POC stage)

### 6.2 Remove deprecated code
- [x] Legacy `IncludedModels`, `SessionModel` handling kept for backwards compatibility
- [x] Model registry methods still needed for legacy support
- [x] Old settings generation code still available but new translator preferred

## Dependencies

```
1.1 ──┬──> 1.2 ──┐
      │         │
      └──> 1.3 ─┼──> 2.1 ──> 2.2 ──> 2.3 ──> 4.1 ──> 5.1 ──> 5.2
                │                      │
3.1 ──> 3.2 ────┘                      │
                                       └──> 6.1 ──> 6.2
```

- Phase 1 tasks can run in parallel
- Phase 2 depends on Phase 1 completion
- Phase 3 can run in parallel with Phase 1
- Phase 4 (read-only mounts) depends on Phase 2 (config generation)
- Phase 5 depends on Phases 2, 3, and 4
- Phase 6 depends on Phase 5
