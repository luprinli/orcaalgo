package risk

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

type mockBroker struct{}

func (m *mockBroker) CancelAllOrders(ctx context.Context) error    { return nil }
func (m *mockBroker) CloseAllPositions(ctx context.Context) error  { return nil }

func newTestKillSwitch() *KillSwitch {
	return NewKillSwitch(&mockBroker{})
}

func TestKillSwitchTrigger(t *testing.T) {
	ks := newTestKillSwitch()

	if ks.IsHalted() {
		t.Error("kill switch should not be triggered initially")
	}

	err := ks.Trigger("test breach")
	if err != nil {
		t.Errorf("Trigger should succeed, got: %v", err)
	}
	if !ks.IsHalted() {
		t.Error("kill switch should be triggered after Trigger call")
	}

	halted, reason, _ := ks.Status()
	if !halted {
		t.Error("Status should report halted=true")
	}
	if reason != "test breach" {
		t.Errorf("expected reason 'test breach', got '%s'", reason)
	}
}

func TestKillSwitchResume(t *testing.T) {
	ks := newTestKillSwitch()

	err := ks.Trigger("test")
	if err != nil {
		t.Errorf("Trigger should succeed, got: %v", err)
	}
	ks.Resume()

	if ks.IsHalted() {
		t.Error("kill switch should not be triggered after Resume")
	}

	halted, _, _ := ks.Status()
	if halted {
		t.Error("Status should report halted=false after resume")
	}
}

func TestKillSwitchIdempotentTrigger(t *testing.T) {
	ks := newTestKillSwitch()

	err1 := ks.Trigger("first")
	if err1 != nil {
		t.Errorf("first Trigger should succeed, got: %v", err1)
	}
	err2 := ks.Trigger("second")
	if err2 == nil {
		t.Error("second Trigger should fail while halted")
	}

	_, reason, _ := ks.Status()
	if reason != "first" {
		t.Errorf("expected original reason 'first', got '%s'", reason)
	}
}

func TestKillSwitchConcurrent(t *testing.T) {
	ks := newTestKillSwitch()
	var wg sync.WaitGroup
	n := 100
	triggered := make(chan struct{}, 1)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := ks.Trigger("concurrent")
			if err == nil {
				select {
				case triggered <- struct{}{}:
				default:
				}
			}
		}()
	}
	wg.Wait()

	if !ks.IsHalted() {
		t.Error("kill switch should be triggered after concurrent access")
	}

	if !ks.IsFlightReady() {
		t.Error("after concurrent Trigger/Resume, flight flag should be back to safe state")
	}

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ks.Resume()
		}()
	}
	wg.Wait()

	halted, _, _ := ks.Status()
	if halted {
		t.Error("kill switch should be resumed after concurrent resume")
	}
}

func TestKillSwitchDefaultValues(t *testing.T) {
	ks := newTestKillSwitch()

	halted, reason, lastTrigger := ks.Status()
	if halted {
		t.Error("default should be not halted")
	}
	if reason != "" {
		t.Errorf("default reason should be empty, got '%s'", reason)
	}
	if !lastTrigger.IsZero() {
		t.Error("default lastTrigger should be zero time")
	}
}

func TestKillSwitchReentrancyGuard(t *testing.T) {
	ks := newTestKillSwitch()
	var wg sync.WaitGroup
	errChan := make(chan error, 200)
	successCount := int32(0)

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := ks.Trigger("reentrancy-test")
			if err != nil {
				errChan <- err
			} else {
				atomic.AddInt32(&successCount, 1)
			}
		}()
	}
	wg.Wait()
	close(errChan)

	if atomic.LoadInt32(&successCount) != 1 {
		t.Errorf("exactly 1 goroutine should succeed, got %d", successCount)
	}

	lockErrors := 0
	alreadyHalted := 0
	for e := range errChan {
		switch {
		case errors.Is(e, ErrLocked):
			lockErrors++
		case errors.Is(e, ErrAlreadyHalted):
			alreadyHalted++
		}
	}
	if lockErrors+alreadyHalted != 199 {
		t.Errorf("expected 199 errors (lock or already-halted), got %d lock + %d halted",
			lockErrors, alreadyHalted)
	}
}

func TestKillSwitchIsInFlight(t *testing.T) {
	ks := newTestKillSwitch()

	if !ks.IsFlightReady() {
		t.Error("new kill switch should be flight-ready")
	}

	err := ks.Trigger("flight-test")
	if err != nil {
		t.Errorf("Trigger should succeed, got: %v", err)
	}

	if !ks.IsFlightReady() {
		t.Error("after Trigger completes, flight should be back to ready")
	}
}

func TestKillSwitchTriggerWhileHalted(t *testing.T) {
	ks := newTestKillSwitch()

	err := ks.Trigger("first")
	if err != nil {
		t.Fatalf("first Trigger should succeed, got: %v", err)
	}

	err = ks.Trigger("second")
	if !errors.Is(err, ErrAlreadyHalted) {
		t.Errorf("expected ErrAlreadyHalted, got: %v", err)
	}
}
