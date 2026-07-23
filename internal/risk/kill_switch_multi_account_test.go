package risk

import (
	"context"
	"sync"
	"testing"

	"github.com/lee-econ/orca-core/internal/broker"
	"github.com/lee-econ/orca-core/internal/broker/paper"
)

type accountCancellerAdapter struct {
	mu    sync.Mutex
	count int
}

func (a *accountCancellerAdapter) CancelAllOrdersAcrossAll(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.count++
}

func (a *accountCancellerAdapter) CloseAllPositionsAcrossAll(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.count++
}

func (a *accountCancellerAdapter) calls() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.count
}

func TestKillSwitchWithAccountCanceller(t *testing.T) {
	mockAC := &accountCancellerAdapter{}

	ks := NewKillSwitch(&mockBroker{})
	ks.SetAccountCanceller(mockAC)

	err := ks.Trigger("test with account canceller")
	if err != nil {
		t.Errorf("Trigger should succeed: %v", err)
	}
	if !ks.IsHalted() {
		t.Error("kill switch should be halted")
	}

	if mockAC.calls() != 2 {
		t.Errorf("expected 2 calls to AccountCanceller (CancelAllOrdersAcrossAll + CloseAllPositionsAcrossAll), got %d", mockAC.calls())
	}
}

func TestKillSwitchWithoutAccountCanceller(t *testing.T) {
	ks := NewKillSwitch(&mockBroker{})

	err := ks.Trigger("test without account canceller")
	if err != nil {
		t.Errorf("Trigger should succeed: %v", err)
	}
	if !ks.IsHalted() {
		t.Error("kill switch should be halted")
	}
}

func TestKillSwitchAccountCancellerNotHalted(t *testing.T) {
	mockAC := &accountCancellerAdapter{}
	ks := NewKillSwitch(&mockBroker{})

	if ks.IsHalted() {
		t.Error("kill switch should not be halted initially")
	}

	ks.SetAccountCanceller(mockAC)

	err := ks.Trigger("test")
	if err != nil {
		t.Fatal(err)
	}

	ks.Resume()

	err = ks.Trigger("second trigger")
	if err != nil {
		t.Errorf("second trigger after resume should succeed: %v", err)
	}
	if mockAC.calls() != 4 {
		t.Errorf("expected 4 total calls (2 per trigger), got %d", mockAC.calls())
	}
}

func TestKillSwitchAccountCancellerConcurrent(t *testing.T) {
	mockAC := &accountCancellerAdapter{}
	ks := NewKillSwitch(&mockBroker{})
	ks.SetAccountCanceller(mockAC)

	var wg sync.WaitGroup
	n := 100

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ks.Trigger("concurrent")
		}()
	}
	wg.Wait()

	if !ks.IsHalted() {
		t.Error("kill switch should be halted after concurrent access")
	}
	if !ks.IsFlightReady() {
		t.Error("flight flag should be back to safe state")
	}
}

func TestAccountCancellerInterface(t *testing.T) {
	registry := broker.NewBrokerDriverRegistry()
	am := broker.NewAccountManager(nil, registry)

	adapter1 := paper.NewAdapter(100000)
	adapter2 := paper.NewAdapter(100000)

	am.RegisterAccount(broker.NewManagedAccount("ks-a", "paper", "KSA", adapter1))
	am.RegisterAccount(broker.NewManagedAccount("ks-b", "paper", "KSB", adapter2))

	am.PlaceOrder(context.Background(), "ks-a", &broker.OrderRequest{Symbol: "SPY", Side: broker.Buy, Type: broker.Market, Quantity: 100})
	am.PlaceOrder(context.Background(), "ks-b", &broker.OrderRequest{Symbol: "QQQ", Side: broker.Buy, Type: broker.Market, Quantity: 100})

	ks := NewKillSwitch(&mockBroker{})
	ks.SetAccountCanceller(am)

	err := ks.Trigger("accounts test")
	if err != nil {
		t.Fatalf("Trigger should succeed: %v", err)
	}

	posA, _ := am.GetPositions(context.Background(), "ks-a")
	posB, _ := am.GetPositions(context.Background(), "ks-b")
	if len(posA) != 0 {
		t.Errorf("expected 0 positions for ks-a, got %d", len(posA))
	}
	if len(posB) != 0 {
		t.Errorf("expected 0 positions for ks-b, got %d", len(posB))
	}
}

func TestKillSwitchIdempotentWithAccountCanceller(t *testing.T) {
	mockAC := &accountCancellerAdapter{}
	ks := NewKillSwitch(&mockBroker{})
	ks.SetAccountCanceller(mockAC)

	err1 := ks.Trigger("first")
	if err1 != nil {
		t.Fatalf("first Trigger should succeed: %v", err1)
	}
	err2 := ks.Trigger("second")
	if err2 == nil {
		t.Error("second Trigger should fail while halted")
	}

	_, reason, _ := ks.Status()
	if reason != "first" {
		t.Errorf("expected original reason 'first', got '%s'", reason)
	}

	if mockAC.calls() != 2 {
		t.Errorf("expected only 2 calls (from first trigger), got %d", mockAC.calls())
	}
}
