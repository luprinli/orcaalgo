CREATE TABLE IF NOT EXISTS strategy_params_version (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id     TEXT NOT NULL,
    version_tag     TEXT NOT NULL,
    params          JSONB NOT NULL DEFAULT '{}',
    in_sample_start DATE,
    in_sample_end   DATE,
    oos_sharpe      DOUBLE PRECISION,
    oos_max_dd      DOUBLE PRECISION,
    oos_return_pct  DOUBLE PRECISION,
    objective_score DOUBLE PRECISION,
    is_active       BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (strategy_id, version_tag)
);

CREATE INDEX IF NOT EXISTS idx_params_version_strategy ON strategy_params_version(strategy_id);
CREATE INDEX IF NOT EXISTS idx_params_version_active ON strategy_params_version(strategy_id, is_active) WHERE is_active = true;
