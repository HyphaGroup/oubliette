# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability within Oubliette, please send an email to security@resetnetwork.org. All security vulnerabilities will be promptly addressed.

Please include the following information:
- Type of vulnerability
- Full paths of source file(s) related to the vulnerability
- Steps to reproduce the issue
- Potential impact
- Any suggested mitigations

**Do not open public issues for security vulnerabilities.**

## Security Considerations

### Authentication
- All MCP API endpoints require Bearer token authentication
- Tokens are stored with bcrypt hashing
- Health endpoints (`/health`, `/ready`) are intentionally unauthenticated for load balancer probes

### Container Isolation
- Each project runs in its own container
- Containers use non-root users by default
- Workspace isolation prevents cross-project access

### Data Protection
- GitHub tokens are stored in project-specific `.env` files (not in metadata.json)
- Session data uses atomic writes to prevent corruption
- File locking prevents concurrent metadata corruption

### Input Validation
- All UUIDs are validated before use
- Path traversal attacks are blocked
- Session IDs follow strict format validation

### Rate Limiting
- Rate limiting is available but must be configured per deployment
- Consider adding rate limits in production

## Security Checklist for Deployment

- [ ] Use HTTPS in production (terminate TLS at load balancer)
- [ ] Rotate API keys regularly
- [ ] Enable audit logging
- [ ] Set up monitoring and alerting
- [ ] Review container security settings
- [ ] Keep dependencies updated
