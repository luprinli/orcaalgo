package alpaca

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/lee-econ/orca-core/internal/broker"
	"github.com/lee-econ/orca-core/internal/types"
)

type AlpacaAdapter struct {
	apiKey    string
	apiSecret string
	baseURL   string
	paper     bool
	client    *http.Client
}

type alpacaOrderRequest struct {
	Symbol      string           `json:"symbol"`
	Qty         float64          `json:"qty,omitempty"`
	Notional    float64          `json:"notional,omitempty"`
	Side        string           `json:"side"`
	Type        string           `json:"type"`
	TimeInForce string           `json:"time_in_force"`
	LimitPrice  types.Price      `json:"limit_price,omitempty"`
	StopPrice   types.Price      `json:"stop_price,omitempty"`
	OrderClass  string           `json:"order_class,omitempty"`
	StopLoss    *alpacaStopLoss  `json:"stop_loss,omitempty"`
	TakeProfit  *alpacaTakeProfit `json:"take_profit,omitempty"`
}

type alpacaStopLoss struct {
	StopPrice  types.Price `json:"stop_price"`
	LimitPrice types.Price `json:"limit_price,omitempty"`
}

type alpacaTakeProfit struct {
	LimitPrice types.Price `json:"limit_price"`
}

type alpacaOrder struct {
	ID            string  `json:"id"`
	ClientOrderID string  `json:"client_order_id"`
	Symbol        string  `json:"symbol"`
	Side          string  `json:"side"`
	Type          string  `json:"type"`
	Qty           string  `json:"qty"`
	FilledQty     string  `json:"filled_qty"`
	FilledAvgPrice string  `json:"filled_avg_price"`
	Status        string  `json:"status"`
}

type alpacaPosition struct {
	Symbol        string `json:"symbol"`
	Qty           string `json:"qty"`
	AvgEntryPrice string `json:"avg_entry_price"`
	MarketValue   string `json:"market_value"`
	UnrealizedPL  string `json:"unrealized_pl"`
}

type alpacaAccount struct {
	ID               string `json:"id"`
	BuyingPower      string `json:"buying_power"`
	Cash             string `json:"cash"`
	PortfolioValue   string `json:"portfolio_value"`
	Equity           string `json:"equity"`
	LastEquity       string `json:"last_equity"`
	Status           string `json:"status"`
}

func NewAdapter() (*AlpacaAdapter, error) {
	key := os.Getenv("ALPACA_API_KEY")
	secret := os.Getenv("ALPACA_API_SECRET")
	if key == "" || secret == "" {
		return nil, fmt.Errorf("ALPACA_API_KEY or ALPACA_API_SECRET not set")
	}

	baseURL := "https://api.alpaca.markets"
	if os.Getenv("ALPACA_PAPER") == "true" || os.Getenv("ALPACA_BASE_URL") != "" {
		if url := os.Getenv("ALPACA_BASE_URL"); url != "" {
			baseURL = url
		} else {
			baseURL = "https://paper-api.alpaca.markets"
		}
	}

	return NewAdapterWithCredentials(key, secret, baseURL), nil
}

