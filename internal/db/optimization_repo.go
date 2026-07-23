package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type OptimizationRun struct {
	ID              uuid.UUID              `json:"id"`
	BacktestRunID   *uuid.UUID             `json:"backtest_run_id,omitempty"`
	Method          string                 `json:"method"`
	ObjectiveMetric string                 `json:"objective_metric"`
	TotalTrials     int                    `json:"total_trials"`
	BestParams      map[string]float64     `json:"best_params"`
	BestMetric      *float64               `json:"best_metric,omitempty"`
	ParamRanges     map[string][]float64   `json:"param_ranges,omitempty"`
	TrainStart      *time.Time             `json:"train_start,omitempty"`
	TrainEnd        *time.Time             `json:"train_end,omitempty"`
	TestStart       *time.Time             `json:"test_start,omitempty"`
	TestEnd         *time.Time             `json:"test_end,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
}

func (r *Repository) SaveOptimizationRun(ctx context.Context, run *OptimizationRun) error {
	query := `INSERT INTO optimization_runs 
		(id, backtest_run_id, method, objective_metric, total_trials, best_params, best_metric, param_ranges,
		 train_start, train_end, test_start, test_end, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`
	_, err := r.pool.Exec(ctx, query,
		run.ID, run.BacktestRunID, run.Method, run.ObjectiveMetric,
		run.TotalTrials, run.BestParams, run.BestMetric, run.ParamRanges,
		run.TrainStart, run.TrainEnd, run.TestStart, run.TestEnd, run.CreatedAt)
	return err
}

func (r *Repository) GetOptimizationRunByID(ctx context.Context, id uuid.UUID) (*OptimizationRun, error) {
	query := `SELECT id, backtest_run_id, method, objective_metric, total_trials, best_params, best_metric, 
	          param_ranges, train_start, train_end, test_start, test_end, created_at
	          FROM optimization_runs WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, id)
	return scanOptimizationRun(row)
}

func (r *Repository) GetOptimizationRunByBacktestID(ctx context.Context, backtestRunID uuid.UUID) (*OptimizationRun, error) {
	query := `SELECT id, backtest_run_id, method, objective_metric, total_trials, best_params, best_metric,
	          param_ranges, train_start, train_end, test_start, test_end, created_at
	          FROM optimization_runs WHERE backtest_run_id = $1`
	row := r.pool.QueryRow(ctx, query, backtestRunID)
	return scanOptimizationRun(row)
}

func (r *Repository) ListOptimizationRuns(ctx context.Context, limit, offset int) ([]*OptimizationRun, error) {
	query := `SELECT id, backtest_run_id, method, objective_metric, total_trials, best_params, best_metric,
	          param_ranges, train_start, train_end, test_start, test_end, created_at
	          FROM optimization_runs ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var runs []*OptimizationRun
	for rows.Next() {
		run, err := scanOptimizationRun(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, nil
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanOptimizationRun(row rowScanner) (*OptimizationRun, error) {
	var run OptimizationRun
	err := row.Scan(
		&run.ID, &run.BacktestRunID, &run.Method, &run.ObjectiveMetric,
		&run.TotalTrials, &run.BestParams, &run.BestMetric, &run.ParamRanges,
		&run.TrainStart, &run.TrainEnd, &run.TestStart, &run.TestEnd, &run.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &run, err
}
