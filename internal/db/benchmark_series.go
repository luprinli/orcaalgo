package db

import (
	"context"
	"time"
)

// BenchmarkSeriesPoint is a single named benchmark-series observation (e.g. a
// daily risk-free yield). The series is name-keyed and (name, timestamp)-unique.
type BenchmarkSeriesPoint struct {
	Time   time.Time
	Value  float64
	Source string
}

// UpsertBenchmarkSeriesPoint inserts or updates one named-series observation.
func (r *Repository) UpsertBenchmarkSeriesPoint(ctx context.Context, name string, t time.Time, value float64, source string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO benchmark_series (name, timestamp, value, source)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (name, timestamp) DO UPDATE SET value = EXCLUDED.value, source = EXCLUDED.source`,
		name, t, value, source,
	)
	return err
}

// LoadBenchmarkSeries returns observations for a named series in [start, end],
// ordered ascending.
func (r *Repository) LoadBenchmarkSeries(ctx context.Context, name string, start, end time.Time) ([]BenchmarkSeriesPoint, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT timestamp, value, COALESCE(source, '')
		FROM benchmark_series
		WHERE name = $1 AND timestamp >= $2 AND timestamp <= $3
		ORDER BY timestamp ASC`, name, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []BenchmarkSeriesPoint
	for rows.Next() {
		var p BenchmarkSeriesPoint
		if err := rows.Scan(&p.Time, &p.Value, &p.Source); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
