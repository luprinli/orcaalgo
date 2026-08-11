package strategy

import (
	"math"
	"time"

	"github.com/lee-econ/orca-core/internal/types"
)

// VolumeScalpRunner implements a session scalp strategy that requires volume
// confirmation before entering. It only enters when current volume exceeds
// the moving average volume multiplied by VolumeMultiplier.
type VolumeScalpRunner struct {
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
	MaxTradesPerDay   int
	dailyTradeCount   int
	currentDay        string

	openingHigh    float64
	openingLow     float64
	rangeSet       bool
	barsInRange    int
	avgVolume      float64
	volumeCount    int
	volumeBuffer   []float64
}

func NewVolumeScalpRunner() *VolumeScalpRunner {
	return &VolumeScalpRunner{
		BaseRunner:        NewBaseRunner(128),
		SessionStartHour:  9,
		SessionStartMin:   30,
		SessionEndHour:    11,
		SessionEndMin:     0,
		RangeMinutes:      5,
		EntryBufferPct:    0.1,
		VolumeMultiplier:  2.0,
		AtrPeriod:         14,
		TakeProfitAtrMult: 1.5,
		StopLossAtrMult:   1.0,
		TimeExitMinutes:   60,
		MaxTradesPerDay:   10,
		openingLow:        math.MaxFloat64,
		volumeBuffer:      make([]float64, 128),
	}
}

func (r *VolumeScalpRunner) Name() string     { return "volume_scalp" }
func (r *VolumeScalpRunner) Type() string     { return "scalp" }
func (r *VolumeScalpRunner) Version() (string, string) { return r.BaseRunner.Version() }

func (r *VolumeScalpRunner) Params() map[string]float64 {
	return map[string]float64{
		"volume_multiplier":   r.VolumeMultiplier,
		"atr_period":          r.AtrPeriod,
		"take_profit_atr_mult": r.TakeProfitAtrMult,
		"stop_loss_atr_mult":  r.StopLossAtrMult,
		"max_trades_per_day":  float64(r.MaxTradesPerDay),
	}
}

func (r *VolumeScalpRunner) SetParams(params map[string]float64) {
	if v, ok := params["volume_multiplier"]; ok { r.VolumeMultiplier = v }
	if v, ok := params["atr_period"]; ok { r.AtrPeriod = v }
	if v, ok := params["take_profit_atr_mult"]; ok { r.TakeProfitAtrMult = v }
	if v, ok := params["stop_loss_atr_mult"]; ok { r.StopLossAtrMult = v }
	if v, ok := params["max_trades_per_day"]; ok { r.MaxTradesPerDay = int(v) }
}

func (r *VolumeScalpRunner) ParamDefs() []ParamDef {
	return []ParamDef{
		{Name: "volume_multiplier", Type: ParamContinuous, Default: 2.0, Min: 1.0, Max: 4.0, Step: 0.5, Group: "Filter", Description: "Volume must exceed avg × this for entry"},
		{Name: "take_profit_atr_mult", Type: ParamContinuous, Default: 1.5, Min: 0.5, Max: 3.0, Step: 0.25, Group: "Exit", Description: "ATR multiplier for take-profit"},
		{Name: "stop_loss_atr_mult", Type: ParamContinuous, Default: 1.0, Min: 0.25, Max: 2.0, Step: 0.25, Group: "Exit", Description: "ATR multiplier for stop-loss"},
		{Name: "max_trades_per_day", Type: ParamInteger, Default: 10, Min: 1, Max: 30, Step: 1, Group: "Risk", Description: "Maximum trades per session"},
	}
}

func (r *VolumeScalpRunner) Evaluate(candle Candle, regime int8) *Signal {
	if regime == 3 {
		return nil
	}

	// Daily trade count reset.
	day := candle.Time.Format("2006-01-02")
	if day != r.currentDay {
		r.currentDay = day
		r.dailyTradeCount = 0
	}
	if r.MaxTradesPerDay > 0 && r.dailyTradeCount >= r.MaxTradesPerDay {
		return nil
	}

	// Volume: track running average.
	if r.avgVolume <= 0 {
		r.avgVolume = candle.Volume
	} else {
		alpha := 2.0 / 21.0
		r.avgVolume = candle.Volume*alpha + r.avgVolume*(1.0-alpha)
	}

	// Volume gate: require volume confirmation.
	if r.avgVolume > 0 && candle.Volume < r.avgVolume*r.VolumeMultiplier {
		return nil
	}

	price := candle.Close
	if price.IsZero() {
		return nil
	}
	r.PushPrice(price, candle.High, candle.Low, candle.Volume)

	hour, min := candle.Time.Hour(), candle.Time.Minute()
	totalMin := hour*60 + min
	sessionStartMin := r.SessionStartHour*60 + r.SessionStartMin
	sessionEndMin := r.SessionEndHour*60 + r.SessionEndMin

	// Exit management.
	if r.PositionOpen {
		barsHeld := r.barsHeld()
		if barsHeld >= int(r.TimeExitMinutes) {
			r.ClosePosition()
			exitSide := "SELL"
			if r.CurrentSide == "SELL" {
				exitSide = "BUY"
			}
			return &Signal{Symbol: candle.Symbol, Side: exitSide, Quantity: 0}
		}
		return nil
	}

	if totalMin < sessionStartMin {
		return nil
	}

	// Range accumulation phase.
	if !r.rangeSet {
		if totalMin >= sessionStartMin && totalMin < sessionStartMin+int(r.RangeMinutes) {
			if candle.High.Float64() > r.openingHigh {
				r.openingHigh = candle.High.Float64()
			}
			if candle.Low.Float64() < r.openingLow {
				r.openingLow = candle.Low.Float64()
			}
			r.barsInRange++
			if float64(r.barsInRange) >= r.RangeMinutes {
				r.rangeSet = true
			}
		}
		return nil
	}

	if totalMin > sessionEndMin {
		return nil
	}

	atr := ATR(r.PriceHistory, r.HistCount, int(r.AtrPeriod))
	if atr <= 0 {
		return nil
	}

	entryBuf := r.EntryBufferPct / 100.0
	breakoutHigh := r.openingHigh * (1.0 + entryBuf)
	breakoutLow := r.openingLow * (1.0 - entryBuf)

	qty := 1.0
	if candle.Close.Float64() >= breakoutHigh {
		stopPrice := candle.Close.Float64() - atr*r.StopLossAtrMult
		profitPrice := candle.Close.Float64() + atr*r.TakeProfitAtrMult
		r.OpenPosition("BUY", candle.Close, types.PriceFromFloat(stopPrice), types.PriceFromFloat(profitPrice), candle.Time)
		r.dailyTradeCount++
		return &Signal{Symbol: candle.Symbol, Side: "BUY", Quantity: qty}
	}

	if candle.Close.Float64() <= breakoutLow {
		stopPrice := candle.Close.Float64() + atr*r.StopLossAtrMult
		profitPrice := candle.Close.Float64() - atr*r.TakeProfitAtrMult
		r.OpenPosition("SELL", candle.Close, types.PriceFromFloat(stopPrice), types.PriceFromFloat(profitPrice), candle.Time)
		r.dailyTradeCount++
		return &Signal{Symbol: candle.Symbol, Side: "SELL", Quantity: qty}
	}

	return nil
}

func (r *VolumeScalpRunner) barsHeld() int {
	if !r.PositionOpen || r.EntryTime.IsZero() {
		return 0
	}
	return int(time.Since(r.EntryTime).Minutes())
}
