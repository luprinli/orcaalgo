-- 000046_signal_funnel.up.sql
-- Add a dedicated signal-funnel column to backtest_results so the per-row
-- signal-gating telemetry (attempts -> passed -> rejections by reason) is
-- queryable without unpacking the metrics JSONB. Additive; existing rows are
-- NULL and can be backfilled from metrics when needed.
ALTER TABLE backtest_results ADD COLUMN IF NOT EXISTS signal_funnel JSONB;
