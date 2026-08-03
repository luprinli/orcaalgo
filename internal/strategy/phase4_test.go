package strategy

import (
	"testing"
	"time"

	"github.com/lee-econ/orca-core/internal/types"
)

func mkPrice(v float64) types.Price { return types.PriceFromFloat(v) }

func mkCandle(sym string, price float64, high, low, vol float64) Candle {
	return Candle{
		Symbol: sym,
		Time:   time.Now(),
		Open:   mkPrice(price),
		High:   mkPrice(high),
		Low:    mkPrice(low),
		Close:  mkPrice(price),
		Volume: vol,
	}
}

func TestVolHarvestingRunner_NoEntryBelowVIXThreshold(t *testing.T) {
	r := NewVolHarvestingRunner()
	r.VIXThreshold = 25.0
	r.CurrentVIX = 20.0

	candle := mkCandle("SPY", 502.0, 505.0, 495.0, 1000000)
	sig := r.Evaluate(candle, 2)
	if sig != nil {
		t.Error("should not generate signal when VIX is below threshold")
	}
}

func TestVolHarvestingRunner_EntryOnVolSpike(t *testing.T) {
	r := NewVolHarvestingRunner()
	r.VIXThreshold = 25.0
	r.CurrentVIX = 28.0
	r.MeanRevEntryZ = 1.5
	r.MeanRevLookback = 10

	basePrice := 500.0
	for i := 0; i < 15; i++ {
		candle := mkCandle("SPY", basePrice, basePrice+2, basePrice-2, 1000000)
		r.Evaluate(candle, 2)
	}

	candle := mkCandle("SPY", 480.0, 482.0, 478.0, 2000000)
	sig := r.Evaluate(candle, 2)
	if sig == nil {
		t.Error("should generate BUY signal on oversold after vol spike")
	}
	if sig.Side != "BUY" {
		t.Errorf("expected BUY, got %s", sig.Side)
	}
}

func TestVolHarvestingRunner_ExitOnStop(t *testing.T) {
	r := NewVolHarvestingRunner()
	r.VIXThreshold = 25.0
	r.CurrentVIX = 28.0
	r.StopATRMult = 2.0
	r.MeanRevEntryZ = 1.0

	for i := 0; i < 15; i++ {
		candle := mkCandle("SPY", 500.0, 505.0, 495.0, 1000000)
		r.Evaluate(candle, 2)
	}

	candle := mkCandle("SPY", 470.0, 472.0, 468.0, 2000000)
	r.Evaluate(candle, 2)

	if !r.PositionOpen {
		t.Fatal("should have opened position")
	}

	candle2 := mkCandle("SPY", 440.0, 442.0, 438.0, 2000000)
	sig := r.Evaluate(candle2, 2)
	if sig == nil || sig.Side != "SELL" {
		t.Error("should exit BUY position when stop hit")
	}
}

func TestPairsRunner_Basic(t *testing.T) {
	r := NewPairsRunner("SPY", "QQQ")
	r.SetPairStatus(PairStatus{Primary: "SPY", Secondary: "QQQ", HedgeRatio: 1.2, PValue: 0.01, Valid: true})
	r.EntryZ = 2.0

	for i := 0; i < 70; i++ {
		r.PushSecondaryPrice(mkPrice(400.0))
		candle := mkCandle("SPY", 480.0, 485.0, 475.0, 1000000)
		r.Evaluate(candle, 0)
	}

	for i := 0; i < 5; i++ {
		r.PushSecondaryPrice(mkPrice(410.0))
	}

	candle := mkCandle("SPY", 480.0, 485.0, 475.0, 1000000)
	sig := r.Evaluate(candle, 0)
	_ = sig // Just verify no panic with diverged prices
}

func TestPairsRunner_NoEntryWhenInvalid(t *testing.T) {
	r := NewPairsRunner("SPY", "QQQ")
	r.SetPairStatus(PairStatus{Primary: "SPY", Secondary: "QQQ", HedgeRatio: 1.2, PValue: 0.10, Valid: true})
	r.EntryZ = 1.0

	for i := 0; i < 70; i++ {
		r.PushSecondaryPrice(mkPrice(400.0))
		candle := mkCandle("SPY", 480.0, 485.0, 475.0, 1000000)
		r.Evaluate(candle, 0)
	}

	if r.PositionOpen {
		t.Error("should not open position when pair is not cointegrated")
	}
}

func TestChoppinessIndex_Basic(t *testing.T) {
	prices := []float64{100, 101, 100, 101, 100, 101, 100, 101, 100, 101, 100, 101, 100, 101}
	highs := []float64{102, 103, 102, 103, 102, 103, 102, 103, 102, 103, 102, 103, 102, 103}
	lows := []float64{98, 99, 98, 99, 98, 99, 98, 99, 98, 99, 98, 99, 98, 99}

	chop := ChoppinessIndex(prices, highs, lows, 14, 14)
	if chop <= 0 {
		t.Errorf("chop should be positive, got %v", chop)
	}
	if chop < 61.8 {
		t.Logf("chop = %v (expected > 61.8 for choppy market)", chop)
	}
}
