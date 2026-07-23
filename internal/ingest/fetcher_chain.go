package ingest

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type FetcherChain struct {
	fetchers []DataFetcher
	logger   *slog.Logger
}

func NewFetcherChain(fetchers []DataFetcher, logger *slog.Logger) *FetcherChain {
	return &FetcherChain{fetchers: fetchers, logger: logger}
}

func (c *FetcherChain) AddFetcher(f DataFetcher) {
	c.fetchers = append(c.fetchers, f)
}

func (c *FetcherChain) FetchCandles(ctx context.Context, ticker string, start, end time.Time, timeframe string) ([]CandleData, error) {
	var lastErr error
	for _, f := range c.fetchers {
		candles, err := f.FetchCandles(ctx, ticker, start, end, timeframe)
		if err == nil && len(candles) > 0 {
			return candles, nil
		}
		if err != nil {
			lastErr = err
			c.logger.WarnContext(ctx, "fetcher_failed",
				"fetcher", f.Name(),
				"ticker", ticker,
				"error", err,
			)
		} else {
			c.logger.WarnContext(ctx, "fetcher_empty",
				"fetcher", f.Name(),
				"ticker", ticker,
			)
		}
	}
	if lastErr != nil {
		return nil, fmt.Errorf("all fetchers exhausted for %s: %w", ticker, lastErr)
	}
	return nil, fmt.Errorf("all fetchers exhausted for %s: no data returned", ticker)
}

func (c *FetcherChain) FetchDailyMetrics(ctx context.Context, ticker string) (*SymbolMetrics, error) {
	var lastErr error
	for _, f := range c.fetchers {
		metrics, err := f.FetchDailyMetrics(ctx, ticker)
		if err == nil && metrics != nil {
			return metrics, nil
		}
		if err != nil {
			lastErr = err
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("all fetchers exhausted for metrics %s", ticker)
}

func (c *FetcherChain) Name() string { return "chain" }
