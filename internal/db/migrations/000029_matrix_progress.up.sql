-- 000029: matrix_progress — durable matrix backtest progress tracking.
-- Enables progress recovery across server restarts via write-through caching.

CREATE TABLE IF NOT EXISTS matrix_progress (
    batch_id        TEXT PRIMARY KEY,
    mode            TEXT NOT NULL DEFAULT 'matrix',
    total           INTEGER NOT NULL DEFAULT 0,
    completed       INTEGER NOT NULL DEFAULT 0,
    failed          INTEGER NOT NULL DEFAULT 0,
    running         INTEGER NOT NULL DEFAULT 0,
    passed          INTEGER NOT NULL DEFAULT 0,
    status          TEXT NOT NULL DEFAULT 'running',
    start_time      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    combos_json     JSONB,
    results_json    JSONB,
    best_sharpe     DOUBLE PRECISION NOT NULL DEFAULT 0,
    best_strategy   TEXT NOT NULL DEFAULT '',
    best_symbol     TEXT NOT NULL DEFAULT '',
    total_trades    INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_matrix_progress_status ON matrix_progress(status);
