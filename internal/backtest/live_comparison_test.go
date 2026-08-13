package backtest

import (
	"math"
	"testing"
	"time"

	"github.com/lee-econ/orca-core/internal/types"
)

func p(v float64) types.Price { return types.PriceFromFloat(v) }

func TestComputeImpliedComparison_Slippage(t *testing.T) {
	matched := []MatchedLiveTrade{
		{Symbol: "SPY", Side: "BUY", EnginePrice: p(100), LivePrice: p(100.5), Quantity: 10},
		{Symbol: "QQQ", Side: "BUY", EnginePrice: p(200), LivePrice: p(200.8), Quantity: 10},
	}
	out := ComputeImpliedComparison(matched)
	// Weighted average signed slippage:
	// SPY: +0.5% = +50 bps; QQQ: +0.4% = +40 bps, equal weight -> 45 bps.
	if math.Abs(out.ImpliedSlippageBps-45.0) > 1e-6 {
		t.Errorf("ImpliedSlippageBps = %f, want ~45", out.ImpliedSlippageBps)
	}
	if out.MatchedCount != 2 {
		t.Errorf("MatchedCount = %d, want 2", out.MatchedCount)
	}
}

func TestComputeImpliedComparison_ShortFillBelowEngine(t *testing.T) {
	matched := []MatchedLiveTrade{
		{Symbol: "SPY", Side: "SELL", EnginePrice: p(100), LivePrice: p(99.0), Quantity: 10},
	}
	out := ComputeImpliedComparison(matched)
	// Signed slippage is (live-engine)/engine regardless of side: (99-100)/100
	// = -1% = -100 bps.
	if out.ImpliedSlippageBps > -90 || out.ImpliedSlippageBps < -110 {
		t.Errorf("ImpliedSlippageBps = %f, want ~-100", out.ImpliedSlippageBps)
	}
}

func TestComputeImpliedComparison_SkipsInvalid(t *testing.T) {
	matched := []MatchedLiveTrade{
		{Symbol: "BAD", Side: "BUY", EnginePrice: p(0), LivePrice: p(100), Quantity: 10},
		{Symbol: "SPY", Side: "BUY", EnginePrice: p(100), LivePrice: p(100), Quantity: 10},
	}
	out := ComputeImpliedComparison(matched)
	if out.MatchedCount != 1 {
		t.Errorf("MatchedCount = %d, want 1 (zero engine price skipped)", out.MatchedCount)
	}
}

func TestComputeImpliedComparison_Penetration(t *testing.T) {
	matched := []MatchedLiveTrade{
		{Symbol: "SPY", Side: "BUY", EnginePrice: p(100), LivePrice: p(99.5), Quantity: 10,
			LimitPrice: p(100), DayLow: p(99.0)},
	}
	out := ComputeImpliedComparison(matched)
	// (100 - 99) / 100 = 1% penetration.
	if math.Abs(out.PenetrationPct-1.0) > 1e-6 {
		t.Errorf("PenetrationPct = %f, want ~1.0", out.PenetrationPct)
	}
}

func TestComputeImpliedComparison_ExpenseGap(t *testing.T) {
	matched := []MatchedLiveTrade{
		{Symbol: "SPY", Side: "BUY", EnginePrice: p(100), LivePrice: p(100), Quantity: 10,
			LiveExpense: 0.0010, EngineExpense: 0.0008},
	}
	out := ComputeImpliedComparison(matched)
	if math.Abs(out.ExpenseGapBps-0.0002) > 1e-9 {
		t.Errorf("ExpenseGapBps = %f, want 0.0002", out.ExpenseGapBps)
	}
}

func TestMaxEquityDivergencePct(t *testing.T) {
	backtest := []float64{100, 101, 103, 102}
	live := []float64{100, 101.5, 103.3, 102.2}
	div := MaxEquityDivergencePct(backtest, live)
	// Largest gap: (101.5-101)/101 = 0.495%.
	if math.Abs(div-0.495) > 0.01 {
		t.Errorf("MaxEquityDivergencePct = %f, want ~0.495", div)
	}
	if MaxEquityDivergencePct(nil, nil) != 0 {
		t.Errorf("empty curves should return 0")
	}
}

func TestMatchedLiveTrade_HasDate(t *testing.T) {
	m := MatchedLiveTrade{Symbol: "SPY", Date: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)}
	if m.Date.IsZero() {
		t.Error("date should be preserved")
	}
}
