package backtest

import (
	"encoding/json"
	"fmt"
	"time"
)

type MultiMetricStandard struct {
	MinSharpeRatio   float64 `json:"min_sharpe_ratio"`
	MaxDrawdownPct   float64 `json:"max_drawdown_pct"`
	MinPassProbPct   float64 `json:"min_pass_probability_pct"`
	MinProfitFactor  float64 `json:"min_profit_factor"`
	MinNumTrades     int     `json:"min_num_trades"`
	MinSortinoRatio  float64 `json:"min_sortino_ratio"`
	MaxSharpeDegradation float64 `json:"max_sharpe_degradation_pct"`
}

type MetricVerdict struct {
	Metric   string  `json:"metric"`
	Required float64 `json:"required"`
	Actual   float64 `json:"actual"`
	Passed   bool    `json:"passed"`
}

type MultiMetricVerdict struct {
	Passed     bool            `json:"passed"`
	Verdicts   []MetricVerdict `json:"verdicts"`
	PassedCount int            `json:"passed_count"`
	TotalCount  int            `json:"total_count"`
	Timestamp   time.Time      `json:"timestamp"`
}

func DefaultMultiMetricStandard() MultiMetricStandard {
	return MultiMetricStandard{
		MinSharpeRatio:       1.0,
		MaxDrawdownPct:       8.0,
		MinPassProbPct:       80.0,
		MinProfitFactor:      1.5,
		MinNumTrades:         30,
		MinSortinoRatio:      0.0,
		MaxSharpeDegradation: 50.0,
	}
}

func LenientMultiMetricStandard() MultiMetricStandard {
	return MultiMetricStandard{
		MinSharpeRatio:       0.5,
		MaxDrawdownPct:       15.0,
		MinPassProbPct:       60.0,
		MinProfitFactor:      1.2,
		MinNumTrades:         15,
		MinSortinoRatio:      0.0,
		MaxSharpeDegradation: 70.0,
	}
}

func StrictMultiMetricStandard() MultiMetricStandard {
	return MultiMetricStandard{
		MinSharpeRatio:       1.5,
		MaxDrawdownPct:       5.0,
		MinPassProbPct:       90.0,
		MinProfitFactor:      2.0,
		MinNumTrades:         50,
		MinSortinoRatio:      1.2,
		MaxSharpeDegradation: 30.0,
	}
}

func EvaluateOOSMultiMetric(owr *OptimizedWalkForwardResult, mcResult *MonteCarloResult, standard MultiMetricStandard) MultiMetricVerdict {
	verdict := MultiMetricVerdict{
		Timestamp: time.Now(),
	}

	if owr == nil || len(owr.Windows) == 0 {
		verdict.Verdicts = []MetricVerdict{
			{Metric: "walk_forward_data", Required: 1, Actual: 0, Passed: false},
		}
		return verdict
	}

	var totalOOSSharpe float64
	var totalOOSTrades int
	var worstDrawdown float64 = 0
	var totalOOSWinRate float64

	for _, w := range owr.Windows {
		totalOOSSharpe += w.OutSampleSharpe
		totalOOSTrades += w.OOSTrades
		if w.OOSReturnPct < worstDrawdown {
			worstDrawdown = w.OOSReturnPct
		}
		totalOOSWinRate += w.OOSWinRate
	}

	avgOOSSharpe := 0.0
	avgWinRate := 0.0
	if len(owr.Windows) > 0 {
		avgOOSSharpe = totalOOSSharpe / float64(len(owr.Windows))
		avgWinRate = totalOOSWinRate / float64(len(owr.Windows))
	}

	maxDD := -worstDrawdown
	if worstDrawdown >= 0 {
		maxDD = 0
	}

	passProb := 0.0
	if mcResult != nil {
		passProb = mcResult.PassProbability
	}

	profitFactor := owr.ProfitFactor

	verdicts := []MetricVerdict{
		{
			Metric:   "oos_sharpe_ratio",
			Required: standard.MinSharpeRatio,
			Actual:   avgOOSSharpe,
			Passed:   avgOOSSharpe >= standard.MinSharpeRatio,
		},
		{
			Metric:   "max_drawdown_pct",
			Required: standard.MaxDrawdownPct,
			Actual:   maxDD,
			Passed:   maxDD <= standard.MaxDrawdownPct,
		},
		{
			Metric:   "pass_probability_pct",
			Required: standard.MinPassProbPct,
			Actual:   passProb,
			Passed:   passProb >= standard.MinPassProbPct,
		},
		{
			Metric:   "profit_factor",
			Required: standard.MinProfitFactor,
			Actual:   profitFactor,
			Passed:   profitFactor >= standard.MinProfitFactor,
		},
		{
			Metric:   "min_trades",
			Required: float64(standard.MinNumTrades),
			Actual:   float64(totalOOSTrades),
			Passed:   totalOOSTrades >= standard.MinNumTrades,
		},
	}

	if standard.MinSortinoRatio > 0 {
		var totalSortino float64
		for _, w := range owr.Windows {
			totalSortino += w.InSampleSharpe
		}
		avgSortino := 0.0
		if len(owr.Windows) > 0 {
			avgSortino = totalSortino / float64(len(owr.Windows))
		}
		verdicts = append(verdicts, MetricVerdict{
			Metric:   "sortino_ratio",
			Required: standard.MinSortinoRatio,
			Actual:   avgSortino,
			Passed:   avgSortino >= standard.MinSortinoRatio,
		})
	}

	if standard.MaxSharpeDegradation > 0 && owr.SharpeDegradation > 0 {
		verdicts = append(verdicts, MetricVerdict{
			Metric:   "sharpe_degradation_pct",
			Required: standard.MaxSharpeDegradation,
			Actual:   owr.SharpeDegradation,
			Passed:   owr.SharpeDegradation <= standard.MaxSharpeDegradation,
		})
	}

	_ = avgWinRate

	verdict.Verdicts = verdicts
	verdict.TotalCount = len(verdicts)
	passedCount := 0
	allPassed := true
	for _, v := range verdicts {
		if v.Passed {
			passedCount++
		} else {
			allPassed = false
		}
	}
	verdict.PassedCount = passedCount
	verdict.Passed = allPassed

	return verdict
}

