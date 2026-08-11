package strategy

import "github.com/lee-econ/orca-core/internal/types"

type MACrossoverRunner struct {
	*BaseRunner

	FastPeriod      float64
	SlowPeriod      float64
	RsiPeriod       float64
	RsiOverbought   float64
	RsiOversold     float64
	UseRsiFilter    bool
	UseMacdFilter   bool
	UseBollExit     bool
	AtrPeriod       float64
	AtrMultiplier   float64
	prevFast        float64
	prevSlow        float64
}

func NewMACrossoverRunner() *MACrossoverRunner {
	return &MACrossoverRunner{
		BaseRunner:     NewBaseRunner(256),
		FastPeriod:     9,
		SlowPeriod:     21,
		RsiPeriod:      14,
		RsiOverbought:  70,
		RsiOversold:    30,
		UseRsiFilter:   true,
		UseMacdFilter:  true,
		UseBollExit:    true,
		AtrPeriod:      14,
		AtrMultiplier:  2.0,
	}
}

func (r *MACrossoverRunner) Name() string { return "ma_crossover" }
func (r *MACrossoverRunner) Type() string { return "trend" }
func (r *MACrossoverRunner) Version() (irVersion string, canonicalVersion string) { return r.BaseRunner.Version() }

func (r *MACrossoverRunner) Reset() {
	r.BaseRunner.Reset()
	r.prevFast = 0
	r.prevSlow = 0
}

func (r *MACrossoverRunner) Params() map[string]float64 {
	return map[string]float64{
		"fast_period":    r.FastPeriod,
		"slow_period":    r.SlowPeriod,
		"rsi_period":     r.RsiPeriod,
		"rsi_overbought": r.RsiOverbought,
		"rsi_oversold":   r.RsiOversold,
		"atr_period":     r.AtrPeriod,
		"atr_multiplier": r.AtrMultiplier,
		"use_rsi_filter":  boolToFloat(r.UseRsiFilter),
		"use_macd_filter": boolToFloat(r.UseMacdFilter),
		"use_boll_exit":   boolToFloat(r.UseBollExit),
	}
}

func (r *MACrossoverRunner) SetParams(params map[string]float64) {
	if v, ok := params["fast_period"]; ok {
		r.FastPeriod = v
	}
	if v, ok := params["slow_period"]; ok {
		r.SlowPeriod = v
	}
	if v, ok := params["rsi_period"]; ok {
		r.RsiPeriod = v
	}
	if v, ok := params["rsi_overbought"]; ok {
		r.RsiOverbought = v
	}
	if v, ok := params["rsi_oversold"]; ok {
		r.RsiOversold = v
	}
	if v, ok := params["atr_period"]; ok {
		r.AtrPeriod = v
	}
	if v, ok := params["atr_multiplier"]; ok {
		r.AtrMultiplier = v
	}
	if v, ok := params["use_rsi_filter"]; ok {
		r.UseRsiFilter = v >= 0.5
	}
	if v, ok := params["use_macd_filter"]; ok {
		r.UseMacdFilter = v >= 0.5
	}
	if v, ok := params["use_boll_exit"]; ok {
		r.UseBollExit = v >= 0.5
	}
}

func (r *MACrossoverRunner) ParamDefs() []ParamDef {
	return []ParamDef{
		{Name: "fast_period", Type: ParamInteger, Default: 9, Min: 3, Max: 30, Step: 3, Group: "Entry", Description: "Fast EMA period for crossover detection"},
		{Name: "slow_period", Type: ParamInteger, Default: 21, Min: 10, Max: 100, Step: 5, Group: "Entry", Description: "Slow EMA period for crossover detection"},
		{Name: "rsi_period", Type: ParamInteger, Default: 14, Min: 5, Max: 30, Step: 1, Group: "Filter", Description: "RSI lookback period"},
		{Name: "rsi_overbought", Type: ParamContinuous, Default: 70, Min: 60, Max: 85, Step: 5, Group: "Filter", Description: "RSI overbought threshold — skip BUY signals above this value"},
		{Name: "rsi_oversold", Type: ParamContinuous, Default: 30, Min: 15, Max: 40, Step: 5, Group: "Filter", Description: "RSI oversold threshold — skip SELL signals below this value"},
		{Name: "atr_period", Type: ParamInteger, Default: 14, Min: 7, Max: 28, Step: 7, Group: "Risk", Description: "ATR lookback period for stop-loss placement"},
		{Name: "atr_multiplier", Type: ParamContinuous, Default: 2.0, Min: 1.0, Max: 4.0, Step: 0.5, Group: "Risk", Description: "ATR multiplier for stop distance"},
		{Name: "use_rsi_filter", Type: ParamInteger, Default: 1, Min: 0, Max: 1, Step: 1, Group: "Filter", Description: "Enable RSI entry filter (0=off, 1=on)"},
		{Name: "use_macd_filter", Type: ParamInteger, Default: 1, Min: 0, Max: 1, Step: 1, Group: "Filter", Description: "Enable MACD confirmation filter (0=off, 1=on)"},
		{Name: "use_boll_exit", Type: ParamInteger, Default: 1, Min: 0, Max: 1, Step: 1, Group: "Exit", Description: "Enable Bollinger-band reversal exit (0=off, 1=on)"},
	}
}

