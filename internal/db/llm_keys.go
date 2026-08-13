package db

import (
	"context"
	"errors"
)

// LLMKey is the non-secret metadata for a per-user LLM provider key. The secret
// itself lives in the encrypted vault (VaultPath); only the masked suffix is
// stored here so the raw key is never persisted or logged.
type LLMKey struct {
	ID           int64  `json:"id"`
	UserID       string `json:"user_id"`
	Provider     string `json:"provider"`
	VaultPath    string `json:"-"`
	BaseURL      string `json:"base_url"`
	Model        string `json:"model"`
	MaskedSuffix string `json:"masked_suffix"`
	CreatedAt    string `json:"created_at,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

// UpsertLLMKey inserts or replaces a user's key for a provider. The
// (user_id, provider) unique constraint makes the write idempotent.
func (r *Repository) UpsertLLMKey(ctx context.Context, k *LLMKey) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO llm_api_keys (user_id, provider, vault_path, base_url, model, masked_suffix)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, provider)
		DO UPDATE SET vault_path = EXCLUDED.vault_path,
		              base_url = EXCLUDED.base_url,
		              model = EXCLUDED.model,
		              masked_suffix = EXCLUDED.masked_suffix,
		              updated_at = now()`,
		k.UserID, k.Provider, k.VaultPath, k.BaseURL, k.Model, k.MaskedSuffix)
	return err
}

// ListLLMKeys returns a user's stored provider keys, ordered by provider.
func (r *Repository) ListLLMKeys(ctx context.Context, userID string) ([]LLMKey, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, provider, vault_path, base_url, model, masked_suffix,
		       created_at::text, updated_at::text
		FROM llm_api_keys
		WHERE user_id = $1
		ORDER BY provider`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LLMKey
	for rows.Next() {
		var k LLMKey
		if err := rows.Scan(&k.ID, &k.UserID, &k.Provider, &k.VaultPath, &k.BaseURL,
			&k.Model, &k.MaskedSuffix, &k.CreatedAt, &k.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// GetLLMKey returns a user's stored key metadata for a provider, or an error
// when none exists.
func (r *Repository) GetLLMKey(ctx context.Context, userID, provider string) (*LLMKey, error) {
	var k LLMKey
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, provider, vault_path, base_url, model, masked_suffix
		FROM llm_api_keys
		WHERE user_id = $1 AND provider = $2`, userID, provider).
		Scan(&k.ID, &k.UserID, &k.Provider, &k.VaultPath, &k.BaseURL, &k.Model, &k.MaskedSuffix)
	if err != nil {
		return nil, err
	}
	return &k, nil
}

// DeleteLLMKey removes a user's key for a provider. Returns an error if none
// existed.
func (r *Repository) DeleteLLMKey(ctx context.Context, userID, provider string) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM llm_api_keys WHERE user_id = $1 AND provider = $2`, userID, provider)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("llm key not found")
	}
	return nil
}
