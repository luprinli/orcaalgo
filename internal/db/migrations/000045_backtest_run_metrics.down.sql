ALTER TABLE backtest_runs
    DROP COLUMN IF EXISTS sortino_ratio,
    DROP COLUMN IF EXISTS max_drawdown_duration,
    DROP COLUMN IF EXISTS profit_factor,
    DROP COLUMN IF EXISTS avg_trade,
    DROP COLUMN IF EXISTS avg_win,
    DROP COLUMN IF EXISTS avg_loss,
    DROP COLUMN IF EXISTS num_wins,
    DROP COLUMN IF EXISTS num_losses,
    DROP COLUMN IF EXISTS gate_passed;
