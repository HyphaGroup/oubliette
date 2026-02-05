# GitHub Account Registry

Registry of named GitHub accounts for project creation.

## ADDED Requirements

### Requirement: GitHub accounts MUST be stored in a configuration file

The system SHALL load GitHub accounts from `config/github-accounts.json`.

#### Scenario: Load accounts from config file

- Given a valid `config/github-accounts.json` file exists
- When the server starts
- Then accounts are loaded into memory
- And tokens are available for project creation

#### Scenario: Missing config file is handled gracefully

- Given no `config/github-accounts.json` file exists
- When the server starts
- Then the server starts successfully
- And a warning is logged
- And the account registry is empty

### Requirement: Project options MUST be retrievable via MCP tool

The `project_options` tool SHALL return all project configuration options including GitHub accounts.

#### Scenario: Get project options with accounts

- Given accounts "hyphadev" and "personal" exist in the registry
- When calling `project_options`
- Then response includes `github_accounts` section
- And `github_accounts.available` lists account names and descriptions
- And `github_accounts.default` indicates the default account
- And response does not include any tokens

#### Scenario: Get project options with empty registry

- Given the account registry is empty
- When calling `project_options`
- Then response includes `github_accounts` section
- And `github_accounts.available` is an empty array

### Requirement: Projects MUST be creatable with a named GitHub account

The `project_create` tool SHALL accept a `github_account` parameter.

#### Scenario: Create project with named account

- Given account "hyphadev" exists with a valid token
- When calling `project_create` with `github_account="hyphadev"`
- Then the project is created using hyphadev's token
- And the token is stored in the project's environment

#### Scenario: Invalid account name rejected

- Given account "nonexistent" does not exist
- When calling `project_create` with `github_account="nonexistent"`
- Then the request fails with a validation error
- And the error lists available account names

### Requirement: Token resolution MUST follow precedence order

The system SHALL resolve GitHub tokens in order: explicit token > named account > registry default > none.

#### Scenario: Explicit token overrides account

- Given account "hyphadev" exists
- When calling `project_create` with `github_token="ghp_explicit"` and `github_account="hyphadev"`
- Then the explicit token "ghp_explicit" is used
- And the account's token is not used

#### Scenario: Default account used when no token specified

- Given "hyphadev" is set as default account in the registry
- When calling `project_create` without `github_token` or `github_account`
- Then hyphadev's token is used

#### Scenario: No token when registry empty

- Given the account registry is empty or missing
- When calling `project_create` without `github_token` or `github_account`
- Then the project is created without a GitHub token

### Requirement: DEFAULT_GITHUB_TOKEN env var MUST be removed

The system SHALL no longer support the `DEFAULT_GITHUB_TOKEN` environment variable.

#### Scenario: Env var is ignored

- Given `DEFAULT_GITHUB_TOKEN` is set in environment
- And no account registry exists
- When calling `project_create` without `github_token` or `github_account`
- Then the project is created without a GitHub token
- And the env var is not used
