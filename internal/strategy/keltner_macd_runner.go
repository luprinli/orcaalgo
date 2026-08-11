package strategy

import "github.com/lee-econ/orca-core/internal/types"

// KeltnerMACDRunner combines Keltner Channel volatility bands with MACD momentum.
// Entry: price breaks above Keltner upper with MACD bullish → BUY; breaks below lower with MACD bearish → SELL.
// Exit: price reverts to middle band or MACD signal flips.
type KeltnerMACDRunner struct {
	*BaseRunner

	KeltnerPeriod   float64
	MacdRequirement bool
	AtrMultiplier   float64
	peakPrice       types.Price
}

func NewKeltnerMACDRunner() *KeltnerMACDRunner {
	return &KeltnerMACDRunner{
		BaseRunner:      NewBaseRunner(256),
		KeltnerPeriod:   20,
		MacdRequirement: true,
		AtrMultiplier:   2.0,
	}
}

func (r *KeltnerMACDRunner) Name() string { return "keltner_macd" }
func (r *KeltnerMACDRunner) Type() string { return "trend" }
func (r *KeltnerMACDRunner) Version() (irVersion string, canonicalVersion string) { return r.BaseRunner.Version() }

func (r *KeltnerMACDRunner) Reset() {
	r.BaseRunner.Reset()
	r.peakPrice = 0
}

func (r *KeltnerMACDRunner) Params() map[string]float64 {
	return map[string]float64{
		"keltner_period":   r.KeltnerPeriod,
		"macd_requirement": boolToFloat(r.MacdRequirement),
		"atr_multiplier":   r.AtrMultiplier,
	}
}

func (r *KeltnerMACDRunner) SetParams(params map[string]float64) {
	if v, ok := params["keltner_period"]; ok {
		r.KeltnerPeriod = v
	}
	if v, ok := params["macd_requirement"]; ok {
		r.MacdRequirement = v >= 0.5
	}
	if v, ok := params["atr_multiplier"]; ok {
		r.AtrMultiplier = v
	}
}

func (r *KeltnerMACDRunner) ParamDefs() []ParamDef {
	return []ParamDef{
		{Name: "keltner_period", Type: ParamInteger, Default: 20, Min: 10, Max: 50, Step: 5, Group: "Entry", Description: "Keltner Channel lookback period"},
		{Name: "macd_requirement", Type: ParamInteger, Default: 1, Min: 0, Max: 1, Step: 1, Group: "Filter", Description: "Require MACD confirmation (1=yes, 0=no)"},
		{Name: "atr_multiplier", Type: ParamContinuous, Default: 2.0, Min: 1.0, Max: 4.0, Step: 0.5, Group: "Risk", Description: "ATR multiplier for trailing stop"},
	}
}

func (r *KeltnerMACDRunner) Evaluate(candle Candle, regime int8) *Signal {
	if regime == 3 {
		return nil
	}

	price := candle.Close
	if price.IsZero() {
		return nil
	}

	r.PushPriceOnly(price)

	period := int(r.KeltnerPeriod)
	if r.HistCount < period+5 {
		return nil
	}

	upperKC, middleKC, lowerKC := KeltnerChannel(r.PriceHistory, r.PriceHistory, r.PriceHistory, r.HistCount, period)
	if upperKC <= 0 || middleKC <= 0 || lowerKC <= 0 {
		return nil
	}

	atr := ATR(r.PriceHistory, r.HistCount, int(r.KeltnerPeriod))
	macdLine, signalLine := MACD(r.PriceHistory, r.HistCount)
	macdBullish := macdLine > signalLine

	sc := StopLossChecker{}

	if r.PositionOpen {
		stopDist := atr * r.AtrMultiplier

		if r.CurrentSide == "BUY" {
			if price.Compare(r.peakPrice) > 0 {
				r.peakPrice = price
			}
			trailingStop := types.PriceFromFloat(r.peakPrice.Float64() - stopDist)
			macdFlip := r.MacdRequirement && !macdBullish

			if sc.IsStopLossHit(price, trailingStop, "BUY") || macdFlip {
				r.ClosePosition()
				return &Signal{Symbol: candle.Symbol, Side: "SELL", Quantity: 0}
			}
		} else {
			if price.Compare(r.peakPrice) < 0 {
				r.peakPrice = price
			}
			trailingStop := types.PriceFromFloat(r.peakPrice.Float64() + stopDist)
			macdFlip := r.MacdRequirement && macdBullish

			if sc.IsStopLossHit(price, trailingStop, "SELL") || macdFlip {
				r.ClosePosition()
				return &Signal{Symbol: candle.Symbol, Side: "BUY", Quantity: 0}
			}
		}
		return nil
	}

	r.peakPrice = price

	if price.Float64() > upperKC {
		if r.MacdRequirement && !macdBullish {
			return nil
		}
		stopDist := atr * r.AtrMultiplier
		r.OpenPosition("BUY", price, types.PriceFromFloat(price.Float64()-stopDist), types.PriceFromFloat(price.Float64()+stopDist*2), candle.Time)
		return &Signal{Symbol: candle.Symbol, Side: "BUY", Quantity: 1.0}
	}

	if price.Float64() < lowerKC {
		if r.MacdRequirement && macdBullish {
			return nil
		}
		stopDist := atr * r.AtrMultiplier
		r.OpenPosition("SELL", price, types.PriceFromFloat(price.Float64()+stopDist), types.PriceFromFloat(price.Float64()-stopDist*2), candle.Time)
		return &Signal{Symbol: candle.Symbol, Side: "SELL", Quantity: 1.0}
	}

	return nil
}

func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}
