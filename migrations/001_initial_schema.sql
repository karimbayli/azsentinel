-- Sentinel V2: Initial Schema
-- Requires: TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
-- ============================================================
-- Reference Tables
-- ============================================================
CREATE TABLE IF NOT EXISTS targets (
    url TEXT PRIMARY KEY,
    category TEXT NOT NULL CHECK (
        category IN ('GOV', 'BANK', 'ISP', 'MEDIA', 'ANCHOR')
    ),
    criticality INT NOT NULL DEFAULT 5 CHECK (
        criticality BETWEEN 0 AND 10
    ),
    enabled BOOL NOT NULL DEFAULT true,
    display_name TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS nodes (
    node_id TEXT PRIMARY KEY,
    region TEXT NOT NULL,
    country TEXT NOT NULL,
    enabled BOOL NOT NULL DEFAULT true
);
-- ============================================================
-- Hypertables
-- ============================================================
CREATE TABLE IF NOT EXISTS probe_results (
    time TIMESTAMPTZ NOT NULL,
    node_id TEXT NOT NULL,
    target_url TEXT NOT NULL,
    target_category TEXT NOT NULL,
    dns_resolve_ms INT,
    dns_resolved_ip TEXT,
    dns_error TEXT,
    tcp_dial_ms INT,
    tcp_success BOOL NOT NULL DEFAULT false,
    tls_handshake_ms INT,
    tls_valid BOOL,
    tls_expiry TIMESTAMPTZ,
    cert_issuer TEXT,
    http_status INT,
    ttfb_ms INT,
    total_ms INT,
    node_reliable BOOL NOT NULL DEFAULT true,
    error_type TEXT,
    error_detail TEXT
);
SELECT create_hypertable('probe_results', 'time', if_not_exists => TRUE);
CREATE INDEX IF NOT EXISTS idx_probe_results_node_target ON probe_results (node_id, target_url, time DESC);
CREATE INDEX IF NOT EXISTS idx_probe_results_target_time ON probe_results (target_url, time DESC);
-- ----------------------------------------------------------
CREATE TABLE IF NOT EXISTS bgp_events (
    time TIMESTAMPTZ NOT NULL,
    asn INT NOT NULL,
    provider TEXT NOT NULL,
    prefix TEXT NOT NULL,
    event_type TEXT NOT NULL CHECK (event_type IN ('ANNOUNCE', 'WITHDRAW')),
    peer_as INT,
    collector TEXT
);
SELECT create_hypertable('bgp_events', 'time', if_not_exists => TRUE);
CREATE INDEX IF NOT EXISTS idx_bgp_events_asn_time ON bgp_events (asn, time DESC);
-- ----------------------------------------------------------
CREATE TABLE IF NOT EXISTS correlation_results (
    time TIMESTAMPTZ NOT NULL,
    target_url TEXT NOT NULL,
    status TEXT NOT NULL CHECK (
        status IN (
            'HEALTHY',
            'DEGRADED',
            'PARTIAL_OUTAGE',
            'MAJOR_OUTAGE'
        )
    ),
    confidence DOUBLE PRECISION NOT NULL DEFAULT 0,
    node_signal DOUBLE PRECISION NOT NULL DEFAULT 0,
    bgp_signal DOUBLE PRECISION NOT NULL DEFAULT 0,
    social_signal DOUBLE PRECISION NOT NULL DEFAULT 0,
    signals_active TEXT [] NOT NULL DEFAULT '{}',
    total_nodes INT NOT NULL DEFAULT 0,
    failing_nodes INT NOT NULL DEFAULT 0
);
SELECT create_hypertable(
        'correlation_results',
        'time',
        if_not_exists => TRUE
    );
CREATE INDEX IF NOT EXISTS idx_correlation_target_time ON correlation_results (target_url, time DESC);
-- ----------------------------------------------------------
CREATE TABLE IF NOT EXISTS social_signals (
    time TIMESTAMPTZ NOT NULL,
    window_minutes INT NOT NULL DEFAULT 15,
    mention_count INT NOT NULL DEFAULT 0,
    baseline_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
    ratio DOUBLE PRECISION NOT NULL DEFAULT 0,
    sample_keywords TEXT [] NOT NULL DEFAULT '{}'
);
SELECT create_hypertable('social_signals', 'time', if_not_exists => TRUE);
-- ----------------------------------------------------------
CREATE TABLE IF NOT EXISTS node_health (
    time TIMESTAMPTZ NOT NULL,
    node_id TEXT NOT NULL,
    is_alive BOOL NOT NULL DEFAULT true,
    baseline_ok BOOL NOT NULL DEFAULT true,
    probe_count INT NOT NULL DEFAULT 0,
    avg_latency_ms INT NOT NULL DEFAULT 0,
    buffer_depth INT NOT NULL DEFAULT 0,
    version TEXT NOT NULL DEFAULT ''
);
SELECT create_hypertable('node_health', 'time', if_not_exists => TRUE);
CREATE INDEX IF NOT EXISTS idx_node_health_node_time ON node_health (node_id, time DESC);
-- ----------------------------------------------------------
CREATE TABLE IF NOT EXISTS incidents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    target_url TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ,
    peak_confidence DOUBLE PRECISION NOT NULL DEFAULT 0,
    peak_status TEXT NOT NULL DEFAULT 'HEALTHY',
    signals_fired TEXT [] NOT NULL DEFAULT '{}',
    notes TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_incidents_target_started ON incidents (target_url, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_incidents_open ON incidents (resolved_at)
WHERE resolved_at IS NULL;
-- ============================================================
-- Continuous Aggregates
-- ============================================================
-- FIX R-20: HTTP 401/403/304 count as reachable (server is up, just access-restricted)
CREATE MATERIALIZED VIEW IF NOT EXISTS probe_hourly WITH (timescaledb.continuous) AS
SELECT time_bucket('1 hour', time) AS bucket,
    target_url,
    node_id,
    COUNT(*) AS total_probes,
    COUNT(*) FILTER (
        WHERE tcp_success
            AND (
                http_status BETWEEN 200 AND 399
                OR http_status = 401
                OR http_status = 403
            )
    ) AS success_count,
    ROUND(
        100.0 * COUNT(*) FILTER (
            WHERE tcp_success
                AND (
                    http_status BETWEEN 200 AND 399
                    OR http_status = 401
                    OR http_status = 403
                )
        ) / NULLIF(COUNT(*), 0),
        2
    ) AS uptime_pct,
    AVG(total_ms)::INT AS avg_total_ms,
    AVG(ttfb_ms)::INT AS avg_ttfb_ms,
    MAX(total_ms) AS max_total_ms,
    MIN(total_ms) FILTER (
        WHERE total_ms > 0
    ) AS min_total_ms
FROM probe_results
WHERE node_reliable = true
GROUP BY bucket,
    target_url,
    node_id WITH NO DATA;
SELECT add_continuous_aggregate_policy(
        'probe_hourly',
        start_offset => INTERVAL '3 hours',
        end_offset => INTERVAL '1 hour',
        schedule_interval => INTERVAL '1 hour',
        if_not_exists => TRUE
    );
-- ----------------------------------------------------------
CREATE MATERIALIZED VIEW IF NOT EXISTS probe_daily WITH (timescaledb.continuous) AS
SELECT time_bucket('1 day', time) AS bucket,
    target_url,
    node_id,
    COUNT(*) AS total_probes,
    COUNT(*) FILTER (
        WHERE tcp_success
            AND http_status BETWEEN 200 AND 399
    ) AS success_count,
    ROUND(
        100.0 * COUNT(*) FILTER (
            WHERE tcp_success
                AND http_status BETWEEN 200 AND 399
        ) / NULLIF(COUNT(*), 0),
        2
    ) AS uptime_pct,
    AVG(total_ms)::INT AS avg_total_ms,
    AVG(ttfb_ms)::INT AS avg_ttfb_ms,
    MAX(total_ms) AS max_total_ms,
    MIN(total_ms) FILTER (
        WHERE total_ms > 0
    ) AS min_total_ms
FROM probe_results
WHERE node_reliable = true
GROUP BY bucket,
    target_url,
    node_id WITH NO DATA;
SELECT add_continuous_aggregate_policy(
        'probe_daily',
        start_offset => INTERVAL '3 days',
        end_offset => INTERVAL '1 day',
        schedule_interval => INTERVAL '1 day',
        if_not_exists => TRUE
    );
-- ============================================================
-- Retention Policies
-- ============================================================
SELECT add_retention_policy(
        'probe_results',
        INTERVAL '90 days',
        if_not_exists => TRUE
    );
SELECT add_retention_policy(
        'bgp_events',
        INTERVAL '90 days',
        if_not_exists => TRUE
    );
SELECT add_retention_policy(
        'correlation_results',
        INTERVAL '90 days',
        if_not_exists => TRUE
    );
SELECT add_retention_policy(
        'social_signals',
        INTERVAL '90 days',
        if_not_exists => TRUE
    );
SELECT add_retention_policy(
        'node_health',
        INTERVAL '90 days',
        if_not_exists => TRUE
    );
-- Hourly aggregate: 1 year
SELECT add_retention_policy(
        'probe_hourly',
        INTERVAL '1 year',
        if_not_exists => TRUE
    );
-- Daily aggregate: 5 years
SELECT add_retention_policy(
        'probe_daily',
        INTERVAL '5 years',
        if_not_exists => TRUE
    );