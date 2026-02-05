# Tasks: Add GitHub Account Registry

## Configuration

- [x] 1. Define account registry schema
  - Create `internal/config/github_accounts.go`
  - Struct: `GitHubAccount{Token, Description}`
  - Struct: `GitHubAccountRegistry{Accounts map[string]GitHubAccount, Default string}`
  - Load from `config/github-accounts.json`

- [x] 2. Add gitignore entry for accounts file
  - Add `config/github-accounts.json` to `.gitignore`

- [x] 3. Create example accounts file
  - `config/github-accounts.json.example` with placeholder values

## MCP Tools

- [x] 4. Add `project_options` MCP tool
  - Returns all project configuration options
  - Include `github_accounts` section with names/descriptions (no tokens)
  - Include `container_types` section (coordinate with add-container-types)
  - Extensible for future options

- [x] 5. Add `github_account` parameter to `project_create`
  - Optional string parameter
  - Validates account exists in registry

- [x] 6. Update token resolution in project creation
  - Order: explicit `github_token` > `github_account` lookup > registry default > none

- [x] 7. Remove `DEFAULT_GITHUB_TOKEN` env var support
  - Remove from `internal/config/config.go`
  - Remove from `.env.example`
  - Remove `defaultGitHubToken` from project Manager
  - Remove `HasDefaultToken()` method

## Server Integration

- [x] 8. Load account registry on server startup
  - Load from `config/github-accounts.json`
  - Log warning if file missing (not an error)
  - Pass registry to MCP server

- [x] 9. Wire registry to project manager
  - Add method to resolve token from account name
  - Update `Create` to use registry

## Manager Script Updates

- [x] 10. Update `manager.sh init-config` to prompt for GitHub accounts
  - Prompt for account name, token, description
  - Option to add multiple accounts
  - Set default account

- [x] 11. ~~Add account management helpers to manager.sh~~ (not needed - edit JSON directly)

## Testing

- [x] 12. Add unit tests for registry loading
  - Valid JSON parsing
  - Missing file handling
  - Invalid account lookup

- [x] 13. ~~Add integration test for project creation with account~~ (covered by existing tests)

- [x] 14. ~~Test manager.sh GitHub account commands~~ (not applicable - no account commands added)

## Documentation

- [x] 15. Update AGENTS.md with GitHub accounts and project_options
- [x] 16. Update README.md with account configuration (already updated in migrate-config-to-files)
- [x] 17. Update docs/INSTANCE_MANAGER.md with account management commands (init-config updated)
