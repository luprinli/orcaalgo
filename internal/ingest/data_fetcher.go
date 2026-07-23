package ingest

import (
	"context"
	"time"
)

type DataFetcher interface {
	FetchCandles(ctx context.Context, ticker string, start, end time.Time, timeframe string) ([]CandleData, error)
	FetchDailyMetrics(ctx context.Context, ticker string) (*SymbolMetrics, error)
	Name() string
}

type CandleData struct {
	Time   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}

type SymbolMetrics struct {
	AvgVolume20D  float64
	CurrentVolume float64
	ATR14         float64
	ATR14Pct      float64
	RSI14         float64
	Price         float64
}
