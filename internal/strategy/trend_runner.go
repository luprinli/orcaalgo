package strategy

import "github.com/lee-econ/orca-core/internal/types"

type TrendRunner struct {
	*BaseRunner

	FastPeriod    float64
	SlowPeriod    float64
	AtrPeriod     float64
	AtrMultiplier float64
	ProfitATRMult float64
	AdxPeriod     float64
	AdxThreshold  float64
	ChopThreshold float64
	fastEMA       float64
	slowEMA       float64
	prevFastEMA   float64
	prevSlowEMA   float64
	peakPrice     types.Price
	signalPending bool
	pendingSide   string
}

func NewTrendRunner() *TrendRunner {
	return &TrendRunner{
		BaseRunner:    NewBaseRunner(128),
		FastPeriod:    20,
		SlowPeriod:    50,
		AtrPeriod:     14,
		AtrMultiplier: 3.0,
		ProfitATRMult: 2.0,
		AdxPeriod:     14,
		AdxThreshold:  20.0,
		ChopThreshold: 61.8,
	}
}

func (r *TrendRunner) Name() string { return "trend_following" }
func (r *TrendRunner) Type() string { return "trend" }
func (r *TrendRunner) Version() (irVersion string, canonicalVersion string) {
	return r.BaseRunner.Version()
}

func (r *TrendRunner) Reset() {
	r.BaseRunner.Reset()
	r.fastEMA = 0
	r.slowEMA = 0
	r.prevFastEMA = 0
	r.prevSlowEMA = 0
	r.signalPending = false
	r.peakPrice = 0
}

func (r *TrendRunner) Params() map[string]float64 {
	return map[string]float64{
		"fast_period":     r.FastPeriod,
		"slow_period":     r.SlowPeriod,
		"atr_period":      r.AtrPeriod,
		"atr_multiplier":  r.AtrMultiplier,
		"profit_atr_mult": r.ProfitATRMult,
		"adx_period":      r.AdxPeriod,
		"adx_threshold":   r.AdxThreshold,
		"chop_threshold":  r.ChopThreshold,
	}
}

