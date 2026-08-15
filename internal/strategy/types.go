package strategy

import (
	"time"

	"github.com/lee-econ/orca-core/internal/types"
)

type Candle struct {
	Time             time.Time
	Open             types.Price
	High             types.Price
	Low              types.Price
	Close            types.Price
	Volume           float64
	Symbol           string
	AdjustmentFactor float64
	Source           string
	GenerationID     string
}

type Signal struct {
	Symbol     string
	Side       string
	Quantity   float64
	PWin       float64
	StrategyID string
	// StopLoss and TakeProfit let a strategy specify its own risk levels.
	// When set (non-zero), the engine uses them instead of the generic
	// config.StopLoss/config.TakeProfit defaults.
	StopLoss   types.Price
	TakeProfit types.Price
}
