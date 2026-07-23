-- UP: Multi-account support for broker abstraction
CREATE TABLE accounts (
    id                  VARCHAR(64) PRIMARY KEY,
    broker_type         VARCHAR(32) NOT NULL,
    name                VARCHAR(128) NOT NULL,
    prop_firm_profile_id VARCHAR(64),
    balance             DECIMAL(18,6) NOT NULL DEFAULT 0,
    equity              DECIMAL(18,6) NOT NULL DEFAULT 0,
    daily_pnl           DECIMAL(18,6) NOT NULL DEFAULT 0,
    high_water_mark     DECIMAL(18,6) NOT NULL DEFAULT 0,
    is_default          BOOLEAN NOT NULL DEFAULT false,
    metadata            JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE account_positions (
    id              BIGSERIAL PRIMARY KEY,
    account_id      VARCHAR(64) NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    symbol          VARCHAR(16) NOT NULL,
    quantity        DECIMAL(18,6) NOT NULL DEFAULT 0,
    avg_entry_price DECIMAL(18,6) NOT NULL DEFAULT 0,
    market_value    DECIMAL(18,6) NOT NULL DEFAULT 0,
    unrealized_pl   DECIMAL(18,6) NOT NULL DEFAULT 0,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(account_id, symbol)
);

CREATE INDEX ON accounts (broker_type);
CREATE INDEX ON account_positions (account_id);
