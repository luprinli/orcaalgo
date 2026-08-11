package strategy

import (
	"math"

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
func (r *OrbRunner) Version() (irVersion string, canonicalVersion string) { return r.BaseRunner.Version() }

func (r *OrbRunner) Reset() {
	r.BaseRunner.Reset()
	r.openingHigh = 0
	r.openingLow = math.MaxFloat64
	r.rangeSet = false
	r.barsInRange = 0
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
	if regime == 3 {
		return nil
	}

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
				return nil
			}
		}
		return nil
	}

	r.PushPriceOnly(candle.Close)
	minutesInDay := candle.Time.Hour()*60 + candle.Time.Minute()
	sc := StopLossChecker{}
	tc := TakeProfitChecker{}

	if r.PositionOpen {
		exitSide := ""
		if r.CurrentSide == "BUY" {
			if tc.IsTakeProfitHit(candle.Close, r.TakeProfit, "BUY") || sc.IsStopLossHit(candle.Close, r.StopLoss, "BUY") {
				exitSide = "SELL"
			} else if candle.Close.Float64() < r.openingHigh && candle.Low.Float64() < r.openingHigh {
				exitSide = "SELL"
			}
		} else {
			if tc.IsTakeProfitHit(candle.Close, r.TakeProfit, "SELL") || sc.IsStopLossHit(candle.Close, r.StopLoss, "SELL") {
				exitSide = "BUY"
			} else if candle.Close.Float64() > r.openingLow && candle.High.Float64() > r.openingLow {
				exitSide = "BUY"
			}
		}
		if float64(minutesInDay) >= r.CloseExitMinutes {
			exitSide = "SELL"
			if r.CurrentSide == "SELL" {
				exitSide = "BUY"
			}
		}
		if exitSide != "" {
			r.ClosePosition()
			return &Signal{Symbol: candle.Symbol, Side: exitSide, Quantity: 0}
		}
		return nil
	}

	rangeHeight := r.openingHigh - r.openingLow
	if rangeHeight <= 0 {
		return nil
	}

	bufferPct := r.EntryBufferPct / 100.0
	atr := ATR(r.PriceHistory, r.HistCount, int(r.AtrPeriod))
	stopDist := atr * r.AtrMultiplier
	if stopDist < rangeHeight/2 {
		stopDist = rangeHeight / 2
	}
	profitDist := rangeHeight * r.TargetMultiplier

	if candle.Close.Float64() >= r.openingHigh*(1.0+bufferPct) {
		r.OpenPosition("BUY", candle.Close, types.PriceFromFloat(candle.Close.Float64()-stopDist), types.PriceFromFloat(candle.Close.Float64()+profitDist), candle.Time)
		return &Signal{Symbol: candle.Symbol, Side: "BUY", Quantity: 1.0}
	}
	if candle.Close.Float64() <= r.openingLow*(1.0-bufferPct) {
		r.OpenPosition("SELL", candle.Close, types.PriceFromFloat(candle.Close.Float64()+stopDist), types.PriceFromFloat(candle.Close.Float64()-profitDist), candle.Time)
		return &Signal{Symbol: candle.Symbol, Side: "SELL", Quantity: 1.0}
	}

	return nil
}
