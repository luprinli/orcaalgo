package backtest

// This file implements the engine-vs-live backtest comparison: given the trades
// a backtest simulated and the trades that actually executed live, it derives
// the implied execution cost gaps (slippage, limit penetration, expense ratio)
// so the backtest fill model can be validated and recalibrated against reality.
//
// The functions are pure over plain inputs so they are unit-testable and shared
// by the API handler and any offline reconciliation tool.

import (
	"math"
	"time"

	"github.com/lee-econ/orca-core/internal/types"
)

// MatchedLiveTrade pairs one engine (backtest) entry with the corresponding
// live fill for the same symbol and date.
type MatchedLiveTrade struct {
	Symbol        string      `json:"symbol"`
	Side          string      `json:"side"` // "BUY" or "SELL"
	Date          time.Time   `json:"date"`
	EnginePrice   types.Price `json:"engine_price"`
	LivePrice     types.Price `json:"live_price"`
	Quantity      float64     `json:"quantity"`
	LimitPrice    types.Price `json:"limit_price"` // limit order price, 0 for market
	DayLow        types.Price `json:"day_low"`     // day low for penetration measure
	LiveExpense   float64     `json:"live_expense"`
	EngineExpense float64     `json:"engine_expense"`
}

// ImpliedComparison summarises the gap between simulated and realised fills.
type ImpliedComparison struct {
	MatchedCount       int     `json:"matched_count"`
	ImpliedSlippageBps float64 `json:"implied_slippage_bps"` // weighted signed slippage
	ImpliedAvgAbsBps   float64 `json:"implied_avg_abs_bps"`  // weighted absolute slippage
	PenetrationPct     float64 `json:"penetration_pct"`      // weighted (limit-low)/limit
	ExpenseGapBps      float64 `json:"expense_gap_bps"`      // weighted live-engine expense
	EntryPriceGapBps   float64 `json:"entry_price_gap_bps"`  // weighted (live-engine)/engine
}

// ComputeImpliedComparison derives execution-cost gaps from matched engine/live
// trades. Slippage is (live - engine) for longs and (engine - live) for shorts,
// expressed in basis points of the engine price, notional-weighted by quantity.
// Entries with a non-positive engine price or quantity are skipped.
func ComputeImpliedComparison(matched []MatchedLiveTrade) ImpliedComparison {
	out := ImpliedComparison{}
	var (
		slipNum, slipAbsNum, slipDen float64
		penNum, penDen               float64
		expNum, expDen               float64
		gapNum                       float64
	)
	for _, m := range matched {
		engine := m.EnginePrice.Float64()
		live := m.LivePrice.Float64()
		limit := m.LimitPrice.Float64()
		dayLow := m.DayLow.Float64()
		if engine <= 0 || m.Quantity <= 0 {
			continue
		}
		out.MatchedCount++
		weight := m.Quantity
		// Signed slippage in engine-price basis points: (live - engine)/engine.
		// No side flip — the sign simply reports whether the live fill was above
		// or below the simulated entry, matching the reference definition.
		slip := (live - engine) / engine * 10000.0
		slipNum += slip * weight
		slipAbsNum += math.Abs(slip) * weight
		slipDen += weight

		// Entry price gap (unsigned, as a diagnostic).
		gapNum += ((live - engine) / engine) * weight

		// Limit penetration: how far the day low traded through a buy limit.
		if limit > 0 && dayLow > 0 && dayLow < limit {
			penNum += ((limit - dayLow) / limit) * weight
			penDen += weight
		}

		// Expense-ratio gap.
		expNum += (m.LiveExpense - m.EngineExpense) * weight
		expDen += weight
	}
	if slipDen > 0 {
		out.ImpliedSlippageBps = slipNum / slipDen
		out.ImpliedAvgAbsBps = slipAbsNum / slipDen
		out.EntryPriceGapBps = gapNum / slipDen * 10000.0
	}
	if penDen > 0 {
		out.PenetrationPct = penNum / penDen * 100.0
	}
	if expDen > 0 {
		out.ExpenseGapBps = expNum / expDen
	}
	return out
}

// MaxEquityDivergencePct returns the largest relative gap between two
// time-aligned equity curves, as a percentage. Curves need not share timestamps;
// entries are matched by index position (assumed already aligned/regular).
func MaxEquityDivergencePct(backtest, live []float64) float64 {
	n := len(backtest)
	if len(live) < n {
		n = len(live)
	}
	if n == 0 {
		return 0
	}
	maxPct := 0.0
	for i := 0; i < n; i++ {
		if backtest[i] == 0 {
			continue
		}
		pct := math.Abs(live[i]-backtest[i]) / math.Abs(backtest[i]) * 100.0
		if pct > maxPct {
			maxPct = pct
		}
	}
	return maxPct
}
