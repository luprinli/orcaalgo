package broker

import (
	"context"

	"github.com/lee-econ/orca-core/internal/types"
)

type OrderSide string
type OrderType string
type OrderStatus string
type TimeInForce string
type Capability string

const (
	Buy  OrderSide = "BUY"
	Sell OrderSide = "SELL"

	Market OrderType = "MARKET"
	Limit  OrderType = "LIMIT"
	Stop   OrderType = "STOP"

	New             OrderStatus = "NEW"
	PartiallyFilled OrderStatus = "PARTIALLY_FILLED"
	Filled          OrderStatus = "FILLED"
	Canceled        OrderStatus = "CANCELED"
	Rejected        OrderStatus = "REJECTED"

	Day TimeInForce = "DAY"
	GTC TimeInForce = "GTC"
	IOC TimeInForce = "IOC"

	CapPlaceOrder        Capability = "PLACE_ORDER"
	CapCancelOrder       Capability = "CANCEL_ORDER"
	CapCancelAllOrders   Capability = "CANCEL_ALL_ORDERS"
	CapCloseAllPositions Capability = "CLOSE_ALL_POSITIONS"
	CapGetPositions      Capability = "GET_POSITIONS"
	CapGetAccount        Capability = "GET_ACCOUNT"
	CapValidateCredentials Capability = "VALIDATE_CREDENTIALS"
)

type Adapter interface {
	PlaceOrder(ctx context.Context, req *OrderRequest) (*OrderResponse, error)
	CancelOrder(ctx context.Context, orderID string) error
	CancelAllOrders(ctx context.Context) error
	CloseAllPositions(ctx context.Context) error
	GetPositions(ctx context.Context) ([]Position, error)
	GetAccount(ctx context.Context) (*Account, error)
	ValidateCredentials(ctx context.Context) error
}

// CapableAdapter extends Adapter with a manifest declaring supported
// capabilities, a priority for fallback ordering, and a health check.
// Adapters that implement this interface participate in the
// BrokerDriverRegistry capability-based routing system.
type CapableAdapter interface {
	Adapter
	Manifest() AdapterManifest
	HealthCheck(ctx context.Context) error
}

// AdapterManifest declares an adapter's identity, capabilities, and
// routing priority. Lower priority values have higher preference
// (priority 0 = primary, 1 = secondary, 2 = tertiary, etc.).
type AdapterManifest struct {
	ID           string
	BrokerType   string       // "alpaca", "ibkr", "paper"
	Priority     int          // 0 = primary, higher = fallback
	Capabilities []Capability
}

type TransactionalAdapter interface {
	Adapter
	PrepareOrder(ctx context.Context, req *OrderRequest) (brokerID string, err error)
	ConfirmOrder(ctx context.Context, brokerID string) (*OrderResponse, error)
	IsTransactional() bool
}

type OrderRequest struct {
	AccountID   string
	Symbol      string
	Side        OrderSide
	Type        OrderType
	Quantity    float64
	LimitPrice  types.Price
	StopPrice   types.Price
	TimeInForce TimeInForce
	StrategyID  string
	TxID        string
}

type OrderResponse struct {
	BrokerOrderID string
	Status        OrderStatus
	FilledQty     float64
	AvgFillPrice  types.Price
	TxID          string
}

type Position struct {
	Symbol        string      `json:"symbol"`
	Quantity      float64     `json:"quantity"`
	AvgEntryPrice types.Price `json:"avg_entry_price"`
	MarketValue   types.Price `json:"market_value"`
	UnrealizedPL  float64     `json:"unrealized_pl"`
}

type Account struct {
	ID          string      `json:"id"`
	Balance     types.Price `json:"balance"`
	Equity      types.Price `json:"equity"`
	BuyingPower types.Price `json:"buying_power"`
	DailyPL     float64     `json:"daily_pl"`
	Status      string      `json:"status"`
}
