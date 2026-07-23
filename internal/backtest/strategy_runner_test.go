package backtest

import (
	"testing"
	"time"

	strategy "github.com/lee-econ/orca-core/internal/strategy"
)

func TestNewStrategyRunner_Backtest(t *testing.T) {
	sr := strategy.NewMeanReversionRunner(20, 2.5, 0.3, 5)
	if sr == nil {
		t.Fatal("Expected non-nil runner")
	}
	if sr.Lookback != 20 {
		t.Errorf("Expected lookback 20, got %d", sr.Lookback)
	}
	if sr.EntryZ != 2.5 {
		t.Errorf("Expected entryZ 2.5, got %f", sr.EntryZ)
	}
	if sr.ExitZ != 0.3 {
		t.Errorf("Expected exitZ 0.3, got %f", sr.ExitZ)
	}
	if sr.MaxHold != 5 {
		t.Errorf("Expected maxHold 5, got %d", sr.MaxHold)
	}
}

func TestDefaultStrategyRunner_Backtest(t *testing.T) {
	sr := strategy.NewMeanReversionRunner(20, 2.0, 0.5, 60)
	if sr == nil {
		t.Fatal("Expected non-nil runner")
	}
	if sr.Lookback != 20 {
		t.Errorf("Expected lookback 20, got %d", sr.Lookback)
	}
}

func TestStrategyRunner_FeedBacktest(t *testing.T) {
	sr := strategy.NewMeanReversionRunner(10, 2.0, 0.5, 10)

	c := Candle{Time: time.Now(), Open: 100, High: 101, Low: 99, Close: 100, Symbol: "SPY"}
	for i := 0; i < 15; i++ {
		sig := sr.Evaluate(c, 0)
		if sig != nil {
			t.Logf("Signal at bar %d: side=%s", i, sig.Side)
		}
	}
}

func TestStrategyRunner_InsufficientData(t *testing.T) {
	sr := strategy.NewMeanReversionRunner(50, 2.0, 0.5, 5)

	c := Candle{Time: time.Now(), Open: 100, High: 101, Low: 99, Close: 100, Symbol: "SPY"}
	sig := sr.Evaluate(c, 0)
	if sig != nil {
		t.Error("Expected nil signal with insufficient data")
	}
}

func TestStrategyRunner_ZScoreMeanReversion(t *testing.T) {
	sr := strategy.NewMeanReversionRunner(10, 1.5, 0.3, 20)

	c := Candle{Time: time.Now(), Open: 100, High: 101, Low: 99, Close: 100, Symbol: "SPY"}
	for i := 0; i < 15; i++ {
		sr.Evaluate(c, 0)
	}

	spike := Candle{Time: time.Now(), Open: 120, High: 121, Low: 119, Close: 120, Symbol: "SPY"}
	sig := sr.Evaluate(spike, 0)
	if sig != nil {
		if sig.Side != "SELL" {
			t.Errorf("Expected SELL on spike to 120, got %s", sig.Side)
		}
		t.Logf("Spike signal: side=%s", sig.Side)
	}

	crash := Candle{Time: time.Now(), Open: 80, High: 81, Low: 79, Close: 80, Symbol: "SPY"}
	sig2 := sr.Evaluate(crash, 0)
	if sig2 != nil {
		if sig2.Side != "BUY" {
			t.Errorf("Expected BUY on crash to 80, got %s", sig2.Side)
		}
		t.Logf("Crash signal: side=%s", sig2.Side)
	}
}

func TestStrategyRunner_MaxHoldExit(t *testing.T) {
	sr := strategy.NewMeanReversionRunner(5, 0.5, 1.0, 3)

	c := Candle{Time: time.Now(), Open: 100, High: 101, Low: 99, Close: 100, Symbol: "SPY"}
	for i := 0; i < 6; i++ {
		sr.Evaluate(c, 0)
	}

	spike := Candle{Time: time.Now(), Open: 105, High: 106, Low: 104, Close: 105, Symbol: "SPY"}
	sig := sr.Evaluate(spike, 0)
	if sig != nil {
		t.Logf("Entry: side=%s", sig.Side)
	}

	for i := 0; i < 5; i++ {
		exitSig := sr.Evaluate(c, 0)
		if exitSig != nil {
			t.Logf("Bar %d after entry: signal=%v (exit on max-hold?)", i+1, exitSig != nil)
		}
	}
}

func TestStrategyRunner_ExitOnZScore(t *testing.T) {
	sr := strategy.NewMeanReversionRunner(5, 1.0, 1.5, 20)

	c := Candle{Time: time.Now(), Open: 100, High: 101, Low: 99, Close: 100, Symbol: "SPY"}
	for i := 0; i < 6; i++ {
		sr.Evaluate(c, 0)
	}

	spike := Candle{Time: time.Now(), Open: 110, High: 111, Low: 109, Close: 110, Symbol: "SPY"}
	sig := sr.Evaluate(spike, 0)
	if sig != nil {
		t.Logf("Entry signal: side=%s", sig.Side)
	}

	normal := Candle{Time: time.Now(), Open: 100, High: 101, Low: 99, Close: 100, Symbol: "SPY"}
	exitSig := sr.Evaluate(normal, 0)
	if exitSig != nil {
		t.Logf("Exit signal (z-score): was in position, now exited")
	} else {
		t.Log("Position still open (z-score not below exit threshold)")
	}
}
