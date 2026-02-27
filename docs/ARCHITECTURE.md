# Sentinel V2 — Architecture

## System Overview

Sentinel V2 is a multi-node internet infrastructure monitoring platform that assesses the external reachability of Azerbaijan's key digital services from multiple geographic vantage points.

```
┌─────────────────────────────────────────────────────────────┐
│                    CENTRAL SERVER (Azerbaijan)              │
│                                                             │
│  ┌──────────┐  ┌──────────┐  ┌───────────┐  ┌───────────┐ │
│  │ Ingest   │  │ BGP      │  │ Social    │  │ REST API  │ │
│  │ API      │  │ Monitor  │  │ Monitor   │  │ + Status  │ │
│  └────┬─────┘  └────┬─────┘  └─────┬─────┘  └───────────┘ │
│       │              │              │                       │
│       ▼              ▼              ▼                       │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Correlation Engine (30s)                │   │
│  │  node_signal(0.5) + bgp_signal(0.3) + social(0.2)  │   │
│  └───────────────────────┬─────────────────────────────┘   │
│                          │                                  │
│                  ┌───────▼───────┐   ┌──────────┐          │
│                  │  TimescaleDB  │   │ Telegram │          │
│                  │  (PostgreSQL) │   │ Alerter  │          │
│                  └───────────────┘   └──────────┘          │
│                                                             │
│  ┌──────────┐        ┌────────┐       ┌─────────┐         │
│  │ Local    │        │ Caddy  │       │ Grafana │         │
│  │ Prober   │        │ (TLS)  │       │  10+    │         │
│  └──────────┘        └────────┘       └─────────┘         │
└─────────────────────────────────────────────────────────────┘
        ▲                    ▲                    ▲
        │ HMAC/HTTPS         │ HMAC/HTTPS         │ HMAC/HTTPS
┌───────┴──────┐  ┌──────────┴──────┐  ┌──────────┴──────┐
│ Vantage Node │  │  Vantage Node   │  │  Vantage Node   │
│   EU (DE)    │  │    US (East)    │  │   Asia (SG)     │
│              │  │                 │  │                  │
│ ┌──────────┐ │  │ ┌──────────┐   │  │ ┌──────────┐    │
│ │ Prober   │ │  │ │ Prober   │   │  │ │ Prober   │    │
│ └──────────┘ │  │ └──────────┘   │  │ └──────────┘    │
│ ┌──────────┐ │  │ ┌──────────┐   │  │ ┌──────────┐    │
│ │ SQLite   │ │  │ │ SQLite   │   │  │ │ SQLite   │    │
│ │ Buffer   │ │  │ │ Buffer   │   │  │ │ Buffer   │    │
│ └──────────┘ │  │ └──────────┘   │  │ └──────────┘    │
└──────────────┘  └────────────────┘  └─────────────────┘
```

## Key Design Decisions

### 1. Central Brain in Azerbaijan
The central server lives inside AZ for data sovereignty. During national outages, external nodes buffer results locally (SQLite) and flush on reconnect.

### 2. HMAC Authentication
All node→central communication is signed with HMAC-SHA256 to prevent forged probe data injection.

### 3. Anchor-Based Node Reliability
Each probe cycle checks 1.1.1.1, google.com, and cloudflare.com first. If 2+ fail, the node is marked unreliable and excluded from correlation confidence calculations.

### 4. Multi-Signal Correlation
Three independent signals are weighted to compute confidence:
- **Node agreement (0.5)**: Cross-node failure consensus
- **BGP withdrawal (0.3)**: RIPE RIS Live route monitoring
- **Social spike (0.2)**: Telegram keyword mention rates

### 5. ANCHOR Target Exclusion
Anchor targets (1.1.1.1, Google, Cloudflare) are NEVER included in outage alerts — they exist solely to validate node health.

## Data Flow

1. Probe agents run TCP→TLS→HTTP checks every 60s
2. Results are batch-POSTed to central's `/api/v1/ingest/probe-batch` with HMAC signature
3. If central is unreachable, results are buffered in local SQLite
4. Central stores results in TimescaleDB hypertables
5. Correlation engine runs every 30s, computing confidence per-target
6. State transitions trigger Telegram alerts (unless in calibration mode)
7. Grafana queries TimescaleDB for real-time dashboards

## Technology Choices

| Component | Technology | Rationale |
|-----------|-----------|-----------|
| Language | Go 1.24 | Performance, single binary, strong concurrency |
| Database | TimescaleDB | Native hypertables, continuous aggregates, retention policies |
| Probe buffer | SQLite | Zero-dependency, WAL mode, resilient to crashes |
| BGP data | RIPE RIS Live WS | Real-time BGP feed, Caucasus collector rrc21 |
| Alerting | Telegram Bot API | Widely used in AZ, instant delivery |
| Metrics | Prometheus client | Standard observability |
| TLS | Caddy | Automatic Let's Encrypt, zero-config TLS |
| Dashboards | Grafana 10 | Industry standard time-series visualization |
