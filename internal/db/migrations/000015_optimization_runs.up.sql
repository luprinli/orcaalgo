-- 000014_optimization_runs.up.sql
CREATE TABLE IF NOT EXISTS optimization_runs (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    backtest_run_id  UUID REFERENCES backtest_runs(id) ON DELETE SET NULL,
    method           TEXT NOT NULL DEFAULT 'walkforward',
    objective_metric TEXT NOT NULL DEFAULT 'sharpe',
    total_trials     INTEGER NOT NULL DEFAULT 0,
    best_params      JSONB NOT NULL DEFAULT '{}',
    best_metric      FLOAT,
    param_ranges     JSONB,
    train_start      DATE,
    train_end        DATE,
    test_start       DATE,
    test_end         DATE,
    created_at       TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_optimization_runs_method ON optimization_runs(method);
CREATE INDEX IF NOT EXISTS idx_optimization_runs_created ON optimization_runs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_optimization_runs_backtest ON optimization_runs(backtest_run_id);
