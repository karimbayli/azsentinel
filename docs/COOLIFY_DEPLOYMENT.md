# Deploying Sentinel V2 on Coolify (v4.0.0-beta.463)

This guide provides step-by-step instructions for deploying the **Sentinel V2 Central Server** stack using Coolify v4 with a strict focus on **maximum security, privacy, and isolation**.

## 1. Prerequisites

- A fresh Ubuntu 22.04+ / Debian server with [Coolify v4](https://coolify.io) installed.
- Domain names resolving to your server (e.g., `updown.az` and `grafana.updown.az`).
- You have cloned the repository or have it available via Git integration in Coolify.

---

## 2. Infrastructure Architecture & Security Posture

To ensure maximum security:
1. **Private Network:** The database (`timescaledb`) and `grafana` will *not* be exposed to the public internet directly. They communicate exclusively over the internal Docker bridge network managed by Coolify.
2. **Reverse Proxy:** Coolify's Traefik / Caddy proxy handles SSL termination and routes traffic securely to the `central` container.
3. **Secrets Management:** All sensitive variables (HMAC secrets, DB passwords, Telegram tokens) are injected via Coolify's environment variables tab and never hardcoded.

---

## 3. Create the Services via Docker Compose

In your Coolify Dashboard:
1. Navigate to **Projects** -> **[Your Project]** -> **[Your Environment]**.
2. Click **+ New Resource** and select **Docker Compose**.
3. Provide a name (e.g., `sentinel-stack`).
4. Paste the following hardened `docker-compose.yml` into the editor:

```yaml
version: '3.8'

services:
  timescaledb:
    image: timescale/timescaledb:2.17.2-pg16
    container_name: sentinel-db
    restart: unless-stopped
    environment:
      POSTGRES_DB: ${DB_NAME}
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - timescaledb_data:/var/lib/postgresql/data
    healthcheck:
      test: [ "CMD-SHELL", "pg_isready -U ${DB_USER}" ]
      interval: 10s
      timeout: 5s
      retries: 5
    # No ports exposed! Accessible only internally.

  central:
    build:
      context: .
      dockerfile: Dockerfile.central
    container_name: sentinel-central
    restart: unless-stopped
    depends_on:
      timescaledb:
        condition: service_healthy
    environment:
      SENTINEL_DB_HOST: timescaledb
      SENTINEL_DB_PORT: "5432"
      SENTINEL_DB_USER: ${DB_USER}
      SENTINEL_DB_PASSWORD: ${DB_PASSWORD}
      SENTINEL_DB_NAME: ${DB_NAME}
      SENTINEL_HMAC_SECRET: ${HMAC_SECRET}
      SENTINEL_CALIBRATION: "false" # Set to true initially if desired
      SENTINEL_TELEGRAM_BOT_TOKEN: ${TELEGRAM_BOT_TOKEN}
      SENTINEL_TELEGRAM_CHAT_ID: ${TELEGRAM_CHAT_ID}
    # Handled by Coolify proxy, no port mapping needed
    healthcheck:
      test: [ "CMD", "wget", "--spider", "-q", "http://localhost:8080/healthz" ]
      interval: 30s
      timeout: 5s
      retries: 3

  grafana:
    image: grafana/grafana:10.4.1
    container_name: sentinel-grafana
    restart: unless-stopped
    depends_on:
      timescaledb:
        condition: service_healthy
    environment:
      GF_SECURITY_ADMIN_USER: ${GRAFANA_USER}
      GF_SECURITY_ADMIN_PASSWORD: ${GRAFANA_PASSWORD}
      GF_USERS_ALLOW_SIGN_UP: "false"
      GF_SERVER_ROOT_URL: https://${GRAFANA_DOMAIN}/
      GF_SECURITY_DISABLE_GRAVATAR: "true" # Privacy tweak
      GF_ANALYTICS_REPORTING_ENABLED: "false" # Privacy tweak
    volumes:
      - grafana_data:/var/lib/grafana

volumes:
  timescaledb_data:
  grafana_data:
```

## 4. Configuring Secrets & Environment Variables

Go to the **Environment Variables** tab for your newly created stack and add the following securely:

| Variable | Description | Example |
|---|---|---|
| `DB_NAME` | Name of the PostgreSQL database | `sentinel` |
| `DB_USER` | Database user | `sentinel_user` |
| `DB_PASSWORD` | Strong database password | `[Generate a secure 32-char string]` |
| `HMAC_SECRET` | Secret used to sign probe reports | `[Generate using: openssl rand -hex 32]` |
| `TELEGRAM_BOT_TOKEN`| Bot token for alerts | `123456789:ABCdefGHIjkl...` |
| `TELEGRAM_CHAT_ID` | Chat ID for alerts | `-100123456789` |
| `GRAFANA_USER` | Grafana admin username | `admin` |
| `GRAFANA_PASSWORD`| Grafana admin password | `[Generate a secure string]` |
| `GRAFANA_DOMAIN` | The public domain for Grafana | `grafana.updown.az` |

## 5. Domain & Proxy Configuration

In Coolify v4, services are automatically proxied. We must map the public domains to our containers.

1. Navigate to the **Services** section within your Stack in Coolify.
2. Select the `central` container.
   - Set the **Domains** field to: `https://updown.az`
   - Ensure the internal port is set to `8080` (the default for the Go backend).
3. Select the `grafana` container.
   - Set the **Domains** field to: `https://grafana.updown.az`
   - Ensure the internal port is set to `3000`.

**Coolify automatically generates and provisions Let's Encrypt TLS certificates** ensuring end-to-end encryption for the API and frontend.

## 6. Deployment

Click **Deploy** in the top right corner. Coolify will:
1. Pull the official TimescaleDB and Grafana images.
2. Execute the multi-stage build defined in `Dockerfile.central` (which safely compiles the React frontend, then builds the Go binary).
3. Provision the internal networking.
4. Launch the containers and attach the reverse proxy with TLS.

## 7. Post-Deployment Privacy Verification

1. **Verify Isolation:** Run `nmap` or a port scanner against your server IP to ensure ports `5432` (PostgreSQL) and `3000` (Grafana) are *not* publicly exposed on the host. They should only be accessible through Coolify's Traefik edge proxy over HTTPS.
2. **Verify Telemetry:** Our Grafana config disables public avatars and analytics (`GF_SECURITY_DISABLE_GRAVATAR` and `GF_ANALYTICS_REPORTING_ENABLED`).
3. **Verify API Access:** Ensure `/api/v1/ingest/probe-batch` correctly rejects POST requests lacking a valid `X-Sentinel-Signature` header matching the `HMAC_SECRET`.
