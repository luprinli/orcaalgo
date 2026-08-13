-- Phase: per-account broker credentials + environment.
--
-- Secrets live in the encrypted vault (under accounts/{id}); this migration
-- stores only the vault path, a masked suffix for display, and the account
-- environment (paper/live) so raw credentials are never persisted or logged.

ALTER TABLE accounts ADD COLUMN environment TEXT NOT NULL DEFAULT 'paper';
ALTER TABLE accounts ADD COLUMN vault_path TEXT NOT NULL DEFAULT '';
ALTER TABLE accounts ADD COLUMN masked_key TEXT NOT NULL DEFAULT '';
