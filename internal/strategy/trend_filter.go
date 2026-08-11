package strategy

import "github.com/lee-econ/orca-core/internal/types"

// TrendFilter provides reusable trend-direction checks shared by multiple
// strategy runners. Defined once here to prevent copy-paste duplication of
// identical MA-based trend filter logic across grid, rsi2_reversion,
// session_scalp, and volume_scalp strategies.

type TrendFilter struct {
	Period int
	prices []float64
	idx    int
	count  int
}

func NewTrendFilter(period int) *TrendFilter {
	return &TrendFilter{
		Period: period,
		prices: make([]float64, period*2),
	}
}

func (f *TrendFilter) Push(price types.Price) {
	f.prices[f.idx%len(f.prices)] = price.Float64()
	f.idx++
	if f.count < len(f.prices) {
		f.count++
	}
}

func (f *TrendFilter) MA() float64 {
	if f.count < f.Period {
		return 0
	}
	n := f.count
	if n > f.Period {
		n = f.Period
	}
	start := f.idx - n
	if start < 0 {
		start += len(f.prices)
	}
	sum := 0.0
	for i := 0; i < n; i++ {
		pos := (start + i) % len(f.prices)
		sum += f.prices[pos]
	}
	return sum / float64(n)
}

// IsTrendAligned returns true when the side is aligned with the primary trend.
// For a BUY signal, trend is "aligned" when price is above the moving average
// (uptrend). For a SELL signal, trend is aligned when price is below the MA
// (downtrend). Returns true if insufficient data (warmup).
func (f *TrendFilter) IsTrendAligned(side string, price float64) bool {
	ma := f.MA()
	if ma == 0 {
		return true
	}
	switch side {
	case "BUY":
		return price >= ma
	case "SELL":
		return price <= ma
	default:
		return true
	}
}

func (f *TrendFilter) Ready() bool {
	return f.count >= f.Period
}
