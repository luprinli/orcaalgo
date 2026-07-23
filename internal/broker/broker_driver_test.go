package broker

import (
	"context"
	"errors"
	"testing"

	"github.com/lee-econ/orca-core/internal/types"
)

type mockCapableAdapter struct {
	id       string
	brokerType string
	priority int
	healthy  bool
	orders   int
	errOnOp  error
}

func newMockCapable(id, brokerType string, priority int) *mockCapableAdapter {
	return &mockCapableAdapter{id: id, brokerType: brokerType, priority: priority, healthy: true}
}

func (m *mockCapableAdapter) PlaceOrder(ctx context.Context, req *OrderRequest) (*OrderResponse, error) {
	if m.errOnOp != nil {
		return nil, m.errOnOp
	}
	m.orders++
	return &OrderResponse{BrokerOrderID: m.id + "-order", Status: Filled, FilledQty: req.Quantity}, nil
}
func (m *mockCapableAdapter) CancelOrder(ctx context.Context, orderID string) error { return m.errOnOp }
func (m *mockCapableAdapter) CancelAllOrders(ctx context.Context) error              { return m.errOnOp }
func (m *mockCapableAdapter) CloseAllPositions(ctx context.Context) error            { return m.errOnOp }
func (m *mockCapableAdapter) GetPositions(ctx context.Context) ([]Position, error)   { return nil, m.errOnOp }
func (m *mockCapableAdapter) GetAccount(ctx context.Context) (*Account, error) {
	return &Account{ID: m.id, Balance: types.FromFloat64(100000), Status: "ACTIVE"}, m.errOnOp
}
func (m *mockCapableAdapter) ValidateCredentials(ctx context.Context) error { return m.errOnOp }

func (m *mockCapableAdapter) Manifest() AdapterManifest {
	return AdapterManifest{
		ID:         m.id,
		BrokerType: m.brokerType,
		Priority:   m.priority,
		Capabilities: []Capability{
			CapPlaceOrder, CapCancelOrder, CapCancelAllOrders,
			CapCloseAllPositions, CapGetPositions, CapGetAccount,
			CapValidateCredentials,
		},
	}
}
func (m *mockCapableAdapter) HealthCheck(ctx context.Context) error {
	if !m.healthy {
		return errors.New("unhealthy")
	}
	return nil
}

func TestBrokerDriver_ResolvePrimary(t *testing.T) {
	r := NewBrokerDriverRegistry()
	primary := newMockCapable("alpaca", "alpaca", 0)
	secondary := newMockCapable("paper", "paper", 1)
	r.Register(primary)
	r.Register(secondary)
	r.RunHealthChecks(context.Background())

	adapter, err := r.Resolve(context.Background(), CapPlaceOrder)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if adapter.Manifest().ID != "alpaca" {
		t.Errorf("expected primary=alpaca, got %s", adapter.Manifest().ID)
	}
}

func TestBrokerDriver_PriorityFallback(t *testing.T) {
	r := NewBrokerDriverRegistry()
	primary := newMockCapable("alpaca", "alpaca", 0)
	primary.healthy = false
	fallback := newMockCapable("paper", "paper", 1)

	r.Register(primary)
	r.Register(fallback)
	r.RunHealthChecks(context.Background())

	adapter, err := r.Resolve(context.Background(), CapPlaceOrder)
	if err != nil {
		t.Fatalf("Resolve with unhealthy primary: %v", err)
	}
	if adapter.Manifest().ID != "paper" {
		t.Errorf("expected fallback to paper, got %s", adapter.Manifest().ID)
	}
}

func TestBrokerDriver_AllUnhealthy(t *testing.T) {
	r := NewBrokerDriverRegistry()
	primary := newMockCapable("alpaca", "alpaca", 0)
	primary.healthy = false
	r.Register(primary)
	r.RunHealthChecks(context.Background())

	_, err := r.Resolve(context.Background(), CapPlaceOrder)
	if err == nil {
		t.Error("expected error when all adapters unhealthy")
	}
}

func TestBrokerDriver_NoCapabilityMatch(t *testing.T) {
	r := NewBrokerDriverRegistry()
	a := newMockCapable("paper", "paper", 0)
	r.Register(a)
	r.RunHealthChecks(context.Background())

	_, err := r.Resolve(context.Background(), Capability("NONEXISTENT"))
	if err == nil {
		t.Error("expected error for nonexistent capability")
	}
}

func TestBrokerDriver_FallbackOnError(t *testing.T) {
	r := NewBrokerDriverRegistry()
	primary := newMockCapable("alpaca", "alpaca", 0)
	primary.errOnOp = errors.New("alpaca down")
	fallback := newMockCapable("paper", "paper", 1)

	r.Register(primary)
	r.Register(fallback)
	r.RunHealthChecks(context.Background())

	callCount := 0
	err := r.ResolveWithFallback(context.Background(), CapPlaceOrder, func(a Adapter) error {
		callCount++
		if callCount == 1 {
			return errors.New("primary failed")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("ResolveWithFallback: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls (primary failed → fallback), got %d", callCount)
	}
}

func TestBrokerDriver_GetByID(t *testing.T) {
	r := NewBrokerDriverRegistry()
	a := newMockCapable("paper", "paper", 0)
	r.Register(a)

	adapter, ok := r.Get("paper")
	if !ok {
		t.Fatal("Get: adapter not found")
	}
	if adapter.(*mockCapableAdapter).id != "paper" {
		t.Errorf("expected paper adapter, got %v", adapter)
	}
}

func TestBrokerDriver_List(t *testing.T) {
	r := NewBrokerDriverRegistry()
	r.Register(newMockCapable("alpaca", "alpaca", 0))
	r.Register(newMockCapable("paper", "paper", 1))
	r.RunHealthChecks(context.Background())

	statuses := r.List()
	if len(statuses) != 2 {
		t.Errorf("expected 2 adapters, got %d", len(statuses))
	}
	if s, ok := statuses["alpaca"]; !ok || s.Priority != 0 {
		t.Errorf("alpaca status: %+v", s)
	}
	if s, ok := statuses["paper"]; !ok || s.Priority != 1 {
		t.Errorf("paper status: %+v", s)
	}
}

func TestBrokerDriver_ClosesAllPositions(t *testing.T) {
	r := NewBrokerDriverRegistry()
	r.Register(newMockCapable("alpaca", "alpaca", 0))
	r.Register(newMockCapable("paper", "paper", 1))

	r.CloseAllPositions(context.Background())
}
