package risk

import (
	"context"
	"math"

	"github.com/lee-econ/orca-core/internal/monitor"
)

const maxPositionPct = 0.02
const maxTotalNotionalPct = 0.10

// ProcessSignalRequest is the input to RiskPipeline.ProcessSignal.
// Both the backtest Engine.generateSignal() and LiveEngine.ProcessTick()
// construct this from their respective signal representations.
type ProcessSignalRequest struct {
	StrategyID       string
	Symbol           string
	Side             string
	Price            float64 // current bar close for notional calculation
	Confidence       float64 // 0–1 probability (PWin, meta-labeler output)
	BaseSize         float64 // raw quantity from the strategy runner
	ExistingPosition float64 // current open quantity for this symbol (0 if none)
	RunningCapital   float64 // current equity available
	AllowFractional  bool    // permit sub-1-share positions (orchestrator sizing)
	IsExit           bool    // exit signal — skips volatility/capital/position checks
}

// ProcessSignalResult is the output of RiskPipeline.ProcessSignal.
type ProcessSignalResult struct {
	Approved   bool
	Size       float64
	Reason     string // why rejected, or "ok" on approval
}

// RiskPipeline is the canonical signal-audit pipeline shared by the backtest
// Engine and the live LiveEngine. It composes sizing, exposure, capital-gate,
// prop-firm checks, and regime-aware strategy gating in a fixed, enforceable
// order. Adding a new risk check here automatically applies to both backtest
// and live paths.
type RiskPipeline struct {
	SignalGate  SignalGate
	Capital     CapitalGate
	PropFirm    PropFirmGate
	KellyMult   float64

	// RegimeMatrix gates strategies by regime. When set, every signal is
	// checked against the matrix: if the strategy is not allowed in the
	// current regime, the signal is rejected. The matrix also provides
	// per-regime Kelly multiplier overrides.
	RegimeMatrix *RegimeActivationMatrix

	// CurrentRegime is the HMM regime state (0-3) at the time of the signal.
	// Updated by the caller before each ProcessSignal call.
	CurrentRegime int8

	// symbolNotional tracks cumulative open notional per symbol across all
	// positions, for enforcement of the aggregate portfolio notional cap (§E2).
	symbolNotional map[string]float64
}

