-- 000010_candles_dedup.up.sql
-- Add unique constraint for deduplication and source column for auditability

ALTER TABLE candles DROP CONSTRAINT IF EXISTS candles_symbol_timeframe_time_unique;
ALTER TABLE candles ADD CONSTRAINT candles_symbol_timeframe_time_unique UNIQUE (symbol_id, timeframe, time);

ALTER TABLE candles ADD COLUMN IF NOT EXISTS source VARCHAR(32) NOT NULL DEFAULT 'seed';
