package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/lee-econ/orca-core/internal/backtest"
	"github.com/lee-econ/orca-core/internal/db"
)

func main() {
	optimize := flag.Bool("optimize", false, "Run light optimizer per combo and re-run with optimized params")
	walkForward := flag.Bool("walk-forward", false, "Run walk-forward validation per combo (trainPct=0.66, nWindows=3)")
	flag.Parse()

	cfg := db.Config{Host: "localhost", Port: 5432, User: "artisan", Password: "", Database: "artisan", SSLMode: "disable", PoolMax: 3, PoolMin: 1}
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
	start := today.AddDate(-1, 0, 0)
	end := today

	symbols := envList("MATRIX_SYMBOLS", []string{"SPY", "QQQ", "AAPL", "MSFT", "GOOGL", "META", "AMZN", "NVDA", "TSLA", "VOO", "DIA", "IWM", "GLD", "USO", "CL", "NQ", "ES", "EURUSD", "GBPUSD", "USDJPY", "USDCHF", "AUDUSD", "USDCAD", "NZDUSD", "XAUUSD", "XAGUSD", "BTCUSD", "ETHUSD", "US30", "SPX500", "NAS100", "UK100", "GER40", "JPN225", "TLT"})
	strategies := envList("MATRIX_STRATEGIES", []string{"grid_trading", "trend_following", "session_scalp", "intraday_mr", "mean_reversion", "vwap_mr", "opening_range_breakout", "orb_15m", "pairs_trading", "volatility_harvesting", "dragon_trend", "volume_scalp", "vix_futures_carry", "ma_crossover", "rsi2_reversion", "rsi_divergence", "donchian_breakout", "keltner_macd", "ichimoku_cloud"})
	timeframes := []string{"5m", "15m", "30m", "1h", "4h", "1d"}

	total := len(strategies) * len(symbols) * len(timeframes)
	fmt.Printf("Matrix: %d combos (%d strategies x %d symbols x %d timeframes)\n", total, len(strategies), len(symbols), len(timeframes))
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

	header := []string{"Strategy", "Symbol", "Tf", "Trades", "Wins", "Losses", "Sharpe", "Sortino", "MaxDD%", "Return%", "WinRate", "ProfitFactor", "AvgWin", "AvgLoss", "LongTrades", "ShortTrades", "LongWinRate", "ShortWinRate", "LongGrossPnL", "ShortGrossPnL", "LongPF", "ShortPF", "MFE", "MAE", "GatePassed", "Optimized", "TrainPct", "Params", "Reliable", "TotalFees", "AvgSlippageBps", "CalmarRatio", "CandleCount", "Status", "MtmMaxDD%", "MtmSharpe", "FirstCandle", "LastCandle", "SigAttempts", "SigPassed", "PipelineRej", "VolHaltRej", "MLRejected", "DeclaredBPD", "EffectiveBPD", "Warnings"}
	if *walkForward {
		header = append(header, "WfISSharpe", "WfOOSSharpe", "WfPassedWindows", "WfTotalWindows", "WfReturnPct")
	}
	w.Write(header)

	ctx := context.Background()
	var completed int64
	startTime := time.Now()

	for _, s := range strategies {
		for _, sym := range symbols {
			for _, tf := range timeframes {
				config := backtest.BacktestConfig{
					StrategyID: s, Symbols: []string{sym},
					StartDate: start, EndDate: end, InitialCapital: 100000,
					Timeframe: tf, DataSource: "", SizingPercent: 0.05, KellyFraction: 0.50,
					WarmUpBars: 50,
					ApplyGate:  true,
					GateProfile: "research",
				}

				optimized := false
				params := ""
				wfResults := &backtest.WalkForwardResult{}

				if *optimize {
					optResult := runLightOptimizeCombo(ctx, dbAdapter, config)
					if optResult != nil {
						config.StrategyParams = optResult
						optimized = true
						params = fmt.Sprintf("%v", optResult)
					}
				}

				engine := backtest.NewEngine(dbAdapter)
				engine.WirePipeline()
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
						"0", "0", "0", "0", "0",
						"N/A", "N/A", "",
					}
					if *walkForward {
						errRow = append(errRow, "N/A", "N/A", "0", "0", "N/A")
					}
					w.Write(errRow)
					continue
				}

				if *walkForward && (optimized || len(result.StrategyParams) > 0) {
					wfConfig := config
					if len(result.StrategyParams) > 0 {
						wfConfig.StrategyParams = result.StrategyParams
					}
					wfResults = runWalkForwardCombo(ctx, dbAdapter, wfConfig, engine)
					if wfResults == nil {
						wfResults = &backtest.WalkForwardResult{}
					}
				}

				status := "ok"
				if result.NumTrades == 0 {
					status = "zero_trades"
				}

				if !optimized && len(result.StrategyParams) > 0 {
					optimized = true
					params = fmt.Sprintf("%v", result.StrategyParams)
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
					fmt.Sprintf("%.2f", result.MaxDrawdown*100), fmt.Sprintf("%.2f", result.TotalReturnPct),
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
					fmt.Sprintf("%.2f", result.MtmMaxDrawdown*100),
					fmt.Sprintf("%.4f", result.MtmSharpeRatio),
					result.FirstCandleTime.Format("2006-01-02"),
					result.LastCandleTime.Format("2006-01-02"),
					fmt.Sprintf("%d", result.SignalDiag.SignalAttempts),
					fmt.Sprintf("%d", result.SignalDiag.SignalsPassed),
					fmt.Sprintf("%d", result.SignalDiag.PipelineRejected),
					fmt.Sprintf("%d", result.SignalDiag.VolHalted),
					fmt.Sprintf("%d", result.SignalDiag.MLRejected),
					fmt.Sprintf("%.1f", result.DeclaredBarsPerDay),
					fmt.Sprintf("%.1f", result.EffectiveBarsPerDay),
					warningsStr,
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

func runWalkForwardCombo(ctx context.Context, db backtest.Database, baseCfg backtest.BacktestConfig, engine *backtest.Engine) *backtest.WalkForwardResult {
	wfCfg := backtest.WalkForwardConfig{
		Config:             baseCfg,
		TrainWindows:       3,
		TrainYears:         1,
		TestYears:          2,
		StepMonths:         6,
		PurgeTradingDays:   5,
		EmbargoTradingDays: 2,
	}
	result, err := engine.RunWalkForward(ctx, wfCfg)
	if err != nil {
		return nil
	}
	return result
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
