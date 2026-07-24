package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/lee-econ/orca-core/internal/types"
)

type YahooDataFetcher struct {
	client  *http.Client
	baseURL string
	delay   time.Duration
	logger  *slog.Logger
}

func NewYahooDataFetcher(logger *slog.Logger) *YahooDataFetcher {
	return &YahooDataFetcher{
		client:  &http.Client{Timeout: 30 * time.Second},
		baseURL: "https://query1.finance.yahoo.com/v8/finance/chart",
		delay:   200 * time.Millisecond,
		logger:  logger,
	}
}

func (f *YahooDataFetcher) Name() string { return "yahoo" }

func (f *YahooDataFetcher) FetchCandles(ctx context.Context, ticker string, start, end time.Time, timeframe string) ([]CandleData, error) {
	symbol := f.resolveSymbol(ticker)
	url := fmt.Sprintf("%s/%s?period1=%d&period2=%d&interval=%s",
		f.baseURL, symbol, start.Unix(), end.Unix(), f.resolveInterval(timeframe))

	f.logger.DebugContext(ctx, "yahoo_fetch", "url", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("yahoo request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo status %d for %s", resp.StatusCode, ticker)
	}

	var result yahooResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("yahoo decode: %w", err)
	}

	if result.Chart.Error != nil {
		return nil, fmt.Errorf("yahoo error: %v", result.Chart.Error)
	}
	if len(result.Chart.Result) == 0 {
		return nil, nil
	}

	r := result.Chart.Result[0]
	if len(r.Timestamp) == 0 {
		return nil, nil
	}

	quotes := r.Indicators.Quote[0]
	candles := make([]CandleData, len(r.Timestamp))
	for i, ts := range r.Timestamp {
		candles[i] = CandleData{
			Time:   time.Unix(ts, 0).UTC(),
			Open:   types.FromFloat64(safeIndex(quotes.Open, i)),
			High:   types.FromFloat64(safeIndex(quotes.High, i)),
			Low:    types.FromFloat64(safeIndex(quotes.Low, i)),
			Close:  types.FromFloat64(safeIndex(quotes.Close, i)),
			Volume: safeIndex(quotes.Volume, i),
		}
	}

	cleaned := make([]CandleData, 0, len(candles))
	for _, c := range candles {
		if c.Open.IsZero() && c.Close.IsZero() {
			continue
		}
		cleaned = append(cleaned, c)
	}

	f.logger.InfoContext(ctx, "yahoo_fetched", "ticker", ticker, "candles", len(cleaned))
	time.Sleep(f.delay)
	return cleaned, nil
}

func (f *YahooDataFetcher) FetchDailyMetrics(ctx context.Context, ticker string) (*SymbolMetrics, error) {
	return ComputeDailyMetrics(ctx, f, ticker)
}

func (f *YahooDataFetcher) resolveSymbol(ticker string) string {
	ticker = strings.ToUpper(ticker)
	switch ticker {
	case "EURUSD":
		return "EURUSD=X"
	case "GBPUSD":
		return "GBPUSD=X"
	case "USDJPY":
		return "USDJPY=X"
	case "USDCHF":
		return "USDCHF=X"
	case "AUDUSD":
		return "AUDUSD=X"
	case "USDCAD":
		return "USDCAD=X"
	case "NZDUSD":
		return "NZDUSD=X"
	case "US30":
		return "^DJI"
	case "SPX500":
		return "^GSPC"
	case "NAS100":
		return "^IXIC"
	case "UK100":
		return "^FTSE"
	case "GER40":
		return "^GDAXI"
	case "JPN225":
		return "^N225"
	case "XAUUSD":
		return "GLD"
	case "XAGUSD":
		return "SLV"
	case "USOIL":
		return "USO"
	case "UKOIL":
		return "BNO"
	case "BTCUSD":
		return "BTC-USD"
	case "ETHUSD":
		return "ETH-USD"
	default:
		return ticker
	}
}

func (f *YahooDataFetcher) resolveInterval(timeframe string) string {
	switch timeframe {
	case "1d", "1day", "daily":
		return "1d"
	case "1wk", "1week", "weekly":
		return "1wk"
	case "1mo", "1month", "monthly":
		return "1mo"
	default:
		return "1d"
	}
}

func safeIndex(arr []float64, i int) float64 {
	if arr != nil && i < len(arr) {
		return arr[i]
	}
	return 0
}

type yahooResponse struct {
	Chart struct {
		Result []yahooResult `json:"result"`
		Error  interface{}   `json:"error"`
	} `json:"chart"`
}

type yahooResult struct {
	Timestamp  []int64     `json:"timestamp"`
	Indicators struct {
		Quote []yahooQuote `json:"quote"`
	} `json:"indicators"`
}

type yahooQuote struct {
	Open   []float64 `json:"open"`
	High   []float64 `json:"high"`
	Low    []float64 `json:"low"`
	Close  []float64 `json:"close"`
	Volume []float64 `json:"volume"`
}
