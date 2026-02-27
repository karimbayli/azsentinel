# UpDown.az

**Real-time internet infrastructure monitoring for Azerbaijan — from the outside looking in.**

UpDown.az monitors the external reachability of Azerbaijan's critical internet services from multiple geographic vantage points, providing transparent, evidence-based outage detection with multi-signal correlation.

> Beta: `updown.az` → Production: `netwatch.az`

---

## What It Does

From 4 globally distributed probe nodes, UpDown.az performs DNS/TCP/TLS/HTTP checks against 125+ Azerbaijani targets every 60 seconds. When a service goes unreachable, the system correlates probe failures with BGP routing data and social signals to produce a confidence-scored diagnosis:

- **Global outage** — all probes fail → server is down everywhere
- **Local block detected** — AZ probe fails, global probes succeed → domestic filtering
- **External routing issue** — AZ probe succeeds, global probes fail → BGP/transit problem

## Features

| Feature | Description |
|---------|-------------|
| **Multi-Node Probing** | TCP/TLS/HTTP probes from 4 vantage nodes (Frankfurt, Baku, Ashburn, Mumbai) every 60s |
| **125+ Targets** | Government (30), Banking (15), Media (30), ISP (10), Popular Services (30), Global Platforms (20) |
| **AZ/Global Perspective** | Each target card shows "AZ: Reachable" vs "Global: 3/3 OK" with smart block detection |
| **BGP Monitoring** | Real-time RIPE RIS Live feed watching Azerbaijani ASNs (29049, 31721, 39232, 34377, 57021) |
| **Multi-Signal Correlation** | Weighted confidence: Node (50%) + BGP (30%) + Social (20%) |
| **Bilingual UI** | Professional Azerbaijani/English interface with geo-IP auto-detection |
| **Dot-Matrix World Map** | Premium infrastructure-style map with probe locations and arc connections |
| **Calibration Mode** | First 7–30 days learns latency baselines without alerting |
| **Telegram Alerts** | Instant notifications on state transitions with de-duplication |
| **Grafana Dashboards** | Pre-configured TimescaleDB dashboards with continuous aggregates |
| **Database-Driven Targets** | Add/remove targets via SQL — no redeployment needed |
| **Zero Additional Cost** | Runs on existing Hetzner + AZ VPS + Oracle Cloud Always Free |

## Architecture

```
                    Hetzner (Frankfurt)                     Probe Nodes
              ┌──────────────────────────┐
              │  Caddy (TLS :443)        │          ┌──────────────────┐
              │  ┌────────────────────┐  │   HMAC   │  node-eu (DE)    │
              │  │  Central Server    │◄─┼──────────│  node-az (AZ)    │
              │  │  Correlation Engine│  │   HTTPS  │  node-us (US)    │
              │  │  BGP Monitor      │  │          │  node-asia (IN)  │
              │  └────────┬───────────┘  │          └──────────────────┘
              │           │              │
              │  ┌────────▼───────────┐  │          Probe Agent:
              │  │  TimescaleDB       │  │          • Go binary (~15MB)
              │  │  (PostgreSQL 16)   │  │          • SQLite buffer queue
              │  └────────────────────┘  │          • 128MB RAM, 0.25 CPU
              │                          │          • HMAC-signed reports
              │  Grafana (:3000)         │
              └──────────────────────────┘
```

## Monitored Targets

| Category | Count | Examples |
|----------|:-----:|---------|
| **Government** | 30 | e-gov.az, ASAN, all ministries, CBAR, Milli Məclis |
| **Banking & Finance** | 14 | ABB, PASHA, Kapital, BirBank, IBA, Xalq Bank |
| **Fintech** | 1 | M10 Payments |
| **Media & News** | 29 | APA, AzərTAc, Oxu, Trend, Caliber, AzTV, İctimai TV |
| **ISP & Telecom** | 10 | Azercell, Delta, Bakcell, Nar, CityNet, Starlink |
| **Popular Services** | 30 | SOCAR, AZAL, Turbo.az, Tap.az, Wolt, Bolt, ADA |
| **Global Platforms** | 17 | Google, YouTube, Facebook, Telegram, GitHub |
| **Anchors** | 3 | Cloudflare DNS, Google DNS, Cloudflare |

## Quick Start

```bash
# Clone
git clone https://github.com/karimbayli/azsentinel.git && cd azsentinel

# Configure
cp .env.example .env
# Edit .env with your secrets (see docs/DEPLOYMENT.md)

# Generate HMAC secret
openssl rand -hex 32

# Start central stack
docker compose -f deployments/docker-compose.central.yml up -d

# Verify
curl http://localhost:8080/healthz
```

For the full step-by-step guide with probe deployment, see **[docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)**.

## Infrastructure

| Role | Server | Cost |
|------|--------|------|
| Central + EU Probe | Hetzner (Frankfurt) | Existing |
| AZ Probe | Azerbaijan VPS (1 CPU, 2GB) | Existing |
| US Probe | Oracle Cloud Always Free (Ashburn) | $0 |
| Asia Probe | Oracle Cloud Always Free (Ashburn) | $0 |
| **Total additional** | | **$0/month** |

## Development

```bash
make build          # Build binaries
make test           # Run tests
make lint           # Lint code
make dev-central    # Run locally
```

## Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.24 |
| Database | TimescaleDB (PostgreSQL 16) |
| Frontend | Vanilla HTML/CSS/JS, Lucide SVG icons |
| Dashboards | Grafana 10 |
| Reverse Proxy | Caddy 2 (auto TLS) |
| BGP Data | RIPE RIS Live WebSocket |
| Alerting | Telegram Bot API |
| Probe Buffer | SQLite (WAL mode) |
| CI/CD | GitHub Actions → GHCR |
| Containers | Multi-stage Alpine (<20MB images) |

## Documentation

| Doc | Description |
|-----|-------------|
| [DEPLOYMENT.md](docs/DEPLOYMENT.md) | Full deployment guide (16 steps) |
| [ARCHITECTURE.md](docs/ARCHITECTURE.md) | System design and data flow |
| [METHODOLOGY.md](docs/METHODOLOGY.md) | What we measure and how |
| [API.md](docs/API.md) | REST API reference |
| [OPERATIONS_RUNBOOK.md](docs/OPERATIONS_RUNBOOK.md) | Day-to-day operations |
| [SECURITY.md](docs/SECURITY.md) | Security model |
| [TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) | Common issues and fixes |

## License

Private — All rights reserved.
