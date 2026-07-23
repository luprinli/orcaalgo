-- 000028: backtest_tasks — durable task queue for matrix backtest batches.
-- Enables resumability across server restarts and bounded retry of failed combos.

CREATE TABLE IF NOT EXISTS backtest_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id VARCHAR(128) NOT NULL,
    seq INT NOT NULL,
    combo_spec JSONB NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    result JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    attempts INT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_backtest_tasks_batch_status ON backtest_tasks(batch_id, status);
CREATE INDEX IF NOT EXISTS idx_backtest_tasks_batch_seq ON backtest_tasks(batch_id, seq);
CREATE INDEX IF NOT EXISTS idx_backtest_tasks_status ON backtest_tasks(status);
