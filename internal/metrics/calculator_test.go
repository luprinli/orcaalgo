package metrics

import (
	"math"
	"testing"
	"time"
)

func TestSharpe_ZeroReturns(t *testing.T) {
	c := NewCalculator(0.05)
	s := c.Sharpe([]float64{})
	if s != 0 {
		t.Errorf("expected 0, got %f", s)
	}
}

func TestSharpe_KnownValue(t *testing.T) {
	c := NewCalculator(0.05)
	returns := make([]float64, 252)
	for i := range returns {
		returns[i] = 0.001
	}
	s := c.Sharpe(returns)
	if s <= 0 {
		t.Errorf("expected positive sharpe, got %f", s)
	}
}

func TestSharpe_PositiveReturns(t *testing.T) {
	c := NewCalculator(0.0)
	returns := []float64{0.01, 0.02, 0.01, 0.02, 0.01}
	s := c.Sharpe(returns)
	if s <= 0 {
		t.Errorf("expected positive sharpe, got %f", s)
	}
}

func TestSortino_Default(t *testing.T) {
	c := NewCalculator(0.05)
	returns := []float64{0.01, -0.01, 0.02, -0.005, 0.01}
	s := c.Sortino(returns, 0)
	if s == 0 {
		t.Errorf("expected non-zero sortino")
	}
}

func TestSortino_AllPositive(t *testing.T) {
	c := NewCalculator(0.05)
	returns := []float64{0.01, 0.02, 0.015, 0.01, 0.02}
	s := c.Sortino(returns, 0)
	if s <= 0 {
		t.Errorf("expected positive sortino, got %f", s)
	}
	if s < 1e8 {
		t.Errorf("expected very large sortino for zero downside, got %f", s)
	}
}

func TestCAGR(t *testing.T) {
	c := NewCalculator(0.05)
	cagr := c.CAGR(100, 121, 252)
	if math.Abs(cagr-21.0) > 0.5 {
		t.Errorf("expected ~21%%, got %f", cagr)
	}
}

func TestCAGR_ZeroStart(t *testing.T) {
	c := NewCalculator(0.05)
	cagr := c.CAGR(0, 100, 252)
	if cagr != 0 {
		t.Errorf("expected 0 for zero start equity, got %f", cagr)
	}
}

func TestCAGR_ZeroDays(t *testing.T) {
	c := NewCalculator(0.05)
	cagr := c.CAGR(100, 200, 0)
	if cagr != 0 {
		t.Errorf("expected 0 for zero days, got %f", cagr)
	}
}

func TestMaxDrawdown_MonotonicRise(t *testing.T) {
	c := NewCalculator(0.05)
	equity := []float64{100, 101, 102, 103, 104}
	dd, _, _ := c.MaxDrawdown(equity)
	if dd != 0 {
		t.Errorf("expected 0 drawdown for rising equity, got %f", dd)
	}
}

func TestMaxDrawdown_SinglePeakTrough(t *testing.T) {
	c := NewCalculator(0.05)
	equity := []float64{100, 110, 90, 105, 95}
	dd, peakIdx, troughIdx := c.MaxDrawdown(equity)
	if dd >= 0 {
		t.Errorf("expected negative drawdown, got %f", dd)
	}
	if peakIdx != 1 {
		t.Errorf("expected peak at idx 1, got %d", peakIdx)
	}
	if troughIdx != 2 {
		t.Errorf("expected trough at idx 2, got %d", troughIdx)
	}
}

func TestMaxDrawdown_MultiPeak(t *testing.T) {
	c := NewCalculator(0.05)
	equity := []float64{100, 150, 120, 160, 80, 100}
	dd, peakIdx, troughIdx := c.MaxDrawdown(equity)
	if dd >= 0 {
		t.Errorf("expected negative drawdown, got %f", dd)
	}
	if peakIdx != 3 {
		t.Errorf("expected peak at idx 3, got %d", peakIdx)
	}
	if troughIdx != 4 {
		t.Errorf("expected trough at idx 4, got %d", troughIdx)
	}
}

func TestVaR_KnownDistribution(t *testing.T) {
	c := NewCalculator(0.05)
	returns := make([]float64, 1000)
	for i := range returns {
		returns[i] = 0.001
	}
	v := c.VaR(returns, 0.95)
	if v > 0 {
		t.Errorf("VaR should be negative (loss), got %f", v)
	}
}

