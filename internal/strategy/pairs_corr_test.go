package strategy

import (
	"math"
	"testing"
	"time"

	"github.com/lee-econ/orca-core/internal/types"
)

func TestPairsRunnerPairLogCorrelation(t *testing.T) {
	r := NewPairsRunner("EURUSD", "GBPUSD")
	n := 120
	commonWalk := 0.0
	spread := 0.0
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < n; i++ {
		commonWalk += math.Sin(float64(i)/8.0) * 0.001
		spread = 0.7*spread + math.Sin(float64(i)/3.0)*0.002
		g := 1.27 * math.Exp(commonWalk)
		e := 1.08 * math.Exp(commonWalk+spread)
		c := Candle{Time: start.Add(time.Duration(i) * 24 * time.Hour), Symbol: "EURUSD", Close: types.PriceFromFloat(e)}
		r.PushPrice(c.Close, c.High, c.Low, c.Volume)
		r.PushSecondaryPrice(types.PriceFromFloat(g))
	}
	corr := r.pairLogCorrelation()
	t.Logf("pairLogCorrelation = %.4f (HistCount=%d, secHistCount=%d)", corr, r.HistCount, r.secHistCount)
	if corr < 0.5 {
		t.Errorf("expected correlated pair (>=0.5), got %.4f", corr)
	}
}
