package strategy

import (
	"github.com/lee-econ/orca-core/internal/types"
)

// DragonTrendRunner implements the Dragon Capital multi-EMA trend-following
// strategy. It uses 4 EMAs (8, 21, 50, 200) to determine trend strength and
// direction. Position size is proportional to the number of aligned EMAs.
// Entry requires at least MinAligned EMAs pointing in the same direction AND
// ADX above the threshold.
type DragonTrendRunner struct {
	*BaseRunner

	EMAPeriods    []int
	MinAligned    int
	ADXPeriod     int
	ADXThreshold  float64
	ATRPeriod     int
	ATRMultiplier float64
	ProfitATRMult float64

	emaValues     []float64
	prevEMAValues []float64
	peakPrice     types.Price
	signalPending bool
	pendingSide   string
}

func NewDragonTrendRunner() *DragonTrendRunner {
	r := &DragonTrendRunner{
		BaseRunner:    NewBaseRunner(256),
		EMAPeriods:    []int{8, 21, 50, 200},
		MinAligned:    3,
		ADXPeriod:     14,
		ADXThreshold:  25.0,
		ATRPeriod:     14,
		ATRMultiplier: 3.0,
		ProfitATRMult: 3.0,
	}
	r.emaValues = make([]float64, len(r.EMAPeriods))
	r.prevEMAValues = make([]float64, len(r.EMAPeriods))
	return r
}

func (r *DragonTrendRunner) Name() string              { return "dragon_trend" }
func (r *DragonTrendRunner) Type() string              { return "trend" }
func (r *DragonTrendRunner) Version() (string, string) { return r.BaseRunner.Version() }

func (r *DragonTrendRunner) Reset() {
	r.BaseRunner.Reset()
	for i := range r.emaValues {
		r.emaValues[i] = 0
		r.prevEMAValues[i] = 0
	}
	r.signalPending = false
	r.peakPrice = 0
}

func (r *DragonTrendRunner) Params() map[string]float64 {
	return map[string]float64{
		"min_aligned":     float64(r.MinAligned),
		"adx_period":      float64(r.ADXPeriod),
		"adx_threshold":   r.ADXThreshold,
		"atr_period":      float64(r.ATRPeriod),
		"atr_multiplier":  r.ATRMultiplier,
		"profit_atr_mult": r.ProfitATRMult,
	}
}

func (r *DragonTrendRunner) SetParams(params map[string]float64) {
	if v, ok := params["min_aligned"]; ok {
		r.MinAligned = int(v)
	}
	if v, ok := params["adx_period"]; ok {
		r.ADXPeriod = int(v)
	}
	if v, ok := params["adx_threshold"]; ok {
		r.ADXThreshold = v
	}
	if v, ok := params["atr_period"]; ok {
		r.ATRPeriod = int(v)
	}
	if v, ok := params["atr_multiplier"]; ok {
		r.ATRMultiplier = v
	}
	if v, ok := params["profit_atr_mult"]; ok {
		r.ProfitATRMult = v
	}
}

func (r *DragonTrendRunner) ParamDefs() []ParamDef {
	return []ParamDef{
		{Name: "min_aligned", Type: ParamInteger, Default: 3, Min: 2, Max: 4, Step: 1, Group: "Filter", Description: "Minimum aligned EMAs for entry"},
		{Name: "adx_period", Type: ParamInteger, Default: 14, Min: 7, Max: 28, Step: 7, Group: "Filter", Description: "ADX lookback period"},
		{Name: "adx_threshold", Type: ParamContinuous, Default: 20.0, Min: 10, Max: 35, Step: 5, Group: "Filter", Description: "Minimum ADX for entry"},
		{Name: "atr_period", Type: ParamInteger, Default: 14, Min: 7, Max: 28, Step: 7, Group: "Risk", Description: "ATR lookback for stop placement"},
		{Name: "atr_multiplier", Type: ParamContinuous, Default: 2.0, Min: 1.0, Max: 4.0, Step: 0.5, Group: "Risk", Description: "ATR multiplier for stop distance"},
		{Name: "profit_atr_mult", Type: ParamContinuous, Default: 3.0, Min: 1.5, Max: 5.0, Step: 0.5, Group: "Exit", Description: "ATR multiplier for take-profit distance"},
	}
}

