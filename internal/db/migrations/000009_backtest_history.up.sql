ALTER TABLE backtest_runs ADD COLUMN IF NOT EXISTS run_type TEXT NOT NULL DEFAULT 'single';
ALTER TABLE backtest_runs ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE backtest_runs ADD COLUMN IF NOT EXISTS completed_at TIMESTAMPTZ;
ALTER TABLE backtest_runs ADD COLUMN IF NOT EXISTS error_message TEXT;

CREATE TABLE IF NOT EXISTS backtest_results (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id          UUID NOT NULL REFERENCES backtest_runs(id) ON DELETE CASCADE,
    strategy_id     VARCHAR(64) NOT NULL,
    result_type     TEXT NOT NULL DEFAULT 'single',
    trial_index     INTEGER NOT NULL DEFAULT 0,
    parameters      JSONB,
    metrics         JSONB NOT NULL DEFAULT '{}',
    equity_curve    JSONB,
    trades          JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_backtest_results_run_id ON backtest_results(run_id);
CREATE INDEX IF NOT EXISTS idx_backtest_results_strategy ON backtest_results(strategy_id);

ALTER TABLE backtest_runs ALTER COLUMN status DROP DEFAULT;
ALTER TABLE backtest_runs ALTER COLUMN status SET DEFAULT 'completed';
