package market

import (
	"time"
)

// CorporateAction represents a stock split, reverse split, dividend, or
// other adjustment event. Factor is the multiplier to apply to historical
// prices (e.g., 2.0 for a 2:1 split, 0.5 for a 1:2 reverse split).
type CorporateAction struct {
	Symbol string
	Date   time.Time
	Type   string
	Factor float64
}

// AdjustmentProvider returns the cumulative adjustment factor to apply to
// prices for a given symbol as of the specified date. Implementations may
// source data from a database, CSV, or external API.
type AdjustmentProvider interface {
	GetAdjustmentFactor(symbol string, date time.Time) float64
}

// NoOpAdjustmentProvider is a no-op implementation that always returns 1.0
// (no adjustment). Use this when corporate action data is unavailable.
type NoOpAdjustmentProvider struct{}

func (p *NoOpAdjustmentProvider) GetAdjustmentFactor(symbol string, date time.Time) float64 {
	return 1.0
}
