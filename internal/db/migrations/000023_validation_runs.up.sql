CREATE TABLE IF NOT EXISTS validation_runs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id          VARCHAR(64) NOT NULL UNIQUE,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    progress        INTEGER NOT NULL DEFAULT 0,
    current_stage   VARCHAR(30) NOT NULL DEFAULT 'preflight',
    strategy_ids    TEXT[] NOT NULL DEFAULT '{}',
    symbols         TEXT[] NOT NULL DEFAULT '{}',
    start_date      TIMESTAMPTZ,
    end_date        TIMESTAMPTZ,
    initial_capital DOUBLE PRECISION NOT NULL DEFAULT 100000,
    result_json     JSONB,
    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_validation_runs_job_id ON validation_runs (job_id);
CREATE INDEX IF NOT EXISTS idx_validation_runs_status ON validation_runs (status);
CREATE INDEX IF NOT EXISTS idx_validation_runs_created ON validation_runs (created_at DESC);
