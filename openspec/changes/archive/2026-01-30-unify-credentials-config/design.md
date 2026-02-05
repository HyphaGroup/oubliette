# Design: Unified Credentials Configuration

## Context

Oubliette needs to manage multiple types of credentials:
- **Factory API keys** - For Droid runtime
- **GitHub tokens** - For repository access
- **Provider API keys** - Anthropic, OpenAI, Google, etc. for agent runtimes

Currently these are stored in separate files with different patterns, and provider keys aren't properly passed to containers.

## Goals

1. Single file for all credentials (`config/credentials.json`)
2. Consistent pattern: named credentials with defaults per type
3. Project-level credential references (not raw keys in project config)
4. Proper credential injection into containers based on project config
5. Easy rotation - update one place, all referencing projects get new key

## Non-Goals

- Encrypted credential storage (use filesystem permissions)
- External secret management integration (Vault, etc.)
- Per-workspace credentials (project-level is sufficient)
- Backwards compatibility with old config files (rip and replace)

## Decisions

### Credential File Structure

```json
{
  "factory": {
    "credentials": {
      "default": {
        "api_key": "fk-xxx",
        "description": "Primary Factory account"
      },
      "backup": {
        "api_key": "fk-yyy",
        "description": "Backup Factory account"
      }
    },
    "default": "default"
  },
  "github": {
    "credentials": {
      "personal": {
        "token": "ghp_xxx",
        "description": "Personal GitHub account"
      },
      "orgbot": {
        "token": "ghp_yyy",
        "description": "Organization bot"
      }
    },
    "default": "personal"
  },
  "providers": {
    "credentials": {
      "anthropic-main": {
        "provider": "anthropic",
        "api_key": "sk-ant-xxx",
        "description": "Main Anthropic account"
      },
      "anthropic-client-a": {
        "provider": "anthropic",
        "api_key": "sk-ant-yyy",
        "description": "Client A's Anthropic key"
      },
      "openai-main": {
        "provider": "openai",
        "api_key": "sk-xxx",
        "description": "Main OpenAI account"
      }
    },
    "default": "anthropic-main"
  }
}
```

**Rationale**: 
- Separate sections for factory/github/providers allows type-specific validation
- Each section has same pattern: `credentials` map + `default` reference
- Provider credentials include `provider` field for env var mapping

### Project Credential References

Projects reference credentials by name, not raw values:

```json
// project_create params
{
  "name": "client-a-project",
  "credential_refs": {
    "factory": "default",
    "github": "orgbot",
    "provider": "anthropic-client-a"
  }
}
```

```json
// projects/<id>/metadata.json (stored)
{
  "name": "client-a-project",
  "credential_refs": {
    "factory": "default",
    "github": "orgbot", 
    "provider": "anthropic-client-a"
  }
}
```

**Rationale**:
- No secrets in project directories
- Easy to rotate - update credential, all referencing projects get new key
- Can share credentials or have project-specific ones

### Credential Resolution

When creating a container or using credentials:

1. **Project-specific ref** - `credential_refs.provider` from project metadata
2. **Type default** - `providers.default` from credentials.json

No environment variable fallback. All credentials must be in `credentials.json`.

### Environment Variable Mapping

Provider credentials map to container env vars:

| Provider | Environment Variable |
|----------|---------------------|
| `anthropic` | `ANTHROPIC_API_KEY` |
| `openai` | `OPENAI_API_KEY` |
| `google` | `GOOGLE_API_KEY` |

Factory credentials set `FACTORY_API_KEY`.

### Old Files Removed

The following are deleted with no fallback:
- `config/factory.json` - Use `credentials.factory` section
- `config/github-accounts.json` - Use `credentials.github` section
- `internal/config/github_accounts.go` - Logic moved to credentials.go
- `apiKey` field in `models.json` - Use `credentials.providers` section

Server will error on startup if `credentials.json` is missing required sections.

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| Breaking change | Clear error message pointing to credentials.json.example |
| Credentials in plaintext | Document filesystem permissions (0600), same as current |

## Decisions

1. **Per-workspace credentials?** No, project-level is sufficient. Workspaces inherit from project.

2. **Keep apiKey in models.json?** No, remove to avoid duplication. Credentials only in credentials.json.

3. **Keep github_account param?** No, replace with credential_refs.github. Remove old param entirely.
