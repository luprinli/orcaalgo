package ingest

import (
	"context"
	"time"

	"github.com/lee-econ/orca-core/internal/types"
)

type DataFetcher interface {
	FetchCandles(ctx context.Context, ticker string, start, end time.Time, timeframe string) ([]CandleData, error)
	FetchDailyMetrics(ctx context.Context, ticker string) (*SymbolMetrics, error)
	Name() string
}

type CandleData struct {
	Time   time.Time
	Open   types.Price
	High   types.Price
	Low    types.Price
	Close  types.Price
	Volume float64
}

type SymbolMetrics struct {
	AvgVolume20D  float64
	CurrentVolume float64
	ATR14         float64
	ATR14Pct      float64
	RSI14         float64
	Price         types.Price
}
