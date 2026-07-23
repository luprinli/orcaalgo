-- Fix backtest_runs to match the repository write model (single + matrix persistence).
-- The original table (000001) declared strategy_id as a UUID FK and symbol_set as
-- NOT NULL, but the engine persists strategy *names* and a JSON symbols array via
-- CreateBacktestRun — causing every persist to fail with
--   column "symbols" of relation "backtest_runs" does not exist.
-- This aligns the schema with the code (all changes idempotent + backward-read safe).

-- Drop the strategy_id FK (name may vary) so we can widen the column to hold names.
DO $$
DECLARE c text;
BEGIN
    SELECT conname INTO c
      FROM pg_constraint
     WHERE conrelid = 'backtest_runs'::regclass
       AND contype = 'f'
       AND conname LIKE '%strategy_id%';
    IF c IS NOT NULL THEN
        EXECUTE 'ALTER TABLE backtest_runs DROP CONSTRAINT ' || quote_ident(c);
    END IF;
END $$;

ALTER TABLE backtest_runs ALTER COLUMN strategy_id TYPE VARCHAR(64) USING strategy_id::text;
ALTER TABLE backtest_runs ALTER COLUMN strategy_id DROP NOT NULL;

-- The engine does not populate symbol_set / start_date / end_date for every run
-- (e.g. error records); make them nullable so inserts never NOT-NULL-violate.
ALTER TABLE backtest_runs ALTER COLUMN symbol_set DROP NOT NULL;
ALTER TABLE backtest_runs ALTER COLUMN start_date DROP NOT NULL;
ALTER TABLE backtest_runs ALTER COLUMN end_date  DROP NOT NULL;

-- Columns the repository writes but the schema lacked.
ALTER TABLE backtest_runs ADD COLUMN IF NOT EXISTS strategy_ids JSONB;
ALTER TABLE backtest_runs ADD COLUMN IF NOT EXISTS symbols      JSONB;
ALTER TABLE backtest_runs ADD COLUMN IF NOT EXISTS config       JSONB;

-- Some databases already had strategy_ids / symbols as TEXT[] from an earlier
-- partial state; the repository writes JSON, so convert any non-JSONB variant to
-- JSONB (to_jsonb handles both text[] arrays and scalar text).
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
                WHERE table_name='backtest_runs' AND column_name='strategy_ids' AND data_type <> 'jsonb') THEN
        ALTER TABLE backtest_runs ALTER COLUMN strategy_ids DROP DEFAULT;
        ALTER TABLE backtest_runs ALTER COLUMN strategy_ids TYPE JSONB USING to_jsonb(strategy_ids);
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.columns
                WHERE table_name='backtest_runs' AND column_name='symbols' AND data_type <> 'jsonb') THEN
        ALTER TABLE backtest_runs ALTER COLUMN symbols DROP DEFAULT;
        ALTER TABLE backtest_runs ALTER COLUMN symbols TYPE JSONB USING to_jsonb(symbols);
    END IF;
END $$;
