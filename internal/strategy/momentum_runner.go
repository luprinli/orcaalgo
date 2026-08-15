package strategy

import (
	"github.com/lee-econ/orca-core/internal/types"
)

// MomentumRunner implements time-series momentum (12-1): long-only absolute
// momentum. Entry: the 12-month return, skipping the most recent month, is
// positive AND price is above a long SMA (trend filter). Exit: momentum turns
// negative (reversal) or an ATR trailing stop is hit. A large buffer keeps the
// full 252-bar lookback intact for the 5-year 1d window without wrapping.
type MomentumRunner struct {
	*BaseRunner

	Lookback      float64
	SkipRecent    float64
	TrendPeriod   float64
	AtrPeriod     float64
	AtrMultiplier float64
	ProfitATRMult float64

	peakPrice types.Price
}

func NewMomentumRunner() *MomentumRunner {
	return &MomentumRunner{
		BaseRunner:    NewBaseRunner(2048),
		Lookback:      252,
		SkipRecent:    21,
		TrendPeriod:   50,
		AtrPeriod:     20,
		AtrMultiplier: 3.0,
		ProfitATRMult: 0,
	}
}

func (r *MomentumRunner) Name() string { return "momentum_12_1" }
func (r *MomentumRunner) Type() string { return "momentum" }
func (r *MomentumRunner) Version() (irVersion string, canonicalVersion string) {
	return r.BaseRunner.Version()
}

func (r *MomentumRunner) Reset() {
	r.BaseRunner.Reset()
	r.peakPrice = 0
}

func (r *MomentumRunner) Params() map[string]float64 {
	return map[string]float64{
		"lookback":        r.Lookback,
		"skip_recent":     r.SkipRecent,
		"trend_period":    r.TrendPeriod,
		"atr_period":      r.AtrPeriod,
		"atr_multiplier":  r.AtrMultiplier,
		"profit_atr_mult": r.ProfitATRMult,
	}
}

func (r *MomentumRunner) SetParams(params map[string]float64) {
	if v, ok := params["lookback"]; ok {
		r.Lookback = v
	}
	if v, ok := params["skip_recent"]; ok {
		r.SkipRecent = v
	}
	if v, ok := params["trend_period"]; ok {
		r.TrendPeriod = v
	}
	if v, ok := params["atr_period"]; ok {
		r.AtrPeriod = v
	}
	if v, ok := params["atr_multiplier"]; ok {
		r.AtrMultiplier = v
	}
	if v, ok := params["profit_atr_mult"]; ok {
		r.ProfitATRMult = v
	}
}

func (r *MomentumRunner) ParamDefs() []ParamDef {
	return []ParamDef{
		{Name: "lookback", Type: ParamInteger, Default: 252, Min: 63, Max: 504, Step: 21, Group: "Entry", Description: "Momentum lookback in bars (252 = ~12 months on 1d)"},
		{Name: "skip_recent", Type: ParamInteger, Default: 21, Min: 0, Max: 63, Step: 7, Group: "Entry", Description: "Skip most-recent bars to avoid short-term reversal (21 = ~1 month)"},
		{Name: "trend_period", Type: ParamInteger, Default: 50, Min: 20, Max: 200, Step: 10, Group: "Filter", Description: "SMA trend-filter period (long only above)"},
		{Name: "atr_period", Type: ParamInteger, Default: 20, Min: 7, Max: 60, Step: 7, Group: "Risk", Description: "ATR lookback for trailing stop"},
		{Name: "atr_multiplier", Type: ParamContinuous, Default: 3.0, Min: 1.0, Max: 6.0, Step: 0.5, Group: "Risk", Description: "ATR trailing-stop multiplier"},
		{Name: "profit_atr_mult", Type: ParamContinuous, Default: 0.0, Min: 0.0, Max: 10.0, Step: 1.0, Group: "Exit", Description: "ATR take-profit multiplier (0 = disabled)"},
	}
}

// priceAt returns the n-th most recent close (0 = current), reading the ring
// buffer correctly via HistIdx so the momentum lookback is exact even after the
// buffer wraps.
func (r *MomentumRunner) priceAt(n int) float64 {
	idx := (r.HistIdx - 1 - n) % r.BufferSize
	if idx < 0 {
		idx += r.BufferSize
	}
	return r.PriceHistory[idx]
}

// momentum returns the 12-1 return: (price[skip] - price[lookback]) / price[lookback].
func (r *MomentumRunner) momentum() float64 {
	recent := r.priceAt(int(r.SkipRecent))
	past := r.priceAt(int(r.Lookback))
	if past <= 0 {
		return 0
	}
	return (recent - past) / past
}

func (r *MomentumRunner) Evaluate(candle Candle, regime int8) *Signal {
	price := candle.Close
	if price.IsZero() {
		return nil
	}

	r.PushPrice(price, candle.High, candle.Low, candle.Volume)

	if r.HistCount < int(r.Lookback)+2 {
		return nil
	}

	mom := r.momentum()
	sma := SMA(r.LinearPrices(r.HistCount), r.HistCount, int(r.TrendPeriod))
	atr := TrueRangeATR(r.HighHistory, r.LowHistory, r.PriceHistory, r.HistCount, int(r.AtrPeriod))
	stopMult, profitMult := r.RegimeExitMults(regime)

	sc := StopLossChecker{}

	if r.PositionOpen {
		if r.CurrentSide == "BUY" {
			if price.Compare(r.peakPrice) > 0 {
				r.peakPrice = price
			}
			trailingStop := types.PriceFromFloat(r.peakPrice.Float64() - atr*r.AtrMultiplier*stopMult)
			if sc.IsStopLossHit(price, trailingStop, "BUY") || mom < 0 {
				r.ClosePosition()
				return &Signal{Symbol: candle.Symbol, Side: "SELL", Action: SignalExit}
			}
		}
		return nil
	}

	// Long-only absolute momentum with a trend filter.
	if mom > 0 && sma > 0 && price.Float64() > sma {
		r.peakPrice = price
		stopDist := atr * r.AtrMultiplier * stopMult
		var tp types.Price
		if r.ProfitATRMult > 0 {
			tp = types.PriceFromFloat(price.Float64() + stopDist*r.ProfitATRMult*profitMult)
		}
		r.OpenPosition("BUY", price, types.PriceFromFloat(price.Float64()-stopDist), tp, candle.Time)
		return &Signal{Symbol: candle.Symbol, Side: "BUY", Action: SignalEntry, Quantity: 1.0}
	}

	return nil
}
