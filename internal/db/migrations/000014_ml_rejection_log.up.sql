-- 000013: ML rejection log hypertable
-- Tracks every signal rejected by the meta-labeling model for audit,
-- debugging, and model improvement analysis.
-- TimescaleDB hypertable with compression (7-day) and retention (90-day).

CREATE TABLE IF NOT EXISTS ml_rejection_log (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp       TIMESTAMPTZ NOT NULL,
    symbol          TEXT NOT NULL,
    strategy        TEXT NOT NULL,
    model_name      TEXT NOT NULL,
    model_version   TEXT NOT NULL,
    p_win           DOUBLE PRECISION NOT NULL,
    threshold       DOUBLE PRECISION NOT NULL,
    raw_signal      JSONB NOT NULL,
    feature_values  JSONB NOT NULL,
    feature_importance JSONB NOT NULL,
    rejected_at     TIMESTAMPTZ DEFAULT NOW()
);

-- Convert to hypertable for time-series query performance
SELECT create_hypertable('ml_rejection_log', 'timestamp', if_not_exists => true);

-- Enable compression: compress chunks older than 7 days
ALTER TABLE ml_rejection_log SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'model_name',
    timescaledb.compress_orderby = 'timestamp DESC'
);

-- Add compression policy for chunks older than 7 days
SELECT add_compression_policy('ml_rejection_log', INTERVAL '7 days', if_not_exists => true);

-- Add retention policy: keep 90 days of rejection logs
SELECT add_retention_policy('ml_rejection_log', INTERVAL '90 days', if_not_exists => true);

-- Index for common query patterns
CREATE INDEX IF NOT EXISTS idx_ml_rejection_model_time
    ON ml_rejection_log (model_name, timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_ml_rejection_symbol_time
    ON ml_rejection_log (symbol, timestamp DESC);
