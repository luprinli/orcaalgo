package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/lee-econ/orca-core/internal/backtest"
	"github.com/lee-econ/orca-core/internal/benchmark"
	"github.com/lee-econ/orca-core/internal/config"
	"github.com/lee-econ/orca-core/internal/db"
)

func main() {
	optimize := flag.Bool("optimize", false, "Quick full-period light optimization, then re-run with those params (fast approximation; use --walk-forward for OOS-validated params)")
	walkForward := flag.Bool("walk-forward", false, "Embedded walk-forward: per-window IS optimization -> OOS with IVS plateau detection (canonical, single source of truth)")
	pipeline := flag.Bool("pipeline", true, "Wire the RiskPipeline for per-signal gating (HP#17 canonical path; disable with --pipeline=false)")
	dataSource := flag.String("data-source", "stooq", "Data source for candles (stooq | yahoo | synthetic)")
	benchmarkSymbol := flag.String("benchmark", "", "Benchmark ticker for the market-based filter (e.g. SPY). Empty disables BenchmarkPass/IR/Alpha columns.")
	flag.Parse()

	cfg := db.DefaultConfig()
	repo, err := db.NewRepository(cfg)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer repo.Close()
	dbAdapter := &repoAdapter{repo: repo}

	outputPath := os.Getenv("MATRIX_OUTPUT")
	if outputPath == "" {
		outputPath = filepath.Join("data", ".backtest_results", "matrix_results.csv")
	}
	outputPath, _ = filepath.Abs(outputPath)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	// Default 3-year window, matching the frontend matrix runner's default so the
	// CLI and API produce comparable baselines out of the box.
	start := today.AddDate(-3, 0, 0)
	end := today
	// Honor explicit window overrides (set by scripts/run-matrix.ps1). Walk-
	// forward needs a multi-year window, so MATRIX_START/MATRIX_END matter here.
	if s := os.Getenv("MATRIX_START"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			start = t
		}
	}
	if s := os.Getenv("MATRIX_END"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			end = t
		}
	}

	symbols := envList("MATRIX_SYMBOLS", configDefaultTickers())
	strategies := envList("MATRIX_STRATEGIES", configDefaultStrategies())
	timeframes := configDefaultTimeframes()

	total := 0
	for _, s := range strategies {
		for _, tf := range timeframes {
			if backtest.StrategyTimeframeAllowed(s, tf) {
				total += len(symbols)
			}
		}
	}
	fmt.Printf("Matrix: %d combos (%d strategies x %d symbols x %d timeframes, timeframe-restricted)\n", total, len(strategies), len(symbols), len(timeframes))
	fmt.Printf("Period: %s to %s\n", start.Format("2006-01-02"), end.Format("2006-01-02"))
	fmt.Printf("Output: %s\n", outputPath)
	if *optimize {
		fmt.Println("  Light optimizer: ENABLED")
	}
	if *walkForward {
		fmt.Println("  Walk-forward validation: ENABLED (trainPct=0.66, nWindows=3)")
	}
	fmt.Println()

	os.MkdirAll(filepath.Dir(outputPath), 0755)
	f, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("create csv: %v", err)
	}
	defer f.Close()
	w := csv.NewWriter(f)

	header := []string{"Strategy", "Symbol", "Tf", "Trades", "Wins", "Losses", "Sharpe", "Sortino", "MaxDD%", "Return%", "WinRate", "ProfitFactor", "AvgWin", "AvgLoss", "LongTrades", "ShortTrades", "LongWinRate", "ShortWinRate", "LongGrossPnL", "ShortGrossPnL", "LongPF", "ShortPF", "MFE", "MAE", "GatePassed", "Optimized", "TrainPct", "Params", "Reliable", "TotalFees", "AvgSlippageBps", "CalmarRatio", "CandleCount", "Status", "MtmMaxDD%", "MtmSharpe", "FirstCandle", "LastCandle", "SigAttempts", "SigPassed", "PipelineRej", "VolHaltRej", "MLRejected", "RegimeRej", "FillRej", "StrategyNil", "ExitZeroQty", "PipelineRejBreakdown", "DeclaredBPD", "EffectiveBPD", "Warnings", "DataGenID"}
	if *walkForward {
		header = append(header, "WfISSharpe", "WfOOSSharpe", "WfPassedWindows", "WfTotalWindows", "WfReturnPct")
	}
	if *benchmarkSymbol != "" {
		header = append(header, "BenchmarkPass", "BenchmarkIR", "BenchmarkAlpha")
	}
	w.Write(header)

	ctx := context.Background()

	var benchReturns map[string]float64
	if *benchmarkSymbol != "" {
		benchReturns = loadBenchmarkDailyReturns(ctx, dbAdapter, *benchmarkSymbol, start, end)
		if len(benchReturns) > 0 {
			fmt.Printf("Benchmark eval: %s (%d daily bars)\n", *benchmarkSymbol, len(benchReturns))
		} else {
			fmt.Printf("Benchmark eval: WARNING no daily bars for %s — BenchmarkPass/IR/Alpha will be N/A\n", *benchmarkSymbol)
		}
	}
	var completed int64
	startTime := time.Now()

	for _, s := range strategies {
		for _, sym := range symbols {
			for _, tf := range timeframes {
				if !backtest.StrategyTimeframeAllowed(s, tf) {
					continue
				}
				config := backtest.BacktestConfig{
					StrategyID: s, Symbols: []string{sym},
					StartDate: start, EndDate: end, InitialCapital: 100000,
					Timeframe: tf, DataSource: *dataSource, SizingPercent: 0.02, KellyFraction: 0.25,
					WarmUpBars:  50,
					ApplyGate:   true,
					GateProfile: "research",
				}

				optimized := false
				params := ""
				wfResults := &backtest.OptimizedWalkForwardResult{}

				if *optimize {
					optResult := runLightOptimizeCombo(ctx, dbAdapter, config)
					if optResult != nil {
						config.StrategyParams = optResult
						optimized = true
						params = jsonParams(optResult)
					}
				}

				engine := backtest.NewEngine(dbAdapter)
				if *pipeline {
					engine.WirePipeline()
				}
				result, err := engine.Run(ctx, config)
				n := atomic.AddInt64(&completed, 1)

				if err != nil {
					fmt.Printf("[%d/%d] %s %s %s: ERROR %v\n", n, total, s, sym, tf, err)
					errRow := []string{s, sym, tf,
						"0", "0", "0",
						"N/A", "N/A", "N/A", "N/A", "N/A", "N/A",
						"N/A", "N/A",
						"0", "0", "0", "0", "0", "0", "0", "0", "0", "0",
						"false", "false", "0.00", "", "false",
						"0.00", "0.0000", "0.0000", "0",
						fmt.Sprintf("error: %v", err),
						"N/A", "N/A", "N/A", "N/A",
						"0", "0", "0", "0", "0", "0", "0", "0", "0", "",
						"N/A", "N/A", "", "",
					}
					if *walkForward {
						errRow = append(errRow, "N/A", "N/A", "0", "0", "N/A")
					}
					if *benchmarkSymbol != "" {
						errRow = append(errRow, "N/A", "N/A", "N/A")
					}
					w.Write(errRow)
					continue
				}

				if *walkForward && (optimized || len(result.StrategyParams) > 0 || (result.NumTrades >= 20 && result.SharpeRatio > 0)) {
					wfResults = runWalkForwardCombo(ctx, dbAdapter, config, engine)
					if wfResults == nil {
						wfResults = &backtest.OptimizedWalkForwardResult{}
					}
				}

				status := "ok"
				if result.NumTrades == 0 {
					status = "zero_trades"
				}

				if params == "" && len(result.StrategyParams) > 0 {
					params = jsonParams(result.StrategyParams)
				}
				gate := "false"
				if result.MetricGateStatus != nil {
					gate = fmt.Sprintf("%v", result.MetricGateStatus.Passed)
				}

				reliable := "true"
				if result.NumTrades < 20 {
					reliable = "false"
				}

				warningsStr := ""
				if len(result.Warnings) > 0 {
					warningsStr = result.Warnings[0]
					for _, w := range result.Warnings[1:] {
						warningsStr += "; " + w
					}
				}

				sharpe := fmt.Sprintf("%.4f", result.SharpeRatio)
				sortino := fmt.Sprintf("%.4f", result.SortinoRatio)
				profitFactor := fmt.Sprintf("%.4f", result.ProfitFactor)
				if result.NumTrades < 20 {
					sharpe = "N/A"
					sortino = "N/A"
					profitFactor = "N/A"
				}

				optStr := "false"
				if optimized {
					optStr = "true"
				}

				row := []string{s, sym, tf,
					fmt.Sprintf("%d", result.NumTrades), fmt.Sprintf("%d", result.NumWins), fmt.Sprintf("%d", result.NumLosses),
					sharpe, sortino,
					fmt.Sprintf("%.2f", result.MaxDrawdown), fmt.Sprintf("%.2f", result.TotalReturnPct),
					fmt.Sprintf("%.4f", result.WinRate), profitFactor,
					fmt.Sprintf("%.2f", result.AvgWin), fmt.Sprintf("%.2f", result.AvgLoss),
					fmt.Sprintf("%d", result.LongShort.LongTrades), fmt.Sprintf("%d", result.LongShort.ShortTrades),
					fmt.Sprintf("%.4f", result.LongShort.LongWinRate), fmt.Sprintf("%.4f", result.LongShort.ShortWinRate),
					fmt.Sprintf("%.2f", result.LongShort.LongGrossPnL), fmt.Sprintf("%.2f", result.LongShort.ShortGrossPnL),
					fmt.Sprintf("%.4f", result.LongShort.LongPF), fmt.Sprintf("%.4f", result.LongShort.ShortPF),
					fmt.Sprintf("%.4f", result.AvgMAE), fmt.Sprintf("%.4f", result.AvgMFE),
					gate, optStr, fmt.Sprintf("%.2f", result.TrainPct), params, reliable,
					fmt.Sprintf("%.2f", result.TotalFees), fmt.Sprintf("%.4f", result.AvgSlippageBps),
					fmt.Sprintf("%.4f", result.CalmarRatio), fmt.Sprintf("%d", result.CandleCount),
					status,
					fmt.Sprintf("%.2f", result.MtmMaxDrawdown),
					fmt.Sprintf("%.4f", result.MtmSharpeRatio),
					result.FirstCandleTime.Format("2006-01-02"),
					result.LastCandleTime.Format("2006-01-02"),
					fmt.Sprintf("%d", result.SignalDiag.SignalAttempts),
					fmt.Sprintf("%d", result.SignalDiag.SignalsPassed),
					fmt.Sprintf("%d", result.SignalDiag.PipelineRejected),
					fmt.Sprintf("%d", result.SignalDiag.VolHalted),
					fmt.Sprintf("%d", result.SignalDiag.MLRejected),
					fmt.Sprintf("%d", result.SignalDiag.RegimeRejected),
					fmt.Sprintf("%d", result.SignalDiag.FillRejected),
					fmt.Sprintf("%d", result.SignalDiag.StrategyNil),
					fmt.Sprintf("%d", result.SignalDiag.ExitSignalZeroQty),
					pipelineRejectBreakdown(result.SignalDiag.PipelineRejects),
					fmt.Sprintf("%.1f", result.DeclaredBarsPerDay),
					fmt.Sprintf("%.1f", result.EffectiveBarsPerDay),
					warningsStr,
					result.DataGenerationID,
				}

				if *walkForward {
					if wfResults != nil && wfResults.TotalWindows > 0 {
						row = append(row,
							fmt.Sprintf("%.4f", wfResults.OverallSharpe),
							fmt.Sprintf("%.4f", wfResults.AvgOOSSharpe),
							fmt.Sprintf("%d", wfResults.PassedWindows),
							fmt.Sprintf("%d", wfResults.TotalWindows),
							fmt.Sprintf("%.2f", wfResults.TotalReturnPct),
						)
					} else {
						row = append(row, "N/A", "N/A", "0", "0", "N/A")
					}
				}

				if *benchmarkSymbol != "" {
					bp, ir, alpha := evalBenchmark(ctx, result, benchReturns, *benchmarkSymbol)
					row = append(row, bp, ir, alpha)
				}

				w.Write(row)

				if n%200 == 0 {
					w.Flush()
					elapsed := time.Since(startTime)
					rate := float64(n) / elapsed.Seconds()
					rem := time.Duration(float64(total-int(n))/rate) * time.Second
					fmt.Printf("[%d/%d %.1f%%] %s %s %s trades=%d sharpe=%s | %.1f/s ETA %s\n",
						n, total, float64(n)/float64(total)*100, s, sym, tf, result.NumTrades, sharpe, rate, rem.Truncate(time.Second))
				}
			}
		}
	}
	w.Flush()
	fmt.Printf("\nDone! %d combos in %s\n", completed, time.Since(startTime).Truncate(time.Second))
}

