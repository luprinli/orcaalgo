package risk

import "sync"

// RegimeActivationMatrix defines which strategies are allowed in each HMM regime
// and what Kelly multiplier override to use. This is the single source of truth
// for regime-aware strategy gating, shared by backtest and live engines through
// the RiskPipeline.
//
// A nil Allowed[regime] entry means the strategy is blocked in that regime.
// A zero KellyMultiplier[regime] falls back to the pipeline's default KellyMult.
type RegimeActivationMatrix struct {
	mu sync.RWMutex

	// entries maps strategy ID → per-regime configuration.
	entries map[string]*RegimeEntry
}

// RegimeEntry holds the activation rules for one strategy across all 4 regimes.
type RegimeEntry struct {
	StrategyID string

	// Allowed[regime] == true → strategy is active in that regime.
	Allowed [4]bool

	// KellyMultiplier[regime] overrides the pipeline's default Kelly multiplier
	// for this regime. Zero means "use pipeline default". Applied after sizing
	// but before capital authorisation.
	KellyMultiplier [4]float64
}

// NewRegimeActivationMatrix returns a matrix populated with the default
// activation rules from the Senior Quantitative Review (§3.1, §5.2).
func NewRegimeActivationMatrix() *RegimeActivationMatrix {
	m := &RegimeActivationMatrix{
		entries: map[string]*RegimeEntry{
			// --- Primary strategies ---

			"grid_trading": {
				StrategyID:     "grid_trading",
				Allowed:        [4]bool{true, false, false, false},   // Calm only
				KellyMultiplier: [4]float64{0.25, 0, 0, 0},
			},
			"trend_following": {
				StrategyID:     "trend_following",
				Allowed:        [4]bool{false, true, false, false},   // Trending only
				KellyMultiplier: [4]float64{0, 0.25, 0, 0},
			},
			"session_scalp": {
				StrategyID:     "session_scalp",
				Allowed:        [4]bool{true, true, true, false},     // Calm, Trending, HighVol
				KellyMultiplier: [4]float64{0.25, 0.25, 0.15, 0},
			},
			"mean_reversion": {
				StrategyID:     "mean_reversion",
				Allowed:        [4]bool{true, false, false, false},   // Calm only
				KellyMultiplier: [4]float64{0.25, 0, 0, 0},
			},
		"opening_range_breakout": {
			StrategyID:     "opening_range_breakout",
			Allowed:        [4]bool{true, true, true, false},    // Calm, Trending, HighVol
			KellyMultiplier: [4]float64{0.10, 0.25, 0.15, 0},
		},
			"pairs_trading": {
				StrategyID:     "pairs_trading",
				Allowed:        [4]bool{true, false, true, false},    // Calm, HighVol
				KellyMultiplier: [4]float64{0.25, 0, 0.15, 0},
			},
			"stat_arb": {
				StrategyID:     "stat_arb",
				Allowed:        [4]bool{true, false, true, false},    // Calm, HighVol
				KellyMultiplier: [4]float64{0.25, 0, 0.15, 0},
			},
			"volatility_harvesting": {
				StrategyID:     "volatility_harvesting",
				Allowed:        [4]bool{false, false, true, false},   // HighVol only
				KellyMultiplier: [4]float64{0, 0, 0.15, 0},
			},

			// --- Alternative / complementary strategies (Phase 8) ---

			"dragon_trend": {
				StrategyID:     "dragon_trend",
				Allowed:        [4]bool{false, true, true, false},    // Trending, HighVol
				KellyMultiplier: [4]float64{0, 0.25, 0.15, 0},
			},
			"vwap_mr": {
				StrategyID:     "vwap_mr",
				Allowed:        [4]bool{true, false, false, false},   // Calm only
				KellyMultiplier: [4]float64{0.25, 0, 0, 0},
			},
			"volume_scalp": {
				StrategyID:     "volume_scalp",
				Allowed:        [4]bool{true, true, false, false},    // Calm, Trending
				KellyMultiplier: [4]float64{0.25, 0.25, 0, 0},
			},
		"orb_15m": {
			StrategyID:     "orb_15m",
			Allowed:        [4]bool{true, true, true, false},    // Calm, Trending, HighVol
			KellyMultiplier: [4]float64{0.10, 0.25, 0.15, 0},
		},
			"vix_futures_carry": {
				StrategyID:     "vix_futures_carry",
				Allowed:        [4]bool{false, false, true, false},
				KellyMultiplier: [4]float64{0, 0, 0.15, 0},
			},
			"vol_grid": {
				StrategyID:     "vol_grid",
				Allowed:        [4]bool{true, false, false, false},
				KellyMultiplier: [4]float64{0.15, 0, 0, 0},
			},
		"multi_asset_statarb": {
				StrategyID:     "multi_asset_statarb",
				Allowed:        [4]bool{true, false, true, false},    // Calm, HighVol
				KellyMultiplier: [4]float64{0.25, 0, 0.15, 0},
		},
			"rsi2_reversion": {
				StrategyID:     "rsi2_reversion",
				Allowed:        [4]bool{true, true, false, false},    // Calm, Trending
				KellyMultiplier: [4]float64{0.25, 0.25, 0, 0},
		},

		// === Permissive defaults for strategies not explicitly mapped ===
		// Any strategy not in the matrix falls through to the default entry
		// which allows all regimes with Kelly=0.25. This keeps existing
		// strategies (ichimoku, donchian, keltner_macd, ma_crossover,
		// rsi_divergence) running without being gated.
		},
	}
	return m
}

// defaultEntry is returned when a strategy is not explicitly mapped.
var defaultEntry = &RegimeEntry{
	StrategyID:     "default",
	Allowed:        [4]bool{true, true, true, true},
	KellyMultiplier: [4]float64{0.25, 0.25, 0.25, 0.25},
}

// Get returns the regime entry for a strategy, falling back to defaultEntry
// if the strategy is not explicitly mapped.
func (m *RegimeActivationMatrix) Get(strategyID string) *RegimeEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if entry, ok := m.entries[strategyID]; ok {
		return entry
	}
	return defaultEntry
}

// Set adds or overrides a strategy's regime entry.
func (m *RegimeActivationMatrix) Set(entry *RegimeEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries[entry.StrategyID] = entry
}

// IsAllowed returns whether the strategy is permitted to trade in the given regime.
func (m *RegimeActivationMatrix) IsAllowed(strategyID string, regime int8) bool {
	if regime < 0 || regime > 3 {
		return false
	}
	return m.Get(strategyID).Allowed[regime]
}

// KellyForRegime returns the Kelly multiplier override for the strategy in the
// given regime. Returns 0 if no override is set (caller should use pipeline default).
func (m *RegimeActivationMatrix) KellyForRegime(strategyID string, regime int8) float64 {
	if regime < 0 || regime > 3 {
		return 0
	}
	return m.Get(strategyID).KellyMultiplier[regime]
}

// AllStrategies returns all explicitly registered strategy IDs.
func (m *RegimeActivationMatrix) AllStrategies() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0, len(m.entries))
	for id := range m.entries {
		ids = append(ids, id)
	}
	return ids
}
