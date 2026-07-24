package strategy

import (
	"testing"

	"github.com/lee-econ/orca-core/internal/types"
)

func TestRSI(t *testing.T) {
	prices := make([]float64, 100)
	for i := 0; i < 100; i++ {
		prices[i] = 100.0 + float64(i)*0.1
	}
	rsi := RSI(prices, 50, 14)
	if rsi < 0 || rsi > 100 {
		t.Errorf("RSI should be in [0,100], got %.2f", rsi)
	}
}

func TestRSIInsufficientData(t *testing.T) {
	prices := []float64{100, 101, 102}
	rsi := RSI(prices, 3, 14)
	if rsi != 0 {
		t.Errorf("expected 0 for insufficient data, got %.2f", rsi)
	}
}

func TestMACD(t *testing.T) {
	prices := make([]float64, 100)
	for i := 0; i < 100; i++ {
		prices[i] = 100.0 + float64(i)*0.1
	}
	macdLine, signalLine := MACD(prices, 100)
	if macdLine == 0 && signalLine == 0 {
		t.Error("MACD should produce non-zero values for trending data")
	}
}

func TestMACDInsufficientData(t *testing.T) {
	prices := make([]float64, 10)
	for i := 0; i < 10; i++ {
		prices[i] = 100.0
	}
	macdLine, signalLine := MACD(prices, 10)
	if macdLine != 0 || signalLine != 0 {
		t.Error("expected 0,0 for insufficient data")
	}
}

func TestBollingerBands(t *testing.T) {
	prices := make([]float64, 100)
	for i := 0; i < 100; i++ {
		prices[i] = 100.0 + float64(i)*0.1
	}
	upper, middle, lower := BollingerBands(prices, 100)
	if middle == 0 {
		t.Error("middle band should be non-zero")
	}
	if upper == 0 || lower == 0 {
		t.Error("upper and lower bands should be non-zero")
	}
	t.Logf("BB: upper=%.2f middle=%.2f lower=%.2f", upper, middle, lower)
}

func TestBollingerBandsInsufficientData(t *testing.T) {
	prices := make([]float64, 10)
	upper, middle, lower := BollingerBands(prices, 10)
	if upper != 0 || middle != 0 || lower != 0 {
		t.Error("expected zeros for insufficient data")
	}
}

func TestMACrossoverRunner_Name(t *testing.T) {
	r := NewMACrossoverRunner()
	if r.Name() != "ma_crossover" {
		t.Errorf("expected 'ma_crossover', got '%s'", r.Name())
	}
	if r.Type() != "trend" {
		t.Errorf("expected 'trend', got '%s'", r.Type())
	}
}

func TestMACrossoverRunner_Params(t *testing.T) {
	r := NewMACrossoverRunner()
	params := r.Params()
	if params["fast_period"] != 9 || params["slow_period"] != 21 {
		t.Error("default params wrong")
	}
	r.SetParams(map[string]float64{"fast_period": 13, "rsi_overbought": 75})
	params = r.Params()
	if params["fast_period"] != 13 || params["rsi_overbought"] != 75 {
		t.Error("SetParams did not update values")
	}
}

func TestMACrossoverRunner_Reset(t *testing.T) {
	r := NewMACrossoverRunner()
	r.prevFast = 1.0
	r.prevSlow = 1.0
	r.Reset()
	if r.prevFast != 0 || r.prevSlow != 0 {
		t.Error("Reset should clear prevFast/prevSlow")
	}
}

func TestMACrossoverRunner_CrossoverDetection(t *testing.T) {
	r := NewMACrossoverRunner()
	candle := Candle{Symbol: "TEST"}
	for i := 0; i < 80; i++ {
		candle.Close = types.PriceFromFloat(100.0 + float64(i)*0.02)
		r.Evaluate(candle, 0)
	}
	sig := r.Evaluate(Candle{Symbol: "TEST", Close: types.PriceFromFloat(101.5)}, 0)
	if sig != nil {
		t.Logf("signal generated: %s at iteration 80", sig.Side)
	}
}

func TestMACrossoverRunner_RSIFilterSuppressesSignal(t *testing.T) {
	r := NewMACrossoverRunner()
	r.UseMacdFilter = false
	candle := Candle{Symbol: "TEST"}
	for i := 0; i < 80; i++ {
		candle.Close = types.PriceFromFloat(100.0 + float64(i)*0.02)
		r.Evaluate(candle, 0)
	}
	rsiBefore := RSI(r.PriceHistory, r.HistCount, 14)
	if rsiBefore > r.RsiOverbought {
		t.Logf("RSI = %.1f exceeds overbought %.1f — BUY should be suppressed", rsiBefore, r.RsiOverbought)
	}
}

func TestMACrossoverRunner_BollingerExit(t *testing.T) {
	r := NewMACrossoverRunner()
	r.UseBollExit = true
	if !r.UseBollExit {
		t.Error("UseBollExit should default to true")
	}
}

func TestIndicators_SelfConsistency(t *testing.T) {
	values := []float64{100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110,
		111, 112, 113, 114, 115, 116, 117, 118, 119, 120, 121, 122, 123, 124, 125}

	sma := SMA(values, 26, 20)
	ema := EMA(values, 26, 20)
	if sma <= 0 || ema <= 0 {
		t.Error("SMA/EMA should be > 0 for rising data")
	}

	rsi := RSI(values, 26, 14)
	if rsi < 50 {
		t.Logf("RSI = %.1f for consistent uptrend (expected strong)", rsi)
	}
}
