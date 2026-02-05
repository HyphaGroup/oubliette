# Change: Add OpenSpec Integration for Autonomous Spec-Driven Development

## Why

Oubliette agent sessions currently lack structured spec-driven workflows. Agents operate from chat prompts without persistent specification context, making it difficult to:
- Agree on requirements before implementation
- Track what's proposed vs implemented
- Maintain continuity across sessions
- Coordinate recursive task decomposition with explicit specs
- Know when work is complete (completion detection)
- Prioritize which changes to work on next

OpenSpec provides a lightweight specification workflow that aligns humans and AI agents on what to build before code is written. Combined with orchestration capabilities, this enables fully autonomous development loops.

## What Changes

- **ADDED** `@fission-ai/openspec` CLI to container Dockerfile (uses existing NVM Node.js)
- **ADDED** `template/openspec/` directory committed to repository (pre-generated templates)
- **ADDED** OpenSpec slash commands in `template/.factory/commands/`
- **ADDED** `project_changes` MCP tool - thin wrapper around `openspec list --json`
- **ADDED** `project_tasks` MCP tool - thin wrapper around `openspec instructions apply --json`
- **ADDED** Session modes: `plan`, `build`, `interactive` (default)
- **ADDED** Build mode completion detection (all tasks done = complete)
- **MODIFIED** Project creation to copy `template/openspec/` to new projects
- **MODIFIED** Workspaces inherit project openspec/ via shared filesystem (no copy)
- **MODIFIED** `session_message` to accept `mode` and `change_id` parameters

## Impact

- **Affected specs**: openspec-integration (new capability)
- **Affected code**:
  - `internal/container/Dockerfile` - Add openspec CLI installation
  - `template/` - Add openspec/ directory structure
  - `internal/project/manager.go` - Copy openspec/ on project creation
  - `internal/mcp/handlers_project.go` - Add project_changes, project_tasks handlers
  - `internal/mcp/handlers_session.go` - Add mode parameter to session_message
  - `internal/mcp/server.go` - Register new tools
  - `internal/session/types.go` - Add SessionMode type

## Success Criteria

- Container image includes working `openspec` CLI
- New projects automatically have `openspec/` directory with AGENTS.md, project.md
- Spawned agents can run `openspec list`, `openspec validate`, etc.
- Agents recognize `/openspec-proposal`, `/openspec-apply`, `/openspec-archive` commands
- `project_changes` MCP tool returns list of changes with status and ordering
- `project_tasks` MCP tool returns structured task tree with completion status
- Both tools correlate active sessions with tasks via TaskContext
- Planning mode sessions are instructed to create proposals
- Build mode sessions work until all tasks complete (Ralph-style loop)
- Change ordering uses numeric prefix convention (010-, 020-, etc.)
