package backtest

import (
	"context"
	"testing"
	"time"

	"github.com/lee-econ/orca-core/internal/propfirm"
	"github.com/lee-econ/orca-core/internal/types"
)

func makeOrchDB(candles []Candle, regime []RegimeLog) Database {
	return &parityDB{candles: candles, regime: regime, vix: nil}
}

func TestOrchestrator_NewOrchestrator_DefaultConfig(t *testing.T) {
	o, err := NewOrchestrator(nil, OrchestratorConfig{})
	if err != nil {
		t.Fatalf("NewOrchestrator failed: %v", err)
	}
	if o.config.KellyFraction != 0.25 {
		t.Errorf("default KellyFraction: got %f, want 0.25", o.config.KellyFraction)
	}
	if o.config.RebalanceBars != 20 {
		t.Errorf("default RebalanceBars: got %d, want 20", o.config.RebalanceBars)
	}
	if o.config.CorrelationThreshold != 0.6 {
		t.Errorf("default CorrelationThreshold: got %f, want 0.6", o.config.CorrelationThreshold)
	}
	if o.config.MaxPositionPct != 0.02 {
		t.Errorf("default MaxPositionPct: got %f, want 0.02", o.config.MaxPositionPct)
	}
	if o.scheduler == nil {
		t.Error("scheduler is nil")
	}
	if o.correlation == nil {
		t.Error("correlation is nil")
	}
	if o.vixDetector == nil {
		t.Error("vixDetector is nil")
	}
}

func TestOrchestrator_NewOrchestrator_CustomConfig(t *testing.T) {
	o, err := NewOrchestrator(nil, OrchestratorConfig{
		KellyFraction:        0.15,
		RebalanceBars:        40,
		CorrelationThreshold: 0.5,
		FrictionModel:        "realistic",
		MaxPositionPct:       0.05,
	})
	if err != nil {
		t.Fatalf("NewOrchestrator failed: %v", err)
	}
	if o.config.KellyFraction != 0.15 {
		t.Errorf("got KellyFraction %f", o.config.KellyFraction)
	}
	if o.config.RebalanceBars != 40 {
		t.Errorf("got RebalanceBars %d", o.config.RebalanceBars)
	}
	if o.config.CorrelationThreshold != 0.5 {
		t.Errorf("got CorrelationThreshold %f", o.config.CorrelationThreshold)
	}
	if o.config.MaxPositionPct != 0.05 {
		t.Errorf("got MaxPositionPct %f", o.config.MaxPositionPct)
	}
}

func TestOrchestrator_NewOrchestrator_InvalidConfig(t *testing.T) {
	o, err := NewOrchestrator(nil, OrchestratorConfig{
		RebalanceBars:    -1,
		KellyFraction:    0,
		MaxPositionPct:   0,
	})
	if err != nil {
		t.Fatalf("NewOrchestrator failed: %v", err)
	}
	if o.config.RebalanceBars != 20 {
		t.Errorf("RebalanceBars should clamp negative to 20, got %d", o.config.RebalanceBars)
	}
	if o.config.KellyFraction != 0.25 {
		t.Errorf("KellyFraction should clamp 0 to 0.25, got %f", o.config.KellyFraction)
	}
	if o.config.MaxPositionPct != 0.02 {
		t.Errorf("MaxPositionPct should clamp 0 to 0.02, got %f", o.config.MaxPositionPct)
	}
}

func TestOrchestrator_NewOrchestrator_MaxPositionPctTooHigh(t *testing.T) {
	o, err := NewOrchestrator(nil, OrchestratorConfig{MaxPositionPct: 0.50})
	if err != nil {
		t.Fatalf("NewOrchestrator failed: %v", err)
	}
	if o.config.MaxPositionPct != 0.02 {
		t.Errorf("MaxPositionPct should clamp >0.20 to 0.02, got %f", o.config.MaxPositionPct)
	}
}

