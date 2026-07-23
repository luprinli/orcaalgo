-- DOWN migration for regime-aware synthetic data
DROP TABLE IF EXISTS synthetic_generations;
ALTER TABLE candles DROP COLUMN IF EXISTS regime_label;
ALTER TABLE candles DROP COLUMN IF EXISTS data_source;
ALTER TABLE candles DROP COLUMN IF EXISTS generation_id;
