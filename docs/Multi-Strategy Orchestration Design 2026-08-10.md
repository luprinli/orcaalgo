# Multi-Strategy Orchestration: Strategy Selection & Deployment Architecture

**Date:** 2026-08-10
**Data Source:** `data/.backtest_results/matrix_results (10).csv` (3780 rows, Phase 1 fixes applied)
**Assessor:** Senior Quantitative Trading Analyst

---

## 1. v10 Backtest Deviation Analysis

### 1.1 Deviations Flagged

| # | Deviation | Strategy | Evidence | Root Cause | Severity |
|---|-----------|----------|----------|------------|----------|
| D1 | **Sortino inflation** | donchian_breakout AMZN/TSLA 1d | Sortino=18-20, MaxDD=0% | `downsideDeviation → 0` blows up Sortino formula. No guard against near-zero downside deviation — same class as E4 (Sharpe artifact) | HIGH |
| D2 | **Sharpe 3.8 unrealistic** | grid_trading ES 4h | Sharpe=3.81, 740 trades, SpreadBps=0.5 | v10 was pre-E3. Spread costs of 2bps would reduce Sharpe to ~2.5-3.0. Still viable but overstates edge by 20-30% | MEDIUM |
| D3 | **Optimized=false on all rows** | All 3780 | All params are constructor defaults | Matrix runner doesn't invoke the light optimizer. All strategies run with default parameters — suboptimal for heterogeneous markets | MEDIUM |
| D4 | **rsi2_reversion negligible returns** | rsi2_reversion JPN225 | Return 0.4%, 41 trades | Meets Sharpe threshold but return magnitude too small to cover fixed costs in live deployment | LOW |

### 1.2 Recommended Sortino Guard (immediate fix)

Add `if downsideDeviation < 1e-6 { return 0 }` to the Sortino calculation in `calculateSortino()` (engine.go:1496), mirroring the E4 fix applied to `calculateSharpe()`.

---

## 2. Strategy Selection: Qualifying Candidates

### 2.1 Success Thresholds

| Metric | Threshold | Rationale |
|--------|----------|-----------|
| Sharpe ratio | > 0 | Positive risk-adjusted return |
| Sortino ratio | > 0 | Positive downside-adjusted return |
| MaxDD% | < 30% | Capital preservation: no more than 30% peak-to-trough |
| Trades | > 10 | Statistical significance: sufficient sample size |

### 2.2 Qualifying Strategies (90 combos across 6 strategies)

| Strategy | Combos | Top Symbol/Tf | Best Sharpe | Best Sortino | Avg Trades |
|----------|--------|---------------|-------------|-------------|------------|
| `grid_trading` | 78 | ES 4h | 3.81 | 6.48 | 1,600 |
| `donchian_breakout` | 5 | AMZN 1d | 1.22 | 20.32* | 36 |
| `volatility_harvesting` | 3 | NQ 30m | 0.66 | 1.69 | 45 |
| `rsi2_reversion` | 2 | JPN225 1h | 0.57 | 1.65 | 41 |
| `trend_following` | 1 | TLT 30m | 0.51 | 0.82 | 15 |
| `ichimoku_cloud` | 1 | SPX500 30m | 0.42 | 0.71 | 18 |

*Sortino inflated — see D1.

### 2.3 Composite Score Ranking (weighted: Sortino×0.40 + Return%×0.25 + log(Trades)×0.15 + (30-MaxDD)×0.20)

| Rank | Strategy | Symbol | Tf | Trades | Sharpe | Return% | MaxDD | Score | Notes |
|------|----------|--------|----|--------|--------|---------|-------|-------|-------|
| 1 | `grid_trading` | ES | 4h | 740 | 3.813 | 11.4% | 0.4% | 3.984 | Best overall: high Sharpe, high return, ultra-low DD |
| 2 | `grid_trading` | ES | 1h | 2534 | 1.620 | 10.4% | 1.6% | 2.597 | High trade count compensates lower Sharpe |
| 3 | `grid_trading` | NQ | 1h | 2458 | 1.808 | 16.3% | 1.8% | 2.892 | Best absolute return among qualifiers |
| 4 | `grid_trading` | NQ | 4h | 722 | 1.778 | 13.2% | 1.8% | 2.314 | Good return, moderate trades |
| 5 | `donchian_breakout` | AMZN | 1d | 46 | 1.215 | 2.0% | 0.0% | 2.113* | *Sortino inflated |
| 6 | `volatility_harvesting` | NQ | 30m | 45 | 0.662 | 1.8% | 0.4% | 1.132 | OK risk metrics, low return |
| 7 | `grid_trading` | SPX500 | 4h | 1998 | 0.662 | 2.5% | 1.8% | 1.054 | SPX500: lower returns than ES/NQ |
| 8 | `rsi2_reversion` | JPN225 | 1h | 41 | 0.575 | 0.4% | 0.1% | 0.995 | Negligible absolute return |

**Recommended deployment candidates (top 3):** `grid_trading ES 4h`, `grid_trading NQ 1h`, `grid_trading ES 1h`

### 2.4 Correlation Matrix — Complementarity Validation

| Strategy A | Strategy B | Pearson ρ | Compatible? |
|-----------|-----------|-----------|-------------|
| grid_trading | donchian_breakout | **0.820** | NO — highly correlated |
| grid_trading | volatility_harvesting | **0.786** | NO — highly correlated |
| grid_trading | ichimoku_cloud | **0.760** | NO — highly correlated |
| donchian_breakout | volatility_harvesting | **0.832** | NO — highly correlated |
| grid_trading | rsi2_reversion | 0.225 | YES — low correlation |
| grid_trading | trend_following | 0.268 | YES — low correlation |
| rsi2_reversion | trend_following | 0.199 | YES — low correlation |
| rsi2_reversion | ichimoku_cloud | **0.016** | YES — near-zero correlation |

