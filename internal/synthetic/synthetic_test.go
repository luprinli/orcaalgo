package synthetic

import (
	"math"
	"math/rand/v2"
	"testing"
)

func TestForTicker(t *testing.T) {
	c, ok := ForTicker("SPY")
	if !ok {
		t.Fatal("SPY should resolve a calibration")
	}
	if c.BasePrice != 580.0 || c.SigmaDaily != 0.008 {
		t.Errorf("SPY calibration = %+v, want {580 0.008}", c)
	}

	c, ok = ForTicker("BTC-USD")
	if !ok || c.BasePrice != 68000.0 || c.SigmaDaily != 0.030 {
		t.Errorf("BTC-USD calibration = %+v ok=%v, want {68000 0.030}", c, ok)
	}

	if _, ok := ForTicker("NO_SUCH_TICKER"); ok {
		t.Error("unknown ticker should not resolve")
	}
}

func TestIntradayPath(t *testing.T) {
	rng := rand.New(rand.NewPCG(42, 1))
	const nSteps = 1440
	open := 100.0
	closeP := 101.0
	sigma := 0.01

	path := IntradayPath(rng, open, closeP, sigma, nSteps)

	if len(path) != nSteps {
		t.Fatalf("path length = %d, want %d", len(path), nSteps)
	}
	// No NaN/Inf and no negative prices (floored near zero).
	for i, p := range path {
		if math.IsNaN(p) || math.IsInf(p, 0) {
			t.Fatalf("path[%d] = %v (non-finite)", i, p)
		}
		if p <= 0 {
			t.Fatalf("path[%d] = %v (non-positive)", i, p)
		}
	}

	// The path is unconstrained: it should break both open and close at some
	// point given enough steps (proves no clipping).
	maxP, minP := path[0], path[0]
	for _, p := range path {
		if p > maxP {
			maxP = p
		}
		if p < minP {
			minP = p
		}
	}
	if maxP <= open && maxP <= closeP {
		t.Errorf("path never exceeded open/close (max=%.4f, open=%.4f, close=%.4f) — suspicious clipping", maxP, open, closeP)
	}

	// Determinism: same seed → identical path.
	path2 := IntradayPath(rand.New(rand.NewPCG(42, 1)), open, closeP, sigma, nSteps)
	for i := range path {
		if path[i] != path2[i] {
			t.Fatalf("non-deterministic path at index %d", i)
		}
	}
}
