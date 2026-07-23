package retry

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"
)

type RetryConfig struct {
	MaxRetries    int
	BaseDelay     time.Duration
	MaxDelay      time.Duration
	RetryOnStatus []int
}

var DefaultRetryConfig = RetryConfig{
	MaxRetries:    3,
	BaseDelay:     time.Second,
	MaxDelay:      30 * time.Second,
	RetryOnStatus: []int{429, 500, 502, 503, 504},
}

func DoWithRetry(ctx context.Context, fn func() (*http.Response, error), config RetryConfig) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(math.Min(float64(config.BaseDelay)*math.Pow(2, float64(attempt-1)), float64(config.MaxDelay)))
			select { case <-ctx.Done(): return nil, ctx.Err(); case <-time.After(delay): }
		}
		resp, err := fn()
		if err != nil { lastErr = err; continue }
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil { lastErr = err; continue }
		shouldRetry := false
		for _, s := range config.RetryOnStatus {
			if resp.StatusCode == s { shouldRetry = true; break }
		}
		if !shouldRetry && resp.StatusCode < 300 { return data, nil }
		lastErr = fmt.Errorf("request failed %d: %s", resp.StatusCode, string(data))
		if !shouldRetry { return nil, lastErr }
	}
	return nil, fmt.Errorf("all %d retries exhausted: %w", config.MaxRetries, lastErr)
}