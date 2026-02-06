# Operations Runbook

## Health Checks

```bash
curl http://localhost:8080/health    # Liveness
curl http://localhost:8080/ready     # Readiness
curl http://localhost:8080/metrics   # Prometheus metrics
```

## Common Commands

```bash
oubliette                              # Start server
oubliette --daemon                     # Start in background
oubliette token create --name ops --scope admin
oubliette token list
oubliette token revoke <token-id>
```

## Configuration

All config in `oubliette.jsonc`. See [CONFIGURATION.md](CONFIGURATION.md).

## Monitoring

### Key Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `oubliette_requests_total` | Counter | HTTP requests by method/path/status |
| `oubliette_request_duration_seconds` | Histogram | Request latency |
| `oubliette_active_sessions` | Gauge | Active sessions per project |

### Prometheus Config

```yaml
scrape_configs:
  - job_name: 'oubliette'
    static_configs:
      - targets: ['localhost:8080']
```

## Troubleshooting

### Server Won't Start

1. Check provider API keys in `oubliette.jsonc`
2. Verify container runtime: `docker ps` or `container list`
3. Check port 8080 availability
4. Review logs in `data/logs/`

### Sessions Failing

1. Verify container is running: `docker ps | grep oubliette`
2. Check container logs: `docker logs oubliette-<project-id>`
3. Verify disk space
4. Check recursion limits

### Auth Token Issues

1. Verify token exists: `oubliette token list`
2. Check scope matches required operation
3. Restart server after `mcp --setup` to pick up new tokens

### High Memory

Reduce session limits or enable more aggressive cleanup:
- `defaults.limits.max_agents_per_session` in config
- Cleanup runs automatically (5m interval, 1h retention)

## Database Maintenance

Auth and schedule data in SQLite (WAL mode):

```bash
cp data/auth.db data/auth.db.backup
sqlite3 data/auth.db "VACUUM;"
```

## Graceful Shutdown

```bash
kill -TERM $(pgrep -f oubliette)
```

Closes active sessions, stops cleanup/backup, closes databases.

## Version Upgrade

```bash
kill -TERM $(pgrep -f oubliette)
cp -r data data.backup.$(date +%Y%m%d)
# Replace binary
oubliette --daemon
curl http://localhost:8080/health
```
