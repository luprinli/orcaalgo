ALTER TABLE orchestration_runs ADD COLUMN IF NOT EXISTS batch_id TEXT;
CREATE INDEX IF NOT EXISTS idx_orch_runs_batch ON orchestration_runs (batch_id) WHERE batch_id IS NOT NULL;
