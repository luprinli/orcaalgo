package risk

import (
	"context"
)

// CapitalGate authorizes capital allocation and reconciles fills against a
// shared pool. Both backtest (CapitalPoolSim) and live (CapitalPoolManager)
// implement this interface.
type CapitalGate interface {
	// RequestCapital asks the pool to authorize a position of the given size on
	// the given symbol/side for the given strategy. Returns the approved size
	// (may be less than requested, or zero) and a machine-readable reason.
	RequestCapital(ctx context.Context, req CapitalRequest) CapitalResult

	// RecordFill notifies the pool that a trade has closed. The pool updates its
	// running balance, peak balance, daily PnL, and per-strategy allocation.
	RecordFill(strategyID, symbol, side string, pnl, quantity float64)

	// ResetDaily zeroes daily PnL for all strategies and advances the trading day
	// counter.
	ResetDaily()

	// Halted returns true when the pool has been halted (e.g., daily loss or max
	// drawdown breached).
	Halted() bool

	// HaltReason returns a human-readable explanation of why the pool was halted,
	// or an empty string when not halted.
	HaltReason() string

	// TotalBalance returns the current pool equity.
	TotalBalance() float64
}

// PropFirmGate enforces prop-firm rule compliance on every fill and each new
// trading day. Both PropFirmEnforcer (backtest) and propfirm.Manager (live)
// implement this interface.
type PropFirmGate interface {
	// CheckDailyLimits returns whether daily loss and max-drawdown limits
	// have been breached, plus the machine-readable reason when they have.
	CheckDailyLimits() (ok bool, reason string)

	// OnFill is called when a trade closes. The gate updates daily PnL,
	// consistency tracking, and cumulative PnL for profit-target tracking.
	OnFill(pnl float64, balance float64)

	// OnNewDay is called at the start of each trading day. The gate resets
	// daily PnL, applies consistency penalties, increments the trading day
	// counter, and checks profit-target advancement.
	OnNewDay()

	// IsHalted returns whether the gate has been violated and trading should stop.
	IsHalted() bool

	// HaltReason returns a human-readable violation reason.
	HaltReason() string

	// MarkViolated permanently marks this gate as failed with the given reason.
	// Used when KillSwitch or an external circuit-breaker fires.
	MarkViolated(reason string)

	// CurrentPhase returns the active prop-firm phase (1 or 2).
	CurrentPhase() int

	// ProfitTargetMet returns true when the cumulative PnL has reached the
	// current phase's profit target and the minimum trading days are satisfied.
	ProfitTargetMet() bool

	// GetPositionSize caps a raw quantity against the profile's MaxPositionPct
	// and applies regime-multiplier sizing.
	GetPositionSize(baseQuantity float64) float64
}

// SignalGate validates and sizes a raw strategy signal before capital
// authorization. Both Engine.generateSignal and LiveEngine.ProcessTick
// delegate sizing and exposure checks here.
type SignalGate interface {
	// ValidateSignal performs pre-screening checks (volatility halt, capital
	// positivity, rate limiting) and returns whether the signal may proceed.
	ValidateSignal(runningCapital float64) (ok bool, reason string)

	// ApplySizing applies regime, seasonal, Kelly, and confidence multipliers
	// to the raw quantity, returning the risk-adjusted size.
	ApplySizing(baseSize, runningCapital, confidence float64) float64

	// CheckExposure verifies that adding a position of the given notional on the
	// given symbol/side does not violate max-leverage or symbol-concentration
	// limits.
	CheckExposure(symbol, side string, notional float64) (ok bool, reason string)

	// RecordExposure registers an open position so that subsequent exposure
	// checks account for it.
	RecordExposure(symbol, side string, notional float64)

	// RemoveExposure removes a closed position from exposure tracking.
	RemoveExposure(symbol, side string, notional float64)
}

// Compile-time interface compliance checks.
// These are placed in the packages that define the concrete types to avoid
// import cycles.
