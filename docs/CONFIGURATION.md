# Configuration

All configuration is in a single JSONC file: `config/oubliette.jsonc` (or `~/.oubliette/config/oubliette.jsonc` when installed).

Created by `oubliette init`, or copy from `config/oubliette.jsonc.example`.

## Full Reference

```jsonc
{
  "server": {
    "address": ":8080"
  },

  "credentials": {
    "github": {
      "credentials": {
        "default": {
          "token": "ghp_xxx",
          "description": "GitHub account"
        }
      },
      "default": "default"
    },
    "providers": {
      "credentials": {
        "anthropic": {
          "provider": "anthropic",
          "api_key": "sk-ant-xxx",
          "description": "Anthropic API"
        }
      },
      "default": "anthropic"
    }
  },

  "defaults": {
    "limits": {
      "max_recursion_depth": 3,
      "max_agents_per_session": 50,
      "max_cost_usd": 10.00
    },
    "agent": {
      "model": "opus",
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

  "containers": {
    "base": "ghcr.io/hyphagroup/oubliette-base:latest",
    "dev": "ghcr.io/hyphagroup/oubliette-dev:latest"
  },

  "models": {
    "models": {
      "opus": {
        "model": "claude-opus-4-6",
        "displayName": "Opus 4.6 1M",
        "baseUrl": "https://api.anthropic.com",
        "maxOutputTokens": 128000,
        "provider": "anthropic",
        "extraHeaders": {
          "anthropic-beta": "context-1m-2025-08-07"
        }
      },
      "sonnet": {
        "model": "claude-sonnet-4-5",
        "displayName": "Sonnet 4.5",
        "provider": "anthropic"
      }
    },
    "defaults": {
      "session_model": "opus"
    }
  }
}
```

## Credentials

| Type | Purpose | Env Var Injected |
|------|---------|------------------|
| `github` | Repository cloning | `GITHUB_TOKEN` |
| `providers` | AI model access | Provider-specific (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, etc.) |

At least one provider credential is required.

## Agent Defaults

| Field | Values | Description |
|-------|--------|-------------|
| `model` | Model alias from `models` section | Which model to use |
| `autonomy` | `off`, `low`, `medium`, `high` | Permission prompting level |
| `reasoning` | `off`, `low`, `medium`, `high` | Extended thinking budget |

## Model Extra Headers

Models can include custom HTTP headers via `extraHeaders`. Used for beta features like Anthropic's 1M context window:

```jsonc
"extraHeaders": {
  "anthropic-beta": "context-1m-2025-08-07"
}
```

## Container Types

Maps container type names to image references:

```jsonc
"containers": {
  "base": "ghcr.io/hyphagroup/oubliette-base:latest",
  "dev": "ghcr.io/hyphagroup/oubliette-dev:latest",
  "custom": "my-registry.io/my-image:v1.0"
}
```

See [CONTAINER_TYPES.md](CONTAINER_TYPES.md) for details.

## Development Mode

```bash
./build.sh                        # Build local images
OUBLIETTE_DEV=1 ./bin/oubliette   # Use local images
```

In dev mode, containers resolve to local names: `oubliette-base:latest`, `oubliette-dev:latest`.

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OUBLIETTE_DEV` | Use locally-built images | (unset) |
| `OUBLIETTE_HOME` | Config/data directory | `~/.oubliette` |
| `CONTAINER_RUNTIME` | `auto`, `docker`, `apple-container` | `auto` |

## Config Precedence

The server locates its config directory in this order:
1. `--dir` flag
2. `OUBLIETTE_HOME` environment variable
3. `./.oubliette` (current directory)
4. `~/.oubliette` (home directory)
