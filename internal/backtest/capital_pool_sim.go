package backtest

import (
	"context"

	"github.com/lee-econ/orca-core/internal/propfirm"
	"github.com/lee-econ/orca-core/internal/risk"
)

type PoolSimStrategy struct {
	ID          string
	Allocated   float64
	dailyPnL    float64
	PeakBalance float64
	DrawdownPct float64
	OpenLong    map[string]float64
	OpenShort   map[string]float64
	Runner      StrategyRunnerInterface
}

type StrategyRunnerInterface interface {
	Name() string
	Type() string
	Evaluate(candle Candle, regime int8) *Signal
	Reset()
}

type CapitalPoolSim struct {
	propfirm.PoolState
	Profile    *propfirm.Profile
	Strategies map[string]*PoolSimStrategy
}

func NewCapitalPoolSim(profile *propfirm.Profile, startingBalance float64) *CapitalPoolSim {
	c := &CapitalPoolSim{
		Profile:    profile,
		Strategies: make(map[string]*PoolSimStrategy),
	}
	propfirm.InitPoolState(&c.PoolState, startingBalance)
	return c
}

func (c *CapitalPoolSim) AddStrategy(id string, runner StrategyRunnerInterface) {
	c.Strategies[id] = &PoolSimStrategy{
		ID:          id,
		PeakBalance: c.PoolState.TotalBalance,
		OpenLong:    make(map[string]float64),
		OpenShort:   make(map[string]float64),
		Runner:      runner,
	}
}

func (c *CapitalPoolSim) EvaluateAll(candle Candle, regime int8) map[string]*Signal {
	signals := make(map[string]*Signal)

	if c.PoolState.Halted {
		return signals
	}

	for id, strat := range c.Strategies {
		if strat.Runner == nil {
			continue
		}
		signal := strat.Runner.Evaluate(candle, regime)
		if signal == nil {
			continue
		}

		p := c.Profile
		if p == nil {
			p = propfirm.DefaultFTMOProfile()
		}

		if c.PoolState.DailyPnLPct <= -p.MaxDailyLossPct {
			c.PoolState.Halted = true
			c.PoolState.HaltReason = "daily_loss_limit"
			continue
		}

		stratDD := propfirm.DrawdownPct(strat.PeakBalance, c.PoolState.TotalBalance)
		maxStratDD := p.MaxDrawdownPct * 0.5
		if stratDD > maxStratDD {
			continue
		}

		totalOpen := 0
		for _, s := range c.Strategies {
			totalOpen += len(s.OpenLong) + len(s.OpenShort)
		}
		if totalOpen >= p.MaxOpenPositions {
			continue
		}

		bufferPct := 0.002
		correlationMult := 1.0
		if signal.Side == "BUY" && strat.OpenShort[candle.Symbol] > 0 {
			correlationMult = 0.5
		}
		if signal.Side == "SELL" && strat.OpenLong[candle.Symbol] > 0 {
			correlationMult = 0.5
		}

		positionPct := p.MaxPositionPct / 100.0
		regimeMult := 1.0
		if regime >= 0 && regime < 4 {
			regimeMult = p.RegimeMultipliers[regime]
		}

		quantity := (c.PoolState.TotalBalance * positionPct * regimeMult * correlationMult) / candle.Close.Float64()
		maxSize := c.PoolState.TotalBalance * p.MaxPositionPct / 100.0
		if quantity > maxSize/candle.Close.Float64() {
			quantity = maxSize / candle.Close.Float64()
		}
		if quantity < 1 {
			continue
		}

		signal.Quantity *= (1.0 - bufferPct)

		strat.Allocated += signal.Quantity * candle.Close.Float64()
		if signal.Side == "BUY" {
			strat.OpenLong[candle.Symbol] += signal.Quantity
		} else {
			strat.OpenShort[candle.Symbol] += signal.Quantity
		}

		signals[id] = signal
	}

	return signals
}

func (c *CapitalPoolSim) RecordFill(strategyID, symbol, side string, pnl float64, quantity float64) {
	s := c.Strategies[strategyID]
	if s == nil {
		return
	}

	s.dailyPnL += pnl
	s.Allocated -= quantity * 100.0
	if s.Allocated < 0 {
		s.Allocated = 0
	}

	if side == "BUY" {
		s.OpenLong[symbol] -= quantity
		if s.OpenLong[symbol] <= 0 {
			delete(s.OpenLong, symbol)
		}
	} else {
		s.OpenShort[symbol] -= quantity
		if s.OpenShort[symbol] <= 0 {
			delete(s.OpenShort, symbol)
		}
	}

	propfirm.RecordPoolPnL(&c.PoolState, pnl, c.PoolState.TotalBalance-c.PoolState.DailyPnL)
	if c.PoolState.TotalBalance > s.PeakBalance {
		s.PeakBalance = c.PoolState.TotalBalance
	}
	propfirm.CheckDrawdownHalt(&c.PoolState, c.Profile)
}

func (c *CapitalPoolSim) ResetDaily() {
	propfirm.ResetPoolDaily(&c.PoolState)
	for _, s := range c.Strategies {
		s.dailyPnL = 0
	}
}

// --- CapitalGate adapter methods ---

// RequestCapital is an adapter for the risk.CapitalGate interface. In the
// backtest path, capital authorization happens inline inside EvaluateAll;
// this adapter provides a post-hoc entry point for the RiskPipeline.
func (c *CapitalPoolSim) RequestCapital(ctx context.Context, req risk.CapitalRequest) risk.CapitalResult {
	if c.PoolState.Halted {
		return risk.CapitalResult{ApprovedSize: 0, Reason: "pool_halted"}
	}

	p := c.Profile
	if p == nil {
		p = propfirm.DefaultFTMOProfile()
	}

	strat, ok := c.Strategies[req.StrategyID]
	if !ok {
		return risk.CapitalResult{ApprovedSize: req.BaseSize, Reason: "ok"}
	}

	if req.Side == "BUY" && strat.OpenLong[req.Symbol] > 0 {
		return risk.CapitalResult{ApprovedSize: req.BaseSize * 0.5, Reason: "correlation_limit"}
	}
	if req.Side == "SELL" && strat.OpenShort[req.Symbol] > 0 {
		return risk.CapitalResult{ApprovedSize: req.BaseSize * 0.5, Reason: "correlation_limit"}
	}

	totalOpen := 0
	for _, s := range c.Strategies {
		totalOpen += len(s.OpenLong) + len(s.OpenShort)
	}
	if totalOpen >= p.MaxOpenPositions {
		return risk.CapitalResult{ApprovedSize: 0, Reason: "max_open_positions"}
	}

	if c.PoolState.TotalBalance <= 0 {
		return risk.CapitalResult{ApprovedSize: 0, Reason: "no_capital"}
	}

	size := req.BaseSize
	maxSize := c.PoolState.TotalBalance * p.MaxPositionPct / 100.0 / 100.0
	if size > maxSize {
		size = maxSize
	}
	if size > c.PoolState.TotalBalance*0.25 {
		size = c.PoolState.TotalBalance * 0.25
	}

	return risk.CapitalResult{ApprovedSize: size, Reason: "ok"}
}

// Halted returns whether the capital pool has been halted.
func (c *CapitalPoolSim) Halted() bool { return c.PoolState.Halted }

// HaltReason returns the reason the pool was halted.
func (c *CapitalPoolSim) HaltReason() string { return c.PoolState.HaltReason }

// TotalBalance returns the current pool equity.
func (c *CapitalPoolSim) TotalBalance() float64 { return c.PoolState.TotalBalance }

var _ risk.CapitalGate = (*CapitalPoolSim)(nil)

