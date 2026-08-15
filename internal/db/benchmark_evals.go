package db

import (
	"context"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

// BenchmarkEval is a market-based benchmark filter evaluation (Phase 1). It is
// append-only: each evaluation is a distinct historical event with its own
// evaluated_at, so the promotion gate and the quarterly post-mortem can audit
// why a strategy was promoted or retired. Metric fields are nullable so a
// partially-identified evaluation can still be recorded.
type BenchmarkEval struct {
	StrategyID           string
	BenchmarkSpecHash    string
	BenchmarkKind        string
	BenchmarkSymbols     string
	WindowStart          time.Time
	WindowEnd            time.Time
	InformationRatio     *float64
	AlphaAnnualized      *float64
	Beta                 *float64
	DeflatedActiveSharpe *float64
	NTrials              *int
	Passed               bool
}

// BenchmarkEvalView is a persisted evaluation with its id and timestamp,
// suitable for API/UI display.
type BenchmarkEvalView struct {
	ID                   int64     `json:"id"`
	StrategyID           string    `json:"strategy_id"`
	BenchmarkSpecHash    string    `json:"benchmark_spec_hash"`
	BenchmarkKind        string    `json:"benchmark_kind"`
	BenchmarkSymbols     string    `json:"benchmark_symbols"`
	WindowStart          time.Time `json:"window_start"`
	WindowEnd            time.Time `json:"window_end"`
	InformationRatio     *float64  `json:"information_ratio"`
	AlphaAnnualized      *float64  `json:"alpha_annualized"`
	Beta                 *float64  `json:"beta"`
	DeflatedActiveSharpe *float64  `json:"deflated_active_sharpe"`
	NTrials              *int      `json:"n_trials"`
	Passed               bool      `json:"passed"`
	EvaluatedAt          time.Time `json:"evaluated_at"`
}

// InsertBenchmarkEval appends a benchmark evaluation row.
func (r *Repository) InsertBenchmarkEval(ctx context.Context, eval BenchmarkEval) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO benchmark_evals
			(strategy_id, benchmark_spec_hash, benchmark_kind, benchmark_symbols,
			 window_start, window_end, information_ratio, alpha_annualized, beta,
			 deflated_active_sharpe, n_trials, passed)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		eval.StrategyID, eval.BenchmarkSpecHash, eval.BenchmarkKind, eval.BenchmarkSymbols,
		eval.WindowStart, eval.WindowEnd, eval.InformationRatio, eval.AlphaAnnualized, eval.Beta,
		eval.DeflatedActiveSharpe, eval.NTrials, eval.Passed,
	)
	return err
}

// ListBenchmarkEvals returns evaluation rows, optionally filtered by strategy,
// newest first.
func (r *Repository) ListBenchmarkEvals(ctx context.Context, strategyID string, limit int) ([]BenchmarkEvalView, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `
		SELECT id, strategy_id, benchmark_spec_hash, benchmark_kind, benchmark_symbols,
		       window_start, window_end, information_ratio, alpha_annualized, beta,
		       deflated_active_sharpe, n_trials, passed, evaluated_at
		FROM benchmark_evals`
	args := []any{}
	if strategyID != "" {
		query += ` WHERE strategy_id = $1`
		args = append(args, strategyID)
	}
	query += ` ORDER BY evaluated_at DESC LIMIT $` + strconv.Itoa(len(args)+1)
	args = append(args, limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []BenchmarkEvalView
	for rows.Next() {
		var v BenchmarkEvalView
		if err := rows.Scan(&v.ID, &v.StrategyID, &v.BenchmarkSpecHash, &v.BenchmarkKind, &v.BenchmarkSymbols,
			&v.WindowStart, &v.WindowEnd, &v.InformationRatio, &v.AlphaAnnualized, &v.Beta,
			&v.DeflatedActiveSharpe, &v.NTrials, &v.Passed, &v.EvaluatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// GetLatestBenchmarkEval returns the most recent evaluation for a strategy and
// benchmark spec, or (nil, nil) when none exists.
func (r *Repository) GetLatestBenchmarkEval(ctx context.Context, strategyID, specHash string) (*BenchmarkEvalView, error) {
	var v BenchmarkEvalView
	err := r.pool.QueryRow(ctx, `
		SELECT id, strategy_id, benchmark_spec_hash, benchmark_kind, benchmark_symbols,
		       window_start, window_end, information_ratio, alpha_annualized, beta,
		       deflated_active_sharpe, n_trials, passed, evaluated_at
		FROM benchmark_evals
		WHERE strategy_id = $1 AND benchmark_spec_hash = $2
		ORDER BY evaluated_at DESC LIMIT 1`, strategyID, specHash,
	).Scan(&v.ID, &v.StrategyID, &v.BenchmarkSpecHash, &v.BenchmarkKind, &v.BenchmarkSymbols,
		&v.WindowStart, &v.WindowEnd, &v.InformationRatio, &v.AlphaAnnualized, &v.Beta,
		&v.DeflatedActiveSharpe, &v.NTrials, &v.Passed, &v.EvaluatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}
