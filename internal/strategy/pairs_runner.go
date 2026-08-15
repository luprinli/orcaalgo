package strategy

import (
	"math"
	"sync"
	"time"

	"github.com/lee-econ/orca-core/internal/types"
)

// PairStatus holds the cached cointegration test results for a symbol pair.
type PairStatus struct {
	Primary    string
	Secondary  string
	HedgeRatio float64
	PValue     float64
	HalfLife   float64
	LastCheck  time.Time
	Valid      bool
}

// PairsRunner implements a cointegration-based pairs trading strategy.
// It computes the spread between two symbols using a cached hedge ratio
// and trades mean reversion on the spread z-score. The cointegration
// status is updated externally via SetPairStatus (daily batch job).
//
// The runner evaluates candles for the primary symbol only; the secondary
// symbol's price must be pushed via PushSecondaryPrice.
type PairsRunner struct {
	*BaseRunner

	// Pair configuration
	SecondarySymbol string

	// Cointegration state (updated externally)
	pairMu     sync.RWMutex
	pairStatus PairStatus

	// Entry/exit z-score thresholds
	EntryZ float64
	ExitZ  float64

	// Maximum holding period in bars
	MaxHold int

	// Minimum log-price correlation required to open new positions. This is a
	// backtest proxy for the live-path cointegration p-value gate: correlation
	// is a necessary (though not sufficient) condition for cointegration, so an
	// uncorrelated pair is skipped without a full ADF regression.
	MinPairCorrelation float64

	// Rolling window for spread computation
	SpreadLookback int

	// Internal state
	spreadHistory   []float64
	spreadHistIdx   int
	spreadHistCount int
	secPriceHistory []float64
	secHistIdx      int
	secHistCount    int
	barsHeld        int
}

func NewPairsRunner(primary, secondary string) *PairsRunner {
	return &PairsRunner{
		BaseRunner:         NewBaseRunner(256),
		SecondarySymbol:    secondary,
		EntryZ:             2.0,
		ExitZ:              0.5,
		MaxHold:            60,
		MinPairCorrelation: 0.5,
		SpreadLookback:     60,
		spreadHistory:      make([]float64, 256),
		secPriceHistory:    make([]float64, 256),
		pairStatus: PairStatus{
			Primary:   primary,
			Secondary: secondary,
			Valid:     false,
		},
	}
}

func (r *PairsRunner) Name() string              { return "pairs_trading" }
func (r *PairsRunner) Type() string              { return "pairs" }
func (r *PairsRunner) Version() (string, string) { return r.BaseRunner.Version() }

func (r *PairsRunner) Reset() {
	r.BaseRunner.Reset()
	r.spreadHistIdx = 0
	r.spreadHistCount = 0
	r.secHistIdx = 0
	r.secHistCount = 0
	r.barsHeld = 0
}

func (r *PairsRunner) Params() map[string]float64 {
	return map[string]float64{
		"entry_z":              r.EntryZ,
		"exit_z":               r.ExitZ,
		"max_hold":             float64(r.MaxHold),
		"spread_lookback":      float64(r.SpreadLookback),
		"min_pair_correlation": r.MinPairCorrelation,
	}
}

func (r *PairsRunner) SetParams(params map[string]float64) {
	if v, ok := params["entry_z"]; ok {
		r.EntryZ = v
	}
	if v, ok := params["exit_z"]; ok {
		r.ExitZ = v
	}
	if v, ok := params["max_hold"]; ok {
		r.MaxHold = int(v)
	}
	if v, ok := params["spread_lookback"]; ok {
		r.SpreadLookback = int(v)
	}
	if v, ok := params["min_pair_correlation"]; ok {
		r.MinPairCorrelation = v
	}
}

func (r *PairsRunner) ParamDefs() []ParamDef {
	return []ParamDef{
		{Name: "entry_z", Type: ParamContinuous, Default: 2.0, Min: 1.0, Max: 3.0, Step: 0.25, Group: "Signal", Description: "Spread z-score threshold for entry"},
		{Name: "exit_z", Type: ParamContinuous, Default: 0.5, Min: 0.0, Max: 1.5, Step: 0.25, Group: "Signal", Description: "Spread z-score threshold for exit"},
		{Name: "max_hold", Type: ParamInteger, Default: 60, Min: 10, Max: 200, Step: 10, Group: "Risk", Description: "Maximum holding period in bars"},
		{Name: "spread_lookback", Type: ParamInteger, Default: 60, Min: 20, Max: 120, Step: 10, Group: "Signal", Description: "Lookback window for spread mean/std"},
		{Name: "min_pair_correlation", Type: ParamContinuous, Default: 0.5, Min: 0.0, Max: 1.0, Step: 0.05, Group: "Filter", Description: "Minimum log-price correlation to open positions (cointegration proxy)"},
	}
}

