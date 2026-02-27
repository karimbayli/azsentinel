# UpDown.az — Deployment Guide

> Complete step-by-step guide for deploying the UpDown.az internet monitoring platform.
> Infrastructure: Hetzner (Coolify) + AZ VPS + Oracle Cloud Free.

---

## Table of Contents

1. [Prerequisites](#1-prerequisites)
2. [Generate Secrets](#2-generate-secrets)
3. [Push to GitHub & Build Images](#3-push-to-github--build-images)
4. [DNS Configuration](#4-dns-configuration)
5. [Deploy Central Stack on Hetzner](#5-deploy-central-stack-on-hetzner)
6. [Verify Central is Running](#6-verify-central-is-running)
7. [Deploy EU Probe (Hetzner co-located)](#7-deploy-eu-probe-hetzner-co-located)
8. [Deploy AZ Probe (Azerbaijan VPS)](#8-deploy-az-probe-azerbaijan-vps)
9. [Create Oracle Cloud Account](#9-create-oracle-cloud-account)
10. [Deploy US Probe (Oracle Cloud)](#10-deploy-us-probe-oracle-cloud)
11. [Deploy Asia Probe (Oracle Cloud)](#11-deploy-asia-probe-oracle-cloud)
12. [Verify All Probes Reporting](#12-verify-all-probes-reporting)
13. [Calibration Period](#13-calibration-period)
14. [Go Live](#14-go-live)
15. [Post-Launch Maintenance](#15-post-launch-maintenance)
16. [Troubleshooting](#16-troubleshooting)

---

## 1. Prerequisites

Before starting, ensure you have:

- [ ] **Hetzner server** with Coolify installed and accessible
- [ ] **AZ VPS** (1 CPU, 2 GB RAM, 30 GB SSD) with SSH access
- [ ] **GitHub account** with the `azsentinel` repository pushed
- [ ] **Domain**: `updown.az` with access to DNS management panel
- [ ] **SSH key** pair for all servers
- [ ] **Local machine**: Git, Docker (for testing), Go 1.24+ (optional)

### Software versions
```
Docker Engine:  24.x or later
Docker Compose: v2.x (bundled with Docker)
Go:             1.24+ (only needed if building locally)
PostgreSQL:     16 (via TimescaleDB container)
```

---

## 2. Generate Secrets

On your local machine, generate the shared secrets. **Save these — you'll need them on every server.**

### 2.1 Generate HMAC secret
```bash
# This secret authenticates probe→central communication
# MUST be identical on central and ALL probe nodes
openssl rand -hex 32
```
**Save the output as `HMAC_SECRET`.** Example: `a1b2c3d4e5f6...` (64 characters)

### 2.2 Generate database password
```bash
openssl rand -hex 16
```
**Save the output as `DB_PASSWORD`.**

### 2.3 Generate Grafana password
```bash
openssl rand -base64 16
```
**Save the output as `GRAFANA_PASSWORD`.**

### 2.4 Write down all secrets

Create a secure note (password manager recommended) with:
```
HMAC_SECRET=<your-64-char-hex>
DB_PASSWORD=<your-32-char-hex>
GRAFANA_USER=admin
GRAFANA_PASSWORD=<your-base64>
```

> ⚠️ **IMPORTANT**: The `HMAC_SECRET` must be the EXACT SAME value on the central server and every probe node. If they don't match, probes will get 401 Unauthorized errors.

---

## 3. Push to GitHub & Build Images

### 3.1 Ensure repository is up to date
```bash
cd ~/GolandProjects/azsentinel
git add -A
git commit -m "chore: deployment preparation for updown.az"
git push origin main
```

### 3.2 Enable GitHub Container Registry

1. Go to your GitHub repo → **Settings** → **Actions** → **General**
2. Under "Workflow permissions", select **"Read and write permissions"**
3. Click **Save**

### 3.3 Verify CI/CD pipeline runs

1. Go to **Actions** tab in your GitHub repo
2. You should see the "Build & Push Docker Images" workflow running
3. Wait for it to complete (usually 3–5 minutes)
4. Check that images are published:
   - `ghcr.io/<your-username>/azsentinel/central:latest`
   - `ghcr.io/<your-username>/azsentinel/probe:latest`

### 3.4 Make packages public (optional but recommended)

1. Go to your GitHub profile → **Packages**
2. Click each package → **Package settings** → **Change visibility** → **Public**

If you keep packages private, you'll need to configure Docker login on each probe server:
```bash
echo "<GITHUB_TOKEN>" | docker login ghcr.io -u <USERNAME> --password-stdin
```

---

## 4. DNS Configuration

### 4.1 Log in to your AZ domain registrar

Go to your registrar's DNS management panel for `updown.az`.

### 4.2 Add A record

| Type | Name | Value | TTL |
|------|------|-------|-----|
| A | `@` (or `updown.az`) | `<YOUR_HETZNER_SERVER_IP>` | 300 |

### 4.3 Add www redirect (optional)

| Type | Name | Value | TTL |
|------|------|-------|-----|
| CNAME | `www` | `updown.az` | 300 |

### 4.4 Verify DNS propagation

Wait 5–15 minutes, then check:
```bash
dig +short updown.az
# Should return your Hetzner server IP
```

Or use: https://dnschecker.org/#A/updown.az

---

## 5. Deploy Central Stack on Hetzner

### 5.1 SSH into Hetzner server
```bash
ssh root@<HETZNER_IP>
```

### 5.2 Create project directory
```bash
mkdir -p /opt/sentinel && cd /opt/sentinel
```

### 5.3 Clone the repository
```bash
git clone https://github.com/<YOUR_USERNAME>/azsentinel.git .
```

### 5.4 Create production .env

```bash
cp .env.example .env
nano .env
```

Fill in the actual values:
```env
DOMAIN=updown.az

DB_NAME=sentinel
DB_USER=sentinel
DB_PASSWORD=<your-DB_PASSWORD-from-step-2>

HMAC_SECRET=<your-HMAC_SECRET-from-step-2>

GRAFANA_USER=admin
GRAFANA_PASSWORD=<your-GRAFANA_PASSWORD-from-step-2>

TELEGRAM_BOT_TOKEN=
TELEGRAM_CHAT_ID=0

CALIBRATION=true
```

### 5.5 Create central.yaml config

```bash
cp configs/central.example.yaml configs/central.yaml
```

The YAML config's database section uses environment variables (`${SENTINEL_DB_HOST}` etc.), so no manual editing is needed — Docker Compose passes them in.

### 5.6 Start the central stack

```bash
cd /opt/sentinel
docker compose -f deployments/docker-compose.central.yml up -d
```

### 5.7 Watch startup logs

```bash
# Check all services are starting
docker compose -f deployments/docker-compose.central.yml ps

# Watch central app logs
docker compose -f deployments/docker-compose.central.yml logs -f central

# You should see:
# "connected to TimescaleDB"
# "synced config to database"
# "bgp monitor started"
# "server listening on :8080"
```

### 5.8 Wait for TimescaleDB to initialize

First startup takes ~30 seconds as TimescaleDB runs the migration scripts. Watch:
```bash
docker compose -f deployments/docker-compose.central.yml logs -f timescaledb
```
Wait until you see: `database system is ready to accept connections`

### 5.9 Run the seed migration (first time only)

The 001/002 migrations run automatically. Run the target seed migration:
```bash
docker compose -f deployments/docker-compose.central.yml exec timescaledb \
    psql -U sentinel -d sentinel -f /docker-entrypoint-initdb.d/003_seed_targets.sql
```

---

## 6. Verify Central is Running

### 6.1 Health check
```bash
curl -s http://localhost:8080/healthz
# Expected: {"status":"ok"}
```

### 6.2 Check API returns data
```bash
curl -s http://localhost:8080/api/v1/status | head -c 200
# Should return JSON array of target statuses
```

### 6.3 Check HTTPS (after DNS propagates)
```bash
curl -s https://updown.az/healthz
# Expected: {"status":"ok"}
```

### 6.4 Open in browser

Navigate to `https://updown.az` — you should see the UpDown.az dashboard with:
- UPDOWN.AZ header
- Dot-matrix world map
- Target cards grouped by category (will show "Waiting for data..." initially)

### 6.5 Check Grafana

Navigate to `https://updown.az/grafana/`
- Login: `admin` / `<GRAFANA_PASSWORD>`
- Verify TimescaleDB data source is connected

### 6.6 Check Caddy TLS

```bash
curl -sI https://updown.az | head -5
# Should show HTTP/2 200 and HSTS header
```

If you get a certificate error, check Caddy logs:
```bash
docker compose -f deployments/docker-compose.central.yml logs caddy
```

---

## 7. Deploy EU Probe (Hetzner co-located)

Since the Hetzner server runs both central and the EU probe, deploy the probe on the same machine.

### 7.1 Create probe directory
```bash
mkdir -p /opt/sentinel-probe-eu && cd /opt/sentinel-probe-eu
```

### 7.2 Create .env file
```bash
cat > .env << 'EOF'
NODE_ID=node-eu
REGION=eu-frankfurt
COUNTRY=DE
CENTRAL_URL=https://updown.az
HMAC_SECRET=<your-HMAC_SECRET-from-step-2>
EOF
```

### 7.3 Create docker-compose.yml
```bash
cat > docker-compose.yml << 'EOF'
services:
  probe:
    image: ghcr.io/<YOUR_USERNAME>/azsentinel/probe:latest
    container_name: sentinel-probe-eu
    restart: unless-stopped
    env_file: .env
    environment:
      SENTINEL_NODE_ID: ${NODE_ID}
      SENTINEL_REGION: ${REGION}
      SENTINEL_COUNTRY: ${COUNTRY}
      SENTINEL_CENTRAL_URL: ${CENTRAL_URL}
      SENTINEL_HMAC_SECRET: ${HMAC_SECRET}
    volumes:
      - probe_data:/var/lib/sentinel
    deploy:
      resources:
        limits:
          memory: 128M
          cpus: '0.25'
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"

volumes:
  probe_data:
EOF
```

### 7.4 Pull and start
```bash
docker compose pull
docker compose up -d
```

### 7.5 Verify probe is reporting
```bash
docker compose logs -f
# Should see:
# "probe agent running" with interval=60s, targets=125+
# "probe cycle complete" every 60 seconds
# "flush successful" — data sent to central
```

### 7.6 Confirm on central
```bash
curl -s https://updown.az/api/v1/nodes | python3 -m json.tool
# Should show node-eu with is_alive=true
```

---

## 8. Deploy AZ Probe (Azerbaijan VPS)

### 8.1 SSH into AZ VPS
```bash
ssh <user>@<AZ_VPS_IP>
```

### 8.2 Install Docker (if not installed)
```bash
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
# Log out and back in for group change to take effect
exit
ssh <user>@<AZ_VPS_IP>
```

### 8.3 Verify Docker works
```bash
docker --version
docker compose version
```

### 8.4 Create probe directory
```bash
mkdir -p ~/sentinel-probe && cd ~/sentinel-probe
```

### 8.5 Create .env file
```bash
cat > .env << 'EOF'
NODE_ID=node-az
REGION=az-baku
COUNTRY=AZ
CENTRAL_URL=https://updown.az
HMAC_SECRET=<your-HMAC_SECRET-from-step-2>
EOF
```

> ⚠️ **CRITICAL**: The `HMAC_SECRET` here MUST be identical to the one on the central server. Copy-paste it exactly.

### 8.6 Create docker-compose.yml
```bash
cat > docker-compose.yml << 'EOF'
services:
  probe:
    image: ghcr.io/<YOUR_USERNAME>/azsentinel/probe:latest
    container_name: sentinel-probe-az
    restart: unless-stopped
    env_file: .env
    environment:
      SENTINEL_NODE_ID: ${NODE_ID}
      SENTINEL_REGION: ${REGION}
      SENTINEL_COUNTRY: ${COUNTRY}
      SENTINEL_CENTRAL_URL: ${CENTRAL_URL}
      SENTINEL_HMAC_SECRET: ${HMAC_SECRET}
    volumes:
      - probe_data:/var/lib/sentinel
    deploy:
      resources:
        limits:
          memory: 128M
          cpus: '0.25'
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"

volumes:
  probe_data:
EOF
```

### 8.7 Login to GHCR (if packages are private)
```bash
echo "<GITHUB_TOKEN>" | docker login ghcr.io -u <USERNAME> --password-stdin
```

### 8.8 Pull and start
```bash
docker compose pull
docker compose up -d
```

### 8.9 Verify
```bash
docker compose logs -f
# Wait ~60 seconds for first probe cycle
# Should see: "probe cycle complete", "flush successful"
```

### 8.10 Check resource usage
```bash
docker stats sentinel-probe-az --no-stream
# Should show: ~30-60MB RAM, <5% CPU
```

---

## 9. Create Oracle Cloud Account

### 9.1 Sign up for Oracle Cloud Free Tier

1. Go to: https://cloud.oracle.com
2. Click **"Start for Free"**
3. Fill in your details (full name, email, country)

### 9.2 Choose Home Region

> ⚠️ **CRITICAL**: Home region CANNOT BE CHANGED later. Choose carefully.

**Select: US East (Ashburn)**
- Largest free-tier capacity (never runs out of Always Free shapes)
- Good transit path to Azerbaijan
- Covers US perspective for monitoring

### 9.3 Complete sign-up

- Add a credit card (required for verification, won't be charged for Always Free)
- Wait for account activation (usually 1–5 minutes)

### 9.4 Create SSH key pair (if you don't have one for Oracle)
```bash
ssh-keygen -t ed25519 -f ~/.ssh/oracle_sentinel -N ""
cat ~/.ssh/oracle_sentinel.pub
# Copy this public key — you'll paste it when creating VMs
```

---

## 10. Deploy US Probe (Oracle Cloud)

### 10.1 Create Always Free VM

1. Login to Oracle Cloud Console: https://cloud.oracle.com
2. Go to **Compute** → **Instances** → **Create Instance**

| Setting | Value |
|---------|-------|
| **Name** | `sentinel-probe-us` |
| **Compartment** | Your default compartment |
| **Availability Domain** | Any (AD-1 recommended) |
| **Image** | Ubuntu 22.04 (or Oracle Linux 9) |
| **Shape** | **VM.Standard.A1.Flex** (Always Free ARM) |
| **OCPUs** | 1 |
| **Memory** | 6 GB |
| **Boot Volume** | 50 GB (Always Free) |
| **Network** | Create new VCN + subnet (auto-generated) |
| **SSH Key** | Paste your `~/.ssh/oracle_sentinel.pub` |

3. Click **Create**
4. Wait ~2 minutes for the instance to start
5. Copy the **Public IP** from the instance details page

### 10.2 Configure Security List (Firewall)

1. Go to **Networking** → **Virtual Cloud Networks** → Your VCN
2. Click **Security Lists** → **Default Security List**
3. Verify these **Ingress Rules** exist:

| Source | Protocol | Dest Port | Description |
|--------|----------|-----------|-------------|
| `0.0.0.0/0` | TCP | 22 | SSH access |

4. Verify **Egress Rules**:

| Destination | Protocol | Description |
|-------------|----------|-------------|
| `0.0.0.0/0` | All | Allow all outbound |

> No extra inbound ports needed — probes only make outbound connections.

### 10.3 SSH into the VM
```bash
ssh -i ~/.ssh/oracle_sentinel ubuntu@<ORACLE_US_IP>
```

### 10.4 Open OS-level firewall (Oracle Linux only)
```bash
# Oracle Linux has iptables rules by default
sudo iptables -F
sudo netfilter-persistent save
```
For Ubuntu, skip this step.

### 10.5 Install Docker
```bash
sudo apt update && sudo apt upgrade -y
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
exit
ssh -i ~/.ssh/oracle_sentinel ubuntu@<ORACLE_US_IP>
```

### 10.6 Create probe directory
```bash
mkdir -p ~/sentinel-probe && cd ~/sentinel-probe
```

### 10.7 Create .env file
```bash
cat > .env << 'EOF'
NODE_ID=node-us
REGION=us-ashburn
COUNTRY=US
CENTRAL_URL=https://updown.az
HMAC_SECRET=<your-HMAC_SECRET-from-step-2>
EOF
```

### 10.8 Create docker-compose.yml
```bash
cat > docker-compose.yml << 'EOF'
services:
  probe:
    image: ghcr.io/<YOUR_USERNAME>/azsentinel/probe:latest
    container_name: sentinel-probe-us
    restart: unless-stopped
    env_file: .env
    environment:
      SENTINEL_NODE_ID: ${NODE_ID}
      SENTINEL_REGION: ${REGION}
      SENTINEL_COUNTRY: ${COUNTRY}
      SENTINEL_CENTRAL_URL: ${CENTRAL_URL}
      SENTINEL_HMAC_SECRET: ${HMAC_SECRET}
    volumes:
      - probe_data:/var/lib/sentinel
    deploy:
      resources:
        limits:
          memory: 128M
          cpus: '0.25'
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"

volumes:
  probe_data:
EOF
```

### 10.9 Login to GHCR (if private)
```bash
echo "<GITHUB_TOKEN>" | docker login ghcr.io -u <USERNAME> --password-stdin
```

### 10.10 Pull and start
```bash
docker compose pull
docker compose up -d
```

### 10.11 Verify
```bash
docker compose logs -f
# Should see probe running and flushing data to central
```

---

## 11. Deploy Asia Probe (Oracle Cloud)

### 11.1 Create second Always Free VM

Repeat Step 10.1 with:

| Setting | Value |
|---------|-------|
| **Name** | `sentinel-probe-asia` |
| **Shape** | VM.Standard.A1.Flex (1 OCPU, 6 GB) |

> Oracle allows up to 4 OCPUs / 24 GB across Always Free ARM. You've used 1+6 for US, so 3+18 remain.

### 11.2 SSH in and install Docker
```bash
ssh -i ~/.ssh/oracle_sentinel ubuntu@<ORACLE_ASIA_IP>
sudo apt update && sudo apt upgrade -y
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
exit
ssh -i ~/.ssh/oracle_sentinel ubuntu@<ORACLE_ASIA_IP>
```

### 11.3 Create probe with Asia config
```bash
mkdir -p ~/sentinel-probe && cd ~/sentinel-probe

cat > .env << 'EOF'
NODE_ID=node-asia
REGION=asia-ashburn
COUNTRY=US
CENTRAL_URL=https://updown.az
HMAC_SECRET=<your-HMAC_SECRET-from-step-2>
EOF
```

> Note: Both Oracle VMs are in Ashburn since that's your home region. The second VM still adds value as a redundant US vantage point with a different availability domain.

### 11.4 Create docker-compose.yml (same as step 10.8)

```bash
cat > docker-compose.yml << 'EOF'
services:
  probe:
    image: ghcr.io/<YOUR_USERNAME>/azsentinel/probe:latest
    container_name: sentinel-probe-asia
    restart: unless-stopped
    env_file: .env
    environment:
      SENTINEL_NODE_ID: ${NODE_ID}
      SENTINEL_REGION: ${REGION}
      SENTINEL_COUNTRY: ${COUNTRY}
      SENTINEL_CENTRAL_URL: ${CENTRAL_URL}
      SENTINEL_HMAC_SECRET: ${HMAC_SECRET}
    volumes:
      - probe_data:/var/lib/sentinel
    deploy:
      resources:
        limits:
          memory: 128M
          cpus: '0.25'
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"

volumes:
  probe_data:
EOF
```

### 11.5 Pull and start
```bash
docker compose pull
docker compose up -d
docker compose logs -f
```

---

## 12. Verify All Probes Reporting

### 12.1 Check nodes API
```bash
curl -s https://updown.az/api/v1/nodes | python3 -m json.tool
```

Expected — all nodes with `is_alive: true`:
```json
[
    { "node_id": "node-eu",   "region": "eu-frankfurt", "is_alive": true },
    { "node_id": "node-az",   "region": "az-baku",      "is_alive": true },
    { "node_id": "node-us",   "region": "us-ashburn",   "is_alive": true },
    { "node_id": "node-asia", "region": "asia-ashburn",  "is_alive": true }
]
```

### 12.2 Check status API
```bash
curl -s https://updown.az/api/v1/status | python3 -m json.tool | head -30
```

You should see target statuses with `confidence > 0` and `node_breakdown` showing results from multiple nodes.

### 12.3 Check the dashboard

Open `https://updown.az` in a browser. You should see:
- ✅ Dot-matrix map with 4 glowing probe nodes
- ✅ Target cards grouped by category (GOV, BANK, MEDIA, ISP, OTHER, GLOBAL)
- ✅ Each card showing "AZ: Reachable · Global: OK 3/3"
- ✅ Confidence percentages filling in
- ✅ Language toggle (AZ/EN) working

### 12.4 Check BGP feed
```bash
curl -s https://updown.az/api/v1/bgp/events?hours=1 | python3 -m json.tool | head -10
```

---

## 13. Calibration Period

### 13.1 What calibration does

With `CALIBRATION=true`, the system:
- Collects baseline latency data for every target × node combination
- Learns normal response times (avg, p95, p99)
- Builds social signal baselines
- Does **NOT** fire any alerts

### 13.2 How long to calibrate

- **Minimum**: 7 days (covers weekday/weekend patterns)
- **Recommended**: 14–30 days (captures monthly patterns)

### 13.3 Monitor during calibration

Check daily:
```bash
# Are all probes alive?
curl -s https://updown.az/api/v1/nodes | python3 -c \
    "import sys,json; [print(n['node_id'], n['is_alive']) for n in json.load(sys.stdin)]"

# How many probe results stored?
docker compose -f /opt/sentinel/deployments/docker-compose.central.yml exec timescaledb \
    psql -U sentinel -d sentinel -c "SELECT COUNT(*) FROM probe_results;"

# Check baseline data building up
docker compose -f /opt/sentinel/deployments/docker-compose.central.yml exec timescaledb \
    psql -U sentinel -d sentinel -c "SELECT target_url, node_id, avg_total_ms, sample_count FROM latency_baselines ORDER BY sample_count DESC LIMIT 10;"
```

---

## 14. Go Live

### 14.1 Disable calibration mode

SSH into Hetzner:
```bash
cd /opt/sentinel
nano .env
# Change: CALIBRATION=false
```

### 14.2 Restart central
```bash
docker compose -f deployments/docker-compose.central.yml restart central
```

### 14.3 Set up Telegram alerts (optional)

1. Message [@BotFather](https://t.me/BotFather) on Telegram → `/newbot`
2. Name it: `UpDown.az Alert Bot`
3. Copy the bot token
4. Create a Telegram channel/group for alerts
5. Add the bot to the channel as admin
6. Get the chat ID:
   ```bash
   # Send a message to the channel, then:
   curl -s https://api.telegram.org/bot<TOKEN>/getUpdates | python3 -m json.tool
   # Look for chat.id in the output
   ```
7. Update `.env`:
   ```env
   TELEGRAM_BOT_TOKEN=<your-bot-token>
   TELEGRAM_CHAT_ID=<your-chat-id>
   ```
8. Restart:
   ```bash
   docker compose -f deployments/docker-compose.central.yml restart central
   ```

### 14.4 Announce

Share `https://updown.az` — the platform is now live!

---

## 15. Post-Launch Maintenance

### 15.1 Update application

When you push code changes to `main`:
```bash
# On Hetzner (central):
cd /opt/sentinel
git pull
docker compose -f deployments/docker-compose.central.yml pull
docker compose -f deployments/docker-compose.central.yml up -d

# On each probe server:
cd ~/sentinel-probe
docker compose pull
docker compose up -d
```

### 15.2 Add/remove targets (no redeploy needed)

```bash
# SSH into Hetzner, open DB shell:
docker compose -f /opt/sentinel/deployments/docker-compose.central.yml exec timescaledb \
    psql -U sentinel -d sentinel

-- Add a new target:
INSERT INTO targets (url, category, criticality, enabled, display_name)
VALUES ('https://newsite.az', 'OTHER', 5, true, 'New Site');

-- Disable a target (soft delete):
UPDATE targets SET enabled = false WHERE url = 'https://oldsite.az';

-- Check all active targets:
SELECT category, COUNT(*) FROM targets WHERE enabled GROUP BY category ORDER BY category;
```

### 15.3 Database backup

```bash
# Manual backup
docker compose -f /opt/sentinel/deployments/docker-compose.central.yml exec timescaledb \
    pg_dump -U sentinel sentinel | gzip > ~/sentinel-backup-$(date +%Y%m%d).sql.gz

# Automated daily backup (add to crontab):
crontab -e
# Add this line:
0 3 * * * docker compose -f /opt/sentinel/deployments/docker-compose.central.yml exec -T timescaledb pg_dump -U sentinel sentinel | gzip > /opt/backups/sentinel-$(date +\%Y\%m\%d).sql.gz
```

### 15.4 Monitor disk usage

```bash
docker compose -f /opt/sentinel/deployments/docker-compose.central.yml exec timescaledb \
    psql -U sentinel -d sentinel -c \
    "SELECT hypertable_name, pg_size_pretty(hypertable_size(format('%I.%I', hypertable_schema, hypertable_name)::regclass)) FROM timescaledb_information.hypertables;"
```

TimescaleDB auto-manages retention: 90 days raw data, 1 year hourly aggregates, 5 years daily.

### 15.5 Migrate to netwatch.az

When ready to switch domains:

1. Buy `netwatch.az`, add DNS A record → Hetzner IP
2. On Hetzner, update `.env`: `DOMAIN=netwatch.az`
3. On ALL probe servers, update `.env`: `CENTRAL_URL=https://netwatch.az`
4. Restart everything:
   ```bash
   # Hetzner:
   docker compose -f deployments/docker-compose.central.yml up -d
   # Each probe server:
   docker compose up -d
   ```
5. (Optional) Set old `updown.az` to redirect to `netwatch.az`

---

## 16. Troubleshooting

### Probe shows "flush failed"
```bash
# Check central reachability from probe machine:
curl -s https://updown.az/healthz

# Verify HMAC secrets match:
# On probe:
grep HMAC .env
# On central:
grep HMAC /opt/sentinel/.env
# They MUST be identical
```

### Certificate errors on Caddy
```bash
docker compose -f deployments/docker-compose.central.yml logs caddy
# Common causes: DNS not propagated, port 80/443 blocked
```

### TimescaleDB won't start
```bash
df -h  # Check disk space
docker compose -f deployments/docker-compose.central.yml logs timescaledb
```

### Oracle Cloud "Out of capacity"
```
# Try a different Availability Domain (AD-2, AD-3)
# Or retry later — capacity frees up regularly
# Tip: early morning UTC has better availability
```

### No BGP events
```bash
docker compose -f deployments/docker-compose.central.yml logs central | grep bgp
# If stuck, restart: docker compose restart central
```

### Dashboard shows "Waiting for data..."
- Normal for first 60 seconds after probe starts
- Check that at least one probe is flushing: `docker compose logs -f` on any probe
- Check browser console (F12) for API errors

---

## Quick Reference

| Action | Command |
|--------|---------|
| Start central | `cd /opt/sentinel && docker compose -f deployments/docker-compose.central.yml up -d` |
| Stop central | `cd /opt/sentinel && docker compose -f deployments/docker-compose.central.yml down` |
| Central logs | `docker compose -f /opt/sentinel/deployments/docker-compose.central.yml logs -f central` |
| Start probe | `cd ~/sentinel-probe && docker compose up -d` |
| Probe logs | `cd ~/sentinel-probe && docker compose logs -f` |
| DB shell | `docker compose -f /opt/sentinel/deployments/docker-compose.central.yml exec timescaledb psql -U sentinel -d sentinel` |
| Health check | `curl -s https://updown.az/healthz` |
| All nodes | `curl -s https://updown.az/api/v1/nodes` |
| Update images | `docker compose pull && docker compose up -d` |
