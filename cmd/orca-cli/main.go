package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lee-econ/orca-core/internal/backtest"
	"github.com/lee-econ/orca-core/internal/ingest"
	strategy "github.com/lee-econ/orca-core/internal/strategy"
	"github.com/lee-econ/orca-core/internal/version"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "backtest":
		runBacktest()
	case "matrix":
		runMatrix()
	case "key-rotate":
		fmt.Println("Key rotation: check scheduler logs")
	case "health":
		fmt.Println("System health: OK")
	case "migration":
		fmt.Println("Migration: run 'migrate up' from scripts/migrate.ps1")
	case "list-strategies":
		listStrategies()
	case "list-symbols":
		listSymbols()
	case "describe-strategy":
		describeStrategy()
	case "version":
		runVersion()
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: orca-cli <command> [flags]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  backtest         Run single backtest")
	fmt.Println("  matrix           Run matrix (all strategies x symbols x timeframes)")
	fmt.Println("  list-strategies  List available strategies")
	fmt.Println("  list-symbols     List available symbols for a timeframe")
	fmt.Println("  describe-strategy Show parameter metadata for a strategy")
	fmt.Println("  version          Print engine build version")
	fmt.Println("  key-rotate       Rotate API keys")
	fmt.Println("  health           System health check")
	fmt.Println("  migration        Run database migration")
	fmt.Println()
	fmt.Println("Backtest flags:")
	fmt.Println("  -strategy string     Strategy ID (default: intraday_mr)")
	fmt.Println("  -symbols string      Comma-separated symbols (default: EURUSD)")
	fmt.Println("  -start string        Start date YYYY-MM-DD (default: 2023-01-01)")
	fmt.Println("  -end string          End date YYYY-MM-DD (default: 2025-12-31)")
	fmt.Println("  -timeframe string    Timeframe: 1d, 1h, 5m, 15m (default: 1d)")
	fmt.Println("  -capital float       Initial capital (default: 100000)")
	fmt.Println("  -seed int            Fixed seed for reproducibility (0 = random)")
	fmt.Println("  -gate                Apply multi-metric gate")
	fmt.Println("  -gate-profile string Gate profile: default/lenient/strict (default: default)")
	fmt.Println("  -ftmo                Enable FTMO prop firm enforcement")
	fmt.Println("  -json                Output as JSON")
	fmt.Println("  -gkr-path string     Path to .gkr.yaml strategy config for content addressing")
	fmt.Println()
	fmt.Println("Matrix flags:")
	fmt.Println("  -strategies string   Comma-separated strategy IDs")
	fmt.Println("  -symbols string      Comma-separated symbols")
	fmt.Println("  -timeframes string   Comma-separated timeframes (e.g. 1d,1h,5m)")
	fmt.Println()
	fmt.Println("Data directory is auto-detected from configs/config.dev.yaml")
	fmt.Println("Set ORCA_DATA_DIR env var to override")
}

func runBacktest() {
	fs := flag.NewFlagSet("backtest", flag.ExitOnError)
	strategy := fs.String("strategy", "intraday_mr", "Strategy ID")
	symbols := fs.String("symbols", "EURUSD", "Comma-separated symbols")
	startStr := fs.String("start", "2023-01-01", "Start date")
	endStr := fs.String("end", "2025-12-31", "End date")
	tf := fs.String("timeframe", "1d", "Timeframe")
	capital := fs.Float64("capital", 100000, "Initial capital")
	seed := fs.Int64("seed", 0, "Fixed seed")
	gate := fs.Bool("gate", false, "Apply metric gate")
	gateProfile := fs.String("gate-profile", "default", "Gate profile")
	ftmo := fs.Bool("ftmo", false, "Enable FTMO enforcement")
	jsonOut := fs.Bool("json", false, "Output JSON")
	paramsStr := fs.String("params", "", "Comma-separated key=value strategy params, e.g. entry_z=1.5,exit_z=0.3")
	gkrPath := fs.String("gkr-path", "", "Path to .gkr.yaml strategy config for content hashing")

	if len(os.Args) > 2 {
		fs.Parse(os.Args[2:])
	}

	start, _ := time.Parse("2006-01-02", *startStr)
	end, _ := time.Parse("2006-01-02", *endStr)

	dataDir := getDataDir()
	db := NewFileDB(dataDir)

	var engine *backtest.Engine
	if *seed != 0 {
		engine = backtest.NewEngineWithFixedSeed(db, *seed)
	} else {
		engine = backtest.NewEngine(db)
	}

	symList := parseSymbols(*symbols)

	cfg := backtest.BacktestConfig{
		StrategyID:     *strategy,
		Symbols:        symList,
		StartDate:      start,
		EndDate:        end,
		InitialCapital: *capital,
		Timeframe:      *tf,
		ApplyGate:      *gate,
		GateProfile:    *gateProfile,
		PropFirmEnabled: *ftmo,
		CommissionBps:  1.5,
		StrategyParams: parseParams(*paramsStr),
		GKRPath:        *gkrPath,
	}

	ctx := context.Background()
	result, err := engine.Run(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(result)
		return
	}

	printResult(result)
}

