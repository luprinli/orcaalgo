package backtest

import (
	"github.com/lee-econ/orca-core/internal/propfirm"
	"time"
)

type PropFirmEnforcer struct {
	DailyLossLimitPct     float64
	MaxDrawdownPct        float64
	MaxPositionPct        float64
	ConsistencyThreshold  float64
	ConsistencySizeMult   float64
	StartingBalance       float64
	CurrentBalance        float64
	PeakBalance           float64
	DailyPnL              float64
	DailyPnLPct           float64
	CumulativePnL         float64
	IsHalted              bool
	HaltReason            string
	ConsistencyMultiplier float64
	CurrentRegime         int8
	RegimeSizeMultipliers [4]float64
	DailyBreaches         []RuleBreach
	ProfitTargetPct       float64
	MinTradingDays        int
	TradingDays           int
	ProfitTargetMet       bool
	NewsTradingRestricted bool
	CurrentPhase          int
}

type RuleBreach struct {
	Date   time.Time
	Code   string
	Value  float64
	Limit  float64
	Action string
}

func NewPropFirmEnforcerFromProfile(p *propfirm.Profile, startingBalance float64) *PropFirmEnforcer {
	if p == nil {
		p = propfirm.DefaultProfile()
	}
	return &PropFirmEnforcer{
		DailyLossLimitPct:     p.MaxDailyLossPct,
		MaxDrawdownPct:        p.MaxDrawdownPct,
		MaxPositionPct:        p.MaxPositionPct,
		ConsistencyThreshold:  p.ConsistencyThresholdPct,
		ConsistencySizeMult:   p.ConsistencyPenalty,
		StartingBalance:       startingBalance,
		CurrentBalance:        startingBalance,
		PeakBalance:           startingBalance,
		ConsistencyMultiplier: 1.0,
		RegimeSizeMultipliers: p.RegimeMultipliers,
		ProfitTargetPct:       p.ProfitTargetPctPhase1,
		MinTradingDays:        p.MinTradingDays,
		NewsTradingRestricted: !p.NewsTradingAllowed,
		CurrentPhase:          1,
	}
}

func DefaultPropFirmEnforcer(startingBalance float64) *PropFirmEnforcer {
	return NewPropFirmEnforcerFromProfile(propfirm.DefaultProfile(), startingBalance)
}

func (f *PropFirmEnforcer) CheckDailyLoss() bool {
	if f.StartingBalance <= 0 {
		return true
	}
	dailyChange := propfirm.DailyLossPct(f.StartingBalance, f.CurrentBalance)
	f.DailyPnLPct = dailyChange
	if propfirm.DailyLossExceeded(f.StartingBalance, f.CurrentBalance, f.DailyLossLimitPct) {
		f.IsHalted = true
		f.HaltReason = "daily_loss_limit"
		f.DailyBreaches = append(f.DailyBreaches, RuleBreach{
			Date:   time.Now(),
			Code:   "DAILY_DD",
			Value:  dailyChange,
			Limit:  -f.DailyLossLimitPct,
			Action: "halted",
		})
		return false
	}
	return true
}

func (f *PropFirmEnforcer) CheckDrawdown() bool {
	if f.CurrentBalance > f.PeakBalance {
		f.PeakBalance = f.CurrentBalance
	}
	if f.PeakBalance <= 0 {
		return true
	}
	if propfirm.DrawdownExceeded(f.PeakBalance, f.CurrentBalance, f.MaxDrawdownPct) {
		f.IsHalted = true
		f.HaltReason = "max_drawdown"
		return false
	}
	return true
}

func (f *PropFirmEnforcer) CheckConsistency() bool {
	if propfirm.ConsistencyBreached(f.DailyPnLPct, f.ConsistencyThreshold) {
		f.ConsistencyMultiplier = f.ConsistencySizeMult
		f.DailyBreaches = append(f.DailyBreaches, RuleBreach{
			Date:   time.Now(),
			Code:   "CONSISTENCY",
			Value:  f.DailyPnLPct,
			Limit:  f.ConsistencyThreshold,
			Action: "size_reduced",
		})
		return false
	}
	return true
}

func (f *PropFirmEnforcer) GetRegimeMultiplier() float64 {
	r := f.CurrentRegime
	if r < 0 || r > 3 {
		return 1.0
	}
	return f.RegimeSizeMultipliers[r] * f.ConsistencyMultiplier
}

func (f *PropFirmEnforcer) OnFill(pnl float64) {
	f.CurrentBalance += pnl
	f.DailyPnL += pnl
	f.CumulativePnL += pnl
	f.CheckDailyLoss()
	f.CheckDrawdown()
	f.CheckConsistency()
	f.CheckProfitTarget()
}

