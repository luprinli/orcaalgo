package db

import (
	"context"
	"time"
)

type Account struct {
	ID                string                 `json:"id"`
	UserID            string                 `json:"user_id"`
	BrokerType        string                 `json:"broker_type"`
	Name              string                 `json:"name"`
	PropFirmProfileID string                 `json:"prop_firm_profile_id"`
	Balance           float64                `json:"balance"`
	Equity            float64                `json:"equity"`
	DailyPnL          float64                `json:"daily_pnl"`
	HighWaterMark     float64                `json:"high_water_mark"`
	IsDefault         bool                   `json:"is_default"`
	Metadata          map[string]interface{} `json:"metadata"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

type AccountPosition struct {
	ID            int64     `json:"id"`
	AccountID     string    `json:"account_id"`
	Symbol        string    `json:"symbol"`
	Quantity      float64   `json:"quantity"`
	AvgEntryPrice float64   `json:"avg_entry_price"`
	MarketValue   float64   `json:"market_value"`
	UnrealizedPL  float64   `json:"unrealized_pl"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (r *Repository) InsertAccount(ctx context.Context, a *Account) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO accounts (id, user_id, broker_type, name, prop_firm_profile_id, balance, equity, daily_pnl, high_water_mark, is_default, metadata, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, now(), now())
		 ON CONFLICT (id) DO UPDATE SET
		     broker_type=$3, name=$4, prop_firm_profile_id=$5, balance=$6, equity=$7,
		     daily_pnl=$8, high_water_mark=$9, is_default=$10, metadata=$11, updated_at=now()`,
		a.ID, a.UserID, a.BrokerType, a.Name, a.PropFirmProfileID, a.Balance, a.Equity,
		a.DailyPnL, a.HighWaterMark, a.IsDefault, a.Metadata,
	)
	return err
}

func (r *Repository) GetAccount(ctx context.Context, id string) (*Account, error) {
	var a Account
	err := r.pool.QueryRow(ctx,
		`SELECT id, COALESCE(user_id, ''), broker_type, name, prop_firm_profile_id, balance, equity, daily_pnl, high_water_mark, is_default, metadata, created_at, updated_at
		 FROM accounts WHERE id=$1`, id,
	).Scan(&a.ID, &a.UserID, &a.BrokerType, &a.Name, &a.PropFirmProfileID, &a.Balance, &a.Equity,
		&a.DailyPnL, &a.HighWaterMark, &a.IsDefault, &a.Metadata, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *Repository) GetDefaultAccount(ctx context.Context) (*Account, error) {
	var a Account
	err := r.pool.QueryRow(ctx,
		`SELECT id, broker_type, name, prop_firm_profile_id, balance, equity, daily_pnl, high_water_mark, is_default, metadata, created_at, updated_at
		 FROM accounts WHERE is_default=true LIMIT 1`,
	).Scan(&a.ID, &a.BrokerType, &a.Name, &a.PropFirmProfileID, &a.Balance, &a.Equity,
		&a.DailyPnL, &a.HighWaterMark, &a.IsDefault, &a.Metadata, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *Repository) ListAccounts(ctx context.Context) ([]Account, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, COALESCE(user_id, ''), broker_type, name, prop_firm_profile_id, balance, equity, daily_pnl, high_water_mark, is_default, metadata, created_at, updated_at
		 FROM accounts ORDER BY is_default DESC, name ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.ID, &a.UserID, &a.BrokerType, &a.Name, &a.PropFirmProfileID, &a.Balance, &a.Equity,
			&a.DailyPnL, &a.HighWaterMark, &a.IsDefault, &a.Metadata, &a.CreatedAt, &a.UpdatedAt); err != nil {
			continue
		}
		accounts = append(accounts, a)
	}
	return accounts, nil
}

func (r *Repository) ListAccountsByUser(ctx context.Context, userID string) ([]Account, error) {
	query := `SELECT id, COALESCE(user_id, ''), broker_type, name, prop_firm_profile_id, balance, equity, daily_pnl, high_water_mark, is_default, metadata, created_at, updated_at
		 FROM accounts`
	args := []interface{}{}
	if userID != "" {
		query += ` WHERE user_id=$1`
		args = append(args, userID)
	}
	query += ` ORDER BY is_default DESC, name ASC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.ID, &a.UserID, &a.BrokerType, &a.Name, &a.PropFirmProfileID, &a.Balance, &a.Equity,
			&a.DailyPnL, &a.HighWaterMark, &a.IsDefault, &a.Metadata, &a.CreatedAt, &a.UpdatedAt); err != nil {
			continue
		}
		accounts = append(accounts, a)
	}
	return accounts, nil
}

func (r *Repository) ListAccountsByBrokerType(ctx context.Context, brokerType string) ([]Account, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, broker_type, name, prop_firm_profile_id, balance, equity, daily_pnl, high_water_mark, is_default, metadata, created_at, updated_at
		 FROM accounts WHERE broker_type=$1 ORDER BY name ASC`, brokerType,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.ID, &a.BrokerType, &a.Name, &a.PropFirmProfileID, &a.Balance, &a.Equity,
			&a.DailyPnL, &a.HighWaterMark, &a.IsDefault, &a.Metadata, &a.CreatedAt, &a.UpdatedAt); err != nil {
			continue
		}
		accounts = append(accounts, a)
	}
	return accounts, nil
}

func (r *Repository) UpsertAccountBalance(ctx context.Context, id string, balance, equity, dailyPnL, highWaterMark float64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE accounts SET balance=$2, equity=$3, daily_pnl=$4, high_water_mark=$5, updated_at=now() WHERE id=$1`,
		id, balance, equity, dailyPnL, highWaterMark,
	)
	return err
}

func (r *Repository) DeleteAccount(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM accounts WHERE id=$1`, id)
	return err
}

func (r *Repository) UpsertAccountPosition(ctx context.Context, ap *AccountPosition) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO account_positions (account_id, symbol, quantity, avg_entry_price, market_value, unrealized_pl, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, now())
		 ON CONFLICT (account_id, symbol) DO UPDATE SET
		     quantity=$3, avg_entry_price=$4, market_value=$5, unrealized_pl=$6, updated_at=now()`,
		ap.AccountID, ap.Symbol, ap.Quantity, ap.AvgEntryPrice, ap.MarketValue, ap.UnrealizedPL,
	)
	return err
}

func (r *Repository) GetAccountPositions(ctx context.Context, accountID string) ([]AccountPosition, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, account_id, symbol, quantity, avg_entry_price, market_value, unrealized_pl, updated_at
		 FROM account_positions WHERE account_id=$1 AND quantity != 0 ORDER BY symbol`, accountID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []AccountPosition
	for rows.Next() {
		var ap AccountPosition
		if err := rows.Scan(&ap.ID, &ap.AccountID, &ap.Symbol, &ap.Quantity, &ap.AvgEntryPrice,
			&ap.MarketValue, &ap.UnrealizedPL, &ap.UpdatedAt); err != nil {
			continue
		}
		positions = append(positions, ap)
	}
	return positions, nil
}

func (r *Repository) DeleteAccountPosition(ctx context.Context, accountID, symbol string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM account_positions WHERE account_id=$1 AND symbol=$2`,
		accountID, symbol,
	)
	return err
}

func (r *Repository) ClearAccountPositions(ctx context.Context, accountID string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM account_positions WHERE account_id=$1`, accountID,
	)
	return err
}

func (r *Repository) SetDefaultAccount(ctx context.Context, id string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `UPDATE accounts SET is_default=false WHERE is_default=true`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE accounts SET is_default=true WHERE id=$1`, id); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
