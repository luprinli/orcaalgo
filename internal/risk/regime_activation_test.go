package risk

import (
	"testing"
)

func TestRegimeActivationMatrix_DefaultMapping(t *testing.T) {
	m := NewRegimeActivationMatrix()

	tests := []struct {
		strategy string
		regime   int8
		allowed  bool
		kelly    float64
	}{
		// Grid: Calm-only
		{"grid_trading", 0, true, 0.25},
		{"grid_trading", 1, false, 0},
		{"grid_trading", 2, false, 0},
		{"grid_trading", 3, false, 0},

		// Trend: Trending-only
		{"trend_following", 0, false, 0},
		{"trend_following", 1, true, 0.25},
		{"trend_following", 2, false, 0},
		{"trend_following", 3, false, 0},

		// Scalp: Calm, Trending, HighVol
		{"session_scalp", 0, true, 0.25},
		{"session_scalp", 1, true, 0.25},
		{"session_scalp", 2, true, 0.15},
		{"session_scalp", 3, false, 0},

		// MR: Calm-only
		{"mean_reversion", 0, true, 0.25},
		{"mean_reversion", 1, false, 0},
		{"mean_reversion", 2, false, 0},
		{"mean_reversion", 3, false, 0},

		// ORB: Trending, HighVol
		{"opening_range_breakout", 0, true, 0.10},
		{"opening_range_breakout", 1, true, 0.25},
		{"opening_range_breakout", 2, true, 0.15},
		{"opening_range_breakout", 3, false, 0},

		// Pairs: Calm, HighVol
		{"pairs_trading", 0, true, 0.25},
		{"pairs_trading", 1, false, 0},
		{"pairs_trading", 2, true, 0.15},
		{"pairs_trading", 3, false, 0},

		// Vol Harv: HighVol-only
		{"volatility_harvesting", 0, false, 0},
		{"volatility_harvesting", 1, false, 0},
		{"volatility_harvesting", 2, true, 0.15},
		{"volatility_harvesting", 3, false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.strategy+"_regime_"+string(rune('0'+tt.regime)), func(t *testing.T) {
			if got := m.IsAllowed(tt.strategy, tt.regime); got != tt.allowed {
				t.Errorf("IsAllowed(%s, %d) = %v, want %v", tt.strategy, tt.regime, got, tt.allowed)
			}
			if got := m.KellyForRegime(tt.strategy, tt.regime); got != tt.kelly {
				t.Errorf("KellyForRegime(%s, %d) = %v, want %v", tt.strategy, tt.regime, got, tt.kelly)
			}
		})
	}
}

func TestRegimeActivationMatrix_DefaultFallback(t *testing.T) {
	m := NewRegimeActivationMatrix()

	// Unknown strategies should fall through to permissive default.
	if !m.IsAllowed("unknown_strategy", 0) {
		t.Error("unknown strategy should be allowed by default")
	}
	if !m.IsAllowed("unknown_strategy", 3) {
		t.Error("unknown strategy should be allowed by default even in crisis")
	}
	if got := m.KellyForRegime("unknown_strategy", 0); got != 0.25 {
		t.Errorf("unknown strategy kelly = %v, want 0.25", got)
	}
}

func TestRegimeActivationMatrix_Override(t *testing.T) {
	m := NewRegimeActivationMatrix()

	entry := &RegimeEntry{
		StrategyID:      "grid_trading",
		Allowed:         [4]bool{true, false, false, false},
		KellyMultiplier: [4]float64{0.10, 0, 0, 0},
	}
	m.Set(entry)

	if !m.IsAllowed("grid_trading", 0) {
		t.Error("grid should still be allowed in Calm after override")
	}
	if got := m.KellyForRegime("grid_trading", 0); got != 0.10 {
		t.Errorf("grid kelly override = %v, want 0.10", got)
	}
}

func TestRegimeActivationMatrix_InvalidRegime(t *testing.T) {
	m := NewRegimeActivationMatrix()

	if m.IsAllowed("trend_following", -1) {
		t.Error("negative regime should return false")
	}
	if m.IsAllowed("trend_following", 4) {
		t.Error("regime 4 should return false (only 0-3 valid)")
	}
	if got := m.KellyForRegime("trend_following", -1); got != 0 {
		t.Error("negative regime kelly should be 0")
	}
}

func TestRegimeActivationMatrix_PipelineIntegration(t *testing.T) {
	m := NewRegimeActivationMatrix()
	sg := NewSignalGateImpl(
		NewVolatilityHalt(DefaultVolatilityHalt),
		NewPositionSizer(nil),
		NewExposureTracker(5.0, 0.25),
		NewOrderRateLimiter(DefaultOrderRateLimit),
	)
	sg.SetBacktestMode(true)
	sg.SetEquity(100000) // Required for exposure tracker

	p := &RiskPipeline{
		SignalGate:   sg,
		RegimeMatrix: m,
		KellyMult:    0.25,
	}

	// Grid in Calm → allowed, Kelly=0.25
	p.CurrentRegime = 0
	res := p.ProcessSignal(nil, ProcessSignalRequest{
		StrategyID:     "grid_trading",
		Symbol:         "SPY",
		Side:           "BUY",
		Price:          500.0,
		Confidence:     1.0,
		BaseSize:       4,   // shares: 2000 / 500
		RunningCapital: 100000,
	})
	if !res.Approved {
		t.Errorf("grid in Calm should be approved, got: %s", res.Reason)
	}

	// Grid in Trending → blocked
	p.CurrentRegime = 1
	res = p.ProcessSignal(nil, ProcessSignalRequest{
		StrategyID:     "grid_trading",
		Symbol:         "SPY",
		Side:           "BUY",
		Price:          500.0,
		Confidence:     1.0,
		BaseSize:       4,
		RunningCapital: 100000,
	})
	if res.Approved {
		t.Error("grid in Trending should be blocked")
	}

	// Scalp in HighVol → allowed, Kelly=0.15
	p.CurrentRegime = 2
	res = p.ProcessSignal(nil, ProcessSignalRequest{
		StrategyID:     "session_scalp",
		Symbol:         "SPY",
		Side:           "BUY",
		Price:          500.0,
		Confidence:     1.0,
		BaseSize:       4,
		RunningCapital: 100000,
	})
	if !res.Approved {
		t.Errorf("scalp in HighVol should be approved, got: %s", res.Reason)
	}
	// Size should be roughly: 1000 * 0.15 (Kelly) * confidence * regime * VIX multipliers
	// With default PositionSizer and no regime score set, regime multiplier = 1.0 (Calm/Trending default)
	// But wait — we're in HighVol (regime 2) and no regime score is set.
	// PositionSizer.applyMultipliers uses RegimeModerateMult=0.75 for regime 2.
	// So: 1000 * 1.0 (confidence) * 0.75 (regime) * 1.0 (VIX normal) * 0.15 (Kelly) = 112.5
	if res.Size <= 0 {
		t.Error("scalp in HighVol should have non-zero size")
	}
}
