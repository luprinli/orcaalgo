package backtest

import "testing"

func mkCombo(strat, sym, tf string, sharpe, dd, pf float64, trades int, errStr string) ComboResult {
	return ComboResult{
		StrategyID: strat, Symbol: sym, Timeframe: tf,
		SharpeRatio: sharpe, MaxDrawdown: dd, ProfitFactor: pf,
		NumTrades: trades, Error: errStr,
	}
}

func TestClassifyResults_TiersAndViability(t *testing.T) {
	cfg := DefaultRetentionConfig()
	cfg.TopK = 2
	cfg.PlateauBand = 0.5
	cfg.T2SampleCap = 100

	results := []ComboResult{
		mkCombo("s", "A", "1d", 2.0, 5, 2.0, 100, ""),             // island (best)
		mkCombo("s", "B", "1d", 1.9, 6, 1.9, 90, ""),              // island (top-K) / pareto
		mkCombo("s", "C", "1d", 1.6, 7, 1.5, 80, ""),              // plateau (>= 1.9-0.5=1.4) -> T1
		mkCombo("s", "D", "1d", 0.2, 20, 1.0, 40, ""),             // viable, below plateau -> T2
		mkCombo("s", "E", "1d", -1.0, 30, 0.5, 10, ""),            // viable loss -> T2
		mkCombo("s", "F", "1d", 0.0, 0, 0.0, 0, ""),               // zero trades -> T3
		mkCombo("s", "G", "1d", 0.0, 0, 0.0, 0, "no candle data"), // error -> T3
	}

	class := ClassifyResults(results, cfg)

	if class["s|A|1d"] != RetentionT0 {
		t.Errorf("A should be T0, got %d", class["s|A|1d"])
	}
	if class["s|B|1d"] != RetentionT0 {
		t.Errorf("B should be T0, got %d", class["s|B|1d"])
	}
	if class["s|C|1d"] != RetentionT1 {
		t.Errorf("C should be T1 (plateau), got %d", class["s|C|1d"])
	}
	if class["s|D|1d"] != RetentionT2 {
		t.Errorf("D should be T2, got %d", class["s|D|1d"])
	}
	if class["s|F|1d"] != RetentionT3 {
		t.Errorf("F (zero trades) should be T3, got %d", class["s|F|1d"])
	}
	if class["s|G|1d"] != RetentionT3 {
		t.Errorf("G (error) should be T3, got %d", class["s|G|1d"])
	}
}

func TestClassifyResults_T2CapDemotesToT3(t *testing.T) {
	cfg := DefaultRetentionConfig()
	cfg.TopK = 1
	cfg.PlateauBand = 0.0
	cfg.T2SampleCap = 3

	var results []ComboResult
	// 1 clear winner + 20 mediocre viable combos below the plateau.
	results = append(results, mkCombo("s", "WIN", "1d", 3.0, 2, 3.0, 200, ""))
	for i := 0; i < 20; i++ {
		results = append(results, mkCombo("s", string(rune('a'+i)), "1d", -0.5, 25, 0.8, 30, ""))
	}

	class := ClassifyResults(results, cfg)
	t2, t3 := 0, 0
	for _, c := range class {
		switch c {
		case RetentionT2:
			t2++
		case RetentionT3:
			t3++
		}
	}
	if t2 > cfg.T2SampleCap {
		t.Errorf("T2 count %d exceeds cap %d", t2, cfg.T2SampleCap)
	}
	if t3 == 0 {
		t.Errorf("expected some combos demoted to T3 when over the T2 cap")
	}
}

func TestBuildRunSummary(t *testing.T) {
	cfg := DefaultRetentionConfig()
	results := []ComboResult{
		mkCombo("s", "A", "1d", 2.0, 5, 2.0, 100, ""),
		mkCombo("s", "B", "1d", 1.0, 10, 1.5, 50, ""),
		mkCombo("s", "C", "1d", 0.0, 0, 0.0, 0, ""),               // zero trade
		mkCombo("s", "D", "1d", 0.0, 0, 0.0, 0, "no candle data"), // error
	}
	sum := BuildRunSummary(results, cfg)

	if sum.TotalCombos != 4 {
		t.Errorf("total should be 4, got %d", sum.TotalCombos)
	}
	if sum.TradedCombos != 2 {
		t.Errorf("traded should be 2, got %d", sum.TradedCombos)
	}
	if sum.ZeroTrade != 1 || sum.Errored != 1 {
		t.Errorf("zero=%d errored=%d, want 1/1", sum.ZeroTrade, sum.Errored)
	}
	if sum.BestSharpe != 2.0 || sum.BestCombo != "s|A|1d" {
		t.Errorf("best sharpe/combo wrong: %f %s", sum.BestSharpe, sum.BestCombo)
	}
	if sum.FailureReasons["no_data"] != 1 {
		t.Errorf("expected no_data failure reason, got %v", sum.FailureReasons)
	}
	if len(sum.ParetoFront) == 0 {
		t.Errorf("expected non-empty pareto front")
	}
}
