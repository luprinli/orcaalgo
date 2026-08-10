package db

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type OrchestrationRun struct {
	ID            string     `json:"id"`
	CreatedAt     time.Time  `json:"created_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	Status        string     `json:"status"`
	StartDate     time.Time  `json:"start_date"`
	EndDate       time.Time  `json:"end_date"`
	InitialCapital float64   `json:"initial_capital"`
	StrategyIDs   []string   `json:"strategy_ids"`
	SymbolTFPairs []string   `json:"symbol_tf_pairs"`
	PoolSharpe    *float64   `json:"pool_sharpe,omitempty"`
	PoolSortino   *float64   `json:"pool_sortino,omitempty"`
	PoolMaxDD     *float64   `json:"pool_maxdd,omitempty"`
	PoolReturnPct *float64   `json:"pool_return_pct,omitempty"`
	RebalanceCosts *float64  `json:"rebalance_costs,omitempty"`
	ResultJSON    json.RawMessage `json:"result_json,omitempty"`
}

type AllocationEntry struct {
	ID               int64     `json:"id"`
	RunID            string    `json:"run_id"`
	BarTime          time.Time `json:"bar_time"`
	StrategyID       string    `json:"strategy_id"`
	Weight           float64   `json:"weight"`
	AllocatedCapital float64   `json:"allocated_capital"`
	PositionSize     *float64  `json:"position_size,omitempty"`
	IsActive         bool      `json:"is_active"`
}

type StrategyStatus struct {
	StrategyID         string     `json:"strategy_id"`
	Status             string     `json:"status"`
	AllocationPct      float64    `json:"allocation_pct"`
	TrailingSharpe     *float64   `json:"trailing_sharpe,omitempty"`
	TrailingSortino    *float64   `json:"trailing_sortino,omitempty"`
	TrailingMaxDD      *float64   `json:"trailing_maxdd,omitempty"`
	LastSignalAt       *time.Time `json:"last_signal_at,omitempty"`
	ActiveSince        *time.Time `json:"active_since,omitempty"`
	DemotedAt          *time.Time `json:"demoted_at,omitempty"`
	DemotionReason     *string    `json:"demotion_reason,omitempty"`
	OrchestrationRunID *string    `json:"orchestration_run_id,omitempty"`
	LastEvaluated      time.Time  `json:"last_evaluated"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

func (r *Repository) SaveOrchestrationRun(ctx context.Context, run *OrchestrationRun) error {
	sIDs := stringsToPgArray(run.StrategyIDs)
	tfs := stringsToPgArray(run.SymbolTFPairs)

	return r.pool.QueryRow(ctx,
		`INSERT INTO orchestration_runs (start_date, end_date, initial_capital, strategy_ids, symbol_tf_pairs)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at`,
		run.StartDate, run.EndDate, run.InitialCapital, sIDs, tfs,
	).Scan(&run.ID, &run.CreatedAt)
}

func (r *Repository) UpdateOrchestrationRun(ctx context.Context, id string, status string, result *OrchestrationResult) error {
	return r.UpdateOrchestrationRunWithJSON(ctx, id, status, result, nil)
}

func (r *Repository) UpdateOrchestrationRunWithJSON(ctx context.Context, id string, status string, result *OrchestrationResult, resultJSON json.RawMessage) error {
	if resultJSON == nil && result != nil {
		resultJSON, _ = json.Marshal(result)
	}
	var poolSharpe, poolSortino, poolMaxDD, poolReturnPct, rebalanceCosts *float64
	if result != nil {
		poolSharpe = floatPtr(result.PoolSharpe)
		poolSortino = floatPtr(result.PoolSortino)
		poolMaxDD = floatPtr(result.PoolMaxDD)
		poolReturnPct = floatPtr(result.PoolReturnPct)
		rebalanceCosts = floatPtr(result.RebalanceCosts)
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE orchestration_runs
		 SET status=$1, completed_at=NOW(), pool_sharpe=$2, pool_sortino=$3, pool_maxdd=$4,
		     pool_return_pct=$5, rebalance_costs=$6, result_json=$7
		 WHERE id=$8`,
		status,
		poolSharpe,
		poolSortino,
		poolMaxDD,
		poolReturnPct,
		rebalanceCosts,
		resultJSON,
		id,
	)
	return err
}

func floatPtr(f float64) *float64 {
	if f == 0 || math.IsNaN(f) || math.IsInf(f, 0) {
		return nil
	}
	return &f
}

func (r *Repository) LoadOrchestrationRun(ctx context.Context, id string) (*OrchestrationRun, error) {
	run := &OrchestrationRun{}
	var sIDs, tfs string
	var completedAt *time.Time

	err := r.pool.QueryRow(ctx,
		`SELECT id, created_at, completed_at, status, start_date, end_date, initial_capital,
		        strategy_ids::text, symbol_tf_pairs::text, pool_sharpe, pool_sortino, pool_maxdd,
		        pool_return_pct, rebalance_costs, result_json
		 FROM orchestration_runs WHERE id=$1`, id,
	).Scan(&run.ID, &run.CreatedAt, &completedAt, &run.Status,
		&run.StartDate, &run.EndDate, &run.InitialCapital,
		&sIDs, &tfs, &run.PoolSharpe, &run.PoolSortino, &run.PoolMaxDD,
		&run.PoolReturnPct, &run.RebalanceCosts, &run.ResultJSON)
	if err != nil {
		return nil, err
	}
	run.CompletedAt = completedAt
	run.StrategyIDs = pgArrayToStrings(sIDs)
	run.SymbolTFPairs = pgArrayToStrings(tfs)
	return run, nil
}

func (r *Repository) ListOrchestrationRuns(ctx context.Context, limit, offset int) ([]OrchestrationRun, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM orchestration_runs`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, created_at, completed_at, status, start_date, end_date, initial_capital,
		        strategy_ids::text, symbol_tf_pairs::text, pool_sharpe, pool_sortino, pool_maxdd,
		        pool_return_pct, rebalance_costs, result_json
		 FROM orchestration_runs ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var runs []OrchestrationRun
	for rows.Next() {
		var run OrchestrationRun
		var sIDs, tfs string
		var completedAt *time.Time
		if err := rows.Scan(&run.ID, &run.CreatedAt, &completedAt, &run.Status,
			&run.StartDate, &run.EndDate, &run.InitialCapital,
			&sIDs, &tfs, &run.PoolSharpe, &run.PoolSortino, &run.PoolMaxDD,
			&run.PoolReturnPct, &run.RebalanceCosts, &run.ResultJSON); err != nil {
			continue
		}
		run.CompletedAt = completedAt
		run.StrategyIDs = pgArrayToStrings(sIDs)
		run.SymbolTFPairs = pgArrayToStrings(tfs)
		runs = append(runs, run)
	}
	return runs, total, nil
}

func (r *Repository) SaveAllocationHistory(ctx context.Context, runID string, entries []AllocationEntry) error {
	batch := &pgx.Batch{}
	for i := range entries {
		e := &entries[i]
		batch.Queue(
			`INSERT INTO allocation_history (run_id, bar_time, strategy_id, weight, allocated_capital, position_size, is_active)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			runID, e.BarTime, e.StrategyID, e.Weight, e.AllocatedCapital, e.PositionSize, e.IsActive,
		)
	}
	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()
	for range entries {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) LoadAllocationHistory(ctx context.Context, runID string) ([]AllocationEntry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, run_id, bar_time, strategy_id, weight, allocated_capital, position_size, is_active
		 FROM allocation_history WHERE run_id=$1 ORDER BY bar_time`, runID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []AllocationEntry
	for rows.Next() {
		var e AllocationEntry
		if err := rows.Scan(&e.ID, &e.RunID, &e.BarTime, &e.StrategyID, &e.Weight, &e.AllocatedCapital, &e.PositionSize, &e.IsActive); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (r *Repository) UpsertStrategyStatus(ctx context.Context, status *StrategyStatus) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO strategy_status (strategy_id, status, allocation_pct, trailing_sharpe, trailing_sortino,
		     trailing_maxdd, last_signal_at, active_since, demoted_at, demotion_reason, orchestration_run_id, last_evaluated, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
		 ON CONFLICT (strategy_id) DO UPDATE SET
		     status=$2, allocation_pct=$3, trailing_sharpe=$4, trailing_sortino=$5,
		     trailing_maxdd=$6, last_signal_at=$7, active_since=$8, demoted_at=$9,
		     demotion_reason=$10, orchestration_run_id=$11, last_evaluated=$12, updated_at=NOW()`,
		status.StrategyID, status.Status, status.AllocationPct,
		status.TrailingSharpe, status.TrailingSortino,
		status.TrailingMaxDD, status.LastSignalAt, status.ActiveSince,
		status.DemotedAt, status.DemotionReason, status.OrchestrationRunID, status.LastEvaluated,
	)
	return err
}

func (r *Repository) GetStrategyStatus(ctx context.Context, strategyID string) (*StrategyStatus, error) {
	s := &StrategyStatus{}
	err := r.pool.QueryRow(ctx,
		`SELECT strategy_id, status, allocation_pct, trailing_sharpe, trailing_sortino,
		        trailing_maxdd, last_signal_at, active_since, demoted_at, demotion_reason, orchestration_run_id, last_evaluated, updated_at
		 FROM strategy_status WHERE strategy_id=$1`, strategyID,
	).Scan(&s.StrategyID, &s.Status, &s.AllocationPct,
		&s.TrailingSharpe, &s.TrailingSortino,
		&s.TrailingMaxDD, &s.LastSignalAt, &s.ActiveSince,
		&s.DemotedAt, &s.DemotionReason, &s.OrchestrationRunID, &s.LastEvaluated, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *Repository) ListStrategyStatuses(ctx context.Context) ([]StrategyStatus, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT strategy_id, status, allocation_pct, trailing_sharpe, trailing_sortino,
		        trailing_maxdd, last_signal_at, active_since, demoted_at, demotion_reason, orchestration_run_id, last_evaluated, updated_at
		 FROM strategy_status ORDER BY strategy_id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var statuses []StrategyStatus
	for rows.Next() {
		var s StrategyStatus
		if err := rows.Scan(&s.StrategyID, &s.Status, &s.AllocationPct,
			&s.TrailingSharpe, &s.TrailingSortino,
			&s.TrailingMaxDD, &s.LastSignalAt, &s.ActiveSince,
			&s.DemotedAt, &s.DemotionReason, &s.OrchestrationRunID, &s.LastEvaluated, &s.UpdatedAt); err != nil {
			continue
		}
		statuses = append(statuses, s)
	}
	return statuses, nil
}

type OrchestrationResult struct {
	PoolSharpe     float64 `json:"pool_sharpe"`
	PoolSortino    float64 `json:"pool_sortino"`
	PoolMaxDD      float64 `json:"pool_maxdd"`
	PoolReturnPct  float64 `json:"pool_return_pct"`
	RebalanceCosts float64 `json:"rebalance_costs"`
}

func stringsToPgArray(items []string) string {
	escaped := make([]string, len(items))
	for i, s := range items {
		escaped[i] = `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
	}
	return "{" + strings.Join(escaped, ",") + "}"
}

func pgArrayToStrings(raw string) []string {
	if raw == "" || raw == "{}" {
		return nil
	}
	inner := raw[1 : len(raw)-1]
	if inner == "" {
		return nil
	}
	parts := strings.Split(inner, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.Trim(p, `"`)
		result = append(result, p)
	}
	return result
}
