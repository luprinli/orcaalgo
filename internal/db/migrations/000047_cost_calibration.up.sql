-- 000047_cost_calibration.up.sql
-- Per-symbol, per-timeframe transaction-cost calibration coefficients (R2).
-- Calibrated from OHLCV candles via `orca calibrate-costs` and consumed to seed
-- the backtest SlippageModel (SpreadBps / VolumeImpactFactor) instead of the
-- hand-set constants in SlippageForSymbol. Additive; nullable metrics so a
-- partially-identified symbol can still be recorded.
CREATE TABLE IF NOT EXISTS cost_calibration (
    id                 BIGSERIAL PRIMARY KEY,
    symbol_id          BIGINT NOT NULL REFERENCES symbols(id) ON DELETE CASCADE,
    timeframe          TEXT   NOT NULL,
    spread_bps         DOUBLE PRECISION,
    roll_spread_bps    DOUBLE PRECISION,
    impact_eta         DOUBLE PRECISION,
    adverse_select_bps DOUBLE PRECISION,
    estimator          TEXT   NOT NULL DEFAULT 'corwin_schultz',
    calibrated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (symbol_id, timeframe)
);
