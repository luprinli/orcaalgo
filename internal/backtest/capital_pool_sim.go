package backtest

import (
	"github.com/lee-econ/orca-core/internal/propfirm"
)

type PoolSimStrategy struct {
	ID          string
	Allocated   float64
	DailyPnL    float64
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
	Profile          *propfirm.Profile
	TotalBalance     float64
	TotalPeakBalance float64
	DailyPnL         float64
	DailyPnLPct      float64
	Strategies       map[string]*PoolSimStrategy
	Halted           bool
	HaltReason       string
	TradingDays      int
}

func NewCapitalPoolSim(profile *propfirm.Profile, startingBalance float64) *CapitalPoolSim {
	return &CapitalPoolSim{
		Profile:          profile,
		TotalBalance:     startingBalance,
		TotalPeakBalance: startingBalance,
		Strategies:       make(map[string]*PoolSimStrategy),
	}
}

func (c *CapitalPoolSim) AddStrategy(id string, runner StrategyRunnerInterface) {
	c.Strategies[id] = &PoolSimStrategy{
		ID:          id,
		PeakBalance: c.TotalBalance,
		OpenLong:    make(map[string]float64),
		OpenShort:   make(map[string]float64),
		Runner:      runner,
	}
}

func (c *CapitalPoolSim) EvaluateAll(candle Candle, regime int8) map[string]*Signal {
	signals := make(map[string]*Signal)

	if c.Halted {
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

		if c.DailyPnLPct <= -p.MaxDailyLossPct {
			c.Halted = true
			c.HaltReason = "daily_loss_limit"
			continue
		}

		stratDD := propfirm.DrawdownPct(strat.PeakBalance, c.TotalBalance)
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

		quantity := (c.TotalBalance * positionPct * regimeMult * correlationMult) / candle.Close.Float64()
		maxSize := c.TotalBalance * p.MaxPositionPct / 100.0
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

	s.DailyPnL += pnl
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

	c.TotalBalance += pnl
	c.DailyPnL += pnl
	if c.TotalBalance > c.TotalPeakBalance {
		c.TotalPeakBalance = c.TotalBalance
	}
	if c.TotalBalance > s.PeakBalance {
		s.PeakBalance = c.TotalBalance
	}

	if c.Profile != nil && c.TotalBalance > 0 {
		startBalance := c.TotalBalance - c.DailyPnL
		if startBalance > 0 {
			c.DailyPnLPct = c.DailyPnL / startBalance * 100.0
		}
	}

	dd := propfirm.DrawdownPct(c.TotalPeakBalance, c.TotalBalance)
	if c.Profile != nil && c.Profile.MaxDrawdownPct > 0 && dd > c.Profile.MaxDrawdownPct {
		c.Halted = true
		c.HaltReason = "max_drawdown"
	}
}

func (c *CapitalPoolSim) ResetDaily() {
	c.DailyPnL = 0
	c.DailyPnLPct = 0
	c.TradingDays++
	for _, s := range c.Strategies {
		s.DailyPnL = 0
	}
}
