-- Reverting the widened strategy_id back to UUID is lossy (names cannot become
-- UUIDs); we only drop the additive columns. strategy_id/symbol_set nullability
-- is left relaxed (harmless).
ALTER TABLE backtest_runs DROP COLUMN IF EXISTS config;
ALTER TABLE backtest_runs DROP COLUMN IF EXISTS symbols;
ALTER TABLE backtest_runs DROP COLUMN IF EXISTS strategy_ids;
