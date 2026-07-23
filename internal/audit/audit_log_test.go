package audit

import (
	"os"
	"sync"
	"testing"
	"time"
)

func TestAuditLog_AppendAndQuery(t *testing.T) {
	path := t.TempDir() + "/test_audit.db"
	al, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer al.Close()

	if err := al.Append("order_placed", map[string]interface{}{
		"symbol": "SPY", "side": "BUY", "qty": 100,
	}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	if err := al.Append("order_filled", map[string]interface{}{
		"symbol": "SPY", "fill_price": 450.50,
	}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	count, err := al.Count()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 events, got %d", count)
	}

	events, err := al.Query(time.Time{}, "", 10)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].EventType != "order_placed" {
		t.Errorf("expected order_placed, got %s", events[0].EventType)
	}
	if events[1].EventType != "order_filled" {
		t.Errorf("expected order_filled, got %s", events[1].EventType)
	}
}

func TestAuditLog_QueryByType(t *testing.T) {
	path := t.TempDir() + "/test_audit_type.db"
	al, _ := Open(path)
	defer al.Close()

	al.Append("order_placed", map[string]string{"id": "1"})
	al.Append("kill_switch", map[string]string{"reason": "test"})
	al.Append("order_placed", map[string]string{"id": "2"})

	events, err := al.Query(time.Time{}, "order_placed", 10)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 order_placed events, got %d", len(events))
	}

	events, err = al.Query(time.Time{}, "kill_switch", 10)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 kill_switch event, got %d", len(events))
	}
}

func TestAuditLog_ConcurrentWrites(t *testing.T) {
	path := t.TempDir() + "/test_audit_concurrent.db"
	al, _ := Open(path)
	defer al.Close()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			al.Append("test_event", map[string]int{"n": n})
		}(i)
	}
	wg.Wait()

	count, err := al.Count()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 10 {
		t.Errorf("expected 10 events, got %d", count)
	}
}

func TestAuditLog_Durable(t *testing.T) {
	path := t.TempDir() + "/test_audit_durable.db"

	al, _ := Open(path)
	al.Append("test", map[string]string{"key": "value"})
	al.Close()

	al2, _ := Open(path)
	defer al2.Close()

	count, _ := al2.Count()
	if count != 1 {
		t.Fatalf("expected 1 event after reopen, got %d", count)
	}

	events, _ := al2.Query(time.Time{}, "", 10)
	if events[0].EventType != "test" {
		t.Errorf("expected 'test', got %s", events[0].EventType)
	}
}

func TestAuditLog_OpenFromEnv(t *testing.T) {
	path := t.TempDir() + "/test_from_env.db"
	os.Setenv("ORCA_AUDIT_LOG_PATH", path)
	defer os.Unsetenv("ORCA_AUDIT_LOG_PATH")

	al, err := OpenFromEnv()
	if err != nil {
		t.Fatalf("OpenFromEnv: %v", err)
	}

	al.Append("env_test", nil)
	count, _ := al.Count()
	if count != 1 {
		t.Errorf("expected 1 event, got %d", count)
	}
	al.Close()
}