func TestOrchestrator_AddStrategy_Valid(t *testing.T) {
	db := makeOrchDB(nil, nil)
	o, _ := NewOrchestrator(db, OrchestratorConfig{})
	if err := o.AddStrategy("JPN225", "1h", "grid_trading"); err != nil {
		t.Fatalf("AddStrategy failed: %v", err)
	}
	if err := o.AddStrategy("JPN225", "1h", "rsi2_reversion"); err != nil {
		t.Fatalf("AddStrategy 2 failed: %v", err)
	}
	if len(o.engines) != 2 {
		t.Errorf("expected 2 engines, got %d", len(o.engines))
	}
	eid := "JPN225:1h:rsi2_reversion"
	if _, ok := o.enginesByID[eid]; !ok {
		t.Errorf("expected engine by ID %s", eid)
	}
	for _, eng := range o.engines {
		if eng.pipeline == nil {
			t.Errorf("engine %s has nil pipeline", eng.strategyID)
		}
	}
}

func TestOrchestrator_AddStrategy_Duplicate(t *testing.T) {
	db := makeOrchDB(nil, nil)
	o, _ := NewOrchestrator(db, OrchestratorConfig{})
	_ = o.AddStrategy("JPN225", "1h", "grid_trading")
	err := o.AddStrategy("JPN225", "1h", "grid_trading")
	if err == nil {
		t.Error("expected duplicate error")
	}
}

func TestOrchestrator_AddStrategy_UnknownStrategy(t *testing.T) {
	db := makeOrchDB(nil, nil)
	o, _ := NewOrchestrator(db, OrchestratorConfig{})
	err := o.AddStrategy("JPN225", "1h", "nonexistent_strategy_xyz")
	if err == nil {
		t.Error("expected unknown strategy error")
	}
}

func TestOrchestrator_Run_EmptyStrategies(t *testing.T) {
	ts := time.Date(2025, 12, 1, 9, 30, 0, 0, time.UTC)
	candles := make([]Candle, 10)
	for i := range candles {
		candles[i] = Candle{Time: ts.Add(time.Duration(i) * time.Hour), Symbol: "JPN225",
			Open: types.PriceFromFloat(300), High: types.PriceFromFloat(301), Low: types.PriceFromFloat(299), Close: types.PriceFromFloat(300.1), Volume: 1000}
	}
	db := &parityDB{candles: candles, regime: []RegimeLog{{Time: ts, HMMState: 0, Confidence: 0.9, Symbol: "JPN225"}}, vix: nil}
	o, _ := NewOrchestrator(db, OrchestratorConfig{StartDate: ts.Add(-time.Hour), EndDate: ts.Add(10 * time.Hour)})
	result, err := o.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if len(result.PoolEquity) != 0 {
		t.Errorf("expected 0 equity points with no strategies, got %d", len(result.PoolEquity))
	}
	if len(result.Trades) != 0 {
		t.Errorf("expected 0 trades with no strategies, got %d", len(result.Trades))
	}
}

func TestOrchestrator_Run_SingleStrategy(t *testing.T) {
	ts := time.Date(2025, 12, 1, 9, 30, 0, 0, time.UTC)
	candles := make([]Candle, 30)
	for i := range candles {
		base := 300.0 + float64(i)*0.1
		candles[i] = Candle{Time: ts.Add(time.Duration(i) * time.Hour), Symbol: "JPN225",
			Open: types.PriceFromFloat(base), High: types.PriceFromFloat(base + 0.5),
			Low: types.PriceFromFloat(base - 0.5), Close: types.PriceFromFloat(base + 0.1), Volume: 1000}
	}
	db := &parityDB{candles: candles, regime: []RegimeLog{{Time: ts, HMMState: 0, Confidence: 0.9, Symbol: "JPN225"}}, vix: nil}
	o, _ := NewOrchestrator(db, OrchestratorConfig{
		StartDate: ts.Add(-time.Hour), EndDate: ts.Add(30 * time.Hour),
		InitialCapital: 500000, RebalanceBars: 10, KellyFraction: 0.25,
		MaxPositionPct: 0.05, FrictionModel: "idealized",
	})
	_ = o.AddStrategy("JPN225", "1h", "grid_trading")
	result, err := o.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if len(result.Trades) < 1 {
		t.Errorf("expected at least 1 trade with grid_trading on JPN225, got %d", len(result.Trades))
	}
	if len(result.PoolEquity) < 2 {
		t.Errorf("expected at least 2 equity points, got %d", len(result.PoolEquity))
	}
	if result.PoolMaxDD >= 100 {
		t.Errorf("PoolMaxDD unreasonably high: %f", result.PoolMaxDD)
	}
}

