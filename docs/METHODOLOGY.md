# Sentinel V2 — Methodology

## What We Measure

Sentinel V2 measures **external reachability** of configured Azerbaijan internet targets from multiple geographically distributed **vantage points**. For each target, we perform:

1. **TCP Connection** — Can we establish a TCP connection to the target's IP address on port 443? (5-second timeout)
2. **TLS Handshake** — Does the server present a valid TLS certificate? We extract certificate metadata (issuer, expiry).
3. **HTTP GET** — Does the server respond to an HTTP GET request with a valid status code? We measure Time to First Byte (TTFB) and total response time.

Each measurement cycle runs every **60 seconds** from each vantage node simultaneously.

## What We Do NOT Measure

- **Internal ISP backbone performance** — We cannot observe peering points, internal routing, or backbone congestion within Azerbaijan.
- **DNS resolution from within Azerbaijan** — Our DNS resolution happens at each vantage node's location.
- **Geo-blocked content** — Some services may be accessible from external nodes but blocked internally, or vice versa.
- **Application-layer functionality** — We check HTTP status codes, not whether the application itself is functioning correctly.
- **Latency from within Azerbaijan** — Our Azerbaijan vantage point provides one perspective, not a comprehensive internal view.

## Confidence Model

We use a **weighted multi-signal correlation model** to assess the likelihood and severity of an outage. Three independent signals contribute to a confidence score between 0.0 and 1.0:

### Signal Weights

| Signal | Weight | Description |
|--------|--------|-------------|
| Multi-Node Failure Agreement | **0.5** | If >80% of reliable vantage nodes report failure for a target |
| BGP Route Withdrawal | **0.3** | If RIPE RIS detects route withdrawals for AZ ASNs (29049, 31721, 39232, 34377, 57021) |
| Social Mention Spike | **0.2** | If Telegram keyword mentions exceed 3× the 30-day rolling baseline |

### Confidence → Status Mapping

| Confidence Score | Status | Meaning |
|-----------------|--------|---------|
| > 0.8 | **MAJOR_OUTAGE** | High confidence that the target is experiencing a significant outage |
| > 0.5 | **PARTIAL_OUTAGE** | Multiple signals indicate degradation or partial unreachability |
| > 0.3 | **DEGRADED** | Some signals indicate potential issues |
| ≤ 0.3 | **HEALTHY** | Target appears to be operating normally |

### Node Reliability

Before probing targets, each node checks three **baseline anchor** targets: 1.1.1.1, google.com, and cloudflare.com. If 2 or more anchors fail from a node, that node is marked as **NODE_UNRELIABLE**. Unreliable node data is still stored but is **excluded** from correlation confidence calculations, preventing false positives from node-side connectivity issues.

## Known Limitations

1. **Geographic coverage** — Currently 4 vantage points. More nodes would improve confidence.
2. **BGP data latency** — RIPE RIS data may have 1-5 minute delay from actual route changes.
3. **Social signal noise** — Keyword matching may produce false positives from unrelated discussions.
4. **Single AZ vantage point** — Our in-country node provides one perspective, not comprehensive internal coverage.
5. **Central server location** — During national outages, the central server may become unreachable. Vantage nodes buffer data locally.
6. **TLS inspection** — Some CDNs or WAFs may present different certificates or behavior to external probes.

## Vantage Node Locations

| Node ID | Region | Country |
|---------|--------|---------|
| node-az | Baku | Azerbaijan |
| node-eu | Frankfurt | Germany |
| node-us | US East | United States |
| node-asia | Singapore | Singapore |

## Data Retention

| Data Type | Raw Retention | Hourly Aggregates | Daily Aggregates |
|-----------|--------------|-------------------|-----------------|
| Probe Results | 90 days | 1 year | 5 years |
| BGP Events | 90 days | — | — |
| Correlation Results | 90 days | — | — |
| Social Signals | 90 days | — | — |

## Transparency Commitment

This system is designed to provide **transparent, evidence-based** assessments of Azerbaijan's internet infrastructure reachability. We:

- Publish our methodology openly
- Clearly state what we can and cannot measure
- Provide raw data export capabilities
- Never claim internal outage attribution without internal vantage evidence
- Mark all assessments with explainable source attribution
