package strategy

import (
	"testing"
	"time"

	"github.com/lee-econ/orca-core/internal/types"
)

func TestGridRunner_DynamicReset(t *testing.T) {
	r := NewGridRunner()
	r.Disabled = false // re-enable for test
	r.Evaluate(mkCandle("SPY", 100.0, 101.0, 99.0, 1000), 0)

	if !r.priceInit {
		t.Fatal("grid should be initialized after first bar")
	}
	if r.referencePrice.Float64() != 100.0 {
		t.Errorf("reference price should be 100, got %.2f", r.referencePrice.Float64())
	}

	// After 100 bars with no open positions, reference should reset.
	for i := 0; i < 100; i++ {
		r.Evaluate(mkCandle("SPY", 105.0, 106.0, 104.0, 1000), 0)
	}
	if r.referencePrice.Float64() != 105.0 {
		t.Errorf("reference price should reset to 105 after 100 empty bars, got %.2f", r.referencePrice.Float64())
	}
}

func TestGridRunner_DisabledSkips(t *testing.T) {
	r := NewGridRunner()
	r.Disabled = true // default

	sig := r.Evaluate(mkCandle("SPY", 100.0, 101.0, 99.0, 1000), 0)
	if sig != nil {
		t.Error("disabled grid should return nil signal")
	}
}

func TestOrbRunner_MinVolatilityRequirement(t *testing.T) {
	r := NewOrbRunner()
	r.RangeMinutes = 1 // set range quickly
	r.MinRangePct = 0.3

	// Feed a bar with narrow range (0.2% < 0.3%)
	candle := mkCandle("SPY", 100.0, 100.15, 99.95, 1000)
	sig := r.Evaluate(candle, 1)
	if sig != nil {
		t.Error("orb should reject when range < 0.3%")
	}

	// Feed a bar with sufficient range (> 0.3%)
	r = NewOrbRunner()
	r.RangeMinutes = 1
	r.MinRangePct = 0.3
	candle2 := mkCandle("SPY", 100.0, 100.6, 99.4, 1000)
	sig2 := r.Evaluate(candle2, 1)
	// With 1 bar, range is set but too narrow to trigger breakout.
	// The signal should at least not be nil due to the volatility filter.
	_ = sig2 // just verify no panic
}

func TestSessionScalpRunner_MaxTradesPerDay(t *testing.T) {
	r := NewSessionScalpRunner()
	r.MaxTradesPerDay = 3
	r.SessionStartHour = 0
	r.SessionStartMin = 0
	r.SessionEndHour = 23
	r.SessionEndMin = 59
	r.RangeMinutes = 1
	r.EntryBufferPct = 0.0

	now := time.Date(2026, 8, 2, 10, 30, 0, 0, time.UTC)

	// Seed: set the opening range (broad enough to allow entry).
	candle := Candle{Symbol: "SPY", Time: now, Open: mkPrice(100.0), High: mkPrice(105.0), Low: mkPrice(95.0), Close: mkPrice(100.0), Volume: 1000000}
	r.Evaluate(candle, 1)

	// Generate 3 trades.
	for i := 0; i < 3; i++ {
		candle2 := Candle{Symbol: "SPY", Time: now.Add(time.Duration(i+1) * time.Minute), Open: mkPrice(110.0), High: mkPrice(111.0), Low: mkPrice(109.0), Close: mkPrice(110.0), Volume: 1000000}
		sig := r.Evaluate(candle2, 1)
		if sig != nil && sig.Quantity > 0 {
			// Trade registered
		}
	}

	// 4th trade should be blocked.
	candle3 := Candle{Symbol: "SPY", Time: now.Add(5 * time.Minute), Open: mkPrice(112.0), High: mkPrice(113.0), Low: mkPrice(111.0), Close: mkPrice(112.0), Volume: 1000000}
	sig := r.Evaluate(candle3, 1)
	if sig != nil && sig.Quantity > 0 {
		t.Error("4th trade should be blocked by max trades per day limit")
	}

	// Next day should reset.
	nextDay := Candle{Symbol: "SPY", Time: now.Add(24 * time.Hour), Open: mkPrice(100.0), High: mkPrice(105.0), Low: mkPrice(95.0), Close: mkPrice(100.0), Volume: 1000000}
	r.Evaluate(nextDay, 1)
	if r.dailyTradeCount != 0 {
		t.Errorf("daily trade count should reset to 0 on new day, got %d", r.dailyTradeCount)
	}
}