func runLightOptimizeCombo(ctx context.Context, db backtest.Database, baseCfg backtest.BacktestConfig) map[string]float64 {
	optCfg := backtest.LightOptimizeConfig{
		StrategyID:      baseCfg.StrategyID,
		Symbols:         baseCfg.Symbols,
		Timeframe:       baseCfg.Timeframe,
		StartDate:       baseCfg.StartDate,
		EndDate:         baseCfg.EndDate,
		InitialCapital:  baseCfg.InitialCapital,
		GateProfile:     baseCfg.GateProfile,
		SizingPercent:   baseCfg.SizingPercent,
		PropFirmEnabled: baseCfg.PropFirmEnabled,
	}
	return backtest.RunLightOptimize(ctx, db, optCfg)
}

func runWalkForwardCombo(ctx context.Context, db backtest.Database, baseCfg backtest.BacktestConfig, engine *backtest.Engine) *backtest.OptimizedWalkForwardResult {
	owfCfg := backtest.NewOptimizedWalkForwardConfig(baseCfg, baseCfg.StrategyID)
	result, err := engine.RunOptimizedWalkForward(ctx, owfCfg)
	if err != nil {
		return nil
	}
	return result
}

// loadBenchmarkDailyReturns loads 1d stooq candles for the benchmark symbol and
// returns close-to-close decimal returns keyed by "2006-01-02" date. Computed
// once per run — the benchmark series is identical across all combos.
func loadBenchmarkDailyReturns(ctx context.Context, db backtest.Database, symbol string, start, end time.Time) map[string]float64 {
	candles, err := db.LoadCandlesFiltered(ctx, []string{symbol}, start, end, "stooq")
	if err != nil || len(candles) == 0 || len(candles[0]) < 2 {
		return nil
	}
	bars := candles[0]
	out := make(map[string]float64, len(bars)-1)
	for i := 1; i < len(bars); i++ {
		prev := bars[i-1].Close.Float64()
		cur := bars[i].Close.Float64()
		if prev > 0 {
			out[bars[i].Time.Format("2006-01-02")] = cur/prev - 1.0
		}
	}
	return out
}

