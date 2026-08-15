package backtest

import (
	"testing"
	"time"
)

func TestReevaluator_Demotion_MaxDDBreach(t *testing.T) {
	cfg := DefaultReevaluationConfig()
	cfg.MaxDDThreshold["grid_trading"] = 15.0
	sr := NewStrategyReevaluator(cfg, nil)
	now := time.Now()
	states := map[string]StrategyState{"grid_trading": StrategyActive}
	weights := map[string]float64{"grid_trading": 1.0}
	sharpe := map[string]float64{"grid_trading": 2.0}
	dd := map[string]float64{"grid_trading": 20.0}

	results := sr.Evaluate(sharpe, dd, states, weights, now)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Action != "hard_halt" {
		t.Errorf("expected hard_halt, got %s", results[0].Action)
	}
	if results[0].NewState != StrategyViolated {
		t.Errorf("expected Violated, got %s", results[0].NewState)
	}
}

func TestReevaluator_Demotion_SharpeDegradation(t *testing.T) {
	cfg := DefaultReevaluationConfig()
	cfg.SharpeDegradationPct = 0.30
	cfg.DegradationDays = 10
	sr := NewStrategyReevaluator(cfg, map[string]float64{"grid_trading": 2.0})
	now := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	states := map[string]StrategyState{"grid_trading": StrategyActive}
	weights := map[string]float64{"grid_trading": 1.0}

	for day := 1; day <= 12; day++ {
		sharpe := map[string]float64{"grid_trading": 0.3}
		dd := map[string]float64{"grid_trading": 5.0}
		sr.Evaluate(sharpe, dd, states, weights, now.AddDate(0, 0, day))
	}

	results := sr.Evaluate(
		map[string]float64{"grid_trading": 0.3},
		map[string]float64{"grid_trading": 5.0},
		states, weights, now.AddDate(0, 0, 13),
	)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Action != "reduce_allocation" {
		t.Errorf("expected reduce_allocation, got %s", results[0].Action)
	}
}

func TestReevaluator_NoAction_ViolatedUnchanged(t *testing.T) {
	cfg := DefaultReevaluationConfig()
	sr := NewStrategyReevaluator(cfg, nil)
	now := time.Now()
	states := map[string]StrategyState{"grid_trading": StrategyViolated}
	weights := map[string]float64{"grid_trading": 0}
	results := sr.Evaluate(
		map[string]float64{"grid_trading": -1.0},
		map[string]float64{"grid_trading": 30.0},
		states, weights, now,
	)
	if len(results) != 0 {
		t.Errorf("expected 0 results for already-violated strategy, got %d", len(results))
	}
}

func TestReevaluator_RecordFillSlippage_Average(t *testing.T) {
	cfg := DefaultReevaluationConfig()
	sr := NewStrategyReevaluator(cfg, nil)
	for i := 0; i < 10; i++ {
		sr.RecordFillSlippage("a", 2.0)
	}
	avg := sr.AverageFillSlippage("a")
	if avg != 2.0 {
		t.Errorf("expected avg 2.0, got %f", avg)
	}
	avgNone := sr.AverageFillSlippage("unknown")
	if avgNone != 0 {
		t.Errorf("expected 0 for unknown, got %f", avgNone)
	}
}

func TestReevaluator_MissingBenchmark(t *testing.T) {
	cfg := DefaultReevaluationConfig()
	sr := NewStrategyReevaluator(cfg, nil)
	now := time.Now()
	results := sr.Evaluate(
		map[string]float64{"grid_trading": -1.0},
		map[string]float64{"grid_trading": 5.0},
		map[string]StrategyState{"grid_trading": StrategyActive},
		map[string]float64{"grid_trading": 1.0},
		now,
	)
	if len(results) != 0 {
		t.Errorf("expected 0 results when no benchmark, got %d", len(results))
	}
}

func TestReevaluator_Promotion_RegimeReentry(t *testing.T) {
	cfg := DefaultReevaluationConfig()
	sr := NewStrategyReevaluator(cfg, nil)
	now := time.Now()
	results := sr.Evaluate(
		map[string]float64{"grid_trading": 1.5},
		map[string]float64{"grid_trading": 5.0},
		map[string]StrategyState{"grid_trading": StrategyStandby},
		map[string]float64{"grid_trading": 0},
		now,
	)
	if len(results) != 1 {
		t.Fatalf("expected 1 result for standby, got %d", len(results))
	}
	if results[0].Action != "activate" {
		t.Errorf("expected activate, got %s", results[0].Action)
	}
}

func TestReevaluator_BenchmarkFailBlocksPromotion(t *testing.T) {
	cfg := DefaultReevaluationConfig()
	sr := NewStrategyReevaluator(cfg, nil)
	sr.SetBenchmarkPassed("grid_trading", false)
	now := time.Now()
	results := sr.Evaluate(
		map[string]float64{"grid_trading": 1.5},
		map[string]float64{"grid_trading": 5.0},
		map[string]StrategyState{"grid_trading": StrategyStandby},
		map[string]float64{"grid_trading": 0},
		now,
	)
	if len(results) != 0 {
		t.Fatalf("benchmark-failed strategy must not be promoted, got %d results", len(results))
	}
}

func TestReevaluator_BenchmarkPassAllowsPromotion(t *testing.T) {
	cfg := DefaultReevaluationConfig()
	sr := NewStrategyReevaluator(cfg, nil)
	sr.SetBenchmarkPassed("grid_trading", true)
	now := time.Now()
	results := sr.Evaluate(
		map[string]float64{"grid_trading": 1.5},
		map[string]float64{"grid_trading": 5.0},
		map[string]StrategyState{"grid_trading": StrategyStandby},
		map[string]float64{"grid_trading": 0},
		now,
	)
	if len(results) != 1 {
		t.Fatalf("benchmark-passed strategy should be promoted, got %d results", len(results))
	}
	if results[0].Action != "activate" {
		t.Errorf("expected activate, got %s", results[0].Action)
	}
}
