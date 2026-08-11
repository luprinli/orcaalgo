package strategy

import (
	"math"

	"github.com/lee-econ/orca-core/internal/types"
)

// VolHarvestingRunner implements a volatility harvesting / short-volatility
// strategy. It is active in HighVol regime (regime 2) and enters when VIX
// exceeds the entry threshold. The strategy shorts volatility by entering
// directional positions with tight stops, aiming to capture the volatility
// risk premium as markets revert from elevated vol.
type VolHarvestingRunner struct {
	*BaseRunner

	// VIX entry threshold — only short vol when VIX > this level.
	VIXThreshold float64

	// MaxVega is the maximum notional exposure per trade as a fraction of
	// running capital (default: 0.02 = 2%).
	MaxVega float64

	// StopATRMult is the ATR multiplier for stop-loss distance.
	StopATRMult float64

	// ProfitATRMult is the ATR multiplier for take-profit distance.
	ProfitATRMult float64

	// MeanRevLookback is the lookback period for the z-score computation
	// used to identify short-term mean reversion after a vol spike.
	MeanRevLookback int

	// MeanRevEntryZ is the z-score threshold for entry.
	MeanRevEntryZ float64

	// MeanRevExitZ is the z-score threshold for exit.
	MeanRevExitZ float64

	// Current VIX value (updated externally by the engine).
	CurrentVIX float64
}

func NewVolHarvestingRunner() *VolHarvestingRunner {
	return &VolHarvestingRunner{
		BaseRunner:       NewBaseRunner(128),
		VIXThreshold:       20.0,
		MaxVega:          0.02,
		StopATRMult:      2.0,
		ProfitATRMult:    3.0,
		MeanRevLookback:  10,
		MeanRevEntryZ:    1.5,
		MeanRevExitZ:     0.5,
	}
}

func (r *VolHarvestingRunner) Name() string     { return "volatility_harvesting" }
func (r *VolHarvestingRunner) Type() string     { return "volatility_harvesting" }
func (r *VolHarvestingRunner) Version() (string, string) { return r.BaseRunner.Version() }

func (r *VolHarvestingRunner) Reset() {
	r.BaseRunner.Reset()
}

func (r *VolHarvestingRunner) Params() map[string]float64 {
	return map[string]float64{
		"vix_threshold":      r.VIXThreshold,
		"max_vega":           r.MaxVega,
		"stop_atr_mult":      r.StopATRMult,
		"profit_atr_mult":    r.ProfitATRMult,
		"mean_rev_lookback":  float64(r.MeanRevLookback),
		"mean_rev_entry_z":   r.MeanRevEntryZ,
		"mean_rev_exit_z":    r.MeanRevExitZ,
	}
}

func (r *VolHarvestingRunner) SetParams(params map[string]float64) {
	if v, ok := params["vix_threshold"]; ok {
		r.VIXThreshold = v
	}
	if v, ok := params["max_vega"]; ok {
		r.MaxVega = v
	}
	if v, ok := params["stop_atr_mult"]; ok {
		r.StopATRMult = v
	}
	if v, ok := params["profit_atr_mult"]; ok {
		r.ProfitATRMult = v
	}
	if v, ok := params["mean_rev_lookback"]; ok {
		r.MeanRevLookback = int(v)
	}
	if v, ok := params["mean_rev_entry_z"]; ok {
		r.MeanRevEntryZ = v
	}
	if v, ok := params["mean_rev_exit_z"]; ok {
		r.MeanRevExitZ = v
	}
}

func (r *VolHarvestingRunner) ParamDefs() []ParamDef {
	return []ParamDef{
		{Name: "vix_threshold", Type: ParamContinuous, Default: 20.0, Min: 15, Max: 40, Step: 1.0, Group: "Entry", Description: "Minimum VIX level to enter short-vol positions"},
		{Name: "max_vega", Type: ParamContinuous, Default: 0.02, Min: 0.005, Max: 0.05, Step: 0.005, Group: "Sizing", Description: "Max notional exposure per trade as fraction of capital"},
		{Name: "stop_atr_mult", Type: ParamContinuous, Default: 2.0, Min: 1.0, Max: 4.0, Step: 0.5, Group: "Risk", Description: "ATR multiplier for stop-loss distance"},
		{Name: "profit_atr_mult", Type: ParamContinuous, Default: 3.0, Min: 1.5, Max: 5.0, Step: 0.5, Group: "Risk", Description: "ATR multiplier for take-profit distance"},
		{Name: "mean_rev_lookback", Type: ParamInteger, Default: 10, Min: 5, Max: 30, Step: 5, Group: "Signal", Description: "Lookback bars for mean-reversion z-score"},
		{Name: "mean_rev_entry_z", Type: ParamContinuous, Default: 1.5, Min: 0.75, Max: 3.0, Step: 0.25, Group: "Signal", Description: "Z-score threshold for entry"},
		{Name: "mean_rev_exit_z", Type: ParamContinuous, Default: 0.5, Min: 0.0, Max: 1.5, Step: 0.25, Group: "Signal", Description: "Z-score threshold for exit"},
	}
}

