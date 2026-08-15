package backtest

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"sync"
	"time"
)

type OptimizedWalkForwardConfig struct {
	WalkForwardConfig
	OptimizationConfig
	IVSConfig IVSConfig
}

type OptimizedWalkForwardResult struct {
	WalkForwardResult
	BestParamsPerWindow      []map[string]float64
	IVSRobustParamsPerWindow []map[string]float64
	IVSActive                bool
}

type IVSConfig struct {
	Enabled           bool
	NeighborRadius    float64
	PlateauThreshold  float64
	ScoreThresholdPct float64
}

func DefaultIVSConfig() IVSConfig {
	return IVSConfig{
		Enabled:           true,
		NeighborRadius:    0.15,
		PlateauThreshold:  0.50,
		ScoreThresholdPct: 0.80,
	}
}

func (e *Engine) RunOptimizedWalkForward(ctx context.Context, config OptimizedWalkForwardConfig) (*OptimizedWalkForwardResult, error) {
	windows := GenerateWalkForwardWindows(config.WalkForwardConfig)
	if len(windows) == 0 {
		return &OptimizedWalkForwardResult{}, nil
	}

	ivsCfg := config.IVSConfig
	if !ivsCfg.Enabled {
		ivsCfg = IVSConfig{Enabled: false}
	}

	result := &OptimizedWalkForwardResult{
		WalkForwardResult: WalkForwardResult{
			TotalWindows: len(windows),
		},
		BestParamsPerWindow:      make([]map[string]float64, len(windows)),
		IVSRobustParamsPerWindow: make([]map[string]float64, len(windows)),
		IVSActive:                ivsCfg.Enabled,
	}

	for wi, w := range windows {
		searchSpace := config.OptimizationConfig.SearchSpace
		if searchSpace == nil {
			searchSpace = DefaultSearchSpace(config.StrategyID)
		}

		combinations := searchSpace.GenerateAllCombinations()
		if config.MaxCombinations > 0 && len(combinations) > config.MaxCombinations {
			combinations = combinations[:config.MaxCombinations]
		}

		// Anchor baseline: default params on the OOS window. Optimized OOS that
		// does not beat this fixed-parameter baseline is overfit noise.
		anchorCfg := config.Config
		anchorCfg.StartDate = w.TestStart
		anchorCfg.EndDate = w.TestEnd
		anchorOOSSharpe := -1e9
		var anchorPassed bool
		if anchorResult, aerr := e.Run(ctx, anchorCfg); aerr == nil && anchorResult != nil {
			anchorOOSSharpe = anchorResult.SharpeRatio
			anchorPassed = anchorResult.ComplianceReport != nil && anchorResult.ComplianceReport.Passed
		}

		allScores := e.evaluateISCombinations(ctx, config, w, combinations)

		if len(allScores) == 0 {
			continue
		}

		sort.Slice(allScores, func(i, j int) bool {
			return allScores[i].Score > allScores[j].Score
		})

		scoresOnly := make([]float64, len(allScores))
		for k, s := range allScores {
			scoresOnly[k] = s.Score
		}
		sort.Float64s(scoresOnly)
		median := scoresOnly[len(scoresOnly)/2]
		var sumSq float64
		for _, v := range scoresOnly {
			sumSq += (v - median) * (v - median)
		}
		std := math.Sqrt(sumSq / float64(len(scoresOnly)))
		scoreThreshold := median + 2.0*std

		bestParams := allScores[0].Params
		result.BestParamsPerWindow[wi] = bestParams

		var selectedParams map[string]float64
		if ivsCfg.Enabled {
			robustParams, ivsPassed := RunIVS(allScores, searchSpace, ivsCfg)
			if ivsPassed && robustParams != nil {
				selectedParams = robustParams
				result.IVSRobustParamsPerWindow[wi] = robustParams
			} else {
				selectedParams = bestParams
				result.IVSRobustParamsPerWindow[wi] = nil
			}
		} else {
			selectedParams = bestParams
		}

		testCfg := config.Config
		testCfg.StartDate = w.TestStart
		testCfg.EndDate = w.TestEnd
		ApplyOptimizationParams(&testCfg, selectedParams)

		var bestOOSScore float64 = -1e9
		var bestOOSTestResult *BacktestResult

		evaluateCandidates := func(paramsList []map[string]float64) {
			for _, params := range paramsList {
				if params == nil {
					continue
				}
				evalCfg := testCfg
				evalCfg.StartDate = w.TestStart
				evalCfg.EndDate = w.TestEnd
				ApplyOptimizationParams(&evalCfg, params)
				testResult, err := e.Run(ctx, evalCfg)
				if err != nil || testResult == nil {
					continue
				}
				score := ComputeObjective(testResult, config.ObjectiveType, config.ObjectiveWeights)
				if score > bestOOSScore {
					bestOOSScore = score
					bestOOSTestResult = testResult
				}
			}
		}

		evaluateCandidates([]map[string]float64{bestParams, result.IVSRobustParamsPerWindow[wi]})

		if bestOOSTestResult == nil {
			testCfg := config.Config
			testCfg.StartDate = w.TestStart
			testCfg.EndDate = w.TestEnd
			bestOOSTestResult, _ = e.Run(ctx, testCfg)
		}
		if bestOOSTestResult == nil {
			continue
		}

		wr := WindowResult{
			Window:                 wi + 1,
			TrainStart:             w.TrainStart,
			TestStart:              w.TestStart,
			TestEnd:                w.TestEnd,
			InSampleSharpe:         allScores[0].Score,
			OutSampleSharpe:        bestOOSTestResult.SharpeRatio,
			OOSWinRate:             bestOOSTestResult.WinRate,
			OOSReturnPct:           bestOOSTestResult.TotalReturnPct,
			OOSMaxDD:               bestOOSTestResult.MaxDrawdown,
			OOSTrades:              bestOOSTestResult.NumTrades,
			PassedCompliance:       bestOOSTestResult.ComplianceReport != nil && bestOOSTestResult.ComplianceReport.Passed,
			MultiplicityWarning:    allScores[0].Score <= scoreThreshold,
			AnchorOOSSharpe:        anchorOOSSharpe,
			AnchorPassedCompliance: anchorPassed,
		}

		result.Windows = append(result.Windows, wr)
	}

	result.PassedWindows = 0
	for _, w := range result.Windows {
		if w.PassedCompliance {
			result.PassedWindows++
		}
	}

	var totalIS, totalOOS, totalAnchor float64
	for _, w := range result.Windows {
		totalIS += w.InSampleSharpe
		totalOOS += w.OutSampleSharpe
		if w.AnchorOOSSharpe > -1e8 {
			totalAnchor += w.AnchorOOSSharpe
		}
	}
	if len(result.Windows) > 0 {
		result.OverallSharpe = totalIS / float64(len(result.Windows))
		result.AvgOOSSharpe = totalOOS / float64(len(result.Windows))
		if result.OverallSharpe > 0 {
			result.SharpeDegradation = (result.OverallSharpe - result.AvgOOSSharpe) / result.OverallSharpe * 100
		}
		result.AvgAnchorOOSSharpe = totalAnchor / float64(len(result.Windows))
	}

	return result, nil
}

