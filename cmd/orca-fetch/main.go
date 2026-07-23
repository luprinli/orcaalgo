package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lee-econ/orca-core/internal/db"
	"github.com/lee-econ/orca-core/internal/ingest"
)

func main() {
	symbolsFlag := flag.String("symbols", "", "Comma-separated tickers (e.g., SPY,AAPL,EURUSD)")
	universeFlag := flag.Bool("universe", false, "Download/import all symbols from OrcaAlgo universe")
	startFlag := flag.String("start", "", "Start date (YYYY-MM-DD), default: 2 years ago")
	endFlag := flag.String("end", "", "End date (YYYY-MM-DD), default: today")
	sourceFlag := flag.String("source", "stooq", "Data source: stooq, tiingo, yahoo, chain")
	formatFlag := flag.String("format", "db", "Output: db (TimescaleDB), csv")
	timeframeFlag := flag.String("timeframe", "1d", "Timeframe: 1d, 5m")
	dataDirFlag := flag.String("data-dir", "", "Stooq data directory (auto-detected)")
	apiKeyFlag := flag.String("tiingo-key", "", "Tiingo API key (or set TIINGO_API_KEY env var)")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if *symbolsFlag == "" && !*universeFlag {
		fmt.Fprintln(os.Stderr, "Usage: orca-fetch --symbols=SPY,EURUSD or --universe")
		fmt.Fprintln(os.Stderr, "  Sources: stooq (local files), tiingo (API), yahoo (free API), chain (tiingo→yahoo)")
		fmt.Fprintln(os.Stderr, "  Timeframes: 1d (daily), 5m (5-minute intraday)")
		flag.PrintDefaults()
		os.Exit(1)
	}

	var tickers []string
	if *universeFlag {
		if *sourceFlag == "stooq" {
			tickers = ingest.MapOrcaToStooqSymbols()
		} else {
			tickers = defaultUniverseSymbols()
		}
	} else {
		for _, t := range strings.Split(*symbolsFlag, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tickers = append(tickers, t)
			}
		}
	}

	now := time.Now()
	start, err := time.Parse("2006-01-02", *startFlag)
	if err != nil {
		start = now.AddDate(-2, 0, 0)
	}
	end, err := time.Parse("2006-01-02", *endFlag)
	if err != nil {
		end = now
	}

	if *sourceFlag == "stooq" {
		importStooq(tickers, *timeframeFlag, *dataDirFlag, *formatFlag, logger, start, end)
		return
	}

	var fetcher ingest.DataFetcher
	switch *sourceFlag {
	case "tiingo":
		key := *apiKeyFlag
		if key == "" {
			key = os.Getenv("TIINGO_API_KEY")
		}
		fetcher = ingest.NewTiingoDataFetcherWithKey(key, logger)
	case "yahoo":
		fetcher = ingest.NewYahooDataFetcher(logger)
	case "chain":
		tf := ingest.NewTiingoDataFetcherWithKey(os.Getenv("TIINGO_API_KEY"), logger)
		yf := ingest.NewYahooDataFetcher(logger)
		fetcher = ingest.NewFetcherChain([]ingest.DataFetcher{tf, yf}, logger)
	default:
		fmt.Fprintf(os.Stderr, "Unknown source: %s (use stooq, tiingo, yahoo, or chain)\n", *sourceFlag)
		os.Exit(1)
	}

	if *formatFlag == "csv" {
		exportCSV(fetcher, tickers, start, end, *timeframeFlag, logger)
		return
	}

	dbCfg := db.DefaultConfig()
	repo, err := db.NewRepository(dbCfg)
	if err != nil {
		logger.Error("db_connect_failed", "error", err)
		os.Exit(1)
	}
	defer repo.Close()

	downloader := ingest.NewDataDownloader(repo.Pool(), fetcher, logger)

	fmt.Printf("Downloading %d symbols from %s to %s [%s] using %s...\n",
		len(tickers), start.Format("2006-01-02"), end.Format("2006-01-02"), *timeframeFlag, fetcher.Name())

	results, _ := downloader.DownloadSymbols(context.Background(), tickers, start, end, *timeframeFlag)

	totalStored := 0
	totalFetched := 0
	for _, r := range results {
		fmt.Println(r.Summary())
		totalFetched += r.Fetched
		totalStored += r.Stored
	}
	fmt.Printf("\nTotal: %d candles fetched, %d stored [%s]\n", totalFetched, totalStored, *timeframeFlag)
}

