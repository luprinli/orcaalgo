package backtest

import (
	"context"
	"testing"
	"time"
)

// signalFunnelAccounted returns the sum of every signal-disposition counter for
// a SignalDiag. The invariant is that this must equal SignalAttempts — every
// signal attempt is tallied by exactly one disposition (Rule 7).
func signalFunnelAccounted(d SignalDiag) int {
	return d.RegimeRejected + d.NilError + d.StrategyNil + d.ExitSignalZeroQty +
		d.VolHalted + d.PipelineRejected + d.MLRejected + d.FillRejected +
		d.CapitalZero + d.RateLimited + d.BaseSizeZero + d.QuantityTooSmall +
		d.ExposureBlocked + d.SignalsPassed
}

// TestSignalFunnelSumInvariant runs real backtests (mock candles, wired
// pipeline) and asserts the funnel columns sum to SignalAttempts. A mismatch
// means a rejection path is not tallied (or double-tallied).
func TestSignalFunnelSumInvariant(t *testing.T) {
	for _, stratID := range []string{"intraday_mr", "rsi2_reversion", "trend_following", "donchian_breakout"} {
		t.Run(stratID, func(t *testing.T) {
			mock := &mockDB{candles: generateTestCandlesForDB("SPY", 300, 100)}
			eng := NewEngine(mock)
			eng.WirePipeline()
			cfg := BacktestConfig{
				StrategyID:     stratID,
				Symbols:        []string{"SPY"},
				StartDate:      time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
				EndDate:        time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
				InitialCapital: 100000,
				Timeframe:      "1d",
				SizingPercent:  0.02,
				KellyFraction:  0.25,
			}
			result, err := eng.Run(context.Background(), cfg)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			d := result.SignalDiag
			if accounted := signalFunnelAccounted(d); accounted != d.SignalAttempts {
				t.Errorf("funnel sum mismatch: accounted=%d signal_attempts=%d\ndiag=%+v",
					accounted, d.SignalAttempts, d)
			}
			// Passed signals must open exactly one trade each.
			if d.SignalsPassed != d.TradesOpened {
				t.Errorf("SignalsPassed=%d != TradesOpened=%d", d.SignalsPassed, d.TradesOpened)
			}
		})
	}
}

// TestSignalFunnelAccounted_Identity is the pure unit test of the accounting
// identity on a synthetic diag (no engine run).
func TestSignalFunnelAccounted_Identity(t *testing.T) {
	d := SignalDiag{
		SignalAttempts:   100,
		StrategyNil:      50,
		RegimeRejected:   20,
		NilError:         2,
		VolHalted:        3,
		PipelineRejected: 5,
		CapitalZero:      4,
		RateLimited:      1,
		BaseSizeZero:     2,
		QuantityTooSmall: 1,
		ExposureBlocked:  2,
		SignalsPassed:    10,
		TradesOpened:     10,
	}
	if got := signalFunnelAccounted(d); got != d.SignalAttempts {
		t.Errorf("accounted=%d, want %d", got, d.SignalAttempts)
	}
}
