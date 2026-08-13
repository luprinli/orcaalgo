package notify

import (
	"math"
	"testing"

	"github.com/lee-econ/orca-core/internal/types"
)

func p(v float64) types.Price { return types.PriceFromFloat(v) }

func TestNormalCDF_Basic(t *testing.T) {
	// Phi(0) = 0.5, Phi(-1) ~ 0.1587, Phi(1) ~ 0.8413.
	if v := normalCDF(0); math.Abs(v-0.5) > 1e-9 {
		t.Errorf("normalCDF(0) = %f, want 0.5", v)
	}
	if v := normalCDF(1); math.Abs(v-0.8413447) > 1e-4 {
		t.Errorf("normalCDF(1) = %f, want ~0.8413", v)
	}
	if v := normalCDF(-1); math.Abs(v-0.1586553) > 1e-4 {
		t.Errorf("normalCDF(-1) = %f, want ~0.1587", v)
	}
}

func TestLimitFillProbability_AtMarketAlwaysFills(t *testing.T) {
	if p := LimitFillProbability("BUY", 0, 0.02); p != 1.0 {
		t.Errorf("zero distance should fill with probability 1, got %f", p)
	}
}

func TestLimitFillProbability_Bounded(t *testing.T) {
	for _, dist := range []float64{0.001, 0.01, 0.05, 0.20, 0.50} {
		p := LimitFillProbability("BUY", dist, 0.02)
		if p < 0 || p > 1 {
			t.Errorf("probability out of range: %f", p)
		}
	}
}

func TestLimitFillProbability_FartherLessLikely(t *testing.T) {
	near := LimitFillProbability("BUY", 0.01, 0.02)
	far := LimitFillProbability("BUY", 0.20, 0.02)
	if far >= near {
		t.Errorf("farther limit should be less likely to fill: near=%f far=%f", near, far)
	}
}

func TestEstimateSigmaRolling_Fallback(t *testing.T) {
	if s := EstimateSigmaRolling(nil, 20); s != DefaultSigmaFallback {
		t.Errorf("empty series should use fallback, got %f", s)
	}
	if s := EstimateSigmaRolling([]float64{100.0}, 20); s != DefaultSigmaFallback {
		t.Errorf("single-close series should use fallback, got %f", s)
	}
}

func TestEstimateSigmaRolling_ConstantSeries(t *testing.T) {
	closes := make([]float64, 30)
	for i := range closes {
		closes[i] = 100.0
	}
	// Zero variance falls back to the fallback sigma (floor), not NaN.
	s := EstimateSigmaRolling(closes, 20)
	if math.IsNaN(s) || s <= 0 {
		t.Errorf("constant series should fall back to positive sigma, got %f", s)
	}
}

func TestCalculateCashImpact_Directions(t *testing.T) {
	ops := []DispatchSummaryOperation{
		{Ticker: "SPY", OperationType: "open_position", Quantity: 10, Price: p(100), OrderType: "market"},
		{Ticker: "SPY", OperationType: "close_position", Quantity: 10, Price: p(110), OrderType: "market"},
	}
	out := CalculateCashImpact(ops, nil, nil, 20)
	// -1000 (buy) + 1100 (sell) = +100.
	if math.Abs(out.TotalImpact-100.0) > 1e-9 {
		t.Errorf("TotalImpact = %f, want 100", out.TotalImpact)
	}
	if out.Eligible != 2 {
		t.Errorf("Eligible = %d, want 2", out.Eligible)
	}
}

func TestCalculateCashImpact_MissingPricing(t *testing.T) {
	ops := []DispatchSummaryOperation{
		{Ticker: "SPY", OperationType: "open_position", Quantity: 0, Price: p(100), OrderType: "market"},
	}
	out := CalculateCashImpact(ops, nil, nil, 20)
	if out.MissingPricing != 1 {
		t.Errorf("MissingPricing = %d, want 1", out.MissingPricing)
	}
}

func TestCalculateCashImpact_LimitWeighted(t *testing.T) {
	closes := make([]float64, 30)
	base := 100.0
	for i := range closes {
		// Gently trending series so sigma is non-trivial.
		closes[i] = base * (1 + 0.001*float64(i))
	}
	ops := []DispatchSummaryOperation{
		{Ticker: "SPY", OperationType: "open_position", Quantity: 10, Price: p(100), OrderType: "limit"},
	}
	out := CalculateCashImpact(ops, map[string][]float64{"SPY": closes}, map[string]float64{"SPY": 0.01}, 20)
	// Limit buy at 1% below reference: expected fill < full notional.
	if out.EstimatedImpact >= 0 {
		t.Errorf("buy estimated impact should be negative, got %f", out.EstimatedImpact)
	}
	if out.EstimatedImpact <= -1000 {
		t.Errorf("limit buy should not exceed full notional, got %f", out.EstimatedImpact)
	}
	if out.LimitAdjusted != 1 {
		t.Errorf("LimitAdjusted = %d, want 1", out.LimitAdjusted)
	}
}

func TestCalculateOrderSizeStats(t *testing.T) {
	ops := []DispatchSummaryOperation{
		{Ticker: "A", Quantity: 10, Price: p(100)},
		{Ticker: "B", Quantity: 5, Price: p(200)},
		{Ticker: "C", Quantity: 0, Price: p(500)},
	}
	s := CalculateOrderSizeStats(ops)
	if s.Min != 1000 || s.Max != 1000 || s.Avg != 1000 {
		t.Errorf("stats wrong: min=%f max=%f avg=%f", s.Min, s.Max, s.Avg)
	}
	empty := CalculateOrderSizeStats(nil)
	if empty.Min != 0 || empty.Avg != 0 || empty.Max != 0 {
		t.Errorf("empty stats should be zero-valued, got %+v", empty)
	}
}
