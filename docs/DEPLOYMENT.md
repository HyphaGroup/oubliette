# Deployment Guide

## Prerequisites

- **Container runtime**: Docker or Apple Container (for agent workloads)
- **Provider API key**: Anthropic, OpenAI, or other supported provider
- **GitHub token** (optional): For repository cloning

## Quick Start

```bash
curl -fsSL https://raw.githubusercontent.com/HyphaGroup/oubliette/main/install.sh | bash
oubliette init
# Edit ~/.oubliette/config/oubliette.jsonc with API keys
oubliette mcp --setup claude
oubliette --daemon
```

## From Source

```bash
git clone https://github.com/HyphaGroup/oubliette.git && cd oubliette
cp config/oubliette.jsonc.example config/oubliette.jsonc
# Edit config with your API keys
./build.sh
./bin/oubliette
```

## Production Deployment

### Build and Install

```bash
./build.sh
sudo cp bin/oubliette /usr/local/bin/
```

### Systemd Service (Linux)

```ini
[Unit]
Description=Oubliette MCP Server
After=network.target docker.service

[Service]
Type=simple
User=oubliette
WorkingDirectory=/opt/oubliette
ExecStart=/usr/local/bin/oubliette --dir /opt/oubliette
Restart=on-failure
RestartSec=5s
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
```

### launchd (macOS)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>oubliette</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/oubliette</string>
        <string>--dir</string>
        <string>/opt/oubliette</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/opt/oubliette/data/logs/stdout.log</string>
    <key>StandardErrorPath</key>
    <string>/opt/oubliette/data/logs/stderr.log</string>
</dict>
</plist>
```

### Create Admin Token

```bash
oubliette token create --name production-admin --scope admin
```

## TLS/HTTPS

Use a reverse proxy for TLS termination:

**Caddy** (automatic HTTPS):
```caddy
oubliette.yourdomain.com {
    reverse_proxy localhost:8080
}
```

**nginx**:
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
        proxy_buffering off;
    }
}
```

## Health Checks

| Endpoint | Purpose | Auth |
|----------|---------|------|
| `/health` | Liveness | No |
| `/ready` | Readiness | No |
| `/metrics` | Prometheus | No |

## Security Notes

- Use TLS for all external traffic
- Create separate tokens per client
- Rotate tokens regularly
- Server runs on host (manages containers), not inside a container

## Resources

- [OPERATIONS.md](OPERATIONS.md) - Runbook and troubleshooting
- [CONFIGURATION.md](CONFIGURATION.md) - Config reference
- [BACKUPS.md](BACKUPS.md) - Backup and restore
