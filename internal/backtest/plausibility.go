package backtest

import (
	"fmt"
	"sort"
)

// PlausibilityFlag describes a single implausible result pattern detected in a
// matrix of backtest combos. It operationalizes the artifact checks previously
// performed manually in the readiness audit (Sharpe/PF/WR ceilings, long/short
// asymmetry, sentinel values, and timeframe deduplication).
type PlausibilityFlag struct {
	Code       string `json:"code"`
	StrategyID string `json:"strategy_id"`
	Symbol     string `json:"symbol"`
	Timeframe  string `json:"timeframe"`
	Message    string `json:"message"`
}

// Plausibility thresholds. Values beyond these are almost always the signature
// of a data or execution artifact, not a real tradable edge.
const (
	plausibleSharpeCeiling   = 3.0
	plausibleSharpeFloor     = -3.0
	plausibleProfitFactorMax = 10.0
	plausibleWinRateMax      = 90.0
	plausibleWinRateMin      = 5.0
	plausibleReturnMaxPct    = 1000.0
	plausibleReturnMinPct    = -100.0
)

// FlagImplausibleCombos scans a full matrix result set and returns every
// implausible pattern it finds. It is a pure function of the results so it can
// be unit-tested and reused by both the API matrix path and the CLI runner.
func FlagImplausibleCombos(results []ComboResult) []PlausibilityFlag {
	var flags []PlausibilityFlag

	// Per-combo metric ceilings.
	for _, r := range results {
		flags = append(flags, comboPlausibilityFlags(r)...)
	}

	// Timeframe deduplication: identical trade counts across every timeframe for
	// a given strategy+symbol means the timeframe filter is not actually applied
	// (the engine is evaluating the same candle series on every timeframe).
	type key struct{ strategy, symbol string }
	byKey := map[key]map[int]int{}
	tfCount := map[key]int{}
	for _, r := range results {
		k := key{r.StrategyID, r.Symbol}
		if byKey[k] == nil {
			byKey[k] = map[int]int{}
		}
		byKey[k][r.NumTrades]++
		tfCount[k]++
	}
	keys := make([]key, 0, len(byKey))
	for k := range byKey {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].strategy != keys[j].strategy {
			return keys[i].strategy < keys[j].strategy
		}
		return keys[i].symbol < keys[j].symbol
	})
	for _, k := range keys {
		distinct := byKey[k]
		if tfCount[k] < 2 || len(distinct) > 1 {
			continue
		}
		for tc := range distinct {
			if tc == 0 {
				continue // all-zero is a coverage issue, not a timeframe-dedup issue
			}
			flags = append(flags, PlausibilityFlag{
				Code: "timeframe_dedup", StrategyID: k.strategy, Symbol: k.symbol,
				Message: fmt.Sprintf("identical trade count (%d) across %d timeframes — timeframe filter not applied", tc, tfCount[k]),
			})
		}
	}

	return flags
}

// comboPlausibilityFlags returns the per-combo implausibility flags for a
// single result, excluding cross-combo checks (timeframe deduplication).
func comboPlausibilityFlags(r ComboResult) []PlausibilityFlag {
	if r.NumTrades == 0 {
		// Zero-trade combos carry no metrics to judge; data-coverage gaps
		// are surfaced separately by the engine's coverage guard.
		return nil
	}
	var flags []PlausibilityFlag
	if r.SharpeRatio > plausibleSharpeCeiling {
		flags = append(flags, PlausibilityFlag{
			Code: "sharpe_implausible", StrategyID: r.StrategyID, Symbol: r.Symbol, Timeframe: r.Timeframe,
			Message: fmt.Sprintf("Sharpe %.2f exceeds plausible ceiling %.1f", r.SharpeRatio, plausibleSharpeCeiling),
		})
	}
	if r.SharpeRatio < plausibleSharpeFloor {
		flags = append(flags, PlausibilityFlag{
			Code: "sharpe_implausible", StrategyID: r.StrategyID, Symbol: r.Symbol, Timeframe: r.Timeframe,
			Message: fmt.Sprintf("Sharpe %.2f below plausible floor %.1f", r.SharpeRatio, plausibleSharpeFloor),
		})
	}
	if r.ProfitFactor > plausibleProfitFactorMax {
		flags = append(flags, PlausibilityFlag{
			Code: "profit_factor_implausible", StrategyID: r.StrategyID, Symbol: r.Symbol, Timeframe: r.Timeframe,
			Message: fmt.Sprintf("ProfitFactor %.2f exceeds plausible ceiling %.1f", r.ProfitFactor, plausibleProfitFactorMax),
		})
	}
	if r.WinRate > plausibleWinRateMax || r.WinRate < plausibleWinRateMin {
		flags = append(flags, PlausibilityFlag{
			Code: "win_rate_implausible", StrategyID: r.StrategyID, Symbol: r.Symbol, Timeframe: r.Timeframe,
			Message: fmt.Sprintf("WinRate %.2f%% outside plausible range [%.1f, %.1f]", r.WinRate, plausibleWinRateMin, plausibleWinRateMax),
		})
	}
	if r.LongTrades > 0 && r.LongWinRate <= 0 {
		flags = append(flags, PlausibilityFlag{
			Code: "directional_asymmetry", StrategyID: r.StrategyID, Symbol: r.Symbol, Timeframe: r.Timeframe,
			Message: fmt.Sprintf("%d long trades but 0%% long win rate — one-sided fills", r.LongTrades),
		})
	}
	if r.ShortPF >= 999 {
		flags = append(flags, PlausibilityFlag{
			Code: "sentinel_pf", StrategyID: r.StrategyID, Symbol: r.Symbol, Timeframe: r.Timeframe,
			Message: "ShortPF is a sentinel value (>=999) indicating zero short losses",
		})
	}
	if r.TotalReturn > plausibleReturnMaxPct || r.TotalReturn < plausibleReturnMinPct {
		flags = append(flags, PlausibilityFlag{
			Code: "return_implausible", StrategyID: r.StrategyID, Symbol: r.Symbol, Timeframe: r.Timeframe,
			Message: fmt.Sprintf("Return %.2f%% outside plausible range [%.1f, %.1f]", r.TotalReturn, plausibleReturnMinPct, plausibleReturnMaxPct),
		})
	}
	return flags
}

// IsComboImplausible reports whether a single combo violates any per-combo
// plausibility ceiling. Cross-combo checks (timeframe deduplication) are
// handled separately by FlagImplausibleCombos.
func IsComboImplausible(r ComboResult) bool {
	return len(comboPlausibilityFlags(r)) > 0
}
