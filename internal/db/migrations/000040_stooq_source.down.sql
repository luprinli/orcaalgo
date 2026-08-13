-- Rollback: Remove unique constraint and source column.
DROP INDEX IF EXISTS idx_candles_unique_bar;
ALTER TABLE candles DROP COLUMN IF EXISTS source;
