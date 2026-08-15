-- 000048_symbol_carry.up.sql
-- Add interest_rate to symbols so FX-carry strategies can compute interest-rate
-- differentials from live data (currently the carry runner uses a static in-code
-- map; this column is the future live-data source). Additive and nullable.
ALTER TABLE symbols ADD COLUMN IF NOT EXISTS interest_rate DOUBLE PRECISION;
