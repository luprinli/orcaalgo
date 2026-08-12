package db

import (
	"context"
	"fmt"
	"time"
)

func (r *Repository) LoadRegimeLogs(ctx context.Context, start, end time.Time) ([]RegimeLog, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT timestamp, hmm_state, confidence, symbol
		 FROM regime_logs WHERE timestamp >= $1 AND timestamp <= $2
		 ORDER BY timestamp ASC`, start, end,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []RegimeLog
	for rows.Next() {
		var l RegimeLog
		if err := rows.Scan(&l.Time, &l.HMMState, &l.Confidence, &l.Symbol); err != nil {
			continue
		}
		logs = append(logs, l)
	}
	return logs, nil
}

func (r *Repository) LoadSentimentLogs(ctx context.Context, start, end time.Time) ([]SentimentLog, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT timestamp, score, label
		 FROM sentiment_logs WHERE timestamp >= $1 AND timestamp <= $2
		 ORDER BY timestamp ASC`, start, end,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []SentimentLog
	for rows.Next() {
		var l SentimentLog
		if err := rows.Scan(&l.Time, &l.Score, &l.Label); err != nil {
			continue
		}
		logs = append(logs, l)
	}
	return logs, nil
}

func (r *Repository) LoadVIXLogs(ctx context.Context, start, end time.Time) ([]VIXLog, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT timestamp,
		        CASE WHEN vix_value > 200
		             THEN vix_value::DOUBLE PRECISION / 10000.0
		             ELSE vix_value::DOUBLE PRECISION END,
		        CASE WHEN ABS(vix_change) > 200
		             THEN vix_change::DOUBLE PRECISION / 10000.0
		             ELSE vix_change::DOUBLE PRECISION END
		 FROM vix_logs WHERE timestamp >= $1 AND timestamp <= $2
		 ORDER BY timestamp ASC`, start, end,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []VIXLog
	for rows.Next() {
		var l VIXLog
		if err := rows.Scan(&l.Time, &l.VIXValue, &l.VIXChange); err != nil {
			continue
		}
		logs = append(logs, l)
	}
	return logs, nil
}

func (r *Repository) InsertRegimeLog(ctx context.Context, symbol string, state int8, confidence float64) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO regime_logs (symbol, hmm_state, confidence) VALUES ($1, $2, $3)`,
		symbol, state, confidence,
	)
	return err
}

func (r *Repository) InsertKillSwitchEvent(ctx context.Context, reason, source string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO kill_switch_history (reason, source) VALUES ($1, $2)`,
		reason, source,
	)
	return err
}

func (r *Repository) ResolveKillSwitchEvent(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE kill_switch_history SET resolved_at=now() WHERE id=$1`, id,
	)
	return err
}

func (r *Repository) ListKillSwitchHistory(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, triggered_at, reason, source, resolved_at FROM kill_switch_history ORDER BY triggered_at DESC LIMIT $1`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var history []map[string]interface{}
	for rows.Next() {
		var id int64
		var triggered time.Time
		var reason, source string
		var resolved *time.Time
		if err := rows.Scan(&id, &triggered, &reason, &source, &resolved); err != nil {
			continue
		}
		entry := map[string]interface{}{
			"id": id, "triggered_at": triggered, "reason": reason, "source": source,
		}
		if resolved != nil {
			entry["resolved_at"] = *resolved
		}
		history = append(history, entry)
	}
	return history, nil
}

func (r *Repository) InsertAuditLog(ctx context.Context, level, component, message string, metadata map[string]interface{}) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO audit_log (level, component, message, metadata) VALUES ($1, $2, $3, $4)`,
		level, component, message, metadata,
	)
	return err
}

func (r *Repository) ListAuditLogs(ctx context.Context, component string, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `SELECT id, timestamp, level, component, message, metadata FROM audit_log`
	args := []interface{}{}
	if component != "" {
		query += ` WHERE component=$1`
		args = append(args, component)
	}
	query += ` ORDER BY timestamp DESC LIMIT $` + fmt.Sprintf("%d", len(args)+1)
	args = append(args, limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []map[string]interface{}
	for rows.Next() {
		var id int64
		var ts time.Time
		var lvl, comp, msg string
		var meta map[string]interface{}
		if err := rows.Scan(&id, &ts, &lvl, &comp, &msg, &meta); err != nil {
			continue
		}
		logs = append(logs, map[string]interface{}{
			"id": id, "timestamp": ts, "level": lvl, "component": comp, "message": msg, "metadata": meta,
		})
	}
	return logs, nil
}
