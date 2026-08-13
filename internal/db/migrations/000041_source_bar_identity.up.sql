-- Phase: Make `source` part of the candle bar identity.
--
-- Previously the unique key was (symbol_id, timeframe, time), which forced a
-- single row per bar regardless of provenance. That caused bars from
-- incompatible sources (e.g. legacy `seed` synthetic vs real `stooq`) to
-- overwrite each other and produced ~7-10x price-scale discontinuities.
--
-- The source-less uniqueness was enforced by MULTIPLE objects across the
-- codebase: a table constraint (`candles_symbol_timeframe_time_unique`, from
-- migrations 000010/000027 and stooq_importer.EnsureSchema) and a unique index
-- (`candles_symbol_id_timeframe_time_idx`, from migration 000001). Both must be
-- dropped before re-creating a source-inclusive unique key.

-- Step 1: Drop every source-less uniqueness object.
ALTER TABLE candles DROP CONSTRAINT IF EXISTS candles_symbol_timeframe_time_unique;
DROP INDEX IF EXISTS candles_symbol_id_timeframe_time_idx;
DROP INDEX IF EXISTS candles_time_idx;
DROP INDEX IF EXISTS idx_candles_unique_bar;

-- Step 2: Re-create a single source-inclusive unique index. Existing data has
-- at most one row per (symbol_id, timeframe, time) from the previous
-- constraint, so no deduplication is required before creation.
CREATE UNIQUE INDEX IF NOT EXISTS idx_candles_unique_bar
  ON candles (symbol_id, timeframe, time, source);

-- Step 3: Restore the plain time index (dropped above because it was unique).
CREATE INDEX IF NOT EXISTS candles_time_idx ON candles (time DESC);