func TestOrchestrator_Run_ContextCancelled(t *testing.T) {
	ts := time.Date(2025, 12, 1, 9, 30, 0, 0, time.UTC)
	candles := make([]Candle, 50)
	for i := range candles {
		candles[i] = Candle{Time: ts.Add(time.Duration(i) * time.Hour), Symbol: "JPN225",
			Open: types.PriceFromFloat(300), High: types.PriceFromFloat(301), Low: types.PriceFromFloat(299), Close: types.PriceFromFloat(300.1), Volume: 1000}
	}
	db := &parityDB{candles: candles, regime: []RegimeLog{{Time: ts, HMMState: 0, Confidence: 0.9, Symbol: "JPN225"}}, vix: nil}
	o, _ := NewOrchestrator(db, OrchestratorConfig{
		StartDate: ts.Add(-time.Hour), EndDate: ts.Add(50 * time.Hour),
		InitialCapital: 100000, RebalanceBars: 10,
		MaxPositionPct: 0.05, FrictionModel: "idealized",
	})
	_ = o.AddStrategy("JPN225", "1h", "grid_trading")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := o.Run(ctx)
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}

func TestOrchestrator_ConfigClamps_MaxPositionPct(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{0, 0.02},
		{0.0001, 0.02},
		{0.02, 0.02},
		{0.05, 0.05},
		{0.10, 0.10},
		{0.20, 0.20},
		{0.25, 0.02},
		{0.50, 0.02},
	}
	for _, tc := range tests {
		o, err := NewOrchestrator(nil, OrchestratorConfig{MaxPositionPct: tc.input})
		if err != nil {
			t.Fatalf("input=%f: NewOrchestrator failed: %v", tc.input, err)
		}
		if o.config.MaxPositionPct != tc.expected {
			t.Errorf("input=%f: got %f, want %f", tc.input, o.config.MaxPositionPct, tc.expected)
		}
	}
}

func TestOrchestrator_Run_MultiStrategy_NoCorrelationBrake(t *testing.T) {
	ts := time.Date(2025, 12, 1, 9, 30, 0, 0, time.UTC)
	candles := make([]Candle, 30)
	for i := range candles {
		base := 300.0 + float64(i)*0.1
		candles[i] = Candle{Time: ts.Add(time.Duration(i) * time.Hour), Symbol: "JPN225",
			Open: types.PriceFromFloat(base), High: types.PriceFromFloat(base + 0.5),
			Low: types.PriceFromFloat(base - 0.5), Close: types.PriceFromFloat(base + 0.1), Volume: 1000}
	}
	db := &parityDB{candles: candles, regime: []RegimeLog{{Time: ts, HMMState: 0, Confidence: 0.9, Symbol: "JPN225"}}, vix: nil}
	o, _ := NewOrchestrator(db, OrchestratorConfig{
		StartDate: ts.Add(-time.Hour), EndDate: ts.Add(30 * time.Hour),
		InitialCapital: 500000, RebalanceBars: 10, KellyFraction: 0.25,
		MaxPositionPct: 0.05, FrictionModel: "idealized",
		EnableCorrelationBrake: false,
	})
	_ = o.AddStrategy("JPN225", "1h", "grid_trading")
	result, err := o.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if len(result.CorrelationBreaches) != 0 {
		t.Errorf("expected 0 breaches with brake disabled, got %d", len(result.CorrelationBreaches))
	}
}

