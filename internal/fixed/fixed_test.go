package fixed

import (
	"testing"
)

func TestPriceFromFloat64(t *testing.T) {
	tests := []struct {
		input    float64
		expected int64
	}{
		{452.30123, 45_230_123},
		{1.0, 100_000},
		{0.0, 0},
		{0.00001, 1},
		{-50.25, -5_025_000},
		{100.999999, 10_100_000},
	}
	for _, tt := range tests {
		p := FromFloat64(tt.input)
		if int64(p) != tt.expected {
			t.Errorf("FromFloat64(%f): expected %d, got %d", tt.input, tt.expected, int64(p))
		}
	}
}

func TestPriceToFloat64(t *testing.T) {
	p := Price(45_230_123)
	if got := p.ToFloat64(); got != 452.30123 {
		t.Errorf("ToFloat64: expected 452.30123, got %f", got)
	}
}

func TestPriceAdd(t *testing.T) {
	a := Price(1_000_000)
	b := Price(2_000_000)
	if got := a.Add(b); got != Price(3_000_000) {
		t.Errorf("Add: expected 3000000, got %d", got)
	}
}

func TestPriceSub(t *testing.T) {
	a := Price(3_000_000)
	b := Price(1_000_000)
	if got := a.Sub(b); got != Price(2_000_000) {
		t.Errorf("Sub: expected 2000000, got %d", got)
	}
}

func TestPriceMulFloat(t *testing.T) {
	p := Price(10_000_000)
	got := p.MulFloat(0.25)
	if int64(got) != 2_500_000 {
		t.Errorf("MulFloat(0.25): expected 2500000, got %d", got)
	}
}

func TestPriceRatio(t *testing.T) {
	a := Price(4_000_000)
	b := Price(2_000_000)
	if got := a.Ratio(b); got != 2.0 {
		t.Errorf("Ratio: expected 2.0, got %f", got)
	}
}

func TestPriceComparisons(t *testing.T) {
	a := Price(1_000_000)
	b := Price(2_000_000)
	if !a.Lt(b) {
		t.Error("Lt: expected true")
	}
	if a.Gt(b) {
		t.Error("Gt: expected false")
	}
	if !b.Gte(a) {
		t.Error("Gte: expected true")
	}
}

func TestMaxMin(t *testing.T) {
	a := Price(1_000_000)
	b := Price(2_000_000)
	if Max(a, b) != b {
		t.Error("Max: expected b")
	}
	if Min(a, b) != a {
		t.Error("Min: expected a")
	}
}

func TestQtyMulFloat(t *testing.T) {
	q := QtyFromFloat64(100.0)
	got := q.MulFloat(0.25)
	if got.ToFloat64() != 25.0 {
		t.Errorf("Qty*0.25: expected 25.0, got %f", got.ToFloat64())
	}
}

func TestPriceZero(t *testing.T) {
	if !New(0).IsZero() {
		t.Error("zero price should report IsZero=true")
	}
	if New(1).IsZero() {
		t.Error("non-zero price should report IsZero=false")
	}
}