func TestCVaR_GreaterThanVaR(t *testing.T) {
	c := NewCalculator(0.05)
	returns := make([]float64, 100)
	for i := range returns {
		if i%10 == 0 {
			returns[i] = -0.05
		} else {
			returns[i] = 0.01
		}
	}
	vr := c.VaR(returns, 0.95)
	cvr := c.CVaR(returns, 0.95)
	if cvr < vr {
		t.Errorf("CVaR (%f) should be >= VaR (%f)", cvr, vr)
	}
}

func TestProfitFactor_AllWins(t *testing.T) {
	c := NewCalculator(0.05)
	trades := []TradeSummary{
		{PnL: 100}, {PnL: 200}, {PnL: 50},
	}
	pf := c.ProfitFactor(trades)
	if pf <= 0 {
		t.Errorf("expected positive profit factor, got %f", pf)
	}
}

func TestProfitFactor_AllLosses(t *testing.T) {
	c := NewCalculator(0.05)
	trades := []TradeSummary{
		{PnL: -100}, {PnL: -50},
	}
	pf := c.ProfitFactor(trades)
	if pf != 0 {
		t.Errorf("expected 0 for all losses, got %f", pf)
	}
}

func TestProfitFactor_Mixed(t *testing.T) {
	c := NewCalculator(0.05)
	trades := []TradeSummary{
		{PnL: 150}, {PnL: -50}, {PnL: 100}, {PnL: -25},
	}
	pf := c.ProfitFactor(trades)
	if pf <= 0 {
		t.Errorf("expected positive profit factor, got %f", pf)
	}
	expected := 250.0 / 75.0
	if math.Abs(pf-expected) > 0.01 {
		t.Errorf("expected %f, got %f", expected, pf)
	}
}

func TestWinRate(t *testing.T) {
	c := NewCalculator(0.05)
	trades := []TradeSummary{
		{PnL: 100}, {PnL: -50}, {PnL: 50}, {PnL: -25},
	}
	wr := c.WinRate(trades)
	expected := 50.0
	if math.Abs(wr-expected) > 0.01 {
		t.Errorf("expected %f, got %f", expected, wr)
	}
}

func TestWinRate_Empty(t *testing.T) {
	c := NewCalculator(0.05)
	wr := c.WinRate(nil)
	if wr != 0 {
		t.Errorf("expected 0 for no trades, got %f", wr)
	}
}

func TestUlcerIndex_MonotonicRise(t *testing.T) {
	c := NewCalculator(0.05)
	equity := []float64{100, 101, 102, 103, 104}
	ui := c.UlcerIndex(equity)
	if ui != 0 {
		t.Errorf("expected 0 for rising equity, got %f", ui)
	}
}

func TestMonthlyReturns(t *testing.T) {
	c := NewCalculator(0.05)
	daily := []DailyReturn{
		{Date: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), ReturnPct: 1.0},
		{Date: time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC), ReturnPct: 2.0},
		{Date: time.Date(2024, 2, 10, 0, 0, 0, 0, time.UTC), ReturnPct: -0.5},
	}
	monthly := c.MonthlyReturns(daily)
	if len(monthly) != 2 {
		t.Errorf("expected 2 months, got %d", len(monthly))
	}
	jan := monthly[0]
	if jan.ReturnPct != 3.0 {
		t.Errorf("expected Jan return 3.0, got %f", jan.ReturnPct)
	}
}

func TestRollingSharpe(t *testing.T) {
	c := NewCalculator(0.05)
	returns := make([]float64, 50)
	for i := range returns {
		returns[i] = 0.005
	}
	rolling := c.RollingSharpe(returns, 20)
	if len(rolling) == 0 {
		t.Errorf("expected non-empty rolling sharpe, got %d", len(rolling))
	}
}

func TestRollingSharpe_WindowTooBig(t *testing.T) {
	c := NewCalculator(0.05)
	returns := []float64{0.01, 0.02, 0.01}
	rolling := c.RollingSharpe(returns, 10)
	if rolling != nil {
		t.Errorf("expected nil for window > returns, got %v", rolling)
	}
}

