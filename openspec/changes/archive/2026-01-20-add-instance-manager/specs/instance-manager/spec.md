# Spec: Instance Manager

## ADDED Requirements

### Requirement: Automatic release building

The system MUST automatically build releases when needed during create/update operations.

#### Scenario: Build release during instance creation
- **GIVEN** a git repository with tag v1.2.3
- **AND** no release exists for v1.2.3 in `releases/v1.2.3/`
- **WHEN** `./manager.sh create ant-production --version v1.2.3 --port 8081` is executed
- **THEN** git is checked out to v1.2.3
- **AND** `./build.sh` is executed
- **AND** binaries and container image are copied to `releases/v1.2.3/`
- **AND** git is returned to previous state
- **AND** instance creation continues

#### Scenario: Reuse existing release
- **GIVEN** a release v1.2.3 already exists in `releases/v1.2.3/`
- **WHEN** `./manager.sh create internal-tools --version v1.2.3 --port 8082` is executed
- **THEN** the existing release is used
- **AND** no build occurs
- **AND** instance creation continues immediately

### Requirement: Instance creation and configuration

The system MUST support creating isolated instances with separate configurations.

#### Scenario: Create instance with all defaults
- **GIVEN** a git repository with latest tag v1.2.3
- **AND** no instances exist yet
- **WHEN** `./manager.sh create ant-production` is executed
- **AND** user enters FACTORY_API_KEY when prompted
- **THEN** latest tag v1.2.3 is detected automatically
- **AND** port 8081 is auto-assigned (first available)
- **AND** release v1.2.3 is built if not exists
- **AND** directory `instances/ant-production/` is created
- **AND** it contains `config.yaml` with server.address = ":8081"
- **AND** it contains `.env` with FACTORY_API_KEY
- **AND** it contains `projects/`, `data/`, `logs/` subdirectories
- **AND** `config.yaml` references release version v1.2.3
- **AND** systemd/launchd service is created
- **AND** the instance is started
- **AND** health check passes
- **AND** output shows "Instance ant-production running on http://localhost:8081 (v1.2.3)"

#### Scenario: Create instance with explicit version and port
- **GIVEN** a git repository with tag v1.2.3
- **WHEN** `./manager.sh create ant-production --version v1.2.3 --port 8082 --env FACTORY_API_KEY=fk_xxx` is executed
- **THEN** release v1.2.3 is built if not exists
- **AND** directory `instances/ant-production/` is created
- **AND** it contains `config.yaml` with server.address = ":8082"
- **AND** the instance is created and started

#### Scenario: Reject instance creation with nonexistent git tag
- **GIVEN** git tag v9.9.9 does not exist in the repository
- **WHEN** `./manager.sh create test-instance --version v9.9.9` is executed
- **THEN** the command fails with "Git tag v9.9.9 not found"
- **AND** no instance directory is created

#### Scenario: Reject instance creation with duplicate port
- **GIVEN** an instance exists on port 8081
- **WHEN** `./manager.sh create new-instance --port 8081` is executed
- **THEN** the command fails with "Port 8081 already in use"
- **AND** no instance directory is created

### Requirement: Service management integration

The system MUST automatically create and manage systemd/launchd services for instances.

#### Scenario: Create systemd service on Linux
- **GIVEN** a Linux system with systemd
- **AND** instance ant-production is created
- **WHEN** the instance is created
- **THEN** a systemd service file is created at `/etc/systemd/system/oubliette-ant-production.service`
- **AND** the service ExecStart points to the correct release binary
- **AND** the service WorkingDirectory points to the instance directory
- **AND** `systemctl daemon-reload` is executed

#### Scenario: Create launchd service on macOS
- **GIVEN** a macOS system with launchd
- **AND** instance ant-production is created
- **WHEN** the instance is created
- **THEN** a launchd plist is created at `/Library/LaunchDaemons/ai.factory.oubliette.ant-production.plist`
- **AND** the plist ProgramArguments point to the correct release binary

### Requirement: Instance lifecycle operations

The system MUST support starting, stopping, and restarting instances.

#### Scenario: Start instance
- **GIVEN** an instance ant-production exists and is stopped
- **WHEN** `./manager.sh start ant-production` is executed
- **THEN** the systemd/launchd service is started
- **AND** the command waits for health check at the configured port
- **AND** success is reported when health check passes

#### Scenario: Start fails if health check times out
- **GIVEN** an instance that fails to start properly
- **WHEN** `./manager.sh start failing-instance` is executed
- **THEN** the command waits up to 30 seconds for health check
- **AND** the command fails with "Health check timeout"
- **AND** the service is stopped

### Requirement: Instance updates with rollback

The system MUST support updating instances to new versions with automatic rollback on failure.

#### Scenario: Update instance to new version
- **GIVEN** instance ant-production is running v1.2.3
- **AND** release v1.2.4 exists
- **WHEN** `./manager.sh update ant-production --version v1.2.4` is executed
- **THEN** the instance is stopped gracefully
- **AND** `config.yaml` is updated to reference v1.2.4
- **AND** the container image for v1.2.4 is loaded
- **AND** the instance is started
- **AND** health check is performed
- **AND** success is reported

#### Scenario: Update with health check failure triggers rollback
- **GIVEN** instance ant-production is running v1.2.3
- **AND** release v1.2.4 exists but has a bug
- **WHEN** `./manager.sh update ant-production --version v1.2.4` is executed
- **AND** the health check fails after update
- **THEN** the instance is automatically rolled back to v1.2.3
- **AND** the command fails with "Update failed, rolled back to v1.2.3"

#### Scenario: Manual rollback to previous version
- **GIVEN** instance ant-production is running v1.2.4
- **AND** version history shows previous version was v1.2.3
- **WHEN** `./manager.sh rollback ant-production` is executed
- **THEN** the instance is updated to v1.2.3
- **AND** health check passes
- **AND** success is reported

### Requirement: Instance status monitoring

The system MUST provide status information for instances.

#### Scenario: Query single instance status
- **GIVEN** instance ant-production is running on port 8081
- **WHEN** `./manager.sh status ant-production` is executed
- **THEN** output includes instance name, version, status (running/stopped), port, health status
- **AND** output includes uptime if running

#### Scenario: Query all instances status
- **GIVEN** multiple instances exist (ant-production, internal-tools)
- **WHEN** `./manager.sh status` is executed without instance name
- **THEN** output shows a table with all instances
- **AND** each row includes name, version, status, port, health

### Requirement: Release cleanup

The system MUST support pruning old releases while preserving active ones.

#### Scenario: Prune old releases
- **GIVEN** releases v1.2.1, v1.2.2, v1.2.3, v1.2.4, v1.2.5, v1.2.6 exist
- **AND** instance ant-production uses v1.2.6
- **AND** instance internal-tools uses v1.2.5
- **WHEN** `./manager.sh prune-releases --keep 3` is executed
- **THEN** releases v1.2.1, v1.2.2, v1.2.3 are deleted
- **AND** releases v1.2.4, v1.2.5, v1.2.6 are kept
- **AND** confirmation is requested before deletion

#### Scenario: Prevent pruning releases in use
- **GIVEN** releases v1.2.1, v1.2.2 exist
- **AND** instance ant-production uses v1.2.1
- **WHEN** `./manager.sh prune-releases --keep 1` is executed
- **THEN** v1.2.1 is NOT deleted (in use)
- **AND** v1.2.2 MAY be deleted if not in use
