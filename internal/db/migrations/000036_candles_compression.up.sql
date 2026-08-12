SELECT add_compression_policy('candles', INTERVAL '30 days', if_not_exists => true);
SELECT add_retention_policy('candles', INTERVAL '2 years', if_not_exists => true);
