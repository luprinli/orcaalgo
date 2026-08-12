package backtest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	strategy "github.com/lee-econ/orca-core/internal/strategy"
)

// LightOptimizeConfig parameterizes a single strategy's bounded, train/test-split
// parameter sweep (docs/per_strategy_optimization_in_matrix.md §5). It is a plain
// data struct so RunLightOptimize is a pure function of its inputs — unit-testable
// with no API, progress store, or global-state dependencies.
type LightOptimizeConfig struct {
	StrategyID         string
	Symbols            []string // representative set (<= N)
	ValidationSymbols  []string // remaining symbols for the OOS fallback check
	Timeframe          string
	StartDate          time.Time
	EndDate            time.Time
	TrainFraction      float64 // fraction of the window used for training (default 0.67)
	InitialCapital     float64
	DataSource         string
	MaxCombos          int           // max candidate param sets (default 24)
	PerBacktestTimeout time.Duration // per-backtest deadline (default 10s)
	PropFirmEnabled    bool
	GateProfile        string
	SizingPercent      float64
	ObjectiveWeights   [3]float64 // {Sharpe, Drawdown, ProfitFactor}
	PlateauPatience    int        // early stop after N stale combos (default 5)
	EnableCache        bool
	CacheTTL           time.Duration // cache entry lifetime (default 168h)
	RandomSeed         int64         // fixed seed for reproducible sampling (default 1)
}

// ---- Environment-configurable defaults (§6) ---------------------------------

func envFloat64(name string, def float64) float64 {
	if v := os.Getenv(name); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

// LightOptBudget is the maximum number of random candidate parameter sets tried
// per strategy in the light sweep (ORCA_LIGHT_OPT_BUDGET, default 24).
func LightOptBudget() int { return envInt("ORCA_LIGHT_OPT_BUDGET", 24) }

// LightOptSymbolCount is the number of representative symbols used for the sweep
// (ORCA_LIGHT_OPT_SYMBOLS, default 4).
func LightOptSymbolCount() int { return envInt("ORCA_LIGHT_OPT_SYMBOLS", 4) }

// LightOptWindowMonths is the length of the optimization date window in months
// (ORCA_LIGHT_OPT_WINDOW_MONTHS, default 3).
func LightOptWindowMonths() int { return envInt("ORCA_LIGHT_OPT_WINDOW_MONTHS", 3) }

// LightOptTimeout is the per-backtest deadline (ORCA_LIGHT_OPT_TIMEOUT_S, default 10s).
func LightOptTimeout() time.Duration {
	return time.Duration(envInt("ORCA_LIGHT_OPT_TIMEOUT_S", 10)) * time.Second
}

// LightOptPlateauPatience is the early-stop threshold: stop after N consecutive
// candidates with no best-score improvement (ORCA_LIGHT_OPT_PLATEAU_PATIENCE, 5).
func LightOptPlateauPatience() int { return envInt("ORCA_LIGHT_OPT_PLATEAU_PATIENCE", 5) }

// LightOptTrainFraction is the fraction of the window used for training; the rest
// is held out for out-of-sample scoring (ORCA_LIGHT_OPT_TRAIN_FRACTION, 0.80).
func LightOptTrainFraction() float64 { return envFloat64("ORCA_LIGHT_OPT_TRAIN_FRACTION", 0.80) }

// LightOptCacheTTL is the result-cache entry lifetime (ORCA_LIGHT_OPT_CACHE_TTL_HOURS, 168).
func LightOptCacheTTL() time.Duration {
	return time.Duration(envInt("ORCA_LIGHT_OPT_CACHE_TTL_HOURS", 168)) * time.Hour
}

// LightOptTFOverride returns the user-forced optimization timeframe, or "" if the
// caller should choose automatically (ORCA_LIGHT_OPT_TF).
func LightOptTFOverride() string { return os.Getenv("ORCA_LIGHT_OPT_TF") }

// LightOptWeights returns the composite-objective weights {Sharpe, Drawdown,
// ProfitFactor} (ORCA_LIGHT_OPT_WEIGHTS as CSV, default 0.5,0.3,0.2).
func LightOptWeights() [3]float64 {
	def := [3]float64{0.5, 0.3, 0.2}
	v := os.Getenv("ORCA_LIGHT_OPT_WEIGHTS")
	if v == "" {
		return def
	}
	parts := strings.Split(v, ",")
	if len(parts) != 3 {
		return def
	}
	var out [3]float64
	for i, p := range parts {
		f, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return def
		}
		out[i] = f
	}
	return out
}

