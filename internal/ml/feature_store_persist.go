package ml

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func (fs *FeatureStore) Persist(ctx context.Context, pool *pgxpool.Pool, symbol string, ts string) error {
	if pool == nil {
		return nil
	}

	n := fs.count
	if n > 256 {
		n = 256
	}

	prices := make([]float64, n)
	highs := make([]float64, n)
	lows := make([]float64, n)
	volumes := make([]float64, n)
	for i := 0; i < n; i++ {
		idx := (fs.count - n + i) % 256
		prices[i] = fs.prices[idx]
		highs[i] = fs.highs[idx]
		lows[i] = fs.lows[idx]
		volumes[i] = fs.volumes[idx]
	}

	pricesStr := floatSliceToPostgres(prices)
	highsStr := floatSliceToPostgres(highs)
	lowsStr := floatSliceToPostgres(lows)
	volStr := floatSliceToPostgres(volumes)

	_, err := pool.Exec(ctx,
		`INSERT INTO feature_store_state (symbol, updated_at, bar_count, prices, highs, lows, volumes)
		 VALUES ($1, NOW(), $2, $3, $4, $5, $6)
		 ON CONFLICT (symbol) DO UPDATE SET
		   updated_at = NOW(), bar_count = $2, prices = $3, highs = $4, lows = $5, volumes = $6`,
		symbol, n, pricesStr, highsStr, lowsStr, volStr,
	)
	if err != nil {
		return fmt.Errorf("feature store persist: %w", err)
	}
	return nil
}

func LoadFeatureStore(ctx context.Context, pool *pgxpool.Pool, symbol string) (*FeatureStore, string, error) {
	if pool == nil {
		return nil, "", fmt.Errorf("nil pool")
	}

	var barCount int
	var pricesStr, highsStr, lowsStr, volStr string
	var updatedAt string

	err := pool.QueryRow(ctx,
		`SELECT bar_count, prices, highs, lows, volumes, updated_at::text
		 FROM feature_store_state
		 WHERE symbol = $1
		 ORDER BY updated_at DESC LIMIT 1`,
		symbol,
	).Scan(&barCount, &pricesStr, &highsStr, &lowsStr, &volStr, &updatedAt)
	if err != nil {
		return nil, "", fmt.Errorf("feature store load: %w", err)
	}

	prices := parseFloatSlice(pricesStr)
	highs := parseFloatSlice(highsStr)
	lows := parseFloatSlice(lowsStr)
	volumes := parseFloatSlice(volStr)

	return NewFeatureStore(prices, highs, lows, volumes), updatedAt, nil
}

func floatSliceToPostgres(v []float64) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = fmt.Sprintf("%.8f", f)
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func parseFloatSlice(s string) []float64 {
	s = strings.TrimPrefix(s, "{")
	s = strings.TrimSuffix(s, "}")
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]float64, 0, len(parts))
	for _, p := range parts {
		var f float64
		if _, err := fmt.Sscanf(strings.TrimSpace(p), "%f", &f); err == nil {
			result = append(result, f)
		}
	}
	return result
}
