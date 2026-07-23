-- 000009_universe.up.sql
-- Dynamic Universe Selection schema

CREATE TABLE universe_config (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    profile_id          VARCHAR(32) NOT NULL DEFAULT 'default',
    asset_class_filters JSONB NOT NULL DEFAULT '{}',
    dynamic_triggers    JSONB NOT NULL DEFAULT '{}',
    content_hash        TEXT NOT NULL,
    is_active           BOOLEAN NOT NULL DEFAULT false,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_id, name)
);

CREATE INDEX idx_universe_config_active ON universe_config (user_id, is_active);

CREATE TABLE universe_state (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    snapshot_date       DATE NOT NULL,
    symbol_ids          INTEGER[] NOT NULL,
    content_hash        TEXT NOT NULL,
    filters_used        JSONB NOT NULL DEFAULT '{}',
    triggered_additions JSONB NOT NULL DEFAULT '[]',
    triggered_removals  JSONB NOT NULL DEFAULT '[]',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_id, snapshot_date)
);

CREATE INDEX idx_universe_state_date ON universe_state (user_id, snapshot_date DESC);

ALTER TABLE backtest_runs
    ADD COLUMN IF NOT EXISTS universe_snapshot_id UUID REFERENCES universe_state(id),
    ADD COLUMN IF NOT EXISTS universe_config_id   UUID REFERENCES universe_config(id),
    ADD COLUMN IF NOT EXISTS use_universe_snapshots BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE symbols
    ADD COLUMN IF NOT EXISTS market_cap       BIGINT,
    ADD COLUMN IF NOT EXISTS last_volume      BIGINT,
    ADD COLUMN IF NOT EXISTS last_atr_pct     DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS last_rsi         DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS metrics_updated  TIMESTAMPTZ;