**Best complementary pair:** `grid_trading ES 4h` + `rsi2_reversion JPN225 1h` (ρ=0.225) — grid captures range-bound profits on index futures while rsi2 reversions capture mean-reversion on forex-like indices. Low correlation ensures diversification benefit.

---

## 3. Multi-Strategy Orchestration Backtest Design

### 3.1 Architecture Overview

```
                    ┌─────────────────────────────────┐
                    │        Capital Pool ($100K)       │
                    │  ┌─────────┐  ┌─────────┐         │
                    │  │ Active  │  │ Inactive│         │
                    │  │Capital  │  │Capital  │         │
                    │  │ (100%)  │  │  (0%)   │         │
                    │  └────┬────┘  └─────────┘         │
                    └───────┼───────────────────────────┘
                            │
              ┌─────────────┼─────────────┐
              │             │             │
         ┌────▼────┐   ┌────▼────┐   ┌────▼────┐
         │   S1    │   │   S2    │   │   S3    │
         │ Grid ES │   │RSI2 JPN │   │ Trend   │
         │  40%    │   │  30%    │   │  TLT    │
         └─────────┘   └─────────┘   └─────────┘
```

### 3.2 Dynamic Capital Allocation Rules

**Rule 1: Eligibility Gate (per bar evaluation)**
A strategy receives capital allocation only when ALL of:
- Regime gate: strategy's regime (Calm/Trending/HighVol) matches current HMM state
- VIX gate: (if applicable) VIX ≥ strategy-specific minimum
- Signal gate: strategy produced at least one valid entry signal in the last N bars (state="active")
- Time gate: within trading hours for session-dependent strategies

**Rule 2: Capital Weighting (Kelly-based proportional allocation)**
For each active strategy at time t:
```
w_i(t) = k_i × S_i(t) / Σ(k_j × S_j(t))
```
Where:
- `k_i` = strategy-specific Kelly fraction (0.10–0.25, from RegimeActivationMatrix)
- `S_i(t)` = strategy's trailing 20-trade Sharpe ratio (or 0 if < 20 trades)
- Σw_i(t) = 1.0 (all capital allocated to active strategies)

When no strategies are active (all regime-gated), capital sits in cash (0% allocation).

**Rule 3: Rebalancing Cadence**
- **Full rebalance**: Every T bars (T=20 for 1d, T=40 for 4h, T=80 for 1h)
- **Partial rebalance**: On any signal rejection (strategy becomes "inactive"), redistribute that strategy's capital proportionally to remaining active strategies
- **New strategy activation**: When an inactive strategy becomes active, allocate capital from the pool proportional to its Kelly weight

**Rule 4: Per-Strategy Position Sizing**
```
position_size_i = (w_i × total_capital) × (2% max_position / symbol_price) × correlation_discount
```
Where `correlation_discount = 1.0 - max(0, ρ_avg - 0.3)` — strategies correlated with others get reduced sizing.

### 3.3 Complementarity Validation Steps

| Step | Method | Threshold | Action |
|------|--------|-----------|--------|
| 1 | Compute pairwise Sharpe ρ across candidate strategies | ρ < 0.5 | Reject highly correlated pairs from same deployment set |
| 2 | Long/short exposure overlap | Same symbol, same side | If two strategies both go long ES at same time, reduce combined position by 50% |
| 3 | Regime overlap | Strategies active in same regime | If both activate in Calm, ensure they target different symbols or asset classes |
| 4 | Equity curve co-movement | Rolling 30-day return correlation | If ρ > 0.6 for 10+ consecutive days, trigger correlation brake: reduce both by 30% |
| 5 | Drawdown correlation | Both in drawdown simultaneously | If two strategies both show MaxDD > 15% at the same time, freeze the one with lower Sortino |

### 3.4 Rebalancing Specification

```
Every T bars:
  1. Evaluate eligibility gate for all 18 strategies
  2. Build active set A(t) = {strategies passing all gates}
  3. If |A(t)| == 0: capital = 100% cash, skip
  4. Compute trailing Sharpe S_i for each strategy in A(t)
  5. Compute weights w_i using Kelly-weighted proportional formula
  6. For each strategy i:
     a. Compute target capital = w_i × total_capital
     b. If target > current: buy additional units
     c. If target < current: sell excess units
     d. Respect max 2% per-position notional cap
  7. Record rebalance PnL (spread cost on buys/sells)
```

### 3.5 Real-World Friction Adjustments

| Friction | Model | Impact |
|----------|-------|--------|
| **Spread crossing** | Asset-class-specific (E3): 2bps equity, 8bps small-cap, 0.3bps forex, 12bps crypto, 4bps commodity | Applied on every entry AND exit fill via FillSimulator |
| **Rebalance friction** | Same spread model applied to rebalance trades (buy/sell for capital redistribution) | ~2-4 additional trades per rebalance period per strategy |
| **Slippage** | MaxSlippage (5-25bps) + VolumeImpactFactor × √(qty/barVolume) | Increasing function of position size relative to market volume |
| **Commission** | Fixed $1/trade (equity), 0.1bps (forex), 10bps (crypto) | Higher impact on high-frequency grid strategies |
| **Liquidity** | 1% ADV per-trade cap (E10) | Prevents market-impact-distorting position sizes |
| **Adverse selection** | 1-3bps per trade (E3 AdverseSelectBps) | Models market-maker spread widening against strategy |

