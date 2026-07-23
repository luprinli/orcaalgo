package ingest

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CSVExporter struct {
	pool *pgxpool.Pool
}

func NewCSVExporter(pool *pgxpool.Pool) *CSVExporter {
	return &CSVExporter{pool: pool}
}

func (e *CSVExporter) ExportCandles(ctx context.Context, symbols []string, start, end, timeframe, outputDir string) (map[string]string, error) {
	query := `SELECT c.time, c.open_raw, c.high_raw, c.low_raw, c.close_raw, c.volume, s.ticker
		 FROM candles c JOIN symbols s ON c.symbol_id = s.id
		 WHERE s.ticker = ANY($1) AND c.time BETWEEN $2 AND $3`
	args := []interface{}{symbols, start, end}
	if timeframe != "" && timeframe != "1d" {
		query += ` AND c.timeframe = $4 ORDER BY s.ticker, c.time ASC`
		args = append(args, timeframe)
	} else {
		query += ` ORDER BY s.ticker, c.time ASC`
	}

	rows, err := e.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query candles: %w", err)
	}
	defer rows.Close()

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", outputDir, err)
	}

	type rec struct {
		time   time.Time
		open   float64
		high   float64
		low    float64
		close_ float64
		volume float64
	}

	bySymbol := make(map[string][]rec)
	for rows.Next() {
		var ts time.Time
		var ticker string
		var openRaw, highRaw, lowRaw, closeRaw, vol int64
		if err := rows.Scan(&ts, &openRaw, &highRaw, &lowRaw, &closeRaw, &vol, &ticker); err != nil {
			continue
		}
		r := rec{
			time:   ts,
			open:   float64(openRaw) / PRICE_SCALE_I,
			high:   float64(highRaw) / PRICE_SCALE_I,
			low:    float64(lowRaw) / PRICE_SCALE_I,
			close_: float64(closeRaw) / PRICE_SCALE_I,
			volume: float64(vol),
		}
		bySymbol[ticker] = append(bySymbol[ticker], r)
	}

	files := make(map[string]string)
	for ticker, recs := range bySymbol {
		fileName := fmt.Sprintf("%s_%s.csv", strings.ToLower(ticker), timeframe)
		path := filepath.Join(outputDir, fileName)
		f, err := os.Create(path)
		if err != nil {
			continue
		}
		w := csv.NewWriter(f)
		w.Write([]string{"Date", "Open", "High", "Low", "Close", "Volume"})
		for _, r := range recs {
			w.Write([]string{
				r.time.Format("2006-01-02 15:04:05"),
				fmt.Sprintf("%.4f", r.open),
				fmt.Sprintf("%.4f", r.high),
				fmt.Sprintf("%.4f", r.low),
				fmt.Sprintf("%.4f", r.close_),
				fmt.Sprintf("%.0f", r.volume),
			})
		}
		w.Flush()
		f.Close()
		files[ticker] = path
	}
	return files, nil
}
