-- 000021: Feature store state persistence
-- Stores the raw ring buffer state (OHLCV data) per symbol for the FeatureStore.
-- Enables restoring the Go FeatureStore ring buffer after server restart.
-- One row per symbol, updated on each persist (ON CONFLICT upsert).

CREATE TABLE IF NOT EXISTS feature_store_state (
    symbol      TEXT PRIMARY KEY,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    bar_count   INTEGER NOT NULL DEFAULT 0,
    prices      DOUBLE PRECISION[] NOT NULL DEFAULT '{}',
    highs       DOUBLE PRECISION[] NOT NULL DEFAULT '{}',
    lows        DOUBLE PRECISION[] NOT NULL DEFAULT '{}',
    volumes     DOUBLE PRECISION[] NOT NULL DEFAULT '{}'
);

COMMENT ON TABLE feature_store_state IS 'Persisted raw OHLCV ring buffer state for FeatureStore restore after restart';
COMMENT ON COLUMN feature_store_state.prices IS 'Close price ring buffer (max 256 elements)';
COMMENT ON COLUMN feature_store_state.bar_count IS 'Number of bars in the ring buffer';
