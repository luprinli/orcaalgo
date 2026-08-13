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
