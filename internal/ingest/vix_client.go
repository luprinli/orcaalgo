package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type VIXClient struct {
	baseURL    string
	httpClient *http.Client
	lastVIX    float64
	lastChange float64
	lastFetch  time.Time
}

type polygonVIXResponse struct {
	Ticker   string  `json:"ticker"`
	Results  []vixBar `json:"results"`
}

type vixBar struct {
	Close float64 `json:"c"`
	Time  int64   `json:"t"`
}

func NewVIXClient() *VIXClient {
	return &VIXClient{
		baseURL: "https://api.polygon.io",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (v *VIXClient) FetchLatest(ctx context.Context, apiKey string) (float64, float64, error) {
	now := time.Now()
	from := now.AddDate(0, 0, -5).Format("2006-01-02")
	to := now.Format("2006-01-02")

	url := fmt.Sprintf("%s/v2/aggs/ticker/I:VIX/range/1/day/%s/%s?apiKey=%s",
		v.baseURL, from, to, apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, 0, err
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("vix fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("vix fetch: status %d", resp.StatusCode)
	}

	var result polygonVIXResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, 0, fmt.Errorf("vix decode: %w", err)
	}

	if len(result.Results) < 2 {
		return 0, 0, fmt.Errorf("vix: insufficient bars (%d)", len(result.Results))
	}

	latest := result.Results[len(result.Results)-1].Close
	prev := result.Results[len(result.Results)-2].Close
	change := (latest - prev) / prev * 100.0

	v.lastVIX = latest
	v.lastChange = change
	v.lastFetch = time.Now()

	log.Printf("vix: latest=%.2f, change=%.2f%%", latest, change)
	return latest, change, nil
}

func (v *VIXClient) LastVIX() float64 {
	return v.lastVIX
}

func (v *VIXClient) LastChange() float64 {
	return v.lastChange
}

type VIXTick struct {
	Timestamp int64
	VIXValue  float64
	VIXChange float64
}
