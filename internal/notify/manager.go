package notify

import "time"

type EventType string

const (
	EventKillSwitchTriggered EventType = "kill_switch_triggered"
	EventDrawdownWarning     EventType = "drawdown_warning"
	EventDailyLossWarning    EventType = "daily_loss_warning"
	EventRegimeChanged       EventType = "regime_changed"
	EventConsistencyOutlier  EventType = "consistency_outlier"
	EventCredentialExpiry    EventType = "credential_expiry"
	EventOrderFilled         EventType = "order_filled"
	EventPositionClosed      EventType = "position_closed"
	EventBacktestComplete    EventType = "backtest_complete"
	EventMemoryScanDetected  EventType = "memory_scan_detected"
	EventAccountSync         EventType = "account_sync"
)

type Level string

const (
	LevelInfo     Level = "info"
	LevelWarning  Level = "warning"
	LevelCritical Level = "critical"
)

type Event struct {
	Type      EventType              `json:"type"`
	Level     Level                  `json:"level"`
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Details   string                 `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type Notifier interface {
	Name() string
	Send(event Event) error
	IsEnabled() bool
}

type Manager struct {
	notifiers []Notifier
	hub       PushHub
}

type PushHub interface {
	Broadcast(channel string, data interface{})
}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) Register(notifier Notifier) {
	m.notifiers = append(m.notifiers, notifier)
}

func (m *Manager) SetPushHub(hub PushHub) {
	m.hub = hub
}

func (m *Manager) Notifiers() []Notifier {
	return m.notifiers
}

func (m *Manager) Publish(event Event) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	for _, n := range m.notifiers {
		if n.IsEnabled() {
			go func(notifier Notifier) {
				if err := notifier.Send(event); err != nil {
				}
			}(n)
		}
	}

	if m.hub != nil {
		go m.hub.Broadcast("notification", event)
	}
}

func NewEvent(eventType EventType, level Level, title, message string) Event {
	return Event{
		Type:      eventType,
		Level:     level,
		Title:     title,
		Message:   message,
		Timestamp: time.Now(),
	}
}
