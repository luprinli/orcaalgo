package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type SentimentClient struct {
	baseURL    string
	httpClient *http.Client
	lastValue  int
	lastLabel  string
	lastFetch  time.Time
}

type alternativeMeResponse struct {
	Data []struct {
		Value     string `json:"value"`
		Timestamp string `json:"timestamp"`
	} `json:"data"`
}

type cnnFearGreedResponse struct {
	FearGreed struct {
		Score              float64 `json:"score"`
		Rating             string  `json:"rating"`
		Timestamp          string  `json:"timestamp"`
		PreviousClose      float64 `json:"previous_close"`
		Previous1Week      float64 `json:"previous_1_week"`
		Previous1Month     float64 `json:"previous_1_month"`
		Previous1Year      float64 `json:"previous_1_year"`
	} `json:"fear_and_greed"`
}

func NewSentimentClient() *SentimentClient {
	return &SentimentClient{
		baseURL: "https://api.alternative.me/fng/",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *SentimentClient) Fetch(ctx context.Context) (int, string, error) {
	url := s.baseURL + "?limit=1"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, "", err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("sentiment fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, "", fmt.Errorf("sentiment fetch: status %d", resp.StatusCode)
	}

	var result alternativeMeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, "", fmt.Errorf("sentiment decode: %w", err)
	}

	if len(result.Data) == 0 {
		return 0, "", fmt.Errorf("sentiment: no data")
	}

	var value int
	fmt.Sscanf(result.Data[0].Value, "%d", &value)

	label := classifySentiment(value)

	s.lastValue = value
	s.lastLabel = label
	s.lastFetch = time.Now()

	log.Printf("sentiment: Fear&Greed=%d (%s)", value, label)
	return value, label, nil
}

func (s *SentimentClient) LastValue() int {
	return s.lastValue
}

func (s *SentimentClient) LastLabel() string {
	return s.lastLabel
}

func classifySentiment(value int) string {
	switch {
	case value <= 25:
		return "Extreme Fear"
	case value <= 45:
		return "Fear"
	case value <= 55:
		return "Neutral"
	case value <= 75:
		return "Greed"
	default:
		return "Extreme Greed"
	}
}
