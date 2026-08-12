package risk

import "sync"

// SignalGateImpl is the canonical, shared implementation of SignalGate used by
// both the backtest Engine and the live LiveEngine. It centralises volatility
// halting, regime-aware position sizing, Kelly multiplier application, exposure
// tracking, and rate limiting into one concrete type so that both engines
// produce identical sizing for identical inputs.
//
// Architecture (enforced by ProcessSignal order in RiskPipeline):
//   1. ValidateSignal  — volatility halt, capital positivity, rate limit
//   2. ApplySizing     — Kelly, regime, VIX, sentiment, confidence
//   3. CheckExposure   — max leverage, symbol concentration
//   4. RecordExposure  — register open notional
//   5. RemoveExposure  — deregister closed notional
type SignalGateImpl struct {
	mu sync.RWMutex

	volHalt    *VolatilityHalt
	sizer      *PositionSizer
	exposure   *ExposureTracker
	rateLimit  *OrderRateLimiter

	// backtestSkipRateLimit controls whether the wall-clock rate limiter is
	// enforced. In backtest mode years of simulated time are compressed into
	// milliseconds of real time, so a wall-clock limiter would reject nearly
	// all valid signals. Set to true when running in a backtest context.
	backtestSkipRateLimit bool
}

// NewSignalGateImpl creates a SignalGateImpl wired to the provided components.
// All arguments are required; pass nil for rateLimit only in backtest mode
// where backtestSkipRateLimit is true.
func NewSignalGateImpl(
	volHalt *VolatilityHalt,
	sizer *PositionSizer,
	exposure *ExposureTracker,
	rateLimit *OrderRateLimiter,
) *SignalGateImpl {
	return &SignalGateImpl{
		volHalt:   volHalt,
		sizer:     sizer,
		exposure:  exposure,
		rateLimit: rateLimit,
	}
}

func (g *SignalGateImpl) SetBacktestMode(v bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.backtestSkipRateLimit = v
}

func (g *SignalGateImpl) SetVIX(vix float64) {
	g.sizer.UpdateMarketState(vix, 50, g.sizer.regime)
}

func (g *SignalGateImpl) SetRegime(regime int8) {
	g.sizer.UpdateMarketState(g.sizer.vix, g.sizer.sentiment, regime)
}

func (g *SignalGateImpl) SetEquity(equity float64) {
	g.exposure.SetEquity(equity)
}

func (g *SignalGateImpl) SetKellyMultiplier(mult float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.sizer.SetKellyMultiplier(mult)
}

func (g *SignalGateImpl) SetRegimeScore(score float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.sizer.SetRegimeScore(score)
}

// --- SignalGate interface ---

// ValidateSignal performs pre-screening: volatility halt, capital positivity,
// rate limiting.
func (g *SignalGateImpl) ValidateSignal(runningCapital float64) (bool, string) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if runningCapital <= 0 {
		return false, "capital_zero"
	}

	if g.volHalt != nil && g.volHalt.IsHalted() {
		return false, "volatility_halt"
	}

	if !g.backtestSkipRateLimit && g.rateLimit != nil {
		if !g.rateLimit.Allow("signal") {
			return false, "rate_limited"
		}
	}

	return true, ""
}

// ApplySizing applies all risk multipliers to the raw base size: confidence,
// regime, VIX, sentiment, and correlation attenuation. The returned size is
// a share quantity (or capital amount depending on caller context).
func (g *SignalGateImpl) ApplySizing(baseSize, runningCapital, confidence float64) float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if baseSize <= 0 || runningCapital <= 0 {
		return 0
	}

	if g.sizer == nil {
		return baseSize
	}

	return g.sizer.ComputeSizeUncapped(confidence, baseSize, 0)
}

// CheckExposure verifies that the given notional does not violate max-leverage
// or symbol-concentration limits.
func (g *SignalGateImpl) CheckExposure(symbol, side string, notional float64) (bool, string) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.exposure == nil {
		return true, ""
	}

	return g.exposure.CheckOrder(symbol, side, notional)
}

// RecordExposure registers an open position for subsequent exposure checks.
func (g *SignalGateImpl) RecordExposure(symbol, side string, notional float64) {
	if g.exposure == nil {
		return
	}
	g.exposure.AddPosition(symbol, side, notional)
}

// RemoveExposure deregisters a closed position.
func (g *SignalGateImpl) RemoveExposure(symbol, side string, notional float64) {
	if g.exposure == nil {
		return
	}
	g.exposure.RemovePosition(symbol, side, notional)
}

func (g *SignalGateImpl) VolatilityHalt() *VolatilityHalt {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.volHalt
}

func (g *SignalGateImpl) PositionSizer() *PositionSizer {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.sizer
}

func (g *SignalGateImpl) ExposureTracker() *ExposureTracker {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.exposure
}

// ApplyPWin applies the confidence-based sizing adjustment from meta-labeling.
// Formula: baseSize * clamp(pWin*1.5, 0, 1.0)
func (g *SignalGateImpl) ApplyPWin(baseSize, pWin float64) float64 {
	if pWin <= 0 {
		return 0
	}
	return baseSize * clamp(pWin*1.5, 0, 1.0)
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Compile-time check.
var _ SignalGate = (*SignalGateImpl)(nil)