// evaluateISCombinations runs the in-sample (train-window) evaluation for every
// candidate parameter set, in parallel when a database is available (a fresh
// engine per combination), falling back to a serial run on the receiver engine
// otherwise. This is the expensive inner loop of the embedded walk-forward.
func (e *Engine) evaluateISCombinations(ctx context.Context, config OptimizedWalkForwardConfig, w WalkForwardWindow, combinations []map[string]float64) []ParamScore {
	if e.db == nil || len(combinations) < 2 {
		var scores []ParamScore
		for _, params := range combinations {
			trainCfg := config.Config
			trainCfg.StartDate = w.TrainStart
			trainCfg.EndDate = w.TrainEnd
			ApplyOptimizationParams(&trainCfg, params)
			trainResult, err := e.Run(ctx, trainCfg)
			if err != nil || trainResult == nil {
				continue
			}
			score := ComputeObjective(trainResult, config.ObjectiveType, config.ObjectiveWeights)
			scores = append(scores, ParamScore{Params: params, Score: score})
		}
		return scores
	}

	allScores := make([]ParamScore, len(combinations))
	var mu sync.Mutex
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for idx, params := range combinations {
		wg.Add(1)
		go func(idx int, params map[string]float64) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()

			trainCfg := config.Config
			trainCfg.StartDate = w.TrainStart
			trainCfg.EndDate = w.TrainEnd
			ApplyOptimizationParams(&trainCfg, params)

			engine := NewEngine(e.db)
			trainResult, err := engine.Run(ctx, trainCfg)
			if err != nil || trainResult == nil {
				return
			}
			score := ComputeObjective(trainResult, config.ObjectiveType, config.ObjectiveWeights)

			mu.Lock()
			allScores[idx] = ParamScore{Params: params, Score: score}
			mu.Unlock()
		}(idx, params)
	}
	wg.Wait()

	out := allScores[:0]
	for _, s := range allScores {
		if s.Params != nil {
			out = append(out, s)
		}
	}
	return out
}