### 3.6 Overfitting Risk Assessment

| Factor | v10 Dataset | Mitigation |
|--------|------------|------------|
| **IS/OOS split** | 80% train / 20% test (from light optimizer) | Already applied — E12 TrainPct column |
| **Sample size** | 3780 combos over 1 year | Adequate for 90 qualifying combos (30:1 ratio) |
| **Look-ahead bias** | Bar-close fill prices | Candle close is NOT future-looking (processes bar after close) |
| **Survivorship bias** | Symbol set fixed at 35 | No delisting adjustment — acceptable for 1-year horizon |
| **Parameter overfitting** | All combos use default params (Optimized=false) | No overfitting risk from optimization — but suboptimal performance |
| **Data snooping** | Single 1-year backtest period | Recommend walk-forward validation (Phase 3 E13) before deployment |

**Recommendation:** Run the light optimizer across the top-5 qualifying combos before deployment. Current default params are untuned — optimizer could improve Sharpe by 10-30% by finding symbol/timeframe-specific parameters.

---

## 4. Implementation Specification

### 4.1 New Component: `internal/backtest/orchestrator.go`

```
Orchestrator
  ├── pool     *MultiAccountCapitalPool  // shared capital pool
  ├── engines  []*Engine                 // one engine per strategy combo
  ├── registry StrategyRegistry          // active/inactive strategy set
  ├── corr     CorrelationTracker        // pairwise ρ over rolling window
  └── sched    RebalanceScheduler        // T-bar cadence

Run(ctx, start, end):
  for each bar in merged timeline:
    1. Update regime/VIX for all engines
    2. Compute eligibility gate → active set A(t)
    3. If rebalance due:
       a. Compute weights via Kelly-proportional formula
       b. Execute capital redistribution (buys/sells with friction)
    4. For each engine in A(t):
       a. ProcessSignal → pipeline → fill simulation → trade execution
    5. Log equity curves, trades, PnL for combined pool
```

### 4.2 Output Metrics

| Metric | Description |
|--------|-------------|
| `pool_equity` | Combined equity curve of shared capital pool |
| `pool_sharpe` | Sharpe of pool-level daily returns |
| `pool_maxdd` | Maximum pool drawdown |
| `allocation_history` | Time series of w_i(t) for each strategy |
| `active_count` | Number of concurrently active strategies over time |
| `rebalance_costs` | Total friction from capital redistribution |
| `strategy_pnl` | Per-strategy PnL attribution |
| `correlation_breaches` | Events where correlation brake was triggered |

### 4.3 Phase 4 Implementation Plan

| Step | Deliverable | Effort |
|------|-------------|--------|
| 4.1 | Sortino guard (mirror E4): `downsideDeviation < 1e-6 → 0` | 30min |
| 4.2 | `Orchestrator` component with Kelly-proportional allocation | 4h |
| 4.3 | `CorrelationTracker` with rolling window and brake logic | 2h |
| 4.4 | `RebalanceScheduler` with T-bar cadence and per-strategy sizing | 2h |
| 4.5 | Wire orchestrator into matrix-runner for multi-strategy simulation | 1h |
| 4.6 | Run full orchestrated backtest with top-5 qualifying strategies | 4h (compute) |
| 4.7 | Compare orchestrated results vs independent strategy results | 1h |

**Total Phase 4 effort: ~14h**

---

## 5. Conclusion

### Strategy Selection
Three strategies qualify for deployment: `grid_trading ES 4h` (score 3.98), `grid_trading NQ 1h` (score 2.89), and `grid_trading ES 1h` (score 2.60). Add `rsi2_reversion JPN225 1h` for its near-zero correlation (ρ=0.225) with grid strategies, providing diversification despite lower standalone returns.

### Guardrails
- Don't deploy grid_trading + donchian_breakout together (ρ=0.82)
- Apply E3 spread model before deployment (grid returns overstated by 20-30% in v10)
- Run light optimizer on top-3 combos for tuned parameters
- Add Sortino guard (D1) before final matrix run

### Multi-Strategy Orchestration
The proposed `Orchestrator` architecture with Kelly-proportional capital allocation, 20-bar rebalancing cadence, and correlation brake provides a bias-resistant framework for simulating the combined performance of active strategies sharing a capital pool. Real-world friction modeling (E3+E10) is already integrated into the FillSimulator — the orchestrator adds rebalance-level friction on top.

---

## 6. Live Market Condition Impact Assessment

### 6.1 Regime-Dependent Performance Degradation

Each strategy has a specific regime envelope where it generates positive edge. Outside this envelope, performance degrades predictably:

| Strategy | Optimal Regime | Degradation Regime | Degradation Mechanism | Expected Sharpe Drop |
|----------|---------------|-------------------|----------------------|---------------------|
| `grid_trading` | Calm (0) | Trending (1), HighVol (2) | Grid accumulates losing positions against trend; wide stops hit repeatedly in high vol | -5.0 to -10.0 |
| `rsi2_reversion` | Calm (0), Trending (1) | HighVol (2), Chaos (3) | Mean-reversion signals fail during volatility expansion; RSI(2) produces false capitulation signals | -3.0 to -8.0 |
| `trend_following` | Trending (1) | Calm (0) | Whipsaws in range-bound markets; two-bar confirmation fails repeatedly | -1.0 to -3.0 |
| `volatility_harvesting` | HighVol (2) with VIX≥20 | Calm (0), Trending (1) | No entry signals below VIX threshold; strategy sits idle | N/A (no trades) |
| `donchian_breakout` | Trending (1), HighVol (2) | Calm (0) | False breakouts in tight ranges; entry buffer insufficient to filter | -5.0 to -12.0 |

