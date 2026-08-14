-- 000045: Add diagnostic metrics columns to backtest_runs.
--
-- The repository write model (CreateBacktestRun / CreateBacktestRunsBatch)
-- persists Sortino, ProfitFactor, per-trade averages and the win/loss split, but
-- the original 000001 schema never declared these columns. Every persist failed
-- with:
--   column "sortino_ratio" of relation "backtest_runs" does not exist
-- This aligns the schema with the code. All columns are nullable so error and
-- in-flight records that do not yet carry metrics still insert cleanly.

ALTER TABLE backtest_runs
    ADD COLUMN IF NOT EXISTS sortino_ratio         DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS max_drawdown_duration INT,
    ADD COLUMN IF NOT EXISTS profit_factor         DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS avg_trade             DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS avg_win               DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS avg_loss              DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS num_wins              INT,
    ADD COLUMN IF NOT EXISTS num_losses            INT,
    ADD COLUMN IF NOT EXISTS gate_passed           BOOLEAN;
