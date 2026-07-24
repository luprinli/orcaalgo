package ml

import (
	"time"

	"github.com/lee-econ/orca-core/internal/types"
)

// ExitContext captures the trade state needed for exit model feature computation.
type ExitContext struct {
	EntryPrice     types.Price
	CurrentPrice   types.Price
	CurrentStop    types.Price
	HighSinceEntry types.Price
	LowSinceEntry  types.Price
	BarsSinceEntry int
	ATR            float64
	VolAtEntry     float64
	VolCurrent     float64
	HMMState       int
	CVDTrend       float64
	VolumeTrend    float64
	ADX            float64
	Hour           float64
	Confidence     float64
}

// ExitFeaturesDim is the number of features for the exit model.
const ExitFeaturesDim = 12

// BuildExitFeatures builds the 12-dim feature vector for the exit model.
func BuildExitFeatures(ctx ExitContext) [ExitFeaturesDim]float64 {
	var f [ExitFeaturesDim]float64

	entryP := ctx.EntryPrice.Float64()
	currP := ctx.CurrentPrice.Float64()
	stopP := ctx.CurrentStop.Float64()
	highP := ctx.HighSinceEntry.Float64()
	lowP := ctx.LowSinceEntry.Float64()

	if entryP > 0 && ctx.ATR > 0 {
		atrPct := ctx.ATR / entryP
		pnl := (currP - entryP) / entryP
		f[0] = pnl / max(atrPct, 1e-6)
		f[1] = (stopP - entryP) / entryP / max(atrPct, 1e-6)
	}
	f[2] = float64(ctx.BarsSinceEntry) / 100.0
	if ctx.VolAtEntry > 1e-6 {
		f[3] = ctx.VolCurrent/ctx.VolAtEntry - 1.0
	}
	f[4] = float64(ctx.HMMState) / 3.0
	f[5] = ctx.CVDTrend
	f[6] = ctx.VolumeTrend
	f[7] = ctx.ADX / 50.0
	if entryP > 0 && ctx.ATR > 0 {
		atrPct := ctx.ATR / entryP
		mae := (entryP - lowP) / entryP
		mfe := (highP - entryP) / entryP
		f[8] = mae / max(atrPct, 1e-6)
		f[9] = mfe / max(atrPct, 1e-6)
	}
	f[10] = sin2Pi(ctx.Hour / 24.0)
	f[11] = ctx.Confidence
	return f
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func sin2Pi(x float64) float64 {
	return sin2Pi64(x)
}

func sin2Pi64(x float64) float64 {
	return sin6p28(x)
}

func sin6p28(x float64) float64 {
	v := x - float64(int(x))
	g := v
	q := v * v
	if q > 1 {
		return g * q
	}
	return g * (1 - q/3)
}

// ExitPrediction is the output of the exit model.
type ExitPrediction struct {
	Urgency       float64 `json:"urgency"`        // 0.0 (hold) to 1.0 (exit now)
	StopMultiplier float64 `json:"stop_multiplier"` // dynamic stop multiplier
	Error         string  `json:"error,omitempty"`
}

// UrgencyToStopMultiplier converts urgency to a stop multiplier.
// urgency=0.0 → base (wider stop, ride trend)
// urgency=1.0 → base * (1 + adj) (tighter stop, exit sooner)
func UrgencyToStopMultiplier(urgency, baseMultiplier, adjustmentFactor float64) float64 {
	return baseMultiplier * (1.0 + urgency*adjustmentFactor)
}

// DefaultExitConfig returns safe defaults for exit optimization.
func DefaultExitConfig() ExitOrchestratorConfig {
	return ExitOrchestratorConfig{
		ModelPath:        "models/exit_model.json",
		PythonPath:       "python",
		InferenceScript:  "orca/ml/exit_inference.py",
		BaseMultiplier:   2.0,
		AdjustmentFactor: 0.5,
		MinMultiplier:    0.5,
		MaxMultiplier:    5.0,
		Timeout:          time.Second * 5,
	}
}