// ProcessSignal runs the full signal → approval/rejection pipeline.
// Order: volatility halt → sizing → capital gate halt → sizing → exposure →
// capital authorization.
func (p *RiskPipeline) ProcessSignal(ctx context.Context, req ProcessSignalRequest) ProcessSignalResult {
	if req.IsExit {
		return ProcessSignalResult{Approved: true, Size: req.ExistingPosition, Reason: "exit"}
	}

	// 1. Signal gate: volatility halt, running-capital positivity, rate limit.
	if p.SignalGate != nil {
		if ok, reason := p.SignalGate.ValidateSignal(req.RunningCapital); !ok {
			monitor.RecordReject()
			monitor.RecordSignalReject("signal", req.StrategyID)
			return ProcessSignalResult{Approved: false, Reason: "signal:" + reason}
		}
	}

	// 2. Capital gate halt — don't waste time sizing if the pool is already dead.
	if p.Capital != nil && p.Capital.Halted() {
		monitor.RecordReject()
		monitor.RecordSignalReject("capital_halt", req.StrategyID)
		return ProcessSignalResult{Approved: false, Reason: "pool_halted:" + p.Capital.HaltReason()}
	}

	// 3. Prop-firm halt — separate from capital-gate halt (enforcer vs pool).
	if p.PropFirm != nil && p.PropFirm.IsHalted() {
		monitor.RecordReject()
		monitor.RecordSignalReject("propfirm_halt", req.StrategyID)
		return ProcessSignalResult{Approved: false, Reason: "propfirm_halted:" + p.PropFirm.HaltReason()}
	}

	// 3b. Regime activation gate — block strategies not allowed in current regime.
	if p.RegimeMatrix != nil {
		if !p.RegimeMatrix.IsAllowed(req.StrategyID, p.CurrentRegime) {
			monitor.RecordReject()
			monitor.RecordSignalReject("regime_blocked", req.StrategyID)
			return ProcessSignalResult{Approved: false, Reason: "regime_blocked"}
		}
	}

	// 4. Apply sizing: Kelly, regime, seasonal, confidence.
	size := req.BaseSize
	if size <= 0 {
		monitor.RecordReject()
		monitor.RecordSignalReject("size_zero", req.StrategyID)
		return ProcessSignalResult{Approved: false, Reason: "size_zero"}
	}
	if p.SignalGate != nil {
		size = p.SignalGate.ApplySizing(size, req.RunningCapital, req.Confidence)
	}
	kelly := p.KellyMult
	if p.RegimeMatrix != nil {
		if override := p.RegimeMatrix.KellyForRegime(req.StrategyID, p.CurrentRegime); override > 0 {
			kelly = override
		}
	}
	size *= kelly

	// 4b. Soft halt: if the prop-firm gate indicates a soft halt (daily loss
	// between soft and hard thresholds), reduce position size by 50%.
	if p.PropFirm != nil {
		if sh, ok := p.PropFirm.(interface {
			IsSoftHalted() bool
			SoftHaltMultiplier() float64
		}); ok && sh.IsSoftHalted() {
			size *= sh.SoftHaltMultiplier()
		}
	}

	if size <= 0 {
		monitor.RecordReject()
		monitor.RecordSignalReject("size_zero_after_sizing", req.StrategyID)
		return ProcessSignalResult{Approved: false, Reason: "size_zero_after_sizing"}
	}

	// 4c. Universal notional cap: prevent any single position from exceeding
	// a fraction of account equity, regardless of strategy self-sizing. This
	// is the single authoritative position-size guard — no other code path
	// independently enforces position limits (drift safeguard).
	maxNotional := req.RunningCapital * maxPositionPct
	positionNotional := size * req.Price
	if positionNotional > maxNotional && maxNotional > 0 {
		size = maxNotional / req.Price
	}

	// 4d. Portfolio-level aggregate notional cap: prevent cumulative open
	// notional per symbol from exceeding maxTotalNotionalPct of equity.
	// This catches strategies that accumulate multiple simultaneous positions
	// (grid with up to 10 levels, pairs with hedge ratio, etc.).
	if p.symbolNotional == nil {
		p.symbolNotional = make(map[string]float64)
	}
	existingNotional := p.symbolNotional[req.Symbol]
	newNotional := math.Abs(size * req.Price)
	totalNotional := existingNotional + newNotional
	maxTotalNotional := req.RunningCapital * maxTotalNotionalPct
	if totalNotional > maxTotalNotional && maxTotalNotional > 0 {
		available := maxTotalNotional - existingNotional
		if available <= 0 {
			return ProcessSignalResult{Approved: false, Reason: "aggregate_notional_cap:max_reached"}
		}
		size = available / req.Price
	}
	p.symbolNotional[req.Symbol] = existingNotional + math.Abs(size*req.Price)

	// 5. Exposure check: max leverage, symbol concentration.
	if p.SignalGate != nil {
		notional := math.Abs(size * req.Price)
		if ok, reason := p.SignalGate.CheckExposure(req.Symbol, req.Side, notional); !ok {
			monitor.RecordReject()
			monitor.RecordSignalReject("exposure", req.StrategyID)
			return ProcessSignalResult{Approved: false, Reason: "exposure:" + reason}
		}
	}

	// 6. Cross-strategy correlation brake: if any other strategy already has an
	// open position on the same symbol + same direction, halve the total size
	// to avoid overexposure to the same directional bet.
	if p.Capital != nil {
		if cs, ok := p.Capital.(interface {
			HasOpenPosition(string, string) bool
		}); ok && cs.HasOpenPosition(req.Symbol, req.Side) {
			size *= 0.5
		}
	}

	// 7. Capital authorization through the pool.
	if p.Capital != nil {
		capitalReq := CapitalRequest{
			StrategyID: req.StrategyID,
			Confidence: req.Confidence,
			Symbol:     req.Symbol,
			Side:       req.Side,
			BaseSize:   size,
		}
		result := p.Capital.RequestCapital(ctx, capitalReq)
		if result.ApprovedSize <= 0 {
			monitor.RecordReject()
			monitor.RecordSignalReject("capital", req.StrategyID)
			return ProcessSignalResult{Approved: false, Reason: "capital:" + result.Reason}
		}
		size = result.ApprovedSize
	}

	// 6b. Optional: correlation penalty for opposing same-symbol signals.
	if p.Capital != nil {
		if cp, ok := p.Capital.(interface {
			ApplyCorrelationReduction(string, string, float64) float64
		}); ok {
			size = cp.ApplyCorrelationReduction(req.Side, req.Symbol, size)
			if size <= 0 {
				monitor.RecordReject()
				monitor.RecordSignalReject("correlation", req.StrategyID)
				return ProcessSignalResult{Approved: false, Reason: "correlation_penalty"}
			}
		}
	}

	// 6c. Optional: per-strategy drawdown gate (halts individual strategies).
	if p.Capital != nil {
		if psd, ok := p.Capital.(interface {
			PerStrategyDrawdown(string) float64
		}); ok {
			if dd := psd.PerStrategyDrawdown(req.StrategyID); dd > 0.05 {
				monitor.RecordReject()
				monitor.RecordSignalReject("strategy_dd", req.StrategyID)
				return ProcessSignalResult{Approved: false, Reason: "per_strategy_drawdown:" + req.StrategyID}
			}
		}
	}

	// 7. If prop-firm gate provides position-size caps, apply them last.
	if p.PropFirm != nil {
		size = p.PropFirm.GetPositionSize(size)
	}

	// 8. Record the exposure for subsequent checks.
	if p.SignalGate != nil && size > 0 {
		p.SignalGate.RecordExposure(req.Symbol, req.Side, math.Abs(size*req.Price))
	}

	return ProcessSignalResult{Approved: true, Size: size, Reason: "ok"}
}

