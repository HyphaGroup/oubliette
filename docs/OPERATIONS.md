# Oubliette Operations Runbook

This document provides operational guidance for running Oubliette in production.

## Quick Reference

### Health Checks
```bash
# Liveness (is the process running?)
curl http://localhost:8080/health

# Readiness (can it serve requests?)
curl http://localhost:8080/ready

# Metrics (Prometheus format)
curl http://localhost:8080/metrics
```

### Common Commands
```bash
# Start server
./oubliette

# Create admin token
./oubliette token create --name ops-admin --scope admin

# List tokens
./oubliette token list

# Revoke token
./oubliette token revoke <token-id>
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `FACTORY_API_KEY` | Factory API key (required) | - |
| `DEFAULT_GITHUB_TOKEN` | Default GitHub token for projects | - |
| `SERVER_ADDR` | Server listen address | `:8080` |
| `CONTAINER_RUNTIME` | Runtime preference: auto, docker, apple-container | `auto` |
| `DEFAULT_MAX_RECURSION_DEPTH` | Max agent recursion depth | `3` |
| `DEFAULT_MAX_AGENTS_PER_SESSION` | Max agents per session | `50` |
| `DEFAULT_MAX_COST_USD` | Max cost per session | `10.00` |

### Config File (config.yaml)

```yaml
server:
  address: ":8080"

auth:
  factory_api_key: "your-key"
  default_github_token: "ghp_xxx"

runtime:
  preference: "auto"
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
  disk_warn_percent: 80
  disk_error_percent: 90

backup:
  enabled: true
  interval_hours: 24
  retention: 7
  directory: "backups"

paths:
  projects: "projects"
  data: "data"
  logs: "logs"
  sockets: "/tmp/oubliette-sockets"
```

## Monitoring

### Key Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `oubliette_requests_total` | Counter | Total HTTP requests |
| `oubliette_request_duration_seconds` | Histogram | Request latency |
| `oubliette_active_sessions` | Gauge | Currently active sessions |
| `oubliette_session_events_total` | Counter | Total session events |

### Alert Recommendations

```yaml
# Prometheus alert rules
groups:
- name: oubliette
  rules:
  - alert: HighErrorRate
    expr: rate(oubliette_requests_total{status=~"5.."}[5m]) > 0.1
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: High error rate on Oubliette

  - alert: HighLatency
    expr: histogram_quantile(0.95, rate(oubliette_request_duration_seconds_bucket[5m])) > 5
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: High latency on Oubliette

  - alert: TooManySessions
    expr: oubliette_active_sessions > 100
    for: 10m
    labels:
      severity: warning
    annotations:
      summary: High number of active sessions
```

## Troubleshooting

### Server Won't Start

**Symptom**: Server exits immediately on startup

**Checks**:
1. Verify `FACTORY_API_KEY` is set
2. Check Dockerfile exists in working directory
3. Verify container runtime is available:
   ```bash
   docker info  # or container-runtime info
   ```
4. Check logs for specific error

### Sessions Failing

**Symptom**: Sessions fail to spawn or complete

**Checks**:
1. Verify container is running:
   ```bash
   docker ps | grep oubliette
   ```
2. Check container logs:
   ```bash
   docker logs oubliette-<project-id>
   ```
3. Verify disk space:
   ```bash
   df -h /path/to/projects
   ```
4. Check recursion limits haven't been exceeded

### High Memory Usage

**Symptom**: Server memory grows over time

**Checks**:
1. Check active sessions count:
   ```bash
   curl http://localhost:8080/metrics | grep active_sessions
   ```
2. Verify cleanup is running (check logs for "🧹 Cleanup")
3. Force cleanup by restarting server gracefully

### Container Startup Failures

**Symptom**: Containers fail to start for projects

**Checks**:
1. Verify base image exists:
   ```bash
   docker images | grep oubliette
   ```
2. Rebuild image:
   ```bash
   # Via API
   curl -X POST "http://localhost:8080/mcp" \
     -H "Authorization: Bearer $TOKEN" \
     -d '{"method":"tools/call","params":{"name":"image_rebuild","arguments":{"project_id":"xxx"}}}'
   ```
3. Check Docker daemon status

### Auth Token Issues

**Symptom**: 401/403 responses

**Checks**:
1. Verify token exists:
   ```bash
   ./oubliette token list
   ```
2. Check token scope matches required operation
3. Verify token hasn't been revoked
4. Check rate limiting (429 response)

### Data Corruption

**Symptom**: Project or session data unreadable

**Recovery**:
1. Stop server gracefully (SIGTERM)
2. Check backup availability:
   ```bash
   ls backups/
   ```
3. Restore from backup (if available):
   ```bash
   # Manual restore
   tar -xzf backups/<project>_<date>.tar.gz -C projects/
   ```
4. If no backup, delete corrupt project directory and recreate

## Maintenance Procedures

### Backup Verification

```bash
# List recent backups
ls -la backups/

# Verify backup integrity
tar -tzf backups/<project>_<date>.tar.gz

# Test restore to temp location
mkdir /tmp/restore-test
tar -xzf backups/<project>_<date>.tar.gz -C /tmp/restore-test
ls /tmp/restore-test/<project>/
```

### Log Rotation

Logs are written to `logs/oubliette.log`. Configure logrotate:

```
/path/to/oubliette/logs/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    copytruncate
}
```

### Database Maintenance

Auth tokens are stored in SQLite (`data/auth.db`). Periodic maintenance:

```bash
# Backup auth database
cp data/auth.db data/auth.db.backup

# Vacuum database (reclaim space)
sqlite3 data/auth.db "VACUUM;"
```

### Graceful Shutdown

```bash
# Find server PID
pgrep -f oubliette-server

# Send SIGTERM for graceful shutdown
kill -TERM <pid>

# Verify shutdown complete (check logs)
tail -f logs/oubliette.log
```

### Version Upgrade

1. Stop server gracefully
2. Backup data directory
3. Replace binary
4. Start server
5. Verify health checks pass
6. Monitor for errors

```bash
# Step by step
kill -TERM $(pgrep -f oubliette-server)
cp -r data data.backup.$(date +%Y%m%d)
cp oubliette-server-new oubliette-server
./oubliette-server -config config.yaml &
sleep 5
curl http://localhost:8080/health
```

## Capacity Planning

### Resource Requirements

| Load | CPU | Memory | Disk |
|------|-----|--------|------|
| Light (< 10 sessions) | 2 cores | 4 GB | 10 GB |
| Medium (10-50 sessions) | 4 cores | 8 GB | 50 GB |
| Heavy (> 50 sessions) | 8+ cores | 16+ GB | 100+ GB |

### Scaling Considerations

- Horizontal scaling: Run multiple instances with shared storage
- Vertical scaling: Increase container resources
- Rate limiting: Adjust `rate_limit` config for higher throughput
- Session limits: Adjust `max_agents` and `max_depth` for resource control

## Emergency Procedures

### Server Crash Recovery

1. Check system resources (disk, memory)
2. Review crash logs
3. Start server with minimal config
4. Verify data integrity
5. Restore from backup if needed
6. Investigate root cause

### Security Incident

1. Revoke all tokens immediately:
   ```bash
   ./oubliette token list | grep -v ID | awk '{print $1}' | xargs -I{} ./oubliette token revoke {}
   ```
2. Stop server
3. Preserve logs for investigation
4. Review audit logs
5. Rotate `FACTORY_API_KEY`
6. Restart with new tokens

## Support

- GitHub Issues: Report bugs and feature requests
- Logs: Include relevant log excerpts
- Metrics: Include relevant metric values
- Configuration: Include sanitized config (no secrets)
