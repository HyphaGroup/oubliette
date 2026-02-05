# Proposal: Add Instance Manager

## Summary

Add a management CLI (`manager.sh`) to simplify deploying and updating multiple production Oubliette instances from a single repository clone.

## Motivation

**Current deployment approach:**
- Each production instance requires a full git clone (~50MB)
- Each instance requires separate build step
- Updates require `git pull` + rebuild in every instance
- No centralized view of instance health
- Rollback requires manual `git revert` + rebuild
- No automated systemd service creation

**Example current workflow for 3 instances:**
```bash
# Instance 1: Ant Production
cd ~/ant-production
git clone https://github.com/HyphaGroup/oubliette.git .
git checkout v1.2.3
./build.sh
./oubliette-server

# Instance 2: Internal Tools  
cd ~/internal-tools
git clone https://github.com/HyphaGroup/oubliette.git .
git checkout v1.2.3
./build.sh
./oubliette-server

# Instance 3: Customer A
cd ~/customer-a
git clone https://github.com/HyphaGroup/oubliette.git .
git checkout v1.2.3
./build.sh
./oubliette-server

# To update all 3:
cd ~/ant-production && git pull && ./build.sh && restart
cd ~/internal-tools && git pull && ./build.sh && restart
cd ~/customer-a && git pull && ./build.sh && restart
```

**Problems:**
- 150MB disk for 3 clones (vs 50MB for one)
- 3× build time on updates
- Manual service management
- No rollback mechanism
- Config drift across instances

## Proposed Solution

Create `manager.sh` CLI that provides:

1. **Centralized deployment** - One git clone, multiple instances
2. **Immutable releases** - Build once, deploy everywhere
3. **Instance lifecycle** - Create, start, stop, update, rollback
4. **Service automation** - Auto-generate systemd/launchd services
5. **Health monitoring** - Status overview across all instances
6. **Zero-downtime updates** - Atomic updates via symlinks

### Directory Structure

```
~/oubliette-deploy/           # Single deployment root
├── .git/                     # One git clone
├── manager.sh                # Management CLI (new)
├── releases/                 # Immutable release artifacts (new)
│   ├── v1.2.3/
│   │   ├── oubliette-server
│   │   ├── oubliette-client
│   │   ├── oubliette-relay
│   │   ├── oubliette-token
│   │   └── container-image.tar
│   └── v1.2.4/
└── instances/                # Instance data directories (new)
    ├── ant-production/
    │   ├── config.yaml       # Instance config (port, paths, limits)
    │   ├── .env              # Secrets (gitignored)
    │   ├── projects/         # Instance data
    │   ├── data/
    │   └── logs/
    └── internal-tools/
        └── ...
```

### Key Commands

```bash
# Create instance with all defaults (latest tag, auto-assign port, prompts for required env vars)
./manager.sh create ant-production

# Override defaults as needed
./manager.sh create ant-production --version v1.2.3 --port 8081 --env FACTORY_API_KEY=fk_xxx

# This automatically:
# 1. Detects latest tag (or uses --version)
# 2. Assigns next available port starting at 8081 (or uses --port)
# 3. Prompts for FACTORY_API_KEY if not provided via --env
# 4. git checkout <version>
# 5. Builds release if not exists
# 6. Creates instance config
# 7. Generates systemd/launchd service
# 8. Starts the instance

# Lifecycle management
./manager.sh start ant-production
./manager.sh stop ant-production
./manager.sh restart ant-production

# Update to new version (also handles build if needed)
./manager.sh update ant-production --version v1.2.4
./manager.sh update --all --version v1.2.4

# Rollback to previous version
./manager.sh rollback ant-production

# Monitoring
./manager.sh status [instance]
./manager.sh logs ant-production --tail 50

# Cleanup
./manager.sh delete ant-production
./manager.sh prune-releases --keep 5
```

## Scope

### In Scope
- `manager.sh` script with core commands
- Release artifact creation (binaries + container image)
- Instance configuration (YAML + .env)
- Systemd service generation (Linux)
- Launchd service generation (macOS)
- Health check integration
- Instance data isolation
- Rollback mechanism

### Out of Scope (Future)
- Docker Compose integration
- Kubernetes manifests
- Multi-host deployment
- Blue/green deployments
- Automated backups (use existing backup system)

## Alternatives Considered

1. **Docker Compose per instance** - More complex, requires containerizing server itself
2. **Kubernetes** - Overkill for single-host deployments
3. **Keep current approach** - Simple but doesn't scale beyond 2-3 instances
4. **Binary distribution via package manager** - Complex to maintain, less flexible

## Decision

Implement `manager.sh` as a bash script for simplicity and portability. Start with core lifecycle operations, add advanced features iteratively.
