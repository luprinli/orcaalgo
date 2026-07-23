package error

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	apperrors "github.com/lee-econ/orca-core/pkg/errors"
	"github.com/lee-econ/orca-core/internal/notify"
)

type Manager struct {
	pool            *pgxpool.Pool
	notifyManager   *notify.Manager
}

func NewManager(pool *pgxpool.Pool, nm *notify.Manager) *Manager {
	return &Manager{pool: pool, notifyManager: nm}
}

func (m *Manager) Handle(ctx context.Context, err error, component string) {
	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		log.Printf("[%s] error: %v", component, err)
		return
	}

	if m.pool != nil && appErr.IsError() {
		m.persist(ctx, appErr)
	}

	severity := "ERROR"
	if appErr.Severity == apperrors.SeverityWarning {
		severity = "WARNING"
	} else if appErr.Severity == apperrors.SeverityCritical {
		severity = "CRITICAL"
	}
	log.Printf("[%s][%s] %s: %v", component, severity, appErr.Message, appErr.Err)

	if appErr.ShouldNotify() && m.notifyManager != nil {
		level := notify.LevelWarning
		if appErr.Severity == apperrors.SeverityCritical {
			level = notify.LevelCritical
		}
		event := notify.NewEvent(
			notify.EventType("system_error"),
			level,
			appErr.Component+" Error",
			appErr.Message,
		)
		event.Details = appErr.UserAction
		if appErr.Err != nil {
			event.Details += "\n" + appErr.Err.Error()
		}
		m.notifyManager.Publish(event)
	}
}

func (m *Manager) persist(ctx context.Context, e *apperrors.AppError) {
	_, err := m.pool.Exec(ctx,
		`INSERT INTO error_logs (timestamp, user_id, component, category, severity, message, user_action, retryable)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		e.Timestamp,
		nullStr(e.UserID),
		e.Component,
		string(e.Category),
		string(e.Severity),
		e.Message,
		e.UserAction,
		e.Retryable,
	)
	if err != nil {
		log.Printf("error_manager: failed to persist error: %v", err)
	}
}

func (m *Manager) Query(ctx context.Context, component string, limit int) ([]map[string]interface{}, error) {
	if m.pool == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 100
	}

	query := `SELECT id, timestamp, COALESCE(user_id,''), component, category, severity, message, COALESCE(user_action,''), retryable, resolved, resolved_at FROM error_logs`
	args := []interface{}{}
	if component != "" {
		query += ` WHERE component=$1`
		args = append(args, component)
	}
	query += ` ORDER BY timestamp DESC LIMIT $` + itoa(len(args)+1)
	args = append(args, limit)

	rows, err := m.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id string
		var ts time.Time
		var uid, comp, cat, sev, msg, ua string
		var retryable, resolved bool
		var resolvedAt *time.Time
		if err := rows.Scan(&id, &ts, &uid, &comp, &cat, &sev, &msg, &ua, &retryable, &resolved, &resolvedAt); err != nil {
			continue
		}
		r := map[string]interface{}{
			"id": id, "timestamp": ts, "user_id": uid, "component": comp,
			"category": cat, "severity": sev, "message": msg,
			"user_action": ua, "retryable": retryable, "resolved": resolved,
		}
		if resolvedAt != nil {
			r["resolved_at"] = *resolvedAt
		}
		results = append(results, r)
	}
	return results, nil
}

func nullStr(s string) interface{} {
	if s == "" { return nil }
	return s
}

func itoa(n int) string {
	digits := "0123456789"
	if n == 0 { return "0" }
	result := ""
	for n > 0 {
		result = string(digits[n%10]) + result
		n /= 10
	}
	return result
}
