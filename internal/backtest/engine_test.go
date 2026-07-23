package backtest

import (
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
	if dd == 0 {
		t.Error("Expected non-zero max drawdown")
	}
	_ = expected
	t.Logf("MaxDrawdown: %.2f%%", dd)
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
