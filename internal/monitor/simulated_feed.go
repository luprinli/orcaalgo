package monitor

import (
	"context"
	"math/rand"
	"os"
	"time"

	"github.com/lee-econ/orca-core/internal/types"
)

type SimulatedFeed struct {
	hub     *WSHub
	symbols []SimSymbol
	rng     *rand.Rand
}

type SimSymbol struct {
	Ticker string
	Price  types.Price
	Volume float64
}

func NewSimulatedFeed(hub *WSHub) *SimulatedFeed {
	return &SimulatedFeed{
		hub: hub,
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
		symbols: []SimSymbol{
			{Ticker: "SPY", Price: types.FromFloat64(580.0), Volume: 500},
			{Ticker: "QQQ", Price: types.FromFloat64(475.0), Volume: 400},
			{Ticker: "AAPL", Price: types.FromFloat64(225.0), Volume: 300},
			{Ticker: "NVDA", Price: types.FromFloat64(1210.0), Volume: 600},
			{Ticker: "TSLA", Price: types.FromFloat64(345.0), Volume: 350},
			{Ticker: "MSFT", Price: types.FromFloat64(445.0), Volume: 280},
		},
	}
}

func (f *SimulatedFeed) Run(ctx context.Context) {
	if os.Getenv("ORCA_SIM_FEED") != "true" {
		return
	}
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			f.tick()
		}
	}
}

func (f *SimulatedFeed) tick() {
	for i := range f.symbols {
		s := &f.symbols[i]
		change := (f.rng.Float64() - 0.5) * s.Price.Float64() * 0.001
		s.Price = types.FromFloat64(s.Price.Float64() + change)
		if s.Price.Float64() < 10 { s.Price = types.FromFloat64(10) }
		vol := s.Volume * (0.5 + f.rng.Float64())
		side := "BUY"
		if change < 0 { side = "SELL" }
		f.hub.Broadcast("ticks", map[string]interface{}{
			"symbol": s.Ticker, "price": s.Price.Float64(), "volume": vol,
			"side": side, "time": time.Now().Format(time.RFC3339),
		})
	}
}