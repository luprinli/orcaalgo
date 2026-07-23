-- 000016_engine_version.down.sql

ALTER TABLE backtest_runs DROP COLUMN IF EXISTS engine_version;
ALTER TABLE backtest_runs DROP COLUMN IF EXISTS strategy_hash;
ALTER TABLE backtest_runs DROP COLUMN IF EXISTS schema_version;

ALTER TABLE backtest_results DROP COLUMN IF EXISTS engine_version;
ALTER TABLE backtest_results DROP COLUMN IF EXISTS strategy_hash;

ALTER TABLE trade_executions DROP COLUMN IF EXISTS engine_version;
ALTER TABLE trade_executions DROP COLUMN IF EXISTS strategy_hash;
