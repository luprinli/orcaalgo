package strategy

import "github.com/lee-econ/orca-core/internal/types"

// RSI2MeanReversionRunner implements Connors-style 2-period RSI mean reversion.
// Entry: RSI(2) drops below oversold threshold (5) → BUY, rises above overbought (95) → SELL.
// The 2-period RSI is extremely sensitive; values 0-5 signal capitulation, 95-100 signal exhaustion.
// Exits on RSI(2) crossing back above 50 (BUY exit) or below 50 (SELL exit), or ATR stop.
type RSI2MeanReversionRunner struct {
	*BaseRunner

	Oversold       float64
	Overbought     float64
	ExitNeutral    float64
	AtrPeriod      float64
	AtrMultiplier  float64
	MaxHoldBars    float64
	barsInTrade    int
}

func NewRSI2MeanReversionRunner() *RSI2MeanReversionRunner {
	return &RSI2MeanReversionRunner{
		BaseRunner:    NewBaseRunner(256),
		Oversold:      5.0,
		Overbought:    95.0,
		ExitNeutral:   50.0,
		AtrPeriod:     14,
		AtrMultiplier: 1.5,
		MaxHoldBars:   20,
	}
}

func (r *RSI2MeanReversionRunner) Name() string { return "rsi2_reversion" }
func (r *RSI2MeanReversionRunner) Type() string { return "mean_reversion" }
func (r *RSI2MeanReversionRunner) Version() (irVersion string, canonicalVersion string) { return r.BaseRunner.Version() }

func (r *RSI2MeanReversionRunner) Reset() {
	r.BaseRunner.Reset()
	r.barsInTrade = 0
}

func (r *RSI2MeanReversionRunner) Params() map[string]float64 {
	return map[string]float64{
		"oversold":       r.Oversold,
		"overbought":     r.Overbought,
		"exit_neutral":   r.ExitNeutral,
		"atr_period":     r.AtrPeriod,
		"atr_multiplier": r.AtrMultiplier,
		"max_hold_bars":  r.MaxHoldBars,
	}
}

func (r *RSI2MeanReversionRunner) SetParams(params map[string]float64) {
	if v, ok := params["oversold"]; ok {
		r.Oversold = v
	}
	if v, ok := params["overbought"]; ok {
		r.Overbought = v
	}
	if v, ok := params["exit_neutral"]; ok {
		r.ExitNeutral = v
	}
	if v, ok := params["atr_period"]; ok {
		r.AtrPeriod = v
	}
	if v, ok := params["atr_multiplier"]; ok {
		r.AtrMultiplier = v
	}
	if v, ok := params["max_hold_bars"]; ok {
		r.MaxHoldBars = v
	}
}

func (r *RSI2MeanReversionRunner) ParamDefs() []ParamDef {
	return []ParamDef{
		{Name: "oversold", Type: ParamContinuous, Default: 5, Min: 1, Max: 15, Step: 1, Group: "Entry", Description: "RSI(2) buy threshold — values below trigger BUY"},
		{Name: "overbought", Type: ParamContinuous, Default: 95, Min: 85, Max: 99, Step: 1, Group: "Entry", Description: "RSI(2) sell threshold — values above trigger SELL"},
		{Name: "exit_neutral", Type: ParamContinuous, Default: 50, Min: 40, Max: 60, Step: 5, Group: "Exit", Description: "RSI(2) exit level — cross above exits BUY, below exits SELL"},
		{Name: "atr_period", Type: ParamInteger, Default: 14, Min: 7, Max: 28, Step: 7, Group: "Risk", Description: "ATR period for stop-loss distance"},
		{Name: "atr_multiplier", Type: ParamContinuous, Default: 1.5, Min: 0.5, Max: 3.0, Step: 0.25, Group: "Risk", Description: "ATR multiplier for stop-loss"},
		{Name: "max_hold_bars", Type: ParamInteger, Default: 20, Min: 5, Max: 60, Step: 5, Group: "Exit", Description: "Max bars to hold before force-exiting"},
	}
}

func (r *RSI2MeanReversionRunner) Evaluate(candle Candle, regime int8) *Signal {
	if regime == 3 {
		return nil
	}

	price := candle.Close
	if price.IsZero() {
		return nil
	}

	r.PushPriceOnly(price)
	sc := StopLossChecker{}

	if r.PositionOpen {
		r.barsInTrade++
		atr := ATR(r.PriceHistory, r.HistCount, int(r.AtrPeriod))
		stopDist := atr * r.AtrMultiplier
		rsi2 := RSI2(r.PriceHistory, r.HistCount)

		if r.CurrentSide == "BUY" {
			if sc.IsStopLossHit(price, types.PriceFromFloat(r.EntryPrice.Float64()-stopDist), "BUY") || r.barsInTrade >= int(r.MaxHoldBars) {
				r.ClosePosition()
				return &Signal{Symbol: candle.Symbol, Side: "SELL", Quantity: 0}
			}
			if rsi2 > r.ExitNeutral && rsi2 > 0 {
				r.ClosePosition()
				return &Signal{Symbol: candle.Symbol, Side: "SELL", Quantity: 0}
			}
		} else {
			if sc.IsStopLossHit(price, types.PriceFromFloat(r.EntryPrice.Float64()+stopDist), "SELL") || r.barsInTrade >= int(r.MaxHoldBars) {
				r.ClosePosition()
				return &Signal{Symbol: candle.Symbol, Side: "BUY", Quantity: 0}
			}
			if rsi2 < r.ExitNeutral && rsi2 > 0 {
				r.ClosePosition()
				return &Signal{Symbol: candle.Symbol, Side: "BUY", Quantity: 0}
			}
		}
		return nil
	}

	if r.HistCount < 20 {
		return nil
	}

	rsi2 := RSI2(r.PriceHistory, r.HistCount)
	if rsi2 <= 0 {
		return nil
	}

	if rsi2 < r.Oversold {
		atr := ATR(r.PriceHistory, r.HistCount, int(r.AtrPeriod))
		stopDist := atr * r.AtrMultiplier
		r.OpenPosition("BUY", price, types.PriceFromFloat(price.Float64()-stopDist), types.PriceFromFloat(price.Float64()+stopDist*2), candle.Time)
		r.barsInTrade = 0
		return &Signal{Symbol: candle.Symbol, Side: "BUY", Quantity: 100}
	}

	if rsi2 > r.Overbought {
		atr := ATR(r.PriceHistory, r.HistCount, int(r.AtrPeriod))
		stopDist := atr * r.AtrMultiplier
		r.OpenPosition("SELL", price, types.PriceFromFloat(price.Float64()+stopDist), types.PriceFromFloat(price.Float64()-stopDist*2), candle.Time)
		r.barsInTrade = 0
		return &Signal{Symbol: candle.Symbol, Side: "SELL", Quantity: 100}
	}

	return nil
}
