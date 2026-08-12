package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// RevokeToken inserts a JWT ID (jti) into the token_revocations table.
// If the token is already revoked, the INSERT is silently ignored via
// ON CONFLICT DO NOTHING.
func (r *Repository) RevokeToken(ctx context.Context, jti string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO token_revocations (token_jti) VALUES ($1)
		 ON CONFLICT (token_jti) DO NOTHING`,
		jti,
	)
	return err
}

// IsTokenRevoked returns true if the JWT ID is present in the
// token_revocations table, indicating the token has been revoked.
func (r *Repository) IsTokenRevoked(ctx context.Context, jti string) (bool, error) {
	var revokedAt interface{}
	err := r.pool.QueryRow(ctx,
		`SELECT revoked_at FROM token_revocations WHERE token_jti = $1`,
		jti,
	).Scan(&revokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