// hasMultiplicityWarning reports whether any window flagged a multiplicity
// warning (best IS score within 2 std of the median — the "best" params are
// not clearly better than chance after multiple-testing correction).
func (r *OptimizedWalkForwardResult) hasMultiplicityWarning() bool {
	for _, w := range r.Windows {
		if w.MultiplicityWarning {
			return true
		}
	}
	return false
}

// encodeWfBestParams serializes the IVS-robust parameter island (falling back
// to the raw best params) from the window with the highest OOS Sharpe, for
// persistence and promotion. It is the single source of truth for "which params
// survived walk-forward".
func encodeWfBestParams(r *OptimizedWalkForwardResult) string {
	if r == nil || len(r.Windows) == 0 {
		return ""
	}
	bestIdx := 0
	for i := 1; i < len(r.Windows); i++ {
		if r.Windows[i].OutSampleSharpe > r.Windows[bestIdx].OutSampleSharpe {
			bestIdx = i
		}
	}
	var params map[string]float64
	if bestIdx < len(r.IVSRobustParamsPerWindow) && r.IVSRobustParamsPerWindow[bestIdx] != nil {
		params = r.IVSRobustParamsPerWindow[bestIdx]
	} else if bestIdx < len(r.BestParamsPerWindow) {
		params = r.BestParamsPerWindow[bestIdx]
	}
	if params == nil {
		return ""
	}
	b, err := json.Marshal(params)
	if err != nil {
		return ""
	}
	return string(b)
}

