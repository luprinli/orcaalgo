package strategy

import (
	"github.com/lee-econ/orca-core/internal/types"
)

// fxCarryBps maps FX pairs to their static interest-rate differential in basis
// points (base-rate minus quote-rate, approximate central-bank levels). Positive
// = long bias (earn carry), negative = short bias. This replaces the broken
// spot-VIX proxy (`vix_futures_carry`) with an actual carry signal. When live
// interest-rate data becomes available, this map is superseded by the
// `symbols.interest_rate` column (migration 000048).
var fxCarryBps = map[string]float64{
	"AUDUSD": 35,   // AUD ~4.35% - USD ~4.0%
	"GBPUSD": 50,   // GBP ~4.5% - USD ~4.0%
	"USDJPY": 350,  // USD ~4.0% - JPY ~0.5%
	"USDCAD": 100,  // USD ~4.0% - CAD ~3.0%
	"EURUSD": -150, // EUR ~2.5% - USD ~4.0%
}

// CarryRunner implements FX carry with a trend filter and trailing stop.
// Long positive-carry pairs, short negative-carry pairs; the trend filter
// (price vs SMA) provides timing, the ATR trailing stop manages the carry crash.
type CarryRunner struct {
	*BaseRunner

	TrendPeriod   float64
	AtrPeriod     float64
	AtrMultiplier float64
	ProfitATRMult float64
	MinCarryBps   float64

	peakPrice types.Price
}

func NewCarryRunner() *CarryRunner {
	return &CarryRunner{
		BaseRunner:    NewBaseRunner(512),
		TrendPeriod:   100,
		AtrPeriod:     20,
		AtrMultiplier: 2.0,
		ProfitATRMult: 0,
		MinCarryBps:   0,
	}
}

func (r *CarryRunner) Name() string { return "fx_carry" }
func (r *CarryRunner) Type() string { return "carry" }
func (r *CarryRunner) Version() (irVersion string, canonicalVersion string) {
	return r.BaseRunner.Version()
}

func (r *CarryRunner) Reset() {
	r.BaseRunner.Reset()
	r.peakPrice = 0
}

func (r *CarryRunner) Params() map[string]float64 {
	return map[string]float64{
		"trend_period":    r.TrendPeriod,
		"atr_period":      r.AtrPeriod,
		"atr_multiplier":  r.AtrMultiplier,
		"profit_atr_mult": r.ProfitATRMult,
		"min_carry_bps":   r.MinCarryBps,
	}
}

func (r *CarryRunner) SetParams(params map[string]float64) {
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
	if v, ok := params["min_carry_bps"]; ok {
		r.MinCarryBps = v
	}
}

func (r *CarryRunner) ParamDefs() []ParamDef {
	return []ParamDef{
		{Name: "trend_period", Type: ParamInteger, Default: 100, Min: 40, Max: 240, Step: 20, Group: "Filter", Description: "SMA trend-filter period"},
		{Name: "atr_period", Type: ParamInteger, Default: 20, Min: 7, Max: 60, Step: 7, Group: "Risk", Description: "ATR lookback for trailing stop"},
		{Name: "atr_multiplier", Type: ParamContinuous, Default: 2.0, Min: 1.0, Max: 4.0, Step: 0.5, Group: "Risk", Description: "ATR trailing-stop multiplier"},
		{Name: "profit_atr_mult", Type: ParamContinuous, Default: 0.0, Min: 0.0, Max: 10.0, Step: 1.0, Group: "Exit", Description: "ATR take-profit multiplier (0 = disabled)"},
		{Name: "min_carry_bps", Type: ParamContinuous, Default: 0.0, Min: 0.0, Max: 200.0, Step: 25.0, Group: "Filter", Description: "Minimum absolute carry (bps) to trade"},
	}
}

func (r *CarryRunner) Evaluate(candle Candle, regime int8) *Signal {
	if regime == 3 {
		return nil
	}

	carry, ok := fxCarryBps[candle.Symbol]
	if !ok {
		return nil
	}
	if carry > 0 && carry < r.MinCarryBps {
		return nil
	}
	if carry < 0 && -carry < r.MinCarryBps {
		return nil
	}

	price := candle.Close
	if price.IsZero() {
		return nil
	}

	r.PushPrice(price, candle.High, candle.Low, candle.Volume)

	if r.HistCount < int(r.TrendPeriod)+2 {
		return nil
	}

	sma := SMA(r.PriceHistory, r.HistCount, int(r.TrendPeriod))
	atr := ATR(r.PriceHistory, r.HistCount, int(r.AtrPeriod))
	stopMult, profitMult := r.RegimeExitMults(regime)

	sc := StopLossChecker{}

	if r.PositionOpen {
		stopDist := atr * r.AtrMultiplier * stopMult
		if r.CurrentSide == "BUY" {
			if price.Compare(r.peakPrice) > 0 {
				r.peakPrice = price
			}
			trailingStop := types.PriceFromFloat(r.peakPrice.Float64() - stopDist)
			if sc.IsStopLossHit(price, trailingStop, "BUY") {
				r.ClosePosition()
				return &Signal{Symbol: candle.Symbol, Side: "SELL", Quantity: 0}
			}
		} else if r.CurrentSide == "SELL" {
			if price.Compare(r.peakPrice) < 0 {
				r.peakPrice = price
			}
			trailingStop := types.PriceFromFloat(r.peakPrice.Float64() + stopDist)
			if sc.IsStopLossHit(price, trailingStop, "SELL") {
				r.ClosePosition()
				return &Signal{Symbol: candle.Symbol, Side: "BUY", Quantity: 0}
			}
		}
		return nil
	}

	r.peakPrice = price
	stopDist := atr * r.AtrMultiplier * stopMult

	if carry > 0 && sma > 0 && price.Float64() > sma {
		var tp types.Price
		if r.ProfitATRMult > 0 {
			tp = types.PriceFromFloat(price.Float64() + stopDist*r.ProfitATRMult*profitMult)
		}
		r.OpenPosition("BUY", price, types.PriceFromFloat(price.Float64()-stopDist), tp, candle.Time)
		return &Signal{Symbol: candle.Symbol, Side: "BUY", Quantity: 1.0}
	}

	if carry < 0 && sma > 0 && price.Float64() < sma {
		var tp types.Price
		if r.ProfitATRMult > 0 {
			tp = types.PriceFromFloat(price.Float64() - stopDist*r.ProfitATRMult*profitMult)
		}
		r.OpenPosition("SELL", price, types.PriceFromFloat(price.Float64()+stopDist), tp, candle.Time)
		return &Signal{Symbol: candle.Symbol, Side: "SELL", Quantity: 1.0}
	}

	return nil
}
