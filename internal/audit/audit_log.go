package audit

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type AuditLog struct {
	mu   sync.Mutex
	db   *sql.DB
	path string
}

func Open(path string) (*AuditLog, error) {
	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("audit log open: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS audit_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp_ns INTEGER NOT NULL,
			event_type TEXT NOT NULL,
			payload_json TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_audit_events_ts ON audit_events(timestamp_ns);
		CREATE INDEX IF NOT EXISTS idx_audit_events_type ON audit_events(event_type);
	`); err != nil {
		db.Close()
		return nil, fmt.Errorf("audit log migration: %w", err)
	}

	return &AuditLog{db: db, path: path}, nil
}

func OpenFromEnv() (*AuditLog, error) {
	path := os.Getenv("ORCA_AUDIT_LOG_PATH")
	if path == "" {
		return nil, fmt.Errorf("ORCA_AUDIT_LOG_PATH not set")
	}
	return Open(path)
}

func (a *AuditLog) Append(eventType string, payload interface{}) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		payloadBytes = []byte(fmt.Sprintf(`{"error":"marshal: %s"}`, err.Error()))
	}

	_, err = a.db.Exec(
		"INSERT INTO audit_events (timestamp_ns, event_type, payload_json) VALUES (?, ?, ?)",
		time.Now().UnixNano(), eventType, string(payloadBytes),
	)
	if err != nil {
		return fmt.Errorf("audit log append: %w", err)
	}

	return nil
}

func (a *AuditLog) Query(since time.Time, eventType string, limit int) ([]AuditEvent, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	sinceNS := since.UnixNano()

	var rows *sql.Rows
	var err error

	if eventType != "" {
		rows, err = a.db.Query(
			"SELECT id, timestamp_ns, event_type, payload_json FROM audit_events WHERE timestamp_ns > ? AND event_type = ? ORDER BY id LIMIT ?",
			sinceNS, eventType, limit,
		)
	} else {
		rows, err = a.db.Query(
			"SELECT id, timestamp_ns, event_type, payload_json FROM audit_events WHERE timestamp_ns > ? ORDER BY id LIMIT ?",
			sinceNS, limit,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("audit log query: %w", err)
	}
	defer rows.Close()

	var events []AuditEvent
	for rows.Next() {
		var e AuditEvent
		if err := rows.Scan(&e.ID, &e.TimestampNS, &e.EventType, &e.PayloadJSON); err != nil {
			return nil, fmt.Errorf("audit log scan: %w", err)
		}
		events = append(events, e)
	}

	return events, rows.Err()
}

func (a *AuditLog) Count() (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	var count int
	err := a.db.QueryRow("SELECT COUNT(*) FROM audit_events").Scan(&count)
	return count, err
}

func (a *AuditLog) Close() error {
	return a.db.Close()
}

type AuditEvent struct {
	ID          int64  `json:"id"`
	TimestampNS int64  `json:"timestamp_ns"`
	EventType   string `json:"event_type"`
	PayloadJSON string `json:"payload_json"`
}