// SetPairStatus updates the cointegration status. Called by the daily
// cointegration scanner. When Valid=false, the runner will not generate
// new entry signals.
func (r *PairsRunner) SetPairStatus(status PairStatus) {
	r.pairMu.Lock()
	defer r.pairMu.Unlock()
	r.pairStatus = status
}

// PairStatus returns the current cointegration status.
func (r *PairsRunner) PairStatus() PairStatus {
	r.pairMu.RLock()
	defer r.pairMu.RUnlock()
	return r.pairStatus
}

// PushSecondaryPrice records the secondary symbol's close price.
func (r *PairsRunner) PushSecondaryPrice(price types.Price) {
	idx := r.secHistIdx % 256
	r.secPriceHistory[idx] = price.Float64()
	r.secHistIdx++
	if r.secHistCount < 256 {
		r.secHistCount++
	}
}

// HedgeRatio returns the current hedge ratio from the pair status.
// Falls back to computing a simple OLS beta if no cached status exists.
func (r *PairsRunner) HedgeRatio() float64 {
	r.pairMu.RLock()
	defer r.pairMu.RUnlock()
	if r.pairStatus.Valid && r.pairStatus.HedgeRatio != 0 {
		return r.pairStatus.HedgeRatio
	}
	return r.computeSimpleHedge()
}

func (r *PairsRunner) computeSimpleHedge() float64 {
	n := r.HistCount
	if n < 20 || r.secHistCount < 20 {
		return 1.0
	}

	minN := n
	if r.secHistCount < minN {
		minN = r.secHistCount
	}

	var sumX, sumY, sumXY, sumX2 float64
	count := 0
	for i := 0; i < minN; i++ {
		pi := r.PriceHistory[i%r.BufferSize]
		si := r.secPriceHistory[i%256]
		if pi > 0 && si > 0 {
			x := math.Log(si)
			y := math.Log(pi)
			sumX += x
			sumY += y
			sumXY += x * y
			sumX2 += x * x
			count++
		}
	}
	if count < 5 {
		return 1.0
	}
	denom := sumX2*float64(count) - sumX*sumX
	if math.Abs(denom) < 1e-10 {
		return 1.0
	}
	return (sumXY*float64(count) - sumX*sumY) / denom
}

// pairLogCorrelation returns the Pearson correlation of log(primary) and
// log(secondary) over the available price history. Used as a backtest proxy for
// the cointegration check: uncorrelated pairs cannot be cointegrated.
func (r *PairsRunner) pairLogCorrelation() float64 {
	n := r.HistCount
	if n < 20 || r.secHistCount < 20 {
		return 0
	}
	minN := n
	if r.secHistCount < minN {
		minN = r.secHistCount
	}
	var sumX, sumY, sumXY, sumX2, sumY2 float64
	count := 0
	for i := 0; i < minN; i++ {
		pi := r.PriceHistory[i%r.BufferSize]
		si := r.secPriceHistory[i%256]
		if pi > 0 && si > 0 {
			x := math.Log(si)
			y := math.Log(pi)
			sumX += x
			sumY += y
			sumXY += x * y
			sumX2 += x * x
			sumY2 += y * y
			count++
		}
	}
	if count < 5 {
		return 0
	}
	num := sumXY*float64(count) - sumX*sumY
	denX := sumX2*float64(count) - sumX*sumX
	denY := sumY2*float64(count) - sumY*sumY
	den := math.Sqrt(denX * denY)
	if den <= 0 {
		return 0
	}
	return num / den
}

