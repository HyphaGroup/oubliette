# container-images Specification

## Purpose
TBD - created by archiving change optimize-base-container-image. Update Purpose after archive.
## Requirements
### Requirement: Minimal Base Image Dependencies

The base container image SHALL include only the minimal set of system packages required for agent runtime operation.

**Required packages:**
- `ca-certificates` - TLS certificate verification
- `curl` - HTTP client for downloads and API calls
- `git` - Version control operations
- `bash` - Shell execution for agent tools
- `locales` - UTF-8 text handling
- `openssh-client` - SSH key authentication for git
- `procps` - Process management utilities (ps, kill, etc.)

**Explicitly excluded packages:**
- `wget` - curl is sufficient
- `gnupg` - not needed at runtime
- `sudo` - security risk, not needed
- `jq` - agents parse JSON internally

#### Scenario: Base image contains required packages
- **GIVEN** a running oubliette-base container
- **WHEN** checking for required commands
- **THEN** `curl --version` succeeds
- **AND** `git --version` succeeds
- **AND** `bash --version` succeeds
- **AND** `ssh -V` succeeds
- **AND** `ps --version` succeeds

#### Scenario: Base image excludes unnecessary packages
- **GIVEN** a running oubliette-base container
- **WHEN** checking for excluded commands
- **THEN** `which wget` returns not found
- **AND** `which gpg` returns not found
- **AND** `which sudo` returns not found
- **AND** `which jq` returns not found

### Requirement: Pre-installed Ripgrep

The base container image SHALL include ripgrep (rg) pre-installed to avoid runtime download delays.

The ripgrep version SHALL be 14.1.1 or later to match OpenCode's expected version.

#### Scenario: Ripgrep available immediately
- **GIVEN** a freshly started oubliette-base container
- **WHEN** executing `rg --version`
- **THEN** the command succeeds without download
- **AND** version is 14.1.1 or later

#### Scenario: Ripgrep works for code search
- **GIVEN** a running oubliette-base container with source code
- **WHEN** executing `rg "pattern" .`
- **THEN** matching files are returned

### Requirement: Both Agent Runtimes Functional

The base container image SHALL support both OpenCode and Factory Droid agent runtimes.

Both agents SHALL be installed and functional at container start time.

#### Scenario: OpenCode is functional
- **GIVEN** a running oubliette-base container as user gogol
- **WHEN** executing `opencode --version`
- **THEN** the command succeeds and outputs a version number

#### Scenario: Droid is functional
- **GIVEN** a running oubliette-base container as user gogol
- **WHEN** executing `droid --version`
- **THEN** the command succeeds and outputs a version number

#### Scenario: Agents can execute shell commands
- **GIVEN** a running oubliette-base container
- **WHEN** an agent executes a bash command via its shell tool
- **THEN** the command runs in bash shell
- **AND** output is captured correctly

### Requirement: Non-Root User Without Sudo

The base container image SHALL run as a non-root user (gogol) without sudo privileges.

This improves security by limiting container escape vectors.

#### Scenario: Container runs as non-root
- **GIVEN** a running oubliette-base container
- **WHEN** checking current user
- **THEN** `whoami` returns "gogol"
- **AND** `id -u` returns "1000"

#### Scenario: Sudo is not available
- **GIVEN** a running oubliette-base container
- **WHEN** attempting to use sudo
- **THEN** `sudo ls` fails with command not found
- **AND** no passwordless sudo is configured

### Requirement: Oubliette Infrastructure Binaries

The base container image SHALL include oubliette-relay and oubliette-client binaries.

#### Scenario: Relay binary exists
- **GIVEN** the oubliette-base container image
- **WHEN** checking for relay binary
- **THEN** `/usr/local/bin/oubliette-relay` exists
- **AND** is executable

#### Scenario: Client binary exists
- **GIVEN** the oubliette-base container image
- **WHEN** checking for client binary
- **THEN** `/usr/local/bin/oubliette-client` exists
- **AND** is executable

#### Scenario: Init script starts relay
- **GIVEN** a container started with OUBLIETTE_PROJECT_ID set
- **WHEN** container-init.sh runs
- **THEN** oubliette-relay starts in background
- **AND** relay PID is logged

### Requirement: Base Image Size Limit

The base container image SHALL be less than 400MB uncompressed.

This ensures fast pulls and reasonable storage usage.

#### Scenario: Image size within limit
- **GIVEN** the built oubliette-base:latest image
- **WHEN** checking image size with `docker images`
- **THEN** size is less than 400MB

### Requirement: Multi-Architecture Support

The base container image SHALL support both amd64 and arm64 architectures.

#### Scenario: Image builds on amd64
- **GIVEN** an amd64 build environment
- **WHEN** building oubliette-base image
- **THEN** build succeeds
- **AND** all binaries are amd64

#### Scenario: Image builds on arm64
- **GIVEN** an arm64 build environment (e.g., Apple Silicon)
- **WHEN** building oubliette-base image
- **THEN** build succeeds
- **AND** all binaries are arm64

