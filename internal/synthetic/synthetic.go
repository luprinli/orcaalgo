// Package synthetic provides shared per-symbol calibration and synthetic
// price-path generation for the dev seed (internal/db) and the engine's
// in-process fallback (internal/api). It mirrors scripts/stooq_synthetic.py:
// unconstrained geometric Brownian motion with a soft close anchor and full
// fixed-point precision (no intermediate decimal rounding).
package synthetic

import (
	"math"
	"math/rand/v2"

	"github.com/lee-econ/orca-core/internal/config"
)

// Calibration holds the per-symbol parameters needed to generate a realistic
// synthetic price path.
type Calibration struct {
	BasePrice  float64
	SigmaDaily float64
}

// defaultCalibration is the fallback used when a ticker has no explicit
// sigma_daily/base_price in configs/universe.json (legacy dev-seed behavior).
var defaultCalibration = map[string]Calibration{
	"SPY": {580, 0.008}, "QQQ": {480, 0.012}, "AAPL": {220, 0.014}, "MSFT": {440, 0.011},
	"GOOGL": {180, 0.013}, "META": {560, 0.016}, "AMZN": {220, 0.015},
	"NVDA": {120, 0.025}, "TSLA": {250, 0.028}, "VOO": {530, 0.008}, "DIA": {420, 0.007},
	"IWM": {210, 0.012}, "GLD": {220, 0.009}, "USO": {72, 0.015},
	"CL": {70, 0.018}, "NQ": {21000, 0.014}, "ES": {6000, 0.009}, "TLT": {88, 0.008},
	"EURUSD": {1.08, 0.005}, "GBPUSD": {1.28, 0.006}, "USDJPY": {148, 0.007},
	"USDCHF": {0.88, 0.005}, "AUDUSD": {0.66, 0.006}, "USDCAD": {1.36, 0.005},
	"NZDUSD": {0.60, 0.007}, "XAUUSD": {2350, 0.010}, "XAGUSD": {28, 0.016},
	"BTCUSD": {68000, 0.030}, "ETHUSD": {3400, 0.035}, "BTC-USD": {68000, 0.030},
	"ETH-USD": {3400, 0.035},
	"US30":    {41000, 0.007}, "SPX500": {5900, 0.008}, "NAS100": {21000, 0.012},
	"UK100": {8300, 0.008}, "GER40": {21000, 0.010}, "JPN225": {41000, 0.009},
	"^_US": {5900, 0.008}, "^DAX": {21000, 0.010},
}

// ForTicker returns the per-symbol base price and daily volatility, reading
// the canonical universe config first and falling back to built-in defaults
// when the ticker has no explicit sigma_daily. ok is false for unknown tickers.
func ForTicker(ticker string) (Calibration, bool) {
	if s, ok := config.SymbolByTicker(ticker); ok && s.SigmaDaily > 0 {
		base := s.BasePrice
		if base <= 0 {
			base = 100
		}
		return Calibration{BasePrice: base, SigmaDaily: s.SigmaDaily}, true
	}
	if c, ok := defaultCalibration[ticker]; ok {
		return c, true
	}
	return Calibration{}, false
}

// IntradayPath generates an unconstrained GBM intraday price path from open to
// a soft anchor at close, mirroring scripts/stooq_synthetic.py
// _generate_intraday_path. The path is free to break through any level (no
// clipping); a soft blend toward close is applied only in the final half of
// the session, and the path is floored near zero.
//
// nSteps is the number of 1-minute steps in the session (e.g. 1440 for a 24h
// session). The returned slice has nSteps prices.
func IntradayPath(rng *rand.Rand, openPrice, closePrice, sigmaDaily float64, nSteps int) []float64 {
	dt := 1.0 / float64(nSteps)
	path := make([]float64, nSteps)
	logPrice := math.Log(openPrice)
	for i := 0; i < nSteps; i++ {
		logPrice += sigmaDaily * math.Sqrt(dt) * rng.NormFloat64()
		path[i] = math.Exp(logPrice)
	}

	// Soft blend toward the daily close in the final portion of the session.
	const blendStartPct = 0.5
	const blendMaxWeight = 0.5
	mid := int(float64(nSteps) * blendStartPct)
	if mid < 1 {
		mid = 1
	}
	tailLen := nSteps - mid
	if tailLen > 0 && closePrice > 0 {
		for i := mid; i < nSteps; i++ {
			frac := float64(i-mid+1) / float64(tailLen)
			w := blendMaxWeight * frac
			target := path[mid] + (closePrice-path[mid])*frac
			path[i] = path[i]*(1.0-w) + target*w
		}
	}

	// Floor near zero to prevent negative prices in extreme σ scenarios.
	floor := openPrice * 0.01
	for i := range path {
		if path[i] < floor {
			path[i] = floor
		}
	}
	return path
}
