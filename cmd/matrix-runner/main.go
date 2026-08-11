package main

import (
	"context"
	"encoding/csv"
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
	cfg := db.Config{Host: "localhost", Port: 5433, User: "orca", Password: "change_me", Database: "orca_core", SSLMode: "disable", PoolMax: 3, PoolMin: 1}
	repo, err := db.NewRepository(cfg)
	if err != nil { log.Fatalf("connect: %v", err) }
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
	strategies := envList("MATRIX_STRATEGIES", []string{"grid_trading", "trend_following", "session_scalp", "intraday_mr", "vwap_mr", "opening_range_breakout", "orb_15m", "pairs_trading", "volatility_harvesting", "dragon_trend", "volume_scalp", "vix_futures_carry", "ma_crossover", "rsi2_reversion", "donchian_breakout", "keltner_macd", "ichimoku_cloud"})
	timeframes := []string{"5m", "15m", "30m", "1h", "4h", "1d"}

	total := len(strategies) * len(symbols) * len(timeframes)
	fmt.Printf("Matrix: %d combos (%d strategies x %d symbols x %d timeframes)\n", total, len(strategies), len(symbols), len(timeframes))
	fmt.Printf("Period: %s to %s\n", start.Format("2006-01-02"), end.Format("2006-01-02"))
	fmt.Printf("Output: %s\n\n", outputPath)

	os.MkdirAll(filepath.Dir(outputPath), 0755)
	f, err := os.Create(outputPath)
	if err != nil { log.Fatalf("create csv: %v", err) }
	defer f.Close()
	w := csv.NewWriter(f)
	w.Write([]string{"Strategy", "Symbol", "Tf", "Trades", "Wins", "Losses", "Sharpe", "Sortino", "MaxDD%", "Return%", "WinRate", "ProfitFactor", "AvgWin", "AvgLoss", "LongTrades", "ShortTrades", "LongWinRate", "ShortWinRate", "LongGrossPnL", "ShortGrossPnL", "LongPF", "ShortPF", "MFE", "MAE", "GatePassed", "Optimized", "TrainPct", "Params"})

	ctx := context.Background()
	var completed int64
	startTime := time.Now()

	for _, s := range strategies {
		for _, sym := range symbols {
			for _, tf := range timeframes {
				config := backtest.BacktestConfig{
					StrategyID: s, Symbols: []string{sym},
					StartDate: start, EndDate: end, InitialCapital: 100000,
					Timeframe: tf, DataSource: "", SizingPercent: 0.02, KellyFraction: 0.25,
					WarmUpBars: 50,
				}
				engine := backtest.NewEngine(dbAdapter)
				result, err := engine.Run(ctx, config)
				n := atomic.AddInt64(&completed, 1)

				if err != nil {
					fmt.Printf("[%d/%d] %s %s %s: ERROR %v\n", n, total, s, sym, tf, err)
					continue
				}

				opt := "false"
				params := ""
				if len(result.StrategyParams) > 0 { opt = "true" }
				gate := ""
				if result.MetricGateStatus != nil { gate = fmt.Sprintf("%v", *result.MetricGateStatus) }

				w.Write([]string{s, sym, tf,
					fmt.Sprintf("%d", result.NumTrades), fmt.Sprintf("%d", result.NumWins), fmt.Sprintf("%d", result.NumLosses),
					fmt.Sprintf("%.4f", result.SharpeRatio), fmt.Sprintf("%.4f", result.SortinoRatio),
					fmt.Sprintf("%.2f", result.MaxDrawdown*100), fmt.Sprintf("%.2f", result.TotalReturnPct),
					fmt.Sprintf("%.4f", result.WinRate), fmt.Sprintf("%.4f", result.ProfitFactor),
					fmt.Sprintf("%.2f", result.AvgWin), fmt.Sprintf("%.2f", result.AvgLoss),
					fmt.Sprintf("%d", result.LongShort.LongTrades), fmt.Sprintf("%d", result.LongShort.ShortTrades),
					fmt.Sprintf("%.4f", result.LongShort.LongWinRate), fmt.Sprintf("%.4f", result.LongShort.ShortWinRate),
					fmt.Sprintf("%.2f", result.LongShort.LongGrossPnL), fmt.Sprintf("%.2f", result.LongShort.ShortGrossPnL),
					fmt.Sprintf("%.4f", result.LongShort.LongPF), fmt.Sprintf("%.4f", result.LongShort.ShortPF),
					fmt.Sprintf("%.4f", result.AvgMAE), fmt.Sprintf("%.4f", result.AvgMFE),
					gate, opt, fmt.Sprintf("%.2f", result.TrainPct), params,
				})

				if n%200 == 0 {
					w.Flush()
					elapsed := time.Since(startTime)
					rate := float64(n) / elapsed.Seconds()
					rem := time.Duration(float64(total-int(n))/rate) * time.Second
					fmt.Printf("[%d/%d %.1f%%] %s %s %s trades=%d sharpe=%.4f | %.1f/s ETA %s\n",
						n, total, float64(n)/float64(total)*100, s, sym, tf, result.NumTrades, result.SharpeRatio, rate, rem.Truncate(time.Second))
				}
			}
		}
	}
	w.Flush()
	fmt.Printf("\nDone! %d combos in %s\n", completed, time.Since(startTime).Truncate(time.Second))
}

func envList(key string, defaults []string) []string {
	if v := os.Getenv(key); v != "" {
		var out []string
		for _, s := range splitComma(v) {
			if s = trim(s); s != "" { out = append(out, s) }
		}
		if len(out) > 0 { return out }
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
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') { s = s[1:] }
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') { s = s[:len(s)-1] }
	return s
}
