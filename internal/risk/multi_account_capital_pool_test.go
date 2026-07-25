package risk

import (
	"context"
	"testing"

	"github.com/lee-econ/orca-core/internal/propfirm"
)

func TestMultiAccountCapitalPool_RegisterAndRequest(t *testing.T) {
	m := NewMultiAccountCapitalPool()
	profile := propfirm.DefaultFTMOProfile()
	state := &propfirm.State{
		StartingBalance: 100000.0,
		PeakBalance:     100000.0,
		ConsistencyMult: 1.0,
	}

	m.RegisterPool("acct-1", profile, state)
	m.RegisterPool("acct-2", profile, &propfirm.State{
		StartingBalance: 200000.0,
		PeakBalance:     200000.0,
		ConsistencyMult: 1.0,
	})

	result1, err := m.RequestCapital(context.Background(), "acct-1", CapitalRequest{
		StrategyID: "s1", Confidence: 0.5, Symbol: "SPY", Side: "BUY", BaseSize: 100000,
	})
	if err != nil {
		t.Fatalf("RequestCapital should succeed: %v", err)
	}
	if result1.ApprovedSize <= 0 {
		t.Errorf("expected approved size > 0 for acct-1, got %f", result1.ApprovedSize)
	}

	result2, err := m.RequestCapital(context.Background(), "acct-2", CapitalRequest{
		StrategyID: "s2", Confidence: 0.5, Symbol: "QQQ", Side: "BUY", BaseSize: 200000,
	})
	if err != nil {
		t.Fatalf("RequestCapital should succeed: %v", err)
	}
	if result2.ApprovedSize <= 0 {
		t.Errorf("expected approved size > 0 for acct-2, got %f", result2.ApprovedSize)
	}
}

func TestMultiAccountCapitalPool_AccountNotFound(t *testing.T) {
	m := NewMultiAccountCapitalPool()

	_, err := m.RequestCapital(context.Background(), "nonexistent", CapitalRequest{
		StrategyID: "s1", Confidence: 0.5, Symbol: "SPY", Side: "BUY", BaseSize: 100000,
	})
	if err == nil {
		t.Error("expected error for nonexistent account")
	}
}

func TestMultiAccountCapitalPool_Isolation(t *testing.T) {
	m := NewMultiAccountCapitalPool()
	profile := propfirm.DefaultFTMOProfile()
	profile.MaxOpenPositions = 1

	m.RegisterPool("acct-a", profile, &propfirm.State{
		StartingBalance: 100000.0, PeakBalance: 100000.0,
	})
	m.RegisterPool("acct-b", profile, &propfirm.State{
		StartingBalance: 100000.0, PeakBalance: 100000.0,
	})

	result1, _ := m.RequestCapital(context.Background(), "acct-a", CapitalRequest{
		StrategyID: "s1", Confidence: 0.5, Symbol: "SPY", Side: "BUY", BaseSize: 100000,
	})
	if result1.ApprovedSize <= 0 {
		t.Fatalf("first request for acct-a should be approved, got reason: %s", result1.Reason)
	}

	result2, _ := m.RequestCapital(context.Background(), "acct-b", CapitalRequest{
		StrategyID: "s2", Confidence: 0.5, Symbol: "QQQ", Side: "BUY", BaseSize: 100000,
	})
	if result2.ApprovedSize <= 0 {
		t.Errorf("acct-b should be independent; first request should be approved, got reason: %s", result2.Reason)
	}
}

func TestMultiAccountCapitalPool_RecordFill(t *testing.T) {
	m := NewMultiAccountCapitalPool()
	profile := propfirm.DefaultFTMOProfile()
	state := &propfirm.State{
		StartingBalance: 100000.0,
		PeakBalance:     100000.0,
	}

	m.RegisterPool("acct-1", profile, state)

	m.RequestCapital(context.Background(), "acct-1", CapitalRequest{
		StrategyID: "s1", Confidence: 0.5, Symbol: "SPY", Side: "BUY", BaseSize: 100000,
	})
	if err := m.RecordFill("acct-1", "s1", "SPY", "BUY", 500.0, 100.0); err != nil {
		t.Fatalf("RecordFill should succeed: %v", err)
	}

	balance, _ := m.TotalBalance("acct-1")
	if balance != 100500.0 {
		t.Errorf("expected balance 100500, got %f", balance)
	}
}