// evalBenchmark computes the market-based benchmark filter verdict for one combo
// by aligning its daily returns to the (shared) benchmark series and shelling
// out to `orca benchmark-filter`. Returns CSV cells (pass, IR, alpha).
func evalBenchmark(ctx context.Context, result *backtest.BacktestResult, benchReturns map[string]float64, symbol string) (string, string, string) {
	if result == nil || len(benchReturns) == 0 {
		return "N/A", "N/A", "N/A"
	}
	var strat, bench []float64
	for _, d := range result.DailyReturns {
		if r, ok := benchReturns[d.Date.Format("2006-01-02")]; ok {
			strat = append(strat, d.Return/100.0)
			bench = append(bench, r)
		}
	}
	if len(strat) < 3 {
		return "N/A", "N/A", "N/A"
	}
	verdict, err := benchmark.Evaluate(ctx, strat, bench, "equity_index", symbol, 0, 1)
	if err != nil {
		return "N/A", "N/A", "N/A"
	}
	ir := "N/A"
	alpha := "N/A"
	if verdict.Metrics.InformationRatio != nil {
		ir = fmt.Sprintf("%.4f", *verdict.Metrics.InformationRatio)
	}
	if verdict.Metrics.AlphaAnnualized != nil {
		alpha = fmt.Sprintf("%.4f", *verdict.Metrics.AlphaAnnualized)
	}
	return fmt.Sprintf("%t", verdict.Passed), ir, alpha
}

