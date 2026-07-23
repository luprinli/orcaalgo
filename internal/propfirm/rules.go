package propfirm

// Canonical prop-firm rule math — the SINGLE source of truth shared by the live
// Manager (this package), the backtest PropFirmEnforcer, and the risk package's
// formula helpers (which delegate here). It lives in package propfirm because both
// risk and backtest import propfirm while propfirm imports neither, so this avoids
// the import cycle that originally forced the rules to be reimplemented in parallel
// (docs/backtest_live_parity_audit_report.md R2 / RC-1).
//
// These functions are pure (no state, no clock) and encode the exact formulas that
// previously lived inline in Manager.CheckDailyLimits and risk/formulas.go, so
// swapping callers onto them is behavior-preserving.

// DrawdownPct returns peak-to-current drawdown as a percentage of the peak balance.
// A non-positive peak yields 0 (drawdown undefined without a positive peak).
func DrawdownPct(peakBalance, currentBalance float64) float64 {
	if peakBalance <= 0 {
		return 0
	}
	return (peakBalance - currentBalance) / peakBalance * 100
}

// DailyLossPct returns the change from starting balance as a percentage
// (negative = loss). A non-positive starting balance yields 0.
func DailyLossPct(startingBalance, currentBalance float64) float64 {
	if startingBalance <= 0 {
		return 0
	}
	return (currentBalance - startingBalance) / startingBalance * 100
}

// DailyLossExceeded reports whether the loss from the starting balance breaches the
// max daily-loss limit (maxDailyLossPct is a positive percentage, e.g. 5.0 for 5%).
// Uses <= to match the historical risk.IsDailyLossExceeded semantics.
func DailyLossExceeded(startingBalance, currentBalance, maxDailyLossPct float64) bool {
	return DailyLossPct(startingBalance, currentBalance) <= -maxDailyLossPct
}

// DrawdownExceeded reports whether peak-to-current drawdown breaches the max
// drawdown limit (maxDrawdownPct is a positive percentage, e.g. 10.0 for 10%).
func DrawdownExceeded(peakBalance, currentBalance, maxDrawdownPct float64) bool {
	return DrawdownPct(peakBalance, currentBalance) > maxDrawdownPct
}

// ConsistencyBreached reports whether a single day's P&L percentage exceeds the
// consistency threshold (profit-concentration rule). Callers gate this on whether
// the consistency rule is enabled for the active profile.
func ConsistencyBreached(dailyPnLPct, thresholdPct float64) bool {
	return dailyPnLPct > thresholdPct
}
