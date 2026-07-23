-- Tiered retention for matrix backtest results (see docs/backtest_retention_policy.md).
-- retention_class: 0=T0 island (full, permanent), 1=T1 neighborhood (metrics+downsampled
-- equity), 2=T2 landscape sample (metrics only), 3=T3 tail (aggregate only — rarely a row).
ALTER TABLE backtest_results ADD COLUMN IF NOT EXISTS retention_class SMALLINT NOT NULL DEFAULT 2;
CREATE INDEX IF NOT EXISTS idx_backtest_results_retention ON backtest_results(retention_class, created_at);

-- Permanent per-run aggregate: preserves the shape of the whole parameter space
-- (score distribution, viability, failure taxonomy, Pareto front, effective trials)
-- even after off-island rows are pruned — avoids survivorship bias.
CREATE TABLE IF NOT EXISTS backtest_run_summaries (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_run_id     VARCHAR(64) NOT NULL,
    run_id           UUID,
    total_combos     INTEGER NOT NULL DEFAULT 0,
    traded_combos    INTEGER NOT NULL DEFAULT 0,
    zero_trade       INTEGER NOT NULL DEFAULT 0,
    errored          INTEGER NOT NULL DEFAULT 0,
    effective_trials INTEGER NOT NULL DEFAULT 0,
    score_histogram  JSONB NOT NULL DEFAULT '{}',
    viability        JSONB NOT NULL DEFAULT '{}',
    failure_reasons  JSONB NOT NULL DEFAULT '{}',
    pareto_front     JSONB NOT NULL DEFAULT '[]',
    best_sharpe      DOUBLE PRECISION,
    best_combo       VARCHAR(160),
    engine_version   VARCHAR(64),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_backtest_run_summaries_batch ON backtest_run_summaries(batch_run_id);
CREATE INDEX IF NOT EXISTS idx_backtest_run_summaries_created ON backtest_run_summaries(created_at);
