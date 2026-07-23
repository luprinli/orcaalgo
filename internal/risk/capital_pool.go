package risk

import (
	"sync"

	"github.com/lee-econ/orca-core/internal/propfirm"
)

type StrategyAllocation struct {
	StrategyID  string
	Allocated   float64
	Exposure    float64
	DailyPnL    float64
	PeakBalance float64
	DrawdownPct float64
	OpenLong    map[string]float64
	OpenShort   map[string]float64
}

type CapitalRequest struct {
	StrategyID string
	Confidence float64
	Symbol     string
	Side       string
	BaseSize   float64
}

type CapitalResult struct {
	ApprovedSize float64
	Reason       string
}

type CapitalPoolManager struct {
	mu               sync.RWMutex
	accountID        string
	profile          *propfirm.Profile
	state            *propfirm.State
	totalBalance     float64
	totalPeakBalance float64
	strategies       map[string]*StrategyAllocation
	positionSizer    *PositionSizer
}

func NewCapitalPoolManager(profile *propfirm.Profile, state *propfirm.State) *CapitalPoolManager {
	startingBalance := 100000.0
	if state != nil {
		startingBalance = state.StartingBalance
	}
	return &CapitalPoolManager{
		profile:          profile,
		state:            state,
		totalBalance:     startingBalance,
		totalPeakBalance: startingBalance,
		strategies:       make(map[string]*StrategyAllocation),
		positionSizer:    NewPositionSizer(profile),
	}
}

func NewCapitalPoolManagerWithAccount(accountID string, profile *propfirm.Profile, state *propfirm.State) *CapitalPoolManager {
	cpm := NewCapitalPoolManager(profile, state)
	cpm.accountID = accountID
	return cpm
}

func (c *CapitalPoolManager) AccountID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.accountID
}

func (c *CapitalPoolManager) SetAccountID(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.accountID = id
}

func (c *CapitalPoolManager) SetProfile(profile *propfirm.Profile) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.profile = profile
	c.positionSizer = NewPositionSizer(profile)
}

func (c *CapitalPoolManager) UpdateState(state *propfirm.State) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = state
	if state != nil {
		c.totalBalance = state.StartingBalance + state.CumulativePnL
		if c.totalBalance > c.totalPeakBalance {
			c.totalPeakBalance = c.totalBalance
		}
	}
}

func (c *CapitalPoolManager) RequestCapital(req CapitalRequest) CapitalResult {
	c.mu.Lock()
	defer c.mu.Unlock()

	p := c.profile
	if p == nil {
		p = propfirm.DefaultFTMOProfile()
	}

	strat, ok := c.strategies[req.StrategyID]
	if !ok {
		strat = &StrategyAllocation{
			StrategyID:  req.StrategyID,
			OpenLong:    make(map[string]float64),
			OpenShort:   make(map[string]float64),
			PeakBalance: c.totalBalance,
		}
		c.strategies[req.StrategyID] = strat
	}

	if c.state != nil && c.state.Violated {
		return CapitalResult{ApprovedSize: 0, Reason: "prop_firm_violated"}
	}

	if req.Side == "BUY" && strat.OpenLong[req.Symbol] > 0 {
		existing := strat.OpenLong[req.Symbol]
		size := c.positionSizer.ComputeSize(req.Confidence, req.BaseSize, req.Symbol, strat.Allocated, existing)
		if size <= 0 {
			return CapitalResult{ApprovedSize: 0, Reason: "correlation_limit"}
		}
		return CapitalResult{ApprovedSize: size, Reason: "ok"}
	}
	if req.Side == "SELL" && strat.OpenShort[req.Symbol] > 0 {
		existing := strat.OpenShort[req.Symbol]
		size := c.positionSizer.ComputeSize(req.Confidence, req.BaseSize, req.Symbol, strat.Allocated, existing)
		if size <= 0 {
			return CapitalResult{ApprovedSize: 0, Reason: "correlation_limit"}
		}
		return CapitalResult{ApprovedSize: size, Reason: "ok"}
	}

	totalOpen := 0
	longCount, shortCount := 0, 0
	for _, s := range c.strategies {
		for range s.OpenLong {
			longCount++
		}
		for range s.OpenShort {
			shortCount++
		}
		totalOpen += longCount + shortCount
	}
	if totalOpen >= p.MaxOpenPositions {
		return CapitalResult{ApprovedSize: 0, Reason: "max_open_positions"}
	}

	if c.state != nil {
		if propfirm.DailyLossExceeded(c.state.StartingBalance, c.totalBalance, p.MaxDailyLossPct) {
			return CapitalResult{ApprovedSize: 0, Reason: "daily_loss_limit"}
		}
	}

	stratDrawdown := propfirm.DrawdownPct(strat.PeakBalance, c.totalBalance)
	maxStratDD := p.MaxDrawdownPct * 0.5
	if stratDrawdown > maxStratDD {
		return CapitalResult{ApprovedSize: 0, Reason: "per_strategy_drawdown"}
	}

	size := c.positionSizer.ComputeSize(req.Confidence, req.BaseSize, req.Symbol, strat.Allocated, 0)
	if size <= 0 {
		return CapitalResult{ApprovedSize: 0, Reason: "size_zero"}
	}

	strat.Allocated += size
	if req.Side == "BUY" {
		strat.OpenLong[req.Symbol] += size
	} else {
		strat.OpenShort[req.Symbol] += size
	}

	return CapitalResult{ApprovedSize: size, Reason: "ok"}
}

func (c *CapitalPoolManager) RecordFill(strategyID, symbol, side string, pnl float64, quantity float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	s := c.strategies[strategyID]
	if s == nil {
		s = &StrategyAllocation{
			StrategyID:  strategyID,
			OpenLong:    make(map[string]float64),
			OpenShort:   make(map[string]float64),
			PeakBalance: c.totalBalance,
		}
		c.strategies[strategyID] = s
	}

	s.DailyPnL += pnl
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
	s.Allocated -= quantity
	if s.Allocated < 0 {
		s.Allocated = 0
	}

	c.totalBalance += pnl
	if c.totalBalance > s.PeakBalance {
		s.PeakBalance = c.totalBalance
	}
	if c.totalBalance > c.totalPeakBalance {
		c.totalPeakBalance = c.totalBalance
	}

	if c.totalBalance > 0 {
		s.DrawdownPct = propfirm.DrawdownPct(s.PeakBalance, c.totalBalance)
	}
}

func (c *CapitalPoolManager) ResetDaily() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, s := range c.strategies {
		s.DailyPnL = 0
	}
}

func (c *CapitalPoolManager) StrategyMetrics() []StrategyAllocation {
	c.mu.RLock()
	defer c.mu.RUnlock()

	metrics := make([]StrategyAllocation, 0, len(c.strategies))
	for _, s := range c.strategies {
		metrics = append(metrics, *s)
	}
	return metrics
}

func (c *CapitalPoolManager) TotalBalance() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.totalBalance
}

func (c *CapitalPoolManager) TotalExposure() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var exposure float64
	for _, s := range c.strategies {
		exposure += s.Allocated
	}
	return exposure
}