func (r *PairsRunner) Evaluate(candle Candle, regime int8) *Signal {
	price := candle.Close
	if price.IsZero() {
		return nil
	}

	r.PushPrice(price, candle.High, candle.Low, candle.Volume)

	if r.HistCount < r.SpreadLookback || r.secHistCount < r.SpreadLookback {
		return nil
	}

	// Check if cointegration is valid (skip if explicitly invalidated).
	pairOK := true
	r.pairMu.RLock()
	if r.pairStatus.Valid && r.pairStatus.PValue > 0.05 {
		pairOK = false
	}
	r.pairMu.RUnlock()
	if !pairOK {
		if r.PositionOpen {
			r.ClosePosition()
			return &Signal{Symbol: candle.Symbol, Side: invertSideForClose(r.CurrentSide), Action: SignalExit}
		}
		return nil
	}

	// Compute current spread using the hedge ratio.
	beta := r.HedgeRatio()
	if beta <= 0 {
		beta = 1.0
	}
	spread := math.Log(price.Float64()) - beta*math.Log(r.secPriceHistory[(r.secHistIdx-1)%256])
	if math.IsNaN(spread) || math.IsInf(spread, 0) {
		return nil
	}

	r.spreadHistory[r.spreadHistIdx%256] = spread
	r.spreadHistIdx++
	if r.spreadHistCount < 256 {
		r.spreadHistCount++
	}

	// Compute spread mean and std over lookback window.
	spreadMean, spreadStd := r.computeSpreadStats()
	if spreadStd <= 0 {
		return nil
	}
	zScore := (spread - spreadMean) / spreadStd

	// Exit management.
	if r.PositionOpen {
		r.barsHeld++
		exitZ := r.ExitZ

		// Time-based exit.
		if r.barsHeld >= r.MaxHold {
			r.ClosePosition()
			return &Signal{Symbol: candle.Symbol, Side: invertSideForClose(r.CurrentSide), Action: SignalExit}
		}

		// Exit when spread reverts to mean.
		if r.CurrentSide == "BUY" && zScore >= exitZ {
			r.ClosePosition()
			return &Signal{Symbol: candle.Symbol, Side: "SELL", Action: SignalExit}
		}
		if r.CurrentSide == "SELL" && zScore <= -exitZ {
			r.ClosePosition()
			return &Signal{Symbol: candle.Symbol, Side: "BUY", Action: SignalExit}
		}
		return nil
	}

	// Entry signals — gated by the cointegration proxy. Uncorrelated pairs are
	// skipped for new entries (but existing positions still exit normally above).
	if r.MinPairCorrelation > 0 && r.pairLogCorrelation() < r.MinPairCorrelation {
		return nil
	}

	if zScore <= -r.EntryZ {
		// Spread is below mean → go LONG primary (expect spread to widen).
		r.barsHeld = 0
		r.EntryPrice = price
		r.PositionOpen = true
		r.CurrentSide = "BUY"
		return &Signal{Symbol: candle.Symbol, Side: "BUY", Action: SignalEntry, Quantity: 1.0}
	}

	if zScore >= r.EntryZ {
		// Spread is above mean → go SHORT primary (expect spread to contract).
		r.barsHeld = 0
		r.EntryPrice = price
		r.PositionOpen = true
		r.CurrentSide = "SELL"
		return &Signal{Symbol: candle.Symbol, Side: "SELL", Action: SignalEntry, Quantity: 1.0}
	}

	return nil
}

func (r *PairsRunner) computeSpreadStats() (mean, stdDev float64) {
	n := r.spreadHistCount
	if n < r.SpreadLookback {
		return 0, 0
	}
	start := n - r.SpreadLookback
	if start < 0 {
		start = 0
	}
	var sum float64
	count := 0
	for i := start; i < n; i++ {
		idx := i % 256
		val := r.spreadHistory[idx]
		if !math.IsNaN(val) {
			sum += val
			count++
		}
	}
	if count < 2 {
		return 0, 0
	}
	mean = sum / float64(count)
	var variance float64
	for i := start; i < n; i++ {
		idx := i % 256
		val := r.spreadHistory[idx]
		if !math.IsNaN(val) {
			diff := val - mean
			variance += diff * diff
		}
	}
	stdDev = sampleStd(variance, count)
	return
}

func invertSideForClose(side string) string {
	if side == "BUY" {
		return "SELL"
	}
	return "BUY"
}

// Compile-time check that PairsRunner satisfies Strategy.
var _ Strategy = (*PairsRunner)(nil)
