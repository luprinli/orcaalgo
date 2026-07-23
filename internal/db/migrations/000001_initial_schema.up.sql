-- Initial schema migration
-- UP

CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE symbols (
    id              SERIAL PRIMARY KEY,
    ticker          VARCHAR(16) NOT NULL,
    exchange        VARCHAR(32) NOT NULL,
    asset_type      VARCHAR(16) NOT NULL DEFAULT 'equity',
    tick_size       NUMERIC(12,8) NOT NULL DEFAULT 0.01,
    lot_size        NUMERIC(12,8) NOT NULL DEFAULT 1,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(ticker, exchange)
);

CREATE TABLE providers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(64) NOT NULL UNIQUE,
    type        VARCHAR(16) NOT NULL,
    driver      VARCHAR(32) NOT NULL,
    is_enabled  BOOLEAN NOT NULL DEFAULT false,
    config      JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE provider_symbols (
    provider_id UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    symbol_id   INT NOT NULL REFERENCES symbols(id) ON DELETE CASCADE,
    feed_type   VARCHAR(16) NOT NULL,
    priority    SMALLINT NOT NULL DEFAULT 100,
    is_enabled  BOOLEAN NOT NULL DEFAULT true,
    PRIMARY KEY (provider_id, symbol_id, feed_type)
);

CREATE TABLE credentials (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id    UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    key_label      VARCHAR(64) NOT NULL,
    vault_path     VARCHAR(256) NOT NULL,
    is_active      BOOLEAN NOT NULL DEFAULT false,
    last_validated TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE market_ticks (
    time         TIMESTAMPTZ NOT NULL,
    symbol_id    INT NOT NULL REFERENCES symbols(id),
    price_raw    BIGINT NOT NULL,
    volume_raw   BIGINT NOT NULL,
    bid_price    BIGINT,
    ask_price    BIGINT,
    bid_size     BIGINT,
    ask_size     BIGINT,
    exchange     VARCHAR(16),
    conditions   VARCHAR(32)
);
SELECT create_hypertable('market_ticks', 'time');
CREATE INDEX ON market_ticks (symbol_id, time DESC);

CREATE TABLE candles (
    time         TIMESTAMPTZ NOT NULL,
    symbol_id    INT NOT NULL REFERENCES symbols(id),
    timeframe    VARCHAR(8) NOT NULL,
    open_raw     BIGINT NOT NULL,
    high_raw     BIGINT NOT NULL,
    low_raw      BIGINT NOT NULL,
    close_raw    BIGINT NOT NULL,
    volume       BIGINT NOT NULL
);
SELECT create_hypertable('candles', 'time');
CREATE UNIQUE INDEX ON candles (symbol_id, timeframe, time DESC);

CREATE TABLE strategies (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(128) NOT NULL UNIQUE,
    type            VARCHAR(64) NOT NULL,
    parameters      JSONB NOT NULL,
    enabled         BOOLEAN DEFAULT false,
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE backtest_runs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id     UUID REFERENCES strategies(id),
    symbol_set      TEXT[] NOT NULL,
    start_date      DATE NOT NULL,
    end_date        DATE NOT NULL,
    initial_capital DECIMAL(18,2) NOT NULL,
    status          VARCHAR(32) DEFAULT 'pending',
    sharpe_ratio    DOUBLE PRECISION,
    max_drawdown    DOUBLE PRECISION,
    total_return    DOUBLE PRECISION,
    win_rate        DOUBLE PRECISION,
    num_trades      INT,
    results_json    JSONB,
    created_at      TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE regime_logs (
    id              BIGSERIAL PRIMARY KEY,
    timestamp       TIMESTAMPTZ NOT NULL DEFAULT now(),
    symbol          VARCHAR(16) NOT NULL,
    hmm_state       SMALLINT NOT NULL,
    confidence      DOUBLE PRECISION NOT NULL,
    vix_contango    DOUBLE PRECISION,
    additional_ctx  JSONB
);
CREATE INDEX ON regime_logs (timestamp DESC);

CREATE TABLE adversarial_news (
    id               BIGSERIAL PRIMARY KEY,
    detected_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    headline         TEXT NOT NULL,
    source           VARCHAR(128) NOT NULL,
    sentiment_score  DOUBLE PRECISION,
    confidence       DOUBLE PRECISION,
    was_corroborated BOOLEAN DEFAULT false,
    symbol_affected  VARCHAR(16)[]
);
CREATE INDEX ON adversarial_news (detected_at DESC);

CREATE TABLE consistency_logs (
    date            DATE PRIMARY KEY,
    daily_pnl_pct   DOUBLE PRECISION NOT NULL,
    is_outlier      BOOLEAN DEFAULT false,
    action_taken    VARCHAR(64),
    created_at      TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE trade_executions (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id            UUID REFERENCES strategies(id),
    symbol                 VARCHAR(16) NOT NULL,
    side                   VARCHAR(4) NOT NULL,
    quantity               DECIMAL(18,6) NOT NULL,
    price                  DECIMAL(18,6) NOT NULL,
    hmm_regime             SMALLINT,
    risk_approved          BOOLEAN DEFAULT true,
    consistency_multiplier DOUBLE PRECISION DEFAULT 1.0,
    rejected_reason        VARCHAR(256),
    executed_at            TIMESTAMPTZ DEFAULT now(),
    broker_order_id        VARCHAR(128)
);
SELECT create_hypertable('trade_executions', 'executed_at');

SELECT add_compression_policy('market_ticks', INTERVAL '7 days');
SELECT add_retention_policy('market_ticks', INTERVAL '30 days');
