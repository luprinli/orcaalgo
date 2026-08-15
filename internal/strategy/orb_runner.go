package strategy

import (
	"math"
	"time"

	"github.com/lee-econ/orca-core/internal/types"
)

type OrbRunner struct {
	*BaseRunner

	RangeMinutes     float64
	EntryBufferPct   float64
	AtrPeriod        float64
	AtrMultiplier    float64
	TargetMultiplier float64
	CloseExitMinutes float64
	MinRangePct      float64
	openingHigh      float64
	openingLow       float64
	rangeSet         bool
	barsInRange      int
	skipDay          bool
	lastDay          time.Time
}

func NewOrbRunner() *OrbRunner {
	return &OrbRunner{
		BaseRunner:       NewBaseRunner(128),
		RangeMinutes:     5,
		EntryBufferPct:   0.1,
		AtrPeriod:        14,
		AtrMultiplier:    2.0,
		TargetMultiplier: 2.0,
		CloseExitMinutes: 390,
		MinRangePct:      0.1,
		openingLow:       math.MaxFloat64,
	}
}

func (r *OrbRunner) Name() string { return "opening_range_breakout" }
func (r *OrbRunner) Type() string { return "breakout" }
func (r *OrbRunner) Version() (irVersion string, canonicalVersion string) {
	return r.BaseRunner.Version()
}

func (r *OrbRunner) Reset() {
	r.BaseRunner.Reset()
	r.resetOpeningRange()
	r.lastDay = time.Time{}
}

// resetOpeningRange clears the accumulated opening-range state so a fresh
// range can form for the new day.
func (r *OrbRunner) resetOpeningRange() {
	r.openingHigh = 0
	r.openingLow = math.MaxFloat64
	r.rangeSet = false
	r.barsInRange = 0
	r.skipDay = false
}

// sameDay reports whether two timestamps fall on the same UTC calendar day.
func sameDay(a, b time.Time) bool {
	ay, am, ad := a.UTC().Date()
	by, bm, bd := b.UTC().Date()
	return ay == by && am == bm && ad == bd
}

func (r *OrbRunner) Params() map[string]float64 {
	return map[string]float64{
		"range_minutes":     r.RangeMinutes,
		"entry_buffer_pct":  r.EntryBufferPct,
		"atr_period":        r.AtrPeriod,
		"atr_multiplier":    r.AtrMultiplier,
		"target_multiplier": r.TargetMultiplier,
	}
}

func (r *OrbRunner) SetParams(params map[string]float64) {
	if v, ok := params["range_minutes"]; ok {
		r.RangeMinutes = v
	}
	if v, ok := params["entry_buffer_pct"]; ok {
		r.EntryBufferPct = v
	}
	if v, ok := params["atr_period"]; ok {
		r.AtrPeriod = v
	}
	if v, ok := params["atr_multiplier"]; ok {
		r.AtrMultiplier = v
	}
	if v, ok := params["target_multiplier"]; ok {
		r.TargetMultiplier = v
	}
}

func (r *OrbRunner) ParamDefs() []ParamDef {
	return []ParamDef{
		{Name: "range_minutes", Type: ParamInteger, Default: 5, Min: 1, Max: 15, Step: 1, Group: "Entry", Description: "Number of minutes to form the opening range"},
		{Name: "entry_buffer_pct", Type: ParamContinuous, Default: 0.1, Min: 0.01, Max: 1.0, Step: 0.05, Group: "Entry", Description: "Percentage buffer beyond range high/low to trigger entry"},
		{Name: "atr_period", Type: ParamInteger, Default: 14, Min: 7, Max: 28, Step: 7, Group: "Risk", Description: "ATR lookback period for stop/target placement"},
		{Name: "atr_multiplier", Type: ParamContinuous, Default: 2.0, Min: 1.0, Max: 4.0, Step: 0.5, Group: "Risk", Description: "ATR multiplier for stop distance"},
		{Name: "target_multiplier", Type: ParamContinuous, Default: 2.0, Min: 1.0, Max: 5.0, Step: 0.5, Group: "Exit", Description: "Risk:reward target multiplier (relative to stop distance)"},
	}
}

