package broker

import (
	"context"
	"errors"
	"testing"

	"github.com/lee-econ/orca-core/internal/types"
)

type preflightAdapter struct {
	positions   []Position
	account     *Account
	posErr      error
	accountErr  error
}

func (a *preflightAdapter) PlaceOrder(context.Context, *OrderRequest) (*OrderResponse, error) { return nil, nil }
func (a *preflightAdapter) CancelOrder(context.Context, string) error                         { return nil }
func (a *preflightAdapter) CancelAllOrders(context.Context) error                             { return nil }
func (a *preflightAdapter) CloseAllPositions(context.Context) error                           { return nil }
func (a *preflightAdapter) ValidateCredentials(context.Context) error                         { return nil }
func (a *preflightAdapter) GetPositions(context.Context) ([]Position, error)                  { return a.positions, a.posErr }
func (a *preflightAdapter) GetAccount(context.Context) (*Account, error)                      { return a.account, a.accountErr }

func buyReq(symbol string, qty, limit float64) *OrderRequest {
	return &OrderRequest{Symbol: symbol, Side: Buy, Quantity: qty, LimitPrice: types.PriceFromFloat(limit)}
}

func TestPreflight_NilAdapterPasses(t *testing.T) {
	if r := Preflight(context.Background(), nil, buyReq("SPY", 1, 100)); r.Skip {
		t.Errorf("nil adapter should not skip, got %+v", r)
	}
}

func TestPreflight_BuyPositionExists(t *testing.T) {
	a := &preflightAdapter{positions: []Position{{Symbol: "SPY", Quantity: 10}}}
	r := Preflight(context.Background(), a, buyReq("SPY", 1, 100))
	if !r.Skip || r.Reason != "position already open" {
		t.Errorf("expected skip (position already open), got %+v", r)
	}
}

func TestPreflight_BuyNoPositionPasses(t *testing.T) {
	a := &preflightAdapter{positions: []Position{{Symbol: "QQQ", Quantity: 1}}}
	r := Preflight(context.Background(), a, buyReq("SPY", 1, 100))
	if r.Skip {
		t.Errorf("expected pass, got %+v", r)
	}
}

func TestPreflight_SellNoPositionSkips(t *testing.T) {
	a := &preflightAdapter{}
	r := Preflight(context.Background(), a, &OrderRequest{Symbol: "SPY", Side: Sell, Quantity: 1})
	if !r.Skip || r.Reason != "no position to close" {
		t.Errorf("expected skip (no position to close), got %+v", r)
	}
}

func TestPreflight_SellWithPositionPasses(t *testing.T) {
	a := &preflightAdapter{positions: []Position{{Symbol: "SPY", Quantity: 5}}}
	r := Preflight(context.Background(), a, &OrderRequest{Symbol: "SPY", Side: Sell, Quantity: 5})
	if r.Skip {
		t.Errorf("expected pass, got %+v", r)
	}
}

func TestPreflight_InsufficientBuyingPower(t *testing.T) {
	a := &preflightAdapter{
		account: &Account{BuyingPower: types.PriceFromFloat(500)},
	}
	r := Preflight(context.Background(), a, buyReq("SPY", 10, 100)) // notional 1000 > 500
	if !r.Skip || r.Reason != "insufficient buying power" {
		t.Errorf("expected skip (insufficient buying power), got %+v", r)
	}
}

func TestPreflight_SufficientBuyingPower(t *testing.T) {
	a := &preflightAdapter{
		account: &Account{BuyingPower: types.PriceFromFloat(2000)},
	}
	r := Preflight(context.Background(), a, buyReq("SPY", 10, 100)) // notional 1000 < 2000
	if r.Skip {
		t.Errorf("expected pass, got %+v", r)
	}
}

func TestPreflight_BrokerErrorFailsOpen(t *testing.T) {
	a := &preflightAdapter{posErr: errors.New("broker down")}
	r := Preflight(context.Background(), a, buyReq("SPY", 1, 100))
	if r.Skip {
		t.Errorf("broker error should fail open (non-skipping), got %+v", r)
	}
}

func TestPreflight_MarketOrderSkipsBuyingPowerCheck(t *testing.T) {
	a := &preflightAdapter{account: &Account{BuyingPower: types.PriceFromFloat(1)}}
	r := Preflight(context.Background(), a, &OrderRequest{Symbol: "SPY", Side: Buy, Quantity: 10, LimitPrice: types.Price(0)})
	if r.Skip {
		t.Errorf("market order (no limit price) should skip the buying-power check, got %+v", r)
	}
}
