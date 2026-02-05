# Token Subcommand Specification

## REMOVED Requirements

### Requirement: Separate oubliette-token binary

The `oubliette-token` binary SHALL be removed. Token management is consolidated into the main `oubliette` binary.

**Reason**: Simplify CLI by having a single unified binary.

**Migration**: Replace `oubliette-token <cmd>` with `oubliette token <cmd>`.

#### Scenario: Old binary no longer exists

- Given the oubliette project is built
- When checking for oubliette-token binary
- Then bin/oubliette-token does not exist
- And cmd/token/ directory does not exist

## ADDED Requirements

### Requirement: Token subcommand provides token management

The system SHALL provide an `oubliette token <action>` subcommand that manages authentication tokens with create, list, revoke, and info actions.

#### Scenario: Create token

- Given oubliette is initialized with a data directory
- When user runs oubliette token create --name "Test" --scope admin
- Then a new token is created
- And the token ID is displayed
- And user is warned to save it

#### Scenario: List tokens

- Given tokens exist in the auth store
- When user runs oubliette token list
- Then all tokens are displayed in tabular format
- And token IDs are masked for security

#### Scenario: Revoke token

- Given a token exists with ID oub_xxx
- When user runs oubliette token revoke oub_xxx
- Then the token is deleted from the auth store
- And confirmation message is displayed

#### Scenario: Get token info

- Given a token exists with ID oub_xxx
- When user runs oubliette token info oub_xxx
- Then token details are displayed (name, scope, created, last used)

#### Scenario: Show token help

- Given oubliette binary exists
- When user runs oubliette token without arguments
- Then usage information is displayed
- And available actions are listed
