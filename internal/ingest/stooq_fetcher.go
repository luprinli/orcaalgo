package ingest

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/lee-econ/orca-core/internal/types"
)

type StooqFileFetcher struct {
	dataDir string
	logger  *slog.Logger
}

func NewStooqFileFetcher(dataDir string, logger *slog.Logger) *StooqFileFetcher {
	return &StooqFileFetcher{dataDir: dataDir, logger: logger}
}

func (f *StooqFileFetcher) Name() string { return "stooq" }

func (f *StooqFileFetcher) FetchCandles(ctx context.Context, ticker string, start, end time.Time, timeframe string) ([]CandleData, error) {
	path, err := f.resolvePath(ticker, timeframe)
	if err != nil {
		return nil, err
	}
	startDefault := time.Time{}
	if start.Equal(startDefault) || start.IsZero() {
		start = time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	endDefault := time.Time{}
	if end.Equal(endDefault) || end.IsZero() {
		end = time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	return f.readCSV(path, start, end, ticker)
}

func (f *StooqFileFetcher) FetchDailyMetrics(ctx context.Context, ticker string) (*SymbolMetrics, error) {
	return ComputeDailyMetrics(ctx, f, ticker)
}

func (f *StooqFileFetcher) resolvePath(ticker string, timeframe string) (string, error) {
	ticker = strings.ToUpper(ticker)
	base := "world"

	var subdir string
	fileName := strings.ToLower(ticker) + ".txt"

	switch {
	case ticker == "EURUSD" || ticker == "GBPUSD" || ticker == "USDJPY" || ticker == "USDCHF" ||
		ticker == "AUDUSD" || ticker == "USDCAD" || ticker == "NZDUSD" ||
		ticker == "EURGBP" || ticker == "EURJPY" || ticker == "EURCHF" ||
		ticker == "GBPJPY" || ticker == "GBPCHF" || ticker == "CHFJPY" ||
		ticker == "AUDJPY" || ticker == "AUDNZD" || ticker == "AUDCHF" ||
		ticker == "CADJPY" || ticker == "NZDJPY" || ticker == "NZDCHF" ||
		ticker == "EURAUD" || ticker == "EURCAD" || ticker == "EURNZD" ||
		ticker == "GBPAUD" || ticker == "GBPCAD" || ticker == "GBPNZD":
		subdir = "currencies/major"

	case ticker == "US30" || ticker == "^DJI":
		fileName = "^dji.txt"
		subdir = "indices"
	case ticker == "SPX500" || ticker == "^SPX" || ticker == "^GSPC":
		fileName = "^spx.txt"
		subdir = "indices"
	case ticker == "NAS100" || ticker == "^NDQ" || ticker == "^IXIC":
		fileName = "^ndq.txt"
		subdir = "indices"
	case ticker == "UK100" || ticker == "^FTSE":
		fileName = "^ukx.txt"
		subdir = "indices"
	case ticker == "GER40" || ticker == "^DAX" || ticker == "^GDAXI":
		fileName = "^dax.txt"
		subdir = "indices"
	case ticker == "JPN225" || ticker == "^NKX" || ticker == "^N225":
		fileName = "^nkx.txt"
		subdir = "indices"

	case ticker == "BTCUSD" || ticker == "BTC":
		fileName = "btc.v.txt"
		subdir = "cryptocurrencies"
	case ticker == "ETHUSD" || ticker == "ETH":
		fileName = "eth.v.txt"
		subdir = "cryptocurrencies"

	case strings.HasPrefix(ticker, "^") && !strings.Contains(ticker, "XAU") && !strings.Contains(ticker, "XAG"):
		fileName = strings.ToLower(ticker) + ".txt"
		subdir = "indices"

	case ticker == "XAUUSD":
		fileName = "xauusd.txt"
		subdir = "currencies/other"
	case ticker == "XAGUSD":
		fileName = "xagusd.txt"
		subdir = "currencies/other"
	case ticker == "USOIL":
		fileName = "usoil.txt"
		subdir = "currencies/other"
	case ticker == "UKOIL":
		fileName = "ukoil.txt"
		subdir = "currencies/other"

	default:
		subdir = "currencies/other"
	}

	var path string
	baseParent := filepath.Dir(f.dataDir)
	subParts := strings.Split(subdir, "/")

	switch timeframe {
	case "5", "5m":
		parts := append([]string{baseParent, "5 min", base}, subParts...)
		parts = append(parts, fileName)
		path = filepath.Join(parts...)
	case "60", "60m", "1h":
		parts := append([]string{baseParent, "hourly", base}, subParts...)
		parts = append(parts, fileName)
		path = filepath.Join(parts...)
	default:
		parts := append([]string{f.dataDir, base}, subParts...)
		parts = append(parts, fileName)
		path = filepath.Join(parts...)
	}
	path = filepath.Clean(path)
	return path, nil
}

func (f *StooqFileFetcher) readCSV(path string, start, end time.Time, ticker string) ([]CandleData, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("stooq open %s: %w", path, err)
	}
	defer file.Close()

	var candles []CandleData
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if lineNum == 1 {
			continue
		}

		parts := strings.SplitN(line, ",", 10)
		if len(parts) < 8 {
			continue
		}

		dateStr := strings.TrimSpace(parts[2])
		timeStr := strings.TrimSpace(parts[3])

		if len(dateStr) != 8 {
			continue
		}
		year, err := strconv.Atoi(dateStr[0:4])
		if err != nil {
			log.Printf("stooq: bad year in row %d: %q", lineNum, dateStr)
			continue
		}
		month, err := strconv.Atoi(dateStr[4:6])
		if err != nil {
			log.Printf("stooq: bad month in row %d: %q", lineNum, dateStr)
			continue
		}
		day, err := strconv.Atoi(dateStr[6:8])
		if err != nil {
			log.Printf("stooq: bad day in row %d: %q", lineNum, dateStr)
			continue
		}

		hour, minute := 0, 0
		if len(timeStr) >= 4 {
			hour, err = strconv.Atoi(timeStr[0:2])
			if err != nil {
				log.Printf("stooq: bad hour in row %d: %q", lineNum, timeStr)
				continue
			}
			minute, err = strconv.Atoi(timeStr[2:4])
			if err != nil {
				log.Printf("stooq: bad minute in row %d: %q", lineNum, timeStr)
				continue
			}
		}

		t := time.Date(year, time.Month(month), day, hour, minute, 0, 0, time.UTC)
		if t.Before(start) || t.After(end) {
			continue
		}

		open, err := strconv.ParseFloat(strings.TrimSpace(parts[4]), 64)
		if err != nil {
			log.Printf("stooq: bad open in row %d: %q", lineNum, parts[4])
			continue
		}
		high, err := strconv.ParseFloat(strings.TrimSpace(parts[5]), 64)
		if err != nil {
			log.Printf("stooq: bad high in row %d: %q", lineNum, parts[5])
			continue
		}
		low, err := strconv.ParseFloat(strings.TrimSpace(parts[6]), 64)
		if err != nil {
			log.Printf("stooq: bad low in row %d: %q", lineNum, parts[6])
			continue
		}
		close_, err := strconv.ParseFloat(strings.TrimSpace(parts[7]), 64)
		if err != nil {
			log.Printf("stooq: bad close in row %d: %q", lineNum, parts[7])
			continue
		}
		volume, err := strconv.ParseFloat(strings.TrimSpace(parts[8]), 64)
		if err != nil {
			log.Printf("stooq: bad volume in row %d: %q", lineNum, parts[8])
			continue
		}

		if close_ <= 0 {
			continue
		}
		if open <= 0 {
			open = close_
		}
		if high <= 0 {
			high = close_
		}
		if low <= 0 {
			low = close_
		}

		candles = append(candles, CandleData{
			Time:   t,
			Open:   types.FromFloat64(open),
			High:   types.FromFloat64(high),
			Low:    types.FromFloat64(low),
			Close:  types.FromFloat64(close_),
			Volume: volume,
		})
	}

	if err := scanner.Err(); err != nil {
		return candles, fmt.Errorf("stooq scan: %w", err)
	}

	return candles, nil
}

func (f *StooqFileFetcher) ListAvailableSymbols(timeframe string) ([]string, error) {
	var symbols []string

	mapping := map[string]string{
		"eurusd": "EURUSD", "gbpusd": "GBPUSD", "usdjpy": "USDJPY",
		"usdchf": "USDCHF", "audusd": "AUDUSD", "usdcad": "USDCAD", "nzdusd": "NZDUSD",
		"^dji": "US30", "^spx": "SPX500", "^ndq": "NAS100",
		"^dax": "GER40", "^nkx": "JPN225", "^ukx": "UK100",
		"btc.v": "BTCUSD", "eth.v": "ETHUSD",
		"xauusd": "XAUUSD", "xagusd": "XAGUSD",
	}

	for stooqName, orcaName := range mapping {
		path, err := f.resolvePath(orcaName, timeframe)
		if err != nil {
			continue
		}
		if _, statErr := os.Stat(path); statErr == nil {
			symbols = append(symbols, orcaName)
		}
		_ = stooqName
	}

	return symbols, nil
}
