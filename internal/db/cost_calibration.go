package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// CostCalibration is a per-symbol, per-timeframe set of calibrated transaction
// cost coefficients (R2). Spread and impact fields are nullable so a
// partially identified symbol can still be recorded.
type CostCalibration struct {
	SymbolID         int64
	Timeframe        string
	SpreadBps        *float64
	RollSpreadBps    *float64
	ImpactEta        *float64
	AdverseSelectBps *float64
	Estimator        string
}

// CostCalibrationView is a cost calibration row with its resolved ticker,
// suitable for API/UI display.
type CostCalibrationView struct {
	Ticker           string    `json:"ticker"`
	Timeframe        string    `json:"timeframe"`
	SpreadBps        *float64  `json:"spread_bps"`
	RollSpreadBps    *float64  `json:"roll_spread_bps"`
	ImpactEta        *float64  `json:"impact_eta"`
	AdverseSelectBps *float64  `json:"adverse_select_bps"`
	Estimator        string    `json:"estimator"`
	CalibratedAt     time.Time `json:"calibrated_at"`
}

// UpsertCostCalibration inserts or updates cost-calibration coefficients for a
// symbol. The (symbol_id, timeframe) unique key makes the write idempotent and
// refreshes calibrated_at on update.
func (r *Repository) UpsertCostCalibration(ctx context.Context, symbol string, cc CostCalibration) error {
	var symbolID int64
	if err := r.pool.QueryRow(ctx, `SELECT id FROM symbols WHERE ticker = $1`, symbol).Scan(&symbolID); err != nil {
		return err
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO cost_calibration
			(symbol_id, timeframe, spread_bps, roll_spread_bps, impact_eta, adverse_select_bps, estimator)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (symbol_id, timeframe) DO UPDATE SET
			spread_bps = EXCLUDED.spread_bps,
			roll_spread_bps = EXCLUDED.roll_spread_bps,
			impact_eta = EXCLUDED.impact_eta,
			adverse_select_bps = EXCLUDED.adverse_select_bps,
			estimator = EXCLUDED.estimator,
			calibrated_at = now()`,
		symbolID, cc.Timeframe, cc.SpreadBps, cc.RollSpreadBps, cc.ImpactEta, cc.AdverseSelectBps, cc.Estimator,
	)
	return err
}

// ListCostCalibration returns cost calibrations for a symbol (or all symbols
// when `symbol` is empty) as display views, ordered by ticker then timeframe.
func (r *Repository) ListCostCalibration(ctx context.Context, symbol string) ([]CostCalibrationView, error) {
	query := `
		SELECT s.ticker, cc.timeframe, cc.spread_bps, cc.roll_spread_bps,
		       cc.impact_eta, cc.adverse_select_bps, cc.estimator, cc.calibrated_at
		FROM cost_calibration cc
		JOIN symbols s ON cc.symbol_id = s.id`
	args := []any{}
	if symbol != "" {
		query += ` WHERE s.ticker = $1`
		args = append(args, symbol)
	}
	query += ` ORDER BY s.ticker ASC, cc.timeframe ASC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CostCalibrationView
	for rows.Next() {
		var v CostCalibrationView
		if err := rows.Scan(&v.Ticker, &v.Timeframe, &v.SpreadBps, &v.RollSpreadBps, &v.ImpactEta, &v.AdverseSelectBps, &v.Estimator, &v.CalibratedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// GetCostCalibrationForSymbol returns the calibration for one symbol+timeframe,
// or (nil, nil) when none is recorded.
func (r *Repository) GetCostCalibrationForSymbol(ctx context.Context, symbol, timeframe string) (*CostCalibrationView, error) {
	var v CostCalibrationView
	err := r.pool.QueryRow(ctx, `
		SELECT s.ticker, cc.timeframe, cc.spread_bps, cc.roll_spread_bps,
		       cc.impact_eta, cc.adverse_select_bps, cc.estimator, cc.calibrated_at
		FROM cost_calibration cc
		JOIN symbols s ON cc.symbol_id = s.id
		WHERE s.ticker = $1 AND cc.timeframe = $2`, symbol, timeframe,
	).Scan(&v.Ticker, &v.Timeframe, &v.SpreadBps, &v.RollSpreadBps, &v.ImpactEta, &v.AdverseSelectBps, &v.Estimator, &v.CalibratedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}
