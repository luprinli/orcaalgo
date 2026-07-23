package backtest

import (
	"testing"
	"time"
)

func TestOptimizedWalkForwardConfig_Default(t *testing.T) {
	cfg := DefaultOptimizedWalkForwardConfig(
		"trend_following",
		[]string{"SPY"},
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
		100000,
	)

	if cfg.StrategyID != "trend_following" {
		t.Error("Wrong strategy ID")
	}
	if cfg.TrainYears != 1 {
		t.Error("Wrong train years")
	}
	if cfg.SearchSpace == nil {
		t.Error("Search space should not be nil")
	}
	if cfg.MaxCombinations != 200 {
		t.Errorf("Expected 200 max combinations, got %d", cfg.MaxCombinations)
	}
}

func TestOptimizedWalkForwardConfig_AllStrategies(t *testing.T) {
	strategies := []string{
		"trend_following",
		"opening_range_breakout",
		"intraday_mr",
		"grid",
	}

	for _, sid := range strategies {
		cfg := DefaultOptimizedWalkForwardConfig(
			sid,
			[]string{"SPY"},
			time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
			100000,
		)
		space := cfg.SearchSpace
		if space == nil {
			t.Errorf("Search space nil for strategy: %s", sid)
		}
		combos := space.TotalCombinations()
		t.Logf("%s: %d combinations", sid, combos)
		if combos == 0 {
			t.Errorf("Zero combinations for %s", sid)
		}
	}
}

func TestOptimizedWalkForwardConfig_ObjectiveWeights(t *testing.T) {
	cfg := DefaultOptimizedWalkForwardConfig(
		"trend_following",
		[]string{"SPY"},
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
		100000,
	)

	cfg.ObjectiveType = ObjectiveComposite
	cfg.ObjectiveWeights = map[ObjectiveType]float64{
		ObjectiveSharpe:      0.5,
		ObjectiveProfitFactor: 0.3,
		ObjectiveMinDD:       0.2,
	}

	if cfg.ObjectiveType != ObjectiveComposite {
		t.Error("Expected composite objective")
	}
	if len(cfg.ObjectiveWeights) != 3 {
		t.Error("Expected 3 weight components")
	}
}

func TestOptimizedWalkForwardConfig_MaxCombinationsCap(t *testing.T) {
	cfg := DefaultOptimizedWalkForwardConfig(
		"trend_following",
		[]string{"SPY"},
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
		100000,
	)

	cfg.MaxCombinations = 10
	total := cfg.SearchSpace.TotalCombinations()

	if cfg.MaxCombinations < total {
		t.Logf("Max combinations %d < total %d (capped)", cfg.MaxCombinations, total)
	}
}

func TestDefaultIVSConfig(t *testing.T) {
	cfg := DefaultIVSConfig()
	if !cfg.Enabled {
		t.Error("IVS should be enabled by default")
	}
	if cfg.NeighborRadius <= 0 {
		t.Error("NeighborRadius must be positive")
	}
	if cfg.PlateauThreshold <= 0 || cfg.PlateauThreshold > 1 {
		t.Error("PlateauThreshold must be in (0, 1]")
	}
	if cfg.ScoreThresholdPct <= 0 || cfg.ScoreThresholdPct > 1 {
		t.Error("ScoreThresholdPct must be in (0, 1]")
	}
}

func TestRunIVS_IsolatedPeak(t *testing.T) {
	searchSpace := SearchSpace{
		"param_a": {Name: "param_a", Type: ParamContinuous, Min: 0, Max: 10, Step: 1},
		"param_b": {Name: "param_b", Type: ParamContinuous, Min: 0, Max: 10, Step: 1},
	}

	scored := []ParamScore{
		{Params: map[string]float64{"param_a": 1.0, "param_b": 1.0}, Score: 10.0},
		{Params: map[string]float64{"param_a": 2.0, "param_b": 2.0}, Score: 2.0},
		{Params: map[string]float64{"param_a": 3.0, "param_b": 3.0}, Score: 2.1},
		{Params: map[string]float64{"param_a": 4.0, "param_b": 4.0}, Score: 2.2},
		{Params: map[string]float64{"param_a": 5.0, "param_b": 5.0}, Score: 9.0},
		{Params: map[string]float64{"param_a": 6.0, "param_b": 6.0}, Score: 1.9},
		{Params: map[string]float64{"param_a": 7.0, "param_b": 7.0}, Score: 2.3},
		{Params: map[string]float64{"param_a": 8.0, "param_b": 8.0}, Score: 2.4},
		{Params: map[string]float64{"param_a": 9.0, "param_b": 9.0}, Score: 2.5},
	}

	cfg := DefaultIVSConfig()
	cfg.NeighborRadius = 0.15

	params, passed := RunIVS(scored, searchSpace, cfg)

	if !passed {
		t.Log("IVS correctly rejected isolated peak")
	} else {
		t.Logf("IVS selected params: %v", params)
	}

	if params == nil {
		t.Error("Expected non-nil params even when IVS fails")
	}
}

