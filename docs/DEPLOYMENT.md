# Oubliette Deployment Guide

This guide covers deploying Oubliette to production environments.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start (Development)](#quick-start-development)
- [Production Deployment](#production-deployment)
- [Configuration](#configuration)
- [TLS/HTTPS Setup](#tlshttps-setup)
- [Monitoring Setup](#monitoring-setup)
- [Backup and Recovery](#backup-and-recovery)
- [Health Checks](#health-checks)
- [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Container Runtime (Required)

Oubliette manages **project containers** using either Docker or Apple Container. The Oubliette server itself runs as a native binary on the host.

#### Apple Container (Recommended for macOS)

- **macOS only**: Built by Apple, uses macOS Virtualization framework
- **Better performance**: Native macOS integration, faster I/O
- **Install**: `brew install apple/apple/container`
- **Start**: `container system start`

```bash
# Verify availability
container system status
```

#### Docker (Recommended for Linux)

- **Cross-platform**: Works on macOS, Linux, Windows
- **Install**: [Docker Engine](https://docs.docker.com/engine/install/) or [Docker Desktop](https://www.docker.com/products/docker-desktop/)

```bash
# Verify availability
docker ps
```

**Runtime Selection**: Oubliette auto-detects available runtime (prefers Apple Container on macOS ARM64). Override with `CONTAINER_RUNTIME` env var.

### Other Requirements

- **Go 1.24+**: For building binaries
- **Factory API Key**: Get from [Factory AI Settings](https://app.factory.ai/settings/api-keys)
- **GitHub Token** (optional): For project git operations - [Generate here](https://github.com/settings/tokens?type=beta)
  - Required permissions: `contents:write`, `pull_requests:write`, `issues:write`

### System Requirements

- CPU: 2+ cores (4+ recommended for production)
- RAM: 4GB minimum (8GB+ recommended)
- Disk: 20GB minimum (depends on project count)
- Network: Outbound HTTPS to Factory API and GitHub

---

## Quick Start (Development)

The fastest way to run Oubliette locally.

### Option A: Install Script (Recommended)

```bash
# Install binary
curl -fsSL https://raw.githubusercontent.com/HyphaGroup/oubliette/main/install.sh | bash

# Initialize configuration
oubliette init

# Add your API keys to ~/.oubliette/config/credentials.json

# Configure MCP for your AI tool
oubliette mcp --setup droid

# Start the server
oubliette --config-dir ~/.oubliette/config
```

See [INSTALLATION.md](INSTALLATION.md) for detailed setup instructions.

### Option B: Build from Source

```bash
git clone https://github.com/HyphaGroup/oubliette.git
cd oubliette

# Configure
cp config/credentials.json.example config/credentials.json
# Edit config/credentials.json with your API keys

# Build all binaries
./build.sh
```

### 2. Build Container Image

```bash
# Docker
docker build -f internal/container/Dockerfile -t oubliette-base:latest .

# Or Apple Container
container build -f internal/container/Dockerfile -t oubliette-base:latest .
```

### 3. Start Server

```bash
# Run server (auto-detects container runtime)
./oubliette-server

# Or use hot-reload development mode
./dev.sh
```

### 4. Verify

```bash
curl http://localhost:8080/health   # {"status":"ok"}
curl http://localhost:8080/ready    # {"status":"ready"}
```

### 5. Create Token

```bash
./oubliette token create --name dev-admin --scope admin
# Save the generated token
```

---

## Production Deployment

For production, run Oubliette as a system service with proper configuration.

### 1. Build and Install

```bash
git clone https://github.com/HyphaGroup/oubliette.git
cd oubliette
./build.sh

# Install binaries
sudo cp bin/oubliette /usr/local/bin/
sudo chmod +x /usr/local/bin/oubliette
```

### 2. Build Container Image

```bash
# Docker (Linux)
docker build -f internal/container/Dockerfile -t oubliette-base:latest .

# Apple Container (macOS)
container build -f internal/container/Dockerfile -t oubliette-base:latest .
```

### 3. Create Configuration

```bash
sudo mkdir -p /etc/oubliette
sudo nano /etc/oubliette/config.yaml
```

**Example config.yaml:**

```yaml
server:
  address: ":8080"

auth:
  factory_api_key: "fk_your_key_here"
  default_github_token: "github_pat_your_token_here"

runtime:
  preference: "auto"  # Or "docker" or "apple-container"
  memory: "4G"
  cpus: 4

recursion:
  max_depth: 3
  max_agents: 50
  max_cost_usd: 10.0

rate_limit:
  requests_per_second: 10.0
  burst: 20

cleanup:
  enabled: true
  interval_minutes: 5
  retention_minutes: 60

backup:
  enabled: true
  interval_hours: 24
  retention: 7
  directory: "/opt/oubliette/backups"

paths:
  projects: "/opt/oubliette/projects"
  data: "/opt/oubliette/data"
  logs: "/opt/oubliette/logs"
  sockets: "/var/run/oubliette-sockets"
```

### 4. Create System Service

#### Linux (systemd)

```bash
sudo nano /etc/systemd/system/oubliette.service
```

```ini
[Unit]
Description=Oubliette MCP Server
After=network.target docker.service
Requires=docker.service

[Service]
Type=simple
User=oubliette
Group=oubliette
WorkingDirectory=/opt/oubliette
SupplementaryGroups=docker
ExecStart=/usr/local/bin/oubliette-server -config /etc/oubliette/config.yaml
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/oubliette /var/run/oubliette-sockets /var/run/docker.sock
LimitNOFILE=65536
LimitNPROC=4096
Restart=on-failure
RestartSec=5s
StandardOutput=journal
StandardError=journal
SyslogIdentifier=oubliette

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable oubliette
sudo systemctl start oubliette
sudo systemctl status oubliette
```

#### macOS (launchd)

```bash
sudo nano /Library/LaunchDaemons/ai.factory.oubliette.plist
```

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>ai.factory.oubliette</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/oubliette-server</string>
        <string>-config</string>
        <string>/etc/oubliette/config.yaml</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/opt/oubliette/logs/stdout.log</string>
    <key>StandardErrorPath</key>
    <string>/opt/oubliette/logs/stderr.log</string>
    <key>WorkingDirectory</key>
    <string>/opt/oubliette</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/usr/local/bin:/usr/bin:/bin:/opt/homebrew/bin</string>
    </dict>
</dict>
</plist>
```

```bash
sudo launchctl load /Library/LaunchDaemons/ai.factory.oubliette.plist
sudo launchctl list | grep oubliette
```

### 5. Create Admin Token

```bash
oubliette token create --name production-admin --scope admin
```

---

## Configuration

### Environment Variables vs Config File

**Precedence** (highest to lowest):
1. Environment variables
2. Config file (`-config` flag)
3. Default values

**Use environment variables for**: Secrets, deployment-specific overrides  
**Use config file for**: Structured configuration, resource limits

### Key Settings

| Setting | Environment | Config | Default |
|---------|------------|--------|---------|
| Listen address | `SERVER_ADDR` | `server.address` | `:8080` |
| Factory API key | `FACTORY_API_KEY` | `auth.factory_api_key` | (required) |
| GitHub token | `DEFAULT_GITHUB_TOKEN` | `auth.default_github_token` | (optional) |
| Container runtime | `CONTAINER_RUNTIME` | `runtime.preference` | `auto` |
| Max recursion depth | `DEFAULT_MAX_RECURSION_DEPTH` | `recursion.max_depth` | `3` |
| Max agents/session | `DEFAULT_MAX_AGENTS_PER_SESSION` | `recursion.max_agents` | `50` |

---

## TLS/HTTPS Setup

**Production deployments MUST use TLS** to protect Bearer tokens.

### Recommended: Reverse Proxy

Use Caddy or nginx for TLS termination.

**Caddy** (automatic HTTPS):

```caddy
oubliette.yourdomain.com {
    reverse_proxy localhost:8080
}
```

**nginx** (with Let's Encrypt):

```nginx
server {
    listen 443 ssl http2;
    server_name oubliette.yourdomain.com;
    
    ssl_certificate /etc/letsencrypt/live/oubliette.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/oubliette.yourdomain.com/privkey.pem;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_buffering off;
    }
}
```

---

## Monitoring Setup

### Prometheus Metrics

Oubliette exports metrics at `/metrics` (no auth required).

**Key metrics:**
- `oubliette_requests_total` - HTTP requests by method/path/status
- `oubliette_request_duration_seconds` - Request latency
- `oubliette_active_sessions` - Active sessions per project
- `oubliette_containers_running` - Running container count
- `oubliette_event_buffer_drops_total` - Dropped events

**Prometheus config:**

```yaml
scrape_configs:
  - job_name: 'oubliette'
    static_configs:
      - targets: ['localhost:8080']
```

---

## Backup and Recovery

### Automatic Backups

Enabled by default. Configure in `config.yaml`:

```yaml
backup:
  enabled: true
  interval_hours: 24
  retention: 7
  directory: "/opt/oubliette/backups"
```

### Manual Backup

```bash
tar czf backup-$(date +%Y%m%d).tar.gz -C /opt/oubliette projects data
```

### Restore

```bash
sudo systemctl stop oubliette
tar xzf backup-20251230.tar.gz -C /opt/oubliette/
sudo systemctl start oubliette
```

---

## Health Checks

| Endpoint | Purpose | Auth |
|----------|---------|------|
| `/health` | Liveness (process running?) | No |
| `/ready` | Readiness (can serve?) | No |
| `/metrics` | Prometheus metrics | No |

```bash
curl http://localhost:8080/health  # {"status":"ok"}
curl http://localhost:8080/ready   # {"status":"ready"}
```

---

## Troubleshooting

### Server Won't Start

```bash
# Check logs
sudo journalctl -u oubliette -n 100

# Common causes:
# - Missing FACTORY_API_KEY
# - Port 8080 in use
# - Docker socket permission denied
```

### Container Creation Fails

```bash
# Check Docker/Apple Container
docker ps           # Docker
container list      # Apple Container

# Common causes:
# - Runtime not running
# - Insufficient disk space
# - Image not built
```

### High Memory Usage

```bash
# Reduce concurrency and enable cleanup
# In config.yaml:
recursion:
  max_agents: 25
cleanup:
  enabled: true
  retention_minutes: 30
```

---

## Security Notes

- **Run server on host** (not in container) - manages project containers
- **Use TLS** for all external traffic
- **Restrict network access** to port 8080
- **Create separate tokens** for different clients
- **Rotate tokens** regularly

---

## Resources

- [Operations Guide](OPERATIONS.md)
- [Security Policy](SECURITY.md)
- [Design Patterns](PATTERNS.md)
- [Production Readiness](PRODUCTION_READINESS_PLAN_v2.md)
