package backtest

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"

	strategy "github.com/lee-econ/orca-core/internal/strategy"
)

type ParamType string

const (
	ParamContinuous  ParamType = "continuous"
	ParamInteger     ParamType = "integer"
	ParamCategorical ParamType = "categorical"
)

type ParamConstraint struct {
	Name         string
	Type         ParamType
	Min          float64
	Max          float64
	Step         float64
	CategoricalValues []string
	Condition   *ConditionalRule
}

type ConditionalRule struct {
	LeftParam  string
	Operator   string
	RightParam string
}

func (c ParamConstraint) Grid() []float64 {
	if c.Type == ParamCategorical {
		vals := make([]float64, len(c.CategoricalValues))
		for i := range c.CategoricalValues {
			vals[i] = float64(i)
		}
		return vals
	}
	if c.Step <= 0 {
		c.Step = 1.0
	}
	var vals []float64
	for v := c.Min; v <= c.Max+1e-9; v += c.Step {
		vals = append(vals, math.Round(v*1e6)/1e6)
	}
	return vals
}

type SearchSpace map[string]ParamConstraint

func (s SearchSpace) TotalCombinations() int {
	total := 1
	for _, c := range s {
		total *= len(c.Grid())
	}
	return total
}

func (s SearchSpace) checkCondition(combo map[string]float64) bool {
	for _, c := range s {
		if c.Condition == nil {
			continue
		}
		leftVal, lok := combo[c.Condition.LeftParam]
		rightVal, rok := combo[c.Condition.RightParam]
		if !lok || !rok {
			continue
		}
		switch c.Condition.Operator {
		case "lt":
			if !(leftVal < rightVal) {
				return false
			}
		case "gt":
			if !(leftVal > rightVal) {
				return false
			}
		case "lte":
			if !(leftVal <= rightVal) {
				return false
			}
		case "gte":
			if !(leftVal >= rightVal) {
				return false
			}
		}
	}
	return true
}

func (s SearchSpace) GenerateAllCombinations() []map[string]float64 {
	raw := s.generateAllRaw()
	filtered := make([]map[string]float64, 0, len(raw))
	for _, combo := range raw {
		if s.checkCondition(combo) {
			filtered = append(filtered, combo)
		}
	}
	return filtered
}

func (s SearchSpace) generateAllRaw() []map[string]float64 {
	var names []string
	for name := range s {
		names = append(names, name)
	}
	sort.Strings(names)

	grids := make([][]float64, len(names))
	for i, name := range names {
		grids[i] = s[name].Grid()
	}

	var results []map[string]float64
	indices := make([]int, len(names))

	for {
		combo := make(map[string]float64, len(names))
		for i, name := range names {
			if s[name].Type == ParamInteger {
				combo[name] = float64(int(grids[i][indices[i]]))
			} else {
				combo[name] = grids[i][indices[i]]
			}
		}
		results = append(results, combo)

		pos := len(names) - 1
		for pos >= 0 {
			indices[pos]++
			if indices[pos] < len(grids[pos]) {
				break
			}
			indices[pos] = 0
			pos--
		}
		if pos < 0 {
			break
		}
	}

	return results
}

func (s SearchSpace) GenerateRandomCombinations(count int, seed int64) []map[string]float64 {
	rng := rand.New(rand.NewSource(seed))
	var names []string
	for name := range s {
		names = append(names, name)
	}
	sort.Strings(names)

	grids := make([][]float64, len(names))
	for i, name := range names {
		grids[i] = s[name].Grid()
	}

	maxAttempts := count * 10
	seen := make(map[string]bool)
	results := make([]map[string]float64, 0, count)

	for len(results) < count && maxAttempts > 0 {
		maxAttempts--
		combo := make(map[string]float64, len(names))
		keyParts := make([]string, len(names))
		for i, name := range names {
			idx := rng.Intn(len(grids[i]))
			val := grids[i][idx]
			if s[name].Type == ParamInteger {
				combo[name] = float64(int(val))
			} else {
				combo[name] = val
			}
			keyParts[i] = fmt.Sprintf("%s=%.4f", name, combo[name])
		}
		key := strings.Join(keyParts, ";")
		if seen[key] {
			continue
		}
		if !s.checkCondition(combo) {
			continue
		}
		seen[key] = true
		results = append(results, combo)
	}
	return results
}

func (s SearchSpace) GenerateCombinations(method SearchMethod, budget int, seed int64) []map[string]float64 {
	switch method {
	case SearchRandom:
		return s.GenerateRandomCombinations(budget, seed)
	case SearchGrid:
		all := s.GenerateAllCombinations()
		if budget > 0 && len(all) > budget {
			return all[:budget]
		}
		return all
	default:
		return s.GenerateAllCombinations()
	}
}

type ObjectiveType string

