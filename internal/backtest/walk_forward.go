package backtest

import (
	"context"
	"time"
)

type WalkForwardConfig struct {
	Config             BacktestConfig
	TrainWindows       int
	TrainYears         int
	TestYears          int
	StepMonths         int
	PurgeTradingDays   int
	EmbargoTradingDays int
}

type WalkForwardWindow struct {
	TrainStart time.Time
	TrainEnd   time.Time
	TestStart  time.Time
	TestEnd    time.Time
}

type WalkForwardResult struct {
	Windows           []WindowResult `json:"windows"`
	OverallSharpe     float64        `json:"overall_sharpe"`
	OverallWinRate    float64        `json:"overall_win_rate"`
	ProfitFactor      float64        `json:"profit_factor"`
	TotalReturnPct    float64        `json:"total_return_pct"`
	PassedWindows     int            `json:"passed_windows"`
	TotalWindows      int            `json:"total_windows"`
	AvgOOSSharpe      float64        `json:"avg_oos_sharpe"`
	SharpeDegradation float64        `json:"sharpe_degradation"`
	// AvgAnchorOOSSharpe is the mean fixed-parameter (default params) OOS Sharpe
	// across windows — the overfit-detection baseline.
	AvgAnchorOOSSharpe float64 `json:"avg_anchor_oos_sharpe"`
}

type WindowResult struct {
	Window              int       `json:"window"`
	TrainStart          time.Time `json:"train_start"`
	TestStart           time.Time `json:"test_start"`
	TestEnd             time.Time `json:"test_end"`
	InSampleSharpe      float64   `json:"in_sample_sharpe"`
	OutSampleSharpe     float64   `json:"out_sample_sharpe"`
	OutSampleSortino    float64   `json:"out_sample_sortino"`
	OOSWinRate          float64   `json:"oos_win_rate"`
	OOSReturnPct        float64   `json:"oos_return_pct"`
	OOSMaxDD            float64   `json:"oos_max_dd"`
	OOSProfitFactor     float64   `json:"oos_profit_factor"`
	OOSTrades           int       `json:"oos_trades"`
	PassedCompliance    bool      `json:"passed_compliance"`
	MultiplicityWarning bool      `json:"multiplicity_warning"`
	// AnchorOOSSharpe is the fixed-parameter (default params) out-of-sample
	// Sharpe for the same window. Comparing optimized OOS against this baseline
	// detects overfitting: if the optimized OOS does not beat the anchor, the
	// parameter island is noise.
	AnchorOOSSharpe        float64 `json:"anchor_oos_sharpe"`
	AnchorPassedCompliance bool    `json:"anchor_passed_compliance"`
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
			Window:           i + 1,
			TrainStart:       w.TrainStart,
			TestStart:        w.TestStart,
			TestEnd:          w.TestEnd,
			InSampleSharpe:   trainResult.SharpeRatio,
			OutSampleSharpe:  testResult.SharpeRatio,
			OutSampleSortino: testResult.SortinoRatio,
			OOSWinRate:       testResult.WinRate,
			OOSReturnPct:     testResult.TotalReturnPct,
			OOSMaxDD:         testResult.MaxDrawdown,
			OOSTrades:        testResult.NumTrades,
			PassedCompliance: testResult.ComplianceReport != nil && testResult.ComplianceReport.Passed,
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
