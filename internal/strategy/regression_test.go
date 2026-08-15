package strategy

import (
	"math"
	"testing"
	"time"

	"github.com/lee-econ/orca-core/internal/types"
)

// Regression tests for the strategy-logic audit fixes (docs/strategy_logic_audit.md):
// true-range ATR, circular-buffer correctness, return-vol filter, regime de-duplication,
// and the trend trailing-stop arithmetic.

func TestTrueRangeATR_UsesTrueRange(t *testing.T) {
	highs := []float64{10, 11, 12, 13, 14}
	lows := []float64{9, 9.5, 10, 11, 12}
	closes := []float64{9.5, 10.5, 11.5, 12.5, 13.5}
	// bars 1..4 TRs: 1.5, 2, 2, 2 → mean 1.875.
	got := TrueRangeATR(highs, lows, closes, 5, 4)
	if math.Abs(got-1.875) > 1e-9 {
		t.Errorf("TrueRangeATR = %v, want 1.875", got)
	}
}

func TestTrueRangeATR_ExceedsCloseToCloseOnGaps(t *testing.T) {
	// A gap-down day: high-low range is small, but the gap from prev close is large.
	highs := []float64{100, 100.2, 95.5}
	lows := []float64{99, 99.8, 95.0}
	closes := []float64{99.5, 100.0, 95.2}
	// bar index 2 true range = max(95.5-95.0, |95.5-100.0|, |95.0-100.0|) = max(0.5, 4.5, 5.0) = 5.0
	// (a close-to-close measure would see |95.2-100.0| = 4.8, understating the range).
	got := TrueRangeATR(highs, lows, closes, 3, 1)
	if math.Abs(got-5.0) > 1e-9 {
		t.Errorf("TrueRangeATR = %v, want 5.0", got)
	}
}

func TestTrueRangeATR_CircularBuffer(t *testing.T) {
	// A ring buffer of size 4 that has wrapped: logical order is [30, 40, 50, 60]
	// but stored circularly at indices [2, 3, 0, 1].
	size := 4
	highs := make([]float64, size)
	lows := make([]float64, size)
	closes := make([]float64, size)
	logical := []float64{30, 40, 50, 60}
	for i, v := range logical {
		idx := (2 + i) % size
		closes[idx] = v
		highs[idx] = v + 1
		lows[idx] = v - 1
	}
	got := TrueRangeATR(highs, lows, closes, 4, 3)
	if got <= 0 {
		t.Fatalf("TrueRangeATR over wrapped buffer = %v, want > 0", got)
	}
}

func TestMeanReversion_ReturnVol(t *testing.T) {
	r := NewMeanReversionRunner(10, 1.5, 0.5, 200)
	// Feed a deterministic price path with known return volatility.
	prices := []float64{100, 101, 100, 101, 100, 101, 100, 101, 100, 101, 100, 101}
	for _, p := range prices {
		r.closeHistory[r.histIndex%(10+200)] = p
		r.histIndex++
		r.histCount++
	}
	vol := r.returnVol(10)
	// Returns alternate ±~1% → std ≈ 0.01. Check it's positive and sane.
	if vol <= 0 || vol > 0.05 {
		t.Errorf("returnVol = %v, want ~0.01", vol)
	}
}

func TestGridRunner_NoRegimeCrisisEarlyReturn(t *testing.T) {
	r := NewGridRunner()
	r.Disabled = false
	candle := Candle{
		Symbol: "X", Close: types.PriceFromFloat(100),
		High: types.PriceFromFloat(101), Low: types.PriceFromFloat(99),
		Volume: 1000, Time: time.Now(),
	}
	// Regime 3 (crisis) must NOT early-return: gating now lives in the pipeline.
	_ = r.Evaluate(candle, 3)
	if !r.priceInit {
		t.Error("grid runner should initialize its reference price in crisis regime (gating de-duplicated to the pipeline)")
	}
}

func TestBaseRunner_DailyTradeCounter(t *testing.T) {
	b := NewBaseRunner(16)
	day1 := time.Date(2024, 1, 2, 9, 30, 0, 0, time.UTC)
	day2 := day1.Add(24 * time.Hour)

	for i := 0; i < 3; i++ {
		if !b.CanTrade(day1, 3) {
			t.Fatalf("trade %d of 3 should be allowed", i+1)
		}
		b.RecordTrade(day1)
	}
	if b.CanTrade(day1, 3) {
		t.Error("trade 4 of 3 should be blocked")
	}

	if !b.CanTrade(day2, 3) {
		t.Error("new day should reset the counter")
	}
	if b.tradesToday != 0 {
		t.Errorf("tradesToday should reset on new day, got %d", b.tradesToday)
	}

	if !b.CanTrade(day2, 0) {
		t.Error("maxPerDay <= 0 should be unlimited")
	}
}

func TestWarmUpNoEarlySignals(t *testing.T) {
	// Indicator-heavy strategies must not emit an entry before their warm-up
	// (Rule 16). The warm-up guard must exceed the indicator's own minimum.
	cases := []struct {
		name string
		run  func() Strategy
		bars int // one below the warm-up minimum
	}{
		{"ichimoku", func() Strategy { return NewIchimokuRunner() }, 51},
		{"donchian", func() Strategy { return NewDonchianBreakoutRunner() }, 21},
		{"keltner", func() Strategy { return NewKeltnerMACDRunner() }, 24},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := c.run()
			for i := 0; i < c.bars; i++ {
				candle := Candle{
					Symbol: "X",
					Time:   time.Date(2024, 1, 2, 9, 30, 0, 0, time.UTC).Add(time.Duration(i) * time.Minute),
					Open:   types.PriceFromFloat(100), High: types.PriceFromFloat(101),
					Low: types.PriceFromFloat(99), Close: types.PriceFromFloat(100 + float64(i)*0.05),
					Volume: 1000,
				}
				if sig := r.Evaluate(candle, 0); sig != nil && sig.Action == SignalEntry {
					t.Fatalf("%s emitted an entry at bar %d (before its warm-up)", c.name, i)
				}
			}
		})
	}
}

