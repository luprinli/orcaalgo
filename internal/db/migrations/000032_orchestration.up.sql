-- Orchestration run metadata — one row per submitted orchestration backtest.
CREATE TABLE IF NOT EXISTS orchestration_runs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ,
    status          TEXT NOT NULL DEFAULT 'running',  -- running, completed, failed
    start_date      DATE NOT NULL,
    end_date        DATE NOT NULL,
    initial_capital NUMERIC(18,4) NOT NULL DEFAULT 100000,
    strategy_ids    TEXT[] NOT NULL,                   -- array of strategy IDs in this run
    symbol_tf_pairs TEXT[] NOT NULL,                   -- array of "SYMBOL:TF" pairs
    pool_sharpe     DOUBLE PRECISION,
    pool_sortino    DOUBLE PRECISION,
    pool_maxdd      DOUBLE PRECISION,
    pool_return_pct DOUBLE PRECISION,
    rebalance_costs NUMERIC(18,4),
    result_json     JSONB                                -- full result payload
);
CREATE INDEX IF NOT EXISTS idx_orch_runs_created ON orchestration_runs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_orch_runs_status ON orchestration_runs (status);

-- Allocation history — per-bar weights for each strategy in the orchestration.
CREATE TABLE IF NOT EXISTS allocation_history (
    id              BIGSERIAL PRIMARY KEY,
    run_id          UUID NOT NULL REFERENCES orchestration_runs(id) ON DELETE CASCADE,
    bar_time        TIMESTAMPTZ NOT NULL,
    strategy_id     TEXT NOT NULL,
    weight          DOUBLE PRECISION NOT NULL,           -- 0.0 to 1.0
    allocated_capital NUMERIC(18,4) NOT NULL,
    position_size   DOUBLE PRECISION,
    is_active       BOOLEAN NOT NULL DEFAULT false
);
CREATE INDEX IF NOT EXISTS idx_alloc_run_time ON allocation_history (run_id, bar_time);
CREATE INDEX IF NOT EXISTS idx_alloc_strategy ON allocation_history (strategy_id, bar_time);

-- Live strategy status — current state for each deployed strategy.
CREATE TABLE IF NOT EXISTS strategy_status (
    strategy_id     TEXT PRIMARY KEY,
    status          TEXT NOT NULL DEFAULT 'inactive',    -- active, inactive, standby, violated, validated
    allocation_pct  DOUBLE PRECISION NOT NULL DEFAULT 0,
    trailing_sharpe DOUBLE PRECISION,
    trailing_sortino DOUBLE PRECISION,
    trailing_maxdd  DOUBLE PRECISION,
    last_signal_at  TIMESTAMPTZ,
    active_since    TIMESTAMPTZ,
    demoted_at      TIMESTAMPTZ,
    demotion_reason TEXT,
    last_evaluated  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
