package broker

import (
	"context"
	"time"

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

	CapPlaceOrder          Capability = "PLACE_ORDER"
	CapCancelOrder         Capability = "CANCEL_ORDER"
	CapCancelAllOrders     Capability = "CANCEL_ALL_ORDERS"
	CapCloseAllPositions   Capability = "CLOSE_ALL_POSITIONS"
	CapGetPositions        Capability = "GET_POSITIONS"
	CapGetAccount          Capability = "GET_ACCOUNT"
	CapValidateCredentials Capability = "VALIDATE_CREDENTIALS"
	CapReconcileFills      Capability = "RECONCILE_FILLS"
	CapReplaceOrder        Capability = "REPLACE_ORDER"
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
	// StopLoss, when > 0, requests a bracketed (OTO) order whose protective
	// stop-loss leg closes the position at this price. 0 means no bracket.
	StopLoss types.Price
	// TakeProfit, when > 0, requests a bracketed take-profit leg at this limit.
	TakeProfit types.Price
}

// OrderUpdate carries the fields that may be modified when replacing an
// open order (e.g. moving a stop or resizing a limit). Zero-valued fields are
// left unchanged by the broker.
type OrderUpdate struct {
	Quantity   float64
	LimitPrice types.Price
	StopPrice  types.Price
}

// ReplaceOrderProvider is implemented by brokers that support modifying an
// open order (PATCH/replace semantics). It is an optional capability — the
// CapReplaceOrder manifest flag advertises support.
type ReplaceOrderProvider interface {
	ReplaceOrder(ctx context.Context, orderID string, update *OrderUpdate) (*OrderResponse, error)
}

// LiquidationRequest describes a full account flatten. A positive
// DiscountPercent places limit orders at that percentage below (longs) / above
// (shorts) the reference price; 0 places market orders. DryRun computes the
// plan without placing any orders.
type LiquidationRequest struct {
	DiscountPercent float64
	DryRun          bool
}

// LiquidationPositionResult is the per-position outcome of a liquidation.
type LiquidationPositionResult struct {
	Symbol         string      `json:"symbol"`
	Quantity       float64     `json:"quantity"`
	ReferencePrice types.Price `json:"reference_price"`
	LimitPrice     types.Price `json:"limit_price"`
	Closed         bool        `json:"closed"`
	Skipped        bool        `json:"skipped"`
	Failed         bool        `json:"failed"`
	Reason         string      `json:"reason,omitempty"`
}

// LiquidationResult summarizes a full-account flatten.
type LiquidationResult struct {
	DryRun    bool                        `json:"dry_run"`
	Closed    int                         `json:"closed"`
	Skipped   int                         `json:"skipped"`
	Failed    int                         `json:"failed"`
	Positions []LiquidationPositionResult `json:"positions"`
}

// Liquidator is implemented by brokers that support a full-account flatten
// with a dry-run preview. Optional capability.
type Liquidator interface {
	Liquidate(ctx context.Context, req *LiquidationRequest) (*LiquidationResult, error)
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

type TradeFill struct {
	OrderID       string
	Symbol        string
	Side          OrderSide
	Quantity      float64
	FillPrice     types.Price
	FillTime      time.Time
	BrokerOrderID string
}

type FillProvider interface {
	GetFills(ctx context.Context, date string) ([]TradeFill, error)
}