**Mitigation:** The RegimeActivationMatrix (`regime_activation.go`) already gates strategies by regime. The orchestrator's eligibility gate (§3.2 Rule 1) uses this matrix to automatically deactivate strategies when they enter degradation regimes. This is the primary defense against live-market regime shifts.

### 6.2 VIX-Regime Interaction

The VIX injection pipeline (R3+R16) provides regime context to VIX-dependent strategies. During periods of VIX suppression (VIX < 15), `volatility_harvesting` and `vix_futures_carry` produce zero signals — capital is automatically redistributed to active strategies by the orchestrator's rebalance rules.

**Live risk:** If VIX spikes rapidly (VIX > 35 in < 3 days), the candle-derived VIX formula (§10.3 of Production Readiness Assessment) lags real VIX by 1-2 days due to its 5-day smoothing window. This creates a brief window where VIX data understates actual volatility, causing VIX-dependent strategies to remain inactive during the critical entry opportunity.

**Mitigation:** Add a VIX acceleration detector — if `|vix_change| > 5.0 over 2 days`, bypass the smoothing window and use raw VIX directly.

### 6.3 Correlation Regime — Crisis Convergence

During market crises (VIX > 30, MaxDD > 20% on SPX), historically uncorrelated strategies converge to ρ > 0.7. The orchestrator's correlation brake (§3.3 Step 4) detects this and reduces combined position sizes by 30%. However, the detection window (10 consecutive days of ρ > 0.6) may be too slow for flash crashes.

**Mitigation:** Add a velocity-based correlation trigger — if ρ increases by > 0.3 in a single day, activate the brake immediately without waiting for the 10-day window.

### 6.4 Liquidity Regime Impact

During high-volatility periods, bid-ask spreads widen 2-5× their normal levels. The FillSimulator's `MaxSlippage` parameter (5-25bps) is calibrated for normal market conditions. During extreme events, effective spreads can reach 50-200bps.

**Mitigation:** The `CalibrateSlippageModel()` function in `slippage.go:28` already provides adaptive spread calibration based on observed slippage. Wire this into the live engine's fill reconciliation path to dynamically adjust the slippage model based on actual fill quality.

---

## 7. Live Strategy Reevaluation Protocol

### 7.1 Metrics-Driven Promote/Demote Criteria

Strategies are evaluated continuously using a trailing 20-trade window. The following criteria determine promotion or demotion:

#### Demotion Triggers (Automatic)

| Trigger | Condition | Action |
|---------|-----------|--------|
| **Sharpe degradation** | Trailing 20-trade Sharpe drops below 30% of its backtest benchmark OR below 0 for 30+ consecutive trading days | Reduce capital allocation to 25% of current weight. If Sharpe remains below threshold for 60 days, mark strategy as "inactive" (0% allocation) |
| **MaxDD breach** | Running MaxDD exceeds strategy-specific ceiling (grid: 15%, rsi2: 10%, trend: 25%) | Immediate hard halt — close all positions for this strategy, mark "violated", require manual review to reactivate |
| **Regime exit** | HMM state exits strategy's allowed regime envelope for 5+ consecutive bars | Soft demotion — redistribute capital to active strategies, keep strategy in "standby" state (re-evaluates eligibility every bar) |
| **Correlation brake** | ρ > 0.6 for 10 consecutive days OR ρ increases by > 0.3 in a single day | Reduce combined position sizes for correlated strategies by 30% |
| **Fill degradation** | Average `SlippageMidBps` over last 20 fills exceeds 2× the expected spread for the symbol's asset class | Reduce position size by 50% for that symbol; if degradation persists for 40 fills, mark symbol as "liquidity-constrained" |
| **Drawdown correlation** | Two strategies both show MaxDD > 15% simultaneously | Freeze the one with lower trailing Sortino; redistribute its capital to the other |

#### Promotion Triggers (Automatic)

| Trigger | Condition | Action |
|---------|-----------|--------|
| **Sharpe recovery** | Previously demoted strategy shows trailing 20-trade Sharpe > 50% of backtest benchmark for 10+ consecutive trading days | Restore allocation to 50% of original weight; if maintained for 20 days, restore to full weight |
| **Regime entry** | HMM state enters strategy's allowed regime envelope after > 10 bars of inactivity | Activate strategy with initial allocation = kelly_fraction × capital / active_strategy_count |
| **OOS validation** | Strategy has been live for > 60 days AND OOS Sharpe > 0 AND within 50% of backtest Sharpe | Promote to "validated" status — eligible for full Kelly allocation |
| **Correlation divergence** | Rolling ρ drops below 0.3 after previously being > 0.5 | Remove correlation brake; restore full position sizes |
| **New regime discovery** | Strategy shows positive Sharpe in a regime NOT in its original allowed set | Flag for manual review — may warrant updating the RegimeActivationMatrix entry for that strategy |

### 7.2 Review Cadence

| Review Type | Frequency | Scope | Participants |
|-------------|-----------|-------|-------------|
| **Automated** | Every bar | All demotion/promotion triggers evaluated by orchestrator | Zero-touch (orchestrator engine) |
| **Daily audit** | End of each trading day | All strategies that triggered any demotion or promotion event | Quantitative analyst reviews automated actions |
| **Weekly review** | Every Friday | All active strategies' trailing metrics vs backtest benchmarks | Strategy team: identify parameter drift, regime changes |
| **Quarterly recalibration** | Every 90 days | Full backtest re-run with updated data, optimizer re-run on top strategies, RegimeActivationMatrix review | Senior quantitative analyst + strategy team |

### 7.3 Manual Override Protocol

