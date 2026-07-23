package backtest

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
)

// Tiered retention for matrix results (see docs/backtest_retention_policy.md).
// Only a small, high-signal slice of the parameter space is kept in full; the
// rest is demoted or aggregated so storage stays ~flat per run regardless of
// matrix size, while preserving the landscape shape and selection statistics.
const (
	RetentionT0 = 0 // island: selected + Pareto front + top-K (full artifacts, permanent)
	RetentionT1 = 1 // neighborhood/plateau (metrics + downsampled equity)
	RetentionT2 = 2 // landscape sample (metrics only)
	RetentionT3 = 3 // tail/degenerate (aggregate only — usually not persisted as a row)
)

// RetentionConfig controls classification and pruning. All env-overridable so the
// policy is tunable without a redeploy (plan §8 / retention doc §Config).
type RetentionConfig struct {
	TopK             int     // # of top-Sharpe combos always kept as T0
	PlateauBand      float64 // T1 if Sharpe >= (islandMinSharpe - PlateauBand)
	T2SampleCap      int     // max T2 rows retained (reservoir-style stride sample)
	MinTradesViable  int     // < this trades => non-viable (T3)
	T1RetentionDays  int     // TTL for T1 rows
	T2RetentionDays  int     // TTL for T2 rows
}

func DefaultRetentionConfig() RetentionConfig {
	return RetentionConfig{
		TopK:            envInt("ORCA_RETENTION_TOPK", 25),
		PlateauBand:     envFloat("ORCA_RETENTION_PLATEAU_BAND", 0.5),
		T2SampleCap:     envInt("ORCA_RETENTION_T2_CAP", 200),
		MinTradesViable: envInt("ORCA_RETENTION_MIN_TRADES", 1),
		T1RetentionDays: envInt("ORCA_RETENTION_T1_DAYS", 365),
		T2RetentionDays: envInt("ORCA_RETENTION_T2_DAYS", 90),
	}
}

func envFloat(name string, def float64) float64 {
	if v := os.Getenv(name); v != "" {
		var f float64
		if _, err := fmt.Sscanf(v, "%g", &f); err == nil {
			return f
		}
	}
	return def
}

// ComboKey is the canonical identifier of a combo in the parameter space.
func ComboKey(r ComboResult) string {
	return fmt.Sprintf("%s|%s|%s", r.StrategyID, r.Symbol, r.Timeframe)
}

func isViable(r ComboResult, cfg RetentionConfig) bool {
	return r.Error == "" && r.NumTrades >= cfg.MinTradesViable
}

// dominates reports whether a strictly dominates b across the objective vector
// (Sharpe↑, MaxDrawdown↓, ProfitFactor↑) — used to compute the Pareto front.
func dominates(a, b ComboResult) bool {
	ge := a.SharpeRatio >= b.SharpeRatio &&
		a.MaxDrawdown <= b.MaxDrawdown &&
		a.ProfitFactor >= b.ProfitFactor
	gt := a.SharpeRatio > b.SharpeRatio ||
		a.MaxDrawdown < b.MaxDrawdown ||
		a.ProfitFactor > b.ProfitFactor
	return ge && gt
}

// ClassifyResults assigns a retention class to every combo. It is a pure function
// of the result set (deterministic given inputs), so it is unit-testable and the
// executor can call it once at end-of-batch when the full distribution is known.
//
// Rules (metric-space, no parameter vectors required):
//   T3  non-viable: error or < MinTradesViable trades (the loss basin / screened out)
//   T0  island: Pareto-non-dominated set ∪ top-K by Sharpe
//   T1  plateau: viable & Sharpe >= (min island Sharpe − PlateauBand)
//   T2  the remaining viable combos, stride-sampled down to T2SampleCap
func ClassifyResults(results []ComboResult, cfg RetentionConfig) map[string]int {
	class := make(map[string]int, len(results))

	var viable []ComboResult
	for _, r := range results {
		k := ComboKey(r)
		if !isViable(r, cfg) {
			class[k] = RetentionT3
			continue
		}
		viable = append(viable, r)
	}
	if len(viable) == 0 {
		return class
	}

	// T0: Pareto front ∪ top-K by Sharpe.
	island := make(map[string]bool)
	for i := range viable {
		dominated := false
		for j := range viable {
			if i != j && dominates(viable[j], viable[i]) {
				dominated = true
				break
			}
		}
		if !dominated {
			island[ComboKey(viable[i])] = true
		}
	}
	bySharpe := make([]ComboResult, len(viable))
	copy(bySharpe, viable)
	sort.SliceStable(bySharpe, func(i, j int) bool { return bySharpe[i].SharpeRatio > bySharpe[j].SharpeRatio })
	for i := 0; i < cfg.TopK && i < len(bySharpe); i++ {
		island[ComboKey(bySharpe[i])] = true
	}

	// Plateau threshold = minimum Sharpe among island members − band.
	islandMinSharpe := math.Inf(1)
	for i := range viable {
		if island[ComboKey(viable[i])] && viable[i].SharpeRatio < islandMinSharpe {
			islandMinSharpe = viable[i].SharpeRatio
		}
	}
	plateauThreshold := islandMinSharpe - cfg.PlateauBand

	// Assign T0 / T1, collect T2 candidates.
	var t2 []ComboResult
	for _, r := range viable {
		k := ComboKey(r)
		switch {
		case island[k]:
			class[k] = RetentionT0
		case r.SharpeRatio >= plateauThreshold:
			class[k] = RetentionT1
		default:
			t2 = append(t2, r)
		}
	}

	// T2: stride-sample the remaining viable combos down to the cap; the rest are
	// demoted to T3 (aggregate only). Deterministic stride keeps a representative
	// spread across the (already Sharpe-unsorted) set.
	if len(t2) <= cfg.T2SampleCap || cfg.T2SampleCap <= 0 {
		for _, r := range t2 {
			class[ComboKey(r)] = RetentionT2
		}
	} else {
		stride := float64(len(t2)) / float64(cfg.T2SampleCap)
		kept := make(map[int]bool, cfg.T2SampleCap)
		for i := 0; i < cfg.T2SampleCap; i++ {
			kept[int(float64(i)*stride)] = true
		}
		for i, r := range t2 {
			if kept[i] {
				class[ComboKey(r)] = RetentionT2
			} else {
				class[ComboKey(r)] = RetentionT3
			}
		}
	}
	return class
}

