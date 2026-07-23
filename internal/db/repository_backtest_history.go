package db

import (
	"context"
	"encoding/json"
	"time"
)

type BacktestRunRecord struct {
	ID               string          `json:"id"`
	StrategyID       string          `json:"strategy_id"`
	RunType          string          `json:"run_type"`
	Status           string          `json:"status"`
	StrategyIDs      []string        `json:"strategy_ids"`
	Symbols          []string        `json:"symbols"`
	StartDate        *time.Time      `json:"start_date,omitempty"`
	EndDate          *time.Time      `json:"end_date,omitempty"`
	InitialCapital   float64         `json:"initial_capital"`
	Config           json.RawMessage `json:"config,omitempty"`
	SharpeRatio      float64         `json:"sharpe_ratio"`
	MaxDrawdown      float64         `json:"max_drawdown"`
	TotalReturn      float64         `json:"total_return"`
	WinRate          float64         `json:"win_rate"`
	NumTrades        int             `json:"num_trades"`
	ResultsJSON      json.RawMessage `json:"results_json,omitempty"`
	ErrorMessage     *string         `json:"error_message,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	CompletedAt      *time.Time      `json:"completed_at,omitempty"`
}

type BacktestResultRecord struct {
	ID            string          `json:"id"`
	RunID         string          `json:"run_id"`
	StrategyID    string          `json:"strategy_id"`
	ResultType    string          `json:"result_type"`
	TrialIndex    int             `json:"trial_index"`
	SchemaVersion int             `json:"schema_version"`
	Parameters    json.RawMessage `json:"parameters,omitempty"`
	Metrics       json.RawMessage `json:"metrics"`
	EquityCurve   json.RawMessage `json:"equity_curve,omitempty"`
	Trades        json.RawMessage `json:"trades,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

func (r *Repository) CreateBacktestRun(ctx context.Context, br *BacktestRunRecord) error {
	sids, _ := json.Marshal(br.StrategyIDs)
	syms, _ := json.Marshal(br.Symbols)
	return r.pool.QueryRow(ctx,
		`INSERT INTO backtest_runs (strategy_id, run_type, status, strategy_ids, symbols, start_date, end_date, initial_capital, config, sharpe_ratio, max_drawdown, total_return, win_rate, num_trades, results_json)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		 RETURNING id`,
		br.StrategyID, br.RunType, br.Status, sids, syms,
		br.StartDate, br.EndDate, br.InitialCapital, br.Config,
		br.SharpeRatio, br.MaxDrawdown, br.TotalReturn, br.WinRate, br.NumTrades, br.ResultsJSON,
	).Scan(&br.ID)
}

func (r *Repository) UpdateBacktestRunStatus(ctx context.Context, id string, status string, errMsg *string, completedAt *time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE backtest_runs SET status=$1, error_message=$2, completed_at=$3, updated_at=now() WHERE id=$4`,
		status, errMsg, completedAt, id,
	)
	return err
}

func (r *Repository) UpdateBacktestRunMetrics(ctx context.Context, id string, sharpe, maxDD, totalReturn, winRate float64, numTrades int, resultsJSON json.RawMessage) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE backtest_runs SET sharpe_ratio=$1, max_drawdown=$2, total_return=$3, win_rate=$4, num_trades=$5, results_json=$6, updated_at=now() WHERE id=$7`,
		sharpe, maxDD, totalReturn, winRate, numTrades, resultsJSON, id,
	)
	return err
}