// ---- Result cache (§4) ------------------------------------------------------

type optCacheEntry struct {
	params  map[string]float64
	expires time.Time
}

var (
	optCacheMu sync.Mutex
	optCache   = make(map[string]optCacheEntry)
)

// lightOptCacheKey derives a stable key from the inputs that determine a sweep's
// result: SHA-256(strategyID + sorted(symbols) + start + end + timeframe).
func lightOptCacheKey(cfg LightOptimizeConfig) string {
	syms := append([]string(nil), cfg.Symbols...)
	sort.Strings(syms)
	h := sha256.New()
	fmt.Fprintf(h, "%s|%s|%s|%s|%s",
		cfg.StrategyID,
		strings.Join(syms, ","),
		cfg.StartDate.UTC().Format(time.RFC3339),
		cfg.EndDate.UTC().Format(time.RFC3339),
		cfg.Timeframe,
	)
	return hex.EncodeToString(h.Sum(nil))
}

func lightOptCacheGet(key string) (map[string]float64, bool) {
	optCacheMu.Lock()
	defer optCacheMu.Unlock()
	e, ok := optCache[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(e.expires) {
		delete(optCache, key)
		return nil, false
	}
	return cloneParams(e.params), true
}

func lightOptCachePut(key string, params map[string]float64, ttl time.Duration) {
	if ttl <= 0 {
		ttl = LightOptCacheTTL()
	}
	optCacheMu.Lock()
	optCache[key] = optCacheEntry{params: cloneParams(params), expires: time.Now().Add(ttl)}
	optCacheMu.Unlock()
}

// ResetLightOptCache clears the in-memory optimization cache (test helper).
func ResetLightOptCache() {
	optCacheMu.Lock()
	optCache = make(map[string]optCacheEntry)
	optCacheMu.Unlock()
}

func cloneParams(p map[string]float64) map[string]float64 {
	if p == nil {
		return nil
	}
	out := make(map[string]float64, len(p))
	for k, v := range p {
		out[k] = v
	}
	return out
}

// ---- Core ------------------------------------------------------------------

// RunLightOptimize runs a small, bounded, train/test-split parameter sweep for ONE
// strategy and returns the best-found params map, or nil if the search produced
// nothing or the out-of-sample validation fallback rejected the result. The caller
// falls back to strategy/registry defaults on a nil return.
//
// It is sequential (one sub-backtest at a time) with per-backtest timeouts and heap
// admission control, so it cannot exhaust memory the way the naive concurrent sweep
// did (§7 post-mortem). All tuning knobs default from environment variables when the
// corresponding config field is zero (§6).
func RunLightOptimize(ctx context.Context, db Database, cfg LightOptimizeConfig) map[string]float64 {
	applyLightOptDefaults(&cfg)

	space := DefaultSearchSpace(cfg.StrategyID)
	if !validSearchSpace(space) {
		return nil // nothing meaningful to optimize — clean no-op
	}

	if cfg.EnableCache {
		key := lightOptCacheKey(cfg)
		if params, ok := lightOptCacheGet(key); ok {
			return params
		}
	}

	candidates := generateCandidates(space, cfg.MaxCombos, cfg.RandomSeed)
	if len(candidates) == 0 {
		return nil
	}

	trainStart, trainEnd, testStart, testEnd := splitWindow(cfg.StartDate, cfg.EndDate, cfg.TrainFraction)

	estimatedBars := estimateBarCount(cfg.Timeframe, trainStart, trainEnd)
	if estimatedBars < 500 {
		return nil
	}
	weights := weightMap(cfg.ObjectiveWeights)

	var (
		bestScore  = math.Inf(-1)
		bestParams map[string]float64
		stale      int
	)
	for _, cand := range candidates {
		if ctx.Err() != nil {
			break
		}
		score, ok := evalLightParams(ctx, db, cfg, cand, cfg.Symbols, trainStart, trainEnd, weights)
		if !ok {
			continue
		}
		if score > bestScore+1e-9 {
			bestScore = score
			bestParams = cloneParams(cand)
			stale = 0
		} else {
			stale++
			if cfg.PlateauPatience > 0 && stale >= cfg.PlateauPatience {
				break // early stop on performance plateau (§2.6)
			}
		}
	}
	if bestParams == nil {
		return nil
	}

	// Out-of-sample validation fallback (§3.3): score the winner and the registry
	// defaults on the held-out test period, using validation symbols where present.
	// If the optimised params do not beat defaults out-of-sample, revert to defaults.
	valSyms := cfg.ValidationSymbols
	if len(valSyms) == 0 {
		valSyms = cfg.Symbols
	}
	bestTest, okBest := evalLightParams(ctx, db, cfg, bestParams, valSyms, testStart, testEnd, weights)
	if !okBest {
		return nil
	}
	defParams := registryDefaultParams(cfg.StrategyID)
	defTest, okDef := evalLightParams(ctx, db, cfg, defParams, valSyms, testStart, testEnd, weights)
	if okDef && bestTest < defTest {
		return nil // optimised params degrade out-of-sample — fall back to defaults
	}

	if cfg.EnableCache {
		lightOptCachePut(lightOptCacheKey(cfg), bestParams, cfg.CacheTTL)
	}
	return bestParams
}

func applyLightOptDefaults(cfg *LightOptimizeConfig) {
	if cfg.MaxCombos <= 0 {
		cfg.MaxCombos = LightOptBudget()
	}
	if cfg.PerBacktestTimeout <= 0 {
		cfg.PerBacktestTimeout = LightOptTimeout()
	}
	if cfg.PlateauPatience == 0 {
		cfg.PlateauPatience = LightOptPlateauPatience()
	}
	if cfg.TrainFraction <= 0 || cfg.TrainFraction >= 1 {
		cfg.TrainFraction = LightOptTrainFraction()
	}
	if cfg.ObjectiveWeights == [3]float64{} {
		cfg.ObjectiveWeights = LightOptWeights()
	}
	if cfg.CacheTTL <= 0 {
		cfg.CacheTTL = LightOptCacheTTL()
	}
	if cfg.RandomSeed == 0 {
		cfg.RandomSeed = 1
	}
	if cfg.InitialCapital <= 0 {
		cfg.InitialCapital = 100000.0
	}
}

// validSearchSpace enforces the §3.5 preconditions: a non-nil space with at least
// two optimisable dimensions (min<max, step>0, or a multi-value categorical).
func validSearchSpace(space SearchSpace) bool {
	if space == nil {
		return false
	}
	valid := 0
	for _, c := range space {
		if c.Type == ParamCategorical {
			if len(c.CategoricalValues) > 1 {
				valid++
			}
			continue
		}
		if c.Min < c.Max && c.Step > 0 {
			valid++
		}
	}
	return valid >= 2
}

// generateCandidates enumerates the full grid when it is no larger than the budget
// (§3.5), otherwise draws a reproducible random sample of MaxCombos sets.
func generateCandidates(space SearchSpace, budget int, seed int64) []map[string]float64 {
	if budget <= 0 {
		budget = 24
	}
	if space.TotalCombinations() <= budget {
		return space.GenerateAllCombinations()
	}
	return space.GenerateRandomCombinations(budget, seed)
}

// splitWindow computes the train/test mini walk-forward boundaries (§2.1). When the
// window is degenerate (too short to split), the full window is used for both so the
// optimizer still returns a result rather than failing.
func splitWindow(start, end time.Time, trainFraction float64) (trainStart, trainEnd, testStart, testEnd time.Time) {
	total := end.Sub(start)
	if total <= 0 {
		return start, end, start, end
	}
	trainDur := time.Duration(float64(total) * trainFraction)
	trainEnd = start.Add(trainDur)
	if !trainEnd.After(start) || !end.After(trainEnd) {
		return start, end, start, end
	}
	return start, trainEnd, trainEnd, end
}

func weightMap(w [3]float64) map[ObjectiveType]float64 {
	return map[ObjectiveType]float64{
		ObjectiveSharpe:       w[0],
		ObjectiveMinDD:        w[1],
		ObjectiveProfitFactor: w[2],
	}
}

// evalLightParams runs one bounded backtest for a candidate parameter set over the
// given period and returns its composite score. It applies the same prop-firm rules
// as the main matrix and penalises parameter sets that breach prop-firm limits (§3.4).
func evalLightParams(ctx context.Context, db Database, cfg LightOptimizeConfig,
	params map[string]float64, symbols []string, start, end time.Time,
	weights map[ObjectiveType]float64) (float64, bool) {

	if len(symbols) == 0 || !end.After(start) {
		return 0, false
	}
	if ctx.Err() != nil {
		return 0, false
	}

	btCfg := BacktestConfig{
		StrategyID:      cfg.StrategyID,
		Symbols:         symbols,
		StartDate:       start,
		EndDate:         end,
		InitialCapital:  cfg.InitialCapital,
		Timeframe:       cfg.Timeframe,
		DataSource:      cfg.DataSource,
		PropFirmEnabled: cfg.PropFirmEnabled,
		StopLoss:        &StopLossConfig{Type: "atr", ATRPeriod: 14, ATRMultiplier: 2.0},
		TakeProfit:      &TakeProfitConfig{Type: "risk_reward", RRRatio: 2.0},
		ApplyGate:       cfg.GateProfile != "" && cfg.GateProfile != "none",
		GateProfile:     cfg.GateProfile,
		SizingPercent:   cfg.SizingPercent,
	}
	// Map universal params (sizing_percent, kelly_fraction) onto config fields and
	// the rest onto the strategy runner, exactly as the optimization pipeline does.
	ApplyOptimizationParams(&btCfg, params)

	AwaitHeadroom(ctx)
	bctx, cancel := context.WithTimeout(ctx, cfg.PerBacktestTimeout)
	defer cancel()

	eng := NewEngine(db)
	eng.WirePipeline()
	res, err := eng.Run(bctx, btCfg)
	if err != nil || res == nil {
		return 0, false
	}

	score := ComputeObjective(res, ObjectiveComposite, weights)
	if cfg.PropFirmEnabled && res.ComplianceReport != nil && res.ComplianceReport.NumBreaches > 0 {
		// Disfavour risk-incompatible parameter sets (§3.4). Monotone for both signs.
		if score > 0 {
			score *= 0.1
		} else {
			score -= 1.0
		}
	}
	return score, true
}

// registryDefaultParams returns the registered default params for a strategy, or an
// empty map if the strategy is unknown.
func registryDefaultParams(strategyID string) map[string]float64 {
	runner := strategy.GlobalRegistry().Get(strategyID)
	if runner == nil {
		return map[string]float64{}
	}
	return runner.Params()
}

func estimateBarCount(timeframe string, start, end time.Time) int {
	days := end.Sub(start).Hours() / 24
	if days <= 0 {
		return 0
	}
	barsPerDay := barsPerDayFromTimeframe(timeframe)
	return int(days * barsPerDay)
}

// ParamSensitivityResult holds per-parameter sensitivity scores and stability
// classifications for a single strategy's search space.
type ParamSensitivityResult struct {
	StrategyID   string
	Weights      [3]float64
	Scores       map[string]float64          // param name -> normalized sensitivity [0,1]
	Stability    map[string]string           // param name -> "robust"|"moderate"|"sensitive"
	BestKnown    map[string]float64          // best params from full sweep
	Evaluations  int                         // total sub-backtests run
	Errors       []string
}

// SensitivityReport aggregates sensitivity results across strategies.
type SensitivityReport struct {
	Results      []ParamSensitivityResult
	GeneratedAt  time.Time
	ConfigHash   string
}

// RunParameterSensitivity evaluates each optimisable dimension independently to
// produce a per-parameter sensitivity score. Parameters with flat optimum regions
// are tagged "robust"; those with steep gradients are tagged "sensitive" and
// flagged as risk-prone for institutional deployment.
//
// The function works by perturbing one parameter at a time around the best-known
// params and measuring degradation in the composite objective. If no best params
// are known, it sweeps the full grid and scores by output variance.
func RunParameterSensitivity(ctx context.Context, db Database, cfg LightOptimizeConfig) ParamSensitivityResult {
	applyLightOptDefaults(&cfg)

	space := DefaultSearchSpace(cfg.StrategyID)
	if !validSearchSpace(space) {
		return ParamSensitivityResult{
			StrategyID: cfg.StrategyID,
			Errors:     []string{"search space invalid or too small"},
		}
	}

	weights := weightMap(cfg.ObjectiveWeights)
	result := ParamSensitivityResult{
		StrategyID: cfg.StrategyID,
		Weights:    cfg.ObjectiveWeights,
		Scores:     make(map[string]float64),
		Stability:  make(map[string]string),
		BestKnown:  make(map[string]float64),
	}

	best := RunLightOptimize(ctx, db, cfg)
	if best == nil {
		best = registryDefaultParams(cfg.StrategyID)
	}
	result.BestKnown = best

	paramNames := sortedParamNames(space)
	if len(paramNames) == 0 {
		return result
	}

	start, end := cfg.StartDate, cfg.EndDate
	if start.IsZero() || end.IsZero() {
		start = time.Now().AddDate(0, -3, 0)
		end = time.Now()
	}

	baselineScore, ok := evalLightParams(ctx, db, cfg, best, cfg.Symbols, start, end, weights)
	if !ok {
		result.Errors = append(result.Errors, "baseline evaluation failed")
		return result
	}
	result.Evaluations++

	for _, pName := range paramNames {
		pCol, ok := space[pName]
		if !ok {
			continue
		}

		var testValues []float64
		switch pCol.Type {
		case ParamCategorical:
			testValues = make([]float64, len(pCol.CategoricalValues))
			for i, v := range pCol.CategoricalValues {
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					testValues[i] = f
				}
			}
		default:
			base := pCol.Min
			if v, ok := best[pName]; ok {
				base = v
			}
			testValues = []float64{}
			mid := pCol.Min + (pCol.Max-pCol.Min)*0.5
			for _, v := range []float64{pCol.Min, base - pCol.Step*2, base, base + pCol.Step*2, pCol.Max, mid} {
				if v >= pCol.Min && v <= pCol.Max && !floatIn(v, testValues) {
					testValues = append(testValues, v)
				}
			}
		}

		scores := make([]float64, len(testValues))
		for i, val := range testValues {
			perturb := cloneParams(best)
			perturb[pName] = val

			score, ok2 := evalLightParams(ctx, db, cfg, perturb, cfg.Symbols, start, end, weights)
			if ok2 {
				scores[i] = score
			}
			result.Evaluations++
		}

		if len(scores) < 2 {
			continue
		}

		var minS, maxS float64 = math.Inf(1), math.Inf(-1)
		for _, s := range scores {
			if s < minS {
				minS = s
			}
			if s > maxS {
				maxS = s
			}
		}

		degradation := 0.0
		if baselineScore != 0 {
			for _, s := range scores {
				d := math.Abs(s-baselineScore) / math.Max(math.Abs(baselineScore), 1e-9)
				if d > degradation {
					degradation = d
				}
			}
		} else {
			rangeNorm := 0.0
			if math.Abs(baselineScore) > 1e-9 {
				rangeNorm = (maxS - minS) / math.Abs(baselineScore)
			} else {
				rangeNorm = maxS - minS
			}
			degradation = rangeNorm
		}

		result.Scores[pName] = math.Min(1.0, degradation)

		switch {
		case degradation < 0.10:
			result.Stability[pName] = "robust"
		case degradation < 0.30:
			result.Stability[pName] = "moderate"
		default:
			result.Stability[pName] = "sensitive"
		}
	}

	return result
}

