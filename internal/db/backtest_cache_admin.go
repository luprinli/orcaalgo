package db

import (
	"context"
	"time"
)

// Backtest cache administration: portable export/import of cached backtest
// result rows and age-based pruning. The "cache" is the backtest_results table
// (parameterised result rows) plus its parent backtest_runs.

// ExportBacktestCache returns every cached backtest result row, for portable
// SQL/JSON transfer between environments.
func (r *Repository) ExportBacktestCache(ctx context.Context) ([]BacktestResultRecord, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, run_id, strategy_id, result_type, trial_index, schema_version,
		       parameters, metrics, equity_curve, trades, created_at
		FROM backtest_results ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []BacktestResultRecord
	for rows.Next() {
		var rec BacktestResultRecord
		if err := rows.Scan(&rec.ID, &rec.RunID, &rec.StrategyID, &rec.ResultType,
			&rec.TrialIndex, &rec.SchemaVersion, &rec.Parameters, &rec.Metrics,
			&rec.EquityCurve, &rec.Trades, &rec.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

// ImportBacktestCache inserts cached result rows, skipping rows whose id already
// exists (idempotent). Returns the number of newly inserted rows.
func (r *Repository) ImportBacktestCache(ctx context.Context, results []BacktestResultRecord) (int, error) {
	inserted := 0
	for _, rec := range results {
		tag, err := r.pool.Exec(ctx, `
			INSERT INTO backtest_results (id, run_id, strategy_id, result_type, trial_index,
			                              schema_version, parameters, metrics, equity_curve, trades)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			ON CONFLICT (id) DO NOTHING`,
			rec.ID, rec.RunID, rec.StrategyID, rec.ResultType, rec.TrialIndex,
			rec.SchemaVersion, rec.Parameters, rec.Metrics, rec.EquityCurve, rec.Trades)
		if err != nil {
			return inserted, err
		}
		inserted += int(tag.RowsAffected())
	}
	return inserted, nil
}

// PruneBacktestCache deletes cached backtest result rows older than `olderThan`
// and returns the number of rows removed. Parent run rows are left to the
// retention policy.
func (r *Repository) PruneBacktestCache(ctx context.Context, olderThan time.Time) (int64, error) {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM backtest_results WHERE created_at < $1`, olderThan)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
