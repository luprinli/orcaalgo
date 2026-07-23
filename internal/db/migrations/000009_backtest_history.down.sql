DROP TABLE IF EXISTS backtest_results;
ALTER TABLE backtest_runs DROP COLUMN IF EXISTS run_type;
ALTER TABLE backtest_runs DROP COLUMN IF EXISTS updated_at;
ALTER TABLE backtest_runs DROP COLUMN IF EXISTS completed_at;
ALTER TABLE backtest_runs DROP COLUMN IF EXISTS error_message;
ALTER TABLE backtest_runs ALTER COLUMN status SET DEFAULT 'pending';
