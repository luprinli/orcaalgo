# Capital Pool & Prop Firm Risk — Live Engine Wiring Plan

**Date:** 2026-07-25
**Status:** Implementation plan (no code changes yet)
**Scope:** Wire `LiveEngine` into existing `MultiAccountCapitalPool`, `ExposureTracker`, `PositionSizer`, `Propfirm.Manager`, and `KillSwitch` infrastructure. Eliminate duplicated risk logic between backtest and live paths.

---

## 1. Current Architecture Gap

```
                    ┌─────────────────────────────────────────────┐
                    │         SHARED PURE FUNCTIONS (OK)          │
                    │  propfirm/rules.go → DrawdownPct, DailyLoss │
                    │  PositionSizer → ComputeSize, ComputeSizeUncapped │
                    │  ExposureTracker → CheckOrder               │
                    └──────────────┬──────────────────────────────┘
                                   │
          ┌────────────────────────┼────────────────────────┐
          │                        │                        │
   ┌──────▼──────┐          ┌──────▼──────┐          ┌──────▼──────┐
   │ Backtest    │          │ Live        │          │ Live        │
   │ Engine      │          │ Engine      │          │ API Server  │
   │             │          │             │          │             │
   │ ftmo ✓      │          │ ftmo ✗      │          │ multiCapPool│
   │ exposure ✓  │          │ exposure ✗  │          │ propfirmMgr │
   │ positionSz ✓│          │ positionSz ✗│          │ killSwitch  │
   │ poolSim ✓   │          │ pool ✗      │          │ acctManager │
   └─────────────┘          └─────────────┘          └─────────────┘
                                                           ↑
                                                    WIRED TO API
                                                    NOT TO ENGINE
```

**The `LiveEngine` calls none of the risk infrastructure that exists in `internal/risk/`.** The API Server wires `MultiAccountCapitalPool`, `Propfirm.Manager`, `KillSwitch`, and `AccountManager` together — for administrative endpoints — but `ProcessTick` bypasses them all.

---

## 2. Abstraction Strategy: `RiskPipeline`

### 2.1 Core Insight

The backtest engine's `generateSignal()` and the live engine's `ProcessTick()` perform the same conceptual sequence of risk checks, just wired to different structs. The sequence is:

```
1. Capital check       → runningCapital > 0?
2. Volatility halt     → IsHalted?
3. Signal evaluation   → runner.Evaluate(candle, regime)
4. Sizing              → regime × seasonal × Kelly × confidence
5. Exposure check      → max leverage / symbol gross
6. Capital gate        → pool.RequestCapital (or inline balance check)
7. Position caps       → max open positions, max position %, correlation
8. Prop-firm rules     → daily loss, drawdown, consistency
9. Approval / Reject
```

### 2.2 Unified Interfaces

Create `internal/risk/pipeline.go` with three interfaces that both engines compose:

```go
// SignalGate validates a signal and applies sizing before capital authorization.
type SignalGate interface {
    ValidateSignal(signal *strategy.Signal, runningCapital float64) (ok bool, reason string)
    ApplySizing(signal *strategy.Signal, capital float64, confidence float64) float64
}

// CapitalGate authorizes capital and reconciles fills.
type CapitalGate interface {
    RequestCapital(ctx context.Context, req CapitalRequest) CapitalResult
    RecordFill(ctx context.Context, strategyID, symbol, side string, pnl, quantity float64)
    Halted() bool
    HaltReason() string
    ResetDaily()
}

// PropFirmGate enforces prop-firm rule compliance on every fill.
type PropFirmGate interface {
    CheckDailyLimits() (ok bool, reason string)
    OnFill(pnl float64)
    OnNewDay()
    IsHalted() bool
    MarkViolated(reason string)
}

// RiskPipeline composes all three gates and the shared ExposureTracker.
type RiskPipeline struct {
    SignalGate
    CapitalGate
    PropFirmGate
    Exposure *ExposureTracker
}
```

### 2.3 Concrete Implementations

