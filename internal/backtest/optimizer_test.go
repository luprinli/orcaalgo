package backtest

import (
	"testing"
)

func TestParamConstraint_ContinuousGrid(t *testing.T) {
	c := ParamConstraint{Name: "atr_multiplier", Type: ParamContinuous, Min: 1.0, Max: 3.0, Step: 0.5}
	grid := c.Grid()
	if len(grid) != 5 {
		t.Errorf("Expected 5 values (1.0, 1.5, 2.0, 2.5, 3.0), got %d", len(grid))
	}
	if grid[0] != 1.0 || grid[4] != 3.0 {
		t.Errorf("Expected [1.0, ..., 3.0], got %v", grid)
	}
}

func TestParamConstraint_IntegerGrid(t *testing.T) {
	c := ParamConstraint{Name: "fast_period", Type: ParamInteger, Min: 5, Max: 15, Step: 5}
	grid := c.Grid()
	if len(grid) != 3 {
		t.Errorf("Expected 3 values (5, 10, 15), got %d", len(grid))
	}
}

func TestSearchSpace_TotalCombinations(t *testing.T) {
	space := SearchSpace{
		"fast_period":    {Type: ParamInteger, Min: 5, Max: 15, Step: 5},
		"atr_multiplier": {Type: ParamContinuous, Min: 1.0, Max: 2.0, Step: 0.5},
	}
	total := space.TotalCombinations()
	expected := 3 * 3
	if total != expected {
		t.Errorf("Expected %d combinations, got %d", expected, total)
	}
}

func TestSearchSpace_GenerateAllCombinations(t *testing.T) {
	space := SearchSpace{
		"range_minutes":    {Name: "range_minutes", Type: ParamInteger, Min: 1, Max: 3, Step: 1},
		"entry_buffer_pct": {Name: "entry_buffer_pct", Type: ParamContinuous, Min: 0.1, Max: 0.3, Step: 0.1},
	}

	combos := space.GenerateAllCombinations()
	expected := 3 * 3
	if len(combos) != expected {
		t.Errorf("Expected %d combinations, got %d", expected, len(combos))
	}

	for i, c := range combos {
		if c["range_minutes"] < 1 || c["range_minutes"] > 3 {
			t.Errorf("Combo %d: range_minutes=%f out of bounds", i, c["range_minutes"])
		}
	}
}

func TestSearchSpace_EmptySearchSpace(t *testing.T) {
	space := SearchSpace{}
	combos := space.GenerateAllCombinations()
	if len(combos) != 1 {
		t.Errorf("Expected 1 empty combo, got %d", len(combos))
	}
}

func TestComputeObjective_Sharpe(t *testing.T) {
	r := &BacktestResult{SharpeRatio: 1.5}
	s := ComputeObjective(r, ObjectiveSharpe, nil)
	if s != 1.5 {
		t.Errorf("Expected 1.5, got %f", s)
	}
}

func TestComputeObjective_MinDD(t *testing.T) {
	r := &BacktestResult{MaxDrawdown: 8.5}
	s := ComputeObjective(r, ObjectiveMinDD, nil)
	if s != -8.5 {
		t.Errorf("Expected -8.5, got %f", s)
	}
}

func TestComputeObjective_DDRatio(t *testing.T) {
	r := &BacktestResult{SharpeRatio: 1.8, MaxDrawdown: 6.0}
	s := ComputeObjective(r, ObjectiveDDRatio, nil)
	expected := 1.8 / 6.0 * 100
	if s != expected {
		t.Errorf("Expected %f, got %f", expected, s)
	}
}

func TestComputeObjective_Composite(t *testing.T) {
	r := &BacktestResult{SharpeRatio: 2.0, MaxDrawdown: 5.0, ProfitFactor: 1.8, WinRate: 60}
	weights := map[ObjectiveType]float64{
		ObjectiveSharpe:      0.4,
		ObjectiveProfitFactor: 0.3,
		ObjectiveMinDD:       0.3,
	}
	s := ComputeObjective(r, ObjectiveComposite, weights)
	if s <= 0 {
		t.Errorf("Expected positive composite score, got %f", s)
	}
}

func TestDefaultSearchSpace_Trend(t *testing.T) {
	space := DefaultSearchSpace("trend_following")
	if space == nil {
		t.Fatal("Expected non-nil search space for trend")
	}
	if _, ok := space["fast_period"]; !ok {
		t.Error("Missing fast_period constraint")
	}
	if _, ok := space["atr_multiplier"]; !ok {
		t.Error("Missing atr_multiplier constraint")
	}
}

func TestDefaultSearchSpace_ORB(t *testing.T) {
	space := DefaultSearchSpace("opening_range_breakout")
	if space == nil {
		t.Fatal("Expected non-nil search space for ORB")
	}
}

func TestDefaultSearchSpace_Unknown(t *testing.T) {
	space := DefaultSearchSpace("nonexistent")
	if space != nil {
		t.Error("Expected nil search space for unknown strategy")
	}
}

func TestOptimizationConfig_Defaults(t *testing.T) {
	cfg := OptimizationConfig{
		StrategyID:    "trend_following",
		ObjectiveType: ObjectiveSharpe,
		MaxCombinations: 100,
	}
	if cfg.StrategyID != "trend_following" {
		t.Error("Strategy ID mismatch")
	}
	if cfg.ObjectiveType != ObjectiveSharpe {
		t.Error("Objective type mismatch")
	}
}