func configDefaultTickers() []string {
	if u, err := config.Load(); err == nil {
		tickers := make([]string, len(u.Symbols))
		for i, s := range u.Symbols {
			tickers[i] = s.Ticker
		}
		return tickers
	}
	return []string{"SPY", "QQQ", "AAPL", "MSFT", "NVDA", "TSLA", "IWM", "GLD", "TLT", "EURUSD", "GBPUSD", "USDJPY", "AUDUSD", "USDCAD", "BTC-USD", "ETH-USD", "^_US", "^DAX"}
}

func configDefaultStrategies() []string {
	if u, err := config.Load(); err == nil && len(u.Strategies) > 0 {
		return u.Strategies
	}
	return []string{"grid_trading", "trend_following", "session_scalp", "intraday_mr", "vwap_mr", "opening_range_breakout", "orb_15m", "pairs_trading", "volatility_harvesting", "dragon_trend", "volume_scalp", "vix_futures_carry", "ma_crossover", "rsi2_reversion", "donchian_breakout", "keltner_macd", "ichimoku_cloud"}
}

func configDefaultTimeframes() []string {
	if u, err := config.Load(); err == nil && len(u.Timeframes) > 0 {
		return u.Timeframes
	}
	return []string{"5m", "15m", "30m", "1h", "4h", "1d"}
}

func envList(key string, defaults []string) []string {
	if v := os.Getenv(key); v != "" {
		var out []string
		for _, s := range splitComma(v) {
			if s = trim(s); s != "" {
				out = append(out, s)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return defaults
}

func splitComma(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}

func trim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

// jsonParams serializes a params map as compact JSON so the `Params` CSV column
// is machine-parseable. The CI Kelly scan (validate-matrix.ps1) matches the
// JSON form `"kelly_fraction":0.25`; Go's `%v` map formatting would not.
func jsonParams(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

// pipelineRejectBreakdown renders the reason-keyed pipeline rejection counts as
// a compact JSON object (e.g. {"exposure:equity_negative":3}), or "" when empty.
func pipelineRejectBreakdown(m map[string]int) string {
	if len(m) == 0 {
		return ""
	}
	b, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(b)
}
