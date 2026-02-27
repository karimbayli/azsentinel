# 🛰️ Sentinel V2

**Multi-node internet infrastructure monitoring platform for Azerbaijan's digital ecosystem.**

Sentinel V2 monitors the external reachability of Azerbaijan's critical internet services from multiple geographic vantage points, providing transparent, evidence-based outage detection.

## Features

- **Multi-Node Probing** — TCP/TLS/HTTP probes from 4+ global vantage nodes every 60 seconds
- **Resilient Architecture** — SQLite-backed local buffer queues survive central server outages
- **BGP Monitoring** — Real-time RIPE RIS WebSocket feed watching AZ ASNs for route withdrawals
- **Social Signal Analysis** — Telegram channel keyword monitoring in Azerbaijani, Russian, and English
- **Multi-Signal Correlation** — Weighted confidence scoring combining node failures (0.5), BGP events (0.3), and social spikes (0.2)
- **Telegram Alerts** — Instant notifications on state transitions with de-duplication
- **Public Transparency** — Open methodology documentation, never claims internal attribution
- **Calibration Mode** — First 30 days of operation learns baselines without alerting
- **Grafana Dashboards** — Pre-configured TimescaleDB dashboards with continuous aggregates

## Quick Start

```bash
# Clone
git clone <repo> && cd sentinel-v2

# Configure
cp .env.example .env
# Edit .env with your values

# Generate HMAC secret
bash scripts/generate-certs.sh

# Start central server (TimescaleDB + Grafana + Caddy)
docker compose -f deployments/docker-compose.central.yml up -d

# Verify
curl http://localhost:8080/healthz
```

## Architecture

```
Central Server (AZ)                    Vantage Nodes
┌──────────────────────┐         ┌─────────────────┐
│  Ingest API          │◄────────│  Probe Agent    │
│  Correlation Engine  │  HMAC   │  SQLite Buffer  │
│  BGP Monitor         │  HTTPS  │                 │
│  Social Monitor      │         │  node-eu (DE)   │
│  TimescaleDB         │         │  node-us (US)   │
│  Grafana + Caddy     │         │  node-asia (SG) │
└──────────────────────┘         └─────────────────┘
```

## Monitored Targets

| Category | Targets | Examples |
|----------|---------|---------|
| Government | 4 | e-gov.az, asan.gov.az, taxes.gov.az |
| Banking | 4 | abb-bank.az, pashabank.az, kapitalbank.az |
| ISP | 4 | delta.az, aztelekom.az, bakcell.az |
| Media | 3 | oxu.az, report.az, 1news.az |
| Anchors | 3 | 1.1.1.1, google.com, cloudflare.com |

## Development

```bash
# Build
make build

# Test
make test

# Lint
make lint

# Run locally
make dev-central
```

## Documentation

- [Architecture](docs/ARCHITECTURE.md) — System design and data flow
- [Methodology](docs/METHODOLOGY.md) — What we measure and how
- [Deployment](docs/DEPLOYMENT.md) — Installation and deployment guide
- [API Reference](docs/API.md) — REST API documentation
- [Operations Runbook](docs/OPERATIONS_RUNBOOK.md) — Day-to-day operations
- [Security](docs/SECURITY.md) — Security model and recommendations
- [Troubleshooting](docs/TROUBLESHOOTING.md) — Common issues and fixes

## Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.24 |
| Database | TimescaleDB (PostgreSQL 16) |
| Dashboards | Grafana 10 |
| Reverse Proxy | Caddy 2 |
| BGP Data | RIPE RIS Live WebSocket |
| Alerting | Telegram Bot API |
| Metrics | Prometheus client |
| Probe Buffer | SQLite (WAL mode) |

## License

Private — All rights reserved.
