package strategy

import (
	"math"
	"testing"
)

// Golden/property tests for the canonical indicator library (Rule 1). Manual
// indicators get hand-computed golden values; cinar-backed wrappers get
// invariant properties (constant → constant, direction, insufficient-data → 0).

func TestMean_Golden(t *testing.T) {
	got := Mean([]float64{1, 2, 3, 4, 5}, 5, 5)
	if got != 3.0 {
		t.Errorf("Mean = %v, want 3.0", got)
	}
}

func TestZScore_Golden(t *testing.T) {
	if got := ZScore(15, 10, 2); got != 2.5 {
		t.Errorf("ZScore = %v, want 2.5", got)
	}
	if got := ZScore(5, 10, 0); got != 0 {
		t.Errorf("ZScore with std=0 = %v, want 0", got)
	}
}

func TestChoppinessIndex_Golden(t *testing.T) {
	// Monotonic trend: sum(TR) == range, so CHOP == 0.
	highs := []float64{10, 11, 12, 13, 14}
	lows := []float64{9, 10, 11, 12, 13}
	prices := []float64{9.5, 10.5, 11.5, 12.5, 13.5}
	if got := ChoppinessIndex(prices, highs, lows, 5, 5); got != 0.0 {
		t.Errorf("ChoppinessIndex(trend) = %v, want 0", got)
	}
}

func TestADX_TrendHighFlatZero(t *testing.T) {
	trendHighs := []float64{10, 11, 12, 13, 14, 15, 16, 17, 18, 19}
	trendLows := []float64{9, 10, 11, 12, 13, 14, 15, 16, 17, 18}
	trendPrices := []float64{9.5, 10.5, 11.5, 12.5, 13.5, 14.5, 15.5, 16.5, 17.5, 18.5}
	if got := ADX(trendPrices, trendHighs, trendLows, 10, 5); got <= 25 {
		t.Errorf("ADX(trend) = %v, want > 25", got)
	}
	flat := make([]float64, 10)
	for i := range flat {
		flat[i] = 10
	}
	if got := ADX(flat, flat, flat, 10, 5); got != 0 {
		t.Errorf("ADX(flat) = %v, want 0", got)
	}
}

func TestConstantSeriesInvariants(t *testing.T) {
	const n = 60
	close := make([]float64, n)
	high := make([]float64, n)
	low := make([]float64, n)
	vol := make([]float64, n)
	for i := range close {
		close[i] = 100
		high[i] = 101
		low[i] = 99
		vol[i] = 1000
	}

	if got := EMA(close, n, 10); got != 100 {
		t.Errorf("EMA(constant) = %v, want 100", got)
	}
	if got := SMA(close, n, 10); got != 100 {
		t.Errorf("SMA(constant) = %v, want 100", got)
	}
	if got := StdDev(close, n, 10); got != 0 {
		t.Errorf("StdDev(constant) = %v, want 0", got)
	}
	if macd, signal := MACD(close, n); macd != 0 || signal != 0 {
		t.Errorf("MACD(constant) = (%v, %v), want (0, 0)", macd, signal)
	}
	if upper, middle, lower := BollingerBands(close, n); upper != middle || lower != middle {
		t.Errorf("BollingerBands(constant) upper/middle/lower = %v/%v/%v, want equal", upper, middle, lower)
	}
	if got := ForceIndex(close, vol, n); math.IsNaN(got) {
		t.Errorf("ForceIndex(constant) = NaN")
	}
	if got := OBV(close, vol, n); got != 0 {
		t.Errorf("OBV(constant) = %v, want 0", got)
	}
	if got := VWAP(close, vol, n, 10); got != 100 {
		t.Errorf("VWAP(constant) = %v, want 100", got)
	}
}

func TestForceIndex_Direction(t *testing.T) {
	const n = 30
	closes := make([]float64, n)
	volumes := make([]float64, n)
	for i := range closes {
		closes[i] = 100 + float64(i) // monotonic up
		volumes[i] = 1000
	}
	if got := ForceIndex(closes, volumes, n); got <= 0 {
		t.Errorf("ForceIndex(up) = %v, want > 0", got)
	}
}

