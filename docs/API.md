# Sentinel V2 — API Reference

Base URL: `https://your-domain.com`

## Public Endpoints

### GET /healthz
Service health check.

**Response:**
```json
{"status": "ok", "version": "1.0.0", "time": "2025-01-15T10:30:00Z"}
```

### GET /api/v1/methodology
Transparency page with measurement methodology, confidence model, and limitations.

### GET /api/v1/status
Current status of all monitored targets.

**Response:** Array of `TargetStatus` objects:
```json
[
  {
    "target": {"url": "https://e-gov.az", "category": "GOV", "criticality": 10, "display_name": "E-Government Portal"},
    "status": "HEALTHY",
    "confidence": 0.0,
    "last_check": "2025-01-15T10:30:00Z",
    "active_incident": null
  }
]
```

### GET /api/v1/status/{target}
Single target status with per-node breakdown.

**Parameters:** `target` — URL-encoded target identifier (e.g., `e-gov.az`)

**Response:**
```json
{
  "target": {"url": "https://e-gov.az", ...},
  "status": "HEALTHY",
  "confidence": 0.0,
  "node_breakdown": [
    {"node_id": "node-az", "region": "az-baku", "tcp_success": true, "http_status": 200, "total_ms": 150, "reliable": true}
  ]
}
```

### GET /api/v1/incidents?from=&to=
Incident history within a time range.

**Parameters:**
- `from` — RFC3339 timestamp (default: 24h ago)
- `to` — RFC3339 timestamp (default: now)

### GET /api/v1/bgp/events?hours=24
BGP route events.

**Parameters:** `hours` — lookback window (1-168, default: 24)

### GET /api/v1/nodes
Current health of all vantage nodes.

### GET /api/v1/history/{target}?hours=24
Probe result history for a specific target.

**Parameters:**
- `target` — URL identifier
- `hours` — lookback window (1-720, default: 24)

### GET /api/v1/export/csv
Export last 7 days of probe results as CSV.

## Authenticated Endpoints

### POST /api/v1/ingest/probe-batch
Receive probe results from vantage nodes.

**Headers:**
- `X-Sentinel-Signature` — HMAC-SHA256 hex digest of the request body
- `X-Sentinel-Node` — Node identifier
- `Content-Type: application/json`

**Body:**
```json
{
  "node_id": "node-eu",
  "region": "eu-frankfurt",
  "timestamp": "2025-01-15T10:30:00Z",
  "version": "1.0.0",
  "results": [
    {
      "time": "2025-01-15T10:30:00Z",
      "node_id": "node-eu",
      "target_url": "https://e-gov.az",
      "target_category": "GOV",
      "tcp_dial_ms": 45,
      "tcp_success": true,
      "tls_handshake_ms": 120,
      "tls_valid": true,
      "http_status": 200,
      "ttfb_ms": 250,
      "total_ms": 500,
      "node_reliable": true
    }
  ]
}
```

**Response:** `{"status": "accepted"}`

## Error Responses

All errors return JSON:
```json
{"error": "description of the error"}
```

| Code | Meaning |
|------|---------|
| 400 | Bad request (invalid parameters) |
| 401 | Missing HMAC signature |
| 403 | Invalid HMAC signature |
| 404 | Target not found |
| 500 | Internal server error |