func importStooq(tickers []string, timeframe, dataDir, format string, logger *slog.Logger, start, end time.Time) {
	if dataDir == "" {
		dataDir = ingest.FindDataDirectory()
	}
	logger.Info("stooq_import", "data_dir", dataDir, "timeframe", timeframe, "symbols", len(tickers))

	fetcher := ingest.NewStooqFileFetcher(dataDir, logger)

	if format == "csv" {
		fmt.Println("Date,Open,High,Low,Close,Volume,Symbol")
		for _, ticker := range tickers {
			candles, err := fetcher.FetchCandles(context.Background(), ticker, start, end, timeframe)
			if err != nil {
				continue
			}
			for _, c := range candles {
				fmt.Printf("%s,%.4f,%.4f,%.4f,%.4f,%.0f,%s\n",
					c.Time.Format("2006-01-02 15:04"), c.Open, c.High, c.Low, c.Close, c.Volume, ticker)
			}
		}
		return
	}

	dbCfg := db.DefaultConfig()
	if portEnv := os.Getenv("ORCA_DB_PORT"); portEnv != "" {
		if p, err := strconv.Atoi(portEnv); err == nil {
			dbCfg.Port = p
		}
	}
	if os.Getenv("ORCA_DB_PORT") != "" {
		fmt.Fprintf(os.Stderr, "DB connecting to %s:%s@%s:%s/%s\n",
			os.Getenv("ORCA_DB_USER"), "***", os.Getenv("ORCA_DB_HOST"), os.Getenv("ORCA_DB_PORT"), os.Getenv("ORCA_DB_NAME"))
	}
	repo, err := db.NewRepository(dbCfg)
	if err != nil {
		logger.Error("db_connect_failed", "error", err,
			"hint", "Set ORCA_DB_PASSWORD env var if needed")
		os.Exit(1)
	}
	defer repo.Close()
	fmt.Fprintf(os.Stderr, "DB connected OK\n")

	importer := ingest.NewStooqImporter(repo.Pool(), fetcher, logger)

	if err := importer.EnsureSchema(context.Background()); err != nil {
		logger.Warn("ensure_schema_failed", "error", err)
	}

	fmt.Printf("Importing %d symbols from Stooq [%s] to TimescaleDB...\n", len(tickers), timeframe)

	var results []*ingest.ImportResult
	for _, ticker := range tickers {
		result, err := importer.ImportSymbol(context.Background(), ticker, timeframe)
		if err != nil {
			logger.Error("stooq_import_error", "ticker", ticker, "error", err)
			if result == nil {
				result = &ingest.ImportResult{Ticker: ticker, Timeframe: timeframe, Error: err}
			}
		}
		results = append(results, result)
		fmt.Println(result.Summary())
	}

	totalStored := 0
	totalFetched := 0
	for _, r := range results {
		totalFetched += r.Fetched
		totalStored += r.Stored
	}
	fmt.Printf("\nTotal: %d candles fetched, %d stored [%s]\n", totalFetched, totalStored, timeframe)
	fmt.Println("Backtest ready: POST /api/v1/backtests with these symbols + start/end dates")

	counts, _ := repo.CountTable(context.Background(), "candles")
	fmt.Printf("Total candles in DB: %d\n", counts)
}

func exportCSV(fetcher ingest.DataFetcher, tickers []string, start, end time.Time, timeframe string, logger *slog.Logger) {
	fmt.Println("Date,Open,High,Low,Close,Volume,Symbol")
	for _, ticker := range tickers {
		candles, err := fetcher.FetchCandles(context.Background(), ticker, start, end, timeframe)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR %s: %v\n", ticker, err)
			continue
		}
		for _, c := range candles {
			fmt.Printf("%s,%.4f,%.4f,%.4f,%.4f,%.0f,%s\n",
				c.Time.Format("2006-01-02 15:04"), c.Open, c.High, c.Low, c.Close, c.Volume, ticker)
		}
	}
}

func defaultUniverseSymbols() []string {
	return []string{
		"EURUSD", "GBPUSD", "USDJPY", "USDCHF", "AUDUSD", "USDCAD", "NZDUSD",
		"US30", "SPX500", "NAS100", "UK100", "GER40", "JPN225",
		"XAUUSD", "XAGUSD", "USOIL", "UKOIL",
		"BTCUSD", "ETHUSD",
	}
}