// SanitizeTradePnL clamps a raw trade PnL to a reasonable bound and guards
// against NaN/Inf values. It returns the sanitized PnL and a boolean indicating
// whether clamping occurred. This is the single canonical PnL sanitization point:
// all trade-exit paths (mid-simulation, end-of-data closeout, live fill, backtest
// fill) must route through this function to prevent backtest/live drift.
func SanitizeTradePnL(pnl, quantity, entryPrice, referenceCapital float64) (float64, bool) {
	if math.IsNaN(pnl) || math.IsInf(pnl, 0) {
		return 0, true
	}
	notional := quantity * entryPrice
	maxByNotional := notional * 0.50
	maxByCapital := referenceCapital * 0.10
	maxPnL := maxByNotional
	if maxByCapital < maxByNotional {
		maxPnL = maxByCapital
	}
	if maxPnL < referenceCapital*0.02 {
		maxPnL = referenceCapital * 0.02
	}
	if math.Abs(pnl) > maxPnL {
		return 0, true
	}
	return pnl, false
}

// ReconcileFill is the canonical fill → bookkeeping pipeline. Both the
// backtest Engine trade-close and LiveEngine.SignalOutcome call this.
func (p *RiskPipeline) ReconcileFill(strategyID, symbol, side string, pnl, quantity, price float64) {
	safePnL, _ := SanitizeTradePnL(pnl, quantity, price, p.capitalBalance())

	if p.Capital != nil {
		p.Capital.RecordFill(strategyID, symbol, side, safePnL, quantity)
	}

	if p.PropFirm != nil && p.Capital != nil {
		p.PropFirm.OnFill(safePnL, p.Capital.TotalBalance())
		if ok, reason := p.PropFirm.CheckDailyLimits(); !ok {
			p.PropFirm.MarkViolated(reason)
		}
	}

	// Release the aggregate notional cap for this symbol's closed position.
	if p.symbolNotional == nil {
		p.symbolNotional = make(map[string]float64)
	}
	p.symbolNotional[symbol] -= math.Abs(quantity * price)
	if p.symbolNotional[symbol] < 0 {
		p.symbolNotional[symbol] = 0
	}

	if p.SignalGate != nil {
		p.SignalGate.RemoveExposure(symbol, side, math.Abs(quantity*price))
	}
}

// ReconcileFillWithoutPropFirm is a variant that skips prop-firm checks.
// Used when the caller wants to handle prop-firm bookkeeping separately
// (e.g., in the backtest Engine which calls ftmo.OnFill directly).
func (p *RiskPipeline) ReconcileFillWithoutPropFirm(strategyID, symbol, side string, pnl, quantity, price float64) {
	safePnL, _ := SanitizeTradePnL(pnl, quantity, price, p.capitalBalance())

	if p.Capital != nil {
		p.Capital.RecordFill(strategyID, symbol, side, safePnL, quantity)
	}

	if p.symbolNotional == nil {
		p.symbolNotional = make(map[string]float64)
	}
	p.symbolNotional[symbol] -= math.Abs(quantity * price)
	if p.symbolNotional[symbol] < 0 {
		p.symbolNotional[symbol] = 0
	}

	if p.SignalGate != nil {
		p.SignalGate.RemoveExposure(symbol, side, math.Abs(quantity*price))
	}
}

func (p *RiskPipeline) capitalBalance() float64 {
	if p.Capital != nil {
		return p.Capital.TotalBalance()
	}
	return 0
}
