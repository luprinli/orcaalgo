-- 000017_engine_version.up.sql
-- Add engine version and strategy hash tracking for backtest-live parity audit

ALTER TABLE backtest_runs ADD COLUMN IF NOT EXISTS engine_version VARCHAR(64) NOT NULL DEFAULT 'unknown';
ALTER TABLE backtest_runs ADD COLUMN IF NOT EXISTS strategy_hash VARCHAR(128);
ALTER TABLE backtest_runs ADD COLUMN IF NOT EXISTS schema_version INTEGER NOT NULL DEFAULT 1;

ALTER TABLE backtest_results ADD COLUMN IF NOT EXISTS engine_version VARCHAR(64) NOT NULL DEFAULT 'unknown';
ALTER TABLE backtest_results ADD COLUMN IF NOT EXISTS strategy_hash VARCHAR(128);

ALTER TABLE trade_executions ADD COLUMN IF NOT EXISTS engine_version VARCHAR(64) NOT NULL DEFAULT 'unknown';
ALTER TABLE trade_executions ADD COLUMN IF NOT EXISTS strategy_hash VARCHAR(128);
