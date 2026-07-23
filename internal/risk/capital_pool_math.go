package risk

import "github.com/lee-econ/orca-core/internal/propfirm"

type CapitalPoolMath struct {
	TotalBalance     float64
	TotalPeakBalance float64
	StartingBalance  float64
}

func (cpm *CapitalPoolMath) UpdatePeakBalance(balance float64) {
	if balance > cpm.TotalPeakBalance {
		cpm.TotalPeakBalance = balance
	}
}

func (cpm *CapitalPoolMath) DrawdownPct(balance float64) float64 {
	return propfirm.DrawdownPct(cpm.TotalPeakBalance, balance)
}

func (cpm *CapitalPoolMath) DailyLossPct(balance float64) float64 {
	return propfirm.DailyLossPct(cpm.StartingBalance, balance)
}

func (cpm *CapitalPoolMath) IsDailyLossExceeded(balance, maxDailyLossPct float64) bool {
	return propfirm.DailyLossExceeded(cpm.StartingBalance, balance, maxDailyLossPct)
}

func (cpm *CapitalPoolMath) IsDrawdownExceeded(balance, maxDrawdownPct float64) bool {
	return propfirm.DrawdownExceeded(cpm.TotalPeakBalance, balance, maxDrawdownPct)
}
