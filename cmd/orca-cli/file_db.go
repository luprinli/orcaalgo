package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/lee-econ/orca-core/internal/backtest"
	"github.com/lee-econ/orca-core/internal/ingest"
)

type FileDB struct {
	fetcher *ingest.StooqFileFetcher
	logger  *slog.Logger
}

func NewFileDB(dataDir string) *FileDB {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	return &FileDB{
		fetcher: ingest.NewStooqFileFetcher(dataDir, logger),
		logger:  logger,
	}
}

func (db *FileDB) LoadCandles(ctx context.Context, symbols []string, start, end time.Time) ([][]backtest.Candle, error) {
	return db.loadCandlesWithTF(ctx, symbols, start, end, "1d")
}

func (db *FileDB) LoadCandlesFiltered(ctx context.Context, symbols []string, start, end time.Time, source string) ([][]backtest.Candle, error) {
	return db.loadCandlesWithTF(ctx, symbols, start, end, "1d")
}

func (db *FileDB) LoadCandlesTF(ctx context.Context, symbols []string, start, end time.Time, timeframe string) ([][]backtest.Candle, error) {
	return db.loadCandlesWithTF(ctx, symbols, start, end, timeframe)
}

func (db *FileDB) LoadCandlesTFFiltered(ctx context.Context, symbols []string, start, end time.Time, timeframe string, source string) ([][]backtest.Candle, error) {
	return db.loadCandlesWithTF(ctx, symbols, start, end, timeframe)
}

func (db *FileDB) loadCandlesWithTF(ctx context.Context, symbols []string, start, end time.Time, timeframe string) ([][]backtest.Candle, error) {
	result := make([][]backtest.Candle, len(symbols))
	for i, sym := range symbols {
		candles, err := db.fetcher.FetchCandles(ctx, sym, start, end, timeframe)
		if err != nil {
			db.logger.Warn("file fetch failed, using empty", "symbol", sym, "err", err)
			result[i] = []backtest.Candle{}
			continue
		}
		btCandles := make([]backtest.Candle, len(candles))
		for j, c := range candles {
			btCandles[j] = backtest.Candle{
				Time:   c.Time,
				Open:   c.Open,
				High:   c.High,
				Low:    c.Low,
				Close:  c.Close,
				Volume: c.Volume,
				Symbol: sym,
			}
		}
		result[i] = btCandles
	}
	return result, nil
}

func (db *FileDB) LoadAllCandles(ctx context.Context, symbols []string, start, end time.Time, timeframe string) (map[string][]backtest.Candle, error) {
	result := make(map[string][]backtest.Candle)
	for _, sym := range symbols {
		candles, err := db.fetcher.FetchCandles(ctx, sym, start, end, timeframe)
		if err != nil {
			db.logger.Warn("file fetch failed", "symbol", sym, "err", err)
			result[sym] = []backtest.Candle{}
			continue
		}
		btCandles := make([]backtest.Candle, len(candles))
		for j, c := range candles {
			btCandles[j] = backtest.Candle{
				Time:   c.Time,
				Open:   c.Open,
				High:   c.High,
				Low:    c.Low,
				Close:  c.Close,
				Volume: c.Volume,
				Symbol: sym,
			}
		}
		result[sym] = btCandles
	}
	return result, nil
}

func (db *FileDB) LoadRegimeLogs(ctx context.Context, start, end time.Time) ([]backtest.RegimeLog, error) {
	return nil, nil
}

func (db *FileDB) LoadVIXLogs(ctx context.Context, start, end time.Time) ([]backtest.VIXLog, error) {
	return nil, nil
}

func (db *FileDB) LoadSentimentLogs(ctx context.Context, start, end time.Time) ([]backtest.SentimentLog, error) {
	return nil, nil
}

func (db *FileDB) CountCandles(ctx context.Context) (int64, error) {
	return 0, nil
}

func (db *FileDB) CountSyntheticCandles(ctx context.Context) (int64, error) {
	return 0, nil
}

func (db *FileDB) CountRegimeLogs(ctx context.Context) (int64, error) {
	return 0, nil
}

func (db *FileDB) LoadUniverseSnapshots(ctx context.Context, start, end time.Time) ([]backtest.UniverseSnapshot, error) {
	return nil, nil
}
