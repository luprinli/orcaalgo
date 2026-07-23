package ml

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/lee-econ/orca-core/internal/strategy"
)

// FeatureStore computes 21-dim feature vectors for ML inference.
// Feature computation mirrors orca/ml/features.py exactly to prevent
// train/serve skew. All vectorized Python operations are replicated
// as explicit Go loops.
type FeatureStore struct {
	// Cache for pre-computed indicators
	prices [256]float64 // ring buffer of close prices
	highs  [256]float64
	lows   [256]float64
	volumes [256]float64
	count  int
}

// NewFeatureStore returns a new feature store with the given warmup data.
func NewFeatureStore(initialCloses, initialHighs, initialLows, initialVolumes []float64) *FeatureStore {
	fs := &FeatureStore{}
	n := len(initialCloses)
	if n > 256 {
		n = 256
	}
	for i := 0; i < n; i++ {
		fs.prices[i] = initialCloses[i]
		if i < len(initialHighs) {
			fs.highs[i] = initialHighs[i]
		}
		if i < len(initialLows) {
			fs.lows[i] = initialLows[i]
		}
		if i < len(initialVolumes) {
			fs.volumes[i] = initialVolumes[i]
		}
	}
	fs.count = n
	return fs
}

// Push adds a new bar to the ring buffer.
func (fs *FeatureStore) Push(c strategy.Candle) {
	idx := fs.count % 256
	fs.prices[idx] = c.Close
	fs.highs[idx] = c.High
	fs.lows[idx] = c.Low
	fs.volumes[idx] = c.Volume
	fs.count++
}

// Compute computes the full 21-dim feature vector from the current buffer state.
// This is the canonical Go implementation. Must produce bit-identical results
// to orca/ml/features.py compute_full_feature_vector for the same inputs.
func (fs *FeatureStore) Compute(
	ts time.Time,
	hmmAlpha [4]float64,
	hmmConfidence float64,
	signalType int,
	signalStrength float64,
	cvdDivergence float64,
	spreadPct float64,
) (*FeatureVector, error) {
	if fs.count < 40 {
		return nil, fmt.Errorf("insufficient data: %d bars (need 40)", fs.count)
	}

	n := fs.count
	if n > 256 {
		n = 256
	}

	// Extract working slices from ring buffer
	closes := make([]float64, n)
	highs := make([]float64, n)
	lows := make([]float64, n)
	volumes := make([]float64, n)
	for i := 0; i < n; i++ {
		idx := (fs.count - n + i) % 256
		closes[i] = fs.prices[idx]
		highs[i] = fs.highs[idx]
		lows[i] = fs.lows[idx]
		volumes[i] = fs.volumes[idx]
	}

	fv := FeatureVector{}

	// ── Price features (indices 0-4) ───────────────────────────────────────────
	current := closes[n-1]

	// 0: ret1
	if closes[n-2] > 0 {
		fv[0] = float32(math.Log(current / closes[n-2]))
	}

	// 1: ret5
	if n >= 6 && closes[n-6] > 0 {
		fv[1] = float32(math.Log(current / closes[n-6]))
	}

	// 2: ret20
	if n >= 21 && closes[n-21] > 0 {
		fv[2] = float32(math.Log(current / closes[n-21]))
	}

	// 3: volatility20 — EWMA of log returns (span=20, alpha=2/21)
	if n >= 22 {
		vol := computeEWMAVolatility(closes[n-22:], 20)
		fv[3] = float32(vol)
	}

	// 4: atr_ratio — True Range ATR(14) / Close
	atr := strategy.TrueRangeATR(highs, lows, closes, n, 14)
	if current > 0 && atr > 0 {
		fv[4] = float32(atr / current)
	}

	// ── Indicator features (indices 5-11) ──────────────────────────────────────

	// 5: rsi14
	fv[5] = float32(computeRSI(closes, 14))

	// 6: macd_hist
	ema12 := computeEMA(closes, 12)
	ema26 := computeEMA(closes, 26)
	macdLine := ema12 - ema26
	// Signal line: 9-period EMA of MACD (simple mean for now)
	macdSeries := computeMACDSeries(closes, 12, 26)
	signalLine := computeEMA(macdSeries, 9)
	fv[6] = float32(macdLine - signalLine)

	// 7: adx14
	fv[7] = float32(computeADX(highs, lows, closes, 14))

	// 8: bb_percent_b — Bollinger %B(20, 2)
	sma20 := computeSMA(closes, 20)
	std20 := computeStdDev(closes, sma20, 20)
	if std20 > 0 {
		lower := sma20 - 2*std20
		fv[8] = float32((current - lower) / (4 * std20))
	} else {
		fv[8] = 0.5
	}

	// 9: volume_ratio
	avgVol := computeSMA(volumes[:n-1], 20)
	if avgVol > 0 {
		fv[9] = float32(volumes[n-1] / avgVol)
	} else {
		fv[9] = 1.0
	}

	// 10: cvd_divergence
	fv[10] = float32(cvdDivergence)

	// 11: spread_pct
	fv[11] = float32(spreadPct)

	// ── Regime features (indices 12-16) ────────────────────────────────────────
	fv[12] = float32(hmmAlpha[0])
	fv[13] = float32(hmmAlpha[1])
	fv[14] = float32(hmmAlpha[2])
	fv[15] = float32(hmmAlpha[3])
	fv[16] = float32(hmmConfidence)

	// ── Signal features (indices 17-18) ────────────────────────────────────────
	fv[17] = float32(signalType)
	fv[18] = float32(signalStrength)

	// ── Time features (indices 19-20) ──────────────────────────────────────────
	hour := float64(ts.Hour()) + float64(ts.Minute())/60.0 + float64(ts.Second())/3600.0
	fv[19] = float32(math.Sin(2 * math.Pi * hour / 24.0))
	fv[20] = float32(math.Cos(2 * math.Pi * hour / 24.0))

	return &fv, nil
}

