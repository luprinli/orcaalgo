package risk

import (
	"context"
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
	Suspended   bool
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
	mu            sync.RWMutex
	poolState     propfirm.PoolState
	accountID     string
	profile       *propfirm.Profile
	state         *propfirm.State
	strategies    map[string]*StrategyAllocation
	positionSizer *PositionSizer
}

func NewCapitalPoolManager(profile *propfirm.Profile, state *propfirm.State) *CapitalPoolManager {
	startingBalance := 100000.0
	if state != nil {
		startingBalance = state.StartingBalance
	}
	cpm := &CapitalPoolManager{
		profile:       profile,
		state:         state,
		strategies:    make(map[string]*StrategyAllocation),
		positionSizer: NewPositionSizer(profile),
	}
	propfirm.InitPoolState(&cpm.poolState, startingBalance)
	return cpm
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
		c.poolState.TotalBalance = state.StartingBalance + state.CumulativePnL
		if c.poolState.TotalBalance > c.poolState.TotalPeakBalance {
			c.poolState.TotalPeakBalance = c.poolState.TotalBalance
		}
	}
}

func (c *CapitalPoolManager) RequestCapital(ctx context.Context, req CapitalRequest) CapitalResult {
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
			PeakBalance: c.poolState.TotalBalance,
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
	for _, s := range c.strategies {
		totalOpen += len(s.OpenLong) + len(s.OpenShort)
	}
	if totalOpen >= p.MaxOpenPositions {
		return CapitalResult{ApprovedSize: 0, Reason: "max_open_positions"}
	}

	if c.state != nil {
		// Daily loss is measured from the day's opening balance (TotalBalance
		// minus today's PnL), NOT from the inception StartingBalance — otherwise
		// the check degrades into a cumulative-loss-from-inception halt and the
		// daily reset becomes meaningless (parity with PropFirmEnforcer).
		dayStart := c.poolState.TotalBalance - c.poolState.DailyPnL
		if dayStart > 0 && propfirm.DailyLossExceeded(dayStart, c.poolState.TotalBalance, p.MaxDailyLossPct) {
			return CapitalResult{ApprovedSize: 0, Reason: "daily_loss_limit"}
		}
	}

	stratDrawdown := propfirm.DrawdownPct(strat.PeakBalance, c.poolState.TotalBalance)
	maxStratDD := p.MaxDrawdownPct * 0.5
	if stratDrawdown > maxStratDD {
		strat.Suspended = true
		return CapitalResult{ApprovedSize: 0, Reason: "per_strategy_drawdown_suspended"}
	}

	if strat.Suspended {
		return CapitalResult{ApprovedSize: 0, Reason: "strategy_suspended"}
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
			PeakBalance: c.poolState.TotalBalance,
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

	c.poolState.TotalBalance += pnl
	if c.poolState.TotalBalance > s.PeakBalance {
		s.PeakBalance = c.poolState.TotalBalance
	}
	propfirm.UpdatePeakBalance(&c.poolState)

	if c.poolState.TotalBalance > 0 {
		s.DrawdownPct = propfirm.DrawdownPct(s.PeakBalance, c.poolState.TotalBalance)
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

// ResumeStrategy clears the Suspended flag for a strategy and resets its
// peak balance to the current total balance so it can trade again.
func (c *CapitalPoolManager) ResumeStrategy(strategyID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if s, ok := c.strategies[strategyID]; ok {
		s.Suspended = false
		s.PeakBalance = c.poolState.TotalBalance
	}
}

// SuspendedStrategies returns the IDs of all currently suspended strategies.
func (c *CapitalPoolManager) SuspendedStrategies() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var ids []string
	for id, s := range c.strategies {
		if s.Suspended {
			ids = append(ids, id)
		}
	}
	return ids
}

func (c *CapitalPoolManager) TotalBalance() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.poolState.TotalBalance
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

// Halted returns true when the pool's prop-firm state has been violated.
func (c *CapitalPoolManager) Halted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.state == nil {
		return false
	}
	return c.state.Violated
}

// HaltReason returns the violation reason from the prop-firm state.
func (c *CapitalPoolManager) HaltReason() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.state == nil {
		return ""
	}
	return c.state.ViolationReason
}

var _ CapitalGate = (*CapitalPoolManager)(nil)

// HasOpenPosition returns true if any strategy has an open position on the
// given symbol + side. Used for cross-strategy correlation braking.
func (c *CapitalPoolManager) HasOpenPosition(symbol, side string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, s := range c.strategies {
		if side == "BUY" && s.OpenLong[symbol] > 0 {
			return true
		}
		if side == "SELL" && s.OpenShort[symbol] > 0 {
			return true
		}
	}
	return false
}

