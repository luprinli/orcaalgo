DROP TABLE IF EXISTS backtest_run_summaries;
DROP INDEX IF EXISTS idx_backtest_results_retention;
ALTER TABLE backtest_results DROP COLUMN IF EXISTS retention_class;