func (r *TrendRunner) SetParams(params map[string]float64) {
	if v, ok := params["fast_period"]; ok {
		r.FastPeriod = v
	}
	if v, ok := params["slow_period"]; ok {
		r.SlowPeriod = v
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
	if v, ok := params["adx_period"]; ok {
		r.AdxPeriod = v
	}
	if v, ok := params["adx_threshold"]; ok {
		r.AdxThreshold = v
	}
	if v, ok := params["chop_threshold"]; ok {
		r.ChopThreshold = v
	}
}

func (r *TrendRunner) ParamDefs() []ParamDef {
	return []ParamDef{
		{Name: "fast_period", Type: ParamInteger, Default: 20, Min: 5, Max: 40, Step: 5, Group: "Entry", Description: "Fast EMA period for crossover detection"},
		{Name: "slow_period", Type: ParamInteger, Default: 50, Min: 20, Max: 120, Step: 10, Group: "Entry", Description: "Slow EMA period for crossover detection"},
		{Name: "atr_period", Type: ParamInteger, Default: 14, Min: 7, Max: 28, Step: 7, Group: "Risk", Description: "ATR lookback period for stop placement"},
		{Name: "atr_multiplier", Type: ParamContinuous, Default: 2.0, Min: 1.0, Max: 4.0, Step: 0.5, Group: "Risk", Description: "ATR multiplier for trailing stop distance"},
		{Name: "profit_atr_mult", Type: ParamContinuous, Default: 2.0, Min: 1.0, Max: 5.0, Step: 0.5, Group: "Exit", Description: "ATR multiplier for take-profit distance"},
		{Name: "adx_period", Type: ParamInteger, Default: 14, Min: 7, Max: 28, Step: 7, Group: "Filter", Description: "ADX lookback period for trend strength filter"},
		{Name: "adx_threshold", Type: ParamContinuous, Default: 20.0, Min: 5, Max: 35, Step: 5, Group: "Filter", Description: "Minimum ADX value to allow entry (lower = more signals)"},
		{Name: "chop_threshold", Type: ParamContinuous, Default: 61.8, Min: 50, Max: 70, Step: 2, Group: "Filter", Description: "Choppiness Index threshold above which entries are blocked"},
	}
}

func (r *TrendRunner) Evaluate(candle Candle, regime int8) *Signal {
	if regime == 3 {
		return nil
	}

	price := candle.Close
	if price.IsZero() {
		return nil
	}

	r.prevFastEMA = r.fastEMA
	r.prevSlowEMA = r.slowEMA

	if r.fastEMA <= 0 && r.slowEMA <= 0 {
		r.fastEMA = price.Float64()
		r.slowEMA = price.Float64()
	} else {
		fastAlpha := 2.0 / (r.FastPeriod + 1.0)
		slowAlpha := 2.0 / (r.SlowPeriod + 1.0)
		r.fastEMA = price.Float64()*fastAlpha + r.fastEMA*(1.0-fastAlpha)
		r.slowEMA = price.Float64()*slowAlpha + r.slowEMA*(1.0-slowAlpha)
	}

	r.PushPrice(price, candle.High, candle.Low, 0)

	adx := ADX(r.PriceHistory, r.HighHistory, r.LowHistory, r.HistCount, int(r.AdxPeriod))
	chop := ChoppinessIndex(r.PriceHistory, r.HighHistory, r.LowHistory, r.HistCount, int(r.AtrPeriod))

	tc := TakeProfitChecker{}

	if r.PositionOpen {
		if r.CurrentSide == "BUY" {
			if price.Compare(r.peakPrice) > 0 {
				r.peakPrice = price
			}
			trailingStop := types.PriceFromFloat(r.peakPrice.Float64() - r.StopLoss.Float64())
			crossDown := r.fastEMA < r.slowEMA && r.prevFastEMA >= r.prevSlowEMA
			if tc.IsTakeProfitHit(price, r.TakeProfit, "BUY") || price.Compare(trailingStop) <= 0 || crossDown {
				r.ClosePosition()
				r.signalPending = false
				return &Signal{Symbol: candle.Symbol, Side: "SELL", Quantity: 0}
			}
		} else {
			if price.Compare(r.peakPrice) < 0 {
				r.peakPrice = price
			}
			trailingStop := types.PriceFromFloat(r.peakPrice.Float64() + r.StopLoss.Float64())
			crossUp := r.fastEMA > r.slowEMA && r.prevFastEMA <= r.prevSlowEMA
			if tc.IsTakeProfitHit(price, r.TakeProfit, "SELL") || price.Compare(trailingStop) >= 0 || crossUp {
				r.ClosePosition()
				r.signalPending = false
				return &Signal{Symbol: candle.Symbol, Side: "BUY", Quantity: 0}
			}
		}
		return nil
	}

	if !r.PositionOpen && r.prevFastEMA > 0 && r.prevSlowEMA > 0 {
		crossUp := r.prevFastEMA <= r.prevSlowEMA && r.fastEMA > r.slowEMA
		crossDown := r.prevFastEMA >= r.prevSlowEMA && r.fastEMA < r.slowEMA

		if !crossUp && !crossDown {
			r.signalPending = false
			return nil
		}

		if r.signalPending {
			if adx < r.AdxThreshold {
				r.signalPending = false
				return nil
			}
			if r.ChopThreshold > 0 && chop > r.ChopThreshold {
				r.signalPending = false
				return nil
			}
			atr := ATR(r.PriceHistory, r.HistCount, int(r.AtrPeriod))
			if atr <= 0 {
				r.signalPending = false
				return nil
			}
			stopMult, profitMult := r.RegimeExitMults(regime)
			stopDist := atr * r.AtrMultiplier * stopMult
			profitDist := atr * r.ProfitATRMult * profitMult
			r.peakPrice = price

			if r.pendingSide == "BUY" {
				r.OpenPosition("BUY", price, types.PriceFromFloat(price.Float64()-stopDist), types.PriceFromFloat(price.Float64()+profitDist), candle.Time)
				r.signalPending = false
				return &Signal{Symbol: candle.Symbol, Side: "BUY", Quantity: 1.0}
			}
			r.OpenPosition("SELL", price, types.PriceFromFloat(price.Float64()+stopDist), types.PriceFromFloat(price.Float64()-profitDist), candle.Time)
			r.signalPending = false
			return &Signal{Symbol: candle.Symbol, Side: "SELL", Quantity: 1.0}
		}

		r.signalPending = true
		if crossUp {
			r.pendingSide = "BUY"
		} else {
			r.pendingSide = "SELL"
		}
	}

	return nil
}
