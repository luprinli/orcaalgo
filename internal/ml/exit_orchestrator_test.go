package ml

import (
	"math"
	"testing"

	"github.com/lee-econ/orca-core/internal/types"
)

func TestBuildExitFeatures(t *testing.T) {
	ctx := ExitContext{
		EntryPrice:     types.FromFloat64(100.0),
		CurrentPrice:   types.FromFloat64(102.0),
		CurrentStop:    types.FromFloat64(98.0),
		HighSinceEntry: types.FromFloat64(103.0),
		LowSinceEntry:  types.FromFloat64(97.0),
		BarsSinceEntry: 5,
		ATR:            1.0,
		VolAtEntry:     0.01,
		VolCurrent:     0.012,
		HMMState:       1,
		CVDTrend:       0.5,
		VolumeTrend:    -0.2,
		ADX:            25.0,
		Hour:           12.0,
		Confidence:     0.75,
	}

	fv := BuildExitFeatures(ctx)
	if len(fv) != ExitFeaturesDim {
		t.Errorf("expected %d features, got %d", ExitFeaturesDim, len(fv))
	}

	// PnL should be positive
	if fv[0] <= 0 {
		t.Error("pnl_atr should be positive for winning trade")
	}

	// Check specific values
	if fv[2] != 0.05 {
		t.Errorf("bars_since_entry expected 0.05, got %f", fv[2])
	}
	if fv[5] != 0.5 {
		t.Errorf("cvd_trend expected 0.5, got %f", fv[5])
	}
	if fv[7] != 0.5 {
		t.Errorf("adx expected 0.5, got %f", fv[7])
	}
	if fv[11] != 0.75 {
		t.Errorf("confidence expected 0.75, got %f", fv[11])
	}
}

func TestBuildExitFeaturesLosingTrade(t *testing.T) {
	ctx := ExitContext{
		EntryPrice:     types.FromFloat64(100.0),
		CurrentPrice:   types.FromFloat64(98.0),
		CurrentStop:    types.FromFloat64(97.0),
		HighSinceEntry: types.FromFloat64(101.0),
		LowSinceEntry:  types.FromFloat64(97.0),
		BarsSinceEntry: 10,
		ATR:            1.0,
		VolAtEntry:     0.01,
		VolCurrent:     0.015,
		HMMState:       2,
		ADX:            35.0,
		Hour:           14.0,
		Confidence:     0.3,
	}

	fv := BuildExitFeatures(ctx)
	if fv[0] >= 0 {
		t.Error("pnl_atr should be negative for losing trade")
	}
}

func TestUrgencyToStopMultiplier(t *testing.T) {
	tests := []struct {
		urgency  float64
		expected float64
	}{
		{0.0, 2.0},
		{0.5, 2.5},
		{1.0, 3.0},
	}
	for _, tt := range tests {
		got := UrgencyToStopMultiplier(tt.urgency, 2.0, 0.5)
		if math.Abs(got-tt.expected) > 1e-9 {
			t.Errorf("UrgencyToStopMultiplier(%f) = %f, want %f", tt.urgency, got, tt.expected)
		}
	}
}

func TestExitOrchestratorDefaults(t *testing.T) {
	cfg := DefaultExitConfig()
	eo := NewExitOrchestrator(cfg)
	if !eo.IsHealthy() {
		t.Error("new exit orchestrator should be healthy")
	}

	eo.Disable()
	if eo.IsHealthy() {
		t.Error("disabled exit orchestrator should not be healthy")
	}

	// Fallback when disabled
	urgency, mult := eo.Evaluate(ExitContext{})
	if urgency != 0.0 || mult != 2.0 {
		t.Errorf("disabled should return (0, base): got (%f, %f)", urgency, mult)
	}
}

func TestExitOrchestratorComputeNewStop(t *testing.T) {
	cfg := DefaultExitConfig()
	eo := NewExitOrchestrator(cfg)
	eo.Disable()

	ctx := ExitContext{
		EntryPrice:   types.FromFloat64(100.0),
		CurrentPrice: types.FromFloat64(102.0),
		ATR:          1.0,
	}

	newStop := eo.ComputeNewStop("BUY", types.FromFloat64(100.0), types.FromFloat64(102.0), 1.0, ctx)
	// With disabled model: base_multiplier=2.0, stop = 102 - 2*1 = 100
	if newStop > 102.0 || newStop < 80.0 {
		t.Errorf("unexpected stop price: %f", newStop)
	}
}

func TestClamp(t *testing.T) {
	if v := clamp(5.0, 0.0, 10.0); v != 5.0 {
		t.Errorf("clamp(5, 0, 10) = %f", v)
	}
	if v := clamp(-1.0, 0.0, 10.0); v != 0.0 {
		t.Errorf("clamp(-1, 0, 10) = %f", v)
	}
	if v := clamp(15.0, 0.0, 10.0); v != 10.0 {
		t.Errorf("clamp(15, 0, 10) = %f", v)
	}
}

func TestDefaultExitConfig(t *testing.T) {
	cfg := DefaultExitConfig()
	if cfg.BaseMultiplier != 2.0 {
		t.Error("default base_multiplier should be 2.0")
	}
	if cfg.AdjustmentFactor != 0.5 {
		t.Error("default adjustment_factor should be 0.5")
	}
	if cfg.MinMultiplier > cfg.MaxMultiplier {
		t.Error("min_multiplier should be <= max_multiplier")
	}
}