func TestComputeSnapshot_Empty(t *testing.T) {
	c := NewCalculator(0.05)
	snap := c.ComputeSnapshot(nil, nil)
	if snap.Equity != 0 {
		t.Errorf("expected zero equity for empty input")
	}
}

func TestComputeSnapshot_WithData(t *testing.T) {
	c := NewCalculator(0.05)
	equity := []MetricEquityPoint{
		{Timestamp: time.Now().UTC(), Equity: 100, Balance: 100, Drawdown: 0},
		{Timestamp: time.Now().UTC().Add(24 * time.Hour), Equity: 101, Balance: 101, Drawdown: 0},
		{Timestamp: time.Now().UTC().Add(48 * time.Hour), Equity: 102, Balance: 102, Drawdown: 0},
	}
	trades := []TradeSummary{
		{ID: "1", Symbol: "AAPL", Side: "buy", PnL: 10, PnLPct: 1.0},
		{ID: "2", Symbol: "AAPL", Side: "sell", PnL: -5, PnLPct: -0.5},
	}
	snap := c.ComputeSnapshot(equity, trades)
	if snap.Equity != 102 {
		t.Errorf("expected equity 102, got %f", snap.Equity)
	}
	if snap.NumTrades != 2 {
		t.Errorf("expected 2 trades, got %d", snap.NumTrades)
	}
}

func TestCalmar_ZeroMaxDD(t *testing.T) {
	c := NewCalculator(0.05)
	cal := c.Calmar(20.0, 0)
	if cal != 0 {
		t.Errorf("expected 0 for zero max DD, got %f", cal)
	}
}

func TestCalmar_WithDrawdown(t *testing.T) {
	c := NewCalculator(0.05)
	cal := c.Calmar(20.0, -10.0)
	if cal <= 0 {
		t.Errorf("expected positive calmar, got %f", cal)
	}
}

func TestCalculator_Constructor(t *testing.T) {
	c := NewCalculator(0.03)
	if c.riskFreeRate != 0.03 {
		t.Errorf("expected 0.03, got %f", c.riskFreeRate)
	}
	if c.tradingDays != 252 {
		t.Errorf("expected 252, got %d", c.tradingDays)
	}
}

func TestCalculator_ConstructorNegativeRate(t *testing.T) {
	c := NewCalculator(-0.01)
	if c.riskFreeRate != 0 {
		t.Errorf("expected 0, got %f", c.riskFreeRate)
	}
}

func TestSetTradingDays(t *testing.T) {
	c := NewCalculator(0.05)
	c.SetTradingDays(365)
	if c.tradingDays != 365 {
		t.Errorf("expected 365, got %d", c.tradingDays)
	}
}

func TestSetTradingDays_Negative(t *testing.T) {
	c := NewCalculator(0.05)
	c.SetTradingDays(-1)
	if c.tradingDays != 252 {
		t.Errorf("expected unchanged 252, got %d", c.tradingDays)
	}
}

func TestDailyReturnsFromEquity(t *testing.T) {
	c := NewCalculator(0.05)
	equity := []MetricEquityPoint{
		{Timestamp: time.Now().UTC(), Equity: 100, Balance: 100},
		{Timestamp: time.Now().UTC().Add(24 * time.Hour), Equity: 101, Balance: 101},
		{Timestamp: time.Now().UTC().Add(48 * time.Hour), Equity: 99, Balance: 99},
	}
	dr := c.DailyReturnsFromEquity(equity)
	if len(dr) != 2 {
		t.Errorf("expected 2 daily returns, got %d", len(dr))
	}
	if math.Abs(dr[0].ReturnPct-1.0) > 0.01 {
		t.Errorf("expected first return 1%%, got %f", dr[0].ReturnPct)
	}
	expectedSecond := -(2.0 / 101.0) * 100.0
	if math.Abs(dr[1].ReturnPct-expectedSecond) > 0.1 {
		t.Errorf("expected second return ~%f, got %f", expectedSecond, dr[1].ReturnPct)
	}
}

func TestVaR_ReturnsAsPct(t *testing.T) {
	c := NewCalculator(0.05)
	returns := make([]float64, 252)
	for i := range returns {
		returns[i] = 0.001
	}
	v := c.VaR(returns, 0.99)
	if v > 0 {
		t.Errorf("VaR should be negative, got %f", v)
	}
}