func (r *OrbRunner) Evaluate(candle Candle, regime int8) *Signal {
	// Reset the opening range at the start of each new trading day. Without
	// this, the range formed over the first few bars of the entire backtest
	// never resets, producing a stale multi-month range and the sub-1% win
	// rates observed in the matrix.
	if !r.lastDay.IsZero() && !sameDay(r.lastDay, candle.Time) {
		// End-of-day exit: force-close any position carried overnight at the
		// new day's first bar. The prior implementation keyed this off UTC
		// minute-of-day vs CloseExitMinutes (a session-relative value), which
		// fired immediately on every intraday bar and turned every ORB entry
		// into a one-bar exit (~2-7% win rate).
		if r.PositionOpen {
			exitSide := "SELL"
			if r.CurrentSide == "SELL" {
				exitSide = "BUY"
			}
			r.ClosePosition()
			r.resetOpeningRange()
			r.lastDay = candle.Time
			return &Signal{Symbol: candle.Symbol, Side: exitSide, Action: SignalExit}
		}
		r.resetOpeningRange()
	}
	r.lastDay = candle.Time

	if !r.rangeSet {
		r.barsInRange++
		if candle.High.Float64() > r.openingHigh {
			r.openingHigh = candle.High.Float64()
		}
		if candle.Low.Float64() < r.openingLow {
			r.openingLow = candle.Low.Float64()
		}
		if float64(r.barsInRange) >= r.RangeMinutes {
			r.rangeSet = true
			// Minimum volatility requirement: reject if opening range is too narrow.
			rangePct := (r.openingHigh - r.openingLow) / candle.Close.Float64() * 100.0
			if r.MinRangePct > 0 && rangePct < r.MinRangePct {
				r.skipDay = true
				return nil
			}
		}
		return nil
	}

	if r.skipDay {
		return nil
	}

	r.PushPrice(candle.Close, candle.High, candle.Low, candle.Volume)
	sc := StopLossChecker{}
	tc := TakeProfitChecker{}

	if r.PositionOpen {
		exitSide := ""
		if r.CurrentSide == "BUY" {
			if tc.IsTakeProfitHit(candle.Close, r.TakeProfit, "BUY") || sc.IsStopLossHit(candle.Close, r.StopLoss, "BUY") {
				exitSide = "SELL"
			}
		} else {
			if tc.IsTakeProfitHit(candle.Close, r.TakeProfit, "SELL") || sc.IsStopLossHit(candle.Close, r.StopLoss, "SELL") {
				exitSide = "BUY"
			}
		}
		if exitSide != "" {
			r.ClosePosition()
			return &Signal{Symbol: candle.Symbol, Side: exitSide, Action: SignalExit}
		}
		return nil
	}

	rangeHeight := r.openingHigh - r.openingLow
	if rangeHeight <= 0 {
		return nil
	}

	bufferPct := r.EntryBufferPct / 100.0
	atr := TrueRangeATR(r.HighHistory, r.LowHistory, r.PriceHistory, r.HistCount, int(r.AtrPeriod))
	stopMult, profitMult := r.RegimeExitMults(regime)
	stopDist := atr * r.AtrMultiplier * stopMult
	if stopDist < rangeHeight/2 {
		stopDist = rangeHeight / 2
	}
	// Target is relative to the stop distance (per the ParamDef: "risk:reward
	// target multiplier relative to stop distance"). Applying it to rangeHeight
	// produced a 4:1 R:R (target = 2x range, stop = range/2); applying it to
	// stopDist yields the intended 2:1 R:R.
	profitDist := stopDist * r.TargetMultiplier * profitMult

	// One entry per day: a canonical opening-range breakout takes a single
	// position at the range break and holds to stop/target/EOD. Re-entering
	// after every stop-out was the overtrading bug (5+ trades/day) that
	// turned a 2:1 R:R breakout into negative Sharpe.
	if !r.CanTrade(candle.Time, 1) {
		return nil
	}

	if candle.Close.Float64() >= r.openingHigh*(1.0+bufferPct) {
		r.OpenPosition("BUY", candle.Close, types.PriceFromFloat(candle.Close.Float64()-stopDist), types.PriceFromFloat(candle.Close.Float64()+profitDist), candle.Time)
		r.RecordTrade(candle.Time)
		return &Signal{Symbol: candle.Symbol, Side: "BUY", Action: SignalEntry, Quantity: 1.0, StopLoss: types.PriceFromFloat(candle.Close.Float64() - stopDist), TakeProfit: types.PriceFromFloat(candle.Close.Float64() + profitDist)}
	}
	if candle.Close.Float64() <= r.openingLow*(1.0-bufferPct) {
		r.OpenPosition("SELL", candle.Close, types.PriceFromFloat(candle.Close.Float64()+stopDist), types.PriceFromFloat(candle.Close.Float64()-profitDist), candle.Time)
		r.RecordTrade(candle.Time)
		return &Signal{Symbol: candle.Symbol, Side: "SELL", Action: SignalEntry, Quantity: 1.0, StopLoss: types.PriceFromFloat(candle.Close.Float64() + stopDist), TakeProfit: types.PriceFromFloat(candle.Close.Float64() - profitDist)}
	}

	return nil
}
