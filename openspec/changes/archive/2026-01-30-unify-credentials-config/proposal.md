# Change: Unify Credentials Configuration

## Why

Currently, credentials are scattered across multiple config files with inconsistent patterns:
- `config/factory.json` - Single Factory API key (no named accounts)
- `config/github-accounts.json` - Named accounts with default
- `config/models.json` - API keys embedded per model definition
- `ANTHROPIC_API_KEY` env var - Only way to pass provider keys to containers

This creates problems:
1. No way to use different provider API keys per project
2. Factory credentials can't have named accounts like GitHub does
3. API keys in `models.json` are never actually passed to containers
4. Inconsistent patterns make it hard to add new credential types

## What Changes

Consolidate all credentials into a single `config/credentials.json` with a unified pattern:
- Named credentials with descriptions
- Default credential per type
- Project-level credential references (not raw keys)
- Credentials passed to containers based on project config

**Files to Create:**
- `config/credentials.json` - Unified credential storage
- `config/credentials.json.example` - Example with documentation
- `internal/config/credentials.go` - Credential loading and registry

**Files to Modify:**
- `internal/config/loader.go` - Load unified credentials, remove old loaders
- `internal/mcp/handlers_container.go` - Pass credentials based on project config
- `internal/mcp/handlers_project.go` - Replace github_account with credential_refs
- `internal/mcp/server.go` - Use unified credential registry
- `internal/project/types.go` - Add CredentialRefs to project
- `cmd/server/main.go` - Load credentials via new system

**Files to Delete:**
- `config/factory.json` and `config/factory.json.example`
- `config/github-accounts.json.example`
- `internal/config/github_accounts.go`
- `internal/config/github_accounts_test.go`

**Files to Update:**
- `config/models.json.example` - Remove `apiKey` field from model definitions

## Impact

- Affected specs: None (new capability)
- Affected code: config loading, container creation, project creation
- **BREAKING**: Old config files no longer supported. Users must create `credentials.json`.
