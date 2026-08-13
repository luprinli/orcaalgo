package backtest

import "testing"

func TestFlagImplausibleCombos(t *testing.T) {
	results := []ComboResult{
		{StrategyID: "orb", Symbol: "SPY", Timeframe: "1d", NumTrades: 100, SharpeRatio: 4.2, ProfitFactor: 1.5, WinRate: 40, LongTrades: 50, ShortTrades: 50, LongWinRate: 40, ShortPF: 1.5, TotalReturn: 20},
		{StrategyID: "orb", Symbol: "QQQ", Timeframe: "1d", NumTrades: 80, SharpeRatio: 1.2, ProfitFactor: 15.0, WinRate: 50, LongTrades: 40, ShortTrades: 40, LongWinRate: 50, ShortPF: 2.0, TotalReturn: 30},
		{StrategyID: "grid", Symbol: "AAPL", Timeframe: "1d", NumTrades: 200, SharpeRatio: 1.0, ProfitFactor: 2.0, WinRate: 95, LongTrades: 100, ShortTrades: 100, LongWinRate: 95, ShortPF: 2.0, TotalReturn: 10},
		{StrategyID: "session", Symbol: "MSFT", Timeframe: "1d", NumTrades: 50, SharpeRatio: 1.0, ProfitFactor: 2.0, WinRate: 50, LongTrades: 40, ShortTrades: 10, LongWinRate: 0, ShortPF: 1000, TotalReturn: 10},
		{StrategyID: "breakout", Symbol: "NVDA", Timeframe: "1d", NumTrades: 30, SharpeRatio: 1.0, ProfitFactor: 2.0, WinRate: 50, LongTrades: 15, ShortTrades: 15, LongWinRate: 50, ShortPF: 1.5, TotalReturn: 1500},
		// timeframe dedup: same strategy+symbol, identical trade counts across TFs
		{StrategyID: "trend", Symbol: "TSLA", Timeframe: "1d", NumTrades: 120, SharpeRatio: 1.0, ProfitFactor: 1.5, WinRate: 50, LongTrades: 60, ShortTrades: 60, LongWinRate: 50, ShortPF: 1.5, TotalReturn: 15},
		{StrategyID: "trend", Symbol: "TSLA", Timeframe: "1h", NumTrades: 120, SharpeRatio: 1.0, ProfitFactor: 1.5, WinRate: 50, LongTrades: 60, ShortTrades: 60, LongWinRate: 50, ShortPF: 1.5, TotalReturn: 15},
		{StrategyID: "trend", Symbol: "TSLA", Timeframe: "5m", NumTrades: 120, SharpeRatio: 1.0, ProfitFactor: 1.5, WinRate: 50, LongTrades: 60, ShortTrades: 60, LongWinRate: 50, ShortPF: 1.5, TotalReturn: 15},
		// clean combo (should produce no flag)
		{StrategyID: "mean_reversion", Symbol: "IWM", Timeframe: "1d", NumTrades: 60, SharpeRatio: 1.4, ProfitFactor: 1.8, WinRate: 55, LongTrades: 30, ShortTrades: 30, LongWinRate: 55, ShortPF: 1.8, TotalReturn: 18},
		// zero-trade combo (should be ignored)
		{StrategyID: "pairs", Symbol: "GLD", Timeframe: "1d", NumTrades: 0},
	}

	flags := FlagImplausibleCombos(results)

	codes := map[string]int{}
	for _, f := range flags {
		codes[f.Code]++
	}

	if codes["sharpe_implausible"] != 1 {
		t.Errorf("expected 1 sharpe_implausible flag, got %d", codes["sharpe_implausible"])
	}
	if codes["profit_factor_implausible"] != 1 {
		t.Errorf("expected 1 profit_factor_implausible flag, got %d", codes["profit_factor_implausible"])
	}
	if codes["win_rate_implausible"] != 1 {
		t.Errorf("expected 1 win_rate_implausible flag, got %d", codes["win_rate_implausible"])
	}
	if codes["directional_asymmetry"] != 1 {
		t.Errorf("expected 1 directional_asymmetry flag, got %d", codes["directional_asymmetry"])
	}
	if codes["sentinel_pf"] != 1 {
		t.Errorf("expected 1 sentinel_pf flag, got %d", codes["sentinel_pf"])
	}
	if codes["return_implausible"] != 1 {
		t.Errorf("expected 1 return_implausible flag, got %d", codes["return_implausible"])
	}
	if codes["timeframe_dedup"] != 1 {
		t.Errorf("expected 1 timeframe_dedup flag, got %d", codes["timeframe_dedup"])
	}
	// The clean combo and zero-trade combo must not produce flags.
	for _, f := range flags {
		if f.Symbol == "IWM" || f.Symbol == "GLD" {
			t.Errorf("unexpected flag for %s: %s", f.Symbol, f.Code)
		}
	}
}
