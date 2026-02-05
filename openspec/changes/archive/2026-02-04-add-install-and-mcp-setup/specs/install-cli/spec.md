# Install Script and MCP Setup CLI Specification

## ADDED Requirements

### Requirement: Install script downloads and installs platform-specific binary

The system SHALL provide an `install.sh` script hosted at the GitHub repo that detects the user's OS and architecture, downloads the appropriate binary from GitHub Releases, verifies the checksum, and installs to the specified location.

**Platform Support:**
- OS: `darwin`, `linux`
- Architecture: `arm64`, `amd64`

**Default install location:** `~/.oubliette/bin/oubliette`

#### Scenario: Fresh install on macOS ARM

- Given user runs install script on darwin/arm64
- When prompted for install location
- And user accepts default
- Then binary is downloaded from GitHub Releases
- And checksum is verified
- And binary is installed to ~/.oubliette/bin/oubliette
- And user is prompted to run oubliette init

#### Scenario: Custom install location

- Given user runs install script
- When prompted for install location
- And user specifies /usr/local/bin
- Then binary is installed to /usr/local/bin/oubliette

#### Scenario: Unsupported platform

- Given user runs install script on Windows
- Then script exits with error code 1
- And displays "Unsupported platform" message

### Requirement: Init command creates directory structure and config

The system SHALL provide an `oubliette init` command that creates the necessary directory structure under `~/.oubliette/`, generates default configuration, and creates an initial admin token.

**Directories created:**
- `~/.oubliette/config/`
- `~/.oubliette/data/projects/`
- `~/.oubliette/data/logs/`
- `~/.oubliette/data/backups/`

**Config files created:**
- `~/.oubliette/config/server.json` - Server address, default model
- `~/.oubliette/config/credentials.json` - API keys (empty template)
- `~/.oubliette/config/config-defaults.json` - Project defaults

Note: The init command reuses the existing `auth.Store` from `internal/auth` to create tokens in `~/.oubliette/data/auth.db`.

#### Scenario: First-time init

- Given oubliette is not initialized
- When user runs oubliette init
- Then directories are created under ~/.oubliette/
- And config files are created with sensible defaults
- And admin token is generated and displayed
- And user is instructed to add Factory API key to credentials.json

#### Scenario: Already initialized

- Given oubliette is already initialized
- When user runs oubliette init
- Then user is warned about existing config
- And prompted to confirm overwrite or skip

### Requirement: MCP setup command configures AI tool integration

The system SHALL provide an `oubliette mcp --setup <tool>` command that configures MCP integration for the specified AI tool by creating an auth token and updating the tool's config file.

**Supported tools:**
- `droid` - Factory Droid (`~/.factory/mcp.json`)
- `claude` - Claude Desktop (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS, `~/.config/claude/claude_desktop_config.json` on Linux)
- `claude-code` - Claude Code VS Code extension (`~/.config/Code/User/globalStorage/anthropic.claude-code/settings.json`)

**MCP server entry format:**
```json
{
  "mcpServers": {
    "oubliette": {
      "command": "~/.oubliette/bin/oubliette",
      "args": ["--stdio"],
      "env": {
        "OUBLIETTE_TOKEN": "<generated-token>",
        "OUBLIETTE_DATA_DIR": "~/.oubliette/data"
      }
    }
  }
}
```

Note: Token creation reuses `auth.Store.CreateToken()` with scope "admin".

#### Scenario: Setup for Droid

- Given oubliette is initialized
- When user runs oubliette mcp --setup droid
- Then auth token is created if needed
- And ~/.factory/mcp.json is updated with oubliette server entry
- And success message shows the changes made

#### Scenario: Tool config does not exist

- Given ~/.factory/mcp.json does not exist
- When user runs oubliette mcp --setup droid
- Then file is created with oubliette entry
- And user is informed file was created

#### Scenario: Preserve existing config entries

- Given ~/.factory/mcp.json has other MCP servers configured
- When user runs oubliette mcp --setup droid
- Then oubliette entry is added or updated
- And other existing entries are preserved

### Requirement: Upgrade command updates to latest release

The system SHALL provide an `oubliette upgrade` command that checks for newer versions and upgrades the binary in-place.

**Version tracking:** The binary SHALL embed a version string at build time using `-ldflags "-X main.Version=v1.0.0"`. The `oubliette --version` flag SHALL print this version.

#### Scenario: Upgrade available

- Given oubliette v1.0.0 is installed
- And v1.1.0 is the latest GitHub Release
- When user runs oubliette upgrade
- Then latest version is downloaded
- And checksum is verified
- And binary is replaced in-place
- And success message shows old and new versions

#### Scenario: Already on latest

- Given oubliette v1.1.0 is installed
- And v1.1.0 is the latest GitHub Release
- When user runs oubliette upgrade
- Then message indicates already on latest version
- And no download occurs

#### Scenario: Check only

- Given oubliette v1.0.0 is installed
- And v1.1.0 is the latest GitHub Release
- When user runs oubliette upgrade --check
- Then message shows upgrade available
- And no download occurs

### Requirement: Version flag displays current version

The system SHALL provide an `oubliette --version` or `oubliette -v` flag that displays the current version.

#### Scenario: Display version

- Given oubliette v1.0.0 is installed
- When user runs oubliette --version
- Then output shows "oubliette v1.0.0"

### Requirement: Release artifacts published to GitHub Releases

The system SHALL build and publish platform-specific binaries to GitHub Releases with checksums.

**Artifacts:**
- `oubliette-darwin-arm64`
- `oubliette-darwin-amd64`
- `oubliette-linux-arm64`
- `oubliette-linux-amd64`
- `checksums.txt` (SHA256)

**Tag format:** `v1.0.0`

**Build flags:** Each binary SHALL be built with `-ldflags "-X main.Version=$TAG"` to embed the version.

#### Scenario: Release build

- Given a new version tag is pushed
- When GitHub Actions workflow runs
- Then binaries are built for all platforms with version embedded
- And checksums file is generated
- And all artifacts are attached to GitHub Release