func (f *PropFirmEnforcer) OnNewDay() {
	f.TradingDays++
	f.CheckProfitTarget()
	f.DailyPnL = 0
	f.DailyPnLPct = 0
	f.ConsistencyMultiplier = 1.0
}

func (f *PropFirmEnforcer) GetPositionSize(baseQuantity float64) float64 {
	if f.CurrentBalance <= 0 {
		return 0
	}
	maxSize := f.CurrentBalance * f.MaxPositionPct / 100.0
	mult := f.GetRegimeMultiplier()
	adjusted := baseQuantity * mult
	if adjusted > maxSize {
		return maxSize
	}
	return adjusted
}

func (f *PropFirmEnforcer) CheckProfitTarget() bool {
	if f.StartingBalance <= 0 {
		return false
	}
	totalReturn := f.CumulativePnL / f.StartingBalance * 100.0
	if totalReturn >= f.ProfitTargetPct && f.TradingDays >= f.MinTradingDays {
		f.ProfitTargetMet = true
		return true
	}
	return false
}

func (f *PropFirmEnforcer) CheckNewsTrading(ts time.Time) bool {
	if !f.NewsTradingRestricted {
		return true
	}
	hour := ts.Hour()
	minute := ts.Minute()
	if hour == 8 && minute >= 30 && minute <= 35 {
		return false
	}
	if hour == 10 && minute >= 0 && minute <= 5 {
		return false
	}
	if hour == 14 && minute >= 0 && minute <= 5 {
		return false
	}
	return true
}

func (f *PropFirmEnforcer) AdvancePhase() {
	f.CurrentPhase++
	f.ProfitTargetMet = false
	f.DailyPnL = 0
	f.DailyPnLPct = 0
	f.CumulativePnL = 0
	f.PeakBalance = f.CurrentBalance
	f.TradingDays = 0
	f.ConsistencyMultiplier = 1.0
}

func (f *PropFirmEnforcer) TotalReturnPct() float64 {
	if f.StartingBalance <= 0 {
		return 0
	}
	return f.CumulativePnL / f.StartingBalance * 100.0
}

func (f *PropFirmEnforcer) Summary() ComplianceReport {
	return ComplianceReport{
		Passed:            !f.IsHalted,
		HaltReason:        f.HaltReason,
		FinalBalance:      f.CurrentBalance,
		PeakBalance:       f.PeakBalance,
		MaxDailyLossPct:   f.maxDailyLossObserved(),
		NumBreaches:       len(f.DailyBreaches),
		Breaches:          f.DailyBreaches,
		TotalReturnPct:    f.TotalReturnPct(),
		ProfitTargetMet:   f.ProfitTargetMet,
		TradingDays:       f.TradingDays,
		MinTradingDays:    f.MinTradingDays,
		CurrentPhase:      f.CurrentPhase,
	}
}

type ComplianceReport struct {
	Passed          bool
	HaltReason      string
	FinalBalance    float64
	PeakBalance     float64
	MaxDailyLossPct float64
	NumBreaches     int
	Breaches        []RuleBreach
	TotalReturnPct  float64
	ProfitTargetMet bool
	TradingDays     int
	MinTradingDays  int
	CurrentPhase    int
}

func (f *PropFirmEnforcer) maxDailyLossObserved() float64 {
	worst := 0.0
	for _, b := range f.DailyBreaches {
		if b.Code == "DAILY_DD" && b.Value < worst {
			worst = b.Value
		}
	}
	return worst
}

// --- PropFirmGate adapter methods ---

// CheckDailyLimits combines daily-loss and drawdown checks into a single
// (ok, reason) return for the PropFirmGate interface.
func (f *PropFirmEnforcer) CheckDailyLimits() (bool, string) {
	if f.CheckDailyLoss() {
		return false, "daily_loss"
	}
	if f.CheckDrawdown() {
		return false, "max_drawdown"
	}
	return true, ""
}

// OnFillAdapter calls the existing OnFill with just pnl. The balance parameter
// is ignored because PropFirmEnforcer tracks CurrentBalance internally.
func (f *PropFirmEnforcer) OnFillAdapter(pnl, balance float64) {
	_ = balance
	f.OnFill(pnl)
}

// MarkViolated sets the IsHalted flag and records the reason.
func (f *PropFirmEnforcer) MarkViolated(reason string) {
	f.IsHalted = true
	f.HaltReason = reason
}
