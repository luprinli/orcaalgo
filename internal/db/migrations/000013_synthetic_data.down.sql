-- 000012_synthetic_data.down.sql

DROP INDEX IF EXISTS idx_ticks_source_gen;
DROP INDEX IF EXISTS idx_candles_source_gen;
DROP TABLE IF EXISTS synthetic_generations CASCADE;
ALTER TABLE market_ticks DROP COLUMN IF EXISTS generation_id;
ALTER TABLE market_ticks DROP COLUMN IF EXISTS source;
ALTER TABLE candles DROP COLUMN IF EXISTS scenario;
ALTER TABLE candles DROP COLUMN IF EXISTS generation_id;
