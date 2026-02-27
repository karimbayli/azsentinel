# Sentinel V2 — Security

## Authentication & Authorization

### Node → Central Communication
- All probe data submissions are signed with **HMAC-SHA256**
- The shared secret is configured identically on central and all nodes
- Invalid signatures result in HTTP 403 Forbidden
- The signature is computed over the full request body

### API Access
- Public endpoints: `/healthz`, `/api/v1/methodology`, `/api/v1/status`, static pages
- Authenticated endpoints: `/api/v1/ingest/probe-batch` (HMAC signature required)
- No user authentication system — the API is read-only for public consumption

## Data Security

### Transport
- All external communication uses TLS (Caddy provides automatic Let's Encrypt certificates)
- Node-to-central HTTPS with HMAC provides both encryption and authentication

### Storage
- TimescaleDB uses standard PostgreSQL authentication
- No PII is stored — social monitoring aggregates metrics only, no message content or user data
- SQLite buffer files on probe nodes contain probe results (no secrets)

### Secret Management
- Secrets are loaded from environment variables or config files
- `.env` file is gitignored by default
- Config files support `${ENV_VAR}` expansion

## Network Security

### Probe Nodes
- Minimal attack surface (single binary, no listening ports)
- Run as dedicated `sentinel` user, not root
- systemd security hardening: `NoNewPrivileges`, `ProtectSystem`, `PrivateTmp`

### Central Server
- Caddy provides automatic TLS with security headers
- X-Content-Type-Options, X-Frame-Options, HSTS enabled
- Database is only accessible within the Docker network

## Threat Model

| Threat | Mitigation |
|--------|-----------|
| Forged probe data | HMAC-SHA256 signature validation |
| Man-in-the-middle | TLS encryption on all channels |
| DDoS on central | Caddy rate limiting, cloud provider firewall |
| Database compromise | Network isolation, strong passwords, regular backups |
| Probe node compromise | Minimal privileges, buffer-only local storage, no outbound secrets |
| Data tampering | TimescaleDB hypertables are append-only by design |

## Recommendations

1. Rotate HMAC secrets quarterly
2. Use strong, unique passwords for database and Grafana
3. Enable firewall rules to restrict database access
4. Regularly update Docker images and dependencies
5. Monitor probe agent logs for authentication failures
6. Restrict SSH access on all servers to key-based auth only