func TestMultiAccountCapitalPool_RecordFillWrongAccount(t *testing.T) {
	m := NewMultiAccountCapitalPool()
	profile := propfirm.DefaultFTMOProfile()

	m.RegisterPool("acct-1", profile, &propfirm.State{
		StartingBalance: 100000.0, PeakBalance: 100000.0,
	})

	err := m.RecordFill("nonexistent", "s1", "SPY", "BUY", 500.0, 100.0)
	if err == nil {
		t.Error("expected error for nonexistent account")
	}
}

func TestMultiAccountCapitalPool_AggregateBalance(t *testing.T) {
	m := NewMultiAccountCapitalPool()
	profile := propfirm.DefaultFTMOProfile()

	m.RegisterPool("a1", profile, &propfirm.State{StartingBalance: 100000.0, PeakBalance: 100000.0})
	m.RegisterPool("a2", profile, &propfirm.State{StartingBalance: 200000.0, PeakBalance: 200000.0})
	m.RegisterPool("a3", profile, &propfirm.State{StartingBalance: 50000.0, PeakBalance: 50000.0})

	agg := m.AggregateBalance()
	if agg != 350000.0 {
		t.Errorf("expected aggregate 350000, got %f", agg)
	}
}

func TestMultiAccountCapitalPool_ResetDaily(t *testing.T) {
	m := NewMultiAccountCapitalPool()
	profile := propfirm.DefaultFTMOProfile()

	m.RegisterPool("a1", profile, &propfirm.State{StartingBalance: 100000.0, PeakBalance: 100000.0})
	m.RegisterPool("a2", profile, &propfirm.State{StartingBalance: 100000.0, PeakBalance: 100000.0})

	m.RequestCapital(context.Background(), "a1", CapitalRequest{StrategyID: "s1", Confidence: 0.5, Symbol: "SPY", Side: "BUY", BaseSize: 100000})
	m.RequestCapital(context.Background(), "a2", CapitalRequest{StrategyID: "s2", Confidence: 0.5, Symbol: "QQQ", Side: "BUY", BaseSize: 100000})

	m.RecordFill("a1", "s1", "SPY", "BUY", 300.0, 100.0)
	m.RecordFill("a2", "s2", "QQQ", "BUY", -200.0, 100.0)

	m.ResetAllDaily()

	metrics1, _ := m.StrategyMetrics("a1")
	for _, m := range metrics1 {
		if m.DailyPnL != 0 {
			t.Errorf("a1 strategy %s daily PnL should be 0 after reset, got %f", m.StrategyID, m.DailyPnL)
		}
	}

	metrics2, _ := m.StrategyMetrics("a2")
	for _, m := range metrics2 {
		if m.DailyPnL != 0 {
			t.Errorf("a2 strategy %s daily PnL should be 0 after reset, got %f", m.StrategyID, m.DailyPnL)
		}
	}
}

func TestMultiAccountCapitalPool_RegisterPoolDirect(t *testing.T) {
	m := NewMultiAccountCapitalPool()
	profile := propfirm.DefaultFTMOProfile()
	state := &propfirm.State{StartingBalance: 100000.0, PeakBalance: 100000.0}

	existing := NewCapitalPoolManager(profile, state)
	m.RegisterPoolDirect("direct-acct", existing)

	pool, err := m.GetPool("direct-acct")
	if err != nil {
		t.Fatalf("GetPool should succeed: %v", err)
	}
	if pool.AccountID() != "direct-acct" {
		t.Errorf("expected accountID direct-acct, got %s", pool.AccountID())
	}
}

func TestMultiAccountCapitalPool_AccountIDs(t *testing.T) {
	m := NewMultiAccountCapitalPool()
	profile := propfirm.DefaultFTMOProfile()

	m.RegisterPool("id-1", profile, &propfirm.State{StartingBalance: 100000.0})
	m.RegisterPool("id-2", profile, &propfirm.State{StartingBalance: 100000.0})
	m.RegisterPool("id-3", profile, &propfirm.State{StartingBalance: 100000.0})

	ids := m.AccountIDs()
	if len(ids) != 3 {
		t.Errorf("expected 3 account IDs, got %d", len(ids))
	}
}

func TestCapitalPoolManager_WithAccount(t *testing.T) {
	profile := propfirm.DefaultFTMOProfile()
	state := &propfirm.State{StartingBalance: 100000.0, PeakBalance: 100000.0}

	pool := NewCapitalPoolManagerWithAccount("my-account", profile, state)
	if pool.AccountID() != "my-account" {
		t.Errorf("expected accountID my-account, got %s", pool.AccountID())
	}

	originalPool := NewCapitalPoolManager(profile, state)
	if originalPool.AccountID() != "" {
		t.Errorf("original pool should have empty accountID, got %s", originalPool.AccountID())
	}
}





