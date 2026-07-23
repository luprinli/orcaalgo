package propfirm

import (
	"testing"
)

func TestDefaultFTMOProfile(t *testing.T) {
	p := DefaultFTMOProfile()

	if p.ID != "ftmo" {
		t.Errorf("Expected 'ftmo', got '%s'", p.ID)
	}
	if p.MaxDailyLossPct != 5.0 {
		t.Errorf("Expected 5.0, got %f", p.MaxDailyLossPct)
	}
	if p.MaxDrawdownPct != 10.0 {
		t.Errorf("Expected 10.0, got %f", p.MaxDrawdownPct)
	}
	if p.MaxPositionPct != 2.0 {
		t.Errorf("Expected 2.0, got %f", p.MaxPositionPct)
	}
	if p.RegimeMultipliers[0] != 1.0 || p.RegimeMultipliers[3] != 0.5 {
		t.Errorf("Expected regime multipliers [1.0, 0.85, 0.75, 0.5], got %v", p.RegimeMultipliers)
	}
}

func TestManager_ActivateProfile(t *testing.T) {
	mgr := NewManager()
	mgr.profiles["ftmo"] = DefaultFTMOProfile()

	if err := mgr.ActivateProfile("ftmo"); err != nil {
		t.Fatalf("Failed to activate FTMO: %v", err)
	}

	active := mgr.ActiveProfile()
	if active == nil {
		t.Fatal("Active profile is nil")
	}
	if active.ID != "ftmo" {
		t.Errorf("Expected 'ftmo', got '%s'", active.ID)
	}

	state := mgr.State()
	if state.ProfileID != "ftmo" {
		t.Errorf("Expected state profile 'ftmo', got '%s'", state.ProfileID)
	}
	if state.StartingBalance != 100000.0 {
		t.Errorf("Expected 100000, got %f", state.StartingBalance)
	}
}

func TestManager_ActivateMissingProfile(t *testing.T) {
	mgr := NewManager()
	err := mgr.ActivateProfile("nonexistent")
	if err == nil {
		t.Error("Expected error for missing profile")
	}
}

func TestManager_RecordFill(t *testing.T) {
	mgr := NewManager()
	mgr.profiles["ftmo"] = DefaultFTMOProfile()
	mgr.ActivateProfile("ftmo")

	mgr.RecordFill(500, 100500)

	state := mgr.State()
	if state.DailyPnL != 500 {
		t.Errorf("Expected 500, got %f", state.DailyPnL)
	}
}

func TestManager_CheckDailyLimits(t *testing.T) {
	mgr := NewManager()
	mgr.profiles["ftmo"] = DefaultFTMOProfile()
	mgr.ActivateProfile("ftmo")

	ok, reason := mgr.CheckDailyLimits()
	if !ok {
		t.Errorf("Expected limits to pass, got reason: %s", reason)
	}

	mgr.RecordFill(-10000, 90000)
	ok, reason = mgr.CheckDailyLimits()
	if ok {
		t.Error("Expected limits to fail with large loss")
	} else {
		t.Logf("Limit check failed correctly: %s", reason)
	}
}

func TestManager_DailyReset(t *testing.T) {
	mgr := NewManager()
	mgr.profiles["ftmo"] = DefaultFTMOProfile()
	mgr.ActivateProfile("ftmo")

	mgr.RecordFill(500, 100500)
	mgr.DailyReset()

	state := mgr.State()
	if state.DailyPnL != 0 {
		t.Errorf("Expected 0, got %f", state.DailyPnL)
	}
	if state.TradingDays != 1 {
		t.Errorf("Expected 1, got %d", state.TradingDays)
	}
}

func TestManager_AdvancePhase(t *testing.T) {
	mgr := NewManager()
	mgr.profiles["ftmo"] = DefaultFTMOProfile()
	mgr.ActivateProfile("ftmo")

	if mgr.State().CurrentPhase != 1 {
		t.Errorf("Expected phase 1, got %d", mgr.State().CurrentPhase)
	}

	mgr.AdvancePhase()

	if mgr.State().CurrentPhase != 2 {
		t.Errorf("Expected phase 2, got %d", mgr.State().CurrentPhase)
	}
}

func TestManager_MarkViolated(t *testing.T) {
	mgr := NewManager()
	mgr.profiles["ftmo"] = DefaultFTMOProfile()
	mgr.ActivateProfile("ftmo")

	mgr.MarkViolated("daily_loss_limit")

	if !mgr.IsHalted() {
		t.Error("Expected halted after mark violated")
	}
	state := mgr.State()
	if !state.Violated {
		t.Error("Expected violated state")
	}
	if state.ViolationReason != "daily_loss_limit" {
		t.Errorf("Expected 'daily_loss_limit', got '%s'", state.ViolationReason)
	}
}

func TestManager_IsHalted(t *testing.T) {
	mgr := NewManager()
	if mgr.IsHalted() {
		t.Error("Expected not halted initially")
	}

	mgr.MarkViolated("max_drawdown")
	if !mgr.IsHalted() {
		t.Error("Expected halted after violation")
	}
}

func TestManager_AllProfiles(t *testing.T) {
	mgr := NewManager()
	mgr.profiles["ftmo"] = DefaultFTMOProfile()
	mgr.profiles["topstep"] = &Profile{ID: "topstep", Name: "TopStep", MaxDailyLossPct: 2.0}

	all := mgr.AllProfiles()
	if len(all) < 2 {
		t.Errorf("Expected at least 2 profiles, got %d", len(all))
	}
}
