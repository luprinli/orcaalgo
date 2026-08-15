package backtest

import (
	"context"
	"testing"
	"time"

	"github.com/lee-econ/orca-core/internal/types"
)

// TestResolveStopTarget_StopFirst locks the same-bar stop-vs-target precedence
// (Rule 10): when one bar's range contains both the stop and the target, the
// conservative stop-loss wins.
func TestResolveStopTarget_StopFirst(t *testing.T) {
	stop := &ActiveStop{
		Side:      "BUY",
		StopPrice: types.PriceFromFloat(90),
		TakePrice: types.PriceFromFloat(110),
	}
	// A bar whose range (85..115) contains both levels.
	candle := Candle{
		Open: types.PriceFromFloat(100), High: types.PriceFromFloat(115),
		Low: types.PriceFromFloat(85), Close: types.PriceFromFloat(95),
	}
	reason, _ := resolveStopTarget(candle, stop)
	if reason != "stop_loss" {
		t.Fatalf("reason = %q, want stop_loss (stop-first on same-bar ambiguity)", reason)
	}
}

// TestResolveStopTarget_TakeProfitOnly verifies the target fires when only it is hit.
func TestResolveStopTarget_TakeProfitOnly(t *testing.T) {
	stop := &ActiveStop{
		Side:      "BUY",
		StopPrice: types.PriceFromFloat(90),
		TakePrice: types.PriceFromFloat(110),
	}
	candle := Candle{
		Open: types.PriceFromFloat(100), High: types.PriceFromFloat(112),
		Low: types.PriceFromFloat(99), Close: types.PriceFromFloat(111),
	}
	reason, _ := resolveStopTarget(candle, stop)
	if reason != "take_profit" {
		t.Fatalf("reason = %q, want take_profit", reason)
	}
}

// TestNextBarEntry_DelayedFill verifies NEXT_BAR execution (Rule 9): a signal
// generated on bar t fills at bar t+1, so the trade's EntryTime is strictly
// after the first candle — never at the signal bar itself.
func TestNextBarEntry_DelayedFill(t *testing.T) {
	mock := &mockDB{candles: generateTestCandlesForDB("SPY", 200, 100)}
	eng := NewEngine(mock)
	eng.WirePipeline()
	cfg := BacktestConfig{
		StrategyID:     "rsi2_reversion",
		Symbols:        []string{"SPY"},
		StartDate:      time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2024, 9, 30, 0, 0, 0, 0, time.UTC),
		InitialCapital: 100000,
		Timeframe:      "1d",
		SizingPercent:  0.02,
		KellyFraction:  0.25,
	}
	result, err := eng.Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(result.Trades) == 0 {
		t.Skip("no trades generated; cannot verify fill delay")
	}
	firstCandleTime := mock.candles[0][0].Time
	for _, tr := range result.Trades {
		if !tr.EntryTime.After(firstCandleTime) {
			t.Errorf("trade filled at %v, which is not after the first candle %v (look-ahead)", tr.EntryTime, firstCandleTime)
		}
	}
	// The signal funnel must stay balanced with the deferred fill.
	if result.SignalDiag.SignalsPassed != result.SignalDiag.TradesOpened {
		t.Errorf("SignalsPassed=%d != TradesOpened=%d after NEXT_BAR deferral",
			result.SignalDiag.SignalsPassed, result.SignalDiag.TradesOpened)
	}
}
