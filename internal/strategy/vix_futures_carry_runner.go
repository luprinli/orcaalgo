package strategy

import (
	"github.com/lee-econ/orca-core/internal/types"
)

// VIXFuturesCarryRunner implements a volatility risk premium harvesting
// strategy. It enters SHORT positions when the VIX term structure suggests
// contango (VIX futures > VIX spot) — capturing the roll yield from selling
// elevated volatility after a spike. The strategy is only active in HighVol
// regime (2).
//
// Since VIX futures data is not yet ingested, the runner uses spot VIX as
// a proxy: when spot VIX exceeds the threshold, the market is assumed to be
// in a vol spike that will revert. The runner then fades the directional
// move with mean-reversion entries.
type VIXFuturesCarryRunner struct {
	*BaseRunner

	// VIXSpot is the current VIX spot reading, updated externally by the engine.
	VIXSpot float64

	// ContangoThreshold is the minimum VIX spot level to enter short-vol.
	// Acts as proxy for VIX futures > VIX spot contango signal.
	ContangoThreshold float64

	// FadeLookback is the lookback period for mean-reversion z-score.
	FadeLookback int

	// FadeEntryZ is the z-score threshold for entry (fading the vol spike).
	FadeEntryZ float64

	// FadeExitZ is the z-score threshold for exit.
	FadeExitZ float64

	// StopATRMult is the ATR multiplier for stop-loss distance.
	StopATRMult float64

	// ProfitATRMult is the ATR multiplier for take-profit distance.
	ProfitATRMult float64

	// MaxHold restricts the maximum holding period in bars.
	MaxHold int

	barsHeld int
}

func NewVIXFuturesCarryRunner() *VIXFuturesCarryRunner {
	return &VIXFuturesCarryRunner{
		BaseRunner:        NewBaseRunner(128),
		ContangoThreshold: 22.0,
		FadeLookback:      10,
		FadeEntryZ:        1.5,
		FadeExitZ:         0.3,
		StopATRMult:       2.0,
		ProfitATRMult:     3.0,
		MaxHold:           30,
	}
}

func (r *VIXFuturesCarryRunner) Name() string              { return "vix_futures_carry" }
func (r *VIXFuturesCarryRunner) Type() string              { return "volatility" }
func (r *VIXFuturesCarryRunner) Version() (string, string) { return r.BaseRunner.Version() }
func (r *VIXFuturesCarryRunner) SetVersion(a, b string)    { r.BaseRunner.SetVersion(a, b) }
func (r *VIXFuturesCarryRunner) SetInstanceHash(h string)  { r.BaseRunner.SetInstanceHash(h) }
func (r *VIXFuturesCarryRunner) InstanceHash() string      { return r.BaseRunner.InstanceHash() }

func (r *VIXFuturesCarryRunner) Reset() {
	r.BaseRunner.Reset()
	r.barsHeld = 0
}

func (r *VIXFuturesCarryRunner) Params() map[string]float64 {
	return map[string]float64{
		"contango_threshold": r.ContangoThreshold,
		"fade_lookback":      float64(r.FadeLookback),
		"fade_entry_z":       r.FadeEntryZ,
		"fade_exit_z":        r.FadeExitZ,
		"stop_atr_mult":      r.StopATRMult,
		"profit_atr_mult":    r.ProfitATRMult,
		"max_hold":           float64(r.MaxHold),
	}
}

func (r *VIXFuturesCarryRunner) SetParams(params map[string]float64) {
	if v, ok := params["contango_threshold"]; ok {
		r.ContangoThreshold = v
	}
	if v, ok := params["fade_lookback"]; ok {
		r.FadeLookback = int(v)
	}
	if v, ok := params["fade_entry_z"]; ok {
		r.FadeEntryZ = v
	}
	if v, ok := params["fade_exit_z"]; ok {
		r.FadeExitZ = v
	}
	if v, ok := params["stop_atr_mult"]; ok {
		r.StopATRMult = v
	}
	if v, ok := params["profit_atr_mult"]; ok {
		r.ProfitATRMult = v
	}
	if v, ok := params["max_hold"]; ok {
		r.MaxHold = int(v)
	}
}

