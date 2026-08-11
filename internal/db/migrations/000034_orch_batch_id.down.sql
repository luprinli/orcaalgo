DROP INDEX IF EXISTS idx_orch_runs_batch;
ALTER TABLE orchestration_runs DROP COLUMN IF EXISTS batch_id;
