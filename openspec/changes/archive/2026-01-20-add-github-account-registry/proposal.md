# Proposal: Add GitHub Account Registry

## Summary

Add a registry of named GitHub accounts with tokens that can be selected when creating projects. Accounts are discoverable via the `project_options` MCP tool.

## Motivation

Currently, projects use either:
1. A `github_token` passed directly to `project_create`
2. A fallback `DEFAULT_GITHUB_TOKEN` from environment

**Problems:**
- No way to manage multiple GitHub accounts (personal, org, bot accounts)
- Tokens must be passed explicitly each time or hardcoded in env
- No visibility into which accounts are available
- Can't easily switch between accounts for different project types

## Proposed Solution

### Account Registry

Store named accounts in `config/github-accounts.json` (gitignored). Loaded once at server startup. Edit file directly to add/remove accounts (no MCP tool to modify - keeps secrets management simple).

```json
{
  "accounts": {
    "hyphadev": {
      "token": "ghp_xxx...",
      "description": "Hypha development bot"
    },
    "personal": {
      "token": "ghp_yyy...",
      "description": "Personal account"
    }
  },
  "default": "hyphadev"
}
```

### MCP Tools

**New tool: `project_options`** (shared with add-container-types)
- Returns all available project configuration options
- GitHub accounts included in response (never exposes tokens)

**Updated tool: `project_create`**
- New parameter: `github_account` (account name from registry)
- Precedence: `github_token` > `github_account` > registry default > none

### Usage

```json
// Get all project options (accounts, container types, etc.)
project_options()
// Returns:
// {
//   "github_accounts": {
//     "available": [{"name": "hyphadev", "description": "..."}],
//     "default": "hyphadev"
//   },
//   "container_types": { ... }
// }

// Create project with specific account
project_create({"name": "my-project", "github_account": "personal"})

// Still works: explicit token (overrides account)
project_create({"name": "my-project", "github_token": "ghp_xxx"})

// Still works: uses default account or env fallback
project_create({"name": "my-project"})
```

## Scope

### In Scope
- `config/github-accounts.json` schema and loading
- GitHub accounts in `project_options` response (never expose tokens)
- `github_account` parameter on `project_create`
- Token resolution: explicit > account > registry default > none
- Remove `DEFAULT_GITHUB_TOKEN` env var and related code
- Gitignore the accounts file

### Out of Scope
- MCP tools to add/remove accounts (edit JSON directly)
- Token validation/verification
- Multiple tokens per account (e.g., fine-grained vs classic)
- GitLab/Bitbucket accounts (future)

## Related Changes

- **add-container-types**: Also adds to `project_options` response
