package propfirm

// PoolState holds the shared balance-tracking and halt-state fields used by
// both CapitalPoolSim (backtest) and CapitalPoolManager (live). Embedding this
// struct eliminates ~50 lines of duplicated field definitions and getter
// methods across the two packages.
type PoolState struct {
	TotalBalance     float64
	TotalPeakBalance float64
	DailyPnL         float64
	DailyPnLPct      float64
	Halted           bool
	HaltReason       string
	TradingDays      int
}

// InitPoolState sets starting values for a newly created pool.
func InitPoolState(s *PoolState, startingBalance float64) {
	s.TotalBalance = startingBalance
	s.TotalPeakBalance = startingBalance
	s.Halted = false
	s.HaltReason = ""
	s.TradingDays = 0
}

// CountTotalOpen returns the sum of all entries across OpenLong and OpenShort
// maps for the given slice of strategy statuses.
func CountTotalOpen(strats []struct{ OpenLong, OpenShort int }) int {
	total := 0
	for _, s := range strats {
		total += s.OpenLong + s.OpenShort
	}
	return total
}

// CheckDailyLossHalt checks whether daily loss has exceeded the profile limit
// and updates PoolState.Halted if so. Returns (halted, reason).
func CheckDailyLossHalt(s *PoolState, startingBalance float64, profile *Profile) (bool, string) {
	if profile == nil || s.Halted {
		return s.Halted, s.HaltReason
	}
	currentBalance := startingBalance + s.DailyPnL
	if DailyLossExceeded(startingBalance, currentBalance, profile.MaxDailyLossPct) {
		s.Halted = true
		s.HaltReason = "daily_loss_limit"
		return true, s.HaltReason
	}
	return false, ""
}

// CheckDrawdownHalt checks whether max drawdown has been exceeded and updates
// PoolState.Halted if so. Returns (halted, reason).
func CheckDrawdownHalt(s *PoolState, profile *Profile) (bool, string) {
	if profile == nil || s.Halted {
		return s.Halted, s.HaltReason
	}
	if profile.MaxDrawdownPct <= 0 {
		return false, ""
	}
	dd := DrawdownPct(s.TotalPeakBalance, s.TotalBalance)
	if dd > profile.MaxDrawdownPct {
		s.Halted = true
		s.HaltReason = "max_drawdown"
		return true, s.HaltReason
	}
	return false, ""
}

// UpdatePeakBalance ensures TotalPeakBalance is the high-water mark of
// TotalBalance, updating in place.
func UpdatePeakBalance(s *PoolState) {
	if s.TotalBalance > s.TotalPeakBalance {
		s.TotalPeakBalance = s.TotalBalance
	}
}

// RecordPoolPnL adds pnl to TotalBalance and DailyPnL, updates the peak
// balance high-water mark, and recomputes DailyPnLPct.
func RecordPoolPnL(s *PoolState, pnl float64, startingBalance float64) {
	s.TotalBalance += pnl
	s.DailyPnL += pnl
	UpdatePeakBalance(s)
	if startingBalance > 0 {
		s.DailyPnLPct = s.DailyPnL / startingBalance * 100.0
	}
}

// ResetPoolDaily zeroes DailyPnL and DailyPnLPct and increments TradingDays.
func ResetPoolDaily(s *PoolState) {
	s.DailyPnL = 0
	s.DailyPnLPct = 0
	s.TradingDays++
}

// PerStratDrawdownCheck returns true if the per-strategy drawdown (at 50% of
// the profile's max drawdown) has been exceeded.
func PerStratDrawdownCheck(stratPeakBalance, totalBalance float64, profile *Profile) bool {
	if profile == nil {
		return false
	}
	maxStratDD := profile.MaxDrawdownPct * 0.5
	if maxStratDD <= 0 {
		return false
	}
	stratDD := DrawdownPct(stratPeakBalance, totalBalance)
	return stratDD > maxStratDD
}

// CorrelationMultiplier returns 0.5 when opening a position on a symbol where
// the opposing side already has an open position, or 1.0 otherwise.
func CorrelationMultiplier(side, symbol string, openLong, openShort map[string]float64) float64 {
	if side == "BUY" && openShort[symbol] > 0 {
		return 0.5
	}
	if side == "SELL" && openLong[symbol] > 0 {
		return 0.5
	}
	return 1.0
}
