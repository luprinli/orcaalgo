-- DOWN: Remove user_id columns from tables
ALTER TABLE accounts DROP COLUMN IF EXISTS user_id;
ALTER TABLE strategies DROP COLUMN IF EXISTS user_id;
ALTER TABLE backtest_runs DROP COLUMN IF EXISTS user_id;
ALTER TABLE trade_executions DROP COLUMN IF EXISTS user_id;
ALTER TABLE credentials DROP COLUMN IF EXISTS user_id;
ALTER TABLE settings DROP COLUMN IF EXISTS user_id;
ALTER TABLE kill_switch_history DROP COLUMN IF EXISTS triggered_by;
