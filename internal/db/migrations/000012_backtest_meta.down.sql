-- 000011_backtest_meta.down.sql

ALTER TABLE backtest_runs DROP COLUMN IF EXISTS timeframe;
ALTER TABLE backtest_runs DROP COLUMN IF EXISTS batch_run_id;
ALTER TABLE backtest_runs DROP COLUMN IF EXISTS optimized_by;
