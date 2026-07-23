package ingest

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type BackfillOrchestrator struct {
	fetcher    DataFetcher
	tickWriter *TickerWriter
	logger     *slog.Logger
}

func NewBackfillOrchestrator(fetcher DataFetcher, tickWriter *TickerWriter, logger *slog.Logger) *BackfillOrchestrator {
	return &BackfillOrchestrator{
		fetcher:    fetcher,
		tickWriter: tickWriter,
		logger:     logger,
	}
}

func (b *BackfillOrchestrator) BackfillSymbol(ctx context.Context, ticker string, lookbackYears int) error {
	b.logger.InfoContext(ctx, "backfill_started", "symbol", ticker, "lookback_years", lookbackYears)

	end := time.Now()
	start := end.AddDate(-lookbackYears, 0, 0)

	candles, err := b.fetcher.FetchCandles(ctx, ticker, start, end, "1d")
	if err != nil {
		return fmt.Errorf("fetch candles for %s: %w", ticker, err)
	}

	b.logger.InfoContext(ctx, "backfill_fetched", "symbol", ticker, "candles", len(candles))

	_ = candles

	return nil
}