const (
	ObjectiveSharpe      ObjectiveType = "sharpe"
	ObjectiveSortino     ObjectiveType = "sortino"
	ObjectiveProfitFactor ObjectiveType = "profit_factor"
	ObjectiveWinRate     ObjectiveType = "win_rate"
	ObjectiveMinDD       ObjectiveType = "min_drawdown"
	ObjectiveDDRatio     ObjectiveType = "sharpe_over_dd"
	ObjectiveComposite   ObjectiveType = "composite"
)

type SearchMethod string

const (
	SearchGrid    SearchMethod = "grid"
	SearchRandom  SearchMethod = "random"
	SearchBayesian SearchMethod = "bayesian"
)

type OptimizationConfig struct {
	StrategyID      string
	SearchSpace     SearchSpace
	ObjectiveType   ObjectiveType
	MaxCombinations int
	ObjectiveWeights map[ObjectiveType]float64
	SearchMethod    SearchMethod
	RandomSeed      int64
}

type OptimizationResult struct {
	BestParams        map[string]float64
	BestScore         float64
	TotalCombinations int
	Evaluated          int
}

type ParamScore struct {
	Params map[string]float64
	Score  float64
}

func ComputeObjective(result *BacktestResult, objType ObjectiveType, weights map[ObjectiveType]float64) float64 {
	if result.NumTrades < 30 {
		return -1e6
	}
	switch objType {
	case ObjectiveSharpe:
		return result.SharpeRatio
	case ObjectiveSortino:
		return result.SortinoRatio
	case ObjectiveProfitFactor:
		return result.ProfitFactor
	case ObjectiveWinRate:
		return result.WinRate
	case ObjectiveMinDD:
		return -result.MaxDrawdown
	case ObjectiveDDRatio:
		if result.MaxDrawdown > 0 {
			return result.SharpeRatio / result.MaxDrawdown * 100
		}
		return result.SharpeRatio
	case ObjectiveComposite:
		score := 0.0
		if result.NumTrades < 30 {
			return -1e6
		}
		if w, ok := weights[ObjectiveSharpe]; ok {
			score += w * (result.SharpeRatio / 2.0)
		}
		if w, ok := weights[ObjectiveProfitFactor]; ok {
			score += w * math.Min(result.ProfitFactor/2.0, 1.0)
		}
		if w, ok := weights[ObjectiveWinRate]; ok {
			score += w * (result.WinRate / 100.0)
		}
		if w, ok := weights[ObjectiveMinDD]; ok {
			ddScore := 1.0
			if result.MaxDrawdown > 0 {
				ddScore = math.Max(0, 1.0-result.MaxDrawdown/20.0)
			}
			score += w * ddScore
		}
		return score
	default:
		return result.SharpeRatio
	}
}

func DefaultSearchSpace(strategyID string) SearchSpace {
	space := defaultStrategySearchSpace(strategyID)
	if space == nil {
		// Unknown strategy: no search space (nothing to optimize, including sizing).
		return nil
	}
	return addUniversalParams(space)
}

// UniversalParamDefs are optimization parameters that apply to EVERY strategy
// regardless of type. They are injected into every search space so the optimizer
// can find an optimal account-level position size alongside strategy-specific
// technical parameters. These map to BacktestConfig fields (not strategy runner
// params) and are extracted by ApplyOptimizationParams.
func UniversalParamDefs() []strategy.ParamDef {
	return []strategy.ParamDef{
		{
			Name: "sizing_percent", Type: strategy.ParamContinuous,
			Default: 0.02, Min: 0.005, Max: 0.10, Step: 0.005,
			Group: "Sizing", Description: "Fraction of account capital risked per trade",
		},
		{
			Name: "kelly_fraction", Type: strategy.ParamContinuous,
			Default: 0.25, Min: 0.10, Max: 0.25, Step: 0.05,
			Group: "Sizing", Description: "Fractional Kelly multiplier applied to base size (capped at 0.25 per HP #6)",
		},
	}
}

// addUniversalParams injects the universal sizing parameters into a search space
// (unless the strategy already defines a parameter of the same name).
func addUniversalParams(space SearchSpace) SearchSpace {
	if space == nil {
		space = make(SearchSpace)
	}
	for _, d := range UniversalParamDefs() {
		if _, exists := space[d.Name]; exists {
			continue
		}
		space[d.Name] = ParamConstraint{
			Name: d.Name, Type: ParamContinuous,
			Min: d.Min, Max: d.Max, Step: d.Step,
		}
	}
	return space
}

// ApplyOptimizationParams applies a parameter combination to a BacktestConfig.
// Universal parameters (e.g. sizing_percent, kelly_fraction) are mapped onto the
// corresponding BacktestConfig fields; all remaining parameters are passed
// through to the strategy runner via StrategyParams. This lets a single search
// space uniformly optimize both account-level sizing and strategy-specific
// technical parameters for any strategy type.
func ApplyOptimizationParams(cfg *BacktestConfig, params map[string]float64) {
	if cfg == nil || params == nil {
		return
	}
	stratParams := make(map[string]float64, len(params))
	for name, val := range params {
		switch name {
		case "sizing_percent":
			cfg.SizingPercent = val
		case "kelly_fraction":
			cfg.KellyFraction = val
		default:
			stratParams[name] = val
		}
	}
	cfg.StrategyParams = stratParams
}