// SetVIX updates the current VIX reading. Called by the engine each bar.
func (r *VolHarvestingRunner) SetVIX(vix float64) {
	r.CurrentVIX = vix
}

func (r *VolHarvestingRunner) Evaluate(candle Candle, regime int8) *Signal {
	price := candle.Close
	if price.IsZero() {
		return nil
	}

	r.PushPrice(price, candle.High, candle.Low, candle.Volume)

	// VIX gate: no entry if VIX is below threshold.
	if r.CurrentVIX < r.VIXThreshold {
		return nil
	}

	// Compute z-score for mean-reversion detection after vol spike.
	mean, stdDev, atr := r.computeStats()
	if stdDev <= 0 || atr <= 0 {
		return nil
	}
	currentPrice := price.Float64()
	zScore := (currentPrice - mean) / stdDev

	tc := TakeProfitChecker{}

	// Exit management
	if r.PositionOpen {
		exitZ := r.MeanRevExitZ
		if r.CurrentSide == "BUY" {
			stopPrice := r.EntryPrice.Float64() - atr*r.StopATRMult
			if currentPrice <= stopPrice ||
				tc.IsTakeProfitHit(price, r.TakeProfit, "BUY") ||
				zScore >= exitZ {
				r.ClosePosition()
				return &Signal{Symbol: candle.Symbol, Side: "SELL", Quantity: 0}
			}
		} else {
			stopPrice := r.EntryPrice.Float64() + atr*r.StopATRMult
			if currentPrice >= stopPrice ||
				tc.IsTakeProfitHit(price, r.TakeProfit, "SELL") ||
				zScore <= -exitZ {
				r.ClosePosition()
				return &Signal{Symbol: candle.Symbol, Side: "BUY", Quantity: 0}
			}
		}
		return nil
	}

	// Entry: mean-reversion after vol spike.
	// Short vol means fading the extreme move.
	if zScore <= -r.MeanRevEntryZ {
		// Oversold after vol spike → go LONG (fade the move).
		stopPrice := currentPrice - atr*r.StopATRMult
		profitPrice := currentPrice + atr*r.ProfitATRMult
		r.OpenPosition("BUY", price,
			types.PriceFromFloat(stopPrice),
			types.PriceFromFloat(profitPrice),
			candle.Time)
		return &Signal{Symbol: candle.Symbol, Side: "BUY", Quantity: 1.0}
	}

	if zScore >= r.MeanRevEntryZ {
		// Overbought after vol spike → go SHORT (fade the move).
		stopPrice := currentPrice + atr*r.StopATRMult
		profitPrice := currentPrice - atr*r.ProfitATRMult
		r.OpenPosition("SELL", price,
			types.PriceFromFloat(stopPrice),
			types.PriceFromFloat(profitPrice),
			candle.Time)
		return &Signal{Symbol: candle.Symbol, Side: "SELL", Quantity: 1.0}
	}

	return nil
}

func (r *VolHarvestingRunner) computeStats() (mean, stdDev, atr float64) {
	n := r.HistCount
	if n < r.MeanRevLookback {
		return 0, 0, 0
	}

	start := n - r.MeanRevLookback
	if start < 0 {
		start = 0
	}

	var sum float64
	count := 0
	for i := start; i < n; i++ {
		idx := i % r.BufferSize
		val := r.PriceHistory[idx]
		if val > 0 {
			sum += val
			count++
		}
	}
	if count < 2 {
		return 0, 0, 0
	}
	mean = sum / float64(count)

	var variance float64
	for i := start; i < n; i++ {
		idx := i % r.BufferSize
		val := r.PriceHistory[idx]
		if val > 0 {
			diff := val - mean
			variance += diff * diff
		}
	}
	stdDev = math.Sqrt(variance / float64(count-1))

	atr = ATR(r.PriceHistory, n, 14)
	return
}
