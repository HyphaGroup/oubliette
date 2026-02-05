# Proposal: Migrate Config to Files

## Summary

Replace `.env` configuration with structured JSON files in `config/` directory. Separate server runtime config from secrets and project defaults.

## Motivation

Current `.env` approach has issues:
- Mix of secrets (`FACTORY_API_KEY`) and non-secrets (`SERVER_ADDR`)
- Flat key-value pairs don't express structure well
- Can't easily track non-secret defaults in git
- Project limits are server-level when they should be project defaults

## Proposed Solution

### Config Directory Structure

```
config/
├── server.json              # Server runtime config (tracked)
├── server.json.example      # Example template (tracked)
├── factory.json             # Factory API key (gitignored)
├── factory.json.example     # Example template (tracked)
├── github-accounts.json     # GitHub accounts (gitignored) 
├── github-accounts.json.example
├── project-defaults.json    # Default project settings (tracked)
└── project-defaults.json.example
```

### server.json (tracked)

```json
{
  "address": ":8080",
  "droid": {
    "default_model": "claude-sonnet-4-5-20250929"
  }
}
```

### factory.json (gitignored)

```json
{
  "api_key": "fk_xxx"
}
```

### project-defaults.json (tracked)

```json
{
  "max_recursion_depth": 3,
  "max_agents_per_session": 50,
  "max_cost_usd": 10.00,
  "container_type": "dev"
}
```

### Project Creation

All project defaults become optional parameters on `project_create`:
- `max_recursion_depth`
- `max_agents_per_session`
- `max_cost_usd`
- `container_type` (from add-container-types)
- `github_account` (from add-github-account-registry)

### project_options Response

Include defaults so callers know what values will be used:

```json
{
  "github_accounts": { ... },
  "container_types": { ... },
  "defaults": {
    "max_recursion_depth": 3,
    "max_agents_per_session": 50,
    "max_cost_usd": 10.00,
    "container_type": "dev"
  }
}
```

## Scope

### In Scope
- Create `config/` directory structure
- `server.json` for runtime config
- `factory.json` for Factory API key
- `project-defaults.json` for project limit defaults
- Add project limit parameters to `project_create`
- Add `defaults` section to `project_options` response
- Remove `.env` file usage entirely
- Update config loading in server startup

### Out of Scope
- `github-accounts.json` (covered by add-github-account-registry)
- Container types in defaults (covered by add-container-types)

## Related Changes

- **add-github-account-registry**: Creates `github-accounts.json`, `project_options` tool
- **add-container-types**: Adds container_type to project creation and project_options
