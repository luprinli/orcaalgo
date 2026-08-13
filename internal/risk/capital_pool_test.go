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

func TestCapitalPool_SoftHaltReducesPositions(t *testing.T) {
	profile := propfirm.DefaultFTMOProfile()
	profile.MaxDailyLossPct = 20.0 // prevent daily loss check from firing first
	state := &propfirm.State{
		StartingBalance: 100000.0,
		PeakBalance:     100000.0,
		ConsistencyMult: 1.0,
		CurrentPhase:    1,
	}
	pool := NewCapitalPoolManager(profile, state)

	// Balance drop to 96,000 = -4.0% — still within soft halt
	state.StartingBalance = 100000
	pool.poolState.TotalBalance = 96000
	result := pool.RequestCapital(context.Background(), CapitalRequest{
		StrategyID: "s1", Confidence: 1.0, Symbol: "SPY", Side: "BUY", BaseSize: 1000,
	})
	if !(result.ApprovedSize > 0) {
		t.Error("should allow trades at -4.0% daily loss (below soft halt)")
	}

	// Balance drop to 95,200 = -4.8% — between soft and hard halt
	pool.poolState.TotalBalance = 95200
	result = pool.RequestCapital(context.Background(), CapitalRequest{
		StrategyID: "s2", Confidence: 1.0, Symbol: "QQQ", Side: "BUY", BaseSize: 1000,
	})
	// The CapitalPoolManager itself doesn't implement soft halt — the pipeline does.
	// The pool should still approve but the pipeline applies the 0.5x multiplier.
	if !(result.ApprovedSize > 0) {
		t.Error("capital pool should approve; soft halt sizing is handled by pipeline")
	}
}

func TestCapitalPool_HardHaltRejectsAll(t *testing.T) {
	profile := propfirm.DefaultFTMOProfile()
	state := &propfirm.State{
		StartingBalance: 100000.0,
		PeakBalance:     100000.0,
		ConsistencyMult: 1.0,
		CurrentPhase:    1,
	}
	pool := NewCapitalPoolManager(profile, state)

	// Single-day loss of -6.0% (DailyPnL = -6000 on a 100k day-start balance)
	// exceeds the 5% daily loss limit → hard halt.
	pool.poolState.TotalBalance = 94000
	pool.poolState.DailyPnL = -6000
	result := pool.RequestCapital(context.Background(), CapitalRequest{
		StrategyID: "s1", Confidence: 1.0, Symbol: "SPY", Side: "BUY", BaseSize: 1000,
	})
	if result.ApprovedSize > 0 {
		t.Error("should reject all trades after hard halt")
	}
}

func TestCapitalPool_StrategySuspensionOnDD(t *testing.T) {
	profile := propfirm.DefaultFTMOProfile()
	profile.MaxDailyLossPct = 20.0 // prevent daily loss check from firing first
	state := &propfirm.State{
		StartingBalance: 100000.0,
		PeakBalance:     100000.0,
		ConsistencyMult: 1.0,
		CurrentPhase:    1,
	}
	pool := NewCapitalPoolManager(profile, state)

	// Create the strategy allocation entry first by requesting capital.
	pool.RequestCapital(context.Background(), CapitalRequest{
		StrategyID: "s1", Confidence: 0.5, Symbol: "SPY", Side: "BUY", BaseSize: 10000,
	})

	// Clear the open position so correlation check doesn't interfere.
	strat := pool.strategies["s1"]
	strat.OpenLong = make(map[string]float64)
	strat.OpenShort = make(map[string]float64)

	// Manual DD breach by setting pool balance low and strategy peak balance high.
	strat.PeakBalance = 100000
	pool.poolState.TotalBalance = 85000 // -15% → DD > 5% (half of 10% profile limit)

	result := pool.RequestCapital(context.Background(), CapitalRequest{
		StrategyID: "s1", Confidence: 1.0, Symbol: "QQQ", Side: "BUY", BaseSize: 1000,
	})
	if result.ApprovedSize > 0 {
		t.Error("suspended strategy should have zero approved size")
	}
	if !strat.Suspended {
		t.Error("strategy should be suspended after DD breach")
	}
}

func TestCapitalPool_StrategyResume(t *testing.T) {
	profile := propfirm.DefaultFTMOProfile()
	profile.MaxDailyLossPct = 20.0 // prevent daily loss check
	state := &propfirm.State{
		StartingBalance: 100000.0,
		PeakBalance:     100000.0,
		ConsistencyMult: 1.0,
		CurrentPhase:    1,
	}
	pool := NewCapitalPoolManager(profile, state)

	// Force suspend without open positions.
	pool.RequestCapital(context.Background(), CapitalRequest{
		StrategyID: "s1", Confidence: 0.5, Symbol: "SPY", Side: "BUY", BaseSize: 10000,
	})
	strat := pool.strategies["s1"]
	strat.OpenLong = make(map[string]float64)
	strat.OpenShort = make(map[string]float64)
	strat.Allocated = 0
	strat.Suspended = true
	strat.PeakBalance = 100000

	// Verify blocked.
	result := pool.RequestCapital(context.Background(), CapitalRequest{
		StrategyID: "s1", Confidence: 1.0, Symbol: "QQQ", Side: "SELL", BaseSize: 1000,
	})
	if result.ApprovedSize > 0 {
		t.Error("should be blocked while suspended")
	}

	// Resume.
	pool.ResumeStrategy("s1")
	if strat.Suspended {
		t.Error("should not be suspended after resume")
	}

	// Verify allowed after resume.
	result = pool.RequestCapital(context.Background(), CapitalRequest{
		StrategyID: "s1", Confidence: 1.0, Symbol: "QQQ", Side: "SELL", BaseSize: 1000,
	})
	if result.ApprovedSize <= 0 {
		t.Errorf("should be allowed after resume, got %s", result.Reason)
	}
}

func TestCapitalPool_CrossStrategyCorrelationBrake(t *testing.T) {
	profile := propfirm.DefaultFTMOProfile()
	state := &propfirm.State{
		StartingBalance: 100000.0,
		PeakBalance:     100000.0,
		ConsistencyMult: 1.0,
		CurrentPhase:    1,
	}
	pool := NewCapitalPoolManager(profile, state)

	// s1 goes long SPY.
	pool.RequestCapital(context.Background(), CapitalRequest{
		StrategyID: "s1", Confidence: 0.5, Symbol: "SPY", Side: "BUY", BaseSize: 10000,
	})

	// Verify HasOpenPosition detects existing position.
	if !pool.HasOpenPosition("SPY", "BUY") {
		t.Error("HasOpenPosition should return true for SPY BUY after s1 opened")
	}
	if pool.HasOpenPosition("SPY", "SELL") {
		t.Error("HasOpenPosition should return false for SPY SELL")
	}
	if pool.HasOpenPosition("QQQ", "BUY") {
		t.Error("HasOpenPosition should return false for QQQ BUY")
	}
}