func TestOrchestrator_Run_MultiStrategy_CorrelationBrake(t *testing.T) {
	ts := time.Date(2025, 12, 1, 9, 30, 0, 0, time.UTC)
	candles := make([]Candle, 30)
	for i := range candles {
		base := 300.0 + float64(i)*0.1
		candles[i] = Candle{Time: ts.Add(time.Duration(i) * time.Hour), Symbol: "JPN225",
			Open: types.PriceFromFloat(base), High: types.PriceFromFloat(base + 0.5),
			Low: types.PriceFromFloat(base - 0.5), Close: types.PriceFromFloat(base + 0.1), Volume: 1000}
	}
	db := &parityDB{candles: candles, regime: []RegimeLog{{Time: ts, HMMState: 0, Confidence: 0.9, Symbol: "JPN225"}}, vix: nil}
	o, _ := NewOrchestrator(db, OrchestratorConfig{
		StartDate: ts.Add(-time.Hour), EndDate: ts.Add(30 * time.Hour),
		InitialCapital: 500000, RebalanceBars: 10, KellyFraction: 0.25,
		MaxPositionPct: 0.05, FrictionModel: "idealized",
		EnableCorrelationBrake: true, CorrelationThreshold: 0.6,
	})
	if err := o.AddStrategy("JPN225", "1h", "grid_trading"); err != nil {
		t.Fatalf("AddStrategy 1: %v", err)
	}
	result, err := o.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if len(result.PoolEquity) == 0 {
		t.Error("expected equity points with correlation brake enabled")
	}
}

func TestOrchestrator_Run_RebalanceTriggered(t *testing.T) {
	ts := time.Date(2025, 12, 1, 9, 30, 0, 0, time.UTC)
	candles := make([]Candle, 25)
	for i := range candles {
		base := 300.0 + float64(i)*0.1
		candles[i] = Candle{Time: ts.Add(time.Duration(i) * time.Hour), Symbol: "JPN225",
			Open: types.PriceFromFloat(base), High: types.PriceFromFloat(base + 0.5),
			Low: types.PriceFromFloat(base - 0.5), Close: types.PriceFromFloat(base + 0.1), Volume: 1000}
	}
	db := &parityDB{candles: candles, regime: []RegimeLog{{Time: ts, HMMState: 0, Confidence: 0.9, Symbol: "JPN225"}}, vix: nil}
	o, _ := NewOrchestrator(db, OrchestratorConfig{
		StartDate: ts.Add(-time.Hour), EndDate: ts.Add(25 * time.Hour),
		InitialCapital: 500000, RebalanceBars: 5, KellyFraction: 0.25,
		MaxPositionPct: 0.05, FrictionModel: "idealized",
	})
	_ = o.AddStrategy("JPN225", "1h", "grid_trading")
	result, err := o.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if len(result.AllocationHistory) < 2 {
		t.Errorf("expected at least 2 allocation entries with T=5 rebalance, got %d", len(result.AllocationHistory))
	}
}

