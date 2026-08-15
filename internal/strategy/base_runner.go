package strategy

import (
	"time"

	"github.com/lee-econ/orca-core/internal/types"
)

type BaseRunner struct {
	PriceHistory  []float64
	HighHistory   []float64
	LowHistory    []float64
	VolumeHistory []float64

	HistIdx    int
	HistCount  int
	BufferSize int

	PositionOpen bool
	EntryPrice   types.Price
	StopLoss     types.Price
	TakeProfit   types.Price
	CurrentSide  string
	EntryTime    time.Time

	tradeDay    string
	tradesToday int

	// Regime-conditional exit multipliers (regime_gating_deep_dive.md §3.2):
	// widen stops in HighVol/Crisis (avoid noise whipsaw) and loosen targets in
	// Trending (let winners run). Default 1.0 = no change; the optimizer sweeps
	// them via RegimeParamDefs (stop_mult_highvol / stop_mult_crisis /
	// profit_mult_trending).
	StopMultHighVol    float64
	StopMultCrisis     float64
	ProfitMultTrending float64

	irVersion        string
	canonicalVersion string
	instanceHash     string
}

func NewBaseRunner(bufferSize int) *BaseRunner {
	return &BaseRunner{
		BufferSize:         bufferSize,
		PriceHistory:       make([]float64, bufferSize),
		HighHistory:        make([]float64, bufferSize),
		LowHistory:         make([]float64, bufferSize),
		VolumeHistory:      make([]float64, bufferSize),
		StopMultHighVol:    1.0,
		StopMultCrisis:     1.0,
		ProfitMultTrending: 1.0,
		irVersion:          "qst-ir/0.4",
		canonicalVersion:   "qst-canonical/0.4",
	}
}

func (b *BaseRunner) SetVersion(irVersion, canonicalVersion string) {
	b.irVersion = irVersion
	b.canonicalVersion = canonicalVersion
}

func (b *BaseRunner) Version() (irVersion string, canonicalVersion string) {
	return b.irVersion, b.canonicalVersion
}

func (b *BaseRunner) SetInstanceHash(h string) {
	b.instanceHash = h
}

func (b *BaseRunner) InstanceHash() string {
	return b.instanceHash
}

func (b *BaseRunner) PushPrice(price, high, low types.Price, volume float64) {
	if !finite(price.Float64()) {
		// Skip non-finite candles (NaN/Inf) so they can't corrupt the indicator
		// history (Rule 18). Valid data is unaffected.
		return
	}
	idx := b.HistIdx % b.BufferSize
	b.PriceHistory[idx] = price.Float64()
	b.HighHistory[idx] = high.Float64()
	b.LowHistory[idx] = low.Float64()
	b.VolumeHistory[idx] = volume
	b.HistIdx++
	if b.HistCount < b.BufferSize {
		b.HistCount++
	}
}

func (b *BaseRunner) PushPriceOnly(price types.Price) {
	b.PushPrice(price, 0, 0, 0)
}

// LinearPrices/Highs/Lows/Volumes return the last n bars in chronological order
// (linearizing the circular buffer) for the cinar indicator wrappers, which
// expect chronological slices (Rule 6). n is clamped to the valid count.
func (b *BaseRunner) LinearPrices(n int) []float64 {
	return linearWindow(b.PriceHistory, b.HistCount, b.HistIdx, b.BufferSize, n)
}
func (b *BaseRunner) LinearHighs(n int) []float64 {
	return linearWindow(b.HighHistory, b.HistCount, b.HistIdx, b.BufferSize, n)
}
func (b *BaseRunner) LinearLows(n int) []float64 {
	return linearWindow(b.LowHistory, b.HistCount, b.HistIdx, b.BufferSize, n)
}
func (b *BaseRunner) LinearVolumes(n int) []float64 {
	return linearWindow(b.VolumeHistory, b.HistCount, b.HistIdx, b.BufferSize, n)
}

func (b *BaseRunner) Reset() {
	b.HistIdx = 0
	b.HistCount = 0
	b.PositionOpen = false
	b.EntryPrice = 0
	b.StopLoss = 0
	b.TakeProfit = 0
	b.CurrentSide = ""
	b.tradeDay = ""
	b.tradesToday = 0
	clearSlice(b.PriceHistory)
	clearSlice(b.HighHistory)
	clearSlice(b.LowHistory)
	clearSlice(b.VolumeHistory)
}

