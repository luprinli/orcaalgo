package risk

import (
	"math"
	"sync"
	"time"
)

type OrderRateLimiter struct {
	mu          sync.Mutex
	maxPerSec   float64
	windowSize  time.Duration
	orders      map[string][]time.Time
}

func NewOrderRateLimiter(maxPerSec float64) *OrderRateLimiter {
	return &OrderRateLimiter{
		maxPerSec:  maxPerSec,
		windowSize: 1 * time.Second,
		orders:     make(map[string][]time.Time),
	}
}

func (l *OrderRateLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.windowSize)

	recent := l.orders[key]
	keep := recent[:0]
	for _, t := range recent {
		if t.After(cutoff) {
			keep = append(keep, t)
		}
	}
	l.orders[key] = keep

	if float64(len(keep)) >= l.maxPerSec {
		return false
	}

	l.orders[key] = append(l.orders[key], now)
	return true
}

func (l *OrderRateLimiter) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.orders = make(map[string][]time.Time)
}

type VolatilityHalt struct {
	mu            sync.Mutex
	returns       []float64
	maxHistory    int
	zThreshold    float64
	isHalted      bool
	haltReason    string
}

func NewVolatilityHalt(zThreshold float64) *VolatilityHalt {
	return &VolatilityHalt{
		returns:    make([]float64, 0, 256),
		maxHistory: 252,
		zThreshold: zThreshold,
	}
}

func (v *VolatilityHalt) Update(price float64) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if len(v.returns) >= 2 {
		prev := v.returns[len(v.returns)-1]
		prevPrice := 0.0
		for i := len(v.returns) - 1; i >= 0; i-- {
			if v.returns[i] != 0 {
				prevPrice = price / (1 + v.returns[i])
				break
			}
		}
		_ = prev
		if prevPrice > 0 {
			ret := (price - prevPrice) / prevPrice
			v.returns = append(v.returns, ret)
		}
	}

	if len(v.returns) > v.maxHistory {
		v.returns = v.returns[len(v.returns)-v.maxHistory:]
	}

	if len(v.returns) >= 10 {
		mean := 0.0
		n := len(v.returns)
		for _, r := range v.returns {
			mean += r
		}
		mean /= float64(n)

		variance := 0.0
		for _, r := range v.returns {
			diff := r - mean
			variance += diff * diff
		}
		variance /= float64(n - 1)
		std := math.Sqrt(variance)

		if std > 0 {
			latestRet := v.returns[len(v.returns)-1]
			z := (latestRet - mean) / std
			if math.Abs(z) > v.zThreshold {
				v.isHalted = true
				v.haltReason = "volatility_z"
			}
		}
	}
}

func (v *VolatilityHalt) UpdateReturn(ret float64) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.returns = append(v.returns, ret)
	if len(v.returns) > v.maxHistory {
		v.returns = v.returns[len(v.returns)-v.maxHistory:]
	}
	if len(v.returns) >= 10 {
		mean := 0.0
		n := len(v.returns)
		for _, r := range v.returns {
			mean += r
		}
		mean /= float64(n)

		variance := 0.0
		for _, r := range v.returns {
			diff := r - mean
			variance += diff * diff
		}
		variance /= float64(n - 1)
		std := math.Sqrt(variance)

		if std > 0 {
			latestRet := v.returns[len(v.returns)-1]
			z := (latestRet - mean) / std
			if math.Abs(z) > v.zThreshold {
				v.isHalted = true
				v.haltReason = "volatility_z"
			} else {
				v.isHalted = false
				v.haltReason = ""
			}
		}
	}
}

func (v *VolatilityHalt) IsHalted() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.isHalted
}

func (v *VolatilityHalt) HaltReason() string {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.haltReason
}

func (v *VolatilityHalt) Reset() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.returns = v.returns[:0]
	v.isHalted = false
	v.haltReason = ""
}

type ExposureTracker struct {
	mu               sync.Mutex
	totalLong        float64
	totalShort       float64
	symbolLong       map[string]float64
	symbolShort      map[string]float64
	maxLeverage      float64
	maxSymbolGross   float64
	equity           float64
}

func NewExposureTracker(maxLeverage, maxSymbolGross float64) *ExposureTracker {
	return &ExposureTracker{
		symbolLong:     make(map[string]float64),
		symbolShort:    make(map[string]float64),
		maxLeverage:    maxLeverage,
		maxSymbolGross: maxSymbolGross,
	}
}

func (e *ExposureTracker) SetEquity(equity float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.equity = equity
}

func (e *ExposureTracker) CheckOrder(symbol, side string, notional float64) (bool, string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.equity <= 0 {
		return false, "equity_negative"
	}

	currentLong := e.totalLong
	currentShort := e.totalShort
	symLong := e.symbolLong[symbol]
	symShort := e.symbolShort[symbol]

	if side == "BUY" {
		currentLong += notional
		symLong += notional
	} else {
		currentShort += notional
		symShort += notional
	}

	grossExposure := currentLong + currentShort
	if e.maxLeverage > 0 && grossExposure/e.equity > e.maxLeverage {
		return false, "max_leverage"
	}

	if e.maxSymbolGross > 0 {
		symGross := symLong + symShort
		if symGross/e.equity > e.maxSymbolGross {
			return false, "max_symbol_gross_exposure"
		}
	}

	return true, ""
}

func (e *ExposureTracker) AddPosition(symbol, side string, notional float64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if side == "BUY" {
		e.totalLong += notional
		e.symbolLong[symbol] += notional
	} else {
		e.totalShort += notional
		e.symbolShort[symbol] += notional
	}
}

func (e *ExposureTracker) RemovePosition(symbol, side string, notional float64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if side == "BUY" {
		e.totalLong -= notional
		e.symbolLong[symbol] -= notional
		if e.symbolLong[symbol] < 0 {
			e.symbolLong[symbol] = 0
		}
	} else {
		e.totalShort -= notional
		e.symbolShort[symbol] -= notional
		if e.symbolShort[symbol] < 0 {
			e.symbolShort[symbol] = 0
		}
	}
}

func (e *ExposureTracker) GrossExposure() float64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.totalLong + e.totalShort
}
