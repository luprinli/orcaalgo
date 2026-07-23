package backtest

import (
	"testing"
	"time"
)

func TestFilterByRegime_EmptyTrades(t *testing.T) {
	result := &BacktestResult{
		Trades:      []Trade{},
		EquityCurve: []EquityPoint{},
	}
	filtered := FilterByRegime(result, 1)
	if filtered.WinRate != 0 {
		t.Errorf("Expected 0 win rate, got %f", filtered.WinRate)
	}
	if filtered.SharpeRatio != 0 {
		t.Errorf("Expected 0 Sharpe, got %f", filtered.SharpeRatio)
	}
}

func TestFilterByRegime_AllSameRegime(t *testing.T) {
	now := time.Now()
	trades := []Trade{
		{Symbol: "SPY", Side: "BUY", Quantity: 10, EntryPrice: 100, ExitPrice: 105, PnL: 50, HMMRegime: 1},
		{Symbol: "SPY", Side: "SELL", Quantity: 10, EntryPrice: 105, ExitPrice: 103, PnL: 20, HMMRegime: 1},
		{Symbol: "SPY", Side: "BUY", Quantity: 10, EntryPrice: 100, ExitPrice: 98, PnL: -20, HMMRegime: 1},
	}
	equity := []EquityPoint{
		{Time: now, Value: 100000, Regime: 1},
		{Time: now.Add(time.Hour), Value: 100050, Regime: 1},
		{Time: now.Add(2 * time.Hour), Value: 100070, Regime: 1},
		{Time: now.Add(3 * time.Hour), Value: 100050, Regime: 1},
	}
	result := &BacktestResult{Trades: trades, EquityCurve: equity, NumTrades: len(trades)}

	filtered := FilterByRegime(result, 1)
	if filtered.NumTrades != 3 {
		t.Errorf("Expected 3 trades, got %d", filtered.NumTrades)
	}
}

func TestFilterByRegime_NoTradesForRegime(t *testing.T) {
	trades := []Trade{
		{Symbol: "SPY", Side: "BUY", Quantity: 10, EntryPrice: 100, ExitPrice: 105, PnL: 50, HMMRegime: 0},
	}
	result := &BacktestResult{Trades: trades, EquityCurve: []EquityPoint{}, NumTrades: 1}

	filtered := FilterByRegime(result, 3)
	if filtered.NumTrades != 0 {
		t.Errorf("Expected 0 trades for regime 3, got %d", filtered.NumTrades)
	}
}

func TestFilterByRegime_MixedRegimes(t *testing.T) {
	now := time.Now()
	trades := []Trade{
		{Symbol: "SPY", Side: "BUY", Quantity: 10, EntryPrice: 100, ExitPrice: 105, PnL: 50, HMMRegime: 0},
		{Symbol: "QQQ", Side: "BUY", Quantity: 10, EntryPrice: 100, ExitPrice: 108, PnL: 80, HMMRegime: 1},
		{Symbol: "AAPL", Side: "SELL", Quantity: 10, EntryPrice: 100, ExitPrice: 95, PnL: 50, HMMRegime: 0},
		{Symbol: "SPY", Side: "SELL", Quantity: 10, EntryPrice: 100, ExitPrice: 97, PnL: 30, HMMRegime: 1},
	}
	equity := []EquityPoint{
		{Time: now, Value: 100000, Regime: 0},
		{Time: now.Add(time.Hour), Value: 100050, Regime: 0},
		{Time: now.Add(2 * time.Hour), Value: 100130, Regime: 1},
		{Time: now.Add(3 * time.Hour), Value: 100180, Regime: 0},
		{Time: now.Add(4 * time.Hour), Value: 100210, Regime: 1},
	}
	result := &BacktestResult{Trades: trades, EquityCurve: equity, NumTrades: len(trades)}

	filtered0 := FilterByRegime(result, 0)
	if filtered0.NumTrades != 2 {
		t.Errorf("Expected 2 trades for regime 0, got %d", filtered0.NumTrades)
	}

	filtered1 := FilterByRegime(result, 1)
	if filtered1.NumTrades != 2 {
		t.Errorf("Expected 2 trades for regime 1, got %d", filtered1.NumTrades)
	}
}

func TestFilterByRegime_InvalidRegime(t *testing.T) {
	trades := []Trade{{Symbol: "SPY", Side: "BUY", Quantity: 10, EntryPrice: 100, ExitPrice: 105, PnL: 50, HMMRegime: 0}}
	result := &BacktestResult{Trades: trades, EquityCurve: []EquityPoint{EquityPoint{Time: time.Now(), Value: 100000, Regime: 0}}}

	filtered := FilterByRegime(result, 5)
	if filtered.NumTrades != 0 {
		t.Errorf("Expected 0 trades for invalid regime, got %d", filtered.NumTrades)
	}
}

func TestFilterByRegime_WinRate(t *testing.T) {
	trades := []Trade{
		{Symbol: "SPY", Side: "BUY", Quantity: 10, EntryPrice: 100, ExitPrice: 105, PnL: 50, HMMRegime: 0},
		{Symbol: "SPY", Side: "BUY", Quantity: 10, EntryPrice: 100, ExitPrice: 98, PnL: -20, HMMRegime: 0},
		{Symbol: "SPY", Side: "BUY", Quantity: 10, EntryPrice: 100, ExitPrice: 107, PnL: 70, HMMRegime: 0},
		{Symbol: "SPY", Side: "BUY", Quantity: 10, EntryPrice: 100, ExitPrice: 103, PnL: 30, HMMRegime: 0},
	}
	result := &BacktestResult{
		Trades:      trades,
		EquityCurve: []EquityPoint{{Time: time.Now(), Value: 100000, Regime: 0}},
		NumTrades:   len(trades),
	}

	filtered := FilterByRegime(result, 0)
	if filtered.WinRate < 50 {
		t.Errorf("Expected win rate >= 50%%, got %f", filtered.WinRate)
	}
}