func (b *BaseRunner) ResetPosition() {
	b.PositionOpen = false
	b.EntryPrice = 0
	b.StopLoss = 0
	b.TakeProfit = 0
	b.CurrentSide = ""
}

func (b *BaseRunner) OpenPosition(side string, entryPrice, stopLoss, takeProfit types.Price, entryTime time.Time) {
	b.PositionOpen = true
	b.CurrentSide = side
	b.EntryPrice = entryPrice
	b.StopLoss = stopLoss
	b.TakeProfit = takeProfit
	b.EntryTime = entryTime
}

func (b *BaseRunner) ClosePosition() {
	b.PositionOpen = false
}

func (b *BaseRunner) OnFill(orderID string, symbol string, side string, entryPrice types.Price, fillPrice types.Price, quantity float64, filledQty float64) {
}

func (b *BaseRunner) OnCancel(orderID string, reason string) {}

func (b *BaseRunner) OnOrderRejected(orderID string, reason string) {}

func (b *BaseRunner) IsStopLossHit(price types.Price) bool {
	if !b.PositionOpen {
		return false
	}
	if b.CurrentSide == "BUY" {
		return price.Compare(b.StopLoss) <= 0
	}
	return price.Compare(b.StopLoss) >= 0
}

func (b *BaseRunner) IsTakeProfitHit(price types.Price) bool {
	if !b.PositionOpen {
		return false
	}
	if b.CurrentSide == "BUY" {
		return price.Compare(b.TakeProfit) >= 0
	}
	return price.Compare(b.TakeProfit) <= 0
}

func (b *BaseRunner) IsTimeExit(maxMinutes float64, currentTime time.Time) bool {
	if !b.PositionOpen {
		return false
	}
	return currentTime.Sub(b.EntryTime).Minutes() >= maxMinutes
}

// CanTrade reports whether another entry is allowed for the given timestamp,
// honoring maxPerDay (<= 0 means unlimited). Resets the counter on a new day.
// This is the shared day-scoped trade-frequency guard (Rule 5) so no runner
// hand-rolls its own daily counter inconsistently.
func (b *BaseRunner) CanTrade(t time.Time, maxPerDay int) bool {
	day := t.Format("2006-01-02")
	if day != b.tradeDay {
		b.tradeDay = day
		b.tradesToday = 0
	}
	if maxPerDay <= 0 {
		return true
	}
	return b.tradesToday < maxPerDay
}

// RecordTrade registers a completed entry for the current day.
func (b *BaseRunner) RecordTrade(t time.Time) {
	day := t.Format("2006-01-02")
	if day != b.tradeDay {
		b.tradeDay = day
		b.tradesToday = 0
	}
	b.tradesToday++
}

// SetRegimeExitParams consumes the shared regime-conditional exit multipliers
// from an optimizer/param map (stop_mult_highvol, stop_mult_crisis,
// profit_mult_trending).
func (b *BaseRunner) SetRegimeExitParams(params map[string]float64) {
	if params == nil {
		return
	}
	if v, ok := params["stop_mult_highvol"]; ok {
		b.StopMultHighVol = v
	}
	if v, ok := params["stop_mult_crisis"]; ok {
		b.StopMultCrisis = v
	}
	if v, ok := params["profit_mult_trending"]; ok {
		b.ProfitMultTrending = v
	}
}

// RegimeExitMults returns the stop and take-profit multipliers for a regime:
// widen stops in HighVol/Crisis, loosen targets in Trending. Returns (1,1) for
// Calm. Callers multiply their stopDist/profitDist by these.
func (b *BaseRunner) RegimeExitMults(regime int8) (stopMult, profitMult float64) {
	stopMult, profitMult = 1.0, 1.0
	switch regime {
	case RegimeHighVol:
		stopMult = b.StopMultHighVol
	case RegimeCrisis:
		stopMult = b.StopMultCrisis
	case RegimeTrending:
		profitMult = b.ProfitMultTrending
	}
	return stopMult, profitMult
}

func clearSlice(s []float64) {
	for i := range s {
		s[i] = 0
	}
}
