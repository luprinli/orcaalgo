package ibkr

import (
	"context"
	"fmt"
	"os"

	"github.com/lee-econ/orca-core/internal/broker"
)

type IBKRAdapter struct {
	host string; port int; clientID int; connected bool
}

func NewAdapter() (*IBKRAdapter, error) {
	return &IBKRAdapter{
		host:     envOrDefault("IBKR_HOST", "127.0.0.1"),
		port:     7497,
		clientID: 1,
	}, nil
}

func (a *IBKRAdapter) PlaceOrder(ctx context.Context, req *broker.OrderRequest) (*broker.OrderResponse, error) {
	return nil, fmt.Errorf("IBKR adapter: live trading not yet implemented. Use TWS API or IB Gateway")
}

func (a *IBKRAdapter) CancelOrder(ctx context.Context, orderID string) error {
	return fmt.Errorf("IBKR adapter: not yet implemented")
}

func (a *IBKRAdapter) CancelAllOrders(ctx context.Context) error {
	return fmt.Errorf("IBKR adapter: not yet implemented")
}

func (a *IBKRAdapter) CloseAllPositions(ctx context.Context) error {
	return fmt.Errorf("IBKR adapter: not yet implemented")
}

func (a *IBKRAdapter) GetPositions(ctx context.Context) ([]broker.Position, error) {
	return nil, fmt.Errorf("IBKR adapter: not yet implemented")
}

func (a *IBKRAdapter) GetAccount(ctx context.Context) (*broker.Account, error) {
	return &broker.Account{ID: "ibkr-stub", Status: "INACTIVE"}, nil
}

func (a *IBKRAdapter) ValidateCredentials(ctx context.Context) error {
	return nil
}

func (a *IBKRAdapter) Manifest() broker.AdapterManifest {
	return broker.AdapterManifest{
		ID:         "ibkr",
		BrokerType: "ibkr",
		Priority:   2,
		Capabilities: []broker.Capability{
			broker.CapPlaceOrder,
			broker.CapCancelOrder,
			broker.CapCancelAllOrders,
			broker.CapCloseAllPositions,
			broker.CapGetPositions,
			broker.CapGetAccount,
			broker.CapValidateCredentials,
		},
	}
}

func (a *IBKRAdapter) HealthCheck(ctx context.Context) error {
	return nil
}

func envOrDefault(key, def string) string { if v := os.Getenv(key); v != "" { return v }; return def }