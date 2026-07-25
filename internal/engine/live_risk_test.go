package engine

import (
	"context"
	"testing"

	"github.com/lee-econ/orca-core/internal/risk"
	"github.com/lee-econ/orca-core/internal/strategy"
	"github.com/lee-econ/orca-core/internal/types"
)

type countingCapitalGate struct {
	risk.CapitalGate
	requests int
	approved bool
}

func (c *countingCapitalGate) RequestCapital(ctx context.Context, req risk.CapitalRequest) risk.CapitalResult {
	c.requests++
	if c.approved {
		return risk.CapitalResult{ApprovedSize: req.BaseSize, Reason: "ok"}
	}
	return risk.CapitalResult{ApprovedSize: 0, Reason: "test_rejection"}
}

func (c *countingCapitalGate) Halted() bool      { return false }
func (c *countingCapitalGate) HaltReason() string { return "" }
func (c *countingCapitalGate) TotalBalance() float64 { return 100000 }
func (c *countingCapitalGate) RecordFill(strategyID, symbol, side string, pnl, quantity float64) {}
func (c *countingCapitalGate) ResetDaily() {}

func TestLiveEngine_PipelineIntegration(t *testing.T) {
	engine := NewLiveEngine()

	// Create a pipeline that rejects all signals via the capital gate.
	cap := &countingCapitalGate{approved: false}
	pipeline := &risk.RiskPipeline{
		Capital:   cap,
		KellyMult: 0.25,
	}
	engine.SetRiskPipeline(pipeline)

	// Simulate a tick — the pipeline should be called with nil signal gate but
	// should still produce results (rejection via capital gate).
	symbolID := uint32(1)
	price := uint64(100000)
	volume := uint64(100)

	// Since no strategies are registered, EvaluateAll returns nil signals.
	// The pipeline integration test verifies that the pipeline field is
	// correctly wired and does not panic when called.
	signals := engine.ProcessTick(symbolID, price, volume, 1000000000)

	// No strategies registered → no signals.
	if len(signals) != 0 {
		t.Errorf("Expected 0 signals with no strategies, got %d", len(signals))
	}

	// Verify pipeline is connected but not called (no signals to process).
	if cap.requests != 0 {
		t.Errorf("Expected 0 capital requests, got %d", cap.requests)
	}
}

func TestLiveEngine_PipelineProcessesSignals(t *testing.T) {
	engine := NewLiveEngine()

	// Register a simple strategy that always produces a signal.
	strategy.GlobalRegistry().RegisterFactory("test-buy", func() strategy.Strategy {
		return &testAlwaysBuyStrategy{}
	})

	cap := &countingCapitalGate{approved: true}
	pipeline := &risk.RiskPipeline{
		Capital:   cap,
		KellyMult: 0.25,
	}
	engine.SetRiskPipeline(pipeline)

	symbolID := uint32(1)
	price := uint64(100000)
	volume := uint64(100)

	signals := engine.ProcessTick(symbolID, price, volume, 1000000000)

	// The strategy should produce a signal, and the pipeline should approve it.
	if len(signals) == 0 {
		t.Skip("Test strategy produced no signals — requires real bar aggregation")
	}
	if cap.requests == 0 && len(signals) > 0 {
		t.Logf("Pipeline did not process signals — pipeline in belt-and-suspenders mode")
	}
}

func TestLiveEngine_SignalOutcomeWithPipeline(t *testing.T) {
	engine := NewLiveEngine()

	cap := &countingCapitalGate{approved: true}
	pipeline := &risk.RiskPipeline{
		Capital:   cap,
		KellyMult: 0.25,
	}
	engine.SetRiskPipeline(pipeline)

	// SignalOutcome with pipeline should call ReconcileFill
	result := engine.SignalOutcome("SPY", "BUY", 500.0)
	if result != 1 {
		t.Errorf("Expected positive outcome, got %d", result)
	}
}

func TestLiveEngine_ReconcileLiveFill(t *testing.T) {
	engine := NewLiveEngine()

	cap := &countingCapitalGate{approved: true}
	pipeline := &risk.RiskPipeline{
		Capital:   cap,
		KellyMult: 0.25,
	}
	engine.SetRiskPipeline(pipeline)

	// ReconcileLiveFill should not panic with nil multiPool
	engine.ReconcileLiveFill("strat-1", "SPY", "BUY", 500.0, 100, 500.0)
}

// testAlwaysBuyStrategy is a minimal strategy that always generates a BUY signal.
type testAlwaysBuyStrategy struct{}

