-- 000009_universe.down.sql

ALTER TABLE symbols
    DROP COLUMN IF EXISTS market_cap,
    DROP COLUMN IF EXISTS last_volume,
    DROP COLUMN IF EXISTS last_atr_pct,
    DROP COLUMN IF EXISTS last_rsi,
    DROP COLUMN IF EXISTS metrics_updated;

ALTER TABLE backtest_runs
    DROP COLUMN IF EXISTS universe_snapshot_id,
    DROP COLUMN IF EXISTS universe_config_id,
    DROP COLUMN IF EXISTS use_universe_snapshots;

DROP TABLE IF EXISTS universe_state;
DROP TABLE IF EXISTS universe_config;
