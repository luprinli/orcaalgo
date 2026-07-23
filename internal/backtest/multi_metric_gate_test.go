package backtest

import (
	"testing"
	"time"
)

func TestDefaultMultiMetricStandard(t *testing.T) {
	std := DefaultMultiMetricStandard()
	if std.MinSharpeRatio != 1.0 {
		t.Error("Expected min Sharpe 1.0")
	}
	if std.MaxDrawdownPct != 8.0 {
		t.Error("Expected max DD 8%")
	}
	if std.MinPassProbPct != 80.0 {
		t.Error("Expected min pass prob 80%")
	}
	if std.MinProfitFactor != 1.5 {
		t.Error("Expected min profit factor 1.5")
	}
	if std.MinNumTrades != 30 {
		t.Error("Expected min 30 trades")
	}
}

func TestLenientMultiMetricStandard(t *testing.T) {
	std := LenientMultiMetricStandard()
	if std.MinSharpeRatio > DefaultMultiMetricStandard().MinSharpeRatio {
		t.Error("Lenient standard should have lower min Sharpe than default")
	}
}

func TestStrictMultiMetricStandard(t *testing.T) {
	std := StrictMultiMetricStandard()
	if std.MinSharpeRatio < DefaultMultiMetricStandard().MinSharpeRatio {
		t.Error("Strict standard should have higher min Sharpe than default")
	}
}

func TestEvaluateBacktestMultiMetric_Pass(t *testing.T) {
	result := &BacktestResult{
		SharpeRatio:    1.8,
		SortinoRatio:   2.1,
		MaxDrawdown:    4.5,
		ProfitFactor:   2.3,
		NumTrades:      45,
		WinRate:        62.0,
	}

	std := DefaultMultiMetricStandard()
	verdict := EvaluateBacktestMultiMetric(result, std)

	if !verdict.Passed {
		t.Errorf("Expected PASS, got FAIL: %s", verdict.Summary())
	}
	if verdict.PassedCount < 5 {
		t.Errorf("Expected at least 5 passing metrics, got %d", verdict.PassedCount)
	}
}

func TestEvaluateBacktestMultiMetric_Fail(t *testing.T) {
	result := &BacktestResult{
		SharpeRatio:    0.3,
		MaxDrawdown:    18.0,
		ProfitFactor:   0.8,
		NumTrades:      10,
		WinRate:        35.0,
	}

	std := DefaultMultiMetricStandard()
	verdict := EvaluateBacktestMultiMetric(result, std)

	if verdict.Passed {
		t.Error("Expected FAIL for poor backtest metrics")
	}
	if verdict.PassedCount > 1 {
		t.Errorf("Expected at most 1 passing metric (win_rate is non-gating), got %d", verdict.PassedCount)
	}

	failMetrics := 0
	for _, v := range verdict.Verdicts {
		if !v.Passed {
			failMetrics++
		}
	}
	if failMetrics < 4 {
		t.Errorf("Expected at least 4 failing metrics, got %d", failMetrics)
	}
}

func TestEvaluateBacktestMultiMetric_Marginal(t *testing.T) {
	result := &BacktestResult{
		SharpeRatio:    1.0,
		MaxDrawdown:    8.0,
		ProfitFactor:   1.5,
		NumTrades:      30,
		WinRate:        50.0,
	}

	std := DefaultMultiMetricStandard()
	verdict := EvaluateBacktestMultiMetric(result, std)

	if !verdict.Passed {
		t.Errorf("Marginal case should pass, got: %s", verdict.Summary())
	}

	for _, v := range verdict.Verdicts {
		if !v.Passed && v.Metric == "sharpe_ratio" && result.SharpeRatio == 1.0 {
			t.Log("Sharpe exactly at threshold may use >= comparison")
		}
	}
}

func TestEvaluateOOSMultiMetric_Pass(t *testing.T) {
	owr := &OptimizedWalkForwardResult{
		WalkForwardResult: WalkForwardResult{
			Windows: []WindowResult{
				{OutSampleSharpe: 1.2, OOSTrades: 15, OOSReturnPct: 2.0, OOSWinRate: 58},
				{OutSampleSharpe: 1.3, OOSTrades: 18, OOSReturnPct: 3.0, OOSWinRate: 60},
				{OutSampleSharpe: 1.1, OOSTrades: 12, OOSReturnPct: -1.0, OOSWinRate: 55},
			},
			ProfitFactor:    2.1,
			SharpeDegradation: 15.0,
		},
	}

	mcResult := &MonteCarloResult{
		PassProbability: 85.0,
	}

	std := DefaultMultiMetricStandard()
	verdict := EvaluateOOSMultiMetric(owr, mcResult, std)

	if !verdict.Passed {
		t.Errorf("Expected PASS for valid OOS walk-forward: %s", verdict.Summary())
	}
}

