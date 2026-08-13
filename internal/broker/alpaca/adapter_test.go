package alpaca

import (
	"testing"

	"github.com/lee-econ/orca-core/internal/broker"
	"github.com/lee-econ/orca-core/internal/types"
)

func TestNewAdapterWithCredentials_UsesPassedValues(t *testing.T) {
	a := NewAdapterWithCredentials("sk-key", "sk-secret", "https://paper-api.alpaca.markets")
	if a.apiKey != "sk-key" {
		t.Errorf("apiKey = %q, want sk-key", a.apiKey)
	}
	if a.apiSecret != "sk-secret" {
		t.Errorf("apiSecret = %q, want sk-secret", a.apiSecret)
	}
	if a.baseURL != "https://paper-api.alpaca.markets" {
		t.Errorf("baseURL = %q, want paper URL", a.baseURL)
	}
	if !a.paper {
		t.Error("paper should be true for paper URL")
	}
}

func TestNewAdapterWithCredentials_LiveDefault(t *testing.T) {
	a := NewAdapterWithCredentials("k", "s", "https://api.alpaca.markets")
	if a.paper {
		t.Error("paper should be false for live URL")
	}
}

func TestNewAdapterWithCredentials_EmptyBaseURLDefaultsLive(t *testing.T) {
	a := NewAdapterWithCredentials("k", "s", "")
	if a.baseURL != "https://api.alpaca.markets" {
		t.Errorf("baseURL = %q, want live URL", a.baseURL)
	}
	if a.paper {
		t.Error("paper should be false for default live URL")
	}
}

func TestBuildOrderRequest_BracketStopLoss(t *testing.T) {
	a := NewAdapterWithCredentials("k", "s", "https://paper-api.alpaca.markets")
	req := &broker.OrderRequest{
		Symbol:      "SPY",
		Side:        broker.Buy,
		Type:        broker.Limit,
		Quantity:    10,
		LimitPrice:  types.PriceFromFloat(100),
		StopLoss:    types.PriceFromFloat(98),
		TimeInForce: broker.Day,
	}
	body := a.buildOrderRequest(req)
	if body.OrderClass != "oto" {
		t.Errorf("OrderClass = %q, want oto", body.OrderClass)
	}
	if body.StopLoss == nil || body.StopLoss.StopPrice.Float64() != 98 {
		t.Errorf("StopLoss = %+v, want stop 98", body.StopLoss)
	}
	if body.TakeProfit != nil {
		t.Errorf("TakeProfit should be nil when unset, got %+v", body.TakeProfit)
	}
}

func TestBuildOrderRequest_NoBracketWithoutStop(t *testing.T) {
	a := NewAdapterWithCredentials("k", "s", "")
	req := &broker.OrderRequest{Symbol: "SPY", Side: broker.Buy, Type: broker.Market, Quantity: 1}
	body := a.buildOrderRequest(req)
	if body.OrderClass != "" || body.StopLoss != nil || body.TakeProfit != nil {
		t.Errorf("no bracket expected, got class=%q stop=%+v take=%+v", body.OrderClass, body.StopLoss, body.TakeProfit)
	}
}

func TestManifest_HasReplaceOrder(t *testing.T) {
	a := NewAdapterWithCredentials("k", "s", "https://paper-api.alpaca.markets")
	has := false
	for _, c := range a.Manifest().Capabilities {
		if c == broker.CapReplaceOrder {
			has = true
		}
	}
	if !has {
		t.Error("alpaca manifest should advertise CapReplaceOrder")
	}
}

func TestLiquidationLimitPrice(t *testing.T) {
	// 5% discount on $100 -> $95.
	if got := liquidationLimitPrice(100, 5).Float64(); got != 95 {
		t.Errorf("limit price = %f, want 95", got)
	}
	// Zero/negative discount -> market (0).
	if got := liquidationLimitPrice(100, 0); got.Float64() != 0 {
		t.Errorf("zero discount should signal market order, got %f", got.Float64())
	}
	// Non-positive reference -> 0.
	if got := liquidationLimitPrice(0, 5); got.Float64() != 0 {
		t.Errorf("non-positive reference should return 0, got %f", got.Float64())
	}
}
