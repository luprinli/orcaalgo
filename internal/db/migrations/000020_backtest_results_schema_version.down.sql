-- 000020_backtest_results_schema_version.down.sql
ALTER TABLE backtest_results DROP COLUMN IF EXISTS schema_version;
