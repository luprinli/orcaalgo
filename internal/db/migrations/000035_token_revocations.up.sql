CREATE TABLE IF NOT EXISTS token_revocations (
    token_jti  TEXT PRIMARY KEY,
    revoked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
