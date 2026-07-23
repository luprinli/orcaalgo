package db

import (
	"context"
	"encoding/json"
	"time"
)

// RunSummaryRecord is the permanent per-matrix-run aggregate that survives row
// pruning (see docs/backtest_retention_policy.md).
type RunSummaryRecord struct {
	BatchRunID      string          `json:"batch_run_id"`
	RunID           *string         `json:"run_id,omitempty"`
	TotalCombos     int             `json:"total_combos"`
	TradedCombos    int             `json:"traded_combos"`
	ZeroTrade       int             `json:"zero_trade"`
	Errored         int             `json:"errored"`
	EffectiveTrials int             `json:"effective_trials"`
	ScoreHistogram  json.RawMessage `json:"score_histogram"`
	Viability       json.RawMessage `json:"viability"`
	FailureReasons  json.RawMessage `json:"failure_reasons"`
	ParetoFront     json.RawMessage `json:"pareto_front"`
	BestSharpe      float64         `json:"best_sharpe"`
	BestCombo       string          `json:"best_combo"`
	EngineVersion   string          `json:"engine_version"`
}

// InsertRunSummary persists the permanent aggregate for a matrix batch.
func (r *Repository) InsertRunSummary(ctx context.Context, s *RunSummaryRecord) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO backtest_run_summaries
		   (batch_run_id, run_id, total_combos, traded_combos, zero_trade, errored,
		    effective_trials, score_histogram, viability, failure_reasons, pareto_front,
		    best_sharpe, best_combo, engine_version)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		s.BatchRunID, s.RunID, s.TotalCombos, s.TradedCombos, s.ZeroTrade, s.Errored,
		s.EffectiveTrials, s.ScoreHistogram, s.Viability, s.FailureReasons, s.ParetoFront,
		s.BestSharpe, s.BestCombo, s.EngineVersion,
	)
	return err
}

// PruneBacktestResults deletes rows of a given retention class older than the
// cutoff. Tier-aware retention (T0 is never pruned) — returns rows removed.
func (r *Repository) PruneBacktestResults(ctx context.Context, retentionClass int, olderThan time.Time) (int64, error) {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM backtest_results WHERE retention_class = $1 AND created_at < $2`,
		retentionClass, olderThan,
	)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