func TestEvaluateOOSMultiMetric_Fail(t *testing.T) {
	owr := &OptimizedWalkForwardResult{
		WalkForwardResult: WalkForwardResult{
			Windows: []WindowResult{
				{OutSampleSharpe: 0.2, OOSTrades: 5, OOSReturnPct: -10.0, OOSWinRate: 30},
				{OutSampleSharpe: 0.1, OOSTrades: 3, OOSReturnPct: -15.0, OOSWinRate: 25},
			},
			ProfitFactor:     0.6,
			SharpeDegradation: 75.0,
		},
	}

	mcResult := &MonteCarloResult{
		PassProbability: 25.0,
	}

	std := DefaultMultiMetricStandard()
	verdict := EvaluateOOSMultiMetric(owr, mcResult, std)

	if verdict.Passed {
		t.Error("Expected FAIL for poor OOS metrics")
	}
	t.Logf("OOS Fail summary: %s", verdict.Summary())
}

func TestEvaluateOOSMultiMetric_NilInputs(t *testing.T) {
	std := DefaultMultiMetricStandard()

	verdict := EvaluateOOSMultiMetric(nil, nil, std)
	if verdict.Passed {
		t.Error("Expected FAIL for nil walk-forward result")
	}

	emptyOwr := &OptimizedWalkForwardResult{}
	verdict = EvaluateOOSMultiMetric(emptyOwr, nil, std)
	if verdict.Passed {
		t.Error("Expected FAIL for empty walk-forward windows")
	}
}

func TestEvaluateBacktestMultiMetric_NilInput(t *testing.T) {
	std := DefaultMultiMetricStandard()
	verdict := EvaluateBacktestMultiMetric(nil, std)
	if verdict.Passed {
		t.Error("Expected FAIL for nil backtest result")
	}
}

func TestMultiMetricVerdict_JSON(t *testing.T) {
	v := MultiMetricVerdict{
		Passed:      true,
		PassedCount: 3,
		TotalCount:  3,
		Timestamp:   time.Now(),
		Verdicts: []MetricVerdict{
			{Metric: "sharpe_ratio", Required: 1.0, Actual: 1.8, Passed: true},
			{Metric: "max_drawdown", Required: 8.0, Actual: 4.5, Passed: true},
			{Metric: "min_trades", Required: 30, Actual: 45, Passed: true},
		},
	}

	jsonStr := v.JSON()
	if jsonStr == "" {
		t.Error("JSON output should not be empty")
	}
	if len(jsonStr) < 50 {
		t.Error("JSON output too short")
	}
}

func TestMultiMetricVerdict_Summary(t *testing.T) {
	v := MultiMetricVerdict{
		Passed:      false,
		PassedCount: 2,
		TotalCount:  3,
		Timestamp:   time.Now(),
		Verdicts: []MetricVerdict{
			{Metric: "sharpe_ratio", Required: 1.0, Actual: 0.3, Passed: false},
			{Metric: "max_drawdown", Required: 8.0, Actual: 4.5, Passed: true},
			{Metric: "min_trades", Required: 30, Actual: 45, Passed: true},
		},
	}

	summary := v.Summary()
	if summary == "" {
		t.Error("Summary should not be empty")
	}
	t.Logf("Summary: %s", summary)
}

func TestEvaluateBacktestMultiMetric_WithSortino(t *testing.T) {
	result := &BacktestResult{
		SharpeRatio:    2.0,
		SortinoRatio:   0.5,
		MaxDrawdown:    3.0,
		ProfitFactor:   2.5,
		NumTrades:      60,
		WinRate:        65.0,
	}

	std := StrictMultiMetricStandard()
	verdict := EvaluateBacktestMultiMetric(result, std)

	if verdict.Passed {
		t.Logf("Strict standard with Sortino 0.5: %s", verdict.Summary())
	}
}

func TestMultiMetricStandard_AllPresets(t *testing.T) {
	standards := map[string]MultiMetricStandard{
		"default": DefaultMultiMetricStandard(),
		"lenient": LenientMultiMetricStandard(),
		"strict":  StrictMultiMetricStandard(),
	}

	result := &BacktestResult{
		SharpeRatio:    1.0,
		SortinoRatio:   0.8,
		MaxDrawdown:    7.0,
		ProfitFactor:   1.6,
		NumTrades:      35,
		WinRate:        55.0,
	}

	for name, std := range standards {
		verdict := EvaluateBacktestMultiMetric(result, std)
		t.Logf("%s standard: %s", name, verdict.Summary())
	}
}