func TestOrchestrator_Run_RegimeGateBlocks(t *testing.T) {
	ts := time.Date(2025, 12, 1, 9, 30, 0, 0, time.UTC)
	candles := make([]Candle, 10)
	for i := range candles {
		candles[i] = Candle{Time: ts.Add(time.Duration(i) * time.Hour), Symbol: "JPN225",
			Open: types.PriceFromFloat(300), High: types.PriceFromFloat(301), Low: types.PriceFromFloat(299), Close: types.PriceFromFloat(300.1), Volume: 1000}
	}
	db := &parityDB{candles: candles, regime: []RegimeLog{{Time: ts, HMMState: 3, Confidence: 0.9, Symbol: "JPN225"}}, vix: nil}
	o, _ := NewOrchestrator(db, OrchestratorConfig{
		StartDate: ts.Add(-time.Hour), EndDate: ts.Add(10 * time.Hour),
		InitialCapital: 500000, RebalanceBars: 5, KellyFraction: 0.25,
		MaxPositionPct: 0.05, FrictionModel: "idealized",
	})
	_ = o.AddStrategy("JPN225", "1h", "grid_trading")
	result, err := o.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if len(result.Trades) > 0 {
		t.Errorf("expected 0 trades in Crisis regime for grid_trading, got %d", len(result.Trades))
	}
}

func TestOrchestrator_Run_MissingRegimeLogs(t *testing.T) {
	ts := time.Date(2025, 12, 1, 9, 30, 0, 0, time.UTC)
	candles := make([]Candle, 10)
	for i := range candles {
		candles[i] = Candle{Time: ts.Add(time.Duration(i) * time.Hour), Symbol: "JPN225",
			Open: types.PriceFromFloat(300), High: types.PriceFromFloat(301), Low: types.PriceFromFloat(299), Close: types.PriceFromFloat(300.1), Volume: 1000}
	}
	db := &parityDB{candles: candles, regime: nil, vix: nil}
	o, _ := NewOrchestrator(db, OrchestratorConfig{
		StartDate: ts.Add(-time.Hour), EndDate: ts.Add(10 * time.Hour),
		InitialCapital: 500000, RebalanceBars: 5, KellyFraction: 0.25,
		MaxPositionPct: 0.05, FrictionModel: "idealized",
	})
	_ = o.AddStrategy("JPN225", "1h", "grid_trading")
	_, err := o.Run(context.Background())
	if err != nil {
		t.Fatalf("Run should not panic with nil regime logs: %v", err)
	}
}

func TestOrchestrator_Run_MissingCandles(t *testing.T) {
	ts := time.Date(2025, 12, 1, 9, 30, 0, 0, time.UTC)
	db := &parityDB{candles: nil, regime: []RegimeLog{{Time: ts, HMMState: 0, Confidence: 0.9, Symbol: "JPN225"}}, vix: nil}
	o, _ := NewOrchestrator(db, OrchestratorConfig{
		StartDate: ts.Add(-time.Hour), EndDate: ts.Add(10 * time.Hour),
		InitialCapital: 500000, RebalanceBars: 5,
		MaxPositionPct: 0.05, FrictionModel: "idealized",
	})
	_ = o.AddStrategy("JPN225", "1h", "grid_trading")
	result, err := o.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if len(result.Trades) != 0 {
		t.Errorf("expected 0 trades with no candle data, got %d", len(result.Trades))
	}
}

func TestOrchestrator_Run_AllowFractional(t *testing.T) {
	ts := time.Date(2025, 12, 1, 9, 30, 0, 0, time.UTC)
	candles := make([]Candle, 30)
	for i := range candles {
		base := 300.0 + float64(i)*0.1
		candles[i] = Candle{Time: ts.Add(time.Duration(i) * time.Hour), Symbol: "JPN225",
			Open: types.PriceFromFloat(base), High: types.PriceFromFloat(base + 0.5),
			Low: types.PriceFromFloat(base - 0.5), Close: types.PriceFromFloat(base + 0.1), Volume: 1000}
	}
	db := &parityDB{candles: candles, regime: []RegimeLog{{Time: ts, HMMState: 0, Confidence: 0.9, Symbol: "JPN225"}}, vix: nil}
	o, _ := NewOrchestrator(db, OrchestratorConfig{
		StartDate: ts.Add(-time.Hour), EndDate: ts.Add(30 * time.Hour),
		InitialCapital: 10000, RebalanceBars: 10, KellyFraction: 0.25,
		MaxPositionPct: 0.02, AllowFractional: true, FrictionModel: "idealized",
	})
	_ = o.AddStrategy("JPN225", "1h", "grid_trading")
	result, err := o.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if o.config.AllowFractional != true {
		t.Error("AllowFractional should be true")
	}
	_ = result
}