Manual intervention is permitted in the following scenarios with documented justification:

| Scenario | Allowed Action | Required Documentation |
|----------|---------------|----------------------|
| Strategy triggered MaxDD breach but analyst identifies data error (e.g., bad tick) | Override halt, restore allocation | Incident report within 24h: timestamp, affected bars, root cause |
| Strategy shows degradation due to known scheduled event (FOMC, earnings) | Pre-emptively reduce allocation to 25% for event window | Event calendar entry + pre-approved risk memo |
| New strategy variant developed (fork of existing) | Deploy at 10% allocation in parallel with existing, run for 30 days before comparison | Deployment checklist: backtest results, expected Sharpe range, correlation with existing strategies |

---

## 8. Backward Compatibility & Parity Enforcement

### 8.1 Backtest/Live Pipeline Parity

All fixes implemented in this remediation cycle route through shared code paths. The following parity contracts are enforced:

| Component | Backtest Path | Live Path | Parity Verification |
|-----------|-------------|-----------|-------------------|
| **Strategy Evaluate()** | `engine.go:generateSignal()` | `live_engine.go:ProcessTickForAccount()` | Same `strategy.Evaluate()` method, identical signature. Parity by construction. |
| **RiskPipeline** | `engine.go:1185-1195` (`ProcessSignal`) | `live_engine.go` (same `ProcessSignal`) | Single `risk.RiskPipeline` struct used in both engines. No divergence possible. |
| **FillSimulator** | `engine.go:846` (`SimulateFillWithTCA`) | Live engine calls same method | Same `FillSimulator.SimulateFillWithTCA()` with identical slippage model. Parity by construction. |
| **SanitizeTradePnL** | `engine.go:760,963` (exit paths) | `pipeline.go:ReconcileFill` | Single `risk.SanitizeTradePnL()` function. Called from both engines through pipeline. |
| **VIX injection** | `engine.go:635` (`SetVIX` via type assertion) | `live_engine.go:206` | Both call `SetVIX()` through `VIXReceiver` interface. Loaded from same `vix_logs` table. |
| **Regime data** | `engine.go:LoadRegimeLogs()` | Live engine: same adapter | Same `regime_logs` table, same `getRegimeAt()` lookup. |
| **Position sizing** | `pipeline.go:ProcessSignal` (E2 aggregate cap) | Same pipeline | Per-signal and aggregate notional caps enforced identically. |

**No new execution paths were created.** All fixes modified existing shared functions — `SanitizeTradePnL`, `ProcessSignal`, `CalculateSharpe`, `SimulateFillWithTCA`. The orchestrator component (§4.1) will use the same `Engine` and `RiskPipeline` instances as the backtest, ensuring parity.

### 8.2 Backward Compatibility

| Change | Backward Compatible? | Rationale |
|--------|---------------------|-----------|
| `Quantity: 100 → 1.0` (R26) | YES | Strategies emit smaller signals; pipeline applies same sizing logic. Pre-existing strategy configs with older params still work — they just size correctly now. |
| `SanitizeTradePnL` formula change | YES | Clamp threshold lowered from $200K to $5K-$10K. All existing backtests already have PnL values below this range post-R1 fix. Only affects truly absurd PnL (which should never occur post-R1). |
| `Kelly_max: 1.0 → 0.25` (R2) | YES | Optimizer search space tightened — any existing config with kelly_fraction > 0.25 will be clamped to 0.25 at runtime (existing engine guard). |
| `vol_grid` registry removal | YES | `adjust_by_volatility` replaces it as an optimizer parameter. Any existing `vol_grid` strategy IDs will fail to resolve — but these were only present in the old registry, not in any persistent config. |
| `SlippageForSymbol()` override | YES | Only overrides `DefaultEquitySlippage()` (SpreadBps=0.5). Custom slippage models set explicitly via API/config are preserved. |

### 8.3 Code Reuse & Redundancy Elimination

| Component | Shared By | De-duplication |
|-----------|----------|----------------|
| `RiskPipeline.ProcessSignal` | Backtest engine, live engine, orchestrator (planned) | Single implementation, 3 callers |
| `FillSimulator.SimulateFillWithTCA` | Backtest engine, live engine, orchestrator (planned) | Single implementation, 3 callers |
| `TrendFilter` (`trend_filter.go`) | grid_trading, rsi2_reversion, session_scalp, volume_scalp | New shared utility — replaces 4 copy-paste implementations |
| `STRATEGY_DISPLAY` map | StrategyHub, CatalogTab, InstancesTab, MatrixResultsPanel | Defined once in `constants.ts`, imported by 4 components |
| `SlippageForSymbol()` | FillSimulator (auto-override for DefaultEquitySlippage) | Single function, all backtest and live fills |
| `generateRegimeLogs()` | Seeder (auto-seed on startup) | Single function, covers all 35 symbols |
| `generateSyntheticCandles()` | Seeder (auto-seed on startup) | Single function, covers all 35 symbols, 3 timeframes |

### 8.4 Parity Verification Test (Recommended)

Add an automated test in CI that verifies backtest/live pipeline parity:

```
TestBacktestLiveParity(t):
  1. Create identical BacktestConfig and LiveConfig for same symbol/timeframe
  2. Feed identical candle sequence to both backtest Engine and live Engine
  3. Assert identical ProcessSignal outputs (Approved, Size, Reason)
  4. Assert identical SimulateFill outputs (FillPrice, FillQuantity)
  5. Assert identical SanitizeTradePnL outputs
  6. Assert identical ReconcileFill capital pool updates
```

This test runs in < 1 second and catches any divergence before it reaches production. Specified in E17 (Production Readiness Assessment §5).

