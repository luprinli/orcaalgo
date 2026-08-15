package strategy

import (
	"time"

	"github.com/lee-econ/orca-core/internal/types"
)

// SessionMomentumRunner implements intraday session-open momentum (time-of-day
// drift) on intraday bars. After the first N bars of the session, enter in the
// direction of the drift from the session open if volume confirms. Exit on time
// or an ATR trailing stop. Evaluate on real stooq 5m/15m bars, not synthetic.
type SessionMomentumRunner struct {
	*BaseRunner

	SessionMinutes    float64
	DriftThresholdPct float64
	VolumeMultiplier  float64
	AtrPeriod         float64
	AtrMultiplier     float64
	ProfitATRMult     float64
	TimeExitMinutes   float64

	currentDay      string
	dayOpenPrice    float64
	sessionOpenTime time.Time
	peakPrice       types.Price
}

func NewSessionMomentumRunner() *SessionMomentumRunner {
	return &SessionMomentumRunner{
		BaseRunner:        NewBaseRunner(256),
		SessionMinutes:    30,
		DriftThresholdPct: 0.15,
		VolumeMultiplier:  1.5,
		AtrPeriod:         14,
		AtrMultiplier:     2.0,
		ProfitATRMult:     0,
		TimeExitMinutes:   120,
	}
}

func (r *SessionMomentumRunner) Name() string { return "session_momentum" }
func (r *SessionMomentumRunner) Type() string { return "momentum" }
func (r *SessionMomentumRunner) Version() (irVersion string, canonicalVersion string) {
	return r.BaseRunner.Version()
}

func (r *SessionMomentumRunner) Reset() {
	r.BaseRunner.Reset()
	r.currentDay = ""
	r.dayOpenPrice = 0
	r.sessionOpenTime = time.Time{}
	r.peakPrice = 0
}

func (r *SessionMomentumRunner) Params() map[string]float64 {
	return map[string]float64{
		"session_minutes":     r.SessionMinutes,
		"drift_threshold_pct": r.DriftThresholdPct,
		"volume_multiplier":   r.VolumeMultiplier,
		"atr_period":          r.AtrPeriod,
		"atr_multiplier":      r.AtrMultiplier,
		"profit_atr_mult":     r.ProfitATRMult,
		"time_exit_minutes":   r.TimeExitMinutes,
	}
}

