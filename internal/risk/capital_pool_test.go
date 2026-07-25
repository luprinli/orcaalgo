package risk

import (
	"context"
	"testing"

	"github.com/lee-econ/orca-core/internal/propfirm"
)

func TestCapitalPool_RequestCapitalApproved(t *testing.T) {
	profile := propfirm.DefaultFTMOProfile()
	state := &propfirm.State{
		StartingBalance: 100000.0,
		PeakBalance:     100000.0,
		ConsistencyMult: 1.0,
		CurrentPhase:    1,
	}
	pool := NewCapitalPoolManager(profile, state)

	result := pool.RequestCapital(context.Background(), CapitalRequest{
		StrategyID: "strat-1",
		Confidence: 0.75,
		Symbol:     "SPY",
		Side:       "BUY",
		BaseSize:   100000.0,
	})

	if result.ApprovedSize <= 0 {
		t.Fatalf("Expected approved size > 0, got %f, reason: %s", result.ApprovedSize, result.Reason)
	}
	if result.Reason != "ok" {
		t.Errorf("Expected reason 'ok', got '%s'", result.Reason)
	}
}

func TestCapitalPool_MaxOpenPositions(t *testing.T) {
	profile := propfirm.DefaultFTMOProfile()
	profile.MaxOpenPositions = 2
	state := &propfirm.State{
		StartingBalance: 100000.0,
		PeakBalance:     100000.0,
	}
	pool := NewCapitalPoolManager(profile, state)

	pool.RequestCapital(context.Background(), CapitalRequest{StrategyID: "s1", Confidence: 0.5, Symbol: "SPY", Side: "BUY", BaseSize: 100000})
	pool.RequestCapital(context.Background(), CapitalRequest{StrategyID: "s2", Confidence: 0.5, Symbol: "QQQ", Side: "SELL", BaseSize: 100000})

	result := pool.RequestCapital(context.Background(), CapitalRequest{StrategyID: "s3", Confidence: 0.5, Symbol: "AAPL", Side: "BUY", BaseSize: 100000})
	if result.ApprovedSize != 0 {
		t.Errorf("Expected 0 size (max open positions exceeded), got %f", result.ApprovedSize)
	}
}

func TestCapitalPool_ViolatedState(t *testing.T) {
	profile := propfirm.DefaultFTMOProfile()
	state := &propfirm.State{
		StartingBalance: 100000.0,
		Violated:        true,
		ViolationReason: "daily_loss_limit",
	}
	pool := NewCapitalPoolManager(profile, state)

	result := pool.RequestCapital(context.Background(), CapitalRequest{
		StrategyID: "s1", Confidence: 0.8, Symbol: "SPY", Side: "BUY", BaseSize: 100000,
	})
	if result.ApprovedSize != 0 {
		t.Errorf("Expected 0 from violated state, got %f", result.ApprovedSize)
	}
}

func TestCapitalPool_PerStrategyDrawdown(t *testing.T) {
	profile := propfirm.DefaultFTMOProfile()
	profile.MaxDrawdownPct = 10.0
	state := &propfirm.State{
		StartingBalance: 100000.0,
		PeakBalance:     100000.0,
	}
	pool := NewCapitalPoolManager(profile, state)

	pool.RequestCapital(context.Background(), CapitalRequest{StrategyID: "s1", Confidence: 0.5, Symbol: "SPY", Side: "BUY", BaseSize: 100000})

	pool.RecordFill("s1", "SPY", "BUY", -8000, 50)

	result := pool.RequestCapital(context.Background(), CapitalRequest{StrategyID: "s1", Confidence: 0.5, Symbol: "QQQ", Side: "BUY", BaseSize: 100000})
	if result.ApprovedSize != 0 {
		t.Errorf("Expected 0 after large drawdown, got %f", result.ApprovedSize)
	}
}

func TestCapitalPool_CorrelationReduction(t *testing.T) {
	profile := propfirm.DefaultFTMOProfile()
	state := &propfirm.State{
		StartingBalance: 100000.0,
		PeakBalance:     100000.0,
	}
	pool := NewCapitalPoolManager(profile, state)

	r1 := pool.RequestCapital(context.Background(), CapitalRequest{StrategyID: "s1", Confidence: 0.5, Symbol: "SPY", Side: "BUY", BaseSize: 100000})
	if r1.ApprovedSize <= 0 {
		t.Fatal("First request should be approved")
	}

	r2 := pool.RequestCapital(context.Background(), CapitalRequest{StrategyID: "s1", Confidence: 0.5, Symbol: "SPY", Side: "BUY", BaseSize: 100000})
	if r2.ApprovedSize <= 0 {
		t.Logf("Correlation reduced second size: %f", r2.ApprovedSize)
	}
}

func TestCapitalPool_RecordFill(t *testing.T) {
	profile := propfirm.DefaultFTMOProfile()
	state := &propfirm.State{
		StartingBalance: 100000.0,
		PeakBalance:     100000.0,
	}
	pool := NewCapitalPoolManager(profile, state)

	pool.RequestCapital(context.Background(), CapitalRequest{StrategyID: "s1", Confidence: 0.5, Symbol: "SPY", Side: "BUY", BaseSize: 100000})
	pool.RecordFill("s1", "SPY", "BUY", 500, 100)

	metrics := pool.StrategyMetrics()
	found := false
	for _, m := range metrics {
		if m.StrategyID == "s1" {
			found = true
			if m.DailyPnL != 500 {
				t.Errorf("Expected daily PnL 500, got %f", m.DailyPnL)
			}
		}
	}
	if !found {
		t.Error("Strategy s1 not found in metrics")
	}
}

func TestCapitalPool_DailyReset(t *testing.T) {
	profile := propfirm.DefaultFTMOProfile()
	state := &propfirm.State{
		StartingBalance: 100000.0,
		PeakBalance:     100000.0,
	}
	pool := NewCapitalPoolManager(profile, state)

	pool.RequestCapital(context.Background(), CapitalRequest{StrategyID: "s1", Confidence: 0.5, Symbol: "SPY", Side: "BUY", BaseSize: 100000})
	pool.RecordFill("s1", "SPY", "BUY", 500, 100)

	pool.ResetDaily()

	metrics := pool.StrategyMetrics()
	for _, m := range metrics {
		if m.DailyPnL != 0 {
			t.Errorf("Expected daily PnL 0 after reset, got %f for %s", m.DailyPnL, m.StrategyID)
		}
	}
}

func TestCapitalPool_TotalBalance(t *testing.T) {
	profile := propfirm.DefaultFTMOProfile()
	state := &propfirm.State{
		StartingBalance: 100000.0,
		PeakBalance:     100000.0,
	}
	pool := NewCapitalPoolManager(profile, state)

	if pool.TotalBalance() != 100000.0 {
		t.Errorf("Expected 100000, got %f", pool.TotalBalance())
	}

	pool.RequestCapital(context.Background(), CapitalRequest{StrategyID: "s1", Confidence: 0.5, Symbol: "SPY", Side: "BUY", BaseSize: 100000})
	pool.RecordFill("s1", "SPY", "BUY", 1000, 100)

	if pool.TotalBalance() != 101000.0 {
		t.Errorf("Expected 101000 after +1000 PnL, got %f", pool.TotalBalance())
	}
}


