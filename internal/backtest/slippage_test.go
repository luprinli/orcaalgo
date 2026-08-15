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

func TestLimitFillProbability_AtMid(t *testing.T) {
	m := DefaultEquitySlippage()
	p := m.LimitFillProbability(0)
	if p < 0.95 || p > 1.0 {
		t.Errorf("At mid should have near-certain fill, got %f", p)
	}
}

func TestLimitFillProbability_FarFromMid(t *testing.T) {
	m := DefaultEquitySlippage()
	p := m.LimitFillProbability(10.0)
	if p > 0.5 {
		t.Errorf("10 bps from mid should have <50%% fill prob, got %f", p)
	}
	p = m.LimitFillProbability(50.0)
	if p < 0.04 || p > 0.15 {
		t.Errorf("50 bps should decay significantly, got %f", p)
	}
}

func TestLimitFillProbability_Clamped(t *testing.T) {
	m := DefaultEquitySlippage()
	p := m.LimitFillProbability(1000.0)
	if p < 0.04 || p > 0.06 {
		t.Errorf("Very far from mid should clamp near minimum, got %f", p)
	}
}

func TestCalibrateSlippageModel_InsufficientSamples(t *testing.T) {
	m := DefaultEquitySlippage()
	original := m.SpreadBps
	calibrated := CalibrateSlippageModel(m, 5.0, 5)
	if calibrated.SpreadBps != original {
		t.Error("Insufficient samples should not change model")
	}
}

func TestCalibrateSlippageModel_AdjustsUp(t *testing.T) {
	m := DefaultEquitySlippage()
	observed := m.SpreadBps + m.AdverseSelectBps + m.MaxSlippage*0.4 + 2.0
	calibrated := CalibrateSlippageModel(m, observed, 50)
	if calibrated.SpreadBps <= m.SpreadBps {
		t.Error("Higher observed slippage should increase model spread")
	}
}

func TestCalibrateSlippageModel_AdjustsDown(t *testing.T) {
	m := DefaultEquitySlippage()
	observed := 0.1
	calibrated := CalibrateSlippageModel(m, observed, 50)
	if calibrated.SpreadBps >= m.SpreadBps {
		t.Error("Lower observed slippage should decrease model spread")
	}
}

func TestAdverseSelectBps_IncreasesSlippage(t *testing.T) {
	mNoAS := SlippageModel{SpreadBps: 0.5, MaxSlippage: 0, AdverseSelectBps: 0}
	mWithAS := SlippageModel{SpreadBps: 0.5, MaxSlippage: 0, AdverseSelectBps: 2.0}
	fsNoAS := NewFillSimulatorWithSeed(mNoAS, 42)
	fsWithAS := NewFillSimulatorWithSeed(mWithAS, 42)

	fillNo := fsNoAS.SimulateFillWithTCA(1, "SPY", 500, 100, "BUY", 500, time.Now(), 500, 500, 0)
	fillWith := fsWithAS.SimulateFillWithTCA(1, "SPY", 500, 100, "BUY", 500, time.Now(), 500, 500, 0)

	if fillWith.FillPrice.Float64() <= fillNo.FillPrice.Float64() {
		t.Errorf("Adverse selection should increase fill price for BUY: noAS=%.4f withAS=%.4f",
			fillNo.FillPrice.Float64(), fillWith.FillPrice.Float64())
	}
}

func TestVolumeImpact_SqrtModel(t *testing.T) {
	m := SlippageModel{SpreadBps: 0, MaxSlippage: 0, AdverseSelectBps: 0, VolumeImpactFactor: 0.5}
	fs := NewFillSimulatorWithSeed(m, 1)

	fillSmall := fs.SimulateFillWithTCA(1, "SPY", 500, 10, "BUY", 500, time.Now(), 500, 500, 100000)
	fillLarge := fs.SimulateFillWithTCA(1, "SPY", 500, 1000, "BUY", 500, time.Now(), 500, 500, 100000)

	if fillLarge.FillPrice.Float64() <= fillSmall.FillPrice.Float64() {
		t.Errorf("Larger qty should have higher fill price due to volume impact: small=%.4f large=%.4f",
			fillSmall.FillPrice.Float64(), fillLarge.FillPrice.Float64())
	}
}

