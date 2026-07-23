package ingest

import (
	"context"
	"time"
)

func ComputeDailyMetrics(ctx context.Context, fetcher DataFetcher, ticker string) (*SymbolMetrics, error) {
	end := time.Now()
	start := end.AddDate(0, -1, 0)
	candles, err := fetcher.FetchCandles(ctx, ticker, start, end, "1d")
	if err != nil {
		return nil, err
	}
	if len(candles) == 0 {
		return nil, &NoDataError{Provider: fetcher.Name(), Ticker: ticker}
	}
	var sumVol float64
	for _, c := range candles {
		sumVol += c.Volume
	}
	avgVol := sumVol / float64(len(candles))
	last := candles[len(candles)-1]
	return &SymbolMetrics{
		AvgVolume20D:  avgVol,
		CurrentVolume: last.Volume,
		Price:         last.Close,
	}, nil
}

type NoDataError struct {
	Provider string
	Ticker   string
}

func (e *NoDataError) Error() string {
	return e.Provider + ": no data for " + e.Ticker
}
