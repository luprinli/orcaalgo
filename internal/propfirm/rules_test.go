package propfirm

import "testing"

func TestDrawdownPct(t *testing.T) {
	if dd := DrawdownPct(100000, 95000); dd < 4.9 || dd > 5.1 {
		t.Errorf("expected ~5.0%%, got %.2f", dd)
	}
	if dd := DrawdownPct(100000, 100000); dd != 0 {
		t.Errorf("expected 0%%, got %.2f", dd)
	}
	if dd := DrawdownPct(0, 0); dd != 0 {
		t.Errorf("expected 0%% for zero balance, got %.2f", dd)
	}
}

func TestDailyLossPct(t *testing.T) {
	if loss := DailyLossPct(100000, 95000); loss > -4.9 || loss < -5.1 {
		t.Errorf("expected ~-5.0%%, got %.2f", loss)
	}
	if loss := DailyLossPct(100000, 102000); loss < 1.9 || loss > 2.1 {
		t.Errorf("expected ~+2.0%%, got %.2f", loss)
	}
	if loss := DailyLossPct(0, 100); loss != 0 {
		t.Errorf("expected 0%% for zero starting balance, got %.2f", loss)
	}
}

func TestDailyLossExceeded(t *testing.T) {
	if !DailyLossExceeded(100000, 94000, 5.0) {
		t.Error("-6%% loss should exceed 5%% limit")
	}
	if DailyLossExceeded(100000, 96000, 5.0) {
		t.Error("-4%% loss should not exceed 5%% limit")
	}
	if DailyLossExceeded(100000, 100000, 5.0) {
		t.Error("no loss should not exceed limit")
	}
}

func TestDrawdownExceeded(t *testing.T) {
	if DrawdownExceeded(100000, 95000, 10.0) {
		t.Error("5%% DD should not exceed 10%% limit")
	}
	if !DrawdownExceeded(100000, 85000, 10.0) {
		t.Error("15%% DD should exceed 10%% limit")
	}
	if DrawdownExceeded(100000, 100000, 10.0) {
		t.Error("no DD should not exceed limit")
	}
}

func TestConsistencyBreached(t *testing.T) {
	if ConsistencyBreached(5.0, 10.0) {
		t.Error("5%% day should not breach 10%% threshold")
	}
	if !ConsistencyBreached(12.0, 10.0) {
		t.Error("12%% day should breach 10%% threshold")
	}
	if ConsistencyBreached(10.0, 10.0) {
		t.Error("exact match should not breach (strict >)")
	}
}
