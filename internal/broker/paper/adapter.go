package paper

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lee-econ/orca-core/internal/broker"
	"github.com/lee-econ/orca-core/internal/types"
)

type PaperAdapter struct {
	mu           sync.RWMutex
	balance      float64
	equity       float64
	startingBalance float64
	positions    map[string]*broker.Position
	orders       map[string]*broker.OrderResponse
	lastPrices   map[string]types.Price
	orderCount   int
	feeConfig    broker.BrokerageFeeConfig
}

func NewAdapter(startingBalance float64) *PaperAdapter {
	return &PaperAdapter{
		balance:      startingBalance,
		equity:       startingBalance,
		startingBalance: startingBalance,
		positions:    make(map[string]*broker.Position),
		orders:       make(map[string]*broker.OrderResponse),
		lastPrices:   make(map[string]types.Price),
		feeConfig:    broker.DefaultBrokerageFee(),
	}
}

func (p *PaperAdapter) Manifest() broker.AdapterManifest {
	return broker.AdapterManifest{
		ID:         "paper",
		BrokerType: "paper",
		Priority:   1,
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

func (p *PaperAdapter) HealthCheck(ctx context.Context) error {
	p.mu.RLock()
	bal := p.balance
	p.mu.RUnlock()
	if bal < 0 {
		return fmt.Errorf("paper: negative balance %.2f", bal)
	}
	return nil
}

func (p *PaperAdapter) PlaceOrder(ctx context.Context, req *broker.OrderRequest) (*broker.OrderResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.orderCount++
	orderID := fmt.Sprintf("paper-%d", p.orderCount)

	resp := &broker.OrderResponse{
		BrokerOrderID: orderID,
		Status:        broker.Filled,
		FilledQty:     req.Quantity,
		AvgFillPrice:  req.LimitPrice,
	}
	if req.Type == broker.Market {
		if lastPrice, ok := p.lastPrices[req.Symbol]; ok {
			resp.AvgFillPrice = lastPrice
		} else {
			resp.AvgFillPrice = req.LimitPrice
			if resp.AvgFillPrice.IsZero() {
				resp.AvgFillPrice = types.FromFloat64(100.0)
			}
		}
	}

	p.lastPrices[req.Symbol] = resp.AvgFillPrice

	cost := resp.AvgFillPrice.MulFloat(req.Quantity).Float64()
	commission := p.feeConfig.CalculateFee(req.Quantity, resp.AvgFillPrice.Float64())

	p.orders[orderID] = resp

	pos, exists := p.positions[req.Symbol]
	if !exists {
		pos = &broker.Position{
			Symbol:        req.Symbol,
			Quantity:      0,
			AvgEntryPrice: types.Price(0),
			MarketValue:   types.Price(0),
			UnrealizedPL:  0,
		}
		p.positions[req.Symbol] = pos
	}

	if req.Side == broker.Buy {
		p.balance -= cost + commission
		p.equity -= commission
		newQty := pos.Quantity + req.Quantity
		if pos.Quantity == 0 {
			pos.AvgEntryPrice = resp.AvgFillPrice
		} else {
			pos.AvgEntryPrice = types.FromFloat64(
				(pos.AvgEntryPrice.Float64()*pos.Quantity + resp.AvgFillPrice.Float64()*req.Quantity) / newQty,
			)
		}
		pos.Quantity = newQty
	} else {
		p.balance += cost - commission
		p.equity -= commission
		newQty := pos.Quantity - req.Quantity
		if newQty <= 0 {
			pos.AvgEntryPrice = types.Price(0)
			newQty = 0
		}
		pos.Quantity = newQty
	}

	pos.MarketValue = types.FromFloat64(pos.Quantity * resp.AvgFillPrice.Float64())
	if pos.Quantity > 0 {
		pos.UnrealizedPL = pos.MarketValue.Float64() - (pos.AvgEntryPrice.Float64() * pos.Quantity)
	} else {
		pos.UnrealizedPL = 0
	}

	p.equity = p.balance + pos.UnrealizedPL
	for _, pn := range p.positions {
		if pn.Symbol != pos.Symbol {
			p.equity += pn.UnrealizedPL
		}
	}

	return resp, nil
}

func (p *PaperAdapter) CancelOrder(ctx context.Context, orderID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if order, ok := p.orders[orderID]; ok {
		order.Status = broker.Canceled
	}
	return nil
}

func (p *PaperAdapter) CancelAllOrders(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, order := range p.orders {
		if order.Status == broker.New || order.Status == broker.PartiallyFilled {
			order.Status = broker.Canceled
		}
	}
	return nil
}

func (p *PaperAdapter) CloseAllPositions(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for key, pos := range p.positions {
		p.equity += pos.UnrealizedPL
		p.balance += pos.UnrealizedPL
		delete(p.positions, key)
	}
	return nil
}

func (p *PaperAdapter) GetPositions(ctx context.Context) ([]broker.Position, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make([]broker.Position, 0, len(p.positions))
	for _, pos := range p.positions {
		if pos.Quantity != 0 {
			result = append(result, *pos)
		}
	}
	return result, nil
}

func (p *PaperAdapter) GetAccount(ctx context.Context) (*broker.Account, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	unrealizedPL := 0.0
	marketValue := 0.0
	for _, pos := range p.positions {
		unrealizedPL += pos.UnrealizedPL
		marketValue += pos.MarketValue.Float64()
	}
	dailyPL := p.balance + unrealizedPL - p.startingBalance

	accountID := fmt.Sprintf("paper-%d", time.Now().Unix())
	return &broker.Account{
		ID:          accountID,
		Balance:     types.FromFloat64(p.balance),
		Equity:      types.FromFloat64(p.balance + unrealizedPL),
		BuyingPower: types.FromFloat64((p.balance + unrealizedPL) * 2),
		DailyPL:     dailyPL,
		Status:      "ACTIVE",
	}, nil
}

func (p *PaperAdapter) ValidateCredentials(ctx context.Context) error {
	return nil
}

func (p *PaperAdapter) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.positions = make(map[string]*broker.Position)
	p.orders = make(map[string]*broker.OrderResponse)
	p.equity = p.startingBalance
	p.balance = p.startingBalance
}

func (p *PaperAdapter) PrepareOrder(ctx context.Context, req *broker.OrderRequest) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.orderCount++
	orderID := fmt.Sprintf("paper-tx-%d", p.orderCount)

	resp := &broker.OrderResponse{
		BrokerOrderID: orderID,
		Status:        broker.New,
		FilledQty:     0,
		AvgFillPrice:  req.LimitPrice,
	}
	if req.Type == broker.Market {
		if lastPrice, ok := p.lastPrices[req.Symbol]; ok {
			resp.AvgFillPrice = lastPrice
		} else {
			resp.AvgFillPrice = types.FromFloat64(100.0)
		}
	}

	p.orders[orderID] = resp
	return orderID, nil
}

func (p *PaperAdapter) ConfirmOrder(ctx context.Context, brokerID string) (*broker.OrderResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	order, ok := p.orders[brokerID]
	if !ok {
		return nil, fmt.Errorf("paper: order %s not found", brokerID)
	}

	p.orderCount++
	fillOrderID := fmt.Sprintf("paper-%d", p.orderCount)
	resp := &broker.OrderResponse{
		BrokerOrderID: fillOrderID,
		Status:        broker.Filled,
		FilledQty:     order.FilledQty,
		AvgFillPrice:  order.AvgFillPrice,
	}
	if resp.FilledQty <= 0 {
		resp.FilledQty = 100
	}

	p.lastPrices[resp.BrokerOrderID] = resp.AvgFillPrice
	p.orders[fillOrderID] = resp

	cost := resp.AvgFillPrice.MulFloat(resp.FilledQty).Float64()
	commission := p.feeConfig.CalculateFee(resp.FilledQty, resp.AvgFillPrice.Float64())
	p.balance -= cost + commission
	p.equity -= commission

	return resp, nil
}

func (p *PaperAdapter) IsTransactional() bool {
	return true
}