func (r *Repository) GetBacktestRun(ctx context.Context, id string) (*BacktestRunRecord, error) {
	br := &BacktestRunRecord{}
	var sIDs, syms json.RawMessage
	var sd, ed *time.Time
	var config, results []byte
	var errMsg *string

	err := r.pool.QueryRow(ctx,
		`SELECT id, strategy_id, run_type, status, strategy_ids, symbols, start_date, end_date, initial_capital, coalesce(config::text,'{}')::jsonb, sharpe_ratio, max_drawdown, total_return, win_rate, num_trades, results_json, error_message, created_at, updated_at, completed_at
		 FROM backtest_runs WHERE id=$1`, id,
	).Scan(&br.ID, &br.StrategyID, &br.RunType, &br.Status, &sIDs, &syms, &sd, &ed, &br.InitialCapital,
		&config, &br.SharpeRatio, &br.MaxDrawdown, &br.TotalReturn, &br.WinRate, &br.NumTrades, &results, &errMsg,
		&br.CreatedAt, &br.UpdatedAt, &br.CompletedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(sIDs, &br.StrategyIDs)
	json.Unmarshal(syms, &br.Symbols)
	br.StartDate = sd
	br.EndDate = ed
	if config != nil { br.Config = config }
	if results != nil { br.ResultsJSON = results }
	br.ErrorMessage = errMsg
	return br, nil
}

func (r *Repository) ListBacktestRuns(ctx context.Context, limit int, runType string) ([]*BacktestRunRecord, error) {
	if limit <= 0 { limit = 50 }
	query := `SELECT id, strategy_id, run_type, status, strategy_ids, symbols, start_date, end_date, initial_capital, coalesce(config::text,'{}')::jsonb, sharpe_ratio, max_drawdown, total_return, win_rate, num_trades, results_json, error_message, created_at, updated_at, completed_at FROM backtest_runs`
	args := []interface{}{limit}
	if runType != "" {
		query += ` WHERE run_type=$1 ORDER BY created_at DESC LIMIT $2`
		args = []interface{}{runType, limit}
	} else {
		query += ` ORDER BY created_at DESC LIMIT $1`
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil { return nil, err }
	defer rows.Close()

	var runs []*BacktestRunRecord
	for rows.Next() {
		br := &BacktestRunRecord{}
		var sIDs, syms json.RawMessage
		var sd, ed *time.Time
		var config, results []byte
		var errMsg *string
		if err := rows.Scan(&br.ID, &br.StrategyID, &br.RunType, &br.Status, &sIDs, &syms, &sd, &ed, &br.InitialCapital, &config, &br.SharpeRatio, &br.MaxDrawdown, &br.TotalReturn, &br.WinRate, &br.NumTrades, &results, &errMsg, &br.CreatedAt, &br.UpdatedAt, &br.CompletedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(sIDs, &br.StrategyIDs)
		json.Unmarshal(syms, &br.Symbols)
		br.StartDate = sd; br.EndDate = ed
		if config != nil { br.Config = config }
		if results != nil { br.ResultsJSON = results }
		br.ErrorMessage = errMsg
		runs = append(runs, br)
	}
	return runs, rows.Err()
}

func (r *Repository) DeleteBacktestRun(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM backtest_runs WHERE id=$1`, id)
	return err
}

func (r *Repository) InsertBacktestResult(ctx context.Context, btr *BacktestResultRecord) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO backtest_results (run_id, strategy_id, result_type, trial_index, parameters, metrics, equity_curve, trades)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		btr.RunID, btr.StrategyID, btr.ResultType, btr.TrialIndex, btr.Parameters, btr.Metrics, btr.EquityCurve, btr.Trades,
	)
	return err
}

func (r *Repository) GetBacktestResults(ctx context.Context, runID string) ([]*BacktestResultRecord, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, run_id, strategy_id, result_type, trial_index, parameters, metrics, equity_curve, trades, created_at
		 FROM backtest_results WHERE run_id=$1 ORDER BY trial_index`, runID)
	if err != nil { return nil, err }
	defer rows.Close()

	var results []*BacktestResultRecord
	for rows.Next() {
		rec := &BacktestResultRecord{}
		if err := rows.Scan(&rec.ID, &rec.RunID, &rec.StrategyID, &rec.ResultType, &rec.TrialIndex, &rec.Parameters, &rec.Metrics, &rec.EquityCurve, &rec.Trades, &rec.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, rec)
	}
	return results, rows.Err()
}

func cnd(runType string) string {
	if runType != "" { return " WHERE run_type='" + runType + "'" }
	return ""
}
