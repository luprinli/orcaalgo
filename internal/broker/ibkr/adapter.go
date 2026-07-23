package ibkr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/lee-econ/orca-core/internal/broker"
	"github.com/lee-econ/orca-core/internal/types"
)

type IBKRConfig struct {
	Host      string
	Port      int
	AccountID string
	ClientID  int
}

type IBKRAdapter struct {
	cfg     IBKRConfig
	client  *RestClient
	mu      sync.RWMutex
	orders  map[string]*broker.OrderResponse
	fillTimes map[string]time.Time
	orderSeq int
}

func NewAdapterWithConfig(cfg IBKRConfig) (*IBKRAdapter, error) {
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	if cfg.Port == 0 {
		cfg.Port = 5000
	}
	if cfg.AccountID == "" {
		cfg.AccountID = "DU000000"
	}
	if cfg.ClientID == 0 {
		cfg.ClientID = 1
	}

	baseURL := fmt.Sprintf("https://%s:%d/v2/api", cfg.Host, cfg.Port)
	client := NewRestClient(baseURL, cfg.AccountID, 30*time.Second)

	return &IBKRAdapter{
		cfg:       cfg,
		client:    client,
		orders:    make(map[string]*broker.OrderResponse),
		fillTimes: make(map[string]time.Time),
	}, nil
}

func NewAdapter() (*IBKRAdapter, error) {
	port, _ := strconv.Atoi(envOrDefault("IBKR_PORT", "5000"))
	return NewAdapterWithConfig(IBKRConfig{
		Host:      envOrDefault("IBKR_HOST", "127.0.0.1"),
		Port:      port,
		AccountID: envOrDefault("IBKR_ACCOUNT_ID", "DU000000"),
		ClientID:  1,
	})
}

func (a *IBKRAdapter) PlaceOrder(ctx context.Context, req *broker.OrderRequest) (*broker.OrderResponse, error) {
	orderType := mapOrderType(req.Type)
	side := "BUY"
	if req.Side == broker.Sell {
		side = "SELL"
	}

	ibReq := ibkrOrderRequest{
		AcctID:    a.cfg.AccountID,
		SecType:   "STK",
		OrderType: orderType,
		Side:      side,
		Quantity:  req.Quantity,
		Tif:       "DAY",
	}
	if req.Type == broker.Limit {
		ibReq.Price = req.LimitPrice.Float64()
	}
	if req.Type == broker.Stop {
		ibReq.AuxPrice = req.StopPrice.Float64()
	}
	switch req.TimeInForce {
	case broker.GTC:
		ibReq.Tif = "GTC"
	case broker.IOC:
		ibReq.Tif = "IOC"
	}

	respBody, err := a.client.request(ctx, "POST", "/iserver/account/"+a.cfg.AccountID+"/orders", ibReq)
	if err != nil {
		return nil, fmt.Errorf("ibkr: place order: %w", err)
	}

	var orders []ibkrOrderResponse
	if err := json.Unmarshal(respBody, &orders); err != nil {
		var single ibkrOrderResponse
		if err2 := json.Unmarshal(respBody, &single); err2 != nil {
			return nil, fmt.Errorf("ibkr: parse order response: %w (body: %s)", err, string(respBody))
		}
		orders = []ibkrOrderResponse{single}
	}

	if len(orders) == 0 {
		return nil, fmt.Errorf("ibkr: no order in response")
	}

	o := orders[0]
	status := mapIBKRStatus(o.OrderStatus)

	resp := &broker.OrderResponse{
		BrokerOrderID: o.OrderID,
		Status:        status,
		FilledQty:     req.Quantity,
		AvgFillPrice:  req.LimitPrice,
	}

	a.mu.Lock()
	a.orderSeq++
	a.orders[o.OrderID] = resp
	if status == broker.Filled {
		a.fillTimes[o.OrderID] = time.Now()
	}
	a.mu.Unlock()

	return resp, nil
}

func (a *IBKRAdapter) CancelOrder(ctx context.Context, orderID string) error {
	path := fmt.Sprintf("/iserver/account/%s/order/%s", a.cfg.AccountID, orderID)
	_, err := a.client.request(ctx, "DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("ibkr: cancel order: %w", err)
	}

	a.mu.Lock()
	if order, ok := a.orders[orderID]; ok {
		order.Status = broker.Canceled
	}
	a.mu.Unlock()
	return nil
}

func (a *IBKRAdapter) CancelAllOrders(ctx context.Context) error {
	path := fmt.Sprintf("/iserver/account/%s/orders", a.cfg.AccountID)
	_, err := a.client.request(ctx, "DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("ibkr: cancel all orders: %w", err)
	}

	a.mu.Lock()
	for _, order := range a.orders {
		order.Status = broker.Canceled
	}
	a.mu.Unlock()
	return nil
}

func (a *IBKRAdapter) CloseAllPositions(ctx context.Context) error {
	positions, err := a.GetPositions(ctx)
	if err != nil {
		return err
	}
	for _, pos := range positions {
		if pos.Quantity == 0 {
			continue
		}
		side := broker.Sell
		if pos.Quantity < 0 {
			side = broker.Buy
		}
		qty := pos.Quantity
		if qty < 0 {
			qty = -qty
		}
		req := &broker.OrderRequest{
			Symbol:    pos.Symbol,
			Side:      side,
			Type:      broker.Market,
			Quantity:  qty,
			AccountID: a.cfg.AccountID,
		}
		if _, err := a.PlaceOrder(ctx, req); err != nil {
			return fmt.Errorf("ibkr: close position %s: %w", pos.Symbol, err)
		}
	}
	return nil
}

