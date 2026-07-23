-- 000015: Remove feature_vectors hypertable and indexes

DROP INDEX IF EXISTS idx_feature_vectors_symbol_time;
DROP INDEX IF EXISTS idx_feature_vectors_regime_time;
DROP TABLE IF EXISTS feature_vectors;
