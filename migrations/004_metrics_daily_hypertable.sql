-- metrics_daily hypertable: stores daily-aggregated Prometheus metrics for
-- long-term retention beyond Prometheus's 15-day window.
--
-- This table is intended to be populated by a cron job (orca export-metrics)
-- that queries the Prometheus API at the end of each UTC day and inserts
-- aggregated values (min, max, avg, p95) for each tracked metric.

CREATE TABLE IF NOT EXISTS metrics_daily (
    day          DATE         NOT NULL,
    metric_name  TEXT         NOT NULL,
    labels       JSONB        DEFAULT '{}',
    value_min    DOUBLE PRECISION,
    value_max    DOUBLE PRECISION,
    value_avg    DOUBLE PRECISION,
    value_p95    DOUBLE PRECISION,
    sample_count INTEGER      DEFAULT 1,
    recorded_at  TIMESTAMPTZ  DEFAULT NOW(),

    PRIMARY KEY (day, metric_name, labels)
);

-- Convert to TimescaleDB hypertable for automatic partitioning by day.
SELECT create_hypertable('metrics_daily', 'day', if_not_exists => TRUE);

-- Enable compression after 7 days to save storage.
ALTER TABLE metrics_daily SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'metric_name',
    timescaledb.compress_orderby = 'day DESC, labels'
);

SELECT add_compression_policy('metrics_daily', INTERVAL '7 days', if_not_exists => TRUE);

-- Tracked metrics (subset of the 19 Prometheus metrics that have business value
-- for long-term analysis):
--
--   orca_daily_pnl_pct              – daily P&L trend
--   orca_strategy_sharpe            – rolling Sharpe per strategy
--   orca_backtest_duration_seconds  – backtest performance over time
--   orca_matrix_combos_completed_total – throughput trend
--   orca_reject_count_total         – signal quality trend
--   orca_propfirm_breach_total      – breach events over time
--   orca_kill_switch_active         – kill switch history
--   orca_regime_state               – regime distribution over time
--   orca_ws_connections             – user activity trend
--   orca_engine_latency_us          – latency trend

-- Example query: average daily P&L for the last 90 days
-- SELECT day, value_avg
-- FROM metrics_daily
-- WHERE metric_name = 'orca_daily_pnl_pct'
--   AND day >= NOW() - INTERVAL '90 days'
-- ORDER BY day;
