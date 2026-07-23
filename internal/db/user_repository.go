package db

import (
	"context"
	"time"
)

type DBUser struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Roles        []string  `json:"roles"`
	IsVerified   bool      `json:"is_verified"`
	IsActive     bool      `json:"is_active"`
	TOTPSecret   string    `json:"-"`
	TOTPEnabled  bool      `json:"totp_enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type PasswordResetToken struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `json:"used"`
	CreatedAt time.Time `json:"created_at"`
}

type EmailVerificationToken struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `json:"used"`
	CreatedAt time.Time `json:"created_at"`
}

type NotificationSettings struct {
	UserID    string                 `json:"user_id"`
	Settings  map[string]interface{} `json:"settings"`
	UpdatedAt time.Time              `json:"updated_at"`
}

func (r *Repository) CreateUser(ctx context.Context, u *DBUser) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, username, email, password_hash, roles, is_verified, is_active, totp_secret, totp_enabled)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		u.ID, u.Username, u.Email, u.PasswordHash, u.Roles, u.IsVerified, u.IsActive, u.TOTPSecret, u.TOTPEnabled,
	)
	return err
}

func (r *Repository) GetUserByID(ctx context.Context, id string) (*DBUser, error) {
	var u DBUser
	err := r.pool.QueryRow(ctx,
		`SELECT id, username, email, password_hash, roles, is_verified, is_active, totp_secret, totp_enabled, created_at, updated_at
		 FROM users WHERE id=$1`, id,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Roles, &u.IsVerified, &u.IsActive,
		&u.TOTPSecret, &u.TOTPEnabled, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) GetUserByUsername(ctx context.Context, username string) (*DBUser, error) {
	var u DBUser
	err := r.pool.QueryRow(ctx,
		`SELECT id, username, email, password_hash, roles, is_verified, is_active, totp_secret, totp_enabled, created_at, updated_at
		 FROM users WHERE username=$1`, username,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Roles, &u.IsVerified, &u.IsActive,
		&u.TOTPSecret, &u.TOTPEnabled, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*DBUser, error) {
	var u DBUser
	err := r.pool.QueryRow(ctx,
		`SELECT id, username, email, password_hash, roles, is_verified, is_active, totp_secret, totp_enabled, created_at, updated_at
		 FROM users WHERE email=$1`, email,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Roles, &u.IsVerified, &u.IsActive,
		&u.TOTPSecret, &u.TOTPEnabled, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) UpdateUserPassword(ctx context.Context, id, passwordHash string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET password_hash=$2, updated_at=now() WHERE id=$1`, id, passwordHash,
	)
	return err
}

func (r *Repository) UpdateUserTOTP(ctx context.Context, id, secret string, enabled bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET totp_secret=$2, totp_enabled=$3, updated_at=now() WHERE id=$1`, id, secret, enabled,
	)
	return err
}

func (r *Repository) MarkUserVerified(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET is_verified=true, updated_at=now() WHERE id=$1`, id,
	)
	return err
}

func (r *Repository) ListUsers(ctx context.Context) ([]DBUser, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, username, email, password_hash, roles, is_verified, is_active, totp_secret, totp_enabled, created_at, updated_at
		 FROM users ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []DBUser
	for rows.Next() {
		var u DBUser
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Roles, &u.IsVerified, &u.IsActive,
			&u.TOTPSecret, &u.TOTPEnabled, &u.CreatedAt, &u.UpdatedAt); err != nil {
			continue
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *Repository) CreatePasswordResetToken(ctx context.Context, t *PasswordResetToken) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO password_reset_tokens (id, user_id, token, expires_at)
		 VALUES ($1, $2, $3, $4)`,
		t.ID, t.UserID, t.Token, t.ExpiresAt,
	)
	return err
}

func (r *Repository) GetPasswordResetToken(ctx context.Context, token string) (*PasswordResetToken, error) {
	var t PasswordResetToken
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, token, expires_at, used, created_at
		 FROM password_reset_tokens WHERE token=$1`, token,
	).Scan(&t.ID, &t.UserID, &t.Token, &t.ExpiresAt, &t.Used, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repository) MarkResetTokenUsed(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE password_reset_tokens SET used=true WHERE id=$1`, id,
	)
	return err
}

func (r *Repository) CreateEmailVerificationToken(ctx context.Context, t *EmailVerificationToken) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO email_verification_tokens (id, user_id, token, expires_at)
		 VALUES ($1, $2, $3, $4)`,
		t.ID, t.UserID, t.Token, t.ExpiresAt,
	)
	return err
}

func (r *Repository) GetEmailVerificationToken(ctx context.Context, token string) (*EmailVerificationToken, error) {
	var t EmailVerificationToken
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, token, expires_at, used, created_at
		 FROM email_verification_tokens WHERE token=$1`, token,
	).Scan(&t.ID, &t.UserID, &t.Token, &t.ExpiresAt, &t.Used, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repository) MarkVerificationTokenUsed(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE email_verification_tokens SET used=true WHERE id=$1`, id,
	)
	return err
}

func (r *Repository) UpsertNotificationSettings(ctx context.Context, userID string, settings map[string]interface{}) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO notification_settings (user_id, settings, updated_at) VALUES ($1, $2, now())
		 ON CONFLICT (user_id) DO UPDATE SET settings=$2, updated_at=now()`,
		userID, settings,
	)
	return err
}

func (r *Repository) GetNotificationSettings(ctx context.Context, userID string) (*NotificationSettings, error) {
	var ns NotificationSettings
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, settings, updated_at FROM notification_settings WHERE user_id=$1`, userID,
	).Scan(&ns.UserID, &ns.Settings, &ns.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &ns, nil
}

func (r *Repository) DeleteNotificationSettings(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM notification_settings WHERE user_id=$1`, userID)
	return err
}

func (r *Repository) UserCount(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}