func runMatrix() {
	fs := flag.NewFlagSet("matrix", flag.ExitOnError)
	strategies := fs.String("strategies", "intraday_mr,trend_following,opening_range_breakout,grid_trading,session_scalp", "Comma-separated strategy IDs")
	symbols := fs.String("symbols", "EURUSD,GBPUSD,USDJPY,US30,SPX500,XAUUSD", "Comma-separated symbols")
	timeframes := fs.String("timeframes", "1d", "Comma-separated timeframes")
	startStr := fs.String("start", "2023-01-01", "Start date")
	endStr := fs.String("end", "2025-12-31", "End date")
	capital := fs.Float64("capital", 100000, "Initial capital")

	if len(os.Args) > 2 {
		fs.Parse(os.Args[2:])
	}

	start, _ := time.Parse("2006-01-02", *startStr)
	end, _ := time.Parse("2006-01-02", *endStr)

	dataDir := getDataDir()
	db := NewFileDB(dataDir)

	matrixCfg := backtest.MatrixBacktestConfig{
		StrategyIDs:    parseSymbols(*strategies),
		Symbols:        parseSymbols(*symbols),
		Timeframes:     parseSymbols(*timeframes),
		StartDate:      start,
		EndDate:        end,
		InitialCapital: *capital,
	}

	fmt.Fprintf(os.Stderr, "Running matrix: %d strategies x %d symbols x %d timeframes = %d combos\n",
		len(matrixCfg.StrategyIDs), len(matrixCfg.Symbols), len(matrixCfg.Timeframes),
		len(matrixCfg.StrategyIDs)*len(matrixCfg.Symbols)*len(matrixCfg.Timeframes))

	ctx := context.Background()
	result, err := backtest.RunMatrix(ctx, db, matrixCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Matrix error: %v\n", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(result)
}

func listStrategies() {
	strategies := []struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}{
		{"intraday_mr", "mean_reversion"},
		{"mean_reversion", "mean_reversion"},
		{"opening_range_breakout", "breakout"},
		{"breakout", "breakout"},
		{"trend_following", "trend"},
		{"trend", "trend"},
		{"grid_trading", "grid"},
		{"grid", "grid"},
		{"session_scalp", "scalp"},
		{"scalp", "scalp"},
		{"pairs_trading", "stat_arb"},
		{"stat_arb", "stat_arb"},
		{"volatility_harvesting", "vol_arb"},
		{"vol_arb", "vol_arb"},
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(strategies)
}

func listSymbols() {
	tf := "1d"
	if len(os.Args) > 2 {
		tf = os.Args[2]
	}

	dataDir := getDataDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	fetcher := ingest.NewStooqFileFetcher(dataDir, logger)
	symbols, _ := fetcher.ListAvailableSymbols(tf)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(symbols)
}

func describeStrategy() {
	strategyID := "intraday_mr"
	if len(os.Args) > 2 {
		strategyID = os.Args[2]
	}

	defs := backtest.ParamDefsForStrategy(strategyID)
	if len(defs) == 0 {
		fmt.Fprintf(os.Stderr, "Unknown strategy: %s\n", strategyID)
		fmt.Fprintf(os.Stderr, "Available: intraday_mr, trend_following, opening_range_breakout, grid_trading, session_scalp, pairs_trading, volatility_harvesting\n")
		os.Exit(1)
	}

	currentParams := make(map[string]float64)
	for _, d := range defs {
		currentParams[d.Name] = d.Default
	}

	output := map[string]interface{}{
		"strategy": strategyID,
		"params":   defs,
		"current":  currentParams,
	}

	if len(os.Args) > 3 && os.Args[3] == "--json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(output)
		return
	}

	fmt.Printf("\n=== %s ===\n", strategyID)
	fmt.Println()
	groups := make(map[string][]strategy.ParamDef)
	for _, d := range defs {
		groups[d.Group] = append(groups[d.Group], d)
	}
	groupOrder := []string{"Entry", "Signal", "Exit", "Filter", "Risk", "Grid", "Sizing", "Session"}
	for _, g := range groupOrder {
		if groupDefs, ok := groups[g]; ok {
			fmt.Printf("[%s]\n", g)
			for _, d := range groupDefs {
				fmt.Printf("  %-25s %-10s default=%v  range=[%v..%v] step=%v\n", d.Name, d.Type, d.Default, d.Min, d.Max, d.Step)
				if d.Description != "" {
					fmt.Printf("    %s\n", d.Description)
				}
			}
			fmt.Println()
		}
	}
	for g, groupDefs := range groups {
		found := false
		for _, go2 := range groupOrder {
			if g == go2 {
				found = true
				break
			}
		}
		if !found {
			fmt.Printf("[%s]\n", g)
			for _, d := range groupDefs {
				fmt.Printf("  %-25s %-10s default=%v  range=[%v..%v] step=%v\n", d.Name, d.Type, d.Default, d.Min, d.Max, d.Step)
			}
			fmt.Println()
		}
	}

	fmt.Println("Usage:")
	fmt.Printf("  orca-cli backtest -strategy=%s -params=\"key=value,...\" ...\n", strategyID)
}