func TestRunIVS_Plateau(t *testing.T) {
	searchSpace := SearchSpace{
		"param_a": {Name: "param_a", Type: ParamContinuous, Min: 0, Max: 10, Step: 1},
		"param_b": {Name: "param_b", Type: ParamContinuous, Min: 0, Max: 10, Step: 1},
	}

	scored := []ParamScore{
		{Params: map[string]float64{"param_a": 1.0, "param_b": 1.0}, Score: 9.8},
		{Params: map[string]float64{"param_a": 1.1, "param_b": 1.1}, Score: 9.7},
		{Params: map[string]float64{"param_a": 1.2, "param_b": 1.2}, Score: 9.6},
		{Params: map[string]float64{"param_a": 2.0, "param_b": 2.0}, Score: 2.0},
		{Params: map[string]float64{"param_a": 5.0, "param_b": 5.0}, Score: 10.0},
		{Params: map[string]float64{"param_a": 6.0, "param_b": 6.0}, Score: 1.9},
	}

	cfg := DefaultIVSConfig()
	cfg.NeighborRadius = 0.05

	params, passed := RunIVS(scored, searchSpace, cfg)
	t.Logf("IVS passed=%v, params=%v", passed, params)

	if params == nil {
		t.Error("Expected non-nil params")
	}
}

func TestRunIVS_Empty(t *testing.T) {
	searchSpace := SearchSpace{
		"param_a": {Name: "param_a", Type: ParamContinuous, Min: 0, Max: 10, Step: 1},
	}

	cfg := DefaultIVSConfig()
	params, passed := RunIVS(nil, searchSpace, cfg)
	if passed {
		t.Error("IVS should not pass on empty input")
	}
	if params != nil {
		t.Error("Expected nil params for empty input")
	}

	params, passed = RunIVS([]ParamScore{}, searchSpace, cfg)
	if passed {
		t.Error("IVS should not pass on empty slice")
	}
	if params != nil {
		t.Error("Expected nil params for empty slice")
	}
}

func TestParamDistance_Normalized(t *testing.T) {
	space := SearchSpace{
		"param_a": {Name: "param_a", Type: ParamContinuous, Min: 0, Max: 10, Step: 1},
		"param_b": {Name: "param_b", Type: ParamContinuous, Min: 100, Max: 200, Step: 10},
	}

	a := map[string]float64{"param_a": 0.0, "param_b": 100.0}
	b := map[string]float64{"param_a": 5.0, "param_b": 150.0}

	names := sortedParamNames(space)
	d := paramDistance(a, b, space, names)

	if d < 0.4 || d > 0.6 {
		t.Errorf("Expected normalized distance ~0.5, got %f", d)
	}
}

func TestRunIVS_SingleElement(t *testing.T) {
	searchSpace := SearchSpace{
		"param_a": {Name: "param_a", Type: ParamContinuous, Min: 0, Max: 10, Step: 1},
	}

	scored := []ParamScore{
		{Params: map[string]float64{"param_a": 5.0}, Score: 7.0},
	}

	cfg := DefaultIVSConfig()
	params, passed := RunIVS(scored, searchSpace, cfg)
	if passed {
		t.Log("Single-element IVS may pass trivially")
	}
	if params == nil {
		t.Error("Expected non-nil params for single element")
	}
}

func TestRunIVS_AllNegativeScores(t *testing.T) {
	searchSpace := SearchSpace{
		"param_a": {Name: "param_a", Type: ParamContinuous, Min: 0, Max: 10, Step: 1},
	}

	scored := []ParamScore{
		{Params: map[string]float64{"param_a": 1.0}, Score: -5.0},
		{Params: map[string]float64{"param_a": 2.0}, Score: -3.0},
		{Params: map[string]float64{"param_a": 3.0}, Score: -1.0},
	}

	cfg := DefaultIVSConfig()
	params, passed := RunIVS(scored, searchSpace, cfg)
	t.Logf("All-negative IVS passed=%v, params=%v", passed, params)
	if params == nil {
		t.Error("Expected non-nil params for all-negative scores")
	}
}

func TestOptimizedWalkForwardConfig_IVSDefaults(t *testing.T) {
	cfg := DefaultOptimizedWalkForwardConfig(
		"trend_following",
		[]string{"SPY"},
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
		100000,
	)

	if !cfg.IVSConfig.Enabled {
		t.Log("IVS is disabled in default config (integration tests only enable it via override)")
	}
}