func TestRegistry_CreateFreshInstances(t *testing.T) {
	// Backtests must use Create (fresh), not Get (shared singleton), so mutable
	// runner state is never shared across concurrent engines (Rule 12).
	a := GlobalRegistry().Create("trend_following")
	b := GlobalRegistry().Create("trend_following")
	if a == b {
		t.Fatal("Create must return distinct instances")
	}
	ta, tb := a.(*TrendRunner), b.(*TrendRunner)
	ta.fastEMA = 123.0
	if tb.fastEMA == 123.0 {
		t.Fatal("mutating one instance must not affect the other")
	}
}

func TestSession_InWindow(t *testing.T) {
	s := NewETSession() // 9:30–16:00 ET (UTC-4)
	if !s.InWindow(time.Date(2024, 1, 2, 13, 30, 0, 0, time.UTC)) {
		t.Error("9:30 ET (13:30 UTC) should be in window")
	}
	if s.InWindow(time.Date(2024, 1, 2, 20, 0, 0, 0, time.UTC)) {
		t.Error("16:00 ET (20:00 UTC) should be out (exclusive end)")
	}
	if s.InWindow(time.Date(2024, 1, 2, 13, 0, 0, 0, time.UTC)) {
		t.Error("9:00 ET (13:00 UTC) should be out (before open)")
	}
	if got := s.DayKey(time.Date(2024, 1, 2, 1, 0, 0, 0, time.UTC)); got != "2024-01-01" {
		t.Errorf("DayKey across midnight = %q, want 2024-01-01 (session-local day)", got)
	}
}

func TestStopModel(t *testing.T) {
	entry := types.PriceFromFloat(100)
	if got := StopPrice(entry, 5, "BUY").Float64(); got != 95 {
		t.Errorf("StopPrice(BUY) = %v, want 95", got)
	}
	if got := StopPrice(entry, 5, "SELL").Float64(); got != 105 {
		t.Errorf("StopPrice(SELL) = %v, want 105", got)
	}
	if got := TargetPrice(entry, 5, "BUY").Float64(); got != 105 {
		t.Errorf("TargetPrice(BUY) = %v, want 105", got)
	}
	if got := TargetPrice(entry, 5, "SELL").Float64(); got != 95 {
		t.Errorf("TargetPrice(SELL) = %v, want 95", got)
	}
	if got := TrailingStop(entry, 5, "BUY").Float64(); got != 95 {
		t.Errorf("TrailingStop(BUY) = %v, want 95", got)
	}
	if got := TrailingStop(entry, 5, "SELL").Float64(); got != 105 {
		t.Errorf("TrailingStop(SELL) = %v, want 105", got)
	}
}

func TestTrendRunner_TrailingStopPriceUsesDistance(t *testing.T) {
	r := NewTrendRunner()
	r.stopDistance = 5.0
	peak := types.PriceFromFloat(105)

	longStop := r.trailingStopPrice(peak, "BUY")
	if longStop.Float64() != 100.0 {
		t.Errorf("long trailing stop = %v, want 100 (peak - distance)", longStop.Float64())
	}
	shortStop := r.trailingStopPrice(peak, "SELL")
	if shortStop.Float64() != 110.0 {
		t.Errorf("short trailing stop = %v, want 110 (peak + distance)", shortStop.Float64())
	}
}

func TestOrbRunner_OneTradePerDay(t *testing.T) {
	r := NewOrbRunner()
	day := time.Date(2024, 1, 2, 14, 30, 0, 0, time.UTC) // 9:30 ET

	// Form the 5-bar opening range (high 100.5, low 99.5).
	for i := 0; i < 5; i++ {
		c := Candle{Symbol: "X", Close: types.PriceFromFloat(100), High: types.PriceFromFloat(100.5), Low: types.PriceFromFloat(99.5), Volume: 1000, Time: day.Add(time.Duration(i) * 5 * time.Minute)}
		_ = r.Evaluate(c, 0)
	}

	// Breakout above the range → single BUY entry.
	breakout := Candle{Symbol: "X", Close: types.PriceFromFloat(101), High: types.PriceFromFloat(101), Low: types.PriceFromFloat(100.8), Volume: 1000, Time: day.Add(30 * time.Minute)}
	sig := r.Evaluate(breakout, 0)
	if sig == nil || sig.Side != "BUY" || sig.Action != SignalEntry {
		t.Fatalf("expected BUY entry on breakout, got %+v", sig)
	}

	// Stop-out (close below stop) → exit.
	stopOut := Candle{Symbol: "X", Close: types.PriceFromFloat(100), High: types.PriceFromFloat(100.5), Low: types.PriceFromFloat(99.5), Volume: 1000, Time: day.Add(40 * time.Minute)}
	ex := r.Evaluate(stopOut, 0)
	if ex == nil || ex.Action != SignalExit {
		t.Fatalf("expected exit on stop-out, got %+v", ex)
	}

	// Same-day re-entry must be blocked (one trade per day).
	reentry := Candle{Symbol: "X", Close: types.PriceFromFloat(101.5), High: types.PriceFromFloat(101.5), Low: types.PriceFromFloat(101), Volume: 1000, Time: day.Add(45 * time.Minute)}
	if sig3 := r.Evaluate(reentry, 0); sig3 != nil {
		t.Fatalf("expected no same-day re-entry, got %+v", sig3)
	}
}