func TestNewFillSimulatorWithSeed_Deterministic(t *testing.T) {
	m := DefaultEquitySlippage()
	fs1 := NewFillSimulatorWithSeed(m, 42)
	fs2 := NewFillSimulatorWithSeed(m, 42)

	tickTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	f1 := fs1.SimulateFill(1, "AAPL", 185.0, 50.0, "BUY", 185.0, tickTime)
	f2 := fs2.SimulateFill(1, "AAPL", 185.0, 50.0, "BUY", 185.0, tickTime)
	if f1.FillPrice != f2.FillPrice || f1.FillQuantity != f2.FillQuantity {
		t.Errorf("With same seed, fills should be deterministic: price(%f vs %f) qty(%f vs %f)",
			f1.FillPrice.Float64(), f2.FillPrice.Float64(), f1.FillQuantity, f2.FillQuantity)
	}
}

func TestFillSimulator_SlippageMidLast(t *testing.T) {
	m := SlippageModel{SpreadBps: 0.5, MaxSlippage: 0, AdverseSelectBps: 0}
	fs := NewFillSimulatorWithSeed(m, 99)
	fill := fs.SimulateFillWithTCA(1, "AAPL", 185.0, 100, "BUY", 185.0, time.Now(), 184.5, 184.0, 0)
	if fill.SlippageMidBps <= 0 {
		t.Error("BUY above mid should have positive SlippageMidBps")
	}
	if fill.SlippageLastBps <= 0 {
		t.Error("BUY above last should have positive SlippageLastBps")
	}
}

func TestSlippageForSymbol_Routing(t *testing.T) {
	cases := []struct {
		symbol    string
		spreadBps float64
	}{
		// Crypto — hyphenated universe tickers must route to CryptoSlippage.
		{"BTC-USD", 12.0},
		{"ETH-USD", 12.0},
		// Legacy canonical forms still route correctly.
		{"BTCUSD", 12.0},
		{"ETHUSD", 12.0},
		// Forex majors.
		{"EURUSD", 0.3},
		{"GBPUSD", 0.3},
		{"USDJPY", 0.3},
		{"AUDUSD", 0.3},
		{"USDCAD", 0.3},
		// Small-cap equities.
		{"NVDA", 8.0},
		{"IWM", 8.0},
		{"TSLA", 8.0},
		// Commodity ETFs.
		{"GLD", 4.0},
		{"TLT", 4.0},
		// Default equity fallback.
		{"SPY", 2.0},
		{"AAPL", 2.0},
		{"QQQ", 2.0},
	}
	for _, c := range cases {
		m := SlippageForSymbol(c.symbol)
		if m.SpreadBps != c.spreadBps {
			t.Errorf("SlippageForSymbol(%q) spread = %v, want %v", c.symbol, m.SpreadBps, c.spreadBps)
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

func TestApplyCalibratedCosts_OverridesPositiveValues(t *testing.T) {
	base := SlippageForSymbol("SPY")
	out := ApplyCalibratedCosts(base, 3.5, 1.25)
	if out.SpreadBps != 3.5 {
		t.Errorf("spread = %v, want 3.5", out.SpreadBps)
	}
	if out.VolumeImpactFactor != 1.25 {
		t.Errorf("impact = %v, want 1.25", out.VolumeImpactFactor)
	}
	if out.AdverseSelectBps != base.AdverseSelectBps {
		t.Errorf("adverse selection should be preserved, got %v", out.AdverseSelectBps)
	}
}

func TestApplyCalibratedCosts_IgnoresUncalibrated(t *testing.T) {
	base := SlippageForSymbol("SPY")
	for _, bad := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		out := ApplyCalibratedCosts(base, bad, bad)
		if out.SpreadBps != base.SpreadBps {
			t.Errorf("spread for %v = %v, want base %v", bad, out.SpreadBps, base.SpreadBps)
		}
		if out.VolumeImpactFactor != base.VolumeImpactFactor {
			t.Errorf("impact for %v = %v, want base %v", bad, out.VolumeImpactFactor, base.VolumeImpactFactor)
		}
	}
}
