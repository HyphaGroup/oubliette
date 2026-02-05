# Tasks: Add Instance Manager

## Core Infrastructure

- [x] 1. Create `manager.sh` skeleton with command parsing
  - Subcommands: create, start, stop, restart, update, rollback, status, logs, delete, prune-releases
  - Help text and usage documentation
  - Error handling framework
  - Auto-initialize releases/ and instances/ directories on first use

- [x] 2. Add git and build utilities
  - Function to checkout specific version
  - Function to check if version tag exists
  - Function to run ./build.sh and capture output
  - Function to detect current git commit/tag

## Release Management

- [x] 3. Implement release building (internal function, not a command)
  - Check if release already exists in `releases/<version>/`
  - If exists, skip build and use existing
  - If not, checkout version and run `./build.sh`
  - Export container image to tar: `docker save oubliette:latest -o container-image.tar` or `container image save`
  - Copy binaries + image tar to `releases/<version>/`
  - Create release manifest with build date, git commit
  - Return to previous git state after build

- [x] 4. Add release validation
  - Check all required binaries exist (oubliette-server, oubliette-client, oubliette-relay, oubliette-token)
  - Verify container image tar is valid
  - Verify release manifest is complete

- [x] 5. Implement `prune-releases` command
  - Keep N most recent releases (default: 5)
  - Never delete releases currently in use by instances
  - Confirm before deletion (--yes to skip)

## Instance Configuration

- [x] 6. Create instance config schema (YAML)
  - Server settings (port, address)
  - Paths (projects, data, logs)
  - Recursion limits
  - Runtime preferences (docker/apple-container)
  - Reference to release version

- [x] 7. Implement `create` command (unified deployment)
  - Parse parameters: name (required), version (optional), port (optional), env vars (optional)
  - If version not specified, auto-detect latest git tag
  - If port not specified, auto-assign next available starting from 8081
  - If FACTORY_API_KEY not provided, prompt interactively
  - Validate version tag exists in git
  - Validate port not already in use by another instance
  - Build release if not exists (calls internal build function)
  - Create instance directory structure
  - Generate config.yaml from template (includes version reference)
  - Generate .env file with secrets
  - Generate systemd/launchd service
  - Start the instance
  - Wait for health check
  - Report success with instance URL, version, and port

## Service Management

- [x] 8. Implement systemd service generation (Linux)
  - Template: `/etc/systemd/system/oubliette-<instance>.service`
  - ExecStart points to `releases/<version>/oubliette-server`
  - WorkingDirectory points to instance directory
  - Auto-restart on failure
  - User/group configuration
  - `systemctl daemon-reload` after creation

- [x] 9. Implement launchd service generation (macOS)
  - Template: `/Library/LaunchDaemons/ai.factory.oubliette.<instance>.plist`
  - Similar configuration to systemd
  - `launchctl load` after creation

- [x] 10. Implement service detection (auto-detect systemd vs launchd)

## Lifecycle Commands

- [x] 11. Implement `start` command
  - Load/start service via systemd or launchd
  - Wait for health check to pass
  - Report success/failure

- [x] 12. Implement `stop` command
  - Stop service gracefully
  - Wait for shutdown confirmation
  - Timeout after 30 seconds

- [x] 13. Implement `restart` command
  - Stop then start
  - Wait for health check

## Update & Rollback

- [x] 14. Implement `update` command
  - Validate target version tag exists in git
  - Build release if not exists (calls internal build function)
  - Stop instance
  - Update config.yaml to point to new version
  - Load container image from new release
  - Start instance
  - Wait for health check
  - Rollback on health check failure

- [x] 15. Implement `rollback` command
  - Read previous version from instance history
  - Update config to previous version
  - Restart instance
  - Update history

- [x] 16. Implement version history tracking
  - Store version history in `instances/<name>/.version-history`
  - Track: version, updated_at, updated_by
  - Support rollback to any previous version

## Monitoring

- [x] 17. Implement `status` command
  - Single instance: detailed status (version, uptime, health, port)
  - All instances: summary table
  - Query health endpoint for each instance
  - Show project/session counts from metrics

- [x] 18. Implement `logs` command
  - Tail logs via journalctl (systemd) or log file (launchd)
  - Support --tail N, --follow, --since

## Cleanup

- [x] 19. Implement `delete` command
  - Stop service
  - Remove systemd/launchd service file
  - Delete instance directory
  - Confirm before deletion (--force to skip)

## Testing

- [x] 20. ~~Add integration tests for manager.sh~~ (deferred to `add-external-surface-coverage-gate`)

## Documentation

- [x] 21. Create `docs/INSTANCE_MANAGER.md`
  - Complete command reference
  - Example workflows
  - Troubleshooting guide

- [x] 22. Update AGENTS.md with instance manager reference
  - Added Quick Start commands for instance management
  - Reference to docs/INSTANCE_MANAGER.md

- [x] 23. Add example instance configs
  - `examples/instance-configs/ant-production.yaml`
  - `examples/instance-configs/internal-tools.yaml`
