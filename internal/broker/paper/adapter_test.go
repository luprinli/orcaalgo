package paper

import (
	"context"
	"testing"

	"github.com/lee-econ/orca-core/internal/broker"
	"github.com/lee-econ/orca-core/internal/types"
)

func TestAdapterPlaceLimitOrder(t *testing.T) {
	adapter := NewAdapter(100000)

	req := &broker.OrderRequest{
		Symbol:      "SPY",
		Side:        broker.Buy,
		Type:        broker.Limit,
		Quantity:    100,
		LimitPrice:  types.Price(58000000),
		TimeInForce: broker.Day,
	}

	resp, err := adapter.PlaceOrder(context.Background(), req)
	if err != nil {
		t.Fatalf("PlaceOrder failed: %v", err)
	}
	if resp.BrokerOrderID == "" {
		t.Error("expected non-empty order ID")
	}
	if resp.Status != broker.Filled {
		t.Errorf("expected status FILLED, got %s", resp.Status)
	}
	if resp.FilledQty != 100 {
		t.Errorf("expected 100 filled, got %f", resp.FilledQty)
	}
	if resp.AvgFillPrice.Float64() != 580.0 {
		t.Errorf("expected fill price 580.0, got %f", resp.AvgFillPrice.Float64())
	}
}

func TestAdapterPlaceMarketOrder(t *testing.T) {
	adapter := NewAdapter(100000)

	req := &broker.OrderRequest{
		Symbol:      "AAPL",
		Side:        broker.Sell,
		Type:        broker.Market,
		Quantity:    50,
		TimeInForce: broker.Day,
	}

	resp, err := adapter.PlaceOrder(context.Background(), req)
	if err != nil {
		t.Fatalf("PlaceOrder failed: %v", err)
	}
	if resp.Status != broker.Filled {
		t.Errorf("expected market order to fill, got %s", resp.Status)
	}
	if resp.FilledQty != 50 {
		t.Errorf("expected 50 filled, got %f", resp.FilledQty)
	}
}

func TestAdapterCancelOrder(t *testing.T) {
	adapter := NewAdapter(100000)

	req := &broker.OrderRequest{
		Symbol:      "SPY",
		Side:        broker.Buy,
		Type:        broker.Limit,
		Quantity:    100,
		LimitPrice:  types.Price(58000000),
		TimeInForce: broker.Day,
	}

	resp, err := adapter.PlaceOrder(context.Background(), req)
	if err != nil {
		t.Fatalf("PlaceOrder failed: %v", err)
	}

	if err := adapter.CancelOrder(context.Background(), resp.BrokerOrderID); err != nil {
		t.Fatalf("CancelOrder failed: %v", err)
	}
}

func TestAdapterCancelAllOrders(t *testing.T) {
	adapter := NewAdapter(100000)

	for i := 0; i < 3; i++ {
		req := &broker.OrderRequest{
			Symbol:      "SPY",
			Side:        broker.Buy,
			Type:        broker.Limit,
			Quantity:    100,
			LimitPrice:  types.Price(58000000),
			TimeInForce: broker.Day,
		}
		_, err := adapter.PlaceOrder(context.Background(), req)
		if err != nil {
			t.Fatalf("PlaceOrder failed: %v", err)
		}
	}

	if err := adapter.CancelAllOrders(context.Background()); err != nil {
		t.Fatalf("CancelAllOrders failed: %v", err)
	}
}

func TestAdapterGetPositions(t *testing.T) {
	adapter := NewAdapter(100000)

	positions, err := adapter.GetPositions(context.Background())
	if err != nil {
		t.Fatalf("GetPositions failed: %v", err)
	}
	if len(positions) > 0 {
		t.Errorf("expected 0 positions initially, got %d", len(positions))
	}

	buyReq := &broker.OrderRequest{
		Symbol:      "SPY",
		Side:        broker.Buy,
		Type:        broker.Market,
		Quantity:    100,
		TimeInForce: broker.Day,
	}
	_, err = adapter.PlaceOrder(context.Background(), buyReq)
	if err != nil {
		t.Fatalf("PlaceOrder failed: %v", err)
	}

	positions, err = adapter.GetPositions(context.Background())
	if err != nil {
		t.Fatalf("GetPositions after buy failed: %v", err)
	}
	if len(positions) != 1 {
		t.Fatalf("expected 1 position, got %d", len(positions))
	}
	if positions[0].Symbol != "SPY" {
		t.Errorf("expected SPY, got %s", positions[0].Symbol)
	}
	if positions[0].Quantity != 100 {
		t.Errorf("expected qty 100, got %f", positions[0].Quantity)
	}
}

func TestAdapterGetAccount(t *testing.T) {
	adapter := NewAdapter(100000)

	account, err := adapter.GetAccount(context.Background())
	if err != nil {
		t.Fatalf("GetAccount failed: %v", err)
	}
	if account.ID == "" {
		t.Error("expected non-empty account ID")
	}
	if account.Balance.Float64() <= 0 {
		t.Errorf("expected positive balance, got %f", account.Balance.Float64())
	}
	if account.Equity.Float64() <= 0 {
		t.Errorf("expected positive equity, got %f", account.Equity.Float64())
	}
	if account.Status == "" {
		t.Error("expected non-empty status")
	}
}

func TestAdapterCloseAllPositions(t *testing.T) {
	adapter := NewAdapter(100000)

	req := &broker.OrderRequest{
		Symbol:      "SPY",
		Side:        broker.Buy,
		Type:        broker.Market,
		Quantity:    100,
		TimeInForce: broker.Day,
	}
	_, err := adapter.PlaceOrder(context.Background(), req)
	if err != nil {
		t.Fatalf("PlaceOrder failed: %v", err)
	}

	positions, _ := adapter.GetPositions(context.Background())
	if len(positions) != 1 {
		t.Fatal("expected position after buy")
	}

	if err := adapter.CloseAllPositions(context.Background()); err != nil {
		t.Fatalf("CloseAllPositions failed: %v", err)
	}

	positions, err = adapter.GetPositions(context.Background())
	if err != nil {
		t.Fatalf("GetPositions after close failed: %v", err)
	}
	if len(positions) != 0 {
		t.Errorf("expected 0 positions after close, got %d", len(positions))
	}
}

func TestAdapterValidateCredentials(t *testing.T) {
	adapter := NewAdapter(100000)
	if err := adapter.ValidateCredentials(context.Background()); err != nil {
		t.Fatalf("ValidateCredentials failed: %v", err)
	}
}

func TestAdapterReset(t *testing.T) {
	adapter := NewAdapter(100000)

	req := &broker.OrderRequest{
		Symbol:      "SPY",
		Side:        broker.Buy,
		Type:        broker.Market,
		Quantity:    100,
		TimeInForce: broker.Day,
	}
	_, _ = adapter.PlaceOrder(context.Background(), req)

	adapter.Reset()

	positions, err := adapter.GetPositions(context.Background())
	if err != nil {
		t.Fatalf("GetPositions after reset failed: %v", err)
	}
	if len(positions) != 0 {
		t.Errorf("expected 0 positions after reset, got %d", len(positions))
	}
}
