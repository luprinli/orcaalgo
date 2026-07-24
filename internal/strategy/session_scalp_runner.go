package strategy

import (
	"math"

	"github.com/lee-econ/orca-core/internal/types"
)

type SessionScalpRunner struct {
	*BaseRunner

	SessionStartHour  int
	SessionStartMin   int
	SessionEndHour    int
	SessionEndMin     int
	RangeMinutes      float64
	EntryBufferPct    float64
	VolumeMultiplier  float64
	AtrPeriod         float64
	TakeProfitAtrMult float64
	StopLossAtrMult   float64
	TimeExitMinutes   float64
	TimezoneOffset    int

	openingHigh  float64
	openingLow   float64
	rangeSet     bool
	barsInRange  int
	volumeBuffer []float64
}

func NewSessionScalpRunner() *SessionScalpRunner {
	return &SessionScalpRunner{
		BaseRunner:        NewBaseRunner(128),
		SessionStartHour:  9,
		SessionStartMin:   30,
		SessionEndHour:    11,
		SessionEndMin:     0,
		RangeMinutes:      5,
		EntryBufferPct:    0.1,
		VolumeMultiplier:  1.5,
		AtrPeriod:         14,
		TakeProfitAtrMult: 1.5,
		StopLossAtrMult:   0.75,
		TimeExitMinutes:   90,
		TimezoneOffset:    0,
		openingLow:        math.MaxFloat64,
		volumeBuffer:      make([]float64, 128),
	}
}

func (r *SessionScalpRunner) Name() string { return "session_scalp" }
func (r *SessionScalpRunner) Type() string { return "scalp" }
func (r *SessionScalpRunner) Version() (irVersion string, canonicalVersion string) { return r.BaseRunner.Version() }

func (r *SessionScalpRunner) Reset() {
	r.BaseRunner.Reset()
	r.openingHigh = 0
	r.openingLow = math.MaxFloat64
	r.rangeSet = false
	r.barsInRange = 0
	for i := range r.volumeBuffer {
		r.volumeBuffer[i] = 0
	}
}

func (r *SessionScalpRunner) Params() map[string]float64 {
	return map[string]float64{
		"range_minutes":       r.RangeMinutes,
		"entry_buffer_pct":    r.EntryBufferPct,
		"volume_multiplier":   r.VolumeMultiplier,
		"atr_period":          r.AtrPeriod,
		"take_profit_atr_mult": r.TakeProfitAtrMult,
		"stop_loss_atr_mult":  r.StopLossAtrMult,
		"time_exit_minutes":   r.TimeExitMinutes,
		"timezone_offset":     float64(r.TimezoneOffset),
	}
}

func (r *SessionScalpRunner) SetParams(params map[string]float64) {
	if v, ok := params["range_minutes"]; ok {
		r.RangeMinutes = v
	}
	if v, ok := params["entry_buffer_pct"]; ok {
		r.EntryBufferPct = v
	}
	if v, ok := params["volume_multiplier"]; ok {
		r.VolumeMultiplier = v
	}
	if v, ok := params["atr_period"]; ok {
		r.AtrPeriod = v
	}
	if v, ok := params["take_profit_atr_mult"]; ok {
		r.TakeProfitAtrMult = v
	}
	if v, ok := params["stop_loss_atr_mult"]; ok {
		r.StopLossAtrMult = v
	}
	if v, ok := params["time_exit_minutes"]; ok {
		r.TimeExitMinutes = v
	}
	if v, ok := params["timezone_offset"]; ok {
		r.TimezoneOffset = int(v)
	}
}

func (r *SessionScalpRunner) ParamDefs() []ParamDef {
	return []ParamDef{
		{Name: "range_minutes", Type: ParamInteger, Default: 5, Min: 1, Max: 15, Step: 1, Group: "Entry", Description: "Number of minutes to form the session opening range"},
		{Name: "entry_buffer_pct", Type: ParamContinuous, Default: 0.1, Min: 0.01, Max: 0.5, Step: 0.05, Group: "Entry", Description: "Percentage buffer beyond range to trigger breakout entry"},
		{Name: "volume_multiplier", Type: ParamContinuous, Default: 1.5, Min: 0.5, Max: 3.0, Step: 0.25, Group: "Filter", Description: "Volume confirmation multiplier (current vol > avg_vol * multiplier)"},
		{Name: "atr_period", Type: ParamInteger, Default: 14, Min: 7, Max: 28, Step: 7, Group: "Risk", Description: "ATR lookback period for stop/target calculation"},
		{Name: "take_profit_atr_mult", Type: ParamContinuous, Default: 1.5, Min: 1.0, Max: 4.0, Step: 0.5, Group: "Exit", Description: "Take-profit as multiple of ATR"},
		{Name: "stop_loss_atr_mult", Type: ParamContinuous, Default: 0.75, Min: 0.25, Max: 2.0, Step: 0.25, Group: "Risk", Description: "Stop-loss as multiple of ATR"},
		{Name: "time_exit_minutes", Type: ParamInteger, Default: 90, Min: 15, Max: 180, Step: 15, Group: "Exit", Description: "Minutes after entry to force-close position"},
		{Name: "timezone_offset", Type: ParamInteger, Default: 0, Min: -12, Max: 14, Step: 1, Group: "Session", Description: "UTC offset in hours for session window (ET = -4/-5)"},
	}
}

