package backtest

import (
	"testing"
	"time"
)

func TestRepresentativeGenerationID(t *testing.T) {
	cases := []struct {
		name    string
		candles []Candle
		want    string
	}{
		{"empty", nil, ""},
		{"no lineage", []Candle{{Time: time.Now()}}, ""},
		{
			"single",
			[]Candle{
				{GenerationID: "abc123", Time: time.Unix(1, 0)},
				{GenerationID: "abc123", Time: time.Unix(2, 0)},
			},
			"abc123",
		},
		{
			"most common wins",
			[]Candle{
				{GenerationID: "gen-a", Time: time.Unix(1, 0)},
				{GenerationID: "gen-b", Time: time.Unix(2, 0)},
				{GenerationID: "gen-a", Time: time.Unix(3, 0)},
				{GenerationID: "gen-a", Time: time.Unix(4, 0)},
			},
			"gen-a",
		},
		{
			"ignores empty",
			[]Candle{
				{GenerationID: "", Time: time.Unix(1, 0)},
				{GenerationID: "gen-x", Time: time.Unix(2, 0)},
			},
			"gen-x",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := representativeGenerationID(c.candles); got != c.want {
				t.Errorf("representativeGenerationID() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestNewFillSimulator_DefaultIsDeterministic(t *testing.T) {
	m := DefaultEquitySlippage()
	fs1 := NewFillSimulator(m)
	fs2 := NewFillSimulator(m)

	tickTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	f1 := fs1.SimulateFill(1, "AAPL", 185.0, 50.0, "BUY", 185.0, tickTime)
	f2 := fs2.SimulateFill(1, "AAPL", 185.0, 50.0, "BUY", 185.0, tickTime)
	if f1.FillPrice != f2.FillPrice || f1.FillQuantity != f2.FillQuantity {
		t.Errorf("default fill simulator should be deterministic: price(%f vs %f) qty(%f vs %f)",
			f1.FillPrice.Float64(), f2.FillPrice.Float64(), f1.FillQuantity, f2.FillQuantity)
	}
}
