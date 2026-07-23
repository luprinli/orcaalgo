package backtest

import (
	"math"
	"math/rand/v2"
	"testing"
	"time"
)

func TestRunMonteCarlo_Deterministic(t *testing.T) {
	returns := generateDailyReturns(252, 0.001, 0.0005)
	baseResult := &BacktestResult{
		DailyReturns: returns,
	}

	cfg := MCConfig{Iterations: 100, BarsPerSim: 60, Seed: 42}
	r1, err := RunMonteCarlo(baseResult, cfg)
	if err != nil {
		t.Fatalf("RunMonteCarlo: %v", err)
	}
	r2, err := RunMonteCarlo(baseResult, cfg)
	if err != nil {
		t.Fatalf("RunMonteCarlo second: %v", err)
	}

	if len(r1.Iterations) != 100 || len(r2.Iterations) != 100 {
		t.Fatalf("expected 100 iterations, got %d and %d", len(r1.Iterations), len(r2.Iterations))
	}

	for i := range r1.Iterations {
		if r1.Iterations[i].PnlPct != r2.Iterations[i].PnlPct {
			t.Fatalf("non-deterministic result at iteration %d: %f vs %f", i, r1.Iterations[i].PnlPct, r2.Iterations[i].PnlPct)
		}
	}
}

func TestRunMonteCarlo_InsufficientReturns(t *testing.T) {
	baseResult := &BacktestResult{DailyReturns: []DailyReturn{}}
	_, err := RunMonteCarlo(baseResult, MCConfig{Iterations: 10, BarsPerSim: 5, Seed: 1})
	if err != ErrInsufficientReturns {
		t.Fatalf("expected ErrInsufficientReturns, got: %v", err)
	}
}

func TestMCSummary_Stats(t *testing.T) {
	returns := generateDailyReturns(500, 0.001, 0.0008)
	baseResult := &BacktestResult{DailyReturns: returns}

	result, err := RunMonteCarlo(baseResult, MCConfig{Iterations: 200, BarsPerSim: 120, Seed: 7})
	if err != nil {
		t.Fatalf("RunMonteCarlo: %v", err)
	}

	s := result.Summary
	if s.NumSimulations != 200 {
		t.Errorf("NumSimulations: expected 200, got %d", s.NumSimulations)
	}
	if s.NumDays != 120 {
		t.Errorf("NumDays: expected 120, got %d", s.NumDays)
	}
	if math.IsNaN(s.AvgPnlPct) {
		t.Error("AvgPnlPct is NaN")
	}
	if math.IsNaN(s.MedianPnlPct) {
		t.Error("MedianPnlPct is NaN")
	}
	if s.P5PnlPct > s.MedianPnlPct {
		t.Errorf("P5PnlPct (%f) should be <= MedianPnlPct (%f)", s.P5PnlPct, s.MedianPnlPct)
	}
	if s.BustProbability < 0 || s.BustProbability > 1 {
		t.Errorf("BustProbability out of range: %f", s.BustProbability)
	}
}

func generateDailyReturns(n int, mean, std float64) []DailyReturn {
	out := make([]DailyReturn, n)
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < n; i++ {
		r := mean + std*gaussianRandom()
		out[i] = DailyReturn{
			Date:   base.AddDate(0, 0, i),
			Return: r,
		}
	}
	return out
}

func gaussianRandom() float64 {
	var sum float64
	for i := 0; i < 12; i++ {
		sum += rand.Float64()
	}
	return (sum - 6.0) / 6.0
}
