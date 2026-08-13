-- Rollback: collapse source back out of the bar identity.

-- Step 1: Remove the source-inclusive unique index.
DROP INDEX IF EXISTS idx_candles_unique_bar;

-- Step 2: If multiple sources now exist for the same bar, keep the lowest-ctid
-- row (the earliest physical row) before re-adding the source-less constraint.
DELETE FROM candles a
USING candles b
WHERE a.ctid > b.ctid
  AND a.symbol_id = b.symbol_id
  AND a.timeframe = b.timeframe
  AND a.time = b.time;

-- Step 3: Restore the previous source-less uniqueness.
ALTER TABLE candles ADD CONSTRAINT IF NOT EXISTS candles_symbol_timeframe_time_unique
  UNIQUE (symbol_id, timeframe, time);
