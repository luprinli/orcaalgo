# OrcaAlgo — Unified Best Practice Specifications Document

**Version:** 1.0.0  
**Date:** 2026-06-11  
**Status:** Governing Reference for All Development Cycles  
**Source Research:** 8-project exhaustive technical audit of `.research/` directory  

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Architectural Standards](#2-architectural-standards)
3. [Mathematical & Statistical Foundations](#3-mathematical--statistical-foundations)
4. [Risk Management Framework](#4-risk-management-framework)
5. [Signal Generation & Strategy Representation](#5-signal-generation--strategy-representation)
6. [Execution & Order Management](#6-execution--order-management)
7. [Data Flow, Persistence & Audit](#7-data-flow-persistence--audit)
8. [Configuration & Parameter Governance](#8-configuration--parameter-governance)
9. [Testing, Validation & Calibration](#9-testing-validation--calibration)
10. [Security, Governance & Profile Gating](#10-security-governance--profile-gating)
11. [UI, Monitoring & Observability](#11-ui-monitoring--observability)
12. [Implementation Roadmap](#12-implementation-roadmap)
13. [Comparative Methodology Matrix](#13-comparative-methodology-matrix)
14. [Antipatterns & Prohibitions](#14-antipatterns--prohibitions)

---

## 1. Executive Summary

This document codifies the **unified best practices** extracted from an exhaustive technical audit of 8 quantitative trading and risk management research projects. It establishes mandatory architecture, mathematical, and operational standards that govern all future development cycles of the OrcaAlgo application.

### 1.1 Research Projects Audited

| # | Project | Language | Domain | Quality Tier |
|---|---------|----------|--------|-------------|
| 1 | **go-trader-main** | Go + Rust | Algo trading platform (Alpaca + Claude AI) | Tier 1 — Production Reference |
| 2 | **go-trader-master** | Go | Exchange simulation (CLOB matching engine) | Tier 1 — Algorithmic Reference |
| 3 | **MT5-PropFirm-Drawdown-Guard** | C# WinForms | Prop firm drawdown guard (pipe → EA) | Tier 2 — Partial Implementation |
| 4 | **MT5-PropFirm-MultiPass-Dashboard** | C# WinForms | Stealth SL/TP manager (deceptive repo) | Tier 3 — Reject, Extract Warnings |
| 5 | **Prop-Matrix-Engine** | C# WinForms | Multi-account risk command center | Tier 2 — Strong Architecture, Incomplete |
| 6 | **PropForge** | JavaScript (vanilla) | Prop firm trading simulator (browser) | Tier 2 — Excellent Simulation Patterns |
| 7 | **Quant-Strategy-Tokenizer** | Python (Pydantic) | Strategy IR & governance system | Tier 1 — Governance Reference |
| 8 | **Quant-toolkit** | Python | Prediction market quant toolkit (9 skills) | Tier 1 — Mathematical Reference |

### 1.2 Synthesis Methodology

Each project was audited across 9 dimensions: Overview, Architecture, Mathematical Models, Implementation Patterns, Algorithms, Data Flow, Configuration, Dependencies, Strengths/Weaknesses. Cross-cutting themes were extracted, compared, and consolidated into the specifications below.

---

## 2. Architectural Standards

### 2.1 Mandatory Architectural Principles

#### 2.1.1 Modular Component Separation (ALL TIER-1 PROJECTS)

OrcaAlgo **MUST** maintain strict separation of concerns across the following bounded contexts:

```
┌──────────────────────────────────────────────────────────────┐
│  BOUNDED CONTEXT MAP (Mandatory)                             │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐ │
│  │ Strategy │   │  Signal  │   │   Risk   │   │Execution │ │
│  │   IR     │──▶│Generator │──▶│  Engine  │──▶│  Engine  │ │
│  │(Tokenizer)│   │(Claude/  │   │(Drawdown │   │(Alpaca/  │ │
│  │          │   │ Local ML)│   │ Guard)   │   │ MT5/BTC) │ │
│  └──────────┘   └──────────┘   └─────┬────┘   └────┬─────┘ │
│                                      │              │       │
│                              ┌───────▼──────┐       │       │
│                              │  Position    │◀──────┘       │
│                              │  Sizer       │               │
│                              │ (Kelly/Vol)  │               │
│                              └──────┬───────┘               │
│                                     │                       │
│  ┌──────────┐   ┌──────────┐   ┌───▼──────┐   ┌──────────┐ │
│  │  Market  │   │   Audit  │   │Portfolio │   │Notification│ │
│  │   Data   │──▶│   Log    │──▶│  State   │──▶│  System   │ │
│  │(Ticker/  │   │(SQLite/  │   │(In-Memory│   │(Ring Buf/ │ │
│  │ Stream)  │   │Append-O) │   │+Persist) │   │ WebSocket)│ │
│  └──────────┘   └──────────┘   └──────────┘   └──────────┘ │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

**Sources:** go-trader-main (module separation), Quant-Strategy-Tokenizer (IR boundary), Prop-Matrix-Engine (domain model isolation)

#### 2.1.2 Immutable Domain Models (Prop-Matrix-Engine, Quant-Strategy-Tokenizer)

All domain types **MUST** be immutable. Use:
- **Python:** Pydantic v2 `BaseModel` with `ConfigDict(frozen=True)`, `extra="forbid"`
- **Go:** Exported structs with unexported fields, constructor-only initialization, no setters
- **C#:** `sealed record` or `readonly struct`
- **TypeScript/JS:** `Object.freeze()` or `Readonly<T>` with `immer` for updates

```python
# MANDATORY PATTERN (from Quant-Strategy-Tokenizer)
class TradeSignal(BaseModel):
    model_config = ConfigDict(frozen=True, extra="forbid")
    symbol: str
    signal: Literal["BUY", "SELL", "HOLD"]
    confidence: float = Field(ge=0.0, le=1.0)
    timestamp: datetime
```

#### 2.1.3 Event-Driven Architecture with Thread Safety (ALL PROJECTS)

All inter-component communication **MUST** use events/callbacks, not direct method calls:

| Language | Mechanism |
|----------|-----------|
| Python | `asyncio.Queue`, callback registration, or `RxPY` |
| Go | Channel-based pub/sub with `RWMutex` |
| C# | `event` delegates with `BeginInvoke` marshaling to UI thread |
| Rust | `tokio::sync::broadcast` channels |

**UI Thread Safety Rule:** ALL background thread/async-task updates to UI **MUST** marshal through the UI thread (Go: no issue, single-threaded; C#: `BeginInvoke`; JS: already single-threaded; Python: `asyncio.call_soon_threadsafe`).

**Source:** Prop-Matrix-Engine `Post()` method pattern, go-trader-main `RegisterSignalCallback`

#### 2.1.4 Dual-Implementation Strategy for Critical Paths (go-trader-main)

For production-critical subsystems (trade execution, audit logging, market data ingestion), **SHOULD** implement in both:
- **Go/Python** — Rapid development, API surface, glue code
- **Rust** — Production-grade reliability, async IO, zero-cost abstractions, SQLite audit

This is not mandatory but is the Tier-1 reference pattern.

### 2.2 Prohibited Architectural Patterns

| Antipattern | Source | Rationale |
|-------------|--------|-----------|
| Global mutable state | go-trader-main `globalStates` | Race conditions, untestable |
| Single monolithic file >1000 LOC | PropForge 1517-line `script.js` | Unmaintainable |
| `panic()` / `throw` for recoverable errors | go-trader-master exchange | Crash-only is unacceptable for trading |
| Inline Kelly reimplementation diverging from canonical | Quant-toolkit `backtest_template.py` | Source of truth fragmentation |
| Direct DOM mutation without diffing | — | Performance degradation |

### 2.3 Factory + Plugin Architecture (go-trader-main, go-trader-master)

All algorithmic components **MUST** use a factory registry pattern:

```go
// MANDATORY PATTERN (from go-trader-main)
type Algorithm interface {
    Process(symbol string, data MarketData, historical []Bar) AlgorithmResult
}

var registry = map[string]func() Algorithm{}

func Register(name string, factory func() Algorithm) {
    registry[name] = factory
}

func Create(name string) Algorithm {
    return registry[name]()
}
```

In Python, use `importlib.metadata` entry points. In C#, use DI container registration.

---

## 3. Mathematical & Statistical Foundations

### 3.1 Universal Position Sizing: Kelly Criterion Family

#### 3.1.1 Canonical Kelly Formula (Binary Outcome Contracts)

```
For YES (long):  f* = (p - c) / (1 - c)
For NO (short):  f* = ((1-p) - (1-c)) / c
```

Where `p` = estimated probability of YES resolving true, `c` = contract price.

**Source:** Quant-toolkit `kelly.py:30-49` (canonical reference implementation)

#### 3.1.2 Kelly for Continuous Return Assets (go-trader-main variant)

```
f* = (p_win * b - (1 - p_win)) / b

where b = win_loss_ratio (default 1.0)
      p_win = confidence from meta-labeling model
```

**Source:** go-trader-main `position_sizing.go:236-318`

#### 3.1.3 MANDATORY Risk Attenuators (Quant-toolkit Tier-1 reference)

Every position sizing calculation **MUST** apply three attenuators in sequence:

```
STEP 1: Edge Uncertainty Discount
    p_discounted = max(p - edge_discount, 0.0)   # YES side
    Default: δ = 0.02 (well-calibrated) or 0.05 (new model)

STEP 2: Fractional Kelly Multiplier
    f_fractional = f* × k
    MANDATORY: k = 0.25 (quarter-Kelly)
    NEVER: k = 1.0 (full Kelly) in production

STEP 3: Hard Exposure Caps
    per_trade_cap = min(f_fractional, 0.02)        # max 2% bankroll per trade
    total_exposure_cap = min(per_trade_cap, 0.30 - current_exposure_pct)  # max 30% total
    Final allocation = min(per_trade_cap, total_exposure_cap)
```

**Source:** Quant-toolkit `kelly.py:74-96`

#### 3.1.4 Volatility-Adjusted Position Sizing (go-trader-main)

For continuous assets:

```
vol_adjustment = clamp(kelly_fraction / (σ / baseline_σ), 0.5, 2.0)
final_size = min(max_size, kelly_fraction * vol_adjustment)
risk_per_trade = final_size * σ
```

Where σ is EWMA volatility: `α = 2/(span+1)`, `EWMVar_t = α·r_t² + (1-α)·EWMVar_{t-1}`

**Source:** go-trader-main `position_sizing.go`, `triple_barrier.go:241-276`

#### 3.1.5 Diversification Scaling (go-trader-main)

For multi-asset portfolios:

```
effective_n = num_positions * (1 - avg_correlation) + avg_correlation
diversification_scaling = clamp(1 / sqrt(effective_n), 0.25, 1.0)
```

**Source:** go-trader-main `position_sizing.go:335-354`

### 3.2 Calibration & Probability Assessment

#### 3.2.1 Brier Score Decomposition (Murphy 1973)

**MANDATORY** for all probability-emitting models:

```
Brier = (1/n) · Σ(p_i - o_i)²

Murphy Decomposition:
    Brier = Reliability - Resolution + Uncertainty

    Reliability = Σ(n_k/n) · (p̄_k - ō_k)²     [→ 0 is perfect]
    Resolution  = Σ(n_k/n) · (ō_k - ō)²        [→ max is best]
    Uncertainty = ō · (1 - ō)                  [constant for dataset]
```

Binning: 10 equal-width bins (0.0–0.1, …, 0.9–1.0), minimum 20 observations per bin for reliability.

**Source:** Quant-toolkit `calibration_audit.py:43-95`

#### 3.2.2 Platt Scaling (Probability Calibration)

When a model emits raw scores (not calibrated probabilities):

```
calibrated_p = σ(A · logit(raw_p) + B)

where σ(x) = 1/(1+e^{-x})
      logit(p) = ln(p/(1-p))

Fit A, B via MLE (negative log-likelihood minimization):
    L(A,B) = -(1/n)·Σ[y_i·ln(σ_i) + (1-y_i)·ln(1-σ_i)]
```

**Fit must use Nelder-Mead** (derivative-free, bounded).  
**Must use train/validation split** with minimum 200 observations per cohort.  
**Must reject fit** if validation Brier does not improve by ≥5% vs raw.

**Source:** Quant-toolkit `platt_fit.py:35-59`

#### 3.2.3 Wilson Confidence Intervals (Wilson 1927)

For any success-rate metric reported to users:

```
z = z-score for confidence level (1.96 for 95%)
denom = 1 + z²/n
center = (p + z²/(2n)) / denom
half_width = z · sqrt(p(1-p)/n + z²/(4n²)) / denom
CI = [max(center - half_width, 0), min(center + half_width, 1)]
```

**MUST flag "insufficient data"** when n < 30.

**Source:** Quant-toolkit `attribute.py:45-52`

### 3.3 Volatility Modeling

#### 3.3.1 EWMA Volatility (Lopez de Prado Standard)

```
α = 2 / (span + 1)           # default span = 20
EWMA_t = α · r_t + (1-α) · EWMA_{t-1}
EWMVar_t = α · (r_t - EWMA_t)² + (1-α) · EWMVar_{t-1}
σ_t = sqrt(EWMVar_t)

where r_t = ln(P_t / P_{t-1})    # log return
```

**Source:** go-trader-main `triple_barrier.go:241-276`

#### 3.3.2 Fractional Differentiation for Stationarity (Lopez de Prado Snippets 5.1/5.4/5.5)

```
w[0] = 1
w[k] = -w[k-1] · (d - k + 1) / k    for k ≥ 1

Fixed-width FFD (recommended for production):
    result[i] = Σ_{j=0}^{W-1} w[j] · series[i-j]

Optimal d found by minimizing |ADF statistic| or maximizing stationarity.
default_d = 0.4, default_W = 20, default_threshold = 1e-4
```

**Source:** go-trader-main `algo/fractional_diff.go`

### 3.4 Triple Barrier Labeling Method (Lopez de Prado Snippet 3.2/3.3)

**MANDATORY** for supervised learning label generation:

```
Upper Barrier: entryPrice · (1 + profitTaking · σ)
Lower Barrier: entryPrice · (1 - stopLoss · σ)
Time Barrier:  entryTime + timeHorizon

Label = +1 if upper hit first (confirm)
        -1 if lower hit first (reversal)
        determined by P_exit vs P_entry if time barrier hit

Default parameters:
    profitTaking  = 2.0 (multiples of σ)
    stopLoss      = 1.0
    timeHorizon   = 5 days
```

**Source:** go-trader-main `algo/triple_barrier.go:280-387`

### 3.5 CUSUM Structural Break Detection

```
S⁺_t = max(0, S⁺_{t-1} + z_t - drift)
S⁻_t = max(0, S⁻_{t-1} - z_t - drift)

where z_t = (r_t - μ) / σ    # standardized return

Signal: S⁺ > threshold → BUY regime change
        S⁻ > threshold → SELL regime change
```

**Source:** go-trader-main `algo/cusum_filter.go:102-114`

### 3.6 Economic Cartography Macro-Regime Model (go-trader-main)

```
Y(t) = Σ A_n · sin(2π · t/P_n + φ_n) + ε(t)

Bands:
    Kondratiev    P=52.0yr  A=1.10  φ=0.25
    Kuznets       P=22.0yr  A=0.70  φ=0.36
    Juglar        P=11.5yr  A=0.90  φ=0.01
    Kitchin       P=4.0yr   A=0.35  φ=0.00
    ε(t) = 0.10·sin(7.31t)·cos(2.73t) + 0.06·sin(13.11t)

Regime Classification:
    CONSTRUCTIVE:  Y(t) > +1.4        Multiplier = 1.10
    DESTRUCTIVE:   Y(t) < -1.4        Multiplier = 0.30
    RISING WATERS: dY/dt > +0.05      Multiplier = 1.00
    EBBING TIDE:   dY/dt < -0.05      Multiplier = 0.60
    CROSSWINDS:    otherwise          Multiplier = 0.80
```

**Applied Multiplier = min(Formula Multiplier, Data Multiplier)** — conservative: data can shrink risk but never expand it optimistically.

**Source:** go-trader-main `cartography/cartography.go`

### 3.7 FRED Macro Overlay Signals

| Signal | Condition | Haircut |
|--------|-----------|---------|
| Sahm Rule Recession | 3MMA(UNRATE) - min_12m > 0.50 | 0.40 |
| HY Credit Spread | OAS ≥ 7% | 0.50 |
| Yield Curve Inversion | 10Y-2Y < 0 | 0.85 |
| NFCI Tightening | NFCI > 0 | 0.85 |

Composed multiplier = **product** of all triggered haircuts.

**Source:** go-trader-main `cartography/fred.go`, `cartography/feed.go`

### 3.8 Decision Algebra (Quant-Strategy-Tokenizer)

For combining multiple signals into a single trading decision, **MUST** use formal monoids:

```
True Monoids (associative, with identity element):

unknown_propagating_and:
    accept(0) < unknown(1) < reject(2) < block(3)
    combine(a,b) = max_kind(a, b) over this priority order

any_accept:
    reject(0) < unknown(1) < accept(2) < block(3)

Aggregators (non-monoid):

weighted_vote:
    margin = Σ(mode_i · weight_i · score_i)
    where accept=+1, reject=-1
    result = accept if margin > 0, reject if margin < 0, unknown if margin == 0

majority:
    result = accept if #accepts > #rejects else reject if #rejects > #accepts else unknown

quorum(p, m):
    result = accept if #accepts ≥ p AND #known ≥ m else unknown
```

**Source:** Quant-Strategy-Tokenizer `qst/decision/reference.py`

### 3.9 Drawdown Mathematics

#### 3.9.1 Trailing High-Water-Mark Drawdown (MANDATORY for prop-firm compliance)

```
HWM(t) = max_{s ≤ t} balance(s)
DD(t) = (HWM(t) - equity(t)) / HWM(t)    # positive fraction, reported as percentage

where equity(t) = balance(t) + unrealized_PnL(t)
```

**Critical:** This is a TRAILING drawdown, not static. The HWM advances when equity reaches new highs. This is how prop firms (FTMO, MFF, TopStep) define drawdown.

**Source:** Quant-toolkit `drawdown.py:61-105`, PropForge trailing DD implementation

#### 3.9.2 Multi-Tier Drawdown Response Protocol

```
Level 0: DD < warn_pct (default 5%)     → "CLEAR" — Normal operations
Level 1: DD ≥ warn_pct                  → "WARN" — Notification only
Level 2: DD ≥ derisk_pct (default 10%)  → "DERISK" — Halve Kelly multiplier, raise edge threshold +2pp
Level 3: DD ≥ halt_pct (default 20%)    → "HALT" — Close all positions, cancel orders, require manual restart
```

Threshold calibration heuristic: `halt_pct ≈ 1.5 × backtest_MDD`

**Source:** Quant-toolkit `drawdown.py` and `SKILL.md`

#### 3.9.3 Composite Drawdown (Realized + Unrealized)

```
totalLoss = startingBalance - (balance + floatingPnL)
breach = totalLoss >= maxTotalLoss
```

**MUST** include unrealized PnL in drawdown calculations. Static balance-only drawdown is insufficient.

**Source:** PropForge trailing DD, Prop-Matrix-Engine `RiskEngine.Evaluate()`

---

## 4. Risk Management Framework

### 4.1 The Triple-Safety-Net Architecture (MANDATORY)

Every trading operation **MUST** pass through three independent risk layers:

```
                         ┌─────────────────────┐
   Signal In ──────────▶│ LAYER 1: Pre-Trade   │
                         │ - Position sizing    │
                         │ - Kelly + attenuators│
                         │ - Margin check       │
                         │ - Lot limit check    │
                         │ - Regime multiplier  │
                         └──────────┬──────────┘
                                    │ Approved
                                    ▼
                         ┌─────────────────────┐
                         │ LAYER 2: In-Trade    │
                         │ - Stop-loss distance │
                         │ - Take-profit target │
                         │ - Max position time  │
                         │ - Trailing stop      │
                         └──────────┬──────────┘
                                    │
                                    ▼
                         ┌─────────────────────┐
                         │ LAYER 3: Circuit     │
                         │ Breakers (Always On) │
                         │ - Daily DD guard     │
                         │ - Total DD guard     │
                         │ - Max positions/day  │
                         │ - Kill-switch latch  │
                         └─────────────────────┘
```

**Source:** go-trader-main (all three layers), Prop-Matrix-Engine (Layer 3), Quant-toolkit (Layer 1), PropForge (all three layers)

### 4.2 Kill-Switch Specification

#### 4.2.1 Trigger Protocol

```
IF any breach condition:
    1. Set _isLocked = true (one-shot latch, idempotent)
    2. Log breach reason with full context (time, values, thresholds)
    3. Send KILL command to execution engine
    4. Execute close-all-positions sequence:
       a. Market-close all open positions (parallel per account)
       b. Cancel all pending/resting orders
       c. Disable new order submission
    5. Lock UI controls (disable trade buttons)
    6. Send alert notification (critical priority)
    7. Persist kill-switch state to prevent auto-recovery on restart
```

#### 4.2.2 Kill-Switch Re-entrancy Guard (MANDATORY)

```
class RiskEngine:
    _killSwitchInFlight: bool = False
    
    def execute_kill_switch(self, reason: str):
        if self._isLocked or self._killSwitchInFlight:
            return  # Idempotent
        self._killSwitchInFlight = True
        try:
            # ... execution ...
        finally:
            self._killSwitchInFlight = False
```

**Source:** Prop-Matrix-Engine `_killSwitchInFlight` guard, MT5-Drawdown-Guard `_isLocked` latch

#### 4.2.3 Breach Conditions (Composite OR Gate)

A breach **MUST** be triggered if ANY of:

```
breach_daily_dd    = (dailyDrawdownPct >= maxDailyDdPct)           # default: 5%
breach_absolute_dd = (currentEquity <= startBalance - maxAbsDd)    # default: $1,000
breach_pos_count   = (openPositions > maxOpenPositions)            # default: 5
breach_trade_count = (trades_today >= maxTradesPerDay)             # default: 10
breach_regime      = (regimeMultiplier <= 0.30)                    # DESTRUCTIVE regime
```

**Source:** MT5-Drawdown-Guard, go-trader-main `algorithm.go:99-105`, Prop-Matrix-Engine

### 4.3 Pre-Breach Buffer (Prop-Matrix-Engine Innovation)

```
triggerAt = max(0, drawdownLimitPercent - preBreachBufferPercent)

Example: DD limit = 5.0%, buffer = 0.5% → trigger at 4.5%
```

This "soft landing" gives the system time to execute the kill-switch BEFORE the formal limit is breached.

**Source:** Prop-Matrix-Engine `RiskEngine.Evaluate()` at `Engines.cs:188-226`

### 4.4 Risk Parameter Defaults (Consolidated from All Projects)

| Parameter | Default | Rationale |
|-----------|---------|-----------|
| Max position size (% portfolio) | 5% | go-trader-main standard |
| Max daily drawdown | 5% | Prop firm standard |
| Max absolute drawdown | $1,000 or 10% AUM | whichever is smaller |
| Max trades per day | 10 | Prevent overtrading |
| Stop loss (% from entry) | 5% | go-trader-main default |
| Take profit (% from entry) | 15% | 3:1 reward-to-risk ratio |
| Fractional Kelly multiplier | 0.25 | Quant-toolkit mandatory |
| Per-trade bankroll cap | 2% | Quant-toolkit hard cap |
| Total exposure cap | 30% | Quant-toolkit hard cap |
| Edge uncertainty discount | 0.02–0.05 | Calibrated vs exploratory |
| Max open positions | 5 | Prevent over-diversification |
| Warning threshold (% of DD limit) | 75% | MT5-Drawdown-Guard standard |

---

## 5. Signal Generation & Strategy Representation

### 5.1 Strategy Intermediate Representation (Quant-Strategy-Tokenizer MANDATORY)

All strategies **MUST** be represented as typed, versioned, hash-bearing Graph Kernel Records (`.gkr.yaml`):

```yaml
# MANDATORY GKR Structure
ir_version: "qst-ir/0.4"
canonical_version: "qst-canonical/0.4"
strategy:
  id: "orca-momentum-v1"
  version: "1.0.0"
  nodes:
    - id: "signal_gen"
      token_ref:
        token_id: "math.ema_crossover"
        version: ">=1.0,<2.0"
      params:
        fast_period: 12
        slow_period: 26
    - id: "risk_filter"
      token_ref:
        token_id: "risk.regime_gate"
        version: ">=1.0"
      params:
        min_multiplier: 0.3
    - id: "position_sizer"
      token_ref:
        token_id: "size.kelly_fractional"
        version: ">=1.0"
      params:
        multiplier: 0.25
        per_trade_cap: 0.02
  outputs:
    signal: "signal_gen.signal"
    size: "position_sizer.contracts"
```

#### 5.1.1 Three-Layer Cryptographic Hashing (MANDATORY)

```
graph_hash    = SHA256(graph_structure_without_params)
param_hash    = SHA256(params_only)
instance_hash = SHA256(graph_hash + param_hash)
```

**All hashes use canonical JSON** (sorted keys, compact separators, no NaN/Inf, UTF-8).

**Source:** Quant-Strategy-Tokenizer `qst/hash/`, `qst/canonical_json.py`

#### 5.1.2 Temporal Contract Validation (MANDATORY)

Every node output port **MUST** declare temporal properties:
- `available_at`: `bar_open(0)` | `bar_close(1)` | `next_bar_open(2)` | `unknown(3)`
- `latency_bars`: integer ≥ 0
- `min_history_bars`: integer ≥ 0
- `unsafe_future`: boolean

System **MUST** validate that no node uses data from the future (look-ahead bias prevention).

**Source:** Quant-Strategy-Tokenizer `qst/ports/temporal.py:27-79`

### 5.2 Signal Generation Approaches

#### 5.2.1 AI-Assisted Signals (go-trader-main reference)

```
Claude AI → ParseClaudeResponse() → TradeSignal {symbol, signal, confidence}
    ├── Cache with 10-min TTL
    ├── Fallback to local rule engine if AI unavailable
    └── Confidence must ≥ 0.65 for execution (Rust), ≥ 0.60 (Go)
```

#### 5.2.2 Rule-Engine Signals (go-trader-main Rust, PropForge)

Compute technical indicators from bar buffer (minimum 200 bars warm-up):
- SMA(20), SMA(50), SMA(200)
- RSI(14) `= 100 - 100/(1 + avgGain/avgLoss)`
- MACD `= EMA12 - EMA26`
- ATR(14), ADX(14)
- Bollinger Bands(20, 2): `SMA ± 2σ`

Signal cascading: apply rules sequentially with confidence scoring.

#### 5.2.3 Meta-Labeling Feature Extraction (go-trader-main)

```
Features:
    1-day price change, 5-day momentum
    Volume ratio = current_volume / avg_volume_5d
    Historical volatility (log returns)
    RSI(14), MACD, Bollinger %B

Normalization: clamp((val - min)/(max - min), 0, 1)

Model: P(trade) = σ(Σ w_i · f_i + bias)
    w = [0.2, 0.2, 0.3, 0.3]
    bias = -0.1
    threshold = 0.6

Kelly-inspired sizing from confidence:
    size = max(0.1, min(1.0, (confidence - (1-confidence)) * 0.5))
```

**Source:** go-trader-main `algo/meta_labeling.go:259-401`

#### 5.2.4 Multi-Algorithm Voting (go-trader-main)

When multiple signal generators produce outputs:

```
For each algorithm result:
    signalWeights[result.signal] += result.confidence * result.weight

combined_signal = argmax(signalWeights)

Tie-breaking: if |totalBuy - totalSell| < 0.2 * totalConfidence → HOLD
```

**Source:** go-trader-main `algo/algorithm_manager.go:175-282`

### 5.3 Market Scanner Pipeline (Quant-toolkit Tier-1)

For discovering tradeable candidates, use the mandatory 6-filter pipeline:

```
ALL OPEN MARKETS
    │
    ▼
[1] Freshness Gate: last_trade_ts < 30 min ago  ──→ "stale" (discard)
    │
    ▼
[2] Extreme Price Guard: 0.05 < price < 0.95     ──→ "extreme-price" (discard)
    │
    ▼
[3] Model Agreement: model_spread ≤ 0.15          ──→ "model-disagreement" (discard)
    │
    ▼
[4] Edge Threshold: edge_yes ≥ 0.08 OR edge_no ≥ 0.06 ──→ "edge-below-threshold"
    │
    ▼
[5] Side Cost Floor: fill_price ≥ 0.30            ──→ "side-cost-floor" (discard)
    │
    ▼
[6] Maker Feasibility:  spread ≥ 2·tick  AND  limit ∈ [0.01, 0.99]
                                                    ──→ "spread-too-tight"
    │
    ▼
  Calculate net_roi = edge/fill_price - fee% - adv_sel%
  if net_roi ≤ 0 → "net-roi-negative" (discard)
    │
    ▼
  CANDIDATES sorted by net_roi descending → top N
```

**Source:** Quant-toolkit `scanner_template.py:61-139`

---

## 6. Execution & Order Management

### 6.1 Order Lifecycle (go-trader-master reference)

```
CREATED → BOOKED → PARTIAL_FILL → FILLED
                                  → CANCELLED
                                  → REJECTED
```

All state transitions **MUST** be atomic and idempotent. Execution reports **MUST** flow back to the strategy layer.

### 6.2 Price-Time Priority Matching (go-trader-master)

```
1. Price priority: Bids descend, asks ascend
2. Time priority: Within same price level, FIFO (earlier orders match first)
3. Resting order's price used for trade: min(bid.time, ask.time)
4. Market orders match at best available price, then cancel unfilled
```

### 6.3 Maker Pricing Logic (Quant-toolkit MANDATORY for limit orders)

```
For YES-limit buy (provide liquidity):
    limit_price = ask - tick
    breakeven_prob = limit_price + maker_fee + adverse_selection_haircut

For NO-limit buy:
    sell_limit_yes = bid + tick
    limit_no = 1 - sell_limit_yes
    breakeven_prob = limit_no + maker_fee + adverse_selection_haircut

Constraint: limit_price ∈ [0.01, 0.99]
           spread ≥ 2 · tick  (otherwise no maker opportunity)
```

**Source:** Quant-toolkit `maker_quote.py:29-89`

### 6.4 Fill Simulation for Backtesting (Quant-toolkit)

```
target_price = ask - tick  (YES side)
if target_price ≤ bid: no fill (crosses spread)
fill_probability_at_one_tick = 0.70 (default)
effective_contracts = round(n · fill_probability)
```

**DO NOT** assume perfect fills at mid-price. Spread and tick-queue position matter.

**Source:** Quant-toolkit `backtest_template.py:73-91`

### 6.5 Position Scaling & Reduction Logic (PropForge)

#### Weighted Average Entry on Same-Side Scaling

```
totalQty = existingQty + newQty
newEntry = (existingEntry · existingQty + newEntry · newQty) / totalQty
```

#### Position Reduction on Opposite-Side Trade

```
closeQty = min(newQty, opposingQty)
pnl = direction · (entry - trade.entry) · closeQty · contractValue
remaining = newQty - closeQty    # if > 0, opens new position (flip)
```

**Source:** PropForge `src/script.js` Trades.place algorithm

### 6.6 Execution Cooldowns & Gating (go-trader-main Rust)

```
Event Evaluation Cooldown:  min 5s between evaluations per symbol
Order Cooldown:             min 60s between order attempts
Confidence Gate:            confidence ≥ 0.65 for auto-trade
Low-Latency Mode:           stream-first, ms-level event gating
```

**Source:** go-trader-main `rust-trader/src/main.rs` CLI flags

### 6.7 Chart-Line to Order Replication (Prop-Matrix-Engine)

When copying trade signals across accounts:

```
For each target client (NOT source):
    parse label string → determine order kind
        "SELL STOP" → SellStop
        "BUY STOP"  → BuyStop
        "SELL"      → SellLimit
        default      → BuyLimit
    Create pending order with defaultVolumeLots
    Submit to target client

WARNING: Flat volume across all accounts is inadequate.
SHOULD scale by: target_balance / source_balance · source_volume
```

**Source:** Prop-Matrix-Engine `ChartLineCopyEngine.ResolveOrderKind()`

### 6.8 Fixed-Point Arithmetic for Prices (go-trader-master)

**MUST NOT use IEEE 754 floating-point** for order prices and quantities. Use:
- Go: `github.com/robaho/fixed` — fixed-point decimal
- Python: `Decimal` with string construction (never `float` → `Decimal`)
- C#: `decimal` type
- Rust: `rust_decimal`

---

## 7. Data Flow, Persistence & Audit

### 7.1 Append-Only Immutable Audit Log (go-trader-main Rust — MANDATORY)

All trade-related events **MUST** be recorded in an append-only audit log:

```
Tables (minimum):
1. signals        — {id, symbol, signal, confidence, reason, created_at}
2. decisions      — {id, signal_id, decision, size, regime_multiplier, created_at}
3. orders         — {id, decision_id, order_id, symbol, side, qty, price, type, status, created_at}
4. fills          — {id, order_id, fill_price, fill_qty, fill_id, filled_at}
5. regime_states  — {id, timestamp, y_value, regime, formula_mult, data_mult, applied_mult}

Properties:
- WAL mode (Write-Ahead Logging for concurrent reads)
- No UPDATE or DELETE operations permitted
- Corrections are NEW rows with reference to original
- All timestamps in UTC ISO 8601
```

**Source:** go-trader-main `rust-trader/src/audit_store.rs`

### 7.2 Market Data Buffer (go-trader-main Rust)

```
BarBuffer:
    Ring buffer per symbol, max 300 OHLCV bars
    Seed from historical data at startup (default 220 bars)
    Push new bars, evict oldest
    Indicators computed from this buffer (must be warm before signals)
```

### 7.3 Data Flow Mandatory Rules

1. **Market Data → Signal → Decision → Order → Fill:** Linear, traceable, auditable
2. **No stale data:** Portfolio must sync every 60s minimum (go-trader-main Rust pattern)
3. **No look-ahead:** Backtesting must use point-in-time data with embargo (Purged CV pattern)
4. **No silent failures:** All data pipeline stages must emit diagnostics with severity levels

### 7.4 Persistence Requirements

| Data | Storage | Frequency |
|------|---------|-----------|
| Account state (balance, positions) | SQLite (prod) / JSON file (dev) | Every change |
| Trade history | SQLite append-only | Every fill/cancel |
| Strategy config (GKR) | YAML files with hash verification | On change |
| Risk parameters | Config file with audit log of changes | On change |
| Market data (bars) | Ring buffer (in-memory) + optional Parquet archive | Streaming |
| Notification log | Ring buffer (max 1000) + optional file | Real-time |
| Regime state | SQLite (timestamped snapshots) | Every evaluation |

---

## 8. Configuration & Parameter Governance

### 8.1 Configuration Hierarchy

```
1. Command-Line Flags (highest priority, override all)
2. Environment Variables (.env file)
3. YAML/JSON Config File (per-environment)
4. GKR Strategy File (per-strategy parameters)
5. Hardcoded Defaults (lowest priority, always provide safe defaults)
```

### 8.2 Immutable Config Pattern (Quant-Strategy-Tokenizer)

```python
# MANDATORY PATTERN
class NumericPolicy(BaseModel):
    model_config = ConfigDict(frozen=True)
    representation: Literal["float64", "decimal", "int64", "bool"]
    deterministic_level: Literal["semantic", "bit_exact", "engine_specific", "platform_dependent"]
    nan_policy: Literal["propagate", "reject", "ignore", "coerce_null"]
    inf_policy: Literal["reject", "propagate", "coerce_null"]
```

Config objects are **frozen** after creation. Changes require explicit versioning.

### 8.3 Parameter Versioning

All configuration changes **MUST** be recorded with:
- `changed_at`: ISO 8601 timestamp
- `changed_by`: user/process identifier
- `previous_value`: old value
- `new_value`: new value
- `reason`: human-readable justification

**Source:** go-trader-main Rust audit pattern extended to config

### 8.4 Environment Segregation (Quant-Strategy-Tokenizer Profile System)

| Profile | Backtest Allowed | Paper Trade | Live Trade | Requirements |
|---------|-----------------|-------------|------------|-------------|
| `research` | Yes | No | No | None |
| `paper` | Yes | Yes (paper) | No | Token maturity ≥ experimental |
| `pretrade` | Yes | Yes | No (dry-run) | Token maturity ≥ frozen, no unsafe_future |
| `production_guarded` | Yes | Yes | Yes (with kill-switch) | All checks, approval records, audit enabled |

**Source:** Quant-Strategy-Tokenizer `qst/profiles/`

### 8.5 Token/Algorithm Maturity Gating

| Maturity | production_guarded | pretrade | paper | research |
|----------|-------------------|----------|-------|----------|
| accepted | pass | pass | pass | pass |
| frozen | pass | pass | pass | pass |
| experimental | error | error | warning | warning |
| deprecated | error | warning | warning | warning |
| reserved_design | error | error | error | error |

---

## 9. Testing, Validation & Calibration

### 9.1 Backtesting Standards

#### 9.1.1 Purged Walk-Forward Cross-Validation (MANDATORY for ML-based strategies)

```
For each fold:
    1. Train data: all samples with timestamps < test_start
    2. Purge: remove train samples that overlap with test events
    3. Embargo: remove embargoPct * N samples after test end
    4. Test on forward window

Parameters:
    num_folds = 5
    embargo_pct = 0.01
    test_size = 0.3 (rolling window)
```

**Source:** go-trader-main `algo/purged_cv.go:165-235`

#### 9.1.2 Sequential Bootstrap (for sampling with overlap)

```
For each bootstrap draw:
    1. Build indicator matrix I[i,j] = 1 if bar_i overlaps with event_j
    2. Compute average uniqueness per sample
    3. Weighted random selection using uniqueness as sampling weights
    4. Return sample with replacement

Standard bootstrap (fallback): uniform random sampling
```

**Source:** go-trader-main `algo/sequential_bootstrap.go:316-387`

#### 9.1.3 Backtesting Execution Realism Checklist

| Requirement | Source |
|-------------|--------|
| Use maker fill prices, not mid-price | Quant-toolkit |
| Model fill probability (not 100% fills at limit) | Quant-toolkit |
| Include fees (maker + taker) | Quant-toolkit |
| Include adverse selection haircut | Quant-toolkit |
| Account for spread crossing | go-trader-master |
| Sequential (not vectorized) day-by-day simulation | Quant-toolkit |
| Rolling bankroll with PnL feedback | Quant-toolkit, PropForge |
| No look-ahead bias (purged CV) | go-trader-main |

### 9.2 Calibration Audit Pipeline (MANDATORY quarterly)

```
Input: (forecast_p, outcome) pairs from ledger

1. Compute Brier Score
2. Generate Reliability Diagram (10 bins)
3. Compute Murphy Decomposition (Reliability, Resolution, Uncertainty)
4. Segment by: side (YES/NO), price bucket, edge bucket, cohort
5. Flag bins with n < 20 as "insufficient data"
6. If any segment shows significant miscalibration → Platt scaling
```

**Source:** Quant-toolkit `calibration_audit.py`

### 9.3 Pre-Flight Checklist (MANDATORY before any live deployment)

```
[ ] 1. Single-instance guard active (no duplicate process)
[ ] 2. Concurrent cron guard active (no overlapping scheduled runs)
[ ] 3. Model version pinned and verified (hash check)
[ ] 4. Strategy GKR validated (graph_hash + param_hash verified)
[ ] 5. Calibration audit passed (Brier within tolerance, all segments)
[ ] 6. Backtest on recent data passed (last 30 days, no degradation)
[ ] 7. Exchange connection healthy (test order successful)
[ ] 8. Kill-switch tested (end-to-end close-all + disable)
[ ] 9. Alert channels verified (test notification received)
[ ] 10. Risk caps confirmed (position limits, DD limits, trade count)
[ ] 11. Balance reconciled (ledger balance == exchange balance within tolerance)
[ ] 12. Code deployed matches VCS tag (git hash verified)
```

**Source:** Quant-toolkit `pre-flight-checklist/SKILL.md`, adapter from go-trader-main

### 9.4 PnL Attribution (MANDATORY for strategy improvement)

Slice trade history by:
1. **Side:** YES vs NO (buy vs sell)
2. **Price bucket:** 0-30c, 30-50c, 50-70c, 70-100c
3. **Edge bucket:** 0-5%, 5-10%, 10-20%, 20%+
4. **Side × Price bucket:** 2D cross product
5. **Cohort / regime:** if data available
6. **Time window:** rolling periods

For each slice, report: n, wins, hit_rate (with Wilson 95% CI), total PnL, total cost, ROI.

**Minimum sample size for statistical significance: n ≥ 30**

**Source:** Quant-toolkit `attribute.py`

---

## 10. Security, Governance & Profile Gating

### 10.1 Custom Code Execution Sandbox (Quant-Strategy-Tokenizer MANDATORY)

When executing custom/community strategy tokens:

```
Four-Stage Boundary:
    1. VERIFY INTEGRITY:
       - Compute all hashes (token_spec, token_pack, implementation_ref,
         runtime_environment, expected_artifact)
       - Reject if any hash mismatch
       - Report risk level assessment
       
    2. CHECK AUTHORIZATION:
       - Verify profile allows custom token execution
       - Find matching approval record
       - Reject if not authorized
       
    3. ISSUE GRANT:
       - Create time-limited execution grant (default TTL: 900s)
       - Pin to specific token version (by hash)
       - Grant carries audit trail
       
    4. EXECUTE (sandboxed):
       - Resolve entrypoint from implementation_ref
       - Execute with inputs
       - Validate output shape, types, decimal strings
       - Emit audit record with full execution context
```

**Source:** Quant-Strategy-Tokenizer `qst/custom_runtime/`

### 10.2 Authentication & Authorization Standards

| Component | Minimum Requirement |
|-----------|-------------------|
| Named pipes (IPC) | ACL/security descriptor (NO unauthenticated pipes) |
| HTTP API | Bearer token or API key, TLS |
| WebSocket | WSS with token authentication |
| Database | File permissions (0600) minimum |
| Config secrets | Environment variables or vault (HashiCorp Vault pattern from go-trader-main) |

### 10.3 Deterministic Identity (Content-Addressable)

All artifacts **MUST** be identifiable by content hash, not by mutable name or path:

```
hash = "sha256:" + SHA256(canonical_json({
    "hash_namespace": "orca-algo-v1/0.1",
    "kind": kind,         # "graph" | "param" | "instance" | "token_spec" | "implementation"
    "payload": payload    # stable, canonicalized representation
})).hexdigest()
```

**Source:** Quant-Strategy-Tokenizer `qst/hash/common.py:26-35`

### 10.4 Source of Truth Rule

There **MUST** be exactly ONE canonical implementation of each mathematical function:

| Function | Canonical Location |
|----------|-------------------|
| Kelly fraction | `kelly.py` (Quant-toolkit) |
| Brier score | `calibration_audit.py` (Quant-toolkit) |
| Platt scaling | `platt_fit.py` (Quant-toolkit) |
| Wilson CI | `attribute.py` (Quant-toolkit) |
| EWMA volatility | `triple_barrier.go` (go-trader-main) |
| Triple barrier labeling | `triple_barrier.go` (go-trader-main) |
| Fractional differentiation | `fractional_diff.go` (go-trader-main) |
| Purged K-Fold CV | `purged_cv.go` (go-trader-main) |
| Drawdown (HWM trailing) | `drawdown.py` (Quant-toolkit) |

**DO NOT reimplement** these functions inline elsewhere. Reference the canonical source.

---

## 11. UI, Monitoring & Observability

### 11.1 Dashboard Requirements

#### 11.1.1 Mandatory Dashboard Panels

```
┌──────────────┬──────────────┬──────────────────┐
│  EQUITY CARD │  RISK CARD   │  POSITIONS CARD   │
│  Balance     │  Daily DD %  │  Open positions   │
│  Equity      │  Total DD $  │  Unrealized PnL   │
│  Floating PnL│  Regime      │  Margin used      │
│  HWM         │  Status      │  Lot count        │
└──────────────┴──────────────┴──────────────────┘
┌─────────────────────────────────────────────────┐
│  RISK DIAGNOSTIC LOG (append-only, timestamped) │
│  [HH:mm:ss.fff] [LEVEL] Message                 │
└─────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────┐
│  RECENT TRADES TABLE                            │
│  Time | Symbol | Side | Qty | Entry | Exit | PnL│
└─────────────────────────────────────────────────┘
```

**Source:** MT5-Drawdown-Guard, Prop-Matrix-Engine, go-trader-main notification system

#### 11.1.2 Visual Risk Signaling Standard

```
GREEN  (#00E6C3): DD < 75% of limit         → SAFE
AMBER  (#F5C451): 75% ≤ DD < 100% of limit  → WARNING
RED    (#FF3B30): DD ≥ 100% of limit         → BREACHED
FLASHING RED:     Kill-switch triggered       → LOCKED
```

**Source:** MT5-Drawdown-Guard `Form1.cs:376-381`

### 11.2 Notification System (go-trader-main reference)

```
Ring buffer, max 100-1000 notifications
    {id, type, priority, symbol, message, timestamp, read}

Priorities:
    CRITICAL:  Kill-switch triggered, connection lost, DD breach
    HIGH:      Trade executed, stop-loss hit, regime change
    NORMAL:    Signal generated, order placed, daily summary
    LOW:       Market data update, heartbeat, config change

REST API endpoints:
    GET  /api/notifications          — list all
    GET  /api/notifications/unread   — unread only
    POST /api/notifications/{id}/read — mark read
```

### 11.3 Structured Logging (MANDATORY)

- **Python:** `structlog` or `logging` with JSON formatter
- **Go:** `slog` (standard library, structured) or `zap`
- **Rust:** `tracing` with JSON subscriber
- **C#:** `ILogger<T>` with Serilog JSON sink
- **JS:** Custom with `console.log` structured objects

**Log levels:** TRACE < DEBUG < INFO < WARN < ERROR < FATAL

**Every log entry MUST include:** `timestamp`, `level`, `message`, `component`, `correlation_id`

### 11.4 Real-Time Charting (PropForge reference)

For any interactive price chart:
- Canvas-based rendering (performant for 200+ candles)
- OHLC candlestick aggregation (1s, 5s, 1min based on source frequency)
- Interactive SL/TP line dragging
- Mouse scroll zoom (0.5x–10x range)
- Double-buffering (prevents flicker on rapid updates)
- Trade entry/exit markers on chart

---

## 12. Implementation Roadmap

### 12.1 Phase 1: Foundation (Weeks 1–4)

| Task | Reference | Priority |
|------|-----------|----------|
| Define domain models (immutable records with Pydantic/dataclass/Go structs) | Prop-Matrix-Engine, Quant-Strategy-Tokenizer | P0 |
| Implement canonical Kelly sizer with all attenuators | Quant-toolkit `kelly.py` | P0 |
| Implement HWM trailing drawdown engine with multi-tier response | Quant-toolkit `drawdown.py` | P0 |
| Implement kill-switch with re-entrancy guard and pre-breach buffer | Prop-Matrix-Engine, MT5-Drawdown-Guard | P0 |
| Set up append-only audit log (SQLite WAL, 5 tables) | go-trader-main Rust | P0 |
| Implement deterministic hashing (canonical JSON + SHA-256) | Quant-Strategy-Tokenizer | P1 |
| Establish config hierarchy with environment segregation | Quant-Strategy-Tokenizer profiles | P1 |

### 12.2 Phase 2: Strategy Engine (Weeks 5–8)

| Task | Reference | Priority |
|------|-----------|----------|
| Implement GKR strategy IR with validation | Quant-Strategy-Tokenizer | P0 |
| Implement temporal contract validation (look-ahead prevention) | Quant-Strategy-Tokenizer | P0 |
| Build factory + plugin registry for algorithm components | go-trader-main | P0 |
| Implement signal generation pipeline (local rule engine) | go-trader-main Rust | P1 |
| Implement multi-algorithm voting with weighted confidence | go-trader-main | P1 |
| Implement position scaling/flipping logic | PropForge | P1 |

### 12.3 Phase 3: Execution & Backtesting (Weeks 9–12)

| Task | Reference | Priority |
|------|-----------|----------|
| Implement purged walk-forward cross-validation | go-trader-main | P0 |
| Implement sequential bootstrap | go-trader-main | P1 |
| Implement triple barrier labeling | go-trader-main | P1 |
| Build market scanner pipeline (6 filters) | Quant-toolkit | P1 |
| Implement backtesting engine with realistic fills | Quant-toolkit, go-trader-master | P0 |
| Implement maker pricing logic | Quant-toolkit | P1 |

### 12.4 Phase 4: Monitoring & Operations (Weeks 13–16)

| Task | Reference | Priority |
|------|-----------|----------|
| Build dashboard with mandatory panels | MT5-Drawdown-Guard, Prop-Matrix-Engine | P0 |
| Implement notification system (ring buffer + REST API) | go-trader-main | P1 |
| Implement structured logging | go-trader-main, Quant-Strategy-Tokenizer | P1 |
| Build calibration audit pipeline | Quant-toolkit | P1 |
| Implement PnL attribution engine | Quant-toolkit | P1 |
| Build pre-flight checklist automation | Quant-toolkit | P1 |

### 12.5 Phase 5: Advanced Features (Ongoing)

| Task | Reference | Priority |
|------|-----------|----------|
| AI-assisted signal generation (Claude/LLM integration) | go-trader-main | P2 |
| Economic cartography macro-regime model | go-trader-main | P2 |
| FRED macro data overlay (Sahm, yield curve, NFCI, HY) | go-trader-main | P2 |
| Fractional differentiation for stationarity | go-trader-main | P2 |
| Custom token execution sandbox | Quant-Strategy-Tokenizer | P2 |
| Multi-account kill-switch coordination | Prop-Matrix-Engine | P2 |
| Chart-line order replication across accounts | Prop-Matrix-Engine | P2 |
| Real-time candlestick charting with interactive SL/TP | PropForge | P2 |

---

## 13. Comparative Methodology Matrix

### 13.1 Position Sizing Approaches

| Method | Project | Complexity | Risk Sensitivity | Recommended For |
|--------|---------|-----------|-----------------|-----------------|
| Full Kelly | go-trader-main, Quant-toolkit | Low | Low | Backtesting only |
| Fractional Kelly (k=0.25) | Quant-toolkit (mandatory) | Low | Medium | All live trading |
| Kelly + Edge Discount + Caps | Quant-toolkit | Medium | High | Production (MANDATORY) |
| Kelly + Vol Adjustment | go-trader-main | Medium | High | Continuous assets |
| Kelly + Diversification Scaling | go-trader-main | High | High | Multi-asset portfolios |
| Kelly + Regime Multiplier | go-trader-main | High | Very High | All-weather systems |

**Verdict:** OrcaAlgo **MUST** implement Fractional Kelly with Edge Discount and Hard Caps as minimum. Vol Adjustment and Regime Multiplier are SHOULD for continuous assets and macro-aware systems respectively.

### 13.2 Risk Monitoring Approaches

| Method | Project | Real-Time | Trailing DD | Pre-Breach Buffer | Multi-Tier | Account-Level |
|--------|---------|-----------|-------------|-------------------|------------|---------------|
| Named Pipe + EA kill | MT5-Drawdown-Guard | Yes | No (static) | No | No | Single |
| Stealth SL manager | MT5-MultiPass | Yes | No | No | No | Single |
| Event-driven risk engine | Prop-Matrix-Engine | Yes | Yes (daily peak) | Yes (0.5%) | No | Multiple |
| Kelly attenuator + DD monitor | Quant-toolkit | No (batch) | Yes (HWM) | No | Yes (3-tier) | Single |
| Simulated phase rules | PropForge | Yes (100ms) | Yes (trailing) | No | No | Single |

**Verdict:** OrcaAlgo **MUST** combine Prop-Matrix-Engine's pre-breach buffer with Quant-toolkit's 3-tier HWM trailing DD and Prop-Matrix-Engine's multi-account architecture.

### 13.3 Strategy Representation Approaches

| Method | Project | Versioned | Hashed | Typed | Human-Readable | Machine-Executable |
|--------|---------|-----------|--------|-------|---------------|-------------------|
| GKR YAML files | Quant-Strategy-Tokenizer | Yes | Yes (3-layer) | Yes (TypeSpec) | Yes (YAML) | No (reference only) |
| Factory-registered Go algos | go-trader-main | No | No | Interface | No | Yes |
| Hardcoded JS phase config | PropForge | No | No | No | Partially | Yes |
| C# domain records + UI | Prop-Matrix-Engine | No | No | Yes (records) | No | Yes |

**Verdict:** OrcaAlgo **MUST** adopt GKR as the canonical strategy representation format. The factory-registered algorithm implementations serve as the execution layer for registered tokens. GKR provides audit, versioning, and governance; factory provides execution.

### 13.4 Exchange Communication Patterns

| Pattern | Project | Latency | Reliability | Complexity | Use Case |
|---------|---------|---------|-------------|------------|----------|
| REST polling (5s) | go-trader-main Go | Medium | Medium | Low | Market data, simple |
| WebSocket streaming | go-trader-main Rust | Low | High | Medium | Real-time data |
| Named pipes (IPC) | MT5-Drawdown-Guard, MT5-MultiPass | Very Low | Medium (local only) | Low | Process-local MT5 |
| FIX 4.4 protocol | go-trader-master | Low | Very High | Very High | Institutional |
| gRPC bidirectional | go-trader-master | Low | High | Medium | Internal services |
| SignalR | Prop-Matrix-Engine | Low | High (auto-reconnect) | Low | .NET ecosystem |
| UDP multicast + TCP replay | go-trader-master | Very Low | High | High | Market data distribution |

**Verdict:** OrcaAlgo **SHOULD** use WebSocket streaming for market data (primary) with REST polling as fallback. For broker execution, use the broker's native API (Alpaca REST, MT5 pipe, etc.). For internal component communication, use gRPC or in-process channels.

### 13.5 Language Selection Decision Matrix

| Criterion | Python | Go | Rust | C# | JavaScript |
|-----------|--------|----|------|-----|------------|
| Mathematical expressiveness | ★★★★★ | ★★★ | ★★★ | ★★★ | ★★ |
| Production reliability | ★★★ | ★★★★ | ★★★★★ | ★★★★ | ★★ |
| Development speed | ★★★★★ | ★★★★ | ★★ | ★★★★ | ★★★ |
| Async/Concurrency | ★★★ | ★★★★★ | ★★★★★ | ★★★★ | ★★★ |
| Type safety | ★★★ | ★★★★ | ★★★★★ | ★★★★ | ★★ |
| Ecosystem (quant/trading) | ★★★★★ | ★★ | ★ | ★★★ | ★★ |
| Audit trail / correctness | ★★★ | ★★★★ | ★★★★★ | ★★★★ | ★★ |

**Verdict:**
- **Python** for: strategy IR, mathematical models, calibration, backtesting, CLI tooling
- **Go** for: API servers, market data ingestion, lightweight services
- **Rust** for: execution engine, audit logging, performance-critical paths
- **C#** for: Windows desktop UI (MT5 integration), SignalR real-time services
- **JavaScript/TypeScript** for: web dashboard, interactive charting

---

## 14. Antipatterns & Prohibitions

### 14.1 Critical Prohibitions (NEVER do these)

| # | Antipattern | Example Source | Consequence |
|---|-------------|---------------|-------------|
| 1 | **Full Kelly in production** | Quant-toolkit SKILL.md explicit warning | Gambler's ruin, >50% chance of 50%+ DD |
| 2 | **Inline mathematical reimplementation** | Quant-toolkit `backtest_template.py` diverging from `kelly.py` | Source-of-truth fragmentation, silent bugs |
| 3 | **Unrealized PnL excluded from DD** | MT5-Drawdown-Guard static balance-only DD | Silent equity erosion, prop firm rule violation |
| 4 | **Static (non-trailing) drawdown for prop firms** | MT5-Drawdown-Guard static absolute DD | Non-compliance with FTMO/MFF/TopStep rules |
| 5 | **Floating-point for order prices** | Industry-wide antipattern | Rounding errors, audit discrepancies |
| 6 | **Global mutable state** | go-trader-main `globalStates` | Race conditions, untestable |
| 7 | **No kill-switch re-entrancy guard** | — | Double-firing, corrupted state |
| 8 | **Perfect fill assumption (mid-price fills)** | Common backtesting error | Overestimated performance by 30-50% |
| 9 | **Unnamed pipe with no ACL** | MT5-MultiPass, MT5-Drawdown-Guard | Any process can inject fake trades |
| 10 | **`panic()`/`throw` for recoverable errors** | go-trader-master exchange config parsing | Crashes kill running strategies |
| 11 | **No temporal validation (look-ahead bias)** | Industry-wide research error | Inflated backtest results |
| 12 | **Single monolithic file >1000 LOC** | PropForge 1517-line `script.js` | Unmaintainable, untestable |
| 13 | **Mixed-up unrelated code in same repo** | MT5-MultiPass MQL5/ directory | Security risk, confusion, license issues |
| 14 | **Star-farming CI/CD bots** | MT5-MultiPass, MT5-Drawdown-Guard | Integrity violation, deceptive metrics |
| 15 | **Missing margin check before order** | — | Over-leverage, account blow-up |

### 14.2 Conditional Prohibitions (do only with explicit approval)

| # | Pattern | Condition Required |
|---|---------|-------------------|
| 1 | AI-only signals (no fallback rule engine) | Must have local fallback with confidence gate |
| 2 | Auto-trading without confidence gate | Must have configurable confidence threshold (min 0.60) |
| 3 | Cross-account trade replication at flat volume | Must scale by account equity ratio |
| 4 | Backtest without purge/embargo | Only for exploratory research, never for production decision |
| 5 | Slippage/spread not modeled in backtest | Must document as "idealized, not production-grade" |
| 6 | Single-threaded order processing per symbol | Acceptable for low-frequency (<1 trade/sec); must shard for HFT |

---

## Appendix A: Key Formula Quick Reference

```
KELLY (BINARY):
    f*_yes = (p - c) / (1 - c)
    f*_no  = ((1-p) - (1-c)) / c

KELLY (CONTINUOUS):
    f* = (p_win * b - (1-p_win)) / b

KELLY PRODUCTION (MANDATORY):
    p_discounted = max(p - δ, 0) or min(p + δ, 1)   # δ = 0.02-0.05
    f_fractional = f* × 0.25
    f_final = min(f_fractional, 0.02, 0.30 - current_exposure_pct)

EWMA VOLATILITY:
    α = 2/(span+1), r_t = ln(P_t/P_{t-1})
    EWMA_t = α·r_t + (1-α)·EWMA_{t-1}
    EWMVar_t = α·(r_t - EWMA_t)² + (1-α)·EWMVar_{t-1}
    σ_t = √(EWMVar_t)

BRIER SCORE:
    Brier = (1/n)·Σ(p_i - o_i)²

MURPHY DECOMPOSITION:
    Brier = Reliability - Resolution + Uncertainty

PLATT SCALING:
    calibrated_p = σ(A·logit(raw_p) + B)
    Fit via MLE (Nelder-Mead)

WILSON CI (95%):
    CI = [max(c-h,0), min(c+h,1)]
    c = (p + z²/(2n)) / (1 + z²/n)
    h = z·√(p(1-p)/n + z²/(4n²)) / (1 + z²/n)

DRAWDOWN (TRAILING HWM):
    HWM(t) = max_{s≤t} equity(s)
    DD(t) = (HWM(t) - equity(t)) / HWM(t)

TRIPLE BARRIER LABELING:
    Upper(t) = entryP · (1 + profitTaking · σ)
    Lower(t) = entryP · (1 - stopLoss · σ)
    Time: entryT + timeHorizon

FRACTIONAL DIFFERENTIATION:
    w[0] = 1
    w[k] = -w[k-1] · (d - k + 1) / k
    FFD[i] = Σ_{j=0}^{W-1} w[j] · series[i-j]
```

---

## Appendix B: Dependency Map (Minimum Required Libraries)

### Python
```
pydantic >= 2.5          # Domain models, validation
pyyaml >= 6.0             # GKR YAML loading
numpy >= 1.24             # Array operations
scipy >= 1.10             # Optimization (Platt scaling)
typer >= 0.9              # CLI framework
structlog >= 23           # Structured logging
pytest >= 7               # Testing
```

### Go
```
github.com/alpacahq/alpaca-trade-api-go/v3   # Broker integration
gonum.org/v1/gonum                            # Matrix operations
modernc.org/sqlite or github.com/mattn/go-sqlite3  # Audit DB
github.com/joho/godotenv                      # Config loading
```

### Rust
```
tokio (full)        # Async runtime
axum                # HTTP framework
rusqlite (bundled)  # SQLite
rust_decimal        # Fixed-point arithmetic
serde / serde_json  # Serialization
tracing             # Structured logging
```

### C# (.NET 8)
```
Microsoft.AspNetCore.SignalR.Client   # Real-time comms
System.IO.Pipes                       # Named pipe IPC
```

---

## Appendix C: Research Project Quality Assessment

| Project | Architecture | Math Correctness | Code Quality | Completeness | Integrity | Overall |
|---------|-------------|-----------------|--------------|--------------|-----------|---------|
| go-trader-main | ★★★★ | ★★★★★ | ★★★★ | ★★★ | ★★★★★ | **Tier 1** |
| go-trader-master | ★★★★★ | ★★★★ | ★★★ | ★★★ | ★★★★★ | **Tier 1** |
| Quant-Strategy-Tokenizer | ★★★★★ | ★★★★★ | ★★★★★ | ★★★★ | ★★★★★ | **Tier 1** |
| Quant-toolkit | ★★★★ | ★★★★★ | ★★★★ | ★★★★ | ★★★★★ | **Tier 1** |
| Prop-Matrix-Engine | ★★★★★ | ★★★★ | ★★★★★ | ★★ | ★★★★ | **Tier 2** |
| PropForge | ★★★ | ★★★★ | ★★★ | ★★★★ | ★★★★ | **Tier 2** |
| MT5-Drawdown-Guard | ★★ | ★★★ | ★★★ | ★ | ★★ | **Tier 3** |
| MT5-MultiPass-Dashboard | ★★ | ★★★ | ★★ | ★ | ★ | **Tier 3 (Reject)** |

---

*End of Unified Best Practice Specifications Document for OrcaAlgo v1.0.0*

*This document is the governing reference for all architecture decisions, mathematical implementations, and operational procedures. Deviations require explicit documented approval with justification.*