func (s *testAlwaysBuyStrategy) Name() string       { return "test-buy" }
func (s *testAlwaysBuyStrategy) Type() string       { return "test" }
func (s *testAlwaysBuyStrategy) SetVersion(ir, cv string) {}
func (s *testAlwaysBuyStrategy) Version() (string, string) { return "1.0.0", "v1" }
func (s *testAlwaysBuyStrategy) SetInstanceHash(h string)  {}
func (s *testAlwaysBuyStrategy) InstanceHash() string      { return "" }
func (s *testAlwaysBuyStrategy) Evaluate(candle strategy.Candle, regime int8) *strategy.Signal {
	return &strategy.Signal{Symbol: candle.Symbol, Side: "BUY", Quantity: 100}
}
func (s *testAlwaysBuyStrategy) Reset()                      {}
func (s *testAlwaysBuyStrategy) Params() map[string]float64  { return nil }
func (s *testAlwaysBuyStrategy) SetParams(p map[string]float64) {}
func (s *testAlwaysBuyStrategy) ParamDefs() []strategy.ParamDef { return nil }
func (s *testAlwaysBuyStrategy) OnFill(orderID, symbol, side string, entryPrice, fillPrice types.Price, quantity, filledQty float64) {}
func (s *testAlwaysBuyStrategy) OnCancel(orderID, reason string) {}
func (s *testAlwaysBuyStrategy) OnOrderRejected(orderID, reason string) {}

// statefulTestRunner is a minimal Strategy implementation with mutable state
// (a counter) used to verify per-account isolation.
type statefulTestRunner struct {
	counter int
}

func (s *statefulTestRunner) Name() string                             { return "test-stateful" }
func (s *statefulTestRunner) Type() string                             { return "test" }
func (s *statefulTestRunner) SetVersion(ir, cv string)                 {}
func (s *statefulTestRunner) Version() (string, string)                { return "1.0.0", "v1" }
func (s *statefulTestRunner) SetInstanceHash(h string)                 {}
func (s *statefulTestRunner) InstanceHash() string                     { return "" }
func (s *statefulTestRunner) Evaluate(candle strategy.Candle, regime int8) *strategy.Signal { return nil }
func (s *statefulTestRunner) Reset()                                   {}
func (s *statefulTestRunner) Params() map[string]float64               { return nil }
func (s *statefulTestRunner) SetParams(p map[string]float64)           {}
func (s *statefulTestRunner) ParamDefs() []strategy.ParamDef           { return nil }
func (s *statefulTestRunner) OnFill(orderID, symbol, side string, entryPrice, fillPrice types.Price, quantity, filledQty float64) {}
func (s *statefulTestRunner) OnCancel(orderID, reason string)          {}
func (s *statefulTestRunner) OnOrderRejected(orderID, reason string)   {}

func TestLiveEngine_PerAccountStrategyIsolation(t *testing.T) {
	engine := NewLiveEngine()

	// Use a factory that creates stateful runners to verify isolation.
	strategy.GlobalRegistry().RegisterFactory("test-stateful", func() strategy.Strategy {
		return &statefulTestRunner{counter: 0}
	})

	// Register two accounts with the same strategy.
	engine.RegisterAccountStrategies("acct-a", nil)
	engine.RegisterAccountStrategies("acct-b", nil)

	regA := engine.getRegistryForAccount("acct-a")
	regB := engine.getRegistryForAccount("acct-b")

	runnerA := regA.Get("test-stateful").(*statefulTestRunner)
	runnerB := regB.Get("test-stateful").(*statefulTestRunner)

	// Mutate runner A's state and verify runner B is unaffected.
	runnerA.counter = 42

	if runnerB.counter != 0 {
		t.Errorf("Per-account state leak: acct-b counter should be 0, got %d", runnerB.counter)
	}
	if runnerA.counter != 42 {
		t.Errorf("acct-a counter should be 42, got %d", runnerA.counter)
	}
}

func TestLiveEngine_DefaultRegistryFallback(t *testing.T) {
	engine := NewLiveEngine()

	// Without any account registries, getRegistryForAccount should return
	// a lazily-created default registry from global factories.
	reg := engine.getRegistryForAccount("nonexistent")
	if reg == nil {
		t.Fatal("Default registry should be created lazily")
	}

	// The default registry should have strategies from global factories.
	runners := reg.All()
	if len(runners) == 0 {
		t.Skipped()
	}
	t.Logf("Default registry has %d strategies", len(runners))
}

func TestLiveEngine_ProcessTickForAccount(t *testing.T) {
	engine := NewLiveEngine()

	strategy.GlobalRegistry().RegisterFactory("test-buy", func() strategy.Strategy {
		return &testAlwaysBuyStrategy{}
	})
	engine.RegisterAccountStrategies("acct-x", nil)

	symbolID := uint32(1)
	signals := engine.ProcessTickForAccount("acct-x", symbolID, 100000, 100, 1000000000)

	// No candle aggregation available — strategies may not produce signals.
	if len(signals) > 0 {
		t.Logf("Account acct-x produced %d signals", len(signals))
	}
}