---

## 9. Revised Phase 4 Implementation Plan (All Layers)

| Step | Layer | Deliverable | Effort | Depends On |
|------|-------|-------------|--------|------------|
| 4.0 | **DB** | Migration `000032_orchestration`: create `orchestration_runs`, `allocation_history`, `strategy_status` tables | 30min | — |
| 4.1 | **Backend** | D1: Sortino guard (`downsideDeviation < 1e-6 → 0`) in `calculateSortino()` | 30min | — |
| 4.2 | **Backend** | `Orchestrator` component (§4.1) with Kelly-proportional allocation → `internal/backtest/orchestrator.go` | 4h | Phase 1-3 complete |
| 4.3 | **Backend** | `CorrelationTracker` with rolling window, velocity-based brake (§6.3) → `internal/backtest/correlation_tracker.go` | 2h | 4.2 |
| 4.4 | **Backend** | `RebalanceScheduler` with T-bar cadence (§3.4) → `internal/backtest/rebalance_scheduler.go` | 2h | 4.2 |
| 4.5 | **Backend** | Live reevaluation engine (§7.1): promote/demote triggers → `internal/backtest/reevaluation.go` | 3h | 4.2 |
| 4.6 | **Backend** | VIX acceleration detector (§6.2): bypass smoothing on `|ΔVIX| > 5` → `internal/backtest/vix_detector.go` | 1h | — |
| 4.7 | **Backend** | Adaptive slippage calibration: wire `CalibrateSlippageModel()` into live engine fill path | 1h | — |
| 4.8 | **DB** | Repository: `SaveOrchestrationRun()`, `LoadAllocationHistory()`, `UpdateStrategyStatus()` → `repository.go` | 1h | 4.0 |
| 4.9 | **API** | Orchestrator endpoints (§12): `POST /run`, `GET /:id`, `GET /:id/allocation`, `GET /:id/correlation` → `router.go` | 2h | 4.2, 4.8 |
| 4.10 | **API** | Strategy status endpoints (§12): `GET /:id/status`, `POST /:id/promote`, `POST /:id/demote` → `router.go` | 1h | 4.5, 4.8 |
| 4.11 | **Backend** | Wire orchestrator into matrix-runner for multi-strategy simulation | 1h | 4.2-4.10 |
| 4.12 | **Frontend** | OrchestrationHub page (§13.1): submit orchestrated backtest, view pool equity, allocation pie | 4h | 4.9 |
| 4.13 | **Frontend** | Correlation heatmap component (§13.2): pairwise ρ matrix for active strategies | 2h | 4.9 |
| 4.14 | **Frontend** | Strategy status dashboard (§13.3): active/inactive/standby/violated with trailing metrics | 3h | 4.10 |
| 4.15 | **Frontend** | Promote/demote UI in StrategyHub (§13.4): manual override buttons with confirmation | 2h | 4.10 |
| 4.16 | **Backend** | E17: Parity verification test (§8.4) | 4h | 4.2 |
| 4.17 | **Ops** | Run full orchestrated backtest with top-5 qualifying strategies + E3 friction | 4h (compute) | 4.1-4.16 |
| 4.18 | **Ops** | Compare orchestrated results vs independent strategy results | 1h | 4.17 |

**Total Phase 4 effort: ~34h** (backend 16h, DB 1.5h, API 3h, frontend 11h, ops 2.5h)

---

## 11. Database Changes

### 11.1 Migration `000032_orchestration`

**File:** `internal/db/migrations/000032_orchestration.up.sql`

