-- 000020_backtest_results_schema_version.up.sql
-- Add schema_version to backtest_results for cross-version result comparison (P10).
-- Backward-safe: NOT NULL with a DEFAULT so existing rows read as version 1.

ALTER TABLE backtest_results ADD COLUMN IF NOT EXISTS schema_version INTEGER NOT NULL DEFAULT 1;
