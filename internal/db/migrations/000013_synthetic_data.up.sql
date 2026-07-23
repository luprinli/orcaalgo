-- 000012_synthetic_data.up.sql
-- Add generation tracking columns and synthetic generations audit table

ALTER TABLE candles ADD COLUMN IF NOT EXISTS generation_id TEXT;
ALTER TABLE candles ADD COLUMN IF NOT EXISTS scenario TEXT;

ALTER TABLE market_ticks ADD COLUMN IF NOT EXISTS source VARCHAR(32) DEFAULT 'api';
ALTER TABLE market_ticks ADD COLUMN IF NOT EXISTS generation_id TEXT;

CREATE TABLE IF NOT EXISTS synthetic_generations (
    id TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    config JSONB NOT NULL,
    symbol TEXT NOT NULL,
    timeframe TEXT NOT NULL,
    n_candles INTEGER,
    validation_status TEXT DEFAULT 'pending',
    validation_report JSONB
);

CREATE INDEX IF NOT EXISTS idx_candles_source_gen
    ON candles (source, generation_id)
    WHERE generation_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_ticks_source_gen
    ON market_ticks (source, generation_id)
    WHERE generation_id IS NOT NULL;