func RunIVS(scoredParams []ParamScore, searchSpace SearchSpace, cfg IVSConfig) (map[string]float64, bool) {
	if len(scoredParams) == 0 {
		return nil, false
	}

	paramNames := sortedParamNames(searchSpace)
	if len(paramNames) == 0 {
		return scoredParams[0].Params, false
	}

	maxScore := scoredParams[0].Score
	if maxScore <= 0 {
		bestScore := scoredParams[0].Score
		for _, sp := range scoredParams {
			if sp.Score > bestScore {
				bestScore = sp.Score
			}
		}
		maxScore = bestScore
	}
	if maxScore <= 0 {
		return scoredParams[0].Params, false
	}

	scoreThreshold := maxScore * cfg.ScoreThresholdPct

	n := len(scoredParams)
	distances := make([][]float64, n)
	for i := 0; i < n; i++ {
		distances[i] = make([]float64, n)
	}

	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			d := paramDistance(scoredParams[i].Params, scoredParams[j].Params, searchSpace, paramNames)
			distances[i][j] = d
			distances[j][i] = d
		}
	}

	type ivsCandidate struct {
		idx           int
		params        map[string]float64
		score         float64
		plateauRatio  float64
		neighborCount int
	}

	var candidates []ivsCandidate
	for i := 0; i < n; i++ {
		neighborCount := 0
		aboveThreshold := 0
		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			if distances[i][j] <= cfg.NeighborRadius {
				neighborCount++
				if scoredParams[j].Score >= scoreThreshold {
					aboveThreshold++
				}
			}
		}

		plateauRatio := 0.0
		if neighborCount > 0 {
			plateauRatio = float64(aboveThreshold) / float64(neighborCount)
		}

		candidates = append(candidates, ivsCandidate{
			idx:           i,
			params:        scoredParams[i].Params,
			score:         scoredParams[i].Score,
			plateauRatio:  plateauRatio,
			neighborCount: neighborCount,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].plateauRatio != candidates[j].plateauRatio {
			return candidates[i].plateauRatio > candidates[j].plateauRatio
		}
		return candidates[i].score > candidates[j].score
	})

	if candidates[0].plateauRatio < cfg.PlateauThreshold {
		bestByScore := candidates[0]
		for _, c := range candidates {
			if c.score > bestByScore.score {
				bestByScore = c
			}
		}
		return bestByScore.params, false
	}

	bestRobust := candidates[0]
	for i := 1; i < len(candidates); i++ {
		if candidates[i].plateauRatio >= cfg.PlateauThreshold && candidates[i].score > bestRobust.score {
			bestRobust = candidates[i]
		}
	}

	return bestRobust.params, true
}

func paramDistance(a, b map[string]float64, space SearchSpace, paramNames []string) float64 {
	sumSq := 0.0
	count := 0

	for _, name := range paramNames {
		va, okA := a[name]
		vb, okB := b[name]
		if !okA || !okB {
			continue
		}

		constraint, ok := space[name]
		if !ok {
			continue
		}

		span := constraint.Max - constraint.Min
		if span <= 0 {
			continue
		}

		normalized := (va - vb) / span
		sumSq += normalized * normalized
		count++
	}

	if count == 0 {
		return 0
	}

	return math.Sqrt(sumSq / float64(count))
}

func sortedParamNames(space SearchSpace) []string {
	names := make([]string, 0, len(space))
	for name := range space {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// NewOptimizedWalkForwardConfig builds an OptimizedWalkForwardConfig with the
// canonical optimization defaults (strategy search space, composite objective,
// IVS plateau detection) wired to the given base backtest config. It is the
// single source of truth for the embedded IS-optimization → OOS walk-forward
// used by the matrix, CLI, and job runner.
func NewOptimizedWalkForwardConfig(base BacktestConfig, strategyID string) OptimizedWalkForwardConfig {
	return OptimizedWalkForwardConfig{
		WalkForwardConfig: WalkForwardConfig{
			Config:             base,
			TrainWindows:       3,
			TrainYears:         1,
			TestYears:          1,
			StepMonths:         6,
			PurgeTradingDays:   5,
			EmbargoTradingDays: 2,
		},
		OptimizationConfig: OptimizationConfig{
			StrategyID:    strategyID,
			SearchSpace:   DefaultSearchSpace(strategyID),
			ObjectiveType: ObjectiveComposite,
			ObjectiveWeights: map[ObjectiveType]float64{
				ObjectiveSharpe:       0.5,
				ObjectiveMinDD:        0.3,
				ObjectiveProfitFactor: 0.2,
			},
			MaxCombinations: 200,
		},
		IVSConfig: DefaultIVSConfig(),
	}
}

func DefaultOptimizedWalkForwardConfig(strategyID string, symbols []string, startDate, endDate time.Time, initialCapital float64) OptimizedWalkForwardConfig {
	cfg := NewOptimizedWalkForwardConfig(BacktestConfig{
		StrategyID:     strategyID,
		Symbols:        symbols,
		StartDate:      startDate,
		EndDate:        endDate,
		InitialCapital: initialCapital,
	}, strategyID)
	// Job-runner defaults: longer anchor and more roll windows.
	cfg.WalkForwardConfig.TrainWindows = 5
	cfg.WalkForwardConfig.StepMonths = 3
	return cfg
}
