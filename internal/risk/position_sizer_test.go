package risk

import (
	"testing"

	"github.com/lee-econ/orca-core/internal/propfirm"
)

func TestPositionSizer_BasicKelly(t *testing.T) {
	ps := NewPositionSizer(propfirm.DefaultFTMOProfile())

	size := ps.ComputeSize(0.8, 100000.0, "SPY", 0, 0)
	if size <= 0 {
		t.Fatal("Expected non-zero size")
	}
	t.Logf("Size with k=0.25, conf=0.8: %f", size)
}

func TestPositionSizer_ZeroConfidence(t *testing.T) {
	ps := NewPositionSizer(propfirm.DefaultFTMOProfile())

	size := ps.ComputeSize(0.0, 100000.0, "SPY", 0, 0)
	if size > 0 {
		t.Errorf("Expected zero size for zero confidence, got %f", size)
	}
}

func TestPositionSizer_VIXSpike(t *testing.T) {
	p := propfirm.DefaultFTMOProfile()
	p.MaxPositionPct = 100.0
	ps := NewPositionSizer(p)
	ps.SetKellyMultiplier(1.0)
	ps.perTradeCap = 1.0
	ps.totalExpCap = 1.0
	ps.UpdateMarketState(40.0, 50, 0)

	sizeVixHigh := ps.ComputeSize(0.8, 50000.0, "SPY", 0, 0)

	ps.UpdateMarketState(15.0, 50, 0)
	sizeVixLow := ps.ComputeSize(0.8, 50000.0, "SPY", 0, 0)

	if sizeVixHigh >= sizeVixLow {
		t.Errorf("Expected VIX=40 size < VIX=15 size, got %f >= %f", sizeVixHigh, sizeVixLow)
	}
}

func TestPositionSizer_ExtremeSentiment(t *testing.T) {
	p := propfirm.DefaultFTMOProfile()
	p.MaxPositionPct = 100.0
	ps := NewPositionSizer(p)
	ps.SetKellyMultiplier(1.0)
	ps.perTradeCap = 1.0
	ps.totalExpCap = 1.0
	ps.UpdateMarketState(15.0, 5, 0)

	sizeExtreme := ps.ComputeSize(0.8, 50000.0, "SPY", 0, 0)

	ps.UpdateMarketState(15.0, 50, 0)
	sizeNeutral := ps.ComputeSize(0.8, 50000.0, "SPY", 0, 0)

	if sizeExtreme >= sizeNeutral {
		t.Errorf("Expected extreme sentiment size < neutral size, got %f >= %f", sizeExtreme, sizeNeutral)
	}
}

func TestPositionSizer_CrisisRegime(t *testing.T) {
	p := propfirm.DefaultFTMOProfile()
	p.MaxPositionPct = 100.0
	ps := NewPositionSizer(p)
	ps.SetKellyMultiplier(1.0)
	ps.perTradeCap = 1.0
	ps.totalExpCap = 1.0
	ps.UpdateMarketState(15.0, 50, 3)

	sizeCrisis := ps.ComputeSize(0.8, 50000.0, "SPY", 0, 0)

	ps.UpdateMarketState(15.0, 50, 0)
	sizeCalm := ps.ComputeSize(0.8, 50000.0, "SPY", 0, 0)

	if sizeCrisis >= sizeCalm {
		t.Errorf("Expected crisis size < calm size, got %f >= %f", sizeCrisis, sizeCalm)
	}
}

func TestPositionSizer_PerTradeCap(t *testing.T) {
	ps := NewPositionSizer(propfirm.DefaultFTMOProfile())

	// The per-trade *value* cap is enforced upstream in the engine
	// (positionPct <= 3% of capital). ComputeSize applies risk multipliers and a
	// total-exposure cap (<= totalExpCap * baseSize); it no longer imposes the
	// dimensionally-incorrect share-based MaxPositionPct cap that previously
	// shrank every position ~50x. Verify the total-exposure cap holds.
	size := ps.ComputeSize(0.999, 100000.0, "SPY", 0, 0)
	maxExposure := 100000.0 * 0.30

	if size > maxExposure*1.01 {
		t.Errorf("Expected size <= total-exposure cap %f, got %f", maxExposure, size)
	}
	if size <= 0 {
		t.Errorf("Expected a positive size, got %f", size)
	}
}

