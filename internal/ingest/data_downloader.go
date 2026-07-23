package ingest

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const PRICE_SCALE_I = 100_000

type DataDownloader struct {
	pool    *pgxpool.Pool
	fetcher DataFetcher
	logger  *slog.Logger
}

func NewDataDownloader(pool *pgxpool.Pool, fetcher DataFetcher, logger *slog.Logger) *DataDownloader {
	return &DataDownloader{pool: pool, fetcher: fetcher, logger: logger}
}

func (d *DataDownloader) DownloadSymbol(ctx context.Context, ticker string, start, end time.Time, timeframe string) (*DownloadResult, error) {
	d.logger.InfoContext(ctx, "download_start", "ticker", ticker, "start", start.Format("2006-01-02"), "end", end.Format("2006-01-02"), "timeframe", timeframe)

	candles, err := d.fetcher.FetchCandles(ctx, ticker, start, end, timeframe)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", ticker, err)
	}
	if len(candles) == 0 {
		return &DownloadResult{Ticker: ticker, Downloaded: 0}, nil
	}

	validation := ValidateCandles(candles, ticker, timeframe)
	if !validation.Valid && validation.NullRows > len(candles)/2 {
		d.logger.WarnContext(ctx, "download_validation_failed", "ticker", ticker, "message", validation.Message)
		return &DownloadResult{Ticker: ticker, Validation: validation}, nil
	}

	stored, skipped, err := d.upsertCandles(ctx, ticker, timeframe, candles)
	if err != nil {
		return &DownloadResult{Ticker: ticker, Validation: validation, Error: err}, err
	}

	d.logger.InfoContext(ctx, "download_complete",
		"ticker", ticker,
		"timeframe", timeframe,
		"fetched", len(candles),
		"stored", stored,
		"skipped", skipped,
	)

	return &DownloadResult{
		Ticker:     ticker,
		Fetched:    len(candles),
		Stored:     stored,
		Skipped:    skipped,
		Validation: validation,
	}, nil
}

func (d *DataDownloader) DownloadSymbols(ctx context.Context, tickers []string, start, end time.Time, timeframe string) ([]*DownloadResult, error) {
	results := make([]*DownloadResult, 0, len(tickers))
	var lastErr error
	for _, ticker := range tickers {
		result, err := d.DownloadSymbol(ctx, ticker, start, end, timeframe)
		if err != nil {
			lastErr = err
			d.logger.ErrorContext(ctx, "download_symbol_failed", "ticker", ticker, "error", err)
			if result == nil {
				result = &DownloadResult{Ticker: ticker, Error: err}
			}
		}
		results = append(results, result)
	}
	return results, lastErr
}

func (d *DataDownloader) upsertCandles(ctx context.Context, ticker, timeframe string, candles []CandleData) (stored int, skipped int, err error) {
	for _, c := range candles {
		openRaw := int64(c.Open * PRICE_SCALE_I)
		highRaw := int64(c.High * PRICE_SCALE_I)
		lowRaw := int64(c.Low * PRICE_SCALE_I)
		closeRaw := int64(c.Close * PRICE_SCALE_I)
		volume := int64(c.Volume)

		tag, execErr := d.pool.Exec(ctx,
			`INSERT INTO candles (time, symbol_id, timeframe, open_raw, high_raw, low_raw, close_raw, volume, source)
			 SELECT $1, COALESCE((SELECT id FROM symbols WHERE ticker=$2 LIMIT 1), 1), $3, $4, $5, $6, $7, $8, 'api'
			 ON CONFLICT (symbol_id, timeframe, time) DO UPDATE SET
			   open_raw=EXCLUDED.open_raw, high_raw=EXCLUDED.high_raw,
			   low_raw=EXCLUDED.low_raw, close_raw=EXCLUDED.close_raw,
			   volume=EXCLUDED.volume
			 WHERE EXISTS (SELECT 1 FROM symbols WHERE ticker=$2)`,
			c.Time, ticker, timeframe, openRaw, highRaw, lowRaw, closeRaw, volume,
		)
		if execErr != nil {
			skipped++
			continue
		}
		if tag.RowsAffected() > 0 {
			stored++
		} else {
			skipped++
		}
	}
	return stored, skipped, nil
}

type DownloadResult struct {
	Ticker     string
	Fetched    int
	Stored     int
	Skipped    int
	Downloaded int
	Error      error
	Validation *ValidationResult
}

func (r *DownloadResult) Summary() string {
	if r.Downloaded > 0 {
		r.Fetched = r.Downloaded
	}
	if r.Error != nil {
		return fmt.Sprintf("%s: ERROR %v", r.Ticker, r.Error)
	}
	if r.Validation != nil && !r.Validation.Valid {
		return fmt.Sprintf("%s: %d fetched, %d stored, %d skipped — %s", r.Ticker, r.Fetched, r.Stored, r.Skipped, r.Validation.Message)
	}
	return fmt.Sprintf("%s: %d fetched, %d stored, %d skipped", r.Ticker, r.Fetched, r.Stored, r.Skipped)
}
