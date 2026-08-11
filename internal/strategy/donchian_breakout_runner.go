package strategy

import "github.com/lee-econ/orca-core/internal/types"

// DonchianBreakoutRunner implements a Donchian Channel breakout strategy.
// Entry: price breaks above Donchian upper band → BUY, breaks below lower band → SELL.
// Exit: reverse breakout, trailing stop, or price crosses back inside channel.
type DonchianBreakoutRunner struct {
	*BaseRunner

	ChannelPeriod   float64
	EntryBufferPct  float64
	AtrPeriod       float64
	AtrMultiplier   float64
	MinRangePct     float64
	peakPrice       types.Price
}

func NewDonchianBreakoutRunner() *DonchianBreakoutRunner {
	return &DonchianBreakoutRunner{
		BaseRunner:     NewBaseRunner(256),
		ChannelPeriod:  20,
		EntryBufferPct: 0.05,
		AtrPeriod:      14,
		AtrMultiplier:  2.0,
		MinRangePct:    0.5,
	}
}

func (r *DonchianBreakoutRunner) Name() string { return "donchian_breakout" }
func (r *DonchianBreakoutRunner) Type() string { return "breakout" }
func (r *DonchianBreakoutRunner) Version() (irVersion string, canonicalVersion string) { return r.BaseRunner.Version() }

func (r *DonchianBreakoutRunner) Reset() {
	r.BaseRunner.Reset()
	r.peakPrice = 0
}

func (r *DonchianBreakoutRunner) Params() map[string]float64 {
	return map[string]float64{
		"channel_period":   r.ChannelPeriod,
		"entry_buffer_pct": r.EntryBufferPct,
		"atr_period":       r.AtrPeriod,
		"atr_multiplier":   r.AtrMultiplier,
		"min_range_pct":    r.MinRangePct,
	}
}

func (r *DonchianBreakoutRunner) SetParams(params map[string]float64) {
	if v, ok := params["channel_period"]; ok {
		r.ChannelPeriod = v
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
	if v, ok := params["min_range_pct"]; ok {
		r.MinRangePct = v
	}
}

func (r *DonchianBreakoutRunner) ParamDefs() []ParamDef {
	return []ParamDef{
		{Name: "channel_period", Type: ParamInteger, Default: 20, Min: 10, Max: 60, Step: 5, Group: "Entry", Description: "Donchian channel lookback period"},
		{Name: "entry_buffer_pct", Type: ParamContinuous, Default: 0.05, Min: 0, Max: 0.5, Step: 0.05, Group: "Entry", Description: "Buffer percentage beyond band to confirm breakout"},
		{Name: "atr_period", Type: ParamInteger, Default: 14, Min: 7, Max: 28, Step: 7, Group: "Risk", Description: "ATR lookback for stop/target calculation"},
		{Name: "atr_multiplier", Type: ParamContinuous, Default: 2.0, Min: 1.0, Max: 4.0, Step: 0.5, Group: "Risk", Description: "ATR multiplier for stop distance"},
		{Name: "min_range_pct", Type: ParamContinuous, Default: 0.2, Min: 0.1, Max: 3.0, Step: 0.1, Group: "Filter", Description: "Minimum channel range as % of mid-price to trade"},
	}
}

func (r *DonchianBreakoutRunner) Evaluate(candle Candle, regime int8) *Signal {
	if regime == 3 {
		return nil
	}

	price := candle.Close
	if price.IsZero() {
		return nil
	}

	r.PushPriceOnly(price)
	sc := StopLossChecker{}

	period := int(r.ChannelPeriod)
	if r.HistCount < period+2 {
		return nil
	}

	upperDC, _, lowerDC := DonchianChannel(r.PriceHistory, r.HistCount, period)
	if upperDC <= 0 || lowerDC <= 0 {
		return nil
	}

	atr := ATR(r.PriceHistory, r.HistCount, int(r.AtrPeriod))
	channelRange := (upperDC - lowerDC) / upperDC * 100.0
	if channelRange < r.MinRangePct {
		return nil
	}

	entryBuffer := r.EntryBufferPct / 100.0

	if r.PositionOpen {
		stopDist := atr * r.AtrMultiplier

		if r.CurrentSide == "BUY" {
			if price.Compare(r.peakPrice) > 0 {
				r.peakPrice = price
			}
			trailingStop := types.PriceFromFloat(r.peakPrice.Float64() - stopDist)
			reversal := price.Float64() < lowerDC

			if sc.IsStopLossHit(price, trailingStop, "BUY") || reversal {
				r.ClosePosition()
				return &Signal{Symbol: candle.Symbol, Side: "SELL", Quantity: 0}
			}
		} else {
			if price.Compare(r.peakPrice) < 0 {
				r.peakPrice = price
			}
			trailingStop := types.PriceFromFloat(r.peakPrice.Float64() + stopDist)
			reversal := price.Float64() > upperDC

			if sc.IsStopLossHit(price, trailingStop, "SELL") || reversal {
				r.ClosePosition()
				return &Signal{Symbol: candle.Symbol, Side: "BUY", Quantity: 0}
			}
		}
		return nil
	}

	r.peakPrice = price

	if price.Float64() >= upperDC*(1.0+entryBuffer) {
		stopDist := atr * r.AtrMultiplier
		r.OpenPosition("BUY", price, types.PriceFromFloat(price.Float64()-stopDist), types.PriceFromFloat(price.Float64()+stopDist*2), candle.Time)
		return &Signal{Symbol: candle.Symbol, Side: "BUY", Quantity: 1.0}
	}

	if price.Float64() <= lowerDC*(1.0-entryBuffer) {
		stopDist := atr * r.AtrMultiplier
		r.OpenPosition("SELL", price, types.PriceFromFloat(price.Float64()+stopDist), types.PriceFromFloat(price.Float64()-stopDist*2), candle.Time)
		return &Signal{Symbol: candle.Symbol, Side: "SELL", Quantity: 1.0}
	}

	return nil
}
