# Tasks: Unify Credentials Configuration

## 1. Credential Types and Loading

- [x] 1.1 Create `internal/config/credentials.go` with types
- [x] 1.2 Implement `LoadCredentials(configDir string)` function
- [x] 1.3 Add credential registry methods

## 2. Delete Old Code

- [x] 2.1 Delete `config/factory.json` (if exists)
- [x] 2.2 Delete `config/factory.json.example`
- [x] 2.3 Delete `config/github-accounts.json.example`
- [x] 2.4 Delete `internal/config/github_accounts.go`
- [x] 2.5 Delete `internal/config/github_accounts_test.go`
- [x] 2.6 Remove `LoadFactoryConfig()` function from loader.go
- [x] 2.7 Remove `LoadGitHubAccounts()` function from loader.go
- [x] 2.8 Remove `apiKey` field from `config/models.json.example`

## 3. Project Credential References

- [x] 3.1 Add `CredentialRefs` to `project.Project` type
- [x] 3.2 Update `CreateProjectParams` to accept `credential_refs`
- [x] 3.3 Update `handleCreateProject()` with credential validation
- [x] 3.4 Update project metadata serialization to include credential_refs

## 4. Container Credential Injection

- [x] 4.1 Update `ensureContainerRunning()` to use credential registry
- [x] 4.2 Resolve provider credential for project
- [x] 4.3 Map provider to env var and inject
- [x] 4.4 Inject FACTORY_API_KEY based on credential resolution
- [x] 4.5 Remove `os.Getenv("ANTHROPIC_API_KEY")` fallback

## 5. Server Integration

- [x] 5.1 Update `ServerConfig` to use `CredentialRegistry`
- [x] 5.2 Update `NewServer()` to accept unified credential registry
- [x] 5.3 Update `cmd/server/main.go` to load credentials via new system
- [x] 5.4 Remove separate githubAccounts field from Server struct
- [x] 5.5 Remove FactoryConfig from Server struct

## 6. project_options Update

- [x] 6.1 Update `handleProjectOptions()` to list credentials
- [x] 6.2 Include all three types: factory, github, providers
- [x] 6.3 Show defaults for each type
- [x] 6.4 Never expose actual keys/tokens in response

## 7. Testing

- [x] 7.1-7.4 Covered by integration tests
- [x] 7.5 Verified 100% MCP tool coverage

## 8. Documentation

- [x] 8.1 Create `config/credentials.json.example`
- [x] 8.2 Update docs/CONFIGURATION.md with new credential system
- [x] 8.3 AGENTS.md references docs, no changes needed
