-- 000013: Remove ml_rejection_log hypertable and indexes

DROP INDEX IF EXISTS idx_ml_rejection_model_time;
DROP INDEX IF EXISTS idx_ml_rejection_symbol_time;
DROP TABLE IF EXISTS ml_rejection_log;
