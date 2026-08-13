package broker

import (
	"context"
	"time"

	"github.com/lee-econ/orca-core/internal/types"
)

// MarketDataProvider is the broker-side data service: asset-universe listing,
// market clock (session gating), latest trade price, and corporate actions.
// It is an optional capability — order-only brokers need not implement it.
type MarketDataProvider interface {
	Assets(ctx context.Context) ([]Asset, error)
	Clock(ctx context.Context) (*MarketClock, error)
	LatestPrice(ctx context.Context, symbol string) (types.Price, error)
	CorporateActions(ctx context.Context, symbols []string) ([]CorporateAction, error)
}

// Asset is a tradable instrument in the broker's universe.
type Asset struct {
	Symbol       string `json:"symbol"`
	Name         string `json:"name"`
	Class        string `json:"class"`
	Exchange     string `json:"exchange"`
	Tradable     bool   `json:"tradable"`
	Shortable    bool   `json:"shortable"`
	EasyToBorrow bool   `json:"easy_to_borrow"`
}

// MarketClock reports whether the market is open and the next open/close times.
type MarketClock struct {
	IsOpen    bool      `json:"is_open"`
	NextOpen  time.Time `json:"next_open"`
	NextClose time.Time `json:"next_close"`
}

// CorporateAction is a broker-reported split/dividend event for a symbol.
type CorporateAction struct {
	Symbol       string    `json:"symbol"`
	Date         time.Time `json:"date"`
	Type         string    `json:"type"` // "split" | "dividend" | ...
	SplitRatio   float64   `json:"split_ratio"`
	CashDividend float64   `json:"cash_dividend"`
}