// GenerateSensitivityReport runs parameter sensitivity analysis for all strategies
// with valid search spaces and returns an aggregated report.
func GenerateSensitivityReport(ctx context.Context, db Database, symbols []string, start, end time.Time, timeframe string) SensitivityReport {
	report := SensitivityReport{
		GeneratedAt: time.Now().UTC(),
		Results:     make([]ParamSensitivityResult, 0),
	}
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s-%s-%s", start.Format(time.RFC3339), end.Format(time.RFC3339), timeframe)))
	report.ConfigHash = hex.EncodeToString(h.Sum(nil))[:16]

	for _, runner := range strategy.GlobalRegistry().All() {
		sID := runner.Name()
		space := DefaultSearchSpace(sID)
		if !validSearchSpace(space) {
			continue
		}
		cfg := LightOptimizeConfig{
			StrategyID:    sID,
			Symbols:       symbols,
			StartDate:     start,
			EndDate:       end,
			Timeframe:     timeframe,
			TrainFraction: 0.80,
		}
		r := RunParameterSensitivity(ctx, db, cfg)
		report.Results = append(report.Results, r)
	}
	return report
}

func floatIn(v float64, vals []float64) bool {
	for _, x := range vals {
		if math.Abs(x-v) < 1e-9 {
			return true
		}
	}
	return false
}
