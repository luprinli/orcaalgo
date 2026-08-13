-- Phase: corporate-action adjustment plumbing (R4).
--
-- The stooq provider is unadjusted across all timeframes (a single consistent
-- convention after the Yahoo retirement), so cross-provider split mismatches
-- no longer occur. This migration adds the schema required to store corporate
-- actions (splits/dividends) and apply a cumulative adjustment factor on load,
-- replacing the hardcoded identity AdjustmentFactor = 1.0 with a data-backed
-- factor once corporate-action data is ingested.

CREATE TABLE IF NOT EXISTS corporate_actions (
    id             BIGSERIAL PRIMARY KEY,
    symbol_id      INTEGER NOT NULL REFERENCES symbols(id) ON DELETE CASCADE,
    action_date    DATE NOT NULL,
    split_ratio    NUMERIC NOT NULL DEFAULT 1.0,   -- e.g. 0.1 for a 10:1 split
    cash_dividend  NUMERIC NOT NULL DEFAULT 0.0,   -- per-share cash dividend
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (symbol_id, action_date)
);

CREATE INDEX IF NOT EXISTS idx_corporate_actions_symbol_date
    ON corporate_actions (symbol_id, action_date DESC);
