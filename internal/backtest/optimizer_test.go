package backtest

import (
	"fmt"
	"sort"
	"testing"

	"github.com/lee-econ/orca-core/internal/strategy"
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

func TestSearchSpace_GenerateAllCombinations_PruningMatchesNaive(t *testing.T) {
	space := SearchSpace{
		"stop_loss_atr_mult": {Name: "stop_loss_atr_mult", Type: ParamContinuous, Min: 0.5, Max: 1.5, Step: 0.5,
			Condition: &ConditionalRule{LeftParam: "stop_loss_atr_mult", Operator: "lt", RightParam: "take_profit_atr_mult"}},
		"take_profit_atr_mult": {Name: "take_profit_atr_mult", Type: ParamContinuous, Min: 1.0, Max: 2.0, Step: 0.5},
		"range_minutes":        {Name: "range_minutes", Type: ParamInteger, Min: 1, Max: 3, Step: 1},
		"volume_multiplier":    {Name: "volume_multiplier", Type: ParamContinuous, Min: 1.0, Max: 2.0, Step: 1.0},
	}

	pruned := space.GenerateAllCombinations()

	// Naive reference: full Cartesian product then filter.
	var naive []map[string]float64
	for _, raw := range space.generateAllRaw() {
		if space.checkCondition(raw) {
			naive = append(naive, raw)
		}
	}

	if len(pruned) != len(naive) {
		t.Fatalf("pruned count %d != naive count %d", len(pruned), len(naive))
	}

	naiveSet := make(map[string]bool, len(naive))
	for _, c := range naive {
		naiveSet[comboKey(c)] = true
	}
	for _, c := range pruned {
		if !naiveSet[comboKey(c)] {
			t.Fatalf("pruned combo %v missing from naive set", c)
		}
	}

	// The stop < take invariant must hold in every emitted combo.
	for _, c := range pruned {
		if c["stop_loss_atr_mult"] >= c["take_profit_atr_mult"] {
			t.Fatalf("invariant violated: %v", c)
		}
	}
}

func comboKey(c map[string]float64) string {
	names := make([]string, 0, len(c))
	for n := range c {
		names = append(names, n)
	}
	sort.Strings(names)
	s := ""
	for _, n := range names {
		s += fmt.Sprintf("%s=%.6f;", n, c[n])
	}
	return s
}

func TestComputeObjective_Sharpe(t *testing.T) {
	r := &BacktestResult{SharpeRatio: 1.5, NumTrades: 30}
	s := ComputeObjective(r, ObjectiveSharpe, nil)
	if s != 1.5 {
		t.Errorf("Expected 1.5, got %f", s)
	}
}

func TestComputeObjective_MinDD(t *testing.T) {
	r := &BacktestResult{MaxDrawdown: 8.5, NumTrades: 30}
	s := ComputeObjective(r, ObjectiveMinDD, nil)
	if s != -8.5 {
		t.Errorf("Expected -8.5, got %f", s)
	}
}

func TestComputeObjective_DDRatio(t *testing.T) {
	r := &BacktestResult{SharpeRatio: 1.8, MaxDrawdown: 6.0, NumTrades: 30}
	s := ComputeObjective(r, ObjectiveDDRatio, nil)
	expected := 1.8 / 6.0 * 100
	if s != expected {
		t.Errorf("Expected %f, got %f", expected, s)
	}
}

func TestComputeObjective_Composite(t *testing.T) {
	r := &BacktestResult{SharpeRatio: 2.0, MaxDrawdown: 5.0, ProfitFactor: 1.8, WinRate: 60, NumTrades: 30}
	weights := map[ObjectiveType]float64{
		ObjectiveSharpe:       0.4,
		ObjectiveProfitFactor: 0.3,
		ObjectiveMinDD:        0.3,
	}
	s := ComputeObjective(r, ObjectiveComposite, weights)
	if s <= 0 {
		t.Errorf("Expected positive composite score, got %f", s)
	}
}

func TestComputeObjective_LowTradesPenalty(t *testing.T) {
	r := &BacktestResult{SharpeRatio: 2.0, NumTrades: 10}
	s := ComputeObjective(r, ObjectiveSharpe, nil)
	if s >= 0 {
		t.Errorf("Expected negative penalty for <30 trades, got %f", s)
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

func TestDefaultSearchSpace_IncludesRegimeParticipation(t *testing.T) {
	// Regime participation (regime_w_calm/trending/highvol/crisis) must be an
	// optimizable, per-strategy parameter in every search space — this is the
	// lever that unblocks single-regime strategies (backtest_diagnosis_report.md §5).
	for _, sid := range []string{"vwap_mr", "intraday_mr", "trend_following", "session_scalp"} {
		space := DefaultSearchSpace(sid)
		if space == nil {
			t.Fatalf("expected non-nil search space for %s", sid)
		}
		for _, name := range []string{
			"regime_w_calm", "regime_w_trending", "regime_w_highvol", "regime_w_crisis",
		} {
			c, ok := space[name]
			if !ok {
				t.Errorf("%s: missing optimizable regime param %s", sid, name)
				continue
			}
			if c.Min != 0.0 || c.Max != 1.0 {
				t.Errorf("%s %s: bounds %v..%v, want 0..1", sid, name, c.Min, c.Max)
			}
		}
	}
}

func TestRegimeParamDefs_DefaultMatchesAllowed(t *testing.T) {
	// vwap_mr is Calm-only in the activation matrix → default participation
	// is {1,0,0,0}; the optimizer starts from the binary Allowed pattern and
	// may soften it (0..1, step 0.25).
	byName := map[string]strategy.ParamDef{}
	for _, d := range RegimeParamDefs("vwap_mr") {
		byName[d.Name] = d
	}
	if byName["regime_w_calm"].Default != 1.0 {
		t.Errorf("calm default = %v, want 1.0", byName["regime_w_calm"].Default)
	}
	if byName["regime_w_trending"].Default != 0.0 {
		t.Errorf("trending default = %v, want 0.0 (blocked by Allowed)", byName["regime_w_trending"].Default)
	}
	// The exit-multiplier regime params are also present.
	for _, name := range []string{"stop_mult_highvol", "stop_mult_crisis", "profit_mult_trending"} {
		if _, ok := byName[name]; !ok {
			t.Errorf("missing regime exit multiplier %s", name)
		}
	}
}

func TestOptimizationConfig_Defaults(t *testing.T) {
	cfg := OptimizationConfig{
		StrategyID:      "trend_following",
		ObjectiveType:   ObjectiveSharpe,
		MaxCombinations: 100,
	}
	if cfg.StrategyID != "trend_following" {
		t.Error("Strategy ID mismatch")
	}
	if cfg.ObjectiveType != ObjectiveSharpe {
		t.Error("Objective type mismatch")
	}
}
