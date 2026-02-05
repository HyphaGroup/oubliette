# Proposal: Add Workspace Isolation Mode

## Summary

Add a project-level setting to control whether agents see the full project directory or only their assigned workspace.

## Motivation

Currently, the entire project directory is mounted at `/workspace` in containers. While the agent's working directory is set to the specific workspace (`/workspace/workspaces/<uuid>/`), the agent can navigate up and access:

- `AGENTS.md` - system prompt (can read/modify)
- `metadata.json` - project configuration
- `sessions/` - all session history
- `workspaces/` - all other users' workspaces
- `.factory/` - project-level config

For multi-tenant chat agents (e.g., users via Signal), this creates privacy and security concerns:
1. Users can instruct the agent to modify its own system prompt
2. Agents can see other users' workspaces and data
3. Session history is accessible

## Proposed Solution

Add two project-level settings:

### 1. `workspace_isolation` (boolean, default: `false`)

When `workspace_isolation: true`:
- Mount only `workspaces/<uuid>/` at `/workspace`
- Copy `AGENTS.md` to workspace root on workspace creation
- Copy `.factory/` to workspace on creation (already happens)
- Agent cannot access project root, other workspaces, or sessions

When `workspace_isolation: false` (current default):
- Mount full project at `/workspace`
- Agent cwd is `workspaces/<uuid>/` but can navigate freely
- Shared access to project files for collaboration

### 2. `protected_paths` (string array, default: `[]`)

Paths relative to workspace root that should be mounted read-only:
- `["AGENTS.md", ".factory/"]` - Protect system prompt and config
- `["AGENTS.md", ".factory/mcp.json", ".factory/settings.json"]` - More granular

When isolation is enabled and protected paths are set:
- Workspace is mounted read-write at `/workspace`
- Each protected path gets an overlay read-only mount
- Agent can read but not modify protected files
- Kernel-enforced (not hook-based)

## Scope

- Project metadata: Add `workspace_isolation` and `protected_paths` fields
- Container creation: Conditional mount based on isolation setting
- Container creation: Add read-only overlay mounts for protected paths
- Workspace creation: Copy `AGENTS.md` when isolation enabled
- MCP tools: Expose settings in `project_create` and `project_get`

## Alternatives Considered

1. **Always isolate** - Breaking change, loses shared file access
2. **Per-workspace setting** - More complex, project-level is sufficient
3. **Hook-based protection** - Requires Droid cooperation, not kernel-enforced
4. **Read-only mounts for specific files** - More granular but more complex

## Decision

Project-level `workspace_isolation` flag with default `false` for backward compatibility.