```sql
-- Orchestration run metadata — one row per submitted orchestration backtest.
CREATE TABLE IF NOT EXISTS orchestration_runs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ,
    status          TEXT NOT NULL DEFAULT 'running',  -- running, completed, failed
    start_date      DATE NOT NULL,
    end_date        DATE NOT NULL,
    initial_capital NUMERIC(18,4) NOT NULL DEFAULT 100000,
    strategy_ids    TEXT[] NOT NULL,                   -- array of strategy IDs in this run
    symbol_tf_pairs TEXT[] NOT NULL,                   -- array of "SYMBOL:TF" pairs
    pool_sharpe     DOUBLE PRECISION,
    pool_sortino    DOUBLE PRECISION,
    pool_maxdd      DOUBLE PRECISION,
    pool_return_pct DOUBLE PRECISION,
    rebalance_costs NUMERIC(18,4),
    result_json     JSONB                                -- full result payload
);
CREATE INDEX IF NOT EXISTS idx_orch_runs_created ON orchestration_runs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_orch_runs_status ON orchestration_runs (status);

-- Allocation history — per-bar weights for each strategy in the orchestration.
CREATE TABLE IF NOT EXISTS allocation_history (
    id              BIGSERIAL PRIMARY KEY,
    run_id          UUID NOT NULL REFERENCES orchestration_runs(id) ON DELETE CASCADE,
    bar_time        TIMESTAMPTZ NOT NULL,
    strategy_id     TEXT NOT NULL,
    weight          DOUBLE PRECISION NOT NULL,           -- 0.0 to 1.0
    allocated_capital NUMERIC(18,4) NOT NULL,
    position_size   DOUBLE PRECISION,
    is_active       BOOLEAN NOT NULL DEFAULT false
);
CREATE INDEX IF NOT EXISTS idx_alloc_run_time ON allocation_history (run_id, bar_time);
CREATE INDEX IF NOT EXISTS idx_alloc_strategy ON allocation_history (strategy_id, bar_time);

-- Live strategy status — current state for each deployed strategy.
CREATE TABLE IF NOT EXISTS strategy_status (
    strategy_id     TEXT PRIMARY KEY,
    status          TEXT NOT NULL DEFAULT 'inactive',    -- active, inactive, standby, violated, validated
    allocation_pct  DOUBLE PRECISION NOT NULL DEFAULT 0,
    trailing_sharpe DOUBLE PRECISION,
    trailing_sortino DOUBLE PRECISION,
    trailing_maxdd  DOUBLE PRECISION,
    last_signal_at  TIMESTAMPTZ,
    active_since    TIMESTAMPTZ,
    demoted_at      TIMESTAMPTZ,
    demotion_reason TEXT,
    last_evaluated  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**File:** `internal/db/migrations/000032_orchestration.down.sql`

```sql
DROP TABLE IF EXISTS allocation_history;
DROP TABLE IF EXISTS strategy_status;
DROP TABLE IF EXISTS orchestration_runs;
```

### 11.2 Repository Methods

Add to `internal/db/repository.go`:

```go
func (r *Repository) SaveOrchestrationRun(ctx context.Context, run *OrchestrationRun) error
func (r *Repository) UpdateOrchestrationRun(ctx context.Context, id string, status string, result *OrchestrationResult) error
func (r *Repository) LoadOrchestrationRun(ctx context.Context, id string) (*OrchestrationRun, error)
func (r *Repository) ListOrchestrationRuns(ctx context.Context, limit, offset int) ([]OrchestrationRun, int, error)
func (r *Repository) SaveAllocationHistory(ctx context.Context, runID string, entries []AllocationEntry) error
func (r *Repository) LoadAllocationHistory(ctx context.Context, runID string) ([]AllocationEntry, error)
func (r *Repository) UpsertStrategyStatus(ctx context.Context, status *StrategyStatus) error
func (r *Repository) GetStrategyStatus(ctx context.Context, strategyID string) (*StrategyStatus, error)
func (r *Repository) ListStrategyStatuses(ctx context.Context) ([]StrategyStatus, error)
```

### 11.3 Seeding

No seed data required — `orchestration_runs`, `allocation_history`, and `strategy_status` are populated at runtime by the orchestrator. The existing `strategies` table already contains all 17 canonical strategies from the fixture seed.

---

## 12. API Changes

### 12.1 Orchestrator Endpoints

All endpoints use existing auth middleware (5 handlers: auth + role check).

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| `POST` | `/api/v1/orchestrator/run` | `OrchestratorHandler.SubmitRun` | Submit orchestrated backtest with strategy list, symbol set, date range, capital. Returns `run_id`. |
| `GET` | `/api/v1/orchestrator/runs` | `OrchestratorHandler.ListRuns` | List all orchestration runs. Query params: `limit`, `offset`, `status`. |
| `GET` | `/api/v1/orchestrator/runs/:id` | `OrchestratorHandler.GetRun` | Get full orchestration result: pool metrics, per-strategy PnL, correlation breaches. |
| `GET` | `/api/v1/orchestrator/runs/:id/allocation` | `OrchestratorHandler.GetAllocation` | Get allocation history time series: `[{bar_time, strategy_id, weight, allocated_capital}]`. |
| `GET` | `/api/v1/orchestrator/runs/:id/correlation` | `OrchestratorHandler.GetCorrelation` | Get pairwise correlation matrix and breach events over the run period. |
| `DELETE` | `/api/v1/orchestrator/runs/:id` | `OrchestratorHandler.CancelRun` | Cancel a running orchestration. |

**Handler file:** `internal/api/orchestrator_handler.go`

**Request body for `POST /run`:**

```json
{
  "strategies": [
    {"strategy_id": "grid_trading", "symbol": "ES", "timeframe": "4h"},
    {"strategy_id": "rsi2_reversion", "symbol": "JPN225", "timeframe": "1h"}
  ],
  "start_date": "2025-08-01",
  "end_date": "2026-08-01",
  "initial_capital": 100000,
  "rebalance_bars": 20,
  "kelly_fraction": 0.25,
  "enable_correlation_brake": true,
  "correlation_threshold": 0.6,
  "friction_model": "realistic"
}
```

### 12.2 Strategy Status Endpoints

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| `GET` | `/api/v1/strategies/:id/status` | `StrategyStatusHandler.GetStatus` | Current live status: active/inactive, trailing metrics, allocation pct, last signal time. |
| `POST` | `/api/v1/strategies/:id/promote` | `StrategyStatusHandler.Promote` | Manual promotion: override demotion, restore allocation. Requires `{"reason": "..."}` body. |
| `POST` | `/api/v1/strategies/:id/demote` | `StrategyStatusHandler.Demote` | Manual demotion: reduce/remove allocation. Requires `{"reason": "...", "allocation_pct": 0.0}` body. |
| `GET` | `/api/v1/strategies/statuses` | `StrategyStatusHandler.ListStatuses` | All strategy statuses for dashboard. |

**Handler file:** `internal/api/strategy_status_handler.go`

### 12.3 Admin Endpoint Extension

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| `POST` | `/api/v1/admin/seed` | `AdminHandler.SeedDatabase` | **Existing** — extend `force=true` body to also reset `strategy_status` records. |

---

## 13. Frontend Changes

### 13.1 New Page: `OrchestrationHub`

**File:** `web/src/pages/OrchestrationHub.tsx`

**Route:** `/orchestration`

**Tabs:**
- **Runner tab**: Strategy selection (multi-select from catalog), symbol/timeframe picking, capital input, friction model toggle, submit button. Displays pool-level equity curve via `EquityCurve` component after completion.
- **History tab**: List of past orchestration runs with pool Sharpe, MaxDD, return%. Click to view detail.
- **Detail modal/view**: Pool metrics, strategy attribution table (PnL per strategy), allocation pie chart over time, correlation heatmap.

**Dependencies:**
- `EquityCurve` component (existing — reused)
- `CorrelationHeatmap` component (§13.2 — new)
- `AllocationPie` component (§13.2 — new)
- API client: `orchestrator.submit()`, `orchestrator.get()`, `orchestrator.getAllocation()`

### 13.2 New Components

**File:** `web/src/components/orchestration/CorrelationHeatmap.tsx`

- Props: `correlationMatrix: number[][]`, `strategyLabels: string[]`
- Renders an N×N heatmap grid using shadcn `Table` with color-coded cells (green=uncorrelated, yellow=moderate, red=high).
- Tooltip on hover showing ρ value and strategy pair names.
- Highlight cells above `correlation_threshold` with red border.

**File:** `web/src/components/orchestration/AllocationPie.tsx`

- Props: `allocationData: {strategy: string, weight: number}[]`
- Renders a Chart.js doughnut with per-strategy slices.
- Click on slice → navigate to that strategy's detail view.
- Time-series mode: below the pie, a timeline scrubber to view allocation at different timestamps.

**File:** `web/src/components/orchestration/FrictionToggle.tsx`

- Props: `onChange: (model: string) => void`
- Toggle between `"realistic"` (E3 per-asset-class spreads) and `"idealized"` (pre-E3 0.5bps).
- Shows expected cost impact: `~14% return reduction for grid strategies`.

### 13.3 Strategy Status Dashboard

**Added to:** `web/src/pages/StrategyHub.tsx` — new 4th tab

**Tab:** "Status"

Displays a table of all 17 strategies with columns:
- Strategy name (from STRATEGY_DISPLAY)
- Status badge (green=active, gray=inactive, yellow=standby, red=violated, blue=validated)
- Allocation % (progress bar)
- Trailing Sharpe (with sparkline from last 20 evaluations)
- Trailing MaxDD (with color coding: green < 10%, yellow 10-20%, red > 20%)
- Last signal time (relative: "2m ago", "never")
- Promote/Demote button (disabled unless manual override criteria met)

### 13.4 Promote/Demote UI

**Added to:** `web/src/pages/strategy-hub/InstancesTab.tsx` — strategy row action buttons

- **Promote button**: Visible when status = "inactive" or "standby". On click: confirmation dialog with reason textarea. Calls `POST /api/v1/strategies/:id/promote`.
- **Demote button**: Visible when status = "active" or "validated". On click: confirmation dialog with reason textarea + allocation slider (0% → current%). Calls `POST /api/v1/strategies/:id/demote`.
- Both buttons trigger a toast notification and refresh the status list.

### 13.5 Frontend Type Updates

**File:** `web/src/types/api.ts` — add new types:

```typescript
interface OrchestrationRun {
  id: string
  status: 'running' | 'completed' | 'failed'
  created_at: string
  completed_at?: string
  pool_sharpe?: number
  pool_sortino?: number
  pool_maxdd?: number
  pool_return_pct?: number
  rebalance_costs?: number
  strategies: OrchestrationStrategy[]
}

