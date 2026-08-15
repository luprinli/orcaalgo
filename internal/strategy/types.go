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

// SignalAction is the semantic role of a Signal: an entry (open a position),
// an exit (close the current position), or none (no action). This replaces the
// fragile "Quantity == 0 means exit" sentinel (Rule 13) — a zero-quantity entry
// is now impossible to confuse with an exit.
type SignalAction int8

const (
	SignalNone  SignalAction = 0
	SignalEntry SignalAction = 1
	SignalExit  SignalAction = 2
)

type Signal struct {
	Symbol   string
	Side     string
	Quantity float64
	// Action is the semantic role of the signal (Entry/Exit/None). Entries carry
	// a positive Quantity; exits use Action=SignalExit with Quantity left zero.
	Action     SignalAction
	PWin       float64
	StrategyID string
	// StopLoss and TakeProfit let a strategy specify its own risk levels.
	// When set (non-zero), the engine uses them instead of the generic
	// config.StopLoss/config.TakeProfit defaults.
	StopLoss   types.Price
	TakeProfit types.Price
}
