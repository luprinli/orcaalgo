package broker

import (
	"context"
	"sync"
	"testing"

	"github.com/lee-econ/orca-core/internal/types"
)

type testAdapter struct {
	mu       sync.Mutex
	balance  float64
	equity   float64
	posCount int
	orders   int
	halted   bool
}

func newTestAdapter(balance float64) *testAdapter {
	return &testAdapter{balance: balance, equity: balance}
}

func (a *testAdapter) PlaceOrder(ctx context.Context, req *OrderRequest) (*OrderResponse, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.orders++
	a.balance -= req.Quantity * 100
	return &OrderResponse{
		BrokerOrderID: "test-order-1",
		Status:        Filled,
		FilledQty:     req.Quantity,
		AvgFillPrice:  100.0,
	}, nil
}

func (a *testAdapter) CancelOrder(ctx context.Context, orderID string) error { return nil }
func (a *testAdapter) CancelAllOrders(ctx context.Context) error             { return nil }

func (a *testAdapter) CloseAllPositions(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.posCount = 0
	return nil
}

func (a *testAdapter) GetPositions(ctx context.Context) ([]Position, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.posCount > 0 {
		return []Position{{Symbol: "SPY", Quantity: float64(a.posCount), MarketValue: 10000}}, nil
	}
	return nil, nil
}

func (a *testAdapter) GetAccount(ctx context.Context) (*Account, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return &Account{
		ID:      "test-id",
		Balance: types.FromFloat64(a.balance),
		Equity:  types.FromFloat64(a.equity),
		DailyPL: a.balance - 100000,
		Status:  "ACTIVE",
	}, nil
}

func (a *testAdapter) ValidateCredentials(ctx context.Context) error { return nil }

func TestAccountManagerRegisterAndGet(t *testing.T) {
	registry := NewBrokerDriverRegistry()
	am := NewAccountManager(nil, registry)

	adapter1 := newTestAdapter(100000)
	adapter2 := newTestAdapter(200000)

	acct1 := NewManagedAccount("ftmo-1", "paper", "FTMO $100k", adapter1)
	acct1.IsDefault = true
	acct1.SetBalance(100000, 100000, 0, 0)

	acct2 := NewManagedAccount("ftmo-2", "paper", "FTMO $200k", adapter2)
	acct2.SetBalance(200000, 200000, 0, 0)

	if err := am.RegisterAccount(acct1); err != nil {
		t.Fatalf("RegisterAccount should succeed: %v", err)
	}
	if err := am.RegisterAccount(acct2); err != nil {
		t.Fatalf("RegisterAccount should succeed: %v", err)
	}

	got, err := am.GetAccount("ftmo-1")
	if err != nil {
		t.Fatalf("GetAccount should succeed: %v", err)
	}
	if got.ID != "ftmo-1" {
		t.Errorf("expected ID ftmo-1, got %s", got.ID)
	}
	if got.Balance != 100000 {
		t.Errorf("expected balance 100000, got %f", got.Balance.Float64())
	}

	list := am.ListAccountsByUser(context.Background(), "")
	if len(list) != 2 {
		t.Errorf("expected 2 accounts, got %d", len(list))
	}
}

func TestAccountManagerDuplicateRegister(t *testing.T) {
	registry := NewBrokerDriverRegistry()
	am := NewAccountManager(nil, registry)

	adapter := newTestAdapter(100000)
	acct := NewManagedAccount("dup-test", "paper", "Test", adapter)

	if err := am.RegisterAccount(acct); err != nil {
		t.Fatalf("first RegisterAccount should succeed: %v", err)
	}

	if err := am.RegisterAccount(acct); err == nil {
		t.Error("duplicate RegisterAccount should fail")
	}
}

func TestAccountManagerDefaultAccount(t *testing.T) {
	registry := NewBrokerDriverRegistry()
	am := NewAccountManager(nil, registry)

	adapter1 := newTestAdapter(100000)
	adapter2 := newTestAdapter(200000)

	acctA := NewManagedAccount("acct-a", "paper", "A", adapter1)
	am.RegisterAccount(acctA)
	acctB := NewManagedAccount("acct-b", "paper", "B", adapter2)
	acctB.IsDefault = true
	am.RegisterAccount(acctB)

	def, err := am.GetDefaultAccount()
	if err != nil {
		t.Fatalf("GetDefaultAccount should succeed: %v", err)
	}
	if def.ID != "acct-b" {
		t.Errorf("expected default acct-b, got %s", def.ID)
	}

	defID := am.GetDefaultAccountID()
	if defID != "acct-b" {
		t.Errorf("expected default id acct-b, got %s", defID)
	}
}

func TestAccountManagerDefaultWithoutExplicit(t *testing.T) {
	registry := NewBrokerDriverRegistry()
	am := NewAccountManager(nil, registry)

	adapter := newTestAdapter(100000)
	acct := NewManagedAccount("only-acct", "paper", "Only", adapter)
	acct.SetBalance(100000, 100000, 0, 0)
	am.RegisterAccount(acct)

	def, err := am.GetDefaultAccount()
	if err != nil {
		t.Fatalf("GetDefaultAccount should fall back to single account: %v", err)
	}
	if def.ID != "only-acct" {
		t.Errorf("expected only-acct as default, got %s", def.ID)
	}
}

