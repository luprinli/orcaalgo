package breaker

import (
	"testing"
	"time"
)

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cb := NewCircuitBreaker(3, time.Minute)
	for i := 0; i < 2; i++ {
		cb.RecordFailure()
		if cb.State() != CircuitClosed {
			t.Fatalf("failure %d: expected closed, got %v", i+1, cb.State())
		}
	}
	cb.RecordFailure()
	if cb.State() != CircuitOpen {
		t.Fatalf("expected open after threshold, got %v", cb.State())
	}
	if cb.Allow() {
		t.Fatal("expected Allow()=false while open before reset timeout")
	}
}

func TestCircuitBreaker_WarnThresholdEmitsOnce(t *testing.T) {
	cb := NewCircuitBreaker(5, time.Minute)
	cb.SetWarnThreshold(2)

	var events []BreakerEvent
	cb.SetObserver(func(e BreakerEvent) { events = append(events, e) })

	cb.RecordFailure()
	if len(events) != 0 {
		t.Fatalf("no warn expected below threshold, got %d events", len(events))
	}
	cb.RecordFailure()
	if len(events) != 1 || !events[0].Warn {
		t.Fatalf("expected one warn at threshold, got %+v", events)
	}
	cb.RecordFailure()
	if len(events) != 1 {
		t.Fatalf("warn should not repeat, got %d events", len(events))
	}
}

func TestCircuitBreaker_ObserverFiresOnOpen(t *testing.T) {
	cb := NewCircuitBreaker(2, time.Minute)
	var events []BreakerEvent
	cb.SetObserver(func(e BreakerEvent) { events = append(events, e) })

	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != CircuitOpen {
		t.Fatalf("expected open, got %v", cb.State())
	}
	if len(events) != 1 || events[0].Warn || events[0].To != CircuitOpen {
		t.Fatalf("expected one open event, got %+v", events)
	}
}

func TestCircuitBreaker_HalfOpenRecovery(t *testing.T) {
	cb := NewCircuitBreaker(1, 1*time.Millisecond)
	cb.RecordFailure()
	if cb.State() != CircuitOpen {
		t.Fatalf("expected open, got %v", cb.State())
	}
	time.Sleep(2 * time.Millisecond)
	if !cb.Allow() {
		t.Fatal("expected Allow()=true after reset timeout")
	}
	cb.RecordSuccess()
	if cb.State() != CircuitClosed {
		t.Fatalf("expected closed after recovery, got %v", cb.State())
	}
}

func TestCircuitBreaker_NoObserverIsSafe(t *testing.T) {
	cb := NewCircuitBreaker(1, time.Minute)
	cb.RecordFailure()
	if cb.State() != CircuitOpen {
		t.Fatalf("expected open, got %v", cb.State())
	}
}
