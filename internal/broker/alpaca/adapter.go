package alpaca

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	Symbol      string  `json:"symbol"`
	Qty         float64 `json:"qty,omitempty"`
	Notional    float64 `json:"notional,omitempty"`
	Side        string  `json:"side"`
	Type        string  `json:"type"`
	TimeInForce string  `json:"time_in_force"`
	LimitPrice  float64 `json:"limit_price,omitempty"`
	StopPrice   float64 `json:"stop_price,omitempty"`
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

	return &AlpacaAdapter{
		apiKey:    key,
		apiSecret: secret,
		baseURL:   baseURL,
		paper:     baseURL != "https://api.alpaca.markets",
		client:    &http.Client{Timeout: 15 * time.Second},
	}, nil
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
		},
	}
}

func (a *AlpacaAdapter) HealthCheck(ctx context.Context) error {
	_, err := a.doRequest(ctx, http.MethodGet, "/v2/account", nil)
	return err
}

func (a *AlpacaAdapter) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	url := a.baseURL + path

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

	alpacaReq := alpacaOrderRequest{
		Symbol:      req.Symbol,
		Qty:         req.Quantity,
		Side:        string(req.Side),
		Type:        string(req.Type),
		TimeInForce: string(req.TimeInForce),
	}

	if req.Type == broker.Limit || req.Type == broker.Stop {
		alpacaReq.LimitPrice = req.LimitPrice.Float64()
	}
	if req.Type == broker.Stop {
		alpacaReq.StopPrice = req.StopPrice.Float64()
	}

	data, err := a.doRequest(ctx, "POST", "/v2/orders", alpacaReq)
	if err != nil {
		return nil, err
	}

	var order alpacaOrder
	if err := json.Unmarshal(data, &order); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}

	filledQty := parseFloat(order.FilledQty)
	avgPrice := parseFloat(order.FilledAvgPrice)

	return &broker.OrderResponse{
		BrokerOrderID: order.ID,
		Status:        broker.OrderStatus(order.Status),
		FilledQty:     filledQty,
		AvgFillPrice:  types.FromFloat64(avgPrice),
	}, nil
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
		log.Printf("alpaca parseFloat: failed to parse %q: %v", s, err)
		return 0
	}
	return f
}
