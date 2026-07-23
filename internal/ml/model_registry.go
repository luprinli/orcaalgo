package ml

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ModelRecord struct {
	ModelHash   string  `json:"model_hash"`
	ModelType   string  `json:"model_type"`
	ModelName   string  `json:"model_name"`
	BrierScore  float64 `json:"brier_score"`
	ROCAUC      float64 `json:"roc_auc"`
	Metadata    string  `json:"metadata_json"`
	CreatedAt   string  `json:"created_at"`
}

func RegisterModel(ctx context.Context, pool *pgxpool.Pool, r ModelRecord) error {
	_, err := pool.Exec(ctx,
		`INSERT INTO ml_models (model_hash, model_type, model_name, brier_score, roc_auc, metadata_json)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (model_hash) DO UPDATE SET metadata_json = $6`,
		r.ModelHash, r.ModelType, r.ModelName, r.BrierScore, r.ROCAUC, r.Metadata,
	)
	if err != nil {
		return fmt.Errorf("model registry insert: %w", err)
	}
	return nil
}

func GetModel(ctx context.Context, pool *pgxpool.Pool, modelHash string) (*ModelRecord, error) {
	var r ModelRecord
	err := pool.QueryRow(ctx,
		`SELECT model_hash, model_type, model_name, brier_score, roc_auc, metadata_json, created_at::text
		 FROM ml_models WHERE model_hash = $1`, modelHash,
	).Scan(&r.ModelHash, &r.ModelType, &r.ModelName, &r.BrierScore, &r.ROCAUC, &r.Metadata, &r.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("model registry get: %w", err)
	}
	return &r, nil
}

func GetLatestModel(ctx context.Context, pool *pgxpool.Pool, modelType string) (*ModelRecord, error) {
	var r ModelRecord
	err := pool.QueryRow(ctx,
		`SELECT model_hash, model_type, model_name, brier_score, roc_auc, metadata_json, created_at::text
		 FROM ml_models WHERE model_type = $1 ORDER BY created_at DESC LIMIT 1`, modelType,
	).Scan(&r.ModelHash, &r.ModelType, &r.ModelName, &r.BrierScore, &r.ROCAUC, &r.Metadata, &r.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("model registry latest %s: %w", modelType, err)
	}
	return &r, nil
}

func VerifyModelHash(ctx context.Context, pool *pgxpool.Pool, expectedHash string) (bool, error) {
	_, err := GetModel(ctx, pool, expectedHash)
	return err == nil, err
}
