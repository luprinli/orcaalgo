package strategy

import "github.com/lee-econ/orca-core/internal/types"

// IchimokuRunner implements a multi-timeframe trend strategy using Ichimoku Cloud.
// Entry: tenkan crosses above kijun AND price is above cloud → BUY; tenkan crosses below AND price below cloud → SELL.
// Exit: tenkan crosses back, price enters cloud, or Chandelier Exit stop is hit.
type IchimokuRunner struct {
	*BaseRunner

	CloudConfirm   bool
	UseChandelier  bool
	AtrMultiplier  float64
	prevTenkan     float64
	prevKijun      float64
}

func NewIchimokuRunner() *IchimokuRunner {
	return &IchimokuRunner{
		BaseRunner:    NewBaseRunner(256),
		CloudConfirm:  true,
		UseChandelier: true,
		AtrMultiplier: 2.0,
	}
}

func (r *IchimokuRunner) Name() string { return "ichimoku_cloud" }
func (r *IchimokuRunner) Type() string { return "trend" }
func (r *IchimokuRunner) Version() (irVersion string, canonicalVersion string) { return r.BaseRunner.Version() }

func (r *IchimokuRunner) Reset() {
	r.BaseRunner.Reset()
	r.prevTenkan = 0
	r.prevKijun = 0
}

func (r *IchimokuRunner) Params() map[string]float64 {
	return map[string]float64{
		"cloud_confirm":   boolToFloat(r.CloudConfirm),
		"use_chandelier":  boolToFloat(r.UseChandelier),
		"atr_multiplier":  r.AtrMultiplier,
	}
}

func (r *IchimokuRunner) SetParams(params map[string]float64) {
	if v, ok := params["cloud_confirm"]; ok {
		r.CloudConfirm = v >= 0.5
	}
	if v, ok := params["use_chandelier"]; ok {
		r.UseChandelier = v >= 0.5
	}
	if v, ok := params["atr_multiplier"]; ok {
		r.AtrMultiplier = v
	}
}

func (r *IchimokuRunner) ParamDefs() []ParamDef {
	return []ParamDef{
		{Name: "cloud_confirm", Type: ParamInteger, Default: 1, Min: 0, Max: 1, Step: 1, Group: "Filter", Description: "Require price above cloud for BUY, below cloud for SELL"},
		{Name: "use_chandelier", Type: ParamInteger, Default: 1, Min: 0, Max: 1, Step: 1, Group: "Exit", Description: "Use Chandelier Exit for stop-loss instead of ATR"},
		{Name: "atr_multiplier", Type: ParamContinuous, Default: 2.0, Min: 1.0, Max: 4.0, Step: 0.5, Group: "Risk", Description: "ATR multiplier for trailing stop distance"},
	}
}

func (r *IchimokuRunner) Evaluate(candle Candle, regime int8) *Signal {
	if regime == 3 {
		return nil
	}

	price := candle.Close
	if price.IsZero() {
		return nil
	}

	r.PushPriceOnly(price)
	sc := StopLossChecker{}

	if r.HistCount < 52 {
		return nil
	}

	tenkan, kijun, senkouA, senkouB, _ := IchimokuCloud(r.PriceHistory, r.PriceHistory, r.PriceHistory, r.HistCount)
	if tenkan <= 0 || kijun <= 0 || senkouA <= 0 || senkouB <= 0 {
		return nil
	}

	atr := ATR(r.PriceHistory, r.HistCount, 14)

	if r.PositionOpen {
		stopDist := atr * r.AtrMultiplier

		if r.UseChandelier {
			longExit, shortExit := ChandelierExit(r.PriceHistory, r.PriceHistory, r.PriceHistory, r.HistCount)
			if longExit > 0 {
				stopDist = price.Float64() - longExit
				if stopDist < 0 {
					stopDist = atr * r.AtrMultiplier
				}
			}
			_ = shortExit
		}

		if r.CurrentSide == "BUY" {
			if tenkan < kijun && r.prevTenkan >= r.prevKijun {
				r.ClosePosition()
				return &Signal{Symbol: candle.Symbol, Side: "SELL", Quantity: 0}
			}
			if sc.IsStopLossHit(price, types.PriceFromFloat(price.Float64()-stopDist), "BUY") {
				r.ClosePosition()
				return &Signal{Symbol: candle.Symbol, Side: "SELL", Quantity: 0}
			}
		} else {
			if tenkan > kijun && r.prevTenkan <= r.prevKijun {
				r.ClosePosition()
				return &Signal{Symbol: candle.Symbol, Side: "BUY", Quantity: 0}
			}
			if sc.IsStopLossHit(price, types.PriceFromFloat(price.Float64()+stopDist), "SELL") {
				r.ClosePosition()
				return &Signal{Symbol: candle.Symbol, Side: "BUY", Quantity: 0}
			}
		}

		r.prevTenkan = tenkan
		r.prevKijun = kijun
		return nil
	}

	r.prevTenkan = tenkan
	r.prevKijun = kijun

	prevTenkanVal, prevKijunVal, prevSa, prevSb, _ := IchimokuCloud(r.PriceHistory, r.PriceHistory, r.PriceHistory, r.HistCount-1)
	if prevTenkanVal <= 0 || prevKijunVal <= 0 {
		return nil
	}
	_ = prevSa
	_ = prevSb

	cloudTop := senkouA
	cloudBottom := senkouB
	if cloudTop < cloudBottom {
		cloudTop, cloudBottom = cloudBottom, cloudTop
	}

	tenkanCrossUp := prevTenkanVal <= prevKijunVal && tenkan > kijun
	tenkanCrossDown := prevTenkanVal >= prevKijunVal && tenkan < kijun

	stopDist := atr * r.AtrMultiplier

	if tenkanCrossUp {
		if r.CloudConfirm && price.Float64() < cloudTop {
			return nil
		}
		r.OpenPosition("BUY", price, types.PriceFromFloat(price.Float64()-stopDist), types.PriceFromFloat(price.Float64()+stopDist*2), candle.Time)
		return &Signal{Symbol: candle.Symbol, Side: "BUY", Quantity: 100}
	}

	if tenkanCrossDown {
		if r.CloudConfirm && price.Float64() > cloudBottom {
			return nil
		}
		r.OpenPosition("SELL", price, types.PriceFromFloat(price.Float64()+stopDist), types.PriceFromFloat(price.Float64()-stopDist*2), candle.Time)
		return &Signal{Symbol: candle.Symbol, Side: "SELL", Quantity: 100}
	}

	return nil
}