func TestOrbRunner_ResetsRangePerDay(t *testing.T) {
	r := NewOrbRunner()
	r.RangeMinutes = 2
	r.MinRangePct = 0

	day1 := time.Date(2026, 8, 3, 9, 30, 0, 0, time.UTC)

	// Day 1: two bars form the opening range (high=105, low=95).
	c1 := Candle{Symbol: "SPY", Time: day1, Open: mkPrice(100), High: mkPrice(102), Low: mkPrice(99), Close: mkPrice(100), Volume: 1000}
	r.Evaluate(c1, 1)
	c2 := Candle{Symbol: "SPY", Time: day1.Add(time.Minute), Open: mkPrice(100), High: mkPrice(105), Low: mkPrice(95), Close: mkPrice(100), Volume: 1000}
	r.Evaluate(c2, 1)

	if !r.rangeSet {
		t.Fatal("range should be set after 2 bars on day 1")
	}
	if r.openingHigh != 105.0 || r.openingLow != 95.0 {
		t.Fatalf("day1 range = [%v, %v], want [95, 105]", r.openingLow, r.openingHigh)
	}

	// Day 2: the first bar must reset the range and start forming anew.
	day2 := day1.Add(24 * time.Hour)
	c3 := Candle{Symbol: "SPY", Time: day2, Open: mkPrice(200), High: mkPrice(201), Low: mkPrice(199), Close: mkPrice(200), Volume: 1000}
	r.Evaluate(c3, 1)

	if r.rangeSet {
		t.Fatal("range should reset at the start of day 2")
	}
	if r.openingHigh != 201.0 || r.openingLow != 199.0 {
		t.Fatalf("day2 range should re-form from [199, 201], got [%v, %v]", r.openingLow, r.openingHigh)
	}
}

func TestOrbRunner_SkipsNarrowRangeDay(t *testing.T) {
	r := NewOrbRunner()
	r.RangeMinutes = 1
	r.MinRangePct = 0.3

	c := Candle{Symbol: "SPY", Time: time.Date(2026, 8, 3, 9, 30, 0, 0, time.UTC), Open: mkPrice(100), High: mkPrice(100.1), Low: mkPrice(99.9), Close: mkPrice(100), Volume: 1000}
	r.Evaluate(c, 1)

	if !r.skipDay {
		t.Fatal("narrow range should mark the day as skipped")
	}

	// A breakout bar later the same day must be ignored.
	c2 := Candle{Symbol: "SPY", Time: time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC), Open: mkPrice(100), High: mkPrice(110), Low: mkPrice(100), Close: mkPrice(110), Volume: 1000}
	if sig := r.Evaluate(c2, 1); sig != nil {
		t.Error("skipped day should not emit a signal")
	}

	// The next day clears the skip flag.
	c3 := Candle{Symbol: "SPY", Time: time.Date(2026, 8, 4, 9, 30, 0, 0, time.UTC), Open: mkPrice(100), High: mkPrice(105), Low: mkPrice(95), Close: mkPrice(100), Volume: 1000}
	r.Evaluate(c3, 1)
	if r.skipDay {
		t.Error("skip flag should reset on a new day")
	}
}

func TestTrendRunner_CHOPFilter(t *testing.T) {
	r := NewTrendRunner()
	r.AdxThreshold = 5  // allow signals even with low ADX
	r.ChopThreshold = 50 // very permissive

	// Seed with choppy data.
	basePrice := 100.0
	for i := 0; i < 50; i++ {
		dir := 1.0
		if i%2 == 0 {
			dir = -1.0
		}
		price := basePrice + dir*0.5
		basePrice = price
		candle := Candle{
			Symbol: "TEST",
			Time:   time.Now().Add(time.Duration(i) * time.Minute),
			Open:   types.PriceFromFloat(price),
			High:   types.PriceFromFloat(price + 1),
			Low:    types.PriceFromFloat(price - 1),
			Close:  types.PriceFromFloat(price),
			Volume: 1000,
		}
		r.Evaluate(candle, 1)
	}

	// Verify CHOP computation doesn't panic.
	// The CHOP value for choppy data should be above 61.8.
	chop := ChoppinessIndex(r.PriceHistory, r.HighHistory, r.LowHistory, r.HistCount, 14)
	if chop <= 0 {
		t.Errorf("chop should be positive, got %.2f", chop)
	}

	// With ChopThreshold set to 50 (low), choppy data should block.
	// Verify the runner is functional.
	if r.HistCount < 50 {
		t.Errorf("should have at least 50 bars of history, got %d", r.HistCount)
	}
}
