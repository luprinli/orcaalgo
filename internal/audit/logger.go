package audit

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditAction string

const (
	ActionLogin              AuditAction = "user_login"
	ActionPasswordReset      AuditAction = "password_reset"
	ActionEmailVerified      AuditAction = "email_verified"
	ActionOrderPlaced        AuditAction = "order_placed"
	ActionOrderCancelled     AuditAction = "order_cancelled"
	ActionPositionClosed     AuditAction = "position_closed"
	ActionKillSwitchTriggered AuditAction = "kill_switch_triggered"
	ActionStrategyCreated    AuditAction = "strategy_created"
	ActionStrategyUpdated     AuditAction = "strategy_updated"
	ActionStrategyDeleted     AuditAction = "strategy_deleted"
	ActionStrategyToggled     AuditAction = "strategy_toggled"
	ActionBacktestRun        AuditAction = "backtest_run"
	ActionBrokerAdded        AuditAction = "broker_added"
	ActionBrokerDeleted      AuditAction = "broker_deleted"
	ActionAccountAdded       AuditAction = "account_added"
	ActionAccountDeleted     AuditAction = "account_deleted"
	ActionSettingsUpdated    AuditAction = "settings_updated"
	ActionNotificationUpdated AuditAction = "notification_settings_updated"
	ActionAdminAction        AuditAction = "admin_action"
	ActionLLMKeyAdded        AuditAction = "llm_key_added"
	ActionLLMKeyDeleted      AuditAction = "llm_key_deleted"
	ActionCredentialStored   AuditAction = "credential_stored"
	ActionCredentialRotated  AuditAction = "credential_rotated"
)

type Entry struct {
	Timestamp    time.Time              `json:"timestamp"`
	UserID       string                 `json:"user_id,omitempty"`
	AccountID    string                 `json:"account_id,omitempty"`
	StrategyID   string                 `json:"strategy_id,omitempty"`
	Action       AuditAction            `json:"action"`
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
	SourceIP     string                 `json:"source_ip,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
}

type Filter struct {
	UserID     string
	Action     AuditAction
	ResourceType string
	Start      time.Time
	End        time.Time
	Limit      int
}

type Logger struct {
	pool *pgxpool.Pool
}

func NewLogger(pool *pgxpool.Pool) *Logger {
	return &Logger{pool: pool}
}

func (l *Logger) Log(ctx context.Context, entry Entry) error {
	if l.pool == nil {
		return nil
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}
	_, err := l.pool.Exec(ctx,
		`INSERT INTO audit_logs (timestamp, user_id, account_id, strategy_id, action, resource_type, resource_id, details, source_ip, user_agent)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		entry.Timestamp,
		nullStr(entry.UserID),
		nullStr(entry.AccountID),
		nullStr(entry.StrategyID),
		string(entry.Action),
		entry.ResourceType,
		entry.ResourceID,
		entry.Details,
		entry.SourceIP,
		entry.UserAgent,
	)
	return err
}

func (l *Logger) Query(ctx context.Context, filter Filter) ([]Entry, error) {
	if l.pool == nil {
		return nil, nil
	}

	query := `SELECT id, timestamp, COALESCE(user_id,''), COALESCE(account_id,''), COALESCE(strategy_id,''), action, resource_type, resource_id, details, COALESCE(source_ip,''), COALESCE(user_agent,'') FROM audit_logs WHERE 1=1`
	args := []interface{}{}
	argN := 0

	if filter.UserID != "" {
		argN++
		query += ` AND user_id = $` + itoa(argN)
		args = append(args, filter.UserID)
	}
	if filter.Action != "" {
		argN++
		query += ` AND action = $` + itoa(argN)
		args = append(args, string(filter.Action))
	}
	if filter.ResourceType != "" {
		argN++
		query += ` AND resource_type = $` + itoa(argN)
		args = append(args, filter.ResourceType)
	}
	if !filter.Start.IsZero() {
		argN++
		query += ` AND timestamp >= $` + itoa(argN)
		args = append(args, filter.Start)
	}
	if !filter.End.IsZero() {
		argN++
		query += ` AND timestamp <= $` + itoa(argN)
		args = append(args, filter.End)
	}

	query += ` ORDER BY timestamp DESC`

	if filter.Limit > 0 {
		argN++
		query += ` LIMIT $` + itoa(argN)
		args = append(args, filter.Limit)
	} else {
		query += ` LIMIT 200`
	}

	rows, err := l.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		var id string
		if err := rows.Scan(&id, &e.Timestamp, &e.UserID, &e.AccountID, &e.StrategyID, &e.Action, &e.ResourceType, &e.ResourceID, &e.Details, &e.SourceIP, &e.UserAgent); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func itoa(n int) string {
	digits := "0123456789"
	if n == 0 {
		return "0"
	}
	result := ""
	for n > 0 {
		result = string(digits[n%10]) + result
		n /= 10
	}
	return result
}