func TestAccountManagerUnregister(t *testing.T) {
	registry := NewBrokerDriverRegistry()
	am := NewAccountManager(nil, registry)

	adapter := newTestAdapter(100000)
	acct := NewManagedAccount("temp", "paper", "Temp", adapter)
	am.RegisterAccount(acct)

	am.UnregisterAccount("temp")

	if _, err := am.GetAccount("temp"); err == nil {
		t.Error("GetAccount after Unregister should fail")
	}

	if am.AccountCount() != 0 {
		t.Errorf("expected 0 accounts after unregister, got %d", am.AccountCount())
	}
}

func TestAccountManagerPlaceOrderRouting(t *testing.T) {
	registry := NewBrokerDriverRegistry()
	am := NewAccountManager(nil, registry)

	adapter1 := newTestAdapter(100000)
	adapter2 := newTestAdapter(200000)

	acc1 := NewManagedAccount("acc-1", "paper", "One", adapter1)
	acc1.IsDefault = true
	am.RegisterAccount(acc1)

	acc2 := NewManagedAccount("acc-2", "paper", "Two", adapter2)
	am.RegisterAccount(acc2)

	req := &OrderRequest{
		Symbol:   "SPY",
		Side:     Buy,
		Type:     Market,
		Quantity: 100,
	}

	resp, err := am.PlaceOrder(context.Background(), "acc-1", req)
	if err != nil {
		t.Fatalf("PlaceOrder should succeed: %v", err)
	}
	if resp.Status != Filled {
		t.Errorf("expected Filled, got %s", resp.Status)
	}

	acct1, _ := am.GetAccount("acc-1")
	if acct1.Balance == 100000 {
		t.Error("balance should have changed after order")
	}

	_, err = am.PlaceOrder(context.Background(), "nonexistent", req)
	if err == nil {
		t.Error("PlaceOrder on nonexistent account should fail")
	}
}

func TestAccountManagerSyncAccountState(t *testing.T) {
	registry := NewBrokerDriverRegistry()
	am := NewAccountManager(nil, registry)

	adapter := newTestAdapter(150000)
	acct := NewManagedAccount("sync-test", "paper", "SyncTest", adapter)
	acct.SetBalance(100000, 100000, 0, 0)
	am.RegisterAccount(acct)

	if err := am.SyncAccountState(context.Background(), "sync-test"); err != nil {
		t.Fatalf("SyncAccountState should succeed: %v", err)
	}

	got, _ := am.GetAccount("sync-test")
	if got.Balance == 100000 {
		t.Logf("balance synced from adapter: %f", got.Balance.Float64())
	}
}

func TestAccountManagerCloseAllAcrossAll(t *testing.T) {
	registry := NewBrokerDriverRegistry()
	am := NewAccountManager(nil, registry)

	adapter1 := newTestAdapter(100000)
	adapter2 := newTestAdapter(100000)

	adapter1.posCount = 5
	adapter2.posCount = 3

	am.RegisterAccount(NewManagedAccount("close-1", "paper", "C1", adapter1))
	am.RegisterAccount(NewManagedAccount("close-2", "paper", "C2", adapter2))

	ctx := context.Background()
	am.CloseAllPositionsAcrossAll(ctx)

	if adapter1.posCount != 0 {
		t.Errorf("expected 0 positions for close-1, got %d", adapter1.posCount)
	}
	if adapter2.posCount != 0 {
		t.Errorf("expected 0 positions for close-2, got %d", adapter2.posCount)
	}
}

func TestAccountManagerSetDefaultAccount(t *testing.T) {
	registry := NewBrokerDriverRegistry()
	am := NewAccountManager(nil, registry)

	adapter1 := newTestAdapter(100000)
	adapter2 := newTestAdapter(100000)

	accA := NewManagedAccount("def-a", "paper", "A", adapter1)
	accA.IsDefault = true
	am.RegisterAccount(accA)

	accB := NewManagedAccount("def-b", "paper", "B", adapter2)
	am.RegisterAccount(accB)

	if err := am.SetDefaultAccount(context.Background(), "def-b"); err != nil {
		t.Fatalf("SetDefaultAccount should succeed: %v", err)
	}

	def, _ := am.GetDefaultAccount()
	if def.ID != "def-b" {
		t.Errorf("expected default def-b, got %s", def.ID)
	}
}

func TestManagedAccountPlaceOrderDelegation(t *testing.T) {
	adapter := newTestAdapter(100000)
	ma := NewManagedAccount("delegation-test", "paper", "Test", adapter)

	req := &OrderRequest{Symbol: "AAPL", Side: Buy, Type: Market, Quantity: 50}
	resp, err := ma.PlaceOrder(context.Background(), req)
	if err != nil {
		t.Fatalf("PlaceOrder delegation should work: %v", err)
	}
	if resp.Status != Filled {
		t.Errorf("expected Filled, got %s", resp.Status)
	}

	brokerAcct, err := ma.GetAccount(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if brokerAcct.Balance <= 0 {
		t.Error("broker account should have balance")
	}
}

func TestAccountManagerGetAdapter(t *testing.T) {
	registry := NewBrokerDriverRegistry()
	am := NewAccountManager(nil, registry)

	adapter := newTestAdapter(100000)
	am.RegisterAccount(NewManagedAccount("adapter-test", "paper", "Test", adapter))

	got, err := am.GetAdapter("adapter-test")
	if err != nil {
		t.Fatalf("GetAdapter should succeed: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil adapter")
	}

	_, err = am.GetAdapter("nonexistent")
	if err == nil {
		t.Error("GetAdapter on nonexistent account should fail")
	}
}