func (a *IBKRAdapter) GetPositions(ctx context.Context) ([]broker.Position, error) {
	respBody, err := a.client.request(ctx, "GET", "/iserver/account/positions", nil)
	if err != nil {
		return nil, fmt.Errorf("ibkr: get positions: %w", err)
	}

	var ibPositions []ibkrPosition
	if err := json.Unmarshal(respBody, &ibPositions); err != nil {
		return nil, fmt.Errorf("ibkr: parse positions: %w", err)
	}

	positions := make([]broker.Position, 0, len(ibPositions))
	for _, ip := range ibPositions {
		positions = append(positions, broker.Position{
			Symbol:        ip.ContractDesc,
			Quantity:      ip.Position,
			AvgEntryPrice: types.FromFloat64(ip.AvgCost),
			MarketValue:   types.FromFloat64(ip.MktValue),
			UnrealizedPL:  ip.UnrealizedPnl,
		})
	}
	return positions, nil
}

func (a *IBKRAdapter) GetAccount(ctx context.Context) (*broker.Account, error) {
	respBody, err := a.client.request(ctx, "GET", "/iserver/account/pnl/partitioned", nil)
	if err != nil {
		return nil, fmt.Errorf("ibkr: get account: %w", err)
	}

	var pnl ibkrPnlResponse
	if err := json.Unmarshal(respBody, &pnl); err != nil {
		return nil, fmt.Errorf("ibkr: parse pnl: %w", err)
	}

	if len(pnl.Total.ROW) == 0 {
		return &broker.Account{
			ID:     a.cfg.AccountID,
			Status: "ACTIVE",
		}, nil
	}

	row := pnl.Total.ROW[0]
	netLiq := types.FromFloat64(row.NetLiquidation)

	return &broker.Account{
		ID:          a.cfg.AccountID,
		Balance:     types.FromFloat64(row.NetLiquidation - row.UnrealizedPnL),
		Equity:      netLiq,
		BuyingPower: types.FromFloat64(row.NetLiquidation * 2),
		DailyPL:     row.DailyPnL,
		Status:      "ACTIVE",
	}, nil
}

func (a *IBKRAdapter) ValidateCredentials(ctx context.Context) error {
	return a.HealthCheck(ctx)
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
	if a.client == nil {
		return errors.New("ibkr: client not initialized")
	}
	return a.client.authenticate(ctx)
}

func (a *IBKRAdapter) PrepareOrder(ctx context.Context, req *broker.OrderRequest) (string, error) {
	orderType := mapOrderType(req.Type)
	side := "BUY"
	if req.Side == broker.Sell {
		side = "SELL"
	}

	ibReq := ibkrOrderRequest{
		AcctID:    a.cfg.AccountID,
		SecType:   "STK",
		OrderType: orderType,
		Side:      side,
		Quantity:  req.Quantity,
		Tif:       "DAY",
	}
	if req.Type == broker.Limit {
		ibReq.Price = req.LimitPrice.Float64()
	}

	respBody, err := a.client.request(ctx, "POST", "/iserver/account/"+a.cfg.AccountID+"/orders", map[string]interface{}{
		"orders":   []ibkrOrderRequest{ibReq},
		"transmit": false,
	})
	if err != nil {
		return "", fmt.Errorf("ibkr: prepare order: %w", err)
	}

	var orders []ibkrOrderResponse
	json.Unmarshal(respBody, &orders)
	if len(orders) == 0 {
		return "", fmt.Errorf("ibkr: no order in prepare response")
	}

	a.mu.Lock()
	a.orderSeq++
	a.mu.Unlock()
	return orders[0].OrderID, nil
}

func (a *IBKRAdapter) ConfirmOrder(ctx context.Context, brokerID string) (*broker.OrderResponse, error) {
	respBody, err := a.client.request(ctx, "POST", "/iserver/account/"+a.cfg.AccountID+"/order/"+brokerID, map[string]string{
		"transmit": "true",
	})
	if err != nil {
		return nil, fmt.Errorf("ibkr: confirm order: %w", err)
	}

	var o ibkrOrderResponse
	if err := json.Unmarshal(respBody, &o); err != nil {
		return nil, fmt.Errorf("ibkr: parse confirm response: %w", err)
	}

	resp := &broker.OrderResponse{
		BrokerOrderID: o.OrderID,
		Status:        mapIBKRStatus(o.OrderStatus),
	}
	return resp, nil
}

func (a *IBKRAdapter) IsTransactional() bool {
	return true
}

func (a *IBKRAdapter) GetFills(ctx context.Context, date string) ([]broker.TradeFill, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	targetDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("ibkr: invalid date: %w", err)
	}
	targetDay := targetDate.Truncate(24 * time.Hour)

	var fills []broker.TradeFill
	for orderID, order := range a.orders {
		if order.Status != broker.Filled {
			continue
		}
		ft, ok := a.fillTimes[orderID]
		if !ok {
			continue
		}
		if ft.Year() != targetDay.Year() || ft.YearDay() != targetDay.YearDay() {
			continue
		}
		fills = append(fills, broker.TradeFill{
			OrderID:       orderID,
			Symbol:        a.cfg.AccountID,
			Side:          broker.Buy,
			Quantity:      order.FilledQty,
			FillPrice:     order.AvgFillPrice,
			FillTime:      ft,
			BrokerOrderID: order.BrokerOrderID,
		})
	}
	return fills, nil
}

func mapOrderType(t broker.OrderType) string {
	switch t {
	case broker.Market:
		return "MKT"
	case broker.Limit:
		return "LMT"
	case broker.Stop:
		return "STP"
	default:
		return "MKT"
	}
}

func mapIBKRStatus(s string) broker.OrderStatus {
	switch s {
	case "Submitted", "PreSubmitted":
		return broker.New
	case "Filled":
		return broker.Filled
	case "Cancelled":
		return broker.Canceled
	default:
		return broker.New
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
