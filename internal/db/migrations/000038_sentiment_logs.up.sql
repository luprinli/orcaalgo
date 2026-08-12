-- 000038: Create sentiment_logs table for Fear & Greed index storage
-- Supports RiskPipeline sentiment-dependent position sizing
-- Populated by: Go fixtures (generateSentimentLogs), Alternative.me API backfill

CREATE TABLE IF NOT EXISTS sentiment_logs (
    id          BIGSERIAL PRIMARY KEY,
    timestamp   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    score       INTEGER NOT NULL,
    label       VARCHAR(32) NOT NULL,
    source      TEXT NOT NULL DEFAULT 'synthetic'
);

CREATE INDEX IF NOT EXISTS idx_sentiment_logs_timestamp ON sentiment_logs (timestamp DESC);
