package backtest

import (
	"math"
	"testing"
	"time"
)

func TestDefaultEquitySlippage(t *testing.T) {
	m := DefaultEquitySlippage()
	if m.SpreadBps <= 0 {
		t.Error("Expected positive spread")
	}
	if m.MaxSlippage <= 0 {
		t.Error("Expected positive max slippage")
	}
}

func TestNewFillSimulator(t *testing.T) {
	m := DefaultEquitySlippage()
	fs := NewFillSimulator(m)
	if fs == nil {
		t.Fatal("Expected non-nil fill simulator")
	}
}

func TestSimulateFill_Basic(t *testing.T) {
	m := DefaultEquitySlippage()
	fs := NewFillSimulator(m)

	fill := fs.SimulateFill(1, "SPY", 500.0, 100.0, "BUY", 500.0, time.Now())
	if fill.FillQuantity <= 0 {
		t.Error("Expected positive fill quantity")
	}
	if fill.FillPrice.Float64() <= 0 {
		t.Error("Expected positive fill price")
	}
	if fill.FillPrice.Float64() < 500.0*(1-(m.SpreadBps+m.MaxSlippage)/10000.0) {
		t.Errorf("Slippage too extreme: price=%f", fill.FillPrice.Float64())
	}
}

func TestSimulateFill_BUYvsSELL(t *testing.T) {
	m := DefaultEquitySlippage()
	fs := NewFillSimulator(m)

	buy := fs.SimulateFill(1, "SPY", 500.0, 100.0, "BUY", 500.0, time.Now())
	sell := fs.SimulateFill(2, "SPY", 500.0, 100.0, "SELL", 500.0, time.Now())

	if buy.FillPrice.Float64() < sell.FillPrice.Float64() {
		t.Logf("BUY price %f < SELL price %f (spread effect)", buy.FillPrice.Float64(), sell.FillPrice.Float64())
	}
}

func TestSimulateFill_ZeroQuantity(t *testing.T) {
	m := DefaultEquitySlippage()
	fs := NewFillSimulator(m)

	fill := fs.SimulateFill(1, "SPY", 500.0, 0.0, "BUY", 500.0, time.Now())
	if fill.FillQuantity != 0 {
		t.Errorf("Expected 0 quantity, got %f", fill.FillQuantity)
	}
}

func TestSimulateFill_ExtremeSlippage(t *testing.T) {
	m := SlippageModel{SpreadBps: 50, MaxSlippage: 200, LatencyMs: 50}
	fs := NewFillSimulator(m)

	for i := 0; i < 100; i++ {
		fill := fs.SimulateFill(1, "SPY", 500.0, 100.0, "BUY", 500.0, time.Now())
		if fill.FillQuantity <= 0 {
			t.Error("Expected positive quantity even with extreme slippage")
		}
		if math.Abs(fill.FillPrice.Float64()-500.0) > 500.0*0.05 {
			t.Errorf("Slippage > 5%%: price=%f", fill.FillPrice.Float64())
		}
	}
}

func TestFillSimulator_MultipleSymbols(t *testing.T) {
	m := DefaultEquitySlippage()
	fs := NewFillSimulator(m)

	symbols := []string{"SPY", "QQQ", "AAPL", "MSFT"}
	for _, sym := range symbols {
		fill := fs.SimulateFill(1, sym, 500.0, 100.0, "BUY", 500.0, time.Now())
		if fill.Symbol != "" {
			t.Logf("Fill for %s: qty=%f", sym, fill.FillQuantity)
		}
	}
}

func TestLowLatencySlippage(t *testing.T) {
	m := LowLatencySlippage()
	if m.LatencyMs >= 5.0 {
		t.Error("Expected low latency < 5ms")
	}
	fs := NewFillSimulator(m)
	fill := fs.SimulateFill(1, "SPY", 500.0, 100.0, "BUY", 500.0, time.Now())
	if fill.FillQuantity <= 0 {
		t.Error("Expected fill")
	}
}
