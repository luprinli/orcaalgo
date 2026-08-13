package db

import (
	"context"
	"time"
)

// CorporateAction is a split or cash-dividend event for a symbol.
type CorporateAction struct {
	SymbolID     int64
	ActionDate   time.Time
	SplitRatio   float64
	CashDividend float64
}

// LoadCorporateActions returns all corporate actions for the given symbol,
// ordered by action date ascending. Returns an empty slice (not an error) when
// none are recorded — adjustment is then the identity factor 1.0.
func (r *Repository) LoadCorporateActions(ctx context.Context, symbol string) ([]CorporateAction, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT ca.symbol_id, ca.action_date, ca.split_ratio, ca.cash_dividend
		FROM corporate_actions ca
		JOIN symbols s ON ca.symbol_id = s.id
		WHERE s.ticker = $1
		ORDER BY ca.action_date ASC
	`, symbol)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []CorporateAction
	for rows.Next() {
		var a CorporateAction
		if err := rows.Scan(&a.SymbolID, &a.ActionDate, &a.SplitRatio, &a.CashDividend); err != nil {
			return nil, err
		}
		actions = append(actions, a)
	}
	return actions, nil
}

// CorporateActionView is a corporate action with its resolved ticker, suitable
// for API/UI display.
type CorporateActionView struct {
	Ticker       string    `json:"ticker"`
	ActionDate   time.Time `json:"action_date"`
	SplitRatio   float64   `json:"split_ratio"`
	CashDividend float64   `json:"cash_dividend"`
}

// UpsertCorporateAction inserts or updates a corporate action for a symbol. The
// symbol is resolved to its id; an unknown symbol returns an error. The
// (symbol_id, action_date) unique key makes the write idempotent.
func (r *Repository) UpsertCorporateAction(ctx context.Context, symbol string, a CorporateAction) error {
	var symbolID int64
	if err := r.pool.QueryRow(ctx,
		`SELECT id FROM symbols WHERE ticker = $1`, symbol,
	).Scan(&symbolID); err != nil {
		return err
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO corporate_actions (symbol_id, action_date, split_ratio, cash_dividend)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (symbol_id, action_date)
		DO UPDATE SET split_ratio = EXCLUDED.split_ratio, cash_dividend = EXCLUDED.cash_dividend`,
		symbolID, a.ActionDate, a.SplitRatio, a.CashDividend,
	)
	return err
}

// UpsertCorporateActionsBatch upserts multiple actions for one symbol,
// reusing UpsertCorporateAction (idempotent per (symbol_id, action_date)).
func (r *Repository) UpsertCorporateActionsBatch(ctx context.Context, symbol string, actions []CorporateAction) error {
	for _, a := range actions {
		if err := r.UpsertCorporateAction(ctx, symbol, a); err != nil {
			return err
		}
	}
	return nil
}

// ListCorporateActions returns corporate actions for a symbol (or all symbols
// when `symbol` is empty), ordered by action date ascending, as display views.
func (r *Repository) ListCorporateActions(ctx context.Context, symbol string) ([]CorporateActionView, error) {
	query := `
		SELECT s.ticker, ca.action_date, ca.split_ratio, ca.cash_dividend
		FROM corporate_actions ca
		JOIN symbols s ON ca.symbol_id = s.id`
	args := []any{}
	if symbol != "" {
		query += ` WHERE s.ticker = $1`
		args = append(args, symbol)
	}
	query += ` ORDER BY ca.action_date ASC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CorporateActionView
	for rows.Next() {
		var v CorporateActionView
		if err := rows.Scan(&v.Ticker, &v.ActionDate, &v.SplitRatio, &v.CashDividend); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// ApplyCorporateActions populates each candle's AdjustmentFactor from the
// cumulative split ratio of actions on/after the bar's date. Bars BEFORE a
// split are multiplied by the split ratio so their raw prices are comparable to
// post-split bars (and vice-versa). Prices are left raw: the engine applies
// the factor at fill time. With no actions, every factor is the identity 1.0.
func ApplyCorporateActions(candles []Candle, actions []CorporateAction) []Candle {
	out := make([]Candle, len(candles))
	copy(out, candles)
	for i := range out {
		factor := 1.0
		for _, a := range actions {
			if !a.ActionDate.Before(out[i].Time) {
				factor *= a.SplitRatio
			}
		}
		out[i].AdjustmentFactor = factor
	}
	return out
}
