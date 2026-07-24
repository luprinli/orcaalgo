package reactive

import (
	"context"
	"sync"
	"time"

	"github.com/lee-econ/orca-core/internal/types"
)

type SignalState string

const (
	StateIdle      SignalState = "idle"
	StateScheduled SignalState = "scheduled"
	StateOpened    SignalState = "opened"
	StateActive    SignalState = "active"
	StateClosed    SignalState = "closed"
	StateRejected  SignalState = "rejected"
)

type Signal struct {
	ID          string      `json:"id"`
	Symbol      string      `json:"symbol"`
	StrategyID  string      `json:"strategy_id"`
	Side        string      `json:"side"`
	Quantity    float64     `json:"quantity"`
	LimitPrice  types.Price `json:"limit_price"`
	StopPrice   types.Price `json:"stop_price"`
	State       SignalState `json:"state"`
	EntryPrice  types.Price `json:"entry_price,omitempty"`
	ExitPrice   types.Price `json:"exit_price,omitempty"`
	PnL         float64     `json:"pnl,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	Regime      int8        `json:"regime"`
	Confidence  float64     `json:"confidence"`
}

type SignalGenerator func(ctx context.Context, symbol string, price float64) (*Signal, error)

type SignalProcessor interface {
	OnOpen(signal *Signal) error
	OnClose(signal *Signal) error
	OnUpdate(signal *Signal) error
}

type ReactiveRunner interface {
	GenerateSignal(ctx context.Context, symbol string, price float64) (*Signal, error)
	ProcessSignal(signal *Signal) error
	Name() string
	Type() string
}

type EventBus struct {
	mu           sync.RWMutex
	subscribers  map[string][]func(signal *Signal)
	signals      map[string]*Signal
	signalOrder  []string
	maxSignals   int
}

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]func(signal *Signal)),
		signals:     make(map[string]*Signal),
		maxSignals:  1000,
	}
}

func (eb *EventBus) Subscribe(channel string, handler func(signal *Signal)) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.subscribers[channel] = append(eb.subscribers[channel], handler)
}

func (eb *EventBus) Publish(signal *Signal) {
	eb.mu.Lock()
	signal.UpdatedAt = time.Now()
	eb.signals[signal.ID] = signal
	eb.signalOrder = append(eb.signalOrder, signal.ID)
	if len(eb.signalOrder) > eb.maxSignals {
		eb.signalOrder = eb.signalOrder[len(eb.signalOrder)-eb.maxSignals:]
	}
	eb.mu.Unlock()

	eb.mu.RLock()
	handlers := eb.subscribers["signal_lifecycle"]
	eb.mu.RUnlock()

	for _, h := range handlers {
		h(signal)
	}
}

func (eb *EventBus) GetSignals(limit int) []*Signal {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	if limit <= 0 || limit > len(eb.signalOrder) {
		limit = len(eb.signalOrder)
	}
	start := len(eb.signalOrder) - limit
	if start < 0 {
		start = 0
	}

	result := make([]*Signal, 0, limit)
	for i := start; i < len(eb.signalOrder); i++ {
		if s, ok := eb.signals[eb.signalOrder[i]]; ok {
			result = append(result, s)
		}
	}
	return result
}

type LegacyAdapter struct {
	runner interface {
		Name() string
		Type() string
		Evaluate(candle struct {
			Time   time.Time
			Open   types.Price
			High   types.Price
			Low    types.Price
			Close  types.Price
			Volume float64
			Symbol string
		}, regime int8) *struct {
			Side     string
			Quantity float64
			Price    types.Price
			StopLoss types.Price
		}
	}
	signalCount int
}

func NewLegacyAdapter(runner interface {
	Name() string
	Type() string
	Evaluate(candle struct {
		Time   time.Time
		Open   types.Price
		High   types.Price
		Low    types.Price
		Close  types.Price
		Volume float64
		Symbol string
	}, regime int8) *struct {
		Side     string
		Quantity float64
		Price    types.Price
		StopLoss types.Price
	}
}) *LegacyAdapter {
	return &LegacyAdapter{runner: runner}
}

func (la *LegacyAdapter) GenerateSignal(ctx context.Context, symbol string, price float64) (*Signal, error) {
	candle := struct {
		Time   time.Time
		Open   types.Price
		High   types.Price
		Low    types.Price
		Close  types.Price
		Volume float64
		Symbol string
	}{
		Time:   time.Now(),
		Open:   types.PriceFromFloat(price),
		High:   types.PriceFromFloat(price),
		Low:    types.PriceFromFloat(price),
		Close:  types.PriceFromFloat(price),
		Volume: 0,
		Symbol: symbol,
	}

	result := la.runner.Evaluate(candle, 0)
	if result == nil || result.Side == "" {
		return nil, nil
	}

	la.signalCount++
	signal := &Signal{
		ID:         symbol + "-sig-" + string(rune('0'+la.signalCount%10)),
		Symbol:     symbol,
		StrategyID: la.runner.Name(),
		Side:       result.Side,
		Quantity:   result.Quantity,
		LimitPrice: result.Price,
		StopPrice:  result.StopLoss,
		State:      StateScheduled,
		CreatedAt:  time.Now(),
	}
	return signal, nil
}

func (la *LegacyAdapter) ProcessSignal(signal *Signal) error {
	return nil
}

func (la *LegacyAdapter) Name() string {
	return la.runner.Name()
}

func (la *LegacyAdapter) Type() string {
	return la.runner.Type()
}
