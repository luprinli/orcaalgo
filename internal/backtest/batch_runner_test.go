package backtest

import "testing"

func TestStrategyTimeframeAllowed(t *testing.T) {
	cases := []struct {
		strategy  string
		timeframe string
		want      bool
	}{
		// Intraday strategies are blocked on 4h/1d.
		{"orb_15m", "15m", true},
		{"orb_15m", "1h", true},
		{"orb_15m", "4h", false},
		{"orb_15m", "1d", false},
		{"opening_range_breakout", "5m", true},
		{"opening_range_breakout", "1d", false},
		{"session_scalp", "30m", true},
		{"session_scalp", "4h", false},
		{"vwap_mr", "1h", true},
		{"vwap_mr", "1d", false},
		{"intraday_mr", "5m", true},
		{"intraday_mr", "1d", false},
		// Daily/swing strategies are blocked on sub-hourly bars.
		{"ma_crossover", "1h", true},
		{"ma_crossover", "4h", true},
		{"ma_crossover", "1d", true},
		{"ma_crossover", "5m", false},
		{"trend_following", "30m", false},
		{"trend_following", "1d", true},
		{"donchian_breakout", "15m", false},
		{"donchian_breakout", "1d", true},
		{"pairs_trading", "5m", false},
		{"pairs_trading", "1d", true},
		// Timeframe-agnostic strategies run on every timeframe.
		{"rsi2_reversion", "5m", true},
		{"rsi2_reversion", "1d", true},
		{"grid_trading", "1d", true},
		{"vol_grid", "15m", true},
		{"unknown_strategy", "5m", true},
	}
	for _, c := range cases {
		if got := strategyTimeframeAllowed(c.strategy, c.timeframe); got != c.want {
			t.Errorf("strategyTimeframeAllowed(%q, %q) = %v, want %v", c.strategy, c.timeframe, got, c.want)
		}
	}
}

func TestCartesianProduct_FiltersIncompatibleTimeframes(t *testing.T) {
	strategies := []string{"orb_15m", "ma_crossover", "rsi2_reversion"}
	symbols := []string{"SPY", "AAPL"}
	timeframes := []string{"5m", "1h", "1d"}

	combos := cartesianProduct(strategies, symbols, timeframes)

	// orb_15m (intraday) = {5m,1h}; ma_crossover (daily) = {1h,1d};
	// rsi2_reversion (agnostic) = {5m,1h,1d}. Per symbol = 2+2+3 = 7 → ×2 symbols.
	want := 14
	if len(combos) != want {
		t.Fatalf("cartesianProduct combo count = %d, want %d", len(combos), want)
	}

	seen := make(map[string]bool)
	for _, c := range combos {
		seen[c.Strategy+"|"+c.Timeframe] = true
	}
	if seen["orb_15m|1d"] {
		t.Error("orb_15m should not run on 1d")
	}
	if seen["ma_crossover|5m"] {
		t.Error("ma_crossover should not run on 5m")
	}
	if !seen["rsi2_reversion|5m"] {
		t.Error("rsi2_reversion should run on 5m")
	}
	if !seen["orb_15m|5m"] {
		t.Error("orb_15m should run on 5m")
	}
	if !seen["ma_crossover|1d"] {
		t.Error("ma_crossover should run on 1d")
	}
}
