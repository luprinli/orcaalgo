package ingest

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lee-econ/orca-core/internal/config"
)

type StooqImporter struct {
	pool    *pgxpool.Pool
	fetcher *StooqFileFetcher
	logger  *slog.Logger
}

func NewStooqImporter(pool *pgxpool.Pool, fetcher *StooqFileFetcher, logger *slog.Logger) *StooqImporter {
	return &StooqImporter{pool: pool, fetcher: fetcher, logger: logger}
}

func (s *StooqImporter) EnsureSchema(ctx context.Context) error {
	// Drop the legacy source-less unique constraint so each provenance retains
	// its own bars: source is part of the bar identity. The source-inclusive
	// constraint below replaces it (and supersedes the migration 000041 index).
	_, _ = s.pool.Exec(ctx,
		`ALTER TABLE candles DROP CONSTRAINT IF EXISTS candles_symbol_timeframe_time_unique`,
	)

	_, err := s.pool.Exec(ctx,
		`ALTER TABLE candles ADD CONSTRAINT IF NOT EXISTS candles_symbol_timeframe_time_source_unique UNIQUE (symbol_id, timeframe, time, source)`,
	)
	if err != nil {
		s.logger.WarnContext(ctx, "ensure_schema_constraint_failed", "error", err)
	}

	_, err = s.pool.Exec(ctx, `ALTER TABLE candles ADD COLUMN IF NOT EXISTS source VARCHAR(32) NOT NULL DEFAULT 'seed'`)
	if err != nil {
		s.logger.WarnContext(ctx, "ensure_schema_column_failed", "error", err)
	}

	_, err = s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS universe_config (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL,
			profile_id VARCHAR(32) NOT NULL DEFAULT 'default',
			asset_class_filters JSONB NOT NULL DEFAULT '{}',
			dynamic_triggers JSONB NOT NULL DEFAULT '{}',
			content_hash TEXT NOT NULL DEFAULT '',
			is_active BOOLEAN NOT NULL DEFAULT false,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS universe_state (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			snapshot_date DATE NOT NULL,
			symbol_ids INTEGER[] NOT NULL DEFAULT '{}',
			content_hash TEXT NOT NULL DEFAULT '',
			filters_used JSONB NOT NULL DEFAULT '{}',
			triggered_additions JSONB NOT NULL DEFAULT '[]',
			triggered_removals JSONB NOT NULL DEFAULT '[]',
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	if err != nil {
		s.logger.WarnContext(ctx, "ensure_schema_warning", "error", err)
	}
	return nil
}

func (s *StooqImporter) ImportSymbol(ctx context.Context, ticker string, timeframe string) (*ImportResult, error) {
	stooqTimeframe := timeframe
	dbTimeframe := timeframe

	switch timeframe {
	case "5", "5m":
		stooqTimeframe = "5"
		dbTimeframe = "5m"
	case "60", "1H", "1h":
		stooqTimeframe = "60"
		dbTimeframe = "1h"
	case "1d", "D":
		stooqTimeframe = "1d"
		dbTimeframe = "1d"
	}

	s.logger.InfoContext(ctx, "stooq_import_start", "ticker", ticker, "timeframe", dbTimeframe)

	symbolID, err := s.ensureSymbol(ctx, ticker)
	if err != nil {
		return nil, fmt.Errorf("ensure symbol %s: %w", ticker, err)
	}

	candles, err := s.fetcher.FetchCandles(ctx, ticker, time.Time{}, time.Now().AddDate(1, 0, 0), stooqTimeframe)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", ticker, err)
	}

	if len(candles) == 0 {
		return &ImportResult{Ticker: ticker, Timeframe: dbTimeframe, Fetched: 0}, nil
	}

	validation := ValidateCandles(candles, ticker, dbTimeframe)

	stored, skipped, err := s.bulkUpsert(ctx, symbolID, ticker, dbTimeframe, candles)
	if err != nil {
		return &ImportResult{Ticker: ticker, Timeframe: dbTimeframe, Validation: validation, Error: err}, err
	}

	s.logger.InfoContext(ctx, "stooq_import_complete",
		"ticker", ticker,
		"timeframe", dbTimeframe,
		"fetched", len(candles),
		"stored", stored,
		"skipped", skipped,
	)

	return &ImportResult{
		Ticker:     ticker,
		Timeframe:  dbTimeframe,
		Fetched:    len(candles),
		Stored:     stored,
		Skipped:    skipped,
		Validation: validation,
	}, nil
}

func (s *StooqImporter) ImportAll(ctx context.Context, timeframe string) ([]*ImportResult, error) {
	stooqTF := timeframe
	switch timeframe {
	case "5m", "5":
		stooqTF = "5"
	case "1h", "60", "1H":
		stooqTF = "60"
	default:
		stooqTF = "1d"
	}
	symbols, err := s.fetcher.ListAvailableSymbols(stooqTF)
	if err != nil {
		return nil, err
	}

	results := make([]*ImportResult, 0, len(symbols))
	for _, ticker := range symbols {
		result, importErr := s.ImportSymbol(ctx, ticker, timeframe)
		if importErr != nil {
			s.logger.ErrorContext(ctx, "stooq_import_failed", "ticker", ticker, "error", importErr)
			if result == nil {
				result = &ImportResult{Ticker: ticker, Timeframe: timeframe, Error: importErr}
			}
		}
		results = append(results, result)
	}
	return results, nil
}

func (s *StooqImporter) ensureSymbol(ctx context.Context, ticker string) (int32, error) {
	var id int32
	err := s.pool.QueryRow(ctx,
		`INSERT INTO symbols (ticker, exchange, asset_type, tick_size, lot_size, is_active)
		 VALUES ($1, 'STOOQ', $2, $3, 1, true)
		 ON CONFLICT (ticker, exchange) DO UPDATE SET is_active=true
		 RETURNING id`,
		ticker, assetTypeForTicker(ticker), tickSizeForTicker(ticker),
	).Scan(&id)
	return id, err
}

func (s *StooqImporter) bulkUpsert(ctx context.Context, symbolID int32, ticker, timeframe string, candles []CandleData) (stored, skipped int, err error) {
	if len(candles) == 0 {
		return 0, 0, nil
	}

	rows := make([][]interface{}, len(candles))
	for i, c := range candles {
		openRaw := c.Open.Int64()
		highRaw := c.High.Int64()
		lowRaw := c.Low.Int64()
		closeRaw := c.Close.Int64()
		volume := int64(c.Volume)

		rows[i] = []interface{}{
			c.Time,
			symbolID,
			timeframe,
			openRaw,
			highRaw,
			lowRaw,
			closeRaw,
			volume,
			"stooq",
		}
	}

	_, err = s.pool.CopyFrom(
		ctx,
		pgx.Identifier{"candles"},
		[]string{"time", "symbol_id", "timeframe", "open_raw", "high_raw", "low_raw", "close_raw", "volume", "source"},
		pgx.CopyFromRows(rows),
	)

	if err != nil {
		cleanCount := 0
		for _, c := range candles {
			if c.Time.IsZero() || c.Close.IsZero() {
				continue
			}
			cleanCount++
			openRaw := int64(c.Open * PRICE_SCALE_I)
			highRaw := int64(c.High * PRICE_SCALE_I)
			lowRaw := int64(c.Low * PRICE_SCALE_I)
			closeRaw := int64(c.Close * PRICE_SCALE_I)
			volume := int64(c.Volume)

			_, execErr := s.pool.Exec(ctx,
				`INSERT INTO candles (time, symbol_id, timeframe, open_raw, high_raw, low_raw, close_raw, volume, source)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'stooq')
			 ON CONFLICT (symbol_id, timeframe, time, source) DO UPDATE SET
			   open_raw=EXCLUDED.open_raw, high_raw=EXCLUDED.high_raw,
			   low_raw=EXCLUDED.low_raw, close_raw=EXCLUDED.close_raw,
			   volume=EXCLUDED.volume`,
				c.Time, symbolID, timeframe, openRaw, highRaw, lowRaw, closeRaw, volume,
			)
			if execErr == nil {
				stored++
			} else {
				skipped++
			}
		}
		s.logger.InfoContext(ctx, "bulk_insert_fallback", "ticker", ticker, "clean", cleanCount, "stored", stored, "skipped", skipped)
		return stored, skipped, nil
	}

	s.logger.InfoContext(ctx, "bulk_insert_ok", "ticker", ticker, "rows", len(candles))
	return len(candles), 0, nil
}

func assetTypeForTicker(ticker string) string {
	if s, ok := config.SymbolByTicker(ticker); ok {
		return s.AssetClass
	}
	return "forex"
}

func tickSizeForTicker(ticker string) float64 {
	if s, ok := config.SymbolByTicker(ticker); ok {
		return s.TickSize
	}
	return 0.00001
}

func FindDataDirectory() string {
	candidates := []string{
		"data/daily",
		"../data/daily",
		"../../data/daily",
	}
	cwd, _ := os.Getwd()
	candidates = append(candidates, filepath.Join(cwd, "data", "daily"))

	for _, c := range candidates {
		abs, _ := filepath.Abs(c)
		worldPath := filepath.Join(abs, "world")
		if info, err := os.Stat(worldPath); err == nil && info.IsDir() {
			return abs
		}
	}

	cwd, _ = os.Getwd()
	for {
		testPath := filepath.Join(cwd, "data", "daily", "world")
		if info, err := os.Stat(testPath); err == nil && info.IsDir() {
			return filepath.Join(cwd, "data", "daily")
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			break
		}
		cwd = parent
	}

	return "data/daily"
}

type ImportResult struct {
	Ticker     string
	Timeframe  string
	Fetched    int
	Stored     int
	Skipped    int
	Error      error
	Validation *ValidationResult
}

func (r *ImportResult) Summary() string {
	if r.Error != nil {
		return fmt.Sprintf("%s (%s): ERROR %v", r.Ticker, r.Timeframe, r.Error)
	}
	if r.Validation != nil && !r.Validation.Valid {
		return fmt.Sprintf("%s (%s): %d fetched, %d stored, %d skipped — %s",
			r.Ticker, r.Timeframe, r.Fetched, r.Stored, r.Skipped, r.Validation.Message)
	}
	return fmt.Sprintf("%s (%s): %d fetched, %d stored, %d skipped",
		r.Ticker, r.Timeframe, r.Fetched, r.Stored, r.Skipped)
}

func MapOrcaToStooqSymbols() []string {
	tickers, err := config.Tickers()
	if err != nil {
		return nil
	}
	return tickers
}

func StooqFileName(orcaTicker string) string {
	if s, ok := config.SymbolByTicker(orcaTicker); ok && s.StooqTicker != "" {
		return s.StooqTicker + ".txt"
	}
	return strings.ToLower(orcaTicker) + ".txt"
}

func StooqSubdir(orcaTicker string) string {
	if s, ok := config.SymbolByTicker(orcaTicker); ok {
		switch s.AssetClass {
		case "forex_major":
			return "currencies/major"
		case "crypto":
			return "cryptocurrencies"
		case "index_agg", "index_eu":
			return "indices"
		}
	}
	return "currencies/other"
}