func (r *DragonTrendRunner) Evaluate(candle Candle, regime int8) *Signal {
	price := candle.Close
	if price.IsZero() {
		return nil
	}

	pf := price.Float64()
	r.PushPrice(price, candle.High, candle.Low, candle.Volume)

	// Initialize EMAs on first bar.
	if r.emaValues[0] == 0 {
		for i := range r.emaValues {
			r.emaValues[i] = pf
			r.prevEMAValues[i] = pf
		}
		return nil
	}

	// Update EMAs.
	copy(r.prevEMAValues, r.emaValues)
	for i, period := range r.EMAPeriods {
		alpha := 2.0 / (float64(period) + 1.0)
		r.emaValues[i] = pf*alpha + r.emaValues[i]*(1.0-alpha)
	}

	if r.HistCount < r.EMAPeriods[len(r.EMAPeriods)-1]+r.ADXPeriod {
		return nil
	}

	// Count aligned EMAs (price above all = bullish, below all = bearish).
	alignedBull := 0
	alignedBear := 0
	for i := range r.emaValues {
		if r.emaValues[i] > r.prevEMAValues[i] {
			alignedBull++
		} else if r.emaValues[i] < r.prevEMAValues[i] {
			alignedBear++
		}
	}

	adx := ADX(r.PriceHistory, r.HighHistory, r.LowHistory, r.HistCount, r.ADXPeriod)
	_ = float64(alignedBull+alignedBear) / float64(len(r.EMAPeriods))

	if adx < r.ADXThreshold {
		return nil
	}

	tc := TakeProfitChecker{}

	// Exit management.
	if r.PositionOpen {
		if r.CurrentSide == "BUY" {
			if pf > r.peakPrice.Float64() {
				r.peakPrice = price
			}
			atr := TrueRangeATR(r.HighHistory, r.LowHistory, r.PriceHistory, r.HistCount, r.ATRPeriod)
			trailStop := r.peakPrice.Float64() - atr*r.ATRMultiplier
			if pf <= trailStop ||
				tc.IsTakeProfitHit(price, r.TakeProfit, "BUY") ||
				alignedBear >= r.MinAligned {
				r.ClosePosition()
				return &Signal{Symbol: candle.Symbol, Side: "SELL", Action: SignalExit}
			}
		} else {
			if pf < r.peakPrice.Float64() {
				r.peakPrice = price
			}
			atr := TrueRangeATR(r.HighHistory, r.LowHistory, r.PriceHistory, r.HistCount, r.ATRPeriod)
			trailStop := r.peakPrice.Float64() + atr*r.ATRMultiplier
			if pf >= trailStop ||
				tc.IsTakeProfitHit(price, r.TakeProfit, "SELL") ||
				alignedBull >= r.MinAligned {
				r.ClosePosition()
				return &Signal{Symbol: candle.Symbol, Side: "BUY", Action: SignalExit}
			}
		}
		return nil
	}

	// Entry: require minimum aligned EMAs AND ADX > threshold.
	if alignedBull >= r.MinAligned {
		atr := TrueRangeATR(r.HighHistory, r.LowHistory, r.PriceHistory, r.HistCount, r.ATRPeriod)
		if atr <= 0 {
			return nil
		}
		stopMult, profitMult := r.RegimeExitMults(regime)
		stopDist := atr * r.ATRMultiplier * stopMult
		profitDist := atr * r.ProfitATRMult * profitMult
		r.peakPrice = price
		size := 1.0
		r.OpenPosition("BUY", price,
			types.PriceFromFloat(pf-stopDist),
			types.PriceFromFloat(pf+profitDist),
			candle.Time)
		return &Signal{Symbol: candle.Symbol, Side: "BUY", Action: SignalEntry, Quantity: size}
	}

	if alignedBear >= r.MinAligned {
		atr := TrueRangeATR(r.HighHistory, r.LowHistory, r.PriceHistory, r.HistCount, r.ATRPeriod)
		if atr <= 0 {
			return nil
		}
		stopMult, profitMult := r.RegimeExitMults(regime)
		stopDist := atr * r.ATRMultiplier * stopMult
		profitDist := atr * r.ProfitATRMult * profitMult
		r.peakPrice = price
		size := 1.0
		r.OpenPosition("SELL", price,
			types.PriceFromFloat(pf+stopDist),
			types.PriceFromFloat(pf-profitDist),
			candle.Time)
		return &Signal{Symbol: candle.Symbol, Side: "SELL", Action: SignalEntry, Quantity: size}
	}

	return nil
}
