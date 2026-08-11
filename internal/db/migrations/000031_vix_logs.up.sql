CREATE TABLE IF NOT EXISTS vix_logs (
    id          BIGSERIAL PRIMARY KEY,
    timestamp   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    vix_value   DOUBLE PRECISION NOT NULL,
    vix_change  DOUBLE PRECISION NOT NULL DEFAULT 0,
    source      TEXT NOT NULL DEFAULT 'synthetic'
);

CREATE INDEX IF NOT EXISTS idx_vix_logs_timestamp ON vix_logs (timestamp DESC);
