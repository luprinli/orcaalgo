package types

import (
	"encoding/json"
	"testing"
)

func TestPriceFromFloat(t *testing.T) {
	p := PriceFromFloat(1.50150)
	if p.Int64() != 150150 {
		t.Errorf("expected 150150, got %d", p.Int64())
	}
	if p.Float64() != 1.5015 {
		t.Errorf("expected 1.5015, got %f", p.Float64())
	}
}

func TestPriceRoundTrip(t *testing.T) {
	tests := []float64{0.01, 1.0, 100.50, 1500.25, 25000.75}
	for _, f := range tests {
		p := PriceFromFloat(f)
		back := p.Float64()
		if back != f {
			t.Errorf("round-trip failed: %f -> %d -> %f", f, p.Int64(), back)
		}
	}
}

func TestPriceIsZero(t *testing.T) {
	if !Price(0).IsZero() {
		t.Error("Price(0) should be zero")
	}
	if PriceFromFloat(1.0).IsZero() {
		t.Error("Price(1.0) should not be zero")
	}
}

func TestPriceAddSub(t *testing.T) {
	a := PriceFromFloat(100.0)
	b := PriceFromFloat(50.0)
	sum := a.Add(b)
	if sum.Float64() != 150.0 {
		t.Errorf("100 + 50 = %f", sum.Float64())
	}
	diff := a.Sub(b)
	if diff.Float64() != 50.0 {
		t.Errorf("100 - 50 = %f", diff.Float64())
	}
}

func TestPriceMulDiv(t *testing.T) {
	p := PriceFromFloat(100.0)
	if p.Mul(2.0).Float64() != 200.0 {
		t.Errorf("100 * 2 = %f", p.Mul(2.0).Float64())
	}
	if p.Div(4.0).Float64() != 25.0 {
		t.Errorf("100 / 4 = %f", p.Div(4.0).Float64())
	}
	if p.Div(0).Float64() != 0 {
		t.Error("division by zero should return 0")
	}
}

func TestPriceCompare(t *testing.T) {
	a := PriceFromFloat(100.0)
	b := PriceFromFloat(101.0)
	if a.Compare(b) >= 0 {
		t.Error("100 should be < 101")
	}
	if b.Compare(a) <= 0 {
		t.Error("101 should be > 100")
	}
	if a.Compare(PriceFromFloat(100.0)) != 0 {
		t.Error("100 should == 100")
	}
}

func TestPriceMinMax(t *testing.T) {
	a := PriceFromFloat(100.0)
	b := PriceFromFloat(200.0)
	if MinPrice(a, b).Float64() != 100.0 {
		t.Error("min(100, 200) should be 100")
	}
	if MaxPrice(a, b).Float64() != 200.0 {
		t.Error("max(100, 200) should be 200")
	}
}

func TestPriceSpreadBps(t *testing.T) {
	p := PriceFromFloat(100.0)
	withSpread := p.SpreadBps(5.0)
	if withSpread.Float64() != 100.05 {
		t.Errorf("100 + 5 bps should be 100.05, got %f", withSpread.Float64())
	}
	withNegSpread := p.SpreadBps(-5.0)
	if withNegSpread.Float64() != 99.95 {
		t.Errorf("100 - 5 bps should be 99.95, got %f", withNegSpread.Float64())
	}
}

func TestPnLFromPrices(t *testing.T) {
	entry := PriceFromFloat(100.0)
	exit := PriceFromFloat(105.0)
	pnl := PnLFromPrices(entry, exit, 10, "BUY")
	if pnl != 50.0 {
		t.Errorf("buy PnL: (105-100)*10 = 50, got %f", pnl)
	}

	pnlShort := PnLFromPrices(entry, exit, 10, "SELL")
	if pnlShort != -50.0 {
		t.Errorf("short PnL: (100-105)*10 = -50, got %f", pnlShort)
	}
}

func TestNotional(t *testing.T) {
	p := PriceFromFloat(150.0)
	if Notional(p, 10) != 1500.0 {
		t.Errorf("150 * 10 = 1500, got %f", Notional(p, 10))
	}
}

func TestPriceJSON(t *testing.T) {
	p := PriceFromFloat(1.50150)
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(data) != "1.50150" {
		t.Errorf("expected 1.50150, got %s", string(data))
	}

	var p2 Price
	if err := json.Unmarshal(data, &p2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p2.Float64() != 1.5015 {
		t.Errorf("expected 1.5015, got %f", p2.Float64())
	}
}

func TestPriceScaleConsistency(t *testing.T) {
	if PriceScaleFactor != 100000 {
		t.Errorf("PriceScaleFactor should be 100000, got %d", PriceScaleFactor)
	}
	raw := int64(150150)
	p := PriceFromInt64(raw)
	if p.Float64() != 1.5015 {
		t.Errorf("raw 150150 should be 1.5015, got %f", p.Float64())
	}
}