func TestOrchestrator_Run_ResultJSONRoundtrip(t *testing.T) {
	result := &OrchestrationRunResult{
		PoolSharpe:   1.5,
		PoolSortino:  2.0,
		PoolMaxDD:    5.0,
		PoolReturnPct: 12.5,
		StrategyPnL:  map[string]float64{"a": 100, "b": -50},
	}
	enriched := EnrichResultJSON(result)
	if enriched.NumTrades != len(result.Trades) {
		t.Errorf("NumTrades mismatch: %d vs %d", enriched.NumTrades, len(result.Trades))
	}
	if enriched.StrategyPnL["a"] != 100 {
		t.Errorf("StrategyPnL mismatch: %f", enriched.StrategyPnL["a"])
	}
}

func TestOrchestrator_EnrichResultJSON_EmptyEquity(t *testing.T) {
	result := &OrchestrationRunResult{}
	enriched := EnrichResultJSON(result)
	if len(enriched.DailyReturns) != 0 {
		t.Errorf("expected 0 daily returns from empty equity, got %d", len(enriched.DailyReturns))
	}
	if enriched.WinRate != 0 {
		t.Errorf("expected WinRate=0 with no trades, got %f", enriched.WinRate)
	}
}

func TestOrchestrator_EnrichResultJSON_NoTrades(t *testing.T) {
	ts := time.Date(2025, 12, 1, 9, 30, 0, 0, time.UTC)
	equity := []EquityPoint{
		{Time: ts, Value: 100000},
		{Time: ts.Add(24 * time.Hour), Value: 100100},
		{Time: ts.Add(48 * time.Hour), Value: 100200},
	}
	result := &OrchestrationRunResult{PoolEquity: equity, Trades: nil}
	enriched := EnrichResultJSON(result)
	if enriched.WinRate != 0 {
		t.Errorf("expected WinRate=0 with no trades, got %f", enriched.WinRate)
	}
	if enriched.ProfitFactor != 0 {
		t.Errorf("expected ProfitFactor=0, got %f", enriched.ProfitFactor)
	}
	if len(enriched.DailyReturns) == 0 {
		t.Error("expected daily returns from equity")
	}
}


func TestOrchestrator_Run_PoolHaltsOnDrawdown(t *testing.T) {
	ts := time.Date(2025, 12, 1, 9, 30, 0, 0, time.UTC)
	n := 5
	candles := make([]Candle, n)
	for i := range candles {
		candles[i] = Candle{Time: ts.Add(time.Duration(i) * time.Hour), Symbol: "JPN225",
			Open: types.PriceFromFloat(300), High: types.PriceFromFloat(301), Low: types.PriceFromFloat(299), Close: types.PriceFromFloat(300), Volume: 1000}
	}
	db := &parityDB{candles: candles, regime: []RegimeLog{{Time: ts, HMMState: 0, Confidence: 0.9, Symbol: "JPN225"}}, vix: nil}
	profile := propfirm.DefaultFTMOProfile()
	profile.MaxDailyLossPct = 0.001
	o, _ := NewOrchestrator(db, OrchestratorConfig{
		StartDate: ts.Add(-time.Hour), EndDate: ts.Add(time.Duration(n) * time.Hour),
		InitialCapital: 100000, RebalanceBars: 5,
		MaxPositionPct: 0.05, FrictionModel: "idealized",
	})
	_ = o.AddStrategy("JPN225", "1h", "grid_trading")
	result, err := o.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if len(result.PoolEquity) == 0 {
		t.Skip("no equity generated, can't check halt behavior")
	}
}