// NewAdapterWithCredentials builds an adapter from explicit credentials and a
// base URL (per-account BYOK). An empty base URL defaults to the live Alpaca
// endpoint.
func NewAdapterWithCredentials(key, secret, baseURL string) *AlpacaAdapter {
	if baseURL == "" {
		baseURL = "https://api.alpaca.markets"
	}
	return &AlpacaAdapter{
		apiKey:    key,
		apiSecret: secret,
		baseURL:   baseURL,
		paper:     baseURL != "https://api.alpaca.markets",
		client:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (a *AlpacaAdapter) Manifest() broker.AdapterManifest {
	prio := 1 // secondary by default
	if !a.paper {
		prio = 0 // live Alpaca is primary
	}
	return broker.AdapterManifest{
		ID:         "alpaca",
		BrokerType: "alpaca",
		Priority:   prio,
		Capabilities: []broker.Capability{
			broker.CapPlaceOrder,
			broker.CapCancelOrder,
			broker.CapCancelAllOrders,
			broker.CapCloseAllPositions,
			broker.CapGetPositions,
			broker.CapGetAccount,
			broker.CapValidateCredentials,
			broker.CapReplaceOrder,
		},
	}
}

func (a *AlpacaAdapter) HealthCheck(ctx context.Context) error {
	_, err := a.doRequest(ctx, http.MethodGet, "/v2/account", nil)
	return err
}

func (a *AlpacaAdapter) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	return a.doRequestTo(ctx, a.baseURL, method, path, body)
}

// doRequestTo is doRequest against an explicit base URL, used for the Alpaca
// data API (data.alpaca.markets) which differs from the trading host.
func (a *AlpacaAdapter) doRequestTo(ctx context.Context, base, method, path string, body interface{}) ([]byte, error) {
	url := base + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("APCA-API-KEY-ID", a.apiKey)
	req.Header.Set("APCA-API-SECRET-KEY", a.apiSecret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("alpaca API error %d: %s", resp.StatusCode, string(data))
	}

	return data, nil
}

func (a *AlpacaAdapter) PlaceOrder(ctx context.Context, req *broker.OrderRequest) (*broker.OrderResponse, error) {
	if err := a.ValidateCredentials(ctx); err != nil {
		return nil, err
	}

	alpacaReq := a.buildOrderRequest(req)

	data, err := a.doRequest(ctx, "POST", "/v2/orders", alpacaReq)
	if err != nil {
		return nil, err
	}

	return a.parseOrder(data)
}

// buildOrderRequest converts an OrderRequest into the Alpaca payload, including
// the bracketed (OTO) stop-loss/take-profit legs. Extracted so the bracket
// construction is unit-testable without an HTTP round-trip.
func (a *AlpacaAdapter) buildOrderRequest(req *broker.OrderRequest) alpacaOrderRequest {
	alpacaReq := alpacaOrderRequest{
		Symbol:      req.Symbol,
		Qty:         req.Quantity,
		Side:        string(req.Side),
		Type:        string(req.Type),
		TimeInForce: string(req.TimeInForce),
	}

	if req.Type == broker.Limit || req.Type == broker.Stop {
		alpacaReq.LimitPrice = req.LimitPrice
	}
	if req.Type == broker.Stop {
		alpacaReq.StopPrice = req.StopPrice
	}

	if req.StopLoss.Float64() > 0 || req.TakeProfit.Float64() > 0 {
		alpacaReq.OrderClass = "oto"
	}
	if req.StopLoss.Float64() > 0 {
		alpacaReq.StopLoss = &alpacaStopLoss{StopPrice: req.StopLoss}
	}
	if req.TakeProfit.Float64() > 0 {
		alpacaReq.TakeProfit = &alpacaTakeProfit{LimitPrice: req.TakeProfit}
	}

	return alpacaReq
}

func (a *AlpacaAdapter) parseOrder(data []byte) (*broker.OrderResponse, error) {
	var order alpacaOrder
	if err := json.Unmarshal(data, &order); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}
	return &broker.OrderResponse{
		BrokerOrderID: order.ID,
		Status:        broker.OrderStatus(order.Status),
		FilledQty:     parseFloat(order.FilledQty),
		AvgFillPrice:  types.FromFloat64(parseFloat(order.FilledAvgPrice)),
	}, nil
}

// ReplaceOrder modifies an open order (PATCH /v2/orders/{id}), e.g. to move a
// protective stop or resize a limit. Zero-valued update fields are omitted so
// the broker leaves them unchanged.
func (a *AlpacaAdapter) ReplaceOrder(ctx context.Context, orderID string, update *broker.OrderUpdate) (*broker.OrderResponse, error) {
	if update == nil {
		return nil, fmt.Errorf("nil order update")
	}
	body := map[string]interface{}{}
	if update.Quantity > 0 {
		body["qty"] = update.Quantity
	}
	if update.LimitPrice.Float64() > 0 {
		body["limit_price"] = update.LimitPrice
	}
	if update.StopPrice.Float64() > 0 {
		body["stop_price"] = update.StopPrice
	}

	data, err := a.doRequest(ctx, "PATCH", "/v2/orders/"+orderID, body)
	if err != nil {
		return nil, err
	}
	return a.parseOrder(data)
}

// liquidationLimitPrice computes the limit price for a liquidation order at a
// percentage discount from the reference price. A non-positive discount returns
// 0, signalling a market order.
func liquidationLimitPrice(refPrice, discountPercent float64) types.Price {
	if refPrice <= 0 || discountPercent <= 0 {
		return 0
	}
	return types.PriceFromFloat(refPrice * (1 - discountPercent/100.0))
}