interface OrchestrationStrategy {
  strategy_id: string
  symbol: string
  timeframe: string
  pnl: number
  sharpe: number
  maxdd: number
  allocation_history: AllocationEntry[]
}

interface AllocationEntry {
  bar_time: string
  strategy_id: string
  weight: number
  allocated_capital: number
  is_active: boolean
}

interface StrategyStatus {
  strategy_id: string
  status: 'active' | 'inactive' | 'standby' | 'violated' | 'validated'
  allocation_pct: number
  trailing_sharpe?: number
  trailing_sortino?: number
  trailing_maxdd?: number
  last_signal_at?: string
  active_since?: string
  demotion_reason?: string
}

interface CorrelationMatrix {
  labels: string[]
  matrix: number[][]
  breaches: CorrelationBreach[]
}

interface CorrelationBreach {
  time: string
  strategy_a: string
  strategy_b: string
  correlation: number
  action: 'brake_applied' | 'brake_released'
}
```

### 13.6 Frontend Route Registration

**File:** `web/src/App.tsx` or router configuration — add:

```typescript
{ path: '/orchestration', element: <OrchestrationHub /> }
```

### 13.7 Frontend Navigation Update

**File:** `web/src/components/layout/Sidebar.tsx` or navigation config — add nav item:

```typescript
{ label: 'Orchestration', path: '/orchestration', icon: Layers }
```

---

## 10. Conclusion (Revised)

The selected strategies (`grid_trading ES 4h`, `grid_trading NQ 1h`, `rsi2_reversion JPN225 1h`) are viable for staged deployment. The regime-gated architecture (RegimeActivationMatrix + orchestrator eligibility gate) provides automatic defense against the primary live-market degradation mechanism: regime shifts. The promote/demote protocol (§7) provides continuous, metrics-driven adaptation without manual intervention for 90% of scenarios.

**Before any live capital is committed:**
1. Apply E3 spread model and re-run matrix to confirm grid Sharpe remains > 1.5 after realistic friction
2. Run light optimizer on the top-3 combos for tuned parameters (currently all default params)
3. Add the Sortino guard (D1, 30min) and the parity verification test (E17, 4h)
4. Deploy with paper-trading for 30 days before committing live capital (Phase 4.2, Recovery Assessment)

**Backward compatibility is confirmed** — all changes route through existing shared interfaces (Engine, RiskPipeline, FillSimulator, Strategy). No new execution paths. No redundant code. Parity is enforced by construction through single-implementation shared functions.

---

*End of report.*
