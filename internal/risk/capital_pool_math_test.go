package risk

import "testing"

func TestCapitalPoolMath_UpdatePeakBalance(t *testing.T) {
	cpm := &CapitalPoolMath{TotalBalance: 100000, TotalPeakBalance: 100000}
	cpm.UpdatePeakBalance(105000)
	if cpm.TotalPeakBalance != 105000 {
		t.Errorf("peak should update to 105000, got %f", cpm.TotalPeakBalance)
	}
	cpm.UpdatePeakBalance(95000)
	if cpm.TotalPeakBalance != 105000 {
		t.Error("peak should not decrease")
	}
}

func TestCapitalPoolMath_DrawdownPct(t *testing.T) {
	cpm := &CapitalPoolMath{TotalPeakBalance: 100000}
	dd := cpm.DrawdownPct(95000)
	if dd < 4.9 || dd > 5.1 {
		t.Errorf("expected ~5%% DD, got %.2f", dd)
	}
}

func TestCapitalPoolMath_IsDailyLossExceeded(t *testing.T) {
	cpm := &CapitalPoolMath{StartingBalance: 100000}
	if !cpm.IsDailyLossExceeded(93500, 5.0) {
		t.Error("6.5%% loss should exceed 5%% limit")
	}
	if cpm.IsDailyLossExceeded(96000, 5.0) {
		t.Error("4%% loss should not exceed 5%% limit")
	}
}

func TestCapitalPoolMath_IsDrawdownExceeded(t *testing.T) {
	cpm := &CapitalPoolMath{TotalPeakBalance: 100000}
	if cpm.IsDrawdownExceeded(95000, 10.0) {
		t.Error("5%% DD should not exceed 10%% limit")
	}
	if !cpm.IsDrawdownExceeded(85000, 10.0) {
		t.Error("15%% DD should exceed 10%% limit")
	}
}
