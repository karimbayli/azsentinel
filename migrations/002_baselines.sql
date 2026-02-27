-- Sentinel V2: Baseline tables for calibration mode
CREATE TABLE IF NOT EXISTS social_baselines (
    id SERIAL PRIMARY KEY,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    window_minutes INT NOT NULL DEFAULT 15,
    avg_mentions DOUBLE PRECISION NOT NULL DEFAULT 0,
    stddev_mentions DOUBLE PRECISION NOT NULL DEFAULT 0,
    sample_count INT NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS latency_baselines (
    id SERIAL PRIMARY KEY,
    target_url TEXT NOT NULL,
    node_id TEXT NOT NULL,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    avg_total_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    p95_total_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    p99_total_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    avg_ttfb_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    sample_count INT NOT NULL DEFAULT 0,
    UNIQUE (target_url, node_id)
);
CREATE INDEX IF NOT EXISTS idx_latency_baselines_target ON latency_baselines (target_url);