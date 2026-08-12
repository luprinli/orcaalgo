-- VIX DOUBLE PRECISION to BIGINT migration (HP #2 compliance)
-- Scale factor: 10000 (VIX 10-80 range → 100000-800000 BIGINT, 4 decimal places)
-- VIX is an informational signal metric; migrating to fixed-point aligns with HP #2.
-- Idempotent: checks column type before execution; safe to run multiple times.

DO $$
BEGIN
  IF (SELECT data_type FROM information_schema.columns
      WHERE table_name = 'vix_logs' AND column_name = 'vix_value') = 'double precision' THEN

    ALTER TABLE vix_logs ADD COLUMN vix_value_int  BIGINT;
    ALTER TABLE vix_logs ADD COLUMN vix_change_int BIGINT;

    UPDATE vix_logs SET
        vix_value_int  = ROUND((vix_value)::NUMERIC * 10000),
        vix_change_int = ROUND((vix_change)::NUMERIC * 10000);

    ALTER TABLE vix_logs ALTER COLUMN vix_value_int  SET NOT NULL;
    ALTER TABLE vix_logs ALTER COLUMN vix_change_int SET NOT NULL;

    ALTER TABLE vix_logs DROP COLUMN vix_value;
    ALTER TABLE vix_logs DROP COLUMN vix_change;

    ALTER TABLE vix_logs RENAME COLUMN vix_value_int  TO vix_value;
    ALTER TABLE vix_logs RENAME COLUMN vix_change_int TO vix_change;

  END IF;
END $$;
