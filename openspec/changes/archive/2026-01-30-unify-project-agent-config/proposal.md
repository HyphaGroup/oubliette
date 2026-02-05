# Proposal: Unify Project Agent Configuration

## Why

We need to support both Droid and OpenCode agent runtimes within the same project, but currently:
1. Agent configuration is fragmented across multiple files with no single source of truth
2. OpenCode runtime cannot work because its config (`opencode.json`) is never generated
3. Switching runtimes requires manual reconfiguration
4. Model, autonomy, and MCP settings are handled inconsistently between runtimes

This change establishes a canonical config format and automatic translation layer, enabling runtime portability and completing OpenCode runtime support.

## Summary

Consolidate project configuration into a single canonical `config.json` file at the project root, replacing the current `metadata.json`. This file becomes the source of truth for all agent runtime settings, from which runtime-specific configs (`.factory/mcp.json`, `opencode.json`) are generated for both runtimes at project creation time.

## Problem Statement

Currently, agent configuration is fragmented:
1. **Project metadata** (`metadata.json`) stores project info but not agent settings
2. **Droid config** (`.factory/mcp.json`, `.factory/settings.json`) is generated separately
3. **OpenCode config** (`opencode.json`) is not generated at all - blocking OpenCode runtime support
4. **Server defaults** (`config/project-defaults.json`, `config/models.json`) are separate files

This creates several issues:
- No single source of truth for project agent settings
- Cannot easily switch runtimes within a project
- OpenCode runtime requires manual config that doesn't exist
- Model, autonomy, and permission settings scattered across files

## Proposed Solution

### 1. Canonical Config: `projects/<id>/config.json`

Replace `metadata.json` with `config.json` containing all project and agent settings:

```json
{
  "id": "proj_abc123",
  "name": "my-project",
  "description": "Project description",
  "created_at": "2025-01-28T...",
  "default_workspace_id": "uuid",
  
  "container": {
    "type": "dev",
    "image_name": "oubliette:dev",
    "has_dockerfile": false
  },
  
  "agent": {
    "runtime": "droid",
    "model": "claude-sonnet-4-5-20250929",
    "autonomy": "high",
    "reasoning": "medium",
    "disabled_tools": [],
    "mcp_servers": {
      "oubliette-parent": {
        "type": "stdio",
        "command": "/usr/local/bin/oubliette-client",
        "args": ["/mcp/relay.sock"]
      }
    },
    "permissions": {}
  },
  
  "limits": {
    "max_recursion_depth": 3,
    "max_agents_per_session": 50,
    "max_cost_usd": 10.0
  }
}
```

### 2. Runtime Config Generation

At `project_create` time, generate **both** runtime configs:

- `.factory/mcp.json` - Droid MCP servers
- `.factory/settings.json` - Droid settings (model, autonomy, etc.)
- `opencode.json` - OpenCode config with translated schema

This allows seamless runtime switching without regeneration.

### 3. Server Defaults: `config/config-defaults.json` (renamed from project-defaults.json)

Rename and expand to include all agent defaults:

```json
{
  "limits": {
    "max_recursion_depth": 3,
    "max_agents_per_session": 50,
    "max_cost_usd": 10.0
  },
  "agent": {
    "runtime": "droid",
    "model": "claude-sonnet-4-5-20250929",
    "autonomy": "off",
    "reasoning": "medium",
    "mcp_servers": {
      "oubliette-parent": {
        "type": "stdio",
        "command": "/usr/local/bin/oubliette-client",
        "args": ["/mcp/relay.sock"]
      }
    }
  }
}
```

Note: `autonomy: "off"` is the default because agents run in isolated containers with no human to respond to permission prompts.

### 4. Config Translation Layer

New `internal/agent/config/` package with:
- `ToDroidConfig(canonical) -> DroidConfig` - generates `.factory/*`
- `ToOpenCodeConfig(canonical) -> OpenCodeConfig` - generates `opencode.json`

Translation handles schema differences:
| Canonical | Droid | OpenCode |
|-----------|-------|----------|
| `mcp_servers` | `mcpServers` key, `stdio` type | `mcp` key, `local` type |
| `autonomy: "off"` | `--skip-permissions-unsafe` | `"permission": "allow"` |
| `autonomy: "high"` | `--auto high` | `permission: {"*": "allow", ...}` |
| `autonomy: "medium"` | `--auto medium` | `permission: {"read": "allow", "edit": "allow", ...}` |
| `autonomy: "low"` | `--auto low` | `permission: {"read": "allow", "edit": "ask", ...}` |
| `model` | `-m <model>` flag | `model: "provider/model"` |
| `reasoning` | `-r <level>` flag | Model variants/thinking config |

## Benefits

1. **Single source of truth** - All settings in one place
2. **Runtime portability** - Switch runtimes without reconfiguration  
3. **Simpler API** - `project_create` params map directly to config
4. **OpenCode support** - Generated config unblocks OpenCode runtime
5. **Consistency** - Both runtimes always have valid configs

### 5. Read-Only Config Mounts

Runtime config files are mounted **read-only** in containers to prevent agents from modifying their own configuration:

- `config.json` - Read-only (canonical config)
- `.factory/mcp.json` - Read-only (Droid MCP config)
- `.factory/settings.json` - Read-only (Droid settings)
- `opencode.json` - Read-only (OpenCode config)

Configuration can only be changed by:
- Admin via MCP tools (future `project_update` tool)
- Direct host filesystem access

## Out of Scope

- Per-workspace config overrides (future enhancement)
- Runtime config hot-reload (regenerate on settings change)
- `project_update` MCP tool for modifying config (future enhancement)
