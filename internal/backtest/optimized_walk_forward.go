package backtest

import (
	"context"
	"math"
	"sort"
	"time"
)

type OptimizedWalkForwardConfig struct {
	WalkForwardConfig
	OptimizationConfig
	IVSConfig IVSConfig
}

type OptimizedWalkForwardResult struct {
	WalkForwardResult
	BestParamsPerWindow   []map[string]float64
	IVSRobustParamsPerWindow []map[string]float64
	IVSActive             bool
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
		BestParamsPerWindow:   make([]map[string]float64, len(windows)),
		IVSRobustParamsPerWindow: make([]map[string]float64, len(windows)),
		IVSActive:             ivsCfg.Enabled,
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

		var allScores []ParamScore

		for _, params := range combinations {
			trainCfg := config.Config
			trainCfg.StartDate = w.TrainStart
			trainCfg.EndDate = w.TrainEnd
			trainCfg.StrategyParams = params

			trainResult, err := e.Run(ctx, trainCfg)
			if err != nil || trainResult == nil {
				continue
			}

			score := ComputeObjective(trainResult, config.ObjectiveType, config.ObjectiveWeights)
			allScores = append(allScores, ParamScore{Params: params, Score: score})
		}

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
		testCfg.StrategyParams = selectedParams

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
				evalCfg.StrategyParams = params
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
			Window:          wi + 1,
			TrainStart:      w.TrainStart,
			TestStart:       w.TestStart,
			TestEnd:         w.TestEnd,
			InSampleSharpe:  allScores[0].Score,
			OutSampleSharpe: bestOOSTestResult.SharpeRatio,
			OOSWinRate:      bestOOSTestResult.WinRate,
			OOSReturnPct:    bestOOSTestResult.TotalReturnPct,
			OOSMaxDD:        bestOOSTestResult.MaxDrawdown,
			OOSTrades:       bestOOSTestResult.NumTrades,
			PassedCompliance:      bestOOSTestResult.ComplianceReport != nil && bestOOSTestResult.ComplianceReport.Passed,
			MultiplicityWarning:   allScores[0].Score <= scoreThreshold,
		}

		result.Windows = append(result.Windows, wr)
	}

	result.PassedWindows = 0
	for _, w := range result.Windows {
		if w.PassedCompliance {
			result.PassedWindows++
		}
	}

	var totalIS, totalOOS float64
	for _, w := range result.Windows {
		totalIS += w.InSampleSharpe
		totalOOS += w.OutSampleSharpe
	}
	if len(result.Windows) > 0 {
		result.OverallSharpe = totalIS / float64(len(result.Windows))
		result.AvgOOSSharpe = totalOOS / float64(len(result.Windows))
		if result.OverallSharpe > 0 {
			result.SharpeDegradation = (result.OverallSharpe - result.AvgOOSSharpe) / result.OverallSharpe * 100
		}
	}

	return result, nil
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

func DefaultOptimizedWalkForwardConfig(strategyID string, symbols []string, startDate, endDate time.Time, initialCapital float64) OptimizedWalkForwardConfig {
	return OptimizedWalkForwardConfig{
		WalkForwardConfig: WalkForwardConfig{
			Config: BacktestConfig{
				StrategyID:     strategyID,
				Symbols:        symbols,
				StartDate:      startDate,
				EndDate:        endDate,
				InitialCapital: initialCapital,
			},
			TrainWindows: 5,
			TrainYears:   1,
			TestYears:    1,
			StepMonths:   3,
		},
		OptimizationConfig: OptimizationConfig{
			StrategyID:      strategyID,
			SearchSpace:     DefaultSearchSpace(strategyID),
			ObjectiveType:   ObjectiveComposite,
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
