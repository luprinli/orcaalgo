-- 000010_candles_dedup.down.sql

ALTER TABLE candles DROP CONSTRAINT IF EXISTS candles_symbol_timeframe_time_unique;
ALTER TABLE candles DROP COLUMN IF EXISTS source;
