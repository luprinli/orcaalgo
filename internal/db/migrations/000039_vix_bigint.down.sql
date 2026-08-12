-- Rollback: restore DOUBLE PRECISION columns from BIGINT.
-- Idempotent: checks column type before execution.

DO $$
BEGIN
  IF (SELECT data_type FROM information_schema.columns
      WHERE table_name = 'vix_logs' AND column_name = 'vix_value') = 'bigint' THEN

    ALTER TABLE vix_logs ADD COLUMN vix_value_fp  DOUBLE PRECISION;
    ALTER TABLE vix_logs ADD COLUMN vix_change_fp DOUBLE PRECISION;

    UPDATE vix_logs SET
        vix_value_fp  = (vix_value)::DOUBLE PRECISION / 10000.0,
        vix_change_fp = (vix_change)::DOUBLE PRECISION / 10000.0;

    ALTER TABLE vix_logs DROP COLUMN vix_value;
    ALTER TABLE vix_logs DROP COLUMN vix_change;

    ALTER TABLE vix_logs RENAME COLUMN vix_value_fp  TO vix_value;
    ALTER TABLE vix_logs RENAME COLUMN vix_change_fp TO vix_change;

  END IF;
END $$;
