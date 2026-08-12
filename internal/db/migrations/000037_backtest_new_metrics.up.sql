-- 000037: Add new diagnostic/metrics columns to backtest_runs
-- Supports the Backtest Results Audit (2026-08-11) Phase 0-6 enhancements:
--   total_fees, avg_slippage_bps, calmar_ratio, candle_count,
--   mtm_sharpe_ratio, mtm_max_drawdown,
--   first_candle_time, last_candle_time, declared_bars_per_day, effective_bars_per_day
-- These are runtime-computed fields that were previously only available in CSV/JSON but not queryable.

ALTER TABLE backtest_runs
    ADD COLUMN IF NOT EXISTS total_fees DOUBLE PRECISION DEFAULT 0,
    ADD COLUMN IF NOT EXISTS avg_slippage_bps DOUBLE PRECISION DEFAULT 0,
    ADD COLUMN IF NOT EXISTS calmar_ratio DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS candle_count INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS mtm_sharpe_ratio DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS mtm_max_drawdown DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS first_candle_time TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_candle_time TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS declared_bars_per_day DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS effective_bars_per_day DOUBLE PRECISION;
