-- 000011_backtest_meta.up.sql
-- Add optimization metadata to backtest_runs

ALTER TABLE backtest_runs ADD COLUMN IF NOT EXISTS timeframe VARCHAR(8) NOT NULL DEFAULT '1d';
ALTER TABLE backtest_runs ADD COLUMN IF NOT EXISTS batch_run_id UUID;
ALTER TABLE backtest_runs ADD COLUMN IF NOT EXISTS optimized_by VARCHAR(32) DEFAULT 'manual';

CREATE INDEX IF NOT EXISTS idx_backtest_runs_timeframe ON backtest_runs(timeframe);
CREATE INDEX IF NOT EXISTS idx_backtest_runs_batch ON backtest_runs(batch_run_id);
CREATE INDEX IF NOT EXISTS idx_backtest_runs_optimized ON backtest_runs(optimized_by);