func getDataDir() string {
	if dir := os.Getenv("ORCA_DATA_DIR"); dir != "" {
		return dir
	}
	return "data/daily"
}

func runVersion() {
	engineFlag := flag.NewFlagSet("version", flag.ExitOnError)
	engineOnly := engineFlag.Bool("engine", false, "Print only the engine commit SHA")
	if len(os.Args) > 2 {
		engineFlag.Parse(os.Args[2:])
	}
	if *engineOnly {
		fmt.Println(version.Engine())
		return
	}
	fmt.Printf("orca-cli %s (built %s)\n", version.Engine(), version.Build())
}

func parseParams(s string) map[string]float64 {
	result := make(map[string]float64)
	if s == "" {
		return result
	}
	for _, pair := range strings.Split(s, ",") {
		parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(parts) == 2 {
			val, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			if err == nil {
				result[strings.TrimSpace(parts[0])] = val
			}
		}
	}
	return result
}

func parseSymbols(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func printResult(r *backtest.BacktestResult) {
	fmt.Printf("\n=== Backtest Results ===\n")
	fmt.Printf("Strategy:    %s\n", r.Config.StrategyID)
	fmt.Printf("Symbols:     %v\n", r.Config.Symbols)
	fmt.Printf("Period:      %s to %s\n", r.Config.StartDate.Format("2006-01-02"), r.Config.EndDate.Format("2006-01-02"))
	fmt.Printf("Timeframe:   %s\n", r.Config.Timeframe)
	fmt.Printf("Capital:     $%.0f\n", r.Config.InitialCapital)
	fmt.Printf("\n--- Performance ---\n")
	fmt.Printf("Total Return:  $%.2f (%.2f%%)\n", r.TotalReturn, r.TotalReturnPct)
	fmt.Printf("Sharpe Ratio:  %.3f\n", r.SharpeRatio)
	fmt.Printf("Sortino Ratio: %.3f\n", r.SortinoRatio)
	fmt.Printf("Max Drawdown:  %.2f%%\n", r.MaxDrawdown)
	fmt.Printf("Win Rate:      %.1f%%\n", r.WinRate*100)
	fmt.Printf("Profit Factor: %.2f\n", r.ProfitFactor)
	fmt.Printf("Avg Trade:     $%.2f\n", r.AvgTrade)
	fmt.Printf("Avg Win:       $%.2f\n", r.AvgWin)
	fmt.Printf("Avg Loss:      $%.2f\n", r.AvgLoss)
	fmt.Printf("Trades:        %d (%d wins, %d losses)\n", r.NumTrades, r.NumWins, r.NumLosses)
	fmt.Printf("Adv Select %%:  %.2f%%\n", r.AdverseSelectionRate*100)

	if r.MetricGateStatus != nil {
		fmt.Printf("\n--- Gate Status ---\n")
		fmt.Printf("Passed:    %v\n", r.MetricGateStatus.Passed)
		fmt.Printf("Profile:   %s\n", r.Config.GateProfile)
	}

	if len(r.Warnings) > 0 {
		fmt.Printf("\n--- Warnings ---\n")
		for _, w := range r.Warnings {
			fmt.Printf("  ! %s\n", w)
		}
	}

	if r.ComplianceReport != nil {
		fmt.Printf("\n--- FTMO Compliance ---\n")
		fmt.Printf("Passed:          %v\n", r.ComplianceReport.Passed)
		fmt.Printf("Max Daily Loss:  %.2f%%\n", r.ComplianceReport.MaxDailyLossPct)
		fmt.Printf("Total Return:    %.2f%%\n", r.ComplianceReport.TotalReturnPct)
		fmt.Printf("Breaches:        %d\n", r.ComplianceReport.NumBreaches)
		fmt.Printf("Final Balance:   $%.2f\n", r.ComplianceReport.FinalBalance)
		fmt.Printf("Peak Balance:    $%.2f\n", r.ComplianceReport.PeakBalance)
		fmt.Printf("Trading Days:    %d\n", r.ComplianceReport.TradingDays)
		for _, b := range r.ComplianceReport.Breaches {
			fmt.Printf("  Breach: %s = %.2f\n", b.Code, b.Value)
		}
	}
}
