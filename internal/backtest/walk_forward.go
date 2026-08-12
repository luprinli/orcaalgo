package backtest

import (
	"context"
	"time"
)

type WalkForwardConfig struct {
	Config       BacktestConfig
	TrainWindows int
	TrainYears   int
	TestYears    int
	StepMonths   int
	PurgeTradingDays  int
	EmbargoTradingDays int
}

type WalkForwardWindow struct {
	TrainStart time.Time
	TrainEnd   time.Time
	TestStart  time.Time
	TestEnd    time.Time
}

type WalkForwardResult struct {
	Windows         []WindowResult
	OverallSharpe   float64
	OverallWinRate  float64
	ProfitFactor    float64
	TotalReturnPct  float64
	PassedWindows   int
	TotalWindows    int
	AvgOOSSharpe    float64
	SharpeDegradation float64
}

type WindowResult struct {
	Window       int
	TrainStart   time.Time
	TestStart    time.Time
	TestEnd      time.Time
	InSampleSharpe  float64
	OutSampleSharpe float64
	OutSampleSortino float64
	OOSWinRate   float64
	OOSReturnPct float64
	OOSMaxDD     float64
	OOSProfitFactor float64
	OOSTrades            int
	PassedCompliance     bool
	MultiplicityWarning  bool
}

func GenerateWalkForwardWindows(config WalkForwardConfig) []WalkForwardWindow {
	var windows []WalkForwardWindow
	start := config.Config.StartDate
	end := config.Config.EndDate

	trainDuration := time.Duration(config.TrainYears) * 252 * 24 * time.Hour
	stepDuration := time.Duration(config.StepMonths) * 21 * 24 * time.Hour

	currentStart := start
	for currentStart.Add(trainDuration).Before(end) {
		testStart := currentStart.Add(trainDuration)
		testEnd := testStart.Add(time.Duration(config.TestYears) * 252 * 24 * time.Hour)
		if testEnd.After(end) {
			testEnd = end
		}

		windows = append(windows, WalkForwardWindow{
			TrainStart: currentStart,
			TrainEnd:   testStart.Add(-time.Duration(1+config.PurgeTradingDays+config.EmbargoTradingDays) * 24 * time.Hour),
			TestStart:  testStart,
			TestEnd:    testEnd,
		})

		currentStart = currentStart.Add(stepDuration)
		if len(windows) >= config.TrainWindows && config.TrainWindows > 0 {
			break
		}
	}
	return windows
}

func (e *Engine) RunWalkForward(ctx context.Context, config WalkForwardConfig) (*WalkForwardResult, error) {
	windows := GenerateWalkForwardWindows(config)
	result := &WalkForwardResult{
		Windows:      make([]WindowResult, 0, len(windows)),
		TotalWindows: len(windows),
	}

	var totalSharpe, totalOOSSharpe float64
	var totalWins, totalTrades int
	var totalReturn float64

	for i, w := range windows {
		trainCfg := config.Config
		trainCfg.StartDate = w.TrainStart
		trainCfg.EndDate = w.TrainEnd

		testCfg := config.Config
		testCfg.StartDate = w.TestStart
		testCfg.EndDate = w.TestEnd

		trainResult, err := e.Run(ctx, trainCfg)
		if err != nil || trainResult == nil {
			continue
		}

		if len(trainResult.StrategyParams) > 0 {
			testCfg.StrategyParams = trainResult.StrategyParams
		}
		testResult, err := e.Run(ctx, testCfg)
		if err != nil {
			continue
		}

		wr := WindowResult{
			Window:         i + 1,
			TrainStart:     w.TrainStart,
			TestStart:      w.TestStart,
			TestEnd:        w.TestEnd,
			InSampleSharpe: trainResult.SharpeRatio,
			OutSampleSharpe: testResult.SharpeRatio,
			OutSampleSortino: testResult.SortinoRatio,
			OOSWinRate:     testResult.WinRate,
			OOSReturnPct:   testResult.TotalReturnPct,
			OOSMaxDD:       testResult.MaxDrawdown,
			OOSTrades:      testResult.NumTrades,
			PassedCompliance:     testResult.ComplianceReport != nil && testResult.ComplianceReport.Passed,
		}

		if wr.PassedCompliance {
			result.PassedWindows++
		}
		result.Windows = append(result.Windows, wr)
		totalSharpe += trainResult.SharpeRatio
		totalOOSSharpe += testResult.SharpeRatio
		totalWins += testResult.NumWins
		totalTrades += testResult.NumTrades
		totalReturn += testResult.TotalReturnPct
	}

	completed := len(result.Windows)
	if completed > 0 {
		result.AvgOOSSharpe = totalOOSSharpe / float64(completed)
		result.OverallSharpe = totalSharpe / float64(completed)
		result.TotalReturnPct = totalReturn / float64(completed)
		if totalTrades > 0 {
			result.OverallWinRate = float64(totalWins) / float64(totalTrades) * 100
		}
		if totalSharpe > 0 {
			result.SharpeDegradation = (totalSharpe - totalOOSSharpe) / totalSharpe * 100
		}
	}

	return result, nil
}
