package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
)

// ParamVersion represents a versioned set of strategy parameters from an
// optimization run, stored in strategy_params_version.
type ParamVersion struct {
	ID             string             `json:"id"`
	StrategyID     string             `json:"strategy_id"`
	VersionTag     string             `json:"version_tag"`
	Params         map[string]float64 `json:"params"`
	InSampleStart  *time.Time         `json:"in_sample_start,omitempty"`
	InSampleEnd    *time.Time         `json:"in_sample_end,omitempty"`
	OOSSharpe      *float64           `json:"oos_sharpe,omitempty"`
	OOSMaxDD       *float64           `json:"oos_max_dd,omitempty"`
	OOSReturnPct   *float64           `json:"oos_return_pct,omitempty"`
	ObjectiveScore *float64           `json:"objective_score,omitempty"`
	IsActive       bool               `json:"is_active"`
	CreatedAt      time.Time          `json:"created_at"`
}

// SaveParamVersion upserts a parameter version. If a version with the same
// (strategy_id, version_tag) already exists, its params and metrics are updated.
func (r *Repository) SaveParamVersion(ctx context.Context, pv *ParamVersion) error {
	paramsJSON, err := json.Marshal(pv.Params)
	if err != nil {
		return err
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO strategy_params_version
			(strategy_id, version_tag, params, in_sample_start, in_sample_end,
			 oos_sharpe, oos_max_dd, oos_return_pct, objective_score, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (strategy_id, version_tag) DO UPDATE SET
			params = EXCLUDED.params,
			in_sample_start = EXCLUDED.in_sample_start,
			in_sample_end = EXCLUDED.in_sample_end,
			oos_sharpe = EXCLUDED.oos_sharpe,
			oos_max_dd = EXCLUDED.oos_max_dd,
			oos_return_pct = EXCLUDED.oos_return_pct,
			objective_score = EXCLUDED.objective_score,
			is_active = EXCLUDED.is_active
	`, pv.StrategyID, pv.VersionTag, paramsJSON,
		pv.InSampleStart, pv.InSampleEnd,
		pv.OOSSharpe, pv.OOSMaxDD, pv.OOSReturnPct,
		pv.ObjectiveScore, pv.IsActive)
	return err
}

// GetActiveParams returns the currently active parameter set for a strategy.
func (r *Repository) GetActiveParams(ctx context.Context, strategyID string) (*ParamVersion, error) {
	pv, err := r.queryParamVersion(ctx, `
		SELECT id, strategy_id, version_tag, params, in_sample_start, in_sample_end,
		       oos_sharpe, oos_max_dd, oos_return_pct, objective_score, is_active, created_at
		FROM strategy_params_version
		WHERE strategy_id = $1 AND is_active = true
		ORDER BY created_at DESC LIMIT 1
	`, strategyID)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return pv, err
}

// GetParamVersion returns a specific parameter version by strategy and tag.
func (r *Repository) GetParamVersion(ctx context.Context, strategyID, versionTag string) (*ParamVersion, error) {
	pv, err := r.queryParamVersion(ctx, `
		SELECT id, strategy_id, version_tag, params, in_sample_start, in_sample_end,
		       oos_sharpe, oos_max_dd, oos_return_pct, objective_score, is_active, created_at
		FROM strategy_params_version
		WHERE strategy_id = $1 AND version_tag = $2
	`, strategyID, versionTag)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return pv, err
}

// ListParamVersions returns all parameter versions for a strategy, newest first.
func (r *Repository) ListParamVersions(ctx context.Context, strategyID string, limit int) ([]ParamVersion, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, strategy_id, version_tag, params, in_sample_start, in_sample_end,
		       oos_sharpe, oos_max_dd, oos_return_pct, objective_score, is_active, created_at
		FROM strategy_params_version
		WHERE strategy_id = $1
		ORDER BY created_at DESC LIMIT $2
	`, strategyID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []ParamVersion
	for rows.Next() {
		pv, err := scanParamVersion(rows)
		if err != nil {
			return nil, err
		}
		versions = append(versions, *pv)
	}
	return versions, rows.Err()
}

// ActivateParams sets the given version as the active one for its strategy,
// deactivating all other versions. Pass versionID as empty string to
// deactivate all versions without activating a new one.
func (r *Repository) ActivateParams(ctx context.Context, strategyID, versionTag string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE strategy_params_version SET is_active = false
		WHERE strategy_id = $1 AND is_active = true
	`, strategyID)
	if err != nil {
		return err
	}

	if versionTag != "" {
		_, err = tx.Exec(ctx, `
			UPDATE strategy_params_version SET is_active = true
			WHERE strategy_id = $1 AND version_tag = $2
		`, strategyID, versionTag)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// DeactivateAllParams deactivates all parameter versions for a strategy
// (used when reverting to registry defaults).
func (r *Repository) DeactivateAllParams(ctx context.Context, strategyID string) error {
	return r.ActivateParams(ctx, strategyID, "")
}

func (r *Repository) queryParamVersion(ctx context.Context, query string, args ...interface{}) (*ParamVersion, error) {
	row := r.pool.QueryRow(ctx, query, args...)
	return scanParamVersion(row)
}

func scanParamVersion(row pgx.Row) (*ParamVersion, error) {
	var pv ParamVersion
	var paramsJSON []byte
	err := row.Scan(&pv.ID, &pv.StrategyID, &pv.VersionTag, &paramsJSON,
		&pv.InSampleStart, &pv.InSampleEnd,
		&pv.OOSSharpe, &pv.OOSMaxDD, &pv.OOSReturnPct,
		&pv.ObjectiveScore, &pv.IsActive, &pv.CreatedAt)
	if err != nil {
		return nil, err
	}
	if len(paramsJSON) > 0 {
		json.Unmarshal(paramsJSON, &pv.Params)
	}
	if pv.Params == nil {
		pv.Params = make(map[string]float64)
	}
	return &pv, nil
}