func TestRSI_Direction(t *testing.T) {
	const n = 60
	up := make([]float64, n)
	down := make([]float64, n)
	for i := range up {
		up[i] = 100 + float64(i) // monotonic up
		down[i] = 100 - float64(i)
	}
	if got := RSI(up, n, 14); got < 90 {
		t.Errorf("RSI(monotonic up) = %v, want > 90", got)
	}
	if got := RSI(down, n, 14); got > 10 {
		t.Errorf("RSI(monotonic down) = %v, want < 10", got)
	}
	if got := RSI2(up, n); got < 95 {
		t.Errorf("RSI2(monotonic up) = %v, want > 95", got)
	}
}

func TestInsufficientDataReturnsZero(t *testing.T) {
	small := []float64{1, 2, 3}
	// Every wrapper must return 0 (or zero value) below its minimum period.
	if got := EMA(small, 3, 20); got != 0 {
		t.Errorf("EMA insufficient = %v, want 0", got)
	}
	if got := SMA(small, 3, 20); got != 0 {
		t.Errorf("SMA insufficient = %v, want 0", got)
	}
	if got := RSI(small, 3, 14); got != 0 {
		t.Errorf("RSI insufficient = %v, want 0", got)
	}
	if got := RSI2(small, 3); got != 0 {
		t.Errorf("RSI2 insufficient = %v, want 0", got)
	}
	if macd, signal := MACD(small, 3); macd != 0 || signal != 0 {
		t.Errorf("MACD insufficient = (%v, %v), want (0, 0)", macd, signal)
	}
	if _, middle, _ := BollingerBands(small, 3); middle != 0 {
		t.Errorf("BollingerBands insufficient middle = %v, want 0", middle)
	}
	if k, d := StochasticOscillator(small, small, small, 3); k != 0 || d != 0 {
		t.Errorf("StochasticOscillator insufficient = (%v, %v), want (0, 0)", k, d)
	}
	tenkan, _, _, _, _ := IchimokuCloud(small, small, small, 3)
	if tenkan != 0 {
		t.Errorf("IchimokuCloud insufficient tenkan = %v, want 0", tenkan)
	}
	if got := WilliamsR(small, small, small, 3); got != 0 {
		t.Errorf("WilliamsR insufficient = %v, want 0", got)
	}
	if up, down := Aroon(small, small, 3); up != 0 || down != 0 {
		t.Errorf("Aroon insufficient = (%v, %v), want (0, 0)", up, down)
	}
	if got := MFI(small, small, small, small, 3, 14); got != 0 {
		t.Errorf("MFI insufficient = %v, want 0", got)
	}
	if got := CMF(small, small, small, small, 3); got != 0 {
		t.Errorf("CMF insufficient = %v, want 0", got)
	}
	if longExit, shortExit := ChandelierExit(small, small, small, 3); longExit != 0 || shortExit != 0 {
		t.Errorf("ChandelierExit insufficient = (%v, %v), want (0, 0)", longExit, shortExit)
	}
}

func TestDonchianKeltner_Constant(t *testing.T) {
	const n = 30
	// A truly flat series (high == low == close) collapses every band to the middle.
	flat := make([]float64, n)
	for i := range flat {
		flat[i] = 100
	}
	if upper, middle, lower := DonchianChannel(flat, n, 20); upper != middle || lower != middle {
		t.Errorf("DonchianChannel(flat) upper/middle/lower = %v/%v/%v, want equal", upper, middle, lower)
	}
	if upper, middle, lower := KeltnerChannel(flat, flat, flat, n, 20); upper != middle || lower != middle {
		t.Errorf("KeltnerChannel(flat) upper/middle/lower = %v/%v/%v, want equal", upper, middle, lower)
	}
}

func TestFinite(t *testing.T) {
	if finite(math.NaN()) || finite(math.Inf(1)) || finite(math.Inf(-1)) {
		t.Error("finite should reject NaN/Inf")
	}
	if !finite(0.0) || !finite(100.0) || !finite(-3.5) {
		t.Error("finite should accept ordinary numbers")
	}
}
