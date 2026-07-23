package risk

import (
	"math"
	"sync"

	"github.com/lee-econ/orca-core/internal/propfirm"
)

type PositionSizer struct {
	mu          sync.RWMutex
	profile     *propfirm.Profile
	vix         float64
	sentiment   int
	regime      int8
	regimeScore float64
	kellyMult   float64
	perTradeCap float64
	totalExpCap float64
	useMLRegime bool
	lotSize     float64
}

func NewPositionSizer(profile *propfirm.Profile) *PositionSizer {
	ps := &PositionSizer{
		profile:     profile,
		kellyMult:   DefaultKellyMultiplier,
		perTradeCap: DefaultPerTradeCap,
		totalExpCap: DefaultTotalExpCap,
	}
	return ps
}

func (ps *PositionSizer) SetProfile(profile *propfirm.Profile) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.profile = profile
}

func (ps *PositionSizer) UpdateMarketState(vix float64, sentiment int, regime int8) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.vix = vix
	ps.sentiment = sentiment
	ps.regime = regime
}

func (ps *PositionSizer) SetRegimeScore(score float64) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.regimeScore = score
	ps.useMLRegime = true
}

func (ps *PositionSizer) SetKellyMultiplier(mult float64) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.kellyMult = mult
}

func (ps *PositionSizer) SetLotSize(lotSize float64) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.lotSize = lotSize
}

// RoundToLotSize rounds size to the nearest lot increment. A non-positive lotSize
// returns size unchanged. Returns 0 for size <= 0.
func RoundToLotSize(size, lotSize float64) float64 {
	if size <= 0 || lotSize <= 0 {
		return math.Max(0, size)
	}
	return math.Round(size/lotSize) * lotSize
}

func (ps *PositionSizer) ComputeSize(confidence float64, baseSize float64, symbol string, currentAllocation float64, existingPosition float64) float64 {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	size := ps.applyMultipliers(confidence, baseSize, existingPosition)
	if size <= 0 {
		return 0
	}

	// Total-exposure cap (capital semantics): callers that pass a capital-scale
	// baseSize and a running currentAllocation (e.g. the live CapitalPoolManager)
	// use this to keep cumulative exposure <= totalExpCap of capital. This clause
	// is dimensionally meaningful ONLY when baseSize represents capital; the engine
	// path (baseSize = share quantity, currentAllocation = 0) must use
	// ComputeSizeUncapped instead, or it would be capped to totalExpCap of its own
	// share count (a ~3x under-size).
	if currentAllocation+size > baseSize*ps.totalExpCap {
		size = baseSize*ps.totalExpCap - currentAllocation
		if size < 0 {
			size = 0
		}
	}

	return RoundToLotSize(math.Max(0, size), ps.lotSize)
}

// ComputeSizeUncapped applies the risk multipliers (confidence, regime, VIX,
// sentiment, correlation) WITHOUT the capital-scale total-exposure cap. Used by
// the backtest engine, whose baseSize is a share quantity already capped to 3% of
// account capital upstream (engine.go: positionPct <= 0.03). Applying the
// capital-semantics cap here would conflate share count with position value.
func (ps *PositionSizer) ComputeSizeUncapped(confidence float64, baseSize float64, existingPosition float64) float64 {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return RoundToLotSize(math.Max(0, ps.applyMultipliers(confidence, baseSize, existingPosition)), ps.lotSize)
}

// applyMultipliers is the shared sizing core: confidence, regime, VIX, sentiment,
// and correlation attenuators applied to baseSize. Caller must hold ps.mu.
func (ps *PositionSizer) applyMultipliers(confidence float64, baseSize float64, existingPosition float64) float64 {
	if baseSize <= 0 {
		return 0
	}

	p := ps.profile
	if p == nil {
		p = propfirm.DefaultFTMOProfile()
	}

	size := baseSize
	size *= math.Min(confidence, 1.0)

	if p.ConsistencyEnabled {
		if ps.useMLRegime {
			size *= math.Max(ps.regimeScore*RegimeMLScale, 0.0)
		} else {
			if ps.regime == 0 || ps.regime == 1 {
				size *= 1.0
			} else if ps.regime == 2 {
				size *= RegimeModerateMult
			} else {
				size *= RegimeCrisisMult
			}
		}
	}

	if ps.vix > DefaultVIXHigh {
		size *= VIXHighMult
	} else if ps.vix > DefaultVIXMid {
		size *= VIXMidMult
	} else if ps.vix > DefaultVIXLow {
		size *= VIXLowMult
	}

	if ps.sentiment < SentimentExtremeLow || ps.sentiment > SentimentExtremeHigh {
		size *= SentimentExtremeMult
	} else if ps.sentiment < SentimentModerateLow || ps.sentiment > SentimentModerateHigh {
		size *= SentimentModerateMult
	}

	if existingPosition > 0 {
		correlationFactor := 1.0 / (1.0 + existingPosition/baseSize)
		size *= correlationFactor
	}

	return size
}
