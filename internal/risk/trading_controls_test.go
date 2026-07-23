package risk

import (
	"testing"
	"time"
)

// ── OrderRateLimiter ─────────────────────────────────────────────

func TestOrderRateLimiter_Allow_FirstOrder(t *testing.T) {
	lim := NewOrderRateLimiter(5)
	if !lim.Allow("test") {
		t.Fatal("first order should be allowed")
	}
}

func TestOrderRateLimiter_Allow_WithinLimit(t *testing.T) {
	lim := NewOrderRateLimiter(5)
	for i := 0; i < 5; i++ {
		if !lim.Allow("test") {
			t.Fatalf("order %d should be allowed", i+1)
		}
	}
}

func TestOrderRateLimiter_Allow_ExceedsLimit(t *testing.T) {
	lim := NewOrderRateLimiter(3)
	for i := 0; i < 3; i++ {
		lim.Allow("test")
	}
	if lim.Allow("test") {
		t.Fatal("4th order within 1s window should be blocked")
	}
}

func TestOrderRateLimiter_Allow_DifferentKeys(t *testing.T) {
	lim := NewOrderRateLimiter(1)
	lim.Allow("key-a") // fills bucket for key-a
	if !lim.Allow("key-b") {
		t.Fatal("different key should not be affected by key-a's rate limit")
	}
}

func TestOrderRateLimiter_Reset(t *testing.T) {
	lim := NewOrderRateLimiter(1)
	lim.Allow("test") // fills bucket
	lim.Reset()
	if !lim.Allow("test") {
		t.Fatal("after Reset, first order should be allowed")
	}
}

func TestOrderRateLimiter_SlidingWindow_ExpiresOldEntries(t *testing.T) {
	// Use a limiter with a very short window by overriding the window size
	lim := &OrderRateLimiter{
		maxPerSec:  2,
		windowSize: 50 * time.Millisecond,
		orders:     make(map[string][]time.Time),
	}
	lim.Allow("test")
	lim.Allow("test")
	if lim.Allow("test") {
		t.Fatal("3rd order within 50ms should be blocked")
	}
	time.Sleep(100 * time.Millisecond)
	if !lim.Allow("test") {
		t.Fatal("after window expiration, order should be allowed")
	}
}

// ── VolatilityHalt ───────────────────────────────────────────────

func TestVolatilityHalt_NotHalted_Initially(t *testing.T) {
	v := NewVolatilityHalt(3.0)
	if v.IsHalted() {
		t.Fatal("should not be halted initially")
	}
}

func TestVolatilityHalt_NotHalted_NormalReturns(t *testing.T) {
	v := NewVolatilityHalt(3.0)
	rng := []float64{0.001, -0.002, 0.001, 0.000, -0.001, 0.002, 0.001, -0.001, 0.000, 0.001, 0.000}
	for _, r := range rng {
		v.UpdateReturn(r)
	}
	if v.IsHalted() {
		t.Fatal("should not halt on normal returns within threshold")
	}
}

func TestVolatilityHalt_Halted_ExtremeReturn(t *testing.T) {
	v := NewVolatilityHalt(2.0) // lower threshold to trigger easily
	normal := make([]float64, 20)
	for i := range normal {
		normal[i] = 0.001
	}
	for _, r := range normal {
		v.UpdateReturn(r)
	}
	// Inject a 8-sigma event
	v.UpdateReturn(0.08) // extremely high return
	if !v.IsHalted() {
		t.Fatal("should halt on extreme z-score return")
	}
	if v.HaltReason() != "volatility_z" {
		t.Fatalf("halt reason should be volatility_z, got %s", v.HaltReason())
	}
}

func TestVolatilityHalt_Reset(t *testing.T) {
	v := NewVolatilityHalt(2.0)
	normal := make([]float64, 20)
	for i := range normal {
		normal[i] = 0.001
	}
	for _, r := range normal {
		v.UpdateReturn(r)
	}
	v.UpdateReturn(0.08) // trigger halt
	if !v.IsHalted() {
		t.Fatal("should be halted before reset")
	}
	v.Reset()
	if v.IsHalted() {
		t.Fatal("should not be halted after reset")
	}
}

func TestVolatilityHalt_NotHaltedInsufficientData(t *testing.T) {
	v := NewVolatilityHalt(2.0)
	for i := 0; i < 5; i++ {
		v.UpdateReturn(0.05) // extreme, but only 5 data points (< 10)
	}
	if v.IsHalted() {
		t.Fatal("should not halt with less than 10 data points")
	}
}

// ── ExposureTracker ──────────────────────────────────────────────

func TestExposureTracker_CheckOrder_AllowsWithinLimits(t *testing.T) {
	et := NewExposureTracker(2.0, 0.5)
	et.SetEquity(100000)
	ok, reason := et.CheckOrder("AAPL", "BUY", 20000)
	if !ok {
		t.Fatalf("expected allowed, got: %s", reason)
	}
}

func TestExposureTracker_CheckOrder_BlocksExceedsLeverage(t *testing.T) {
	et := NewExposureTracker(1.0, 1.0)
	et.SetEquity(100000)
	et.AddPosition("AAPL", "BUY", 90000)
	ok, reason := et.CheckOrder("MSFT", "BUY", 20000)
	if ok {
		t.Fatal("should block order that exceeds max leverage")
	}
	if reason != "max_leverage" {
		t.Fatalf("expected max_leverage, got %s", reason)
	}
}

func TestExposureTracker_CheckOrder_BlocksSymbolGrossExposure(t *testing.T) {
	et := NewExposureTracker(5.0, 0.3) // max 30% gross per symbol
	et.SetEquity(100000)
	et.AddPosition("AAPL", "BUY", 25000)
	ok, reason := et.CheckOrder("AAPL", "SELL", 10000) // total AAPL gross = 35k = 35%
	if ok {
		t.Fatal("should block order that exceeds per-symbol gross exposure")
	}
	if reason != "max_symbol_gross_exposure" {
		t.Fatalf("expected max_symbol_gross_exposure, got %s", reason)
	}
}

func TestExposureTracker_CheckOrder_BlocksNegativeEquity(t *testing.T) {
	et := NewExposureTracker(2.0, 1.0)
	et.SetEquity(-5000)
	ok, reason := et.CheckOrder("AAPL", "BUY", 1000)
	if ok {
		t.Fatal("should block order when equity is negative")
	}
	if reason != "equity_negative" {
		t.Fatalf("expected equity_negative, got %s", reason)
	}
}

func TestExposureTracker_GrossExposure(t *testing.T) {
	et := NewExposureTracker(5.0, 1.0)
	et.AddPosition("AAPL", "BUY", 30000)
	et.AddPosition("MSFT", "SELL", 20000)
	if et.GrossExposure() != 50000 {
		t.Fatalf("gross exposure should be 50000, got %.0f", et.GrossExposure())
	}
}

func TestExposureTracker_RemovePosition(t *testing.T) {
	et := NewExposureTracker(5.0, 1.0)
	et.AddPosition("AAPL", "BUY", 30000)
	et.AddPosition("AAPL", "BUY", 20000)
	et.RemovePosition("AAPL", "BUY", 15000)
	if et.GrossExposure() != 35000 {
		t.Fatalf("gross exposure after partial remove should be 35000, got %.0f", et.GrossExposure())
	}
}