func (r *SessionScalpRunner) Evaluate(candle Candle, regime int8) *Signal {
	if regime == 3 {
		return nil
	}

	hour, min := candle.Time.Hour(), candle.Time.Minute()
	localHour := hour + r.TimezoneOffset
	if localHour < 0 {
		localHour += 24
	} else if localHour > 23 {
		localHour -= 24
	}
	totalMin := localHour*60 + min
	sessionStartMin := r.SessionStartHour*60 + r.SessionStartMin
	sessionEndMin := r.SessionEndHour*60 + r.SessionEndMin

	if totalMin < sessionStartMin || totalMin >= sessionEndMin {
		return nil
	}

	idx := r.HistIdx % r.BufferSize
	r.PriceHistory[idx] = candle.Close.Float64()
	r.volumeBuffer[idx] = candle.Volume
	r.HistIdx++
	if r.HistCount < r.BufferSize {
		r.HistCount++
	}

	sc := StopLossChecker{}
	tc := TakeProfitChecker{}

	if r.PositionOpen {
		elapsed := candle.Time.Sub(r.EntryTime).Minutes()
		if elapsed >= r.TimeExitMinutes {
			r.ClosePosition()
			exitSide := "SELL"
			if r.CurrentSide == "SELL" {
				exitSide = "BUY"
			}
			return &Signal{Symbol: candle.Symbol, Side: exitSide, Quantity: 0}
		}
		if r.CurrentSide == "BUY" {
			if sc.IsStopLossHit(candle.Low, r.StopLoss, "BUY") || tc.IsTakeProfitHit(candle.High, r.TakeProfit, "BUY") {
				r.ClosePosition()
				return &Signal{Symbol: candle.Symbol, Side: "SELL", Quantity: 0}
			}
		} else {
			if sc.IsStopLossHit(candle.High, r.StopLoss, "SELL") || tc.IsTakeProfitHit(candle.Low, r.TakeProfit, "SELL") {
				r.ClosePosition()
				return &Signal{Symbol: candle.Symbol, Side: "BUY", Quantity: 0}
			}
		}
		return nil
	}

	if !r.rangeSet {
		r.barsInRange++
		if candle.High.Float64() > r.openingHigh {
			r.openingHigh = candle.High.Float64()
		}
		if candle.Low.Float64() < r.openingLow {
			r.openingLow = candle.Low.Float64()
		}
		if float64(r.barsInRange) >= r.RangeMinutes {
			r.rangeSet = true
		}
		return nil
	}

	avgVol := Mean(r.volumeBuffer, r.HistCount, int(r.AtrPeriod))
	if avgVol > 0 && candle.Volume < avgVol*r.VolumeMultiplier {
		return nil
	}

	atr := ATR(r.PriceHistory, r.HistCount, int(r.AtrPeriod))
	if atr <= 0 {
		return nil
	}

	rangeHeight := r.openingHigh - r.openingLow
	if rangeHeight <= 0 {
		return nil
	}

	bufferPct := r.EntryBufferPct / 100.0
	breakoutHigh := r.openingHigh * (1.0 + bufferPct)
	breakoutLow := r.openingLow * (1.0 - bufferPct)

	qty := 100.0
	if regime == 2 {
		qty *= 0.50
	}

	if candle.Close.Float64() >= breakoutHigh {
		r.OpenPosition("BUY", candle.Close, types.PriceFromFloat(candle.Close.Float64()-atr*r.StopLossAtrMult), types.PriceFromFloat(candle.Close.Float64()+atr*r.TakeProfitAtrMult), candle.Time)
		return &Signal{Symbol: candle.Symbol, Side: "BUY", Quantity: qty}
	}

	if candle.Close.Float64() <= breakoutLow {
		r.OpenPosition("SELL", candle.Close, types.PriceFromFloat(candle.Close.Float64()+atr*r.StopLossAtrMult), types.PriceFromFloat(candle.Close.Float64()-atr*r.TakeProfitAtrMult), candle.Time)
		return &Signal{Symbol: candle.Symbol, Side: "SELL", Quantity: qty}
	}

	return nil
}
