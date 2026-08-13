package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/lee-econ/orca-core/internal/config"
	"github.com/lee-econ/orca-core/internal/types"
)

type TiingoDataFetcher struct {
	client  *http.Client
	apiKey  string
	baseURL string
	delay   time.Duration
	logger  *slog.Logger
}

func NewTiingoDataFetcher(logger *slog.Logger) *TiingoDataFetcher {
	apiKey := os.Getenv("TIINGO_API_KEY")
	return &TiingoDataFetcher{
		client:  &http.Client{Timeout: 30 * time.Second},
		apiKey:  apiKey,
		baseURL: "https://api.tiingo.com",
		delay:   200 * time.Millisecond,
		logger:  logger,
	}
}

func NewTiingoDataFetcherWithKey(apiKey string, logger *slog.Logger) *TiingoDataFetcher {
	return &TiingoDataFetcher{
		client:  &http.Client{Timeout: 30 * time.Second},
		apiKey:  apiKey,
		baseURL: "https://api.tiingo.com",
		delay:   200 * time.Millisecond,
		logger:  logger,
	}
}

func (f *TiingoDataFetcher) Name() string { return "tiingo" }

func (f *TiingoDataFetcher) FetchCandles(ctx context.Context, ticker string, start, end time.Time, timeframe string) ([]CandleData, error) {
	if f.apiKey == "" {
		return nil, fmt.Errorf("tiingo: no API key set (TIINGO_API_KEY env var)")
	}

	tiingoTicker := f.resolveSymbol(ticker)
	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")

	url := fmt.Sprintf("%s/tiingo/daily/%s/prices?startDate=%s&endDate=%s&token=%s",
		f.baseURL, tiingoTicker, startStr, endStr, f.apiKey)

	f.logger.DebugContext(ctx, "tiingo_fetch", "url", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tiingo request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		f.logger.WarnContext(ctx, "tiingo_rate_limited", "retry_after", resp.Header.Get("Retry-After"))
		time.Sleep(60 * time.Second)
		return f.FetchCandles(ctx, ticker, start, end, timeframe)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tiingo status %d for %s", resp.StatusCode, ticker)
	}

	var raw []tiingoCandle
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("tiingo decode: %w", err)
	}

	candles := make([]CandleData, 0, len(raw))
	for _, c := range raw {
		t, parseErr := time.Parse("2006-01-02T15:04:05.000Z", c.Date)
		if parseErr != nil {
			t, parseErr = time.Parse("2006-01-02T15:04:05Z", c.Date)
			if parseErr != nil {
				continue
			}
		}
		candles = append(candles, CandleData{
			Time:   t,
			Open:   types.FromFloat64(c.Open),
			High:   types.FromFloat64(c.High),
			Low:    types.FromFloat64(c.Low),
			Close:  types.FromFloat64(c.Close),
			Volume: c.Volume,
		})
	}

	f.logger.InfoContext(ctx, "tiingo_fetched", "ticker", ticker, "candles", len(candles))
	time.Sleep(f.delay)
	return candles, nil
}

func (f *TiingoDataFetcher) FetchDailyMetrics(ctx context.Context, ticker string) (*SymbolMetrics, error) {
	return ComputeDailyMetrics(ctx, f, ticker)
}

func (f *TiingoDataFetcher) resolveSymbol(ticker string) string {
	canonical := ResolveTicker(ticker)
	if s, ok := config.SymbolByTicker(canonical); ok && s.TiingoTicker != "" {
		return s.TiingoTicker
	}
	return canonical
}

type tiingoCandle struct {
	Date        string  `json:"date"`
	Open        float64 `json:"open"`
	High        float64 `json:"high"`
	Low         float64 `json:"low"`
	Close       float64 `json:"close"`
	Volume      float64 `json:"volume"`
	AdjClose    float64 `json:"adjClose"`
	DivCash     float64 `json:"divCash"`
	SplitFactor float64 `json:"splitFactor"`
}
