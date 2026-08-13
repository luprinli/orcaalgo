-- Phase: BYOK (bring-your-own-key) LLM API key management.
--
-- Per-user LLM provider keys. Secrets live in the encrypted vault (under
-- llm/{user_id}/{provider}); this table stores only non-secret metadata
-- (provider, base URL, model, masked suffix) so keys are never persisted or
-- logged in plaintext. One key per (user, provider); upsert replaces it.

CREATE TABLE IF NOT EXISTS llm_api_keys (
    id            BIGSERIAL PRIMARY KEY,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider      TEXT NOT NULL,
    vault_path    TEXT NOT NULL,
    base_url      TEXT NOT NULL DEFAULT '',
    model         TEXT NOT NULL DEFAULT '',
    masked_suffix TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, provider)
);

CREATE INDEX IF NOT EXISTS idx_llm_api_keys_user ON llm_api_keys (user_id);
