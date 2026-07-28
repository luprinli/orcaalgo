package backtest

import "testing"

func TestPropFirmEnforcerDailyLoss(t *testing.T) {
	f := DefaultPropFirmEnforcer(100000.0)

	if !f.CheckDailyLoss() {
		t.Error("should pass with no loss")
	}

	f.CurrentBalance = 94000
	if f.CheckDailyLoss() {
		t.Error("should fail at 6% daily loss")
	}
	if !f.IsHalted() {
		t.Error("should be halted after daily loss breach")
	}
	if f.HaltReason() != "daily_loss_limit" {
		t.Errorf("expected halt reason daily_loss_limit, got %s", f.HaltReason())
	}
}

func TestPropFirmEnforcerDrawdown(t *testing.T) {
	f := DefaultPropFirmEnforcer(100000.0)

	if !f.CheckDrawdown() {
		t.Error("should pass at starting balance")
	}

	f.CurrentBalance = 89000
	if f.CheckDrawdown() {
		t.Error("should fail at 11% drawdown")
	}
	if !f.IsHalted() {
		t.Error("should be halted after drawdown breach")
	}
	if f.HaltReason() != "max_drawdown" {
		t.Errorf("expected halt reason max_drawdown, got %s", f.HaltReason())
	}
}

func TestPropFirmEnforcerDrawdownHWM(t *testing.T) {
	f := DefaultPropFirmEnforcer(100000.0)
	f.CurrentBalance = 110000
	f.CheckDrawdown()
	if f.PeakBalance != 110000 {
		t.Errorf("peak should be 110000, got %.0f", f.PeakBalance)
	}

	f.CurrentBalance = 99000
	if !f.CheckDrawdown() {
		t.Error("10% DD from 110k should still pass")
	}
}

func TestPropFirmEnforcerConsistency(t *testing.T) {
	f := DefaultPropFirmEnforcer(100000.0)

	f.DailyPnLPct = 35.0
	f.CheckConsistency()
	if f.ConsistencyMultiplier != 0.5 {
		t.Errorf("expected consistency multiplier 0.5, got %.2f", f.ConsistencyMultiplier)
	}
	if len(f.DailyBreaches) != 1 {
		t.Errorf("expected 1 consistency breach, got %d", len(f.DailyBreaches))
	}

	f.DailyPnLPct = 10.0
	f.CheckConsistency()
	if len(f.DailyBreaches) != 1 {
		t.Error("should not add breach for normal day")
	}
}

func TestPropFirmEnforcerRegimeMultipliers(t *testing.T) {
	f := DefaultPropFirmEnforcer(100000.0)

	f.CurrentRegime = 0
	if f.GetRegimeMultiplier() != 1.0 {
		t.Errorf("CALM should be 1.0x, got %.2f", f.GetRegimeMultiplier())
	}

	f.CurrentRegime = 3
	if f.GetRegimeMultiplier() != 0.5 {
		t.Errorf("CRISIS should be 0.5x, got %.2f", f.GetRegimeMultiplier())
	}
}

func TestPropFirmEnforcerPositionSize(t *testing.T) {
	f := DefaultPropFirmEnforcer(100000.0)

	size := f.GetPositionSize(200)
	if size != 200 {
		t.Errorf("expected 200 at 1.0x, got %.0f", size)
	}

	f.ConsistencyMultiplier = 0.5
	size = f.GetPositionSize(200)
	if size != 100 {
		t.Errorf("expected 100 at 0.5x consistency, got %.0f", size)
	}

	f.CurrentRegime = 3
	size = f.GetPositionSize(200)
	if size != 50 {
		t.Errorf("expected 50 at CRISIS+consistency, got %.0f", size)
	}
}

func TestPropFirmEnforcerSummary(t *testing.T) {
	f := DefaultPropFirmEnforcer(100000.0)
	summary := f.Summary()

	if !summary.Passed {
		t.Error("should pass by default")
	}
	if summary.FinalBalance != 100000 {
		t.Errorf("expected 100000, got %.0f", summary.FinalBalance)
	}
}

func TestPropFirmEnforcerDailyReset(t *testing.T) {
	f := DefaultPropFirmEnforcer(100000.0)
	f.OnNewDay()
	if f.DailyPnL != 0 {
		t.Errorf("DailyPnL should be 0 after reset, got %f", f.DailyPnL)
	}
	if f.DailyPnLPct != 0 {
		t.Errorf("DailyPnLPct should be 0 after reset, got %f", f.DailyPnLPct)
	}
}