func TestPositionSizer_ExistingPosition(t *testing.T) {
	ps := NewPositionSizer(propfirm.DefaultFTMOProfile())
	ps.SetKellyMultiplier(1.0)
	ps.UpdateMarketState(15.0, 50, 0)

	// Low confidence keeps the size below the total-exposure cap so the
	// existing-position correlation reduction is observable (otherwise both are
	// clamped to the same cap value).
	sizeNoExisting := ps.ComputeSize(0.1, 100000.0, "SPY", 0, 0)
	sizeWithExisting := ps.ComputeSize(0.1, 100000.0, "SPY", 0, 1000)

	if sizeWithExisting >= sizeNoExisting {
		t.Errorf("Expected reduced size with existing position, got %f >= %f", sizeWithExisting, sizeNoExisting)
	}
}

func TestPositionSizer_AllocationCap(t *testing.T) {
	ps := NewPositionSizer(propfirm.DefaultFTMOProfile())
	ps.UpdateMarketState(15.0, 50, 0)

	size := ps.ComputeSize(0.8, 100000.0, "SPY", 29900, 0)
	if size > 1000 {
		t.Errorf("Expected size capped near 30%% total exposure, got %f", size)
	}
}

func TestPositionSizer_NegativeSize(t *testing.T) {
	ps := NewPositionSizer(propfirm.DefaultFTMOProfile())
	ps.UpdateMarketState(40.0, 5, 3)

	size := ps.ComputeSize(0.05, 100000.0, "SPY", 50000, 50000)
	if size < 0 {
		t.Errorf("Size should never be negative, got %f", size)
	}
}

func TestRoundToLotSize(t *testing.T) {
	if r := RoundToLotSize(1500, 1000); r != 2000 {
		t.Errorf("1500 rounded to 1000-lot: expected 2000, got %.0f", r)
	}
	if r := RoundToLotSize(1400, 1000); r != 1000 {
		t.Errorf("1400 rounded to 1000-lot: expected 1000, got %.0f", r)
	}
	if r := RoundToLotSize(0, 1000); r != 0 {
		t.Errorf("zero size: expected 0, got %.0f", r)
	}
	if r := RoundToLotSize(500, 0); r != 500 {
		t.Errorf("zero lot size: expected passthrough 500, got %.0f", r)
	}
	if r := RoundToLotSize(-100, 100); r != 0 {
		t.Errorf("negative size: expected 0, got %.0f", r)
	}
	if r := RoundToLotSize(7.3, 1); r != 7 {
		t.Errorf("7.3 rounded to 1-lot: expected 7, got %.1f", r)
	}
}

func TestPositionSizer_LotRounding(t *testing.T) {
	ps := NewPositionSizer(propfirm.DefaultFTMOProfile())
	ps.SetLotSize(100)
	size := ps.ComputeSizeUncapped(0.8, 250, 0)
	if size != float64(int(size)) || int(size)%100 != 0 {
		t.Errorf("lot-rounded size should be multiple of 100, got %.4f", size)
	}
}

func TestPositionSizer_KellyViolationGuard(t *testing.T) {
	ps := NewPositionSizer(propfirm.DefaultFTMOProfile())

	// Default kelly should be 0.25
	ps.mu.RLock()
	defaultKelly := ps.kellyMult
	ps.mu.RUnlock()
	if defaultKelly != DefaultKellyMultiplier {
		t.Errorf("default kelly = %v, want %v", defaultKelly, DefaultKellyMultiplier)
	}

	// Kelly multiplier can be set, but sizing should respect it.
	ps.SetKellyMultiplier(0.55)
	size := ps.ComputeSizeUncapped(1.0, 100, 0)
	if size > 100 {
		t.Errorf("size with kelly=0.55 should be <= 100 (base), got %.4f", size)
	}

	// Reset to safe value.
	ps.SetKellyMultiplier(0.25)
	size2 := ps.ComputeSizeUncapped(1.0, 100, 0)
	if size2 > 100 {
		t.Errorf("size with kelly=0.25 should be <= 100, got %.4f", size2)
	}
}

func TestPositionSizer_ZeroBaseSize(t *testing.T) {
	ps := NewPositionSizer(propfirm.DefaultFTMOProfile())
	size := ps.ComputeSizeUncapped(1.0, 0, 0)
	if size != 0 {
		t.Errorf("zero base size should return 0, got %.4f", size)
	}
}

func TestPositionSizer_NegativeBaseSize(t *testing.T) {
	ps := NewPositionSizer(propfirm.DefaultFTMOProfile())
	size := ps.ComputeSizeUncapped(1.0, -50, 0)
	if size != 0 {
		t.Errorf("negative base size should return 0, got %.4f", size)
	}
}