func defaultStrategySearchSpace(strategyID string) SearchSpace {
	switch strategyID {
	case "trend_following":
		space := SearchSpaceFromParamDefs(ParamDefsForStrategy("trend_following"))
		space["fast_period"] = ParamConstraint{Name: "fast_period", Type: ParamInteger, Min: 5, Max: 40, Step: 5,
			Condition: &ConditionalRule{LeftParam: "fast_period", Operator: "lt", RightParam: "slow_period"}}
		return space
	case "intraday_mr", "mean_reversion":
		space := SearchSpaceFromParamDefs(ParamDefsForStrategy(strategyID))
		if space == nil {
			return nil
		}
		space["entry_z"] = ParamConstraint{Name: "entry_z", Type: ParamContinuous, Min: 0.75, Max: 2.5, Step: 0.25,
			Condition: &ConditionalRule{LeftParam: "exit_z", Operator: "lt", RightParam: "entry_z"}}
		return space
	case "opening_range_breakout":
		space := SearchSpaceFromParamDefs(ParamDefsForStrategy(strategyID))
		if space == nil {
			return nil
		}
		space["atr_multiplier"] = ParamConstraint{Name: "atr_multiplier", Type: ParamContinuous, Min: 1.0, Max: 3.0, Step: 0.5}
		space["range_minutes"] = ParamConstraint{Name: "range_minutes", Type: ParamInteger, Min: 2, Max: 15, Step: 1}
		return space
	case "session_scalp":
		space := SearchSpaceFromParamDefs(ParamDefsForStrategy("session_scalp"))
		space["stop_loss_atr_mult"] = ParamConstraint{Name: "stop_loss_atr_mult", Type: ParamContinuous, Min: 0.25, Max: 2.0, Step: 0.25,
			Condition: &ConditionalRule{LeftParam: "stop_loss_atr_mult", Operator: "lt", RightParam: "take_profit_atr_mult"}}
		return space
	case "pairs_trading", "stat_arb":
		space := SearchSpaceFromParamDefs(ParamDefsForStrategy("pairs_trading"))
		if space == nil {
			return nil
		}
		space["entry_z"] = ParamConstraint{Name: "entry_z", Type: ParamContinuous, Min: 1.5, Max: 2.5, Step: 0.25,
			Condition: &ConditionalRule{LeftParam: "exit_z", Operator: "lt", RightParam: "entry_z"}}
		return space
	case "volatility_harvesting", "vol_arb":
		space := SearchSpaceFromParamDefs(ParamDefsForStrategy("volatility_harvesting"))
		if space == nil {
			return nil
		}
		space["vix_threshold"] = ParamConstraint{Name: "vix_threshold", Type: ParamContinuous, Min: 15, Max: 35, Step: 2.0}
		space["mean_rev_entry_z"] = ParamConstraint{Name: "mean_rev_entry_z", Type: ParamContinuous, Min: 1.0, Max: 2.5, Step: 0.25,
			Condition: &ConditionalRule{LeftParam: "mean_rev_exit_z", Operator: "lt", RightParam: "mean_rev_entry_z"}}
		return space
	case "dragon_trend":
		space := SearchSpaceFromParamDefs(ParamDefsForStrategy("dragon_trend"))
		if space == nil {
			return nil
		}
		space["adx_threshold"] = ParamConstraint{Name: "adx_threshold", Type: ParamContinuous, Min: 15, Max: 30, Step: 2.0}
		space["atr_multiplier"] = ParamConstraint{Name: "atr_multiplier", Type: ParamContinuous, Min: 1.5, Max: 3.5, Step: 0.5}
		return space
	case "volume_scalp":
		space := SearchSpaceFromParamDefs(ParamDefsForStrategy("volume_scalp"))
		if space == nil {
			return nil
		}
		space["volume_multiplier"] = ParamConstraint{Name: "volume_multiplier", Type: ParamContinuous, Min: 1.0, Max: 3.5, Step: 0.5}
		return space
	default:
		runner := strategy.GlobalRegistry().Get(strategyID)
		if runner != nil {
			return SearchSpaceFromParamDefs(runner.ParamDefs())
		}
		return nil
	}
}

func SearchSpaceFromParamDefs(defs []strategy.ParamDef) SearchSpace {
	space := make(SearchSpace, len(defs))
	for _, d := range defs {
		pt := ParamContinuous
		switch d.Type {
		case strategy.ParamInteger:
			pt = ParamInteger
		case strategy.ParamCategorical:
			pt = ParamCategorical
		}
		space[d.Name] = ParamConstraint{
			Name: d.Name,
			Type: pt,
			Min:  d.Min,
			Max:  d.Max,
			Step: d.Step,
		}
	}
	return space
}
