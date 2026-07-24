package strategy

import (
	"time"

	"github.com/lee-econ/orca-core/internal/types"
)

type Candle struct {
	Time   time.Time
	Open   types.Price
	High   types.Price
	Low    types.Price
	Close  types.Price
	Volume float64
	Symbol string
}

type Signal struct {
	Symbol   string
	Side     string
	Quantity float64
	PWin     float64 // ML meta-labeler win probability (0.0–1.0, 0 if unchecked)
}
