package notify

import (
	"sync"
	"testing"
)

type mockPushHub struct {
	mu       sync.Mutex
	messages []struct {
		channel string
		data    interface{}
	}
}

func (m *mockPushHub) Broadcast(channel string, data interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, struct {
		channel string
		data    interface{}
	}{channel, data})
}

func (m *mockPushHub) lastChannel() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.messages) == 0 {
		return ""
	}
	return m.messages[len(m.messages)-1].channel
}

func (m *mockPushHub) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.messages)
}

func TestNotificationManagerPublish(t *testing.T) {
	mgr := NewManager()
	hub := &mockPushHub{}
	mgr.SetPushHub(hub)

	mgr.Publish(NewEvent(EventKillSwitchTriggered, LevelCritical, "Test", "Test message"))

	hub.count()
}

func TestPushNotifier(t *testing.T) {
	hub := &mockPushHub{}
	pn := NewPushNotifier(hub)

	if !pn.IsEnabled() {
		t.Error("push notifier should be enabled when hub is set")
	}
	if pn.Name() != "push" {
		t.Errorf("expected name 'push', got %s", pn.Name())
	}

	event := NewEvent(EventOrderFilled, LevelInfo, "Fill", "100 shares filled")
	err := pn.Send(event)
	if err != nil {
		t.Errorf("Send should succeed: %v", err)
	}

	if hub.lastChannel() != "notification_order_filled" {
		t.Errorf("expected last channel 'notification_order_filled', got %s", hub.lastChannel())
	}
}

func TestPushNotifierDisabledWhenNoHub(t *testing.T) {
	pn := NewPushNotifier(nil)
	if pn.IsEnabled() {
		t.Error("push notifier should be disabled when no hub")
	}
}

func TestEmailNotifierDisabledWhenNoService(t *testing.T) {
	en := NewEmailNotifier(nil)
	if en.IsEnabled() {
		t.Error("email notifier should be disabled when no service")
	}
}

func TestEventDefaults(t *testing.T) {
	event := NewEvent(EventDrawdownWarning, LevelWarning, "DD Warning", "Drawdown at 8%")
	if event.Type != EventDrawdownWarning {
		t.Errorf("expected EventDrawdownWarning, got %s", event.Type)
	}
	if event.Level != LevelWarning {
		t.Errorf("expected LevelWarning, got %s", event.Level)
	}
	if event.Timestamp.IsZero() {
		t.Error("timestamp should not be zero")
	}
}

func TestNotificationManagerNotifiers(t *testing.T) {
	mgr := NewManager()

	hub := &mockPushHub{}
	pn := NewPushNotifier(hub)
	mgr.Register(pn)

	notifiers := mgr.Notifiers()
	if len(notifiers) != 1 {
		t.Errorf("expected 1 notifier, got %d", len(notifiers))
	}
}

func TestTelegramSeverityRouting(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("TELEGRAM_DEV_CHAT", "dev-chat-id")
	t.Setenv("TELEGRAM_OPS_CHAT", "ops-chat-id")

	tn := NewTelegramNotifier()
	if !tn.IsEnabled() {
		t.Fatal("telegram should be enabled with token + chats")
	}
	if tn.Name() != "telegram" {
		t.Errorf("expected 'telegram', got %s", tn.Name())
	}

	if err := tn.Send(Event{Type: "info_test", Level: LevelInfo, Title: "test"}); err != nil {
		t.Errorf("INFO should not error: %v", err)
	}

	devChats := tn.warningChats()
	if len(devChats) != 1 || devChats[0] != "dev-chat-id" {
		t.Errorf("WARNING should target dev chat, got %v", devChats)
	}

	critChats := tn.criticalChats()
	if len(critChats) != 2 {
		t.Errorf("CRITICAL should target ops+dev chats, got %v (len=%d)", critChats, len(critChats))
	}
}

func TestTelegramRoutingLegacyFallback(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("TELEGRAM_CHAT_ID", "legacy-chat")

	tn := NewTelegramNotifier()
	if !tn.IsEnabled() {
		t.Fatal("telegram should be enabled with legacy config")
	}

	if len(tn.warningChats()) != 1 || len(tn.criticalChats()) != 1 {
		t.Errorf("legacy config: warning=%d critical=%d", len(tn.warningChats()), len(tn.criticalChats()))
	}
}

func TestTelegramRoutingINFOIsSilent(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("TELEGRAM_DEV_CHAT", "dev")

	tn := NewTelegramNotifier()
	err := tn.Send(Event{Type: "daily_update", Level: LevelInfo, Title: "Morning Brief"})
	if err != nil {
		t.Errorf("INFO send should succeed silently: %v", err)
	}
}
