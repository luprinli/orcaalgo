-- 000049_benchmark_filter.up.sql
-- Market-based benchmark filter (Phase 1) — two tables:
--   benchmark_evals  : append-only verdicts (promotion/post-mortem audit trail)
--   benchmark_series : named input series (risk-free yield curve for `risk_free`)

CREATE TABLE IF NOT EXISTS benchmark_evals (
    id                     BIGSERIAL PRIMARY KEY,
    strategy_id            TEXT NOT NULL,
    benchmark_spec_hash    TEXT NOT NULL,
    benchmark_kind         TEXT NOT NULL,
    benchmark_symbols      TEXT,
    window_start           TIMESTAMPTZ NOT NULL,
    window_end             TIMESTAMPTZ NOT NULL,
    information_ratio      DOUBLE PRECISION,
    alpha_annualized       DOUBLE PRECISION,
    beta                   DOUBLE PRECISION,
    deflated_active_sharpe DOUBLE PRECISION,
    n_trials               INT,
    passed                 BOOLEAN NOT NULL,
    evaluated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_benchmark_evals_strategy
    ON benchmark_evals (strategy_id, evaluated_at DESC);

-- `value` for risk_free is the fractional annualized yield (e.g. 0.052 = 5.2%).
CREATE TABLE IF NOT EXISTS benchmark_series (
    id        BIGSERIAL PRIMARY KEY,
    name      TEXT NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    value     DOUBLE PRECISION NOT NULL,
    source    TEXT,
    UNIQUE (name, timestamp)
);