func (r *SessionMomentumRunner) SetParams(params map[string]float64) {
	if v, ok := params["session_minutes"]; ok {
		r.SessionMinutes = v
	}
	if v, ok := params["drift_threshold_pct"]; ok {
		r.DriftThresholdPct = v
	}
	if v, ok := params["volume_multiplier"]; ok {
		r.VolumeMultiplier = v
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
	if v, ok := params["time_exit_minutes"]; ok {
		r.TimeExitMinutes = v
	}
}

func (r *SessionMomentumRunner) ParamDefs() []ParamDef {
	return []ParamDef{
		{Name: "session_minutes", Type: ParamInteger, Default: 30, Min: 5, Max: 120, Step: 5, Group: "Entry", Description: "Bars after session open to measure drift"},
		{Name: "drift_threshold_pct", Type: ParamContinuous, Default: 0.10, Min: 0.02, Max: 1.0, Step: 0.02, Group: "Entry", Description: "Minimum drift from session open to enter (%)"},
		{Name: "volume_multiplier", Type: ParamContinuous, Default: 1.5, Min: 0.5, Max: 3.0, Step: 0.25, Group: "Filter", Description: "Volume confirmation multiplier"},
		{Name: "atr_period", Type: ParamInteger, Default: 14, Min: 7, Max: 28, Step: 7, Group: "Risk", Description: "ATR lookback for trailing stop"},
		{Name: "atr_multiplier", Type: ParamContinuous, Default: 2.0, Min: 1.0, Max: 4.0, Step: 0.5, Group: "Risk", Description: "ATR trailing-stop multiplier"},
		{Name: "profit_atr_mult", Type: ParamContinuous, Default: 0.0, Min: 0.0, Max: 10.0, Step: 1.0, Group: "Exit", Description: "ATR take-profit multiplier (0 = disabled)"},
		{Name: "time_exit_minutes", Type: ParamInteger, Default: 120, Min: 30, Max: 390, Step: 30, Group: "Exit", Description: "Minutes after entry to force-close"},
	}
}

func (r *SessionMomentumRunner) Evaluate(candle Candle, regime int8) *Signal {
	price := candle.Close
	if price.IsZero() {
		return nil
	}

	day := candle.Time.Format("2006-01-02")
	if day != r.currentDay {
		r.currentDay = day
		r.dayOpenPrice = candle.Open.Float64()
		r.sessionOpenTime = candle.Time
	}

	r.PushPrice(price, candle.High, candle.Low, candle.Volume)

	sc := StopLossChecker{}

	if r.PositionOpen {
		if candle.Time.Sub(r.EntryTime).Minutes() >= r.TimeExitMinutes {
			r.ClosePosition()
			side := "SELL"
			if r.CurrentSide == "SELL" {
				side = "BUY"
			}
			return &Signal{Symbol: candle.Symbol, Side: side, Action: SignalExit}
		}
		stopMult, _ := r.RegimeExitMults(regime)
		stopDist := TrueRangeATR(r.HighHistory, r.LowHistory, r.PriceHistory, r.HistCount, int(r.AtrPeriod)) * r.AtrMultiplier * stopMult
		if r.CurrentSide == "BUY" {
			if price.Compare(r.peakPrice) > 0 {
				r.peakPrice = price
			}
			if sc.IsStopLossHit(price, types.PriceFromFloat(r.peakPrice.Float64()-stopDist), "BUY") {
				r.ClosePosition()
				return &Signal{Symbol: candle.Symbol, Side: "SELL", Action: SignalExit}
			}
		} else {
			if price.Compare(r.peakPrice) < 0 {
				r.peakPrice = price
			}
			if sc.IsStopLossHit(price, types.PriceFromFloat(r.peakPrice.Float64()+stopDist), "SELL") {
				r.ClosePosition()
				return &Signal{Symbol: candle.Symbol, Side: "BUY", Action: SignalExit}
			}
		}
		return nil
	}

	if r.sessionOpenTime.IsZero() || r.dayOpenPrice <= 0 {
		return nil
	}
	elapsedMinutes := candle.Time.Sub(r.sessionOpenTime).Minutes()
	if elapsedMinutes < r.SessionMinutes || elapsedMinutes > r.SessionMinutes+30 {
		return nil
	}

	drift := (price.Float64() - r.dayOpenPrice) / r.dayOpenPrice * 100.0
	if drift > -r.DriftThresholdPct && drift < r.DriftThresholdPct {
		return nil
	}

	avgVol := Mean(r.VolumeHistory, r.HistCount, int(r.AtrPeriod))
	if r.VolumeMultiplier > 0 && avgVol > 0 && candle.Volume < avgVol*r.VolumeMultiplier {
		return nil
	}

	atr := TrueRangeATR(r.HighHistory, r.LowHistory, r.PriceHistory, r.HistCount, int(r.AtrPeriod))
	if atr <= 0 {
		return nil
	}
	stopMult, profitMult := r.RegimeExitMults(regime)
	stopDist := atr * r.AtrMultiplier * stopMult
	r.peakPrice = price

	if drift > 0 {
		var tp types.Price
		if r.ProfitATRMult > 0 {
			tp = types.PriceFromFloat(price.Float64() + stopDist*r.ProfitATRMult*profitMult)
		}
		r.OpenPosition("BUY", price, types.PriceFromFloat(price.Float64()-stopDist), tp, candle.Time)
		return &Signal{Symbol: candle.Symbol, Side: "BUY", Action: SignalEntry, Quantity: 1.0}
	}

	var tp types.Price
	if r.ProfitATRMult > 0 {
		tp = types.PriceFromFloat(price.Float64() - stopDist*r.ProfitATRMult*profitMult)
	}
	r.OpenPosition("SELL", price, types.PriceFromFloat(price.Float64()+stopDist), tp, candle.Time)
	return &Signal{Symbol: candle.Symbol, Side: "SELL", Action: SignalEntry, Quantity: 1.0}
}
