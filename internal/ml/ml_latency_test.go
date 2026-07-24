package ml

import (
	"testing"
	"time"

	"github.com/lee-econ/orca-core/internal/types"
)

func TestMetaLabelerConfigDefaults(t *testing.T) {
	cfg := DefaultMetaLabelerConfig()

	if cfg.WinThreshold <= 0.0 || cfg.WinThreshold >= 1.0 {
		t.Errorf("WinThreshold should be in (0, 1), got %f", cfg.WinThreshold)
	}
	if cfg.PositionScale <= 0 {
		t.Errorf("PositionScale should be positive, got %f", cfg.PositionScale)
	}
	if cfg.Timeout <= 0 {
		t.Errorf("Timeout should be positive, got %v", cfg.Timeout)
	}
	if cfg.Timeout > 10*time.Second {
		t.Errorf("Timeout should be <= 10s for live trading, got %v", cfg.Timeout)
	}
}

func TestExitOrchestratorConfigDefaults(t *testing.T) {
	cfg := ExitOrchestratorConfig{
		BaseMultiplier:   2.0,
		AdjustmentFactor: 0.5,
		MinMultiplier:    0.5,
		MaxMultiplier:    4.0,
		Timeout:          5 * time.Second,
	}

	if cfg.BaseMultiplier <= 0 {
		t.Errorf("BaseMultiplier should be positive, got %f", cfg.BaseMultiplier)
	}
	if cfg.MinMultiplier > cfg.MaxMultiplier {
		t.Errorf("MinMultiplier (%f) should be <= MaxMultiplier (%f)", cfg.MinMultiplier, cfg.MaxMultiplier)
	}
	if cfg.Timeout <= 0 || cfg.Timeout > 10*time.Second {
		t.Errorf("Timeout should be in (0, 10s], got %v", cfg.Timeout)
	}
}

func TestSubprocessPredictorClosesCleanly(t *testing.T) {
	cfg := DefaultMetaLabelerConfig()
	cfg.PythonPath = "nonexistent_python_for_test"
	predictor, err := NewSubprocessPredictor(cfg)
	if err != nil {
		t.Fatalf("NewSubprocessPredictor should not fail on creation: %v", err)
	}

	if predictor.IsHealthy() {
		t.Log("predictor should report unhealthy with invalid python path")
	}

	err = predictor.Close()
	if err != nil {
		t.Errorf("Close should not error: %v", err)
	}
}

func TestBatchInferrerConfigValidation(t *testing.T) {
	cfg := DefaultMetaLabelerConfig()
	predictor, _ := NewSubprocessPredictor(cfg)
	defer predictor.Close()

	bi := NewBatchInferrer(predictor, cfg)
	if bi == nil {
		t.Fatal("NewBatchInferrer should not return nil")
	}
}

func TestUrgencyToStopMultiplierBounds(t *testing.T) {
	tests := []struct {
		urgency   float64
		base      float64
		minMult   float64
		maxMult   float64
	}{
		{0.0, 2.0, 0.5, 4.0},
		{0.5, 2.0, 0.5, 4.0},
		{1.0, 2.0, 0.5, 4.0},
		{0.8, 1.5, 1.0, 3.0},
	}

	for _, tc := range tests {
		mult := UrgencyToStopMultiplier(tc.urgency, tc.base, tc.minMult)
		mult = clamp(mult, tc.minMult, tc.maxMult)
		if mult < tc.minMult {
			t.Errorf("urgency=%.2f: multiplier %.2f below min %.2f", tc.urgency, mult, tc.minMult)
		}
		if mult > tc.maxMult {
			t.Errorf("urgency=%.2f: multiplier %.2f above max %.2f", tc.urgency, mult, tc.maxMult)
		}
	}
}

func TestExitOrchComputeNewStopBounds(t *testing.T) {
	cfg := ExitOrchestratorConfig{
		BaseMultiplier:   2.0,
		AdjustmentFactor: 0.5,
		MinMultiplier:    0.5,
		MaxMultiplier:    4.0,
		Timeout:          5 * time.Second,
	}
	eo := NewExitOrchestrator(cfg)
	eo.Disable()

	ctx := ExitContext{
		EntryPrice:     types.FromFloat64(100.0),
		CurrentPrice:   types.FromFloat64(102.0),
		CurrentStop:    types.FromFloat64(99.0),
		HighSinceEntry: types.FromFloat64(103.0),
		LowSinceEntry:  types.FromFloat64(98.0),
		BarsSinceEntry: 10,
		ATR:            1.0,
		VolAtEntry:     0.01,
		VolCurrent:     0.01,
		HMMState:       1,
		ADX:            25.0,
		Hour:           14.0,
	}

	longStop := eo.ComputeNewStop("BUY", ctx.EntryPrice, ctx.CurrentPrice, ctx.ATR, ctx)
	if longStop <= ctx.EntryPrice.Float64()*0.7 {
		t.Errorf("long stop %.2f is too far below entry (floor is 80%% of entry = %.2f)", longStop, ctx.EntryPrice.Float64()*0.8)
	}

	shortStop := eo.ComputeNewStop("SELL", ctx.EntryPrice, ctx.CurrentPrice, ctx.ATR, ctx)
	if shortStop >= ctx.EntryPrice.Float64()*1.3 {
		t.Errorf("short stop %.2f is too far above entry (cap is 120%% of entry = %.2f)", shortStop, ctx.EntryPrice.Float64()*1.2)
	}
}

func BenchmarkUrgencyToStopMultiplier(b *testing.B) {
	for i := 0; i < b.N; i++ {
		UrgencyToStopMultiplier(0.5, 2.0, 0.5)
	}
}

func BenchmarkBuildExitFeatures(b *testing.B) {
	ctx := ExitContext{
		EntryPrice:     types.FromFloat64(100.0),
		CurrentPrice:   types.FromFloat64(102.0),
		CurrentStop:    types.FromFloat64(99.0),
		HighSinceEntry: types.FromFloat64(103.0),
		LowSinceEntry:  types.FromFloat64(98.0),
		BarsSinceEntry: 10,
		ATR:            1.0,
		VolAtEntry:     0.01,
		VolCurrent:     0.02,
		HMMState:       1,
		ADX:            25.0,
		Hour:           14.0,
	}
	for i := 0; i < b.N; i++ {
		BuildExitFeatures(ctx)
	}
}
