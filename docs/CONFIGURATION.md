# Configuration

This document covers all configuration options for Oubliette.

## Config File

All configuration is in a single JSONC file:

```
~/.oubliette/config/oubliette.jsonc
```

Created by `oubliette init`, or copy from `config/oubliette.jsonc.example`.

## Full Configuration Reference

```jsonc
{
  // Server settings
  "server": {
    "address": ":8080",              // Listen address
    "agent_runtime": "auto",         // auto, droid, or opencode
    "droid": {
      "default_model": "sonnet"      // Model shorthand or full ID
    }
  },

  // API credentials
  "credentials": {
    "factory": {
      "credentials": {
        "default": {
          "api_key": "fk-your-factory-api-key",
          "description": "Primary Factory account"
        }
      },
      "default": "default"
    },
    "github": {
      "credentials": {
        "personal": {
          "token": "ghp_your-github-token",
          "description": "Personal GitHub account"
        }
      },
      "default": "personal"
    },
    "providers": {
      "credentials": {
        "anthropic-main": {
          "provider": "anthropic",
          "api_key": "sk-ant-your-key",
          "description": "Main Anthropic account"
        }
      },
      "default": "anthropic-main"
    }
  },

  // Default settings for new projects/sessions
  "defaults": {
    "limits": {
      "max_recursion_depth": 3,
      "max_agents_per_session": 50,
      "max_cost_usd": 10.00
    },
    "agent": {
      "runtime": "droid",            // droid or opencode
      "model": "sonnet",             // Model shorthand
      "autonomy": "off",             // off, low, medium, high
      "reasoning": "medium"          // off, low, medium, high
    },
    "container": {
      "type": "dev"                  // Default container type
    },
    "backup": {
      "enabled": false,              // Enable automatic backups
      "directory": "data/backups",   // Backup directory
      "retention": 7,                // Days to keep
      "interval_hours": 24           // Hours between backups
    }
  },

  // Container type -> image name mappings
  "containers": {
    "base": "ghcr.io/hyphagroup/oubliette-base:latest",
    "dev": "ghcr.io/hyphagroup/oubliette-dev:latest"
  },

  // Model definitions
  "models": {
    "models": {
      "sonnet": {
        "model": "claude-sonnet-4-5",
        "displayName": "Sonnet 4.5",
        "baseUrl": "https://api.anthropic.com",
        "maxOutputTokens": 64000,
        "provider": "anthropic"
      }
    },
    "defaults": {
      "included_models": ["sonnet", "opus"],
      "session_model": "sonnet"
    }
  }
}
```

## Section Details

### server

| Field | Description | Default |
|-------|-------------|---------|
| `address` | Server listen address | `:8080` |
| `agent_runtime` | Runtime: `auto`, `droid`, `opencode` | `auto` |
| `droid.default_model` | Default model shorthand | `sonnet` |

### credentials

Multiple credentials per type, with a default:

| Type | Purpose | Env Var Injected |
|------|---------|------------------|
| `factory` | Factory AI Droid runtime | `FACTORY_API_KEY` |
| `github` | Repository cloning | `GITHUB_TOKEN` |
| `providers` | AI model access | Provider-specific |

Provider environment variables:
- `anthropic` → `ANTHROPIC_API_KEY`
- `openai` → `OPENAI_API_KEY`
- `google` → `GOOGLE_API_KEY`

### defaults.agent

| Field | Values | Description |
|-------|--------|-------------|
| `autonomy` | `off`, `low`, `medium`, `high` | Permission prompting level |
| `reasoning` | `off`, `low`, `medium`, `high` | Extended thinking budget |

### containers

Maps container type names to image references. Default images from ghcr.io:

```jsonc
"containers": {
  "base": "ghcr.io/hyphagroup/oubliette-base:latest",
  "dev": "ghcr.io/hyphagroup/oubliette-dev:latest"
}
```

Add custom container types:

```jsonc
"containers": {
  "base": "ghcr.io/hyphagroup/oubliette-base:latest",
  "dev": "ghcr.io/hyphagroup/oubliette-dev:latest",
  "custom": "my-registry.io/my-image:v1.0"
}
```

See [CONTAINER_TYPES.md](CONTAINER_TYPES.md) for details.

## Development Mode

Set `OUBLIETTE_DEV=1` to use locally-built images:

```bash
./build.sh                        # Build local images
OUBLIETTE_DEV=1 ./bin/oubliette   # Use local images
```

In dev mode, containers default to local names:
- `base` → `oubliette-base:latest`
- `dev` → `oubliette-dev:latest`

## Project Overrides

Projects can override defaults via `project_create`:

```json
{
  "name": "my-project",
  "max_recursion_depth": 5,
  "max_agents_per_session": 100,
  "max_cost_usd": 50.0,
  "autonomy": "medium",
  "reasoning": "high",
  "container_type": "base"
}
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OUBLIETTE_DEV` | Use local images if `1` | (unset) |
| `CONTAINER_RUNTIME` | `auto`, `docker`, `apple-container` | `auto` |

Most settings should be in `oubliette.jsonc` rather than environment variables.
