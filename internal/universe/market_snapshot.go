package universe

import (
	"time"

	"github.com/lee-econ/orca-core/internal/types"
)

type MarketDataSnapshot struct {
	Timestamp      time.Time
	SymbolMetrics  map[string]SymbolSnapshotMetric
	VIX            float64
	FearGreedIndex float64
}

type SymbolSnapshotMetric struct {
	AvgVolume20D  float64
	CurrentVolume float64
	ATR14         float64
	ATR14Pct      float64
	RSI14         float64
	Price         types.Price
	MarketCap     int64
	NewsSentiment float64
}