// Liquidate flattens every position: longs are sold and shorts are covered. A
// positive DiscountPercent places limit orders at that discount from the
// position's implied market price (MarketValue/Quantity); 0 places market
// orders. DryRun computes the plan without placing orders.
func (a *AlpacaAdapter) Liquidate(ctx context.Context, req *broker.LiquidationRequest) (*broker.LiquidationResult, error) {
	if req == nil {
		return nil, fmt.Errorf("nil liquidation request")
	}
	positions, err := a.GetPositions(ctx)
	if err != nil {
		return nil, err
	}

	result := &broker.LiquidationResult{DryRun: req.DryRun}
	for _, p := range positions {
		if p.Quantity == 0 {
			continue
		}
		qty := math.Abs(p.Quantity)
		side := broker.Sell
		if p.Quantity < 0 {
			side = broker.Buy
		}

		lr := broker.LiquidationPositionResult{
			Symbol:   p.Symbol,
			Quantity: qty,
		}

		refPrice := 0.0
		if qty > 0 {
			refPrice = p.MarketValue.Float64() / qty
		}
		if refPrice <= 0 {
			lr.Skipped = true
			lr.Reason = "no reference price"
			result.Skipped++
			result.Positions = append(result.Positions, lr)
			continue
		}
		lr.ReferencePrice = types.PriceFromFloat(refPrice)
		lr.LimitPrice = liquidationLimitPrice(refPrice, req.DiscountPercent)

		orderType := broker.Market
		if lr.LimitPrice.Float64() > 0 {
			orderType = broker.Limit
		}

		if req.DryRun {
			lr.Closed = true
			result.Closed++
			result.Positions = append(result.Positions, lr)
			continue
		}

		orderReq := &broker.OrderRequest{
			Symbol:      p.Symbol,
			Side:        side,
			Type:        orderType,
			Quantity:    qty,
			LimitPrice:  lr.LimitPrice,
			TimeInForce: broker.Day,
		}
		if _, err := a.PlaceOrder(ctx, orderReq); err != nil {
			lr.Failed = true
			lr.Reason = err.Error()
			result.Failed++
		} else {
			lr.Closed = true
			result.Closed++
		}
		result.Positions = append(result.Positions, lr)
	}
	return result, nil
}

func (a *AlpacaAdapter) CancelOrder(ctx context.Context, orderID string) error {
	_, err := a.doRequest(ctx, "DELETE", "/v2/orders/"+orderID, nil)
	return err
}

func (a *AlpacaAdapter) CancelAllOrders(ctx context.Context) error {
	_, err := a.doRequest(ctx, "DELETE", "/v2/orders", nil)
	return err
}

func (a *AlpacaAdapter) CloseAllPositions(ctx context.Context) error {
	_, err := a.doRequest(ctx, "DELETE", "/v2/positions", nil)
	return err
}

func (a *AlpacaAdapter) GetPositions(ctx context.Context) ([]broker.Position, error) {
	data, err := a.doRequest(ctx, "GET", "/v2/positions", nil)
	if err != nil {
		return nil, err
	}

	var alpacaPositions []alpacaPosition
	if err := json.Unmarshal(data, &alpacaPositions); err != nil {
		return nil, fmt.Errorf("unmarshal positions: %w", err)
	}

	result := make([]broker.Position, len(alpacaPositions))
	for i, p := range alpacaPositions {
		result[i] = broker.Position{
			Symbol:        p.Symbol,
			Quantity:      parseFloat(p.Qty),
			AvgEntryPrice: types.FromFloat64(parseFloat(p.AvgEntryPrice)),
			MarketValue:   types.FromFloat64(parseFloat(p.MarketValue)),
			UnrealizedPL:  parseFloat(p.UnrealizedPL),
		}
	}
	return result, nil
}

func (a *AlpacaAdapter) GetAccount(ctx context.Context) (*broker.Account, error) {
	data, err := a.doRequest(ctx, "GET", "/v2/account", nil)
	if err != nil {
		return nil, err
	}

	var acc alpacaAccount
	if err := json.Unmarshal(data, &acc); err != nil {
		return nil, fmt.Errorf("unmarshal account: %w", err)
	}

	equity := parseFloat(acc.Equity)
	lastEquity := parseFloat(acc.LastEquity)

	return &broker.Account{
		ID:          acc.ID,
		Balance:     types.FromFloat64(parseFloat(acc.Cash)),
		Equity:      types.FromFloat64(equity),
		BuyingPower: types.FromFloat64(parseFloat(acc.BuyingPower)),
		DailyPL:     equity - lastEquity,
		Status:      acc.Status,
	}, nil
}

func (a *AlpacaAdapter) ValidateCredentials(ctx context.Context) error {
	if len(a.apiKey) < 10 {
		return fmt.Errorf("invalid API key length")
	}
	if len(a.apiSecret) < 10 {
		return fmt.Errorf("invalid API secret length")
	}
	_, err := a.doRequest(ctx, "GET", "/v2/account", nil)
	return err
}

func parseFloat(s string) float64 {
	if s == "" { return 0 }
	var f float64
	n, err := fmt.Sscanf(s, "%f", &f)
	if n != 1 || err != nil {
		slog.Warn("failed to parse float", "value", s, "error", err, "component", "alpaca")
		return 0
	}
	return f
}