func (r *VIXFuturesCarryRunner) ParamDefs() []ParamDef {
	return []ParamDef{
		{Name: "contango_threshold", Type: ParamContinuous, Default: 22.0, Min: 15, Max: 35, Step: 1.0, Group: "Entry", Description: "Minimum VIX spot for contango proxy"},
		{Name: "fade_lookback", Type: ParamInteger, Default: 10, Min: 5, Max: 30, Step: 5, Group: "Signal", Description: "Lookback bars for fade z-score"},
		{Name: "fade_entry_z", Type: ParamContinuous, Default: 1.5, Min: 1.0, Max: 3.0, Step: 0.25, Group: "Signal", Description: "Z-score for fade entry"},
		{Name: "fade_exit_z", Type: ParamContinuous, Default: 0.3, Min: 0.0, Max: 1.0, Step: 0.1, Group: "Signal", Description: "Z-score for fade exit"},
		{Name: "stop_atr_mult", Type: ParamContinuous, Default: 2.0, Min: 1.0, Max: 4.0, Step: 0.5, Group: "Risk", Description: "ATR multiplier for stop-loss"},
		{Name: "profit_atr_mult", Type: ParamContinuous, Default: 3.0, Min: 1.5, Max: 5.0, Step: 0.5, Group: "Risk", Description: "ATR multiplier for take-profit"},
		{Name: "max_hold", Type: ParamInteger, Default: 30, Min: 10, Max: 60, Step: 5, Group: "Risk", Description: "Max holding period in bars"},
	}
}

// SetVIX updates the current VIX spot reading. Called by the engine each bar.
func (r *VIXFuturesCarryRunner) SetVIX(vix float64) {
	r.VIXSpot = vix
}

func (r *VIXFuturesCarryRunner) Evaluate(candle Candle, regime int8) *Signal {
	price := candle.Close
	if price.IsZero() {
		return nil
	}

	r.PushPrice(price, candle.High, candle.Low, candle.Volume)

	// VIX contango proxy gate: no entry if VIX is below the threshold.
	if r.VIXSpot < r.ContangoThreshold {
		return nil
	}

	mean, stdDev, atr := r.computeFadeStats()
	if stdDev <= 0 || atr <= 0 {
		return nil
	}

	zScore := (price.Float64() - mean) / stdDev

	// Exit management.
	if r.PositionOpen {
		r.barsHeld++

		if r.barsHeld >= r.MaxHold {
			r.ClosePosition()
			return &Signal{Symbol: candle.Symbol, Side: exitSide(r.CurrentSide), Action: SignalExit}
		}

		stopPrice := r.EntryPrice.Float64()
		if r.CurrentSide == "BUY" {
			stopPrice -= atr * r.StopATRMult
			if price.Float64() <= stopPrice || zScore >= r.FadeExitZ {
				r.ClosePosition()
				return &Signal{Symbol: candle.Symbol, Side: "SELL", Action: SignalExit}
			}
		} else {
			stopPrice += atr * r.StopATRMult
			if price.Float64() >= stopPrice || zScore <= -r.FadeExitZ {
				r.ClosePosition()
				return &Signal{Symbol: candle.Symbol, Side: "BUY", Action: SignalExit}
			}
		}
		return nil
	}

	// Entry: fade the vol spike (mean-reversion after extreme move).
	stopMult, profitMult := r.RegimeExitMults(regime)
	if zScore <= -r.FadeEntryZ {
		r.barsHeld = 0
		r.OpenPosition("BUY", price,
			types.PriceFromFloat(price.Float64()-atr*r.StopATRMult*stopMult),
			types.PriceFromFloat(price.Float64()+atr*r.ProfitATRMult*profitMult),
			candle.Time)
		return &Signal{Symbol: candle.Symbol, Side: "BUY", Action: SignalEntry, Quantity: 1.0}
	}

	if zScore >= r.FadeEntryZ {
		r.barsHeld = 0
		r.OpenPosition("SELL", price,
			types.PriceFromFloat(price.Float64()+atr*r.StopATRMult*stopMult),
			types.PriceFromFloat(price.Float64()-atr*r.ProfitATRMult*profitMult),
			candle.Time)
		return &Signal{Symbol: candle.Symbol, Side: "SELL", Action: SignalEntry, Quantity: 1.0}
	}

	return nil
}

func (r *VIXFuturesCarryRunner) computeFadeStats() (mean, stdDev, atr float64) {
	n := r.HistCount
	if n < r.FadeLookback {
		return 0, 0, 0
	}
	start := n - r.FadeLookback
	if start < 0 {
		start = 0
	}
	var sum float64
	count := 0
	for i := start; i < n; i++ {
		if v := r.PriceHistory[i%r.BufferSize]; v > 0 {
			sum += v
			count++
		}
	}
	if count < 2 {
		return 0, 0, 0
	}
	mean = sum / float64(count)
	var variance float64
	for i := start; i < n; i++ {
		if v := r.PriceHistory[i%r.BufferSize]; v > 0 {
			d := v - mean
			variance += d * d
		}
	}
	stdDev = sampleStd(variance, count)
	atr = TrueRangeATR(r.HighHistory, r.LowHistory, r.PriceHistory, n, 14)
	return
}

func exitSide(currentSide string) string {
	if currentSide == "BUY" {
		return "SELL"
	}
	return "BUY"
}

var _ Strategy = (*VIXFuturesCarryRunner)(nil)
