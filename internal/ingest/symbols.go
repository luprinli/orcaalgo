package ingest

import (
	"strings"

	"github.com/lee-econ/orca-core/internal/config"
)

// CanonicalSymbolMap maps provider-specific tickers (lowercased) to the
// canonical ORCA ticker, derived from configs/universe.json — the single
// source of truth shared with Python.
var CanonicalSymbolMap = buildCanonicalMap()

func buildCanonicalMap() map[string]string {
	m := make(map[string]string)
	u, err := config.Load()
	if err != nil {
		return m
	}
	for _, s := range u.Symbols {
		for _, provider := range []string{s.Ticker, s.StooqTicker, s.YahooTicker, s.TiingoTicker} {
			if provider != "" {
				m[strings.ToLower(provider)] = s.Ticker
			}
		}
	}
	return m
}

func ResolveTicker(raw string) string {
	upper := strings.ToUpper(raw)
	if ticker, ok := CanonicalSymbolMap[strings.ToLower(raw)]; ok {
		return ticker
	}
	return upper
}