func EvaluateBacktestMultiMetric(result *BacktestResult, standard MultiMetricStandard) MultiMetricVerdict {
	verdict := MultiMetricVerdict{
		Timestamp: time.Now(),
	}

	if result == nil {
		verdict.Verdicts = []MetricVerdict{
			{Metric: "backtest_result", Required: 1, Actual: 0, Passed: false},
		}
		return verdict
	}

	verdicts := []MetricVerdict{
		{
			Metric:   "sharpe_ratio",
			Required: standard.MinSharpeRatio,
			Actual:   result.SharpeRatio,
			Passed:   result.SharpeRatio >= standard.MinSharpeRatio,
		},
		{
			Metric:   "max_drawdown_pct",
			Required: standard.MaxDrawdownPct,
			Actual:   result.MaxDrawdown,
			Passed:   result.MaxDrawdown <= standard.MaxDrawdownPct,
		},
		{
			Metric:   "profit_factor",
			Required: standard.MinProfitFactor,
			Actual:   result.ProfitFactor,
			Passed:   result.ProfitFactor >= standard.MinProfitFactor,
		},
		{
			Metric:   "min_trades",
			Required: float64(standard.MinNumTrades),
			Actual:   float64(result.NumTrades),
			Passed:   result.NumTrades >= standard.MinNumTrades,
		},
		{
			Metric:   "win_rate",
			Required: 0,
			Actual:   result.WinRate,
			Passed:   true,
		},
	}

	if standard.MinSortinoRatio > 0 {
		verdicts = append(verdicts, MetricVerdict{
			Metric:   "sortino_ratio",
			Required: standard.MinSortinoRatio,
			Actual:   result.SortinoRatio,
			Passed:   result.SortinoRatio >= standard.MinSortinoRatio,
		})
	}

	verdict.Verdicts = verdicts
	verdict.TotalCount = len(verdicts)
	passedCount := 0
	allPassed := true
	for _, v := range verdicts {
		if v.Passed {
			passedCount++
		} else {
			allPassed = false
		}
	}
	verdict.PassedCount = passedCount
	verdict.Passed = allPassed

	return verdict
}

func (v MultiMetricVerdict) JSON() string {
	data, _ := json.MarshalIndent(v, "", "  ")
	return string(data)
}

func (v MultiMetricVerdict) Summary() string {
	if v.Passed {
		return fmt.Sprintf("PASS (%d/%d metrics)", v.PassedCount, v.TotalCount)
	}
	failures := []string{}
	for _, m := range v.Verdicts {
		if !m.Passed {
			failures = append(failures, fmt.Sprintf("%s (%.2f vs req %.2f)", m.Metric, m.Actual, m.Required))
		}
	}
	summary := ""
	for _, f := range failures {
		summary += "  " + f + "\n"
	}
	return fmt.Sprintf("FAIL (%d/%d metrics):\n%s", v.PassedCount, v.TotalCount, summary)
}
