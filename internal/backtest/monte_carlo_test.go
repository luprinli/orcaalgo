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

func TestBootstrapBlockPath_BlockSize(t *testing.T) {
	n := 100
	returns := make([]float64, n)
	for i := 0; i < n; i++ {
		returns[i] = float64(i%7-3) * 0.001
	}

	rng := rand.New(rand.NewPCG(42, 0))
	path := bootstrapBlockPath(returns, 50, rng, 7)
	if len(path) != 50 {
		t.Errorf("Expected 50 path elements, got %d", len(path))
	}
	for _, v := range path {
		if v <= 0 {
			t.Errorf("Path equity should be positive, got %f", v)
		}
	}
}

func TestBootstrapBlockPath_BlockLengthOne(t *testing.T) {
	returns := []float64{0.01, -0.005, 0.02, -0.01, 0.005}
	rng1 := rand.New(rand.NewPCG(1, 0))
	rng2 := rand.New(rand.NewPCG(1, 0))
	path1 := bootstrapBlockPath(returns, 20, rng1, 1)
	path2 := bootstrapBlockPath(returns, 20, rng2, 1)
	if len(path1) != 20 || len(path2) != 20 {
		t.Fatalf("Expected 20 elements each")
	}
	for i := 0; i < 20; i++ {
		if path1[i] != path2[i] {
			t.Fatalf("Non-deterministic with seed: iteration %d diff %f vs %f", i, path1[i], path2[i])
		}
	}
}

func TestBootstrapBlockPath_BlockPreservesSequence(t *testing.T) {
	returns := make([]float64, 50)
	for i := 0; i < 50; i++ {
		returns[i] = float64(i)
	}
	rng := rand.New(rand.NewPCG(99, 0))
	path := bootstrapBlockPath(returns, 10, rng, 5)
	if len(path) != 10 {
		t.Fatalf("Expected 10 elements")
	}
	diffs := make([]float64, 9)
	for i := 0; i < 9; i++ {
		diffs[i] = path[i+1] - path[i]
	}
	nonZero := 0
	for _, d := range diffs {
		if d != 0 {
			nonZero++
		}
	}
	if nonZero < 5 {
		t.Errorf("Block bootstrap should produce varied equity path; only %d/9 non-zero diffs", nonZero)
	}
}

func TestBootstrapBlockPath_WithDrawdowns(t *testing.T) {
	returns := make([]float64, 200)
	for i := 0; i < 200; i++ {
		returns[i] = (float64(i%5-2) * 0.005)
	}
	rng := rand.New(rand.NewPCG(12345, 0))
	path := bootstrapBlockPath(returns, 252, rng, 7)
	if len(path) != 252 {
		t.Fatalf("Expected 252 elements")
	}
	minVal := path[0]
	for _, v := range path {
		if v < minVal {
			minVal = v
		}
	}
	if minVal >= 1.0 {
		t.Errorf("With negative returns, should see drawdowns. Min equity: %f", minVal)
	}
}