| Interface | Backtest Impl | Live Impl | Shared Core |
|---|---|---|---|
| `SignalGate` | `backtestSignalGate` (wraps engine's volHalt + sizing logic) | `liveSignalGate` (wraps adversarial + ML gate) | `PositionSizer` for sizing math |
| `CapitalGate` | `CapitalPoolSim` (for multi-strat) / inline for single-strat | `CapitalPoolManager` (per-account) | `propfirm/rules.go` functions |
| `PropFirmGate` | `PropFirmEnforcer` | `propfirm.Manager` (already shared state/profile) | `propfirm/rules.go` + `propfirm.Profile`/`State` |

**Key principle:** Both `CapitalPoolSim` and `CapitalPoolManager` implement `CapitalGate`. Both `PropFirmEnforcer` and `propfirm.Manager` implement `PropFirmGate`. The `RiskPipeline` struct calls the same methods in the same order regardless of which concrete implementation is wired in.

---

## 3. Phase 1: Extract Shared `BaseCapitalPool`

**Goal:** Eliminate duplicated drawdown tracking, daily PnL accumulation, position counting, and correlation reduction between `CapitalPoolSim` (backtest) and `CapitalPoolManager` (live).

**File:** `internal/risk/base_pool.go` (new)

### 3.1 Shared Fields

```go
type BaseCapitalPool struct {
    Profile          *propfirm.Profile
    TotalBalance     float64
    TotalPeakBalance float64
    DailyPnL         float64
    DailyPnLPct      float64
    TradingDays      int
    Halted           bool
    HaltReason       string
    Strategies       map[string]*StrategyAllocation
}
```

### 3.2 Shared Methods (Moved from Both)

```go
func (b *BaseCapitalPool) DailyLossCheck() (ok bool, reason string)    // delegates to propfirm.DailyLossExceeded
func (b *BaseCapitalPool) DrawdownCheck() (ok bool, reason string)    // delegates to propfirm.DrawdownExceeded
func (b *BaseCapitalPool) PerStratDrawdownCheck(stratID string) (ok bool) // 50% of pool max DD
func (b *BaseCapitalPool) CountTotalOpenPositions() int               // sum across all strategies
func (b *BaseCapitalPool) ApplyCorrelationReduction(side, symbol string) float64 // 0.5x if opposing
func (b *BaseCapitalPool) RecordPnL(pnl float64)                      // updates balance, peak, dailyPnL
func (b *BaseCapitalPool) ResetDaily()
func (b *BaseCapitalPool) IsHalted() bool
```

### 3.3 Embedding

```go
// CapitalPoolSim becomes:
type CapitalPoolSim struct {
    *BaseCapitalPool            // embedded — inherits all shared methods
    mu sync.Mutex               // only for EvaluateAll's internal coordination
    // EvaluateAll stays unique (calls strategy runners directly)
}

// CapitalPoolManager becomes:
type CapitalPoolManager struct {
    *BaseCapitalPool            // embedded — inherits all shared methods
    mu sync.RWMutex             // thread-safe for live multi-account
    accountID string
    state *propfirm.State       // live-specific: carries phase tracking, violation flag
    positionSizer *PositionSizer
    // RequestCapital stays unique (called externally, not EvaluateAll)
}
```

**Lines saved:** Approximately 120 lines of duplicated drawdown/balance/position logic removed from `capital_pool_sim.go` and `capital_pool.go`.

---

## 4. Phase 2: Unify `PropFirmGate` Interface

**Goal:** `PropFirmEnforcer` (backtest) and `propfirm.Manager` (live) implement the same interface so the `RiskPipeline` can call them interchangeably.

### 4.1 Current State

| Method | `PropFirmEnforcer` (backtest) | `propfirm.Manager` (live) | Same? |
|---|---|---|---|
| Daily loss check | `CheckDailyLoss() bool` | `CheckDailyLimits() (bool, string)` | Different signature |
| Drawdown check | `CheckDrawdown() bool` | (inside CheckDailyLimits) | Compound vs separate |
| Consistency | `CheckConsistency() bool` | (inside DailyReset) | Different timing |
| OnFill | `OnFill(pnl float64)` | `RecordFill(pnl, balance)` | Different signatures |
| OnNewDay | `OnNewDay()` | `DailyReset()` | Different names |
| IsHalted | `IsHalted bool` (field) | `IsHalted() bool` | Field vs method |
| MarkViolated | Not present | `MarkViolated(reason)` | Missing in backtest |
| Halt reason | `HaltReason string` (field) | (inside State.ViolationReason) | Different location |
| Advance phase | `AdvancePhase()` | `AdvancePhase()` | Same |

### 4.2 Interface Definition

```go
type PropFirmGate interface {
    // Daily lifecycle
    OnNewDay()
    
    // Fill lifecycle
    OnFill(pnl float64, balance float64)
    
    // Queries
    CheckDailyLimits() (ok bool, reason string)
    IsHalted() bool
    HaltReason() string
    
    // Violation management
    MarkViolated(reason string)
    
    // Phase management
    AdvancePhase()
    CurrentPhase() int
    ProfitTargetMet() bool
    
    // Sizing
    RegimeMultiplier() float64
    GetPositionSize(baseQty float64) float64
}
```

### 4.3 Adapter Pattern

Rather than modifying `PropFirmEnforcer` and `propfirm.Manager` extensively (risks breaking existing backtest tests), add lightweight adapter methods where signatures diverge:

```go
// PropFirmEnforcer adapter (backtest)
func (f *PropFirmEnforcer) CheckDailyLimits() (bool, string) {
    if f.CheckDailyLoss() { return false, "daily_loss" }
    if f.CheckDrawdown() { return false, "max_drawdown" }
    return true, ""
}

// propfirm.Manager adapter (live) — already has the right signature
```

**Lines changed:** ~30 adapter methods added (no existing code modified).

---

## 5. Phase 3: Extract `RiskPipeline`

**File:** `internal/risk/pipeline.go` (new)

### 5.1 The Pipeline Struct

```go
type RiskPipeline struct {
    Pool      CapitalGate
    PropFirm  PropFirmGate
    Exposure  *ExposureTracker
    Sizer     *PositionSizer
    KellyMult float64
}
```

### 5.2 The Single Method Both Engines Call

```go
// ProcessSignal is the canonical signal → approval/rejection pipeline.
// Both Engine.generateSignal() and LiveEngine.ProcessTick() delegate to this.
func (p *RiskPipeline) ProcessSignal(ctx context.Context, req ProcessSignalRequest) ProcessSignalResult {
    // 1. Volatility halt
    if p.volHalt.IsHalted() {
        return rejected("volatility_halt")
    }
    
    // 2. Capital gate halt
    if p.Pool.Halted() {
        return rejected("pool_halted:" + p.Pool.HaltReason())
    }
    
    // 3. Sizing
    size := p.Sizer.ComputeSizeUncapped(req.Confidence, req.BaseSize, req.ExistingPosition)
    size *= p.KellyMult
    if size <= 0 {
        return rejected("size_zero")
    }
    
    // 4. Exposure check
    notional := size * req.Price
    if ok, reason := p.Exposure.CheckOrder(req.Symbol, req.Side, notional); !ok {
        return rejected("exposure:" + reason)
    }
    
    // 5. Capital authorization
    capitalReq := CapitalRequest{
        StrategyID: req.StrategyID,
        Confidence: req.Confidence,
        Symbol:     req.Symbol,
        Side:       req.Side,
        BaseSize:   size,
    }
    result := p.Pool.RequestCapital(ctx, capitalReq)
    if !result.Approved {
        return rejected("capital:" + result.Reason)
    }
    
    // 6. Approved
    return approved(result.ApprovedSize)
}

// ReconcileFill is the canonical fill → bookkeeping pipeline.
// Both Engine trade-close and LiveEngine SignalOutcome delegate to this.
func (p *RiskPipeline) ReconcileFill(strategyID, symbol, side string, pnl, qty float64) {
    p.Pool.RecordFill(context.Background(), strategyID, symbol, side, pnl, qty)
    if p.PropFirm != nil {
        p.PropFirm.OnFill(pnl, p.Pool.TotalBalance())
        if ok, reason := p.PropFirm.CheckDailyLimits(); !ok {
            p.PropFirm.MarkViolated(reason)
        }
    }
    p.Exposure.RemovePosition(symbol, side, qty)
}
```

### 5.3 Wiring in Engine (backtest)

```go
// Engine struct gains:
type Engine struct {
    // ... existing fields ...
    pipeline *risk.RiskPipeline  // replaces direct ftmo/exposure/sizer usage
}

// In NewEngine():
e.pipeline = &risk.RiskPipeline{
    Pool:     newSingleStrategyPool(),  // thin adapter for runningCapital
    PropFirm: e.ftmo,                   // PropFirmEnforcer implements PropFirmGate
    Exposure: e.exposure,
    Sizer:    e.positionSizer,
    KellyMult: e.kellyMult,
}

// generateSignal simplified:
func (e *Engine) generateSignal(...) *Signal {
    result := e.pipeline.ProcessSignal(ctx, ProcessSignalRequest{...})
    if !result.Approved {
        e.signalDiag.recordReject(result.Reason)
        return nil
    }
    // ... return signal with result.ApprovedSize ...
}
```

### 5.4 Wiring in LiveEngine

```go
// LiveEngine struct gains:
type LiveEngine struct {
    // ... existing fields ...
    pipeline *risk.RiskPipeline
    pools    *risk.MultiAccountCapitalPool
}

// In ProcessTick, after adversarial + ML gate:
func (e *LiveEngine) ProcessTick(...) []*strategy.Signal {
    // ... existing adversarial + ML gate ...
    
    for _, sig := range approved {
        result := e.pipeline.ProcessSignal(ctx, risk.ProcessSignalRequest{
            StrategyID:       sig.StrategyID,
            Symbol:           sig.Symbol,
            Side:             sig.Side,
            Price:            candle.Close.Float64(),
            Confidence:       sig.PWin,
            BaseSize:         sig.Quantity,
            ExistingPosition: e.openPositions[sig.Symbol],
        })
        if !result.Approved { continue }
        sig.Quantity = result.ApprovedSize
    }
    // ...
}
```

---

## 6. Phase 4: Per-Account Strategy Isolation

**Goal:** Change `LiveEngine` from shared singleton strategies to per-account isolated instances.

### 6.1 Current Problem

```go
// LiveEngine.ProcessTick:124 — uses SHARED singletons
signals := strategy.GlobalRegistry().EvaluateAll(goCandle, regimeInt8)
```

This means `GridRunner.openPositions`, indicator state, and z-score buffers are shared across all symbols and all potential accounts. A trade on AAPL updates state that affects the next evaluation for MSFT.

### 6.2 Fix: Per-Account Registry

```go
type LiveEngine struct {
    // ... existing ...
    accountRegistries map[string]*strategy.Registry  // one per account
    defaultRegistry   *strategy.Registry              // fallback for single-account
}

func (e *LiveEngine) ProcessTickForAccount(accountID string, symbolID uint32, ...) []*strategy.Signal {
    reg := e.accountRegistries[accountID]
    if reg == nil {
        reg = e.defaultRegistry
    }
    signals := reg.EvaluateAll(goCandle, regimeInt8)  // isolated instances
    // ...
}
```

Each account registry is populated via `registry.Create()` for each strategy type, using factory-created isolated instances (identical to the backtest approach). Per-account parameterization is applied via `SetParams()`.

### 6.3 Shared-Global Fallback

A `defaultRegistry` singleton is maintained for the single-account case (current behavior). Multi-account deployments explicitly create per-account registries. This preserves backward compatibility.

---

## 7. Phase 5: Kill-Switch ↔ Capital Pool Integration

**Goal:** When `KillSwitch.Trigger()` fires, it marks prop-firm violations and resets capital pool state.

### 7.1 Interface Extension

```go
// Add to KillSwitch
type CapitalPoolResetter interface {
    MarkAllViolated(reason string)
    ResetAllDaily()
}

// KillSwitch gains:
func (ks *KillSwitch) SetPoolResetter(r CapitalPoolResetter)
```

### 7.2 Implementation

```go
// MultiAccountCapitalPool implements CapitalPoolResetter
func (mc *MultiAccountCapitalPool) MarkAllViolated(reason string) {
    for _, pool := range mc.pools {
        pool.propfirmManager.MarkViolated(reason)
    }
}

// In KillSwitch.Trigger(), after order cancellation:
if ks.poolResetter != nil {
    ks.poolResetter.MarkAllViolated(reason)
}
```

### 7.3 Wiring in API Server

```go
// router.go NewServer setup:
s.killSwitch.SetPoolResetter(s.multiCapitalPool)
```

---

## 8. Implementation Sequence

### Wave 1 — Zero Risk (4 hours)

| Step | Task | File | Effort |
|------|------|------|--------|
| 1.1 | Define `SignalGate`, `CapitalGate`, `PropFirmGate` interfaces | `internal/risk/interfaces.go` (new) | 0.5h |
| 1.2 | Define `RiskPipeline` struct + `ProcessSignal`/`ReconcileFill` | `internal/risk/pipeline.go` (new) | 1h |
| 1.3 | Add adapter methods to `PropFirmEnforcer` (CheckDailyLimits, etc.) | `backtest/propfirm_enforcer.go` | 0.5h |
| 1.4 | Add adapter methods to `CapitalPoolSim` (implement CapitalGate) | `backtest/capital_pool_sim.go` | 0.5h |
| 1.5 | Add adapter methods to `CapitalPoolManager` (implement CapitalGate) | `risk/capital_pool.go` | 0.5h |
| 1.6 | Unit tests for RiskPipeline with mock gates | `risk/pipeline_test.go` (new) | 1h |

**Verification:** `go test ./internal/risk/...` — new pipeline tests pass. Existing backtest + engine tests still pass (no wiring changed yet).

### Wave 2 — Extract BaseCapitalPool (3 hours)

| Step | Task | File | Effort |
|------|------|------|--------|
| 2.1 | Create `BaseCapitalPool` with shared balance/DD/position methods | `internal/risk/base_pool.go` (new) | 1h |
| 2.2 | Refactor `CapitalPoolManager` to embed `BaseCapitalPool` | `risk/capital_pool.go` | 0.5h |
| 2.3 | Refactor `CapitalPoolSim` to embed `BaseCapitalPool` | `backtest/capital_pool_sim.go` | 0.5h |
| 2.4 | Run all backtest tests to verify refactoring doesn't change behavior | — | 1h |

**Verification:** `go test ./internal/backtest/... ./internal/risk/...` — all existing tests pass. No behavior change, only extraction.

### Wave 3 — Wire Backtest Engine to Pipeline (2 hours)

| Step | Task | File | Effort |
|------|------|------|--------|
| 3.1 | Add `pipeline` field to `Engine`, init in constructor | `backtest/engine.go` | 0.5h |
| 3.2 | Replace `generateSignal` direct calls with `pipeline.ProcessSignal` | `backtest/engine.go` | 0.5h |
| 3.3 | Replace trade-close bookkeeping with `pipeline.ReconcileFill` | `backtest/engine.go` | 0.5h |
| 3.4 | Run all backtest regression tests | — | 0.5h |

**Verification:** `go test ./internal/backtest/... -count=1 -v` — all 119 tests pass. Sharpe/MaxDD/WinRate unchanged (identical risk logic, new dispatch path).

### Wave 4 — Wire LiveEngine to Pipeline (4 hours)

| Step | Task | File | Effort |
|------|------|------|--------|
| 4.1 | Add `pipeline` and `pools` fields to `LiveEngine` | `engine/live_engine.go` | 0.5h |
| 4.2 | Create `NewLiveEngineWithRisk(pipeline, pools)` constructor | `engine/live_engine.go` | 0.5h |
| 4.3 | Wire `ProcessTick` to call `pipeline.ProcessSignal` per signal | `engine/live_engine.go` | 1h |
| 4.4 | Wire `SignalOutcome` to call `pipeline.ReconcileFill` | `engine/live_engine.go` | 0.5h |
| 4.5 | Add `OnNewDay` lifecycle call to pipeline | `engine/live_engine.go` | 0.5h |
| 4.6 | Add live engine risk integration tests (mock pool + prop firm) | `engine/live_risk_test.go` (new) | 1h |

**Verification:** New integration tests pass. `go test ./internal/engine/...`

### Wave 5 — API Server Wiring (2 hours)

| Step | Task | File | Effort |
|------|------|------|--------|
| 5.1 | Create `LiveEngine` with pipeline in server setup | `api/router.go` | 0.5h |
| 5.2 | Wire `KillSwitch` ↔ `MultiAccountCapitalPool` | `api/router.go`, `risk/kill_switch.go` | 0.5h |
| 5.3 | Ensure `propfirm.Manager.ActivateProfile` syncs state to engine pipeline | `api/router.go` | 0.5h |
| 5.4 | End-to-end test: deploy → tick → signal → capital gate → fill → violation | `api/live_e2e_test.go` (new) | 0.5h |

### Wave 6 — Per-Account Strategy Isolation (3 hours)

| Step | Task | File | Effort |
|------|------|------|--------|
| 6.1 | Add `accountRegistries` map to `LiveEngine` | `engine/live_engine.go` | 0.5h |
| 6.2 | Add `RegisterAccountStrategies(accountID string, params map[string]map[string]float64)` | `engine/live_engine.go` | 0.5h |
| 6.3 | Modify `ProcessTick` to route per-account | `engine/live_engine.go` | 1h |
| 6.4 | Wire into `AccountManager` account registration flow | `broker/account_manager.go` | 0.5h |
| 6.5 | Test: two accounts, same strategy, different params, isolated state | `engine/live_risk_test.go` | 0.5h |

---

## 9. Code Drift Prevention

### 9.1 Single Source of Truth for Risk Math

```
propfirm/rules.go ← ALL formulas live here (already done)
    ↑                       ↑
PropFirmEnforcer    propfirm.Manager
    ↑                       ↑
BaseCapitalPool (embedded by both pool implementations)
    ↑                       ↑
CapitalPoolSim      CapitalPoolManager
```

### 9.2 Ordered Risk Pipeline Prevents Reordering Bugs

The `RiskPipeline.ProcessSignal` method enforces the canonical check order. Neither engine can reorder or skip checks — they call the single pipeline method. Adding a new risk check requires adding it in one place (the pipeline), and both engines inherit it.

### 9.3 Interface Compliance Verified at Compile Time

```go
var _ CapitalGate = (*CapitalPoolSim)(nil)
var _ CapitalGate = (*CapitalPoolManager)(nil)
var _ PropFirmGate = (*PropFirmEnforcer)(nil)
var _ PropFirmGate = (*propfirm.Manager)(nil)
```

### 9.4 Shared Test Fixtures

```go
// risk/pipeline_test.go
func TestRiskPipeline_Integration(t *testing.T) {
    // Same test verifies both backtest and live paths:
    t.Run("backtest path", func(t *testing.T) {
        pool := &CapitalPoolSim{...}
        propFirm := &PropFirmEnforcer{...}
        pipeline := newPipeline(pool, propFirm)
        // ... asserts ...
    })
    t.Run("live path", func(t *testing.T) {
        pool := &CapitalPoolManager{...}
        propFirm := &propfirm.Manager{...}
        pipeline := newPipeline(pool, propFirm)
        // ... identical asserts ...
    })
}
```

---

## 10. Rollback Safety

Each wave is independently testable and reversible:

- **Waves 1-2:** Add new files, add adapter methods — zero existing code modified, zero behavior change.
- **Wave 3:** Wires backtest engine through pipeline — if Sharpe/MaxDD regress, revert the `generateSignal` changes (3 lines).
- **Wave 4:** Wires live engine through pipeline — if signals stop flowing, the old `ProcessTick` path is preserved behind a feature flag (`if e.pipeline != nil`).
- **Waves 5-6:** Server wiring and per-account isolation — independent of risk pipeline, can be deployed separately.

**Total estimated effort:** 18 hours across 6 waves, with each wave independently testable and revertible.

---

## 11. Verification Matrix

| What to Verify | How | Wave |
|---|---|---|
| Backtest metrics unchanged (Sharpe, MaxDD, WR, PF) | Run `go test ./internal/backtest/...` | 3 |
| CapitalPoolSim behavior identical after BasePool extraction | Compare test output before/after Wave 2 | 2 |
| Live engine signals still flow with pipeline optional | `ProcessTick` with `nil` pipeline → old behavior | 4 |
| Pipeline rejects correctly on drawdown breach | Inject mock pool returning `Halted=true` | 4 |
| Per-account strategy isolation | Two accounts, same strategy, different params, different signal output | 6 |
| Kill-switch marks violations | Trigger KillSwitch → assert `propfirm.Manager.State().Violated == true` | 5 |
| Code drift prevention | New risk rule added to pipeline → both engines inherit automatically | Ongoing |
