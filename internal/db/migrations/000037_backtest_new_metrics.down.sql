-- Revert 000037: Remove diagnostic/metrics columns from backtest_runs

ALTER TABLE backtest_runs
    DROP COLUMN IF EXISTS total_fees,
    DROP COLUMN IF EXISTS avg_slippage_bps,
    DROP COLUMN IF EXISTS calmar_ratio,
    DROP COLUMN IF EXISTS candle_count,
    DROP COLUMN IF EXISTS mtm_sharpe_ratio,
    DROP COLUMN IF EXISTS mtm_max_drawdown,
    DROP COLUMN IF EXISTS first_candle_time,
    DROP COLUMN IF EXISTS last_candle_time,
    DROP COLUMN IF EXISTS declared_bars_per_day,
    DROP COLUMN IF EXISTS effective_bars_per_day;
