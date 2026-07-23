-- Migration: Add regime_label to candles for regime-aware synthetic data
-- UP

ALTER TABLE candles ADD COLUMN IF NOT EXISTS regime_label INTEGER;
ALTER TABLE candles ADD COLUMN IF NOT EXISTS data_source TEXT DEFAULT 'real'
    CHECK (data_source IN ('real', 'synthetic'));
ALTER TABLE candles ADD COLUMN IF NOT EXISTS generation_id TEXT;

CREATE INDEX IF NOT EXISTS idx_candles_regime ON candles (data_source, generation_id, regime_label);

CREATE TABLE IF NOT EXISTS synthetic_generations (
    id TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    config JSONB NOT NULL,
    symbol TEXT NOT NULL,
    timeframe TEXT NOT NULL,
    n_candles INTEGER,
    n_trading_days INTEGER,
    regime_sequence_id TEXT,
    validation_status TEXT DEFAULT 'pending'
        CHECK (validation_status IN ('pending', 'passed', 'failed')),
    validation_report JSONB,
    regime_transition JSONB,
    model TEXT NOT NULL DEFAULT 'heston',
    seed INTEGER
);

CREATE INDEX IF NOT EXISTS idx_synthetic_gen_symbol ON synthetic_generations (symbol, created_at DESC);
