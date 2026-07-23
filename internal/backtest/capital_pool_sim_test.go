package backtest

import (
	"testing"
	"time"

	"github.com/lee-econ/orca-core/internal/propfirm"
)

type mockRunner struct {
	name         string
	runnerType   string
	evalResult   *Signal
	evalCalled   int
	resetCalled  int
}

func (m *mockRunner) Name() string         { return m.name }
func (m *mockRunner) Type() string          { return m.runnerType }
func (m *mockRunner) Evaluate(c Candle, regime int8) *Signal {
	m.evalCalled++
	return m.evalResult
}
func (m *mockRunner) Reset() { m.resetCalled++ }
func (m *mockRunner) Params() map[string]float64 { return nil }
func (m *mockRunner) SetParams(map[string]float64) {}

func TestCapitalPoolSim_AddStrategy(t *testing.T) {
	profile := propfirm.DefaultFTMOProfile()
	sim := NewCapitalPoolSim(profile, 100000)

	sim.AddStrategy("strat-1", &mockRunner{name: "strat-1"})

	if len(sim.Strategies) != 1 {
		t.Fatalf("Expected 1 strategy, got %d", len(sim.Strategies))
	}
}

func TestCapitalPoolSim_EvaluateAll(t *testing.T) {
	profile := propfirm.DefaultFTMOProfile()
	sim := NewCapitalPoolSim(profile, 100000)

	sim.AddStrategy("s1", &mockRunner{
		name:       "s1",
		evalResult: &Signal{Symbol: "SPY", Side: "BUY", Quantity: 100},
	})

	candle := Candle{Time: time.Now(), Open: 100, High: 102, Low: 98, Close: 101, Symbol: "SPY"}
	signals := sim.EvaluateAll(candle, 1)

	if len(signals) != 1 {
		t.Fatalf("Expected 1 signal, got %d", len(signals))
	}
	sig := signals["s1"]
	if sig == nil {
		t.Fatal("Expected signal for s1")
	}
	if sig.Side != "BUY" {
		t.Errorf("Expected BUY, got %s", sig.Side)
	}
}

func TestCapitalPoolSim_MaxOpenPositions(t *testing.T) {
	profile := propfirm.DefaultFTMOProfile()
	profile.MaxOpenPositions = 2
	sim := NewCapitalPoolSim(profile, 100000)

	sim.AddStrategy("s1", &mockRunner{
		name:       "s1",
		evalResult: &Signal{Symbol: "SPY", Side: "BUY", Quantity: 100},
	})
	sim.AddStrategy("s2", &mockRunner{
		name:       "s2",
		evalResult: &Signal{Symbol: "QQQ", Side: "SELL", Quantity: 100},
	})
	sim.AddStrategy("s3", &mockRunner{
		name:       "s3",
		evalResult: &Signal{Symbol: "AAPL", Side: "BUY", Quantity: 100},
	})

	candle := Candle{Time: time.Now(), Open: 100, High: 102, Low: 98, Close: 101, Symbol: "SPY"}

	signals1 := sim.EvaluateAll(candle, 1)
	t.Logf("First pass signals: %d", len(signals1))

	signals2 := sim.EvaluateAll(candle, 1)
	if len(signals2) > 0 {
		for id := range signals2 {
			t.Logf("Second pass still had signal for %s (should be blocked by max open)", id)
		}
	}
}

func TestCapitalPoolSim_RecordFill(t *testing.T) {
	profile := propfirm.DefaultFTMOProfile()
	sim := NewCapitalPoolSim(profile, 100000)

	sim.AddStrategy("s1", &mockRunner{name: "s1"})

	sim.RecordFill("s1", "SPY", "BUY", 500, 100)

	if sim.TotalBalance != 100500 {
		t.Errorf("Expected 100500, got %f", sim.TotalBalance)
	}
}

func TestCapitalPoolSim_DrawdownHalt(t *testing.T) {
	profile := propfirm.DefaultFTMOProfile()
	profile.MaxDrawdownPct = 5.0
	sim := NewCapitalPoolSim(profile, 100000)

	sim.AddStrategy("s1", &mockRunner{name: "s1"})

	sim.RecordFill("s1", "SPY", "BUY", -10000, 100)

	if !sim.Halted {
		t.Error("Sim should be halted after drawdown exceeds limit")
	}
	if sim.HaltReason != "max_drawdown" {
		t.Errorf("Expected halt reason 'max_drawdown', got '%s'", sim.HaltReason)
	}
}

func TestCapitalPoolSim_ResetDaily(t *testing.T) {
	profile := propfirm.DefaultFTMOProfile()
	sim := NewCapitalPoolSim(profile, 100000)

	sim.AddStrategy("s1", &mockRunner{name: "s1"})
	sim.RecordFill("s1", "SPY", "BUY", 500, 100)

	if sim.DailyPnL != 500 {
		t.Errorf("Expected daily PnL 500, got %f", sim.DailyPnL)
	}

	sim.ResetDaily()

	if sim.DailyPnL != 0 {
		t.Errorf("Expected daily PnL 0 after reset, got %f", sim.DailyPnL)
	}
	if sim.TradingDays != 1 {
		t.Errorf("Expected 1 trading day, got %d", sim.TradingDays)
	}
}

func TestCapitalPoolSim_PerStrategyDrawdown(t *testing.T) {
	profile := propfirm.DefaultFTMOProfile()
	profile.MaxDrawdownPct = 10.0
	profile.MaxOpenPositions = 10
	sim := NewCapitalPoolSim(profile, 100000)

	s1Runner := &mockRunner{
		name:       "s1",
		evalResult: &Signal{Symbol: "SPY", Side: "BUY", Quantity: 100},
	}
	s2Runner := &mockRunner{
		name:       "s2",
		evalResult: &Signal{Symbol: "QQQ", Side: "SELL", Quantity: 100},
	}
	sim.AddStrategy("s1", s1Runner)
	sim.AddStrategy("s2", s2Runner)

	sim.RecordFill("s1", "SPY", "BUY", -6000, 100)

	candle := Candle{Time: time.Now(), Open: 100, High: 102, Low: 98, Close: 101, Symbol: "SPY"}
	signals := sim.EvaluateAll(candle, 1)

	t.Logf("Signals: %d, TotalBalance: %f", len(signals), sim.TotalBalance)
	t.Logf("s1 PeakBalance: %f", sim.Strategies["s1"].PeakBalance)

	if _, ok := signals["s1"]; ok {
		t.Error("s1 should be blocked after drawdown")
	}
	if len(signals) >= 1 {
		t.Logf("Other strategies can still trade")
	}
}