// RunSummary is the permanent, fixed-size aggregate of an entire matrix run. It
// preserves the parameter-space shape (score distribution, viability, failure
// taxonomy, Pareto front, effective trials) so pruned rows can be discarded
// without reintroducing survivorship bias.
type RunSummary struct {
	TotalCombos     int              `json:"total_combos"`
	TradedCombos    int              `json:"traded_combos"`
	ZeroTrade       int              `json:"zero_trade"`
	Errored         int              `json:"errored"`
	EffectiveTrials int              `json:"effective_trials"`
	ScoreHistogram  map[string]int   `json:"score_histogram"`
	Viability       map[string]int   `json:"viability"`
	FailureReasons  map[string]int   `json:"failure_reasons"`
	ParetoFront     []string         `json:"pareto_front"`
	BestSharpe      float64          `json:"best_sharpe"`
	BestCombo       string           `json:"best_combo"`
}

// sharpeBucket maps a Sharpe value to a coarse histogram bucket label.
func sharpeBucket(s float64) string {
	switch {
	case math.IsNaN(s) || math.IsInf(s, 0):
		return "nan"
	case s < -2:
		return "<-2"
	case s < -1:
		return "-2..-1"
	case s < 0:
		return "-1..0"
	case s < 0.5:
		return "0..0.5"
	case s < 1:
		return "0.5..1"
	case s < 1.5:
		return "1..1.5"
	case s < 2:
		return "1.5..2"
	default:
		return ">=2"
	}
}

// BuildRunSummary computes the permanent aggregate for a matrix run.
func BuildRunSummary(results []ComboResult, cfg RetentionConfig) RunSummary {
	sum := RunSummary{
		ScoreHistogram: map[string]int{},
		Viability:      map[string]int{},
		FailureReasons: map[string]int{},
		ParetoFront:    []string{},
		BestSharpe:     math.Inf(-1),
	}
	sum.TotalCombos = len(results)

	var viable []ComboResult
	for _, r := range results {
		sum.ScoreHistogram[sharpeBucket(r.SharpeRatio)]++
		switch {
		case r.Error != "":
			sum.Errored++
			sum.Viability["errored"]++
			sum.FailureReasons[normalizeReason(r.Error)]++
		case r.NumTrades < cfg.MinTradesViable:
			sum.ZeroTrade++
			sum.Viability["zero_trade"]++
			sum.FailureReasons["zero_trades"]++
		default:
			sum.TradedCombos++
			sum.Viability["traded"]++
			viable = append(viable, r)
			if r.SharpeRatio > sum.BestSharpe {
				sum.BestSharpe = r.SharpeRatio
				sum.BestCombo = ComboKey(r)
			}
		}
	}
	sum.EffectiveTrials = sum.TradedCombos

	for i := range viable {
		dominated := false
		for j := range viable {
			if i != j && dominates(viable[j], viable[i]) {
				dominated = true
				break
			}
		}
		if !dominated {
			sum.ParetoFront = append(sum.ParetoFront, ComboKey(viable[i]))
		}
	}
	sort.Strings(sum.ParetoFront)
	if math.IsInf(sum.BestSharpe, -1) {
		sum.BestSharpe = 0
	}
	return sum
}

// normalizeReason collapses an error string into a coarse failure-taxonomy key so
// the FailureReasons tally stays small (one exemplar class per reason).
func normalizeReason(err string) string {
	switch {
	case err == "":
		return "none"
	case containsAny(err, "no candle data", "no data"):
		return "no_data"
	case containsAny(err, "context", "deadline", "timeout"):
		return "timeout"
	case containsAny(err, "hash"):
		return "hash_mismatch"
	default:
		return "other"
	}
}

func containsAny(s string, subs ...string) bool {
	ls := strings.ToLower(s)
	for _, sub := range subs {
		if sub != "" && strings.Contains(ls, strings.ToLower(sub)) {
			return true
		}
	}
	return false
}