// Validate checks the feature vector for NaN, Inf, and dimension.
func (fv *FeatureVector) Validate() bool {
	for i := 0; i < FeatureDim; i++ {
		v := fv[i]
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return false
		}
	}
	return true
}

// ToSlice converts the feature vector to a float32 slice for ONNX inference.
func (fv *FeatureVector) ToSlice() []float32 {
	return fv[:]
}

// ── Internal computation helpers (must match Python exactly) ───────────────────

// computeEWMAVolatility delegates to Python canonical implementation in
// orca/sizing/volatility.py via ewma_bridge.go. Hard Prohibition #1: EWMA
// must not be reimplemented in Go.
func computeEWMAVolatility(logPrices []float64, span int) float64 {
	n := len(logPrices)
	if n < 2 {
		return 0.01
	}
	logReturns := make([]float64, n-1)
	for i := 1; i < n; i++ {
		if logPrices[i-1] > 0 {
			logReturns[i-1] = math.Log(logPrices[i] / logPrices[i-1])
		}
	}
	vol, err := ComputeEWMAVolatility(context.Background(), logReturns, span)
	if err != nil {
		return 0.01
	}
	return vol
}

func computeRSI(closes []float64, period int) float64 {
	n := len(closes)
	if n < period+1 {
		return 50.0
	}
	var sumGain, sumLoss float64
	for i := n - period; i < n; i++ {
		delta := closes[i] - closes[i-1]
		if delta > 0 {
			sumGain += delta
		} else {
			sumLoss -= delta
		}
	}
	avgGain := sumGain / float64(period)
	avgLoss := sumLoss / float64(period)
	if avgLoss < 1e-12 {
		return 100.0
	}
	rs := avgGain / avgLoss
	return 100.0 - 100.0/(1.0+rs)
}

func computeEMA(series []float64, period int) float64 {
	n := len(series)
	if n == 0 {
		return 0
	}
	if period <= 0 {
		period = 1
	}
	alpha := 2.0 / float64(period+1)
	result := series[0]
	for i := 1; i < n; i++ {
		result = alpha*series[i] + (1-alpha)*result
	}
	return result
}

func computeMACDSeries(closes []float64, fast, slow int) []float64 {
	n := len(closes)
	if n < slow {
		return nil
	}
	result := make([]float64, n)
	emaFast := closes[0]
	emaSlow := closes[0]
	alphaFast := 2.0 / float64(fast+1)
	alphaSlow := 2.0 / float64(slow+1)
	for i := 1; i < n; i++ {
		emaFast = alphaFast*closes[i] + (1-alphaFast)*emaFast
		emaSlow = alphaSlow*closes[i] + (1-alphaSlow)*emaSlow
		result[i] = emaFast - emaSlow
	}
	return result
}

func computeADX(highs, lows, closes []float64, period int) float64 {
	n := len(highs)
	if n < period*2 {
		return 25.0
	}
	tr := make([]float64, n)
	plusDM := make([]float64, n)
	minusDM := make([]float64, n)

	for i := 1; i < n; i++ {
		highLow := highs[i] - lows[i]
		highClose := math.Abs(highs[i] - closes[i-1])
		lowClose := math.Abs(lows[i] - closes[i-1])
		tr[i] = math.Max(highLow, math.Max(highClose, lowClose))

		upMove := highs[i] - highs[i-1]
		downMove := lows[i-1] - lows[i]
		if upMove > downMove && upMove > 0 {
			plusDM[i] = upMove
		}
		if downMove > upMove && downMove > 0 {
			minusDM[i] = downMove
		}
	}

	atrVal := computeEMA(tr, period)
	smPlus := computeEMA(plusDM, period)
	smMinus := computeEMA(minusDM, period)

	var plusDI, minusDI float64
	if atrVal > 0 {
		plusDI = 100.0 * smPlus / atrVal
		minusDI = 100.0 * smMinus / atrVal
	}

	diSum := plusDI + minusDI
	if diSum < 1e-12 {
		return 0.0
	}
	dx := math.Abs(plusDI-minusDI) / diSum
	return dx * 100.0
}

func computeSMA(series []float64, period int) float64 {
	n := len(series)
	if n == 0 {
		return 0
	}
	start := n - period
	if start < 0 {
		start = 0
	}
	sum := 0.0
	count := 0
	for i := start; i < n; i++ {
		sum += series[i]
		count++
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func computeStdDev(series []float64, mean float64, period int) float64 {
	n := len(series)
	start := n - period
	if start < 0 {
		start = 0
	}
	sumSq := 0.0
	count := 0
	for i := start; i < n; i++ {
		diff := series[i] - mean
		sumSq += diff * diff
		count++
	}
	if count < 2 {
		return 0
	}
	return math.Sqrt(sumSq / float64(count))
}
