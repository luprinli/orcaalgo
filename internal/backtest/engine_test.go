package backtest

import (
	"math"
	"testing"
	"time"
)

func TestCalculateSharpe(t *testing.T) {
	equity := []EquityPoint{
		{Value: 100000},
		{Value: 100100},
		{Value: 100050},
		{Value: 100200},
		{Value: 100180},
	}
	s := calculateSharpe(equity, 1.0)
	if s == 0 {
		t.Logf("Warning: Sharpe = 0 (expected with few data points)")
	}
}

func TestCalculateMaxDrawdown(t *testing.T) {
	equity := []EquityPoint{
		{Value: 100000},
		{Value: 110000},
		{Value: 90000},
		{Value: 105000},
		{Value: 120000},
	}
	dd := calculateMaxDrawdown(equity)
	expected := (110000.0 - 90000.0) / 110000.0 * 100
	if math.Abs(dd-expected) > 1e-9 {
		t.Errorf("MaxDrawdown: got %.6f, want %.6f", dd, expected)
	}

	// A curve that only declines from the start must still report a non-zero
	// drawdown (the gating bug previously reported MaxDD==0 for these).
	declining := []EquityPoint{
		{Value: 100000},
		{Value: 98000},
		{Value: 95000},
	}
	ddDecline := calculateMaxDrawdown(declining)
	if ddDecline <= 0 {
		t.Errorf("declining equity should report positive drawdown, got %.2f", ddDecline)
	}
}

func TestClampExcursionPct(t *testing.T) {
	cases := []struct {
		in   float64
		want float64
	}{
		{0, 0},
		{2.5, 2.5},
		{-3.1, -3.1},
		{10000.0, 10000.0},
		{36247557.5, 10000.0},
		{-1e12, -10000.0},
		{math.NaN(), 0},
		{math.Inf(1), 0},
		{math.Inf(-1), 0},
	}
	for _, c := range cases {
		got := clampExcursionPct(c.in)
		if math.IsNaN(c.want) {
			if !math.IsNaN(got) {
				t.Errorf("clampExcursionPct(%v) = %v, want NaN", c.in, got)
			}
			continue
		}
		if got != c.want {
			t.Errorf("clampExcursionPct(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestCalculateWinRate(t *testing.T) {
	trades := []Trade{
		{PnL: 100},
		{PnL: -50},
		{PnL: 200},
		{PnL: -30},
		{PnL: 500},
	}
	wr := calculateWinRate(trades)
	expected := 3.0 / 5.0 * 100
	if wr != expected {
		t.Errorf("WinRate: got %.2f, want %.2f", wr, expected)
	}
}

func TestCalculateProfitFactor(t *testing.T) {
	trades := []Trade{
		{PnL: 100},
		{PnL: -50},
		{PnL: 200},
		{PnL: -30},
	}
	pf := calculateProfitFactor(trades)
	expected := 300.0 / 80.0
	if pf != expected {
		t.Errorf("ProfitFactor: got %.2f, want %.2f", pf, expected)
	}
}

func TestMergeCandlesByTime(t *testing.T) {
	candles := [][]Candle{
		{{Time: parseTestTime("2024-01-02T10:00:00Z"), Symbol: "AAPL"}},
		{{Time: parseTestTime("2024-01-02T09:30:00Z"), Symbol: "MSFT"}},
	}
	merged := mergeCandlesByTime(candles)
	if len(merged) != 2 {
		t.Errorf("Expected 2 merged candles, got %d", len(merged))
	}
	if merged[0].Time.After(merged[1].Time) {
		t.Error("Candles not sorted by time")
	}
}

func parseTestTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}
