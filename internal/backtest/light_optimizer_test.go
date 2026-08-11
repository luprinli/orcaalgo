package backtest

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"
)

func TestValidSearchSpace(t *testing.T) {
	if validSearchSpace(nil) {
		t.Fatal("nil space must be invalid")
	}
	// A single optimisable dimension is not enough.
	one := SearchSpace{"a": {Name: "a", Type: ParamInteger, Min: 1, Max: 5, Step: 1}}
	if validSearchSpace(one) {
		t.Fatal("single-dimension space must be invalid")
	}
	// Two valid dimensions is enough.
	two := SearchSpace{
		"a": {Name: "a", Type: ParamInteger, Min: 1, Max: 5, Step: 1},
		"b": {Name: "b", Type: ParamContinuous, Min: 0.1, Max: 0.9, Step: 0.1},
	}
	if !validSearchSpace(two) {
		t.Fatal("two-dimension space must be valid")
	}
	// A real strategy's default search space must be valid.
	if !validSearchSpace(DefaultSearchSpace("intraday_mr")) {
		t.Fatal("intraday_mr default search space must be valid")
	}
	// Unknown strategy yields a nil space -> invalid.
	if validSearchSpace(DefaultSearchSpace("does_not_exist_xyz")) {
		t.Fatal("unknown strategy space must be invalid")
	}
}

func TestSplitWindow(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)
	ts, te, vs, ve := splitWindow(start, end, 0.67)
	if !ts.Equal(start) {
		t.Errorf("train start = %v, want %v", ts, start)
	}
	if !ve.Equal(end) {
		t.Errorf("test end = %v, want %v", ve, end)
	}
	if !te.After(start) || !te.Before(end) {
		t.Errorf("train end %v must be strictly inside the window", te)
	}
	if !vs.Equal(te) {
		t.Errorf("test start %v must equal train end %v", vs, te)
	}

	// Degenerate window: everything collapses to the full window.
	ts2, te2, vs2, ve2 := splitWindow(start, start, 0.67)
	if !ts2.Equal(start) || !te2.Equal(start) || !vs2.Equal(start) || !ve2.Equal(start) {
		t.Error("degenerate window must collapse to the start instant")
	}
}

func TestGenerateCandidatesEnumeratesSmallSpace(t *testing.T) {
	sp := SearchSpace{
		"a": {Name: "a", Type: ParamInteger, Min: 1, Max: 3, Step: 1}, // 3 values
		"b": {Name: "b", Type: ParamInteger, Min: 1, Max: 2, Step: 1}, // 2 values
	}
	total := sp.TotalCombinations() // 6
	got := generateCandidates(sp, 24, 1)
	if len(got) != total {
		t.Errorf("small space should be fully enumerated: got %d, want %d", len(got), total)
	}
}

func TestGenerateCandidatesRandomDeterministic(t *testing.T) {
	sp := DefaultSearchSpace("intraday_mr")
	if sp.TotalCombinations() <= 8 {
		t.Skip("search space too small to exercise random sampling")
	}
	a := generateCandidates(sp, 8, 1)
	b := generateCandidates(sp, 8, 1)
	if len(a) == 0 || len(a) != len(b) {
		t.Fatalf("expected equal non-empty candidate sets, got %d and %d", len(a), len(b))
	}
	key := func(m map[string]float64) string {
		names := make([]string, 0, len(m))
		for n := range m {
			names = append(names, n)
		}
		sort.Strings(names)
		s := ""
		for _, n := range names {
			s += fmt.Sprintf("%s=%.4f;", n, m[n])
		}
		return s
	}
	for i := range a {
		if key(a[i]) != key(b[i]) {
			t.Fatalf("same seed must produce identical candidates at index %d", i)
		}
	}
	if len(a) > 8 {
		t.Fatalf("random sampling must respect the budget: got %d > 8", len(a))
	}
}

func TestLightOptCache(t *testing.T) {
	ResetLightOptCache()
	cfg := LightOptimizeConfig{
		StrategyID: "intraday_mr",
		Symbols:    []string{"SPY", "QQQ"},
		StartDate:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:    time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
		Timeframe:  "1d",
	}
	key := lightOptCacheKey(cfg)

	if _, ok := lightOptCacheGet(key); ok {
		t.Fatal("cache should start empty")
	}
	want := map[string]float64{"entry_z": 2.0, "sizing_percent": 0.02}
	lightOptCachePut(key, want, time.Hour)
	got, ok := lightOptCacheGet(key)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got["entry_z"] != 2.0 || got["sizing_percent"] != 0.02 {
		t.Errorf("cache returned wrong params: %v", got)
	}
	// Symbol-order independence: reordered symbols map to the same key.
	cfg2 := cfg
	cfg2.Symbols = []string{"QQQ", "SPY"}
	if lightOptCacheKey(cfg2) != key {
		t.Error("cache key must be independent of symbol order")
	}
	// Different window -> different key.
	cfg3 := cfg
	cfg3.EndDate = cfg.EndDate.AddDate(0, 1, 0)
	if lightOptCacheKey(cfg3) == key {
		t.Error("different window must produce a different key")
	}

	// TTL expiry.
	lightOptCachePut(key, want, time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	if _, ok := lightOptCacheGet(key); ok {
		t.Error("expired entry must not be returned")
	}
}

func TestRunLightOptimizeUnknownStrategyReturnsNil(t *testing.T) {
	ResetLightOptCache()
	// Unknown strategy -> nil search space -> clean no-op (nil), without touching the DB.
	got := RunLightOptimize(context.Background(), &mockDB{}, LightOptimizeConfig{
		StrategyID: "totally_unknown_strategy",
		Symbols:    []string{"SPY"},
		StartDate:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:    time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
		Timeframe:  "1d",
	})
	if got != nil {
		t.Errorf("unknown strategy must return nil, got %v", got)
	}
}

func TestRunLightOptimizeCancelledContextReturnsNil(t *testing.T) {
	ResetLightOptCache()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled before any work
	got := RunLightOptimize(ctx, &mockDB{}, LightOptimizeConfig{
		StrategyID: "intraday_mr",
		Symbols:    []string{"SPY"},
		StartDate:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:    time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
		Timeframe:  "1d",
	})
	if got != nil {
		t.Errorf("cancelled context must produce no best params, got %v", got)
	}
}

func TestApplyLightOptDefaults(t *testing.T) {
	cfg := LightOptimizeConfig{}
	applyLightOptDefaults(&cfg)
	if cfg.MaxCombos != 24 {
		t.Errorf("default MaxCombos = %d, want 24", cfg.MaxCombos)
	}
	if cfg.PerBacktestTimeout != 10*time.Second {
		t.Errorf("default timeout = %v, want 10s", cfg.PerBacktestTimeout)
	}
	if cfg.PlateauPatience != 5 {
		t.Errorf("default plateau patience = %d, want 5", cfg.PlateauPatience)
	}
	if cfg.TrainFraction != 0.80 {
		t.Errorf("default train fraction = %v, want 0.80", cfg.TrainFraction)
	}
	if cfg.ObjectiveWeights != [3]float64{0.5, 0.3, 0.2} {
		t.Errorf("default weights = %v, want [0.5 0.3 0.2]", cfg.ObjectiveWeights)
	}
	if cfg.InitialCapital <= 0 {
		t.Error("default initial capital must be positive")
	}
	if cfg.RandomSeed != 1 {
		t.Errorf("default seed = %d, want 1", cfg.RandomSeed)
	}
}
