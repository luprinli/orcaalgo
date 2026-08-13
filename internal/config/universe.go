// Package config loads the canonical trading universe from configs/universe.json.
// This is the single source of truth for symbols, strategies, timeframes, and
// data sources, shared between Go and Python consumers.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Universe is the top-level config document.
type Universe struct {
	Version     int               `json:"version"`
	Description string            `json:"description"`
	Symbols     []Symbol          `json:"symbols"`
	Strategies  []string          `json:"strategies"`
	Timeframes  []string          `json:"timeframes"`
	DataSources []string          `json:"data_sources"`
	Pairs       map[string]string `json:"pairs"`
}

// Symbol is a single tradable instrument with provider-specific tickers.
type Symbol struct {
	Ticker       string  `json:"ticker"`
	Exchange     string  `json:"exchange"`
	AssetClass   string  `json:"asset_class"`
	TickSize     float64 `json:"tick_size"`
	LotSize      float64 `json:"lot_size"`
	StooqTicker  string  `json:"stooq_ticker"`
	YahooTicker  string  `json:"yahoo_ticker"`
	TiingoTicker string  `json:"tiingo_ticker"`
	BasePrice    float64 `json:"base_price"`
	SigmaDaily   float64 `json:"sigma_daily"`
}

var (
	mu      sync.RWMutex
	cached  *Universe
	loadErr error
)

// ConfigPath resolves the universe config path. Resolution order:
//  1. ORCA_UNIVERSE_CONFIG environment variable.
//  2. configs/universe.json relative to the working directory, then each
//     ancestor directory (so `go test` from a package dir still finds it).
func ConfigPath() string {
	if p := os.Getenv("ORCA_UNIVERSE_CONFIG"); p != "" {
		return p
	}
	dir, err := os.Getwd()
	if err != nil {
		return filepath.Join("configs", "universe.json")
	}
	for {
		candidate := filepath.Join(dir, "configs", "universe.json")
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Join("configs", "universe.json")
}

// Load reads the universe config, caching the result for subsequent calls.
func Load() (*Universe, error) {
	mu.RLock()
	if cached != nil {
		defer mu.RUnlock()
		return cached, nil
	}
	if loadErr != nil {
		defer mu.RUnlock()
		return nil, loadErr
	}
	mu.RUnlock()

	mu.Lock()
	defer mu.Unlock()
	// Double-check in case another goroutine populated the cache.
	if cached != nil {
		return cached, nil
	}
	if loadErr != nil {
		return nil, loadErr
	}

	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		loadErr = fmt.Errorf("config.Load: %w", err)
		return nil, loadErr
	}
	var u Universe
	if err := json.Unmarshal(data, &u); err != nil {
		loadErr = fmt.Errorf("config.Load: %w", err)
		return nil, loadErr
	}
	cached = &u
	return cached, nil
}

// Tickers returns the canonical ticker list from the universe config.
func Tickers() ([]string, error) {
	u, err := Load()
	if err != nil {
		return nil, err
	}
	out := make([]string, len(u.Symbols))
	for i, s := range u.Symbols {
		out[i] = s.Ticker
	}
	return out, nil
}

// TickerToStooq maps a canonical ticker to its stooq filename token (e.g.
// "spy.us", "eurusd", "btc.v", "^_us").
func TickerToStooq(ticker string) string {
	u, err := Load()
	if err != nil {
		return ""
	}
	for _, s := range u.Symbols {
		if s.Ticker == ticker {
			return s.StooqTicker
		}
	}
	return ""
}

// TickerToYahoo maps a canonical ticker to its Yahoo Finance ticker.
func TickerToYahoo(ticker string) string {
	u, err := Load()
	if err != nil {
		return ""
	}
	for _, s := range u.Symbols {
		if s.Ticker == ticker {
			return s.YahooTicker
		}
	}
	return ""
}

// SymbolByTicker returns the full symbol definition for a canonical ticker.
func SymbolByTicker(ticker string) (*Symbol, bool) {
	u, err := Load()
	if err != nil {
		return nil, false
	}
	for i := range u.Symbols {
		if u.Symbols[i].Ticker == ticker {
			return &u.Symbols[i], true
		}
	}
	return nil, false
}

// SecondaryTicker returns the pairs-trading counterpart for a ticker, or ""
// if none is configured.
func SecondaryTicker(ticker string) string {
	u, err := Load()
	if err != nil {
		return ""
	}
	return u.Pairs[ticker]
}
