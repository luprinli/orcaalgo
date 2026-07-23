package strategy_test

import (
	"testing"

	strategy "github.com/lee-econ/orca-core/internal/strategy"
)

func TestATR_Basic(t *testing.T) {
	prices := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	result := strategy.ATR(prices, 5, 4)
	expected := 1.0
	if result != expected {
		t.Errorf("ATR = %v, want %v", result, expected)
	}
}

func TestATR_InsufficientData(t *testing.T) {
	prices := []float64{1.0, 2.0}
	result := strategy.ATR(prices, 2, 5)
	if result != 0 {
		t.Errorf("ATR with insufficient data = %v, want 0", result)
	}
}

func TestATR_ZeroPeriod(t *testing.T) {
	prices := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	result := strategy.ATR(prices, 5, 0)
	if result != 0 {
		t.Errorf("ATR with zero period = %v, want 0", result)
	}
}

func TestMean_Basic(t *testing.T) {
	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	result := strategy.Mean(values, 5, 5)
	expected := 3.0
	if result != expected {
		t.Errorf("Mean = %v, want %v", result, expected)
	}
}

func TestMean_ShorterLookback(t *testing.T) {
	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	result := strategy.Mean(values, 5, 3)
	expected := 4.0
	if result != expected {
		t.Errorf("Mean with lookback 3 = %v, want %v", result, expected)
	}
}

func TestMean_Empty(t *testing.T) {
	result := strategy.Mean(nil, 0, 5)
	if result != 0 {
		t.Errorf("Mean with nil = %v, want 0", result)
	}
}

func TestEMA_Basic(t *testing.T) {
	values := []float64{10.0, 10.0, 10.0, 10.0, 10.0}
	result := strategy.EMA(values, 5, 5)
	if result != 10.0 {
		t.Errorf("EMA constant = %v, want 10.0", result)
	}
}

func TestStdDev_Basic(t *testing.T) {
	values := []float64{2.0, 4.0, 4.0, 4.0, 5.0, 5.0, 7.0, 9.0}
	result := strategy.StdDev(values, 8, 8)
	if result <= 0 {
		t.Errorf("StdDev = %v, expected > 0", result)
	}
}

func TestStdDev_InsufficientData(t *testing.T) {
	values := []float64{1.0}
	result := strategy.StdDev(values, 1, 5)
	if result != 0 {
		t.Errorf("StdDev = %v, want 0", result)
	}
}

func TestZScore_Basic(t *testing.T) {
	result := strategy.ZScore(15.0, 10.0, 2.0)
	expected := 2.5
	if result != expected {
		t.Errorf("ZScore = %v, want %v", result, expected)
	}
}

func TestZScore_ZeroStd(t *testing.T) {
	result := strategy.ZScore(15.0, 10.0, 0.0)
	if result != 0 {
		t.Errorf("ZScore with zero std = %v, want 0", result)
	}
}

func TestADX_FlatPrices(t *testing.T) {
	prices := make([]float64, 60)
	highs := make([]float64, 60)
	lows := make([]float64, 60)
	for i := range prices {
		prices[i] = 100.0
		highs[i] = 100.0
		lows[i] = 100.0
	}
	result := strategy.ADX(prices, highs, lows, 60, 14)
	if result != 0 {
		t.Errorf("ADX flat = %v, want 0", result)
	}
}

func TestADX_InsufficientData(t *testing.T) {
	prices := make([]float64, 10)
	result := strategy.ADX(prices, prices, prices, 10, 14)
	if result != 0 {
		t.Errorf("ADX insufficient = %v, want 0", result)
	}
}
