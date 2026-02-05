# Consolidate Config Files into Single oubliette.jsonc

## Problem

Configuration is currently split across 4 separate JSON files in `config/`:
- `server.json` - Server address, agent runtime, droid settings
- `credentials.json` - Factory, GitHub, and provider API keys  
- `config-defaults.json` - Limits, agent, container, backup defaults
- `models.json` - Model definitions and defaults

This is fragmented and requires managing multiple files. Additionally, JSON doesn't support comments, making it harder to document configuration options inline.

## Solution

Consolidate all configuration into a single `oubliette.jsonc` file that:
1. Uses JSONC format (JSON with Comments) for inline documentation
2. Contains all sections: server, credentials, defaults, models
3. Has sensible code defaults for optional fields

**Config precedence:**
1. `./config/oubliette.jsonc` (project-local, for development)
2. `~/.oubliette/config/oubliette.jsonc` (user global, for installed binary)

## New Config Structure

```jsonc
{
  // Server configuration
  "server": {
    "address": ":8080",
    "agent_runtime": "auto"  // auto, droid, opencode
  },

  // API credentials (keep these secret!)
  "credentials": {
    "factory": {
      "default": "main",
      "keys": {
        "main": "fk-your-key-here"
      }
    },
    "github": {
      "default": "main",
      "tokens": {
        "main": "ghp_your-token-here"
      }
    },
    "providers": {
      "default": "anthropic",
      "keys": {
        "anthropic": "sk-ant-your-key-here"
      }
    }
  },

  // Default limits for new projects
  "defaults": {
    "limits": {
      "max_recursion_depth": 3,
      "max_agents_per_session": 50,
      "max_cost_usd": 10.00
    },
    "agent": {
      "runtime": "droid",
      "model": "sonnet",
      "autonomy": "off",
      "reasoning": "medium"
    },
    "container": {
      "type": "dev"
    },
    "backup": {
      "enabled": false,
      "directory": "data/backups",
      "retention": 7,
      "interval_hours": 24
    }
  },

  // Model definitions
  "models": {
    "sonnet": {
      "model": "claude-sonnet-4-5",
      "provider": "anthropic",
      "max_output_tokens": 64000
    }
  }
}
```

## Changes

1. Delete all individual JSON config files and their loaders
2. Only `oubliette.jsonc` is supported - no fallback
3. `oubliette init` creates the new unified format
4. Existing deployments must migrate manually (one-time)

## Out of Scope

- Changing the config data model (fields, validation, defaults)
- Environment variable overrides
- Config hot-reloading
- Backwards compatibility with old format