// Evaluate processes each candle and returns a buy/sell signal.
//
// Signal logic:
//  1. Compute fast and slow EMAs on the price history.
//  2. Detect crossover: fast crosses above slow → BUY; fast crosses below slow → SELL.
//  3. If UseRsiFilter: skip BUY when RSI > overbought; skip SELL when RSI < oversold.
//  4. If UseMacdFilter: require MACD line above signal line for BUY; below for SELL.
//  5. If UseBollExit: exit when price closes outside Bollinger Bands (reversal signal).
//  6. Stop-loss: ATR-based trailing stop from entry price.
func (r *MACrossoverRunner) Evaluate(candle Candle, regime int8) *Signal {
	if regime == 3 {
		return nil
	}

	price := candle.Close
	if price.IsZero() {
		return nil
	}

	r.PushPriceOnly(price)

	fastPeriod := int(r.FastPeriod)
	slowPeriod := int(r.SlowPeriod)
	if fastPeriod <= 0 {
		fastPeriod = 9
	}
	if slowPeriod <= 0 {
		slowPeriod = 21
	}

	r.prevFast = EMA(r.PriceHistory, r.HistCount, fastPeriod)
	r.prevSlow = EMA(r.PriceHistory, r.HistCount, slowPeriod)

	fast := r.prevFast
	slow := r.prevSlow

	sc := StopLossChecker{}

	if r.PositionOpen {
		atr := ATR(r.PriceHistory, r.HistCount, int(r.AtrPeriod))
		stopDist := atr * r.AtrMultiplier

		if r.CurrentSide == "BUY" {
			trailingStop := types.PriceFromFloat(price.Float64() - stopDist)
			exhaustion := false
			if r.UseBollExit {
				_, _, lowerBB := BollingerBands(r.PriceHistory, r.HistCount)
				exhaustion = price.Float64() < lowerBB && lowerBB > 0
			}
			crossDown := fast > 0 && slow > 0 && fast < slow
			if sc.IsStopLossHit(price, trailingStop, "BUY") || crossDown || exhaustion {
				r.ClosePosition()
				return &Signal{Symbol: candle.Symbol, Side: "SELL", Quantity: 0}
			}
		} else {
			trailingStop := types.PriceFromFloat(price.Float64() + stopDist)
			exhaustion := false
			if r.UseBollExit {
				upperBB, _, _ := BollingerBands(r.PriceHistory, r.HistCount)
				exhaustion = price.Float64() > upperBB && upperBB > 0
			}
			crossUp := fast > 0 && slow > 0 && fast > slow
			if sc.IsStopLossHit(price, trailingStop, "SELL") || crossUp || exhaustion {
				r.ClosePosition()
				return &Signal{Symbol: candle.Symbol, Side: "BUY", Quantity: 0}
			}
		}
		return nil
	}

	if r.HistCount < slowPeriod+5 {
		return nil
	}

	prevFast := EMA(r.PriceHistory, r.HistCount-1, fastPeriod)
	prevSlow := EMA(r.PriceHistory, r.HistCount-1, slowPeriod)

	crossUp := prevFast > 0 && prevSlow > 0 && prevFast <= prevSlow && fast > slow
	crossDown := prevFast > 0 && prevSlow > 0 && prevFast >= prevSlow && fast < slow

	if crossUp {
		rsiVal := RSI(r.PriceHistory, r.HistCount, int(r.RsiPeriod))
		if r.UseRsiFilter && rsiVal > r.RsiOverbought {
			return nil
		}

		if r.UseMacdFilter {
			macdLine, signalLine := MACD(r.PriceHistory, r.HistCount)
			if macdLine <= signalLine {
				return nil
			}
		}

		atr := ATR(r.PriceHistory, r.HistCount, int(r.AtrPeriod))
		stopDist := atr * r.AtrMultiplier
		r.OpenPosition("BUY", price, types.PriceFromFloat(price.Float64()-stopDist), types.PriceFromFloat(price.Float64()+stopDist*2), candle.Time)
		return &Signal{Symbol: candle.Symbol, Side: "BUY", Quantity: 1.0}
	}

	if crossDown {
		rsiVal := RSI(r.PriceHistory, r.HistCount, int(r.RsiPeriod))
		if r.UseRsiFilter && rsiVal < r.RsiOversold {
			return nil
		}

		if r.UseMacdFilter {
			macdLine, signalLine := MACD(r.PriceHistory, r.HistCount)
			if macdLine >= signalLine {
				return nil
			}
		}

		atr := ATR(r.PriceHistory, r.HistCount, int(r.AtrPeriod))
		stopDist := atr * r.AtrMultiplier
		r.OpenPosition("SELL", price, types.PriceFromFloat(price.Float64()+stopDist), types.PriceFromFloat(price.Float64()-stopDist*2), candle.Time)
		return &Signal{Symbol: candle.Symbol, Side: "SELL", Quantity: 1.0}
	}

	return nil
}
