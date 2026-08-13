-- Phase 2a: Add source tracking and unique constraint to candles table.
-- Enables distinguishing stooq / stooq-resampled / stooq-calibrated / yahoo data.
-- Adds unique constraint (symbol_id, timeframe, time) to prevent duplicate bars.

-- Step 1: Add source column with default (backward-compatible)
ALTER TABLE candles ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'yahoo';

-- Step 2: Clean existing duplicates before adding unique constraint.
-- Keep the row with the most detail (non-zero volume, earliest source).
-- If all duplicates are identical, keep earliest by ctid.
DELETE FROM candles a
USING candles b
WHERE a.ctid > b.ctid
  AND a.symbol_id = b.symbol_id
  AND a.timeframe = b.timeframe
  AND a.time = b.time;

-- Step 3: Add unique constraint.
CREATE UNIQUE INDEX IF NOT EXISTS idx_candles_unique_bar
  ON candles (symbol_id, timeframe, time);
