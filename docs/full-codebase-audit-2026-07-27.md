# OrcaAlgo Full Codebase Audit Report

**Date:** 2026-07-27
**Scope:** All layers — Python (`orca/`), Go (`internal/`), React (`web/src/`), cross-language wiring, test coverage
**Methodology:** 4 parallel exploration agents exhaustively read source files across every package

---

## Executive Summary

| Area | Compliance | Critical Issues |
|------|-----------|-----------------|
| Python math/sizing/models | **PASS** — 19/19 Pydantic models frozen, all canonical math correct | None |
| Python calibration/preflight/attribution | **PASS** — fully functional, all CLI commands present | None |
| Go RiskPipeline + interfaces | **PASS** — wired into both backtest and live engines | PropFirmEnforcer lacks compile-time interface check |
| Go KillSwitch + capital pools | **PASS** — re-entrancy guard, OnTrigger callback propagates to pools | None |
| Go backtest engine | **PARTIAL** | Sharpe annualization inconsistency, FillModel interface unused |
| Go live engine | **PASS** — per-account isolation, pipeline integration | Kelly fraction applied (fixed since audit doc) |
| React frontend | **PARTIAL** | 2 hardcoded indicators, LWC violations in ChartingHub, 22 redirect routes |
| Test coverage — Python | **GOOD** — ~80% module coverage | Temporal ports, drift detection, HMM training untested |
| Test coverage — Go | **GOOD** — risk pipeline, kill switch, propfirm well tested | Security, Alpaca broker, DB repo very light |
| Test coverage — Frontend | **POOR** | **0 E2E tests**, no page component tests, only stores/hooks covered |

---

## PART I — PYTHON LAYER (`orca/`)

### 1.1 Domain Models — COMPLIANT

All 19 Pydantic models across `orca/models/strategy.py`, `orca/models/trade.py`, `orca/models/risk.py` use `ConfigDict(frozen=True, extra="forbid")`. Also `orca/config/feature_flags.py:23` — `FeatureFlags`.

| File | Model | Line | frozen | extra="forbid" |
|------|-------|------|--------|----------------|
| `strategy.py` | `TokenRef` | 8 | YES | YES |
| `strategy.py` | `TypeSpec` | 15 | YES | YES |
| `strategy.py` | `InputSpec` | 23 | YES | YES |
| `strategy.py` | `PortTemporalSpec` | 31 | YES | YES |
| `strategy.py` | `TemporalRule` | 40 | YES | YES |
| `strategy.py` | `OutputSpec` | 57 | YES | YES |
| `strategy.py` | `PortSignature` | 65 | YES | YES |
| `strategy.py` | `Node` | 72 | YES | YES |
| `strategy.py` | `StrategyBody` | 83 | YES | YES |
| `strategy.py` | `Capability` | 92 | YES | YES |
| `strategy.py` | `StrategyIRV04` | 105 | YES | YES |
| `strategy.py` | `RiskProfile` | 116 | YES | YES |
| `trade.py` | `TradeSignal` | 13 | YES | YES |
| `trade.py` | `Order` | 23 | YES | YES |
| `trade.py` | `Fill` | 38 | YES | YES |
| `trade.py` | `Position` | 50 | YES | YES |
| `risk.py` | `BreachCondition` | 12 | YES | YES |
| `risk.py` | `KillSwitchState` | 21 | YES | YES |
| `risk.py` | `RiskSnapshot` | 29 | YES | YES |

**Verdict:** Rule #7 (DON'T mutate domain models) — **fully compliant.**

### 1.2 Canonical Math — ALL CORRECT

| Function | File:Line | Verification |
|----------|-----------|-------------|
| `kelly_fraction_binary` | `sizing/kelly.py:17` | Standard Kelly criterion for binary outcome — correct |
| `kelly_fraction_continuous` | `sizing/kelly.py:33` | `f = (p*R - (1-p)) / R` — correct |
| `kelly_with_attenuators` | `sizing/kelly.py:41` | Default k=0.25, all 3 attenuators (edge discount, fractional, caps) — **matches Rule #6** |
| `ewma_volatility` | `sizing/volatility.py:6` | RiskMetrics approach, alpha = 2/(span+1) — correct |
| `vol_adjusted_size` | `sizing/volatility.py:21` | Ratio capped [0.5, 2.0] — correct |
| `diversification_scaling` | `sizing/volatility.py:30` | `1/sqrt(N_eff)`, clamped [0.25, 1.0] — correct |
| `brier_score` | `math/brier.py:24` | Standard Brier formula — correct |
| `murphy_decomposition` | `math/brier.py:33` | Brier = REL - RES + UNC, per-bin — correct |
| `platt_scale` | `math/platt.py:26` | Nelder-Mead logit calibration, train/val split — correct |
| `wilson_ci` | `math/wilson.py:6` | Standard Wilson score interval — correct |

**Verdict:** Rule #1 (DON'T reimplement canonical math in Go) — **fully compliant.** All math resides in Python.

### 1.3 CLI Entry Points — ALL PRESENT (16 commands)

| Command | `orca/cli.py` Line | Status |
|---------|-------------------|--------|
| `orca validate` | 17 | 4 profiles: research/paper/pretrade/production_guarded |
| `orca calibrate` | 57 | Murphy decomposition + Platt scaling per side |
| `orca preflight` | 130 | 12-point checklist, exits code 1 on failure |
| `orca attribute` | 199 | 4-dimensional PnL slicing (side/price/edge) |
| `orca data-validate` | 233 | Gap/outlier/volume checks |
| `orca hash` | 91 | Graph/param/instance hashes via SHA-256 |
| `orca ir-compile` | 298 | GKR IR → Go JSON config |
| `orca simulate *` | 324-830 | 11 sub-commands for synthetic data pipeline |

### 1.4 Python Issues Found

None critical. Minor observations:

| Issue | Severity | Detail |
|-------|----------|--------|
| `orca/ports/temporal.py` untested | MEDIUM | Temporal contract validation has zero test coverage |
| `orca/ml/drift_detection.py` untested | HIGH | PSI-based drift detection — production-critical, no tests |
| `orca/ml/inference.py` untested | MEDIUM | ML inference subprocess pipeline has no dedicated tests |
| `orca/backtest/monte_carlo.py` untested | MEDIUM | Python Monte Carlo has no dedicated test |

---

## PART II — GO BACKEND (`internal/`)

### 2.1 Risk Pipeline Architecture — IMPLEMENTED

The `RiskPipeline` plan from `docs/capital-pool-live-wiring-plan-2026-07-25.md` has been **substantially implemented**:

| Plan Component | Status | Location |
|---------------|--------|----------|
| `SignalGate` interface | **PRESENT** | `internal/risk/interfaces.go:78-98` |
| `CapitalGate` interface | **PRESENT** | `internal/risk/interfaces.go:10-34` |
| `PropFirmGate` interface | **PRESENT** | `internal/risk/interfaces.go:39-73` |
| `RiskPipeline` struct | **PRESENT** | `internal/risk/pipeline.go:33-38` |
| `ProcessSignal` canonical pipeline | **PRESENT** | `internal/risk/pipeline.go:43-109` (8-step sequence) |
| `ReconcileFill` canonical pipeline | **PRESENT** | `internal/risk/pipeline.go:113-128` |
| `ReconcileFillWithoutPropFirm` variant | **PRESENT** | `internal/risk/pipeline.go:133-140` (for backtest's separate ftmo.OnFill) |
| `BaseCapitalPool` (as `PoolState`) | **PRESENT** | `internal/propfirm/pool_base.go:3-15` |
| Wired into backtest `Engine` | **PRESENT** | `internal/backtest/engine.go:1082-1098` (`generateSignal` calls pipeline) |
| Wired into `LiveEngine` | **PRESENT** | `internal/engine/live_engine.go:257-276` (`ProcessTickForAccount` calls pipeline) |
| KillSwitch → pool propagation | **PRESENT** | `internal/risk/kill_switch.go:139-143` (OnTrigger → MarkAllViolated) |
| Per-account strategy isolation | **PRESENT** | `internal/engine/live_engine.go:74-93` (`RegisterAccountStrategies` creates per-account registries) |

### 2.2 Interface Implementation Status

| Interface | Backtest Impl | Live Impl | Compile-Time Check |
|-----------|--------------|-----------|-------------------|
| `CapitalGate` | `CapitalPoolSim` (adapter methods) | `CapitalPoolManager` | Live: YES (`cap_pool.go:271`). Backtest: NO explicit check |
| `PropFirmGate` | `PropFirmEnforcer` (adapter methods, but `IsHalted`/`HaltReason` are **fields** not methods) | `propfirm.Manager` (adapter methods in `profile.go:237-289`) | Neither has explicit `var _` check |
| `SignalGate` | Not separately — wired inline via `PositionSizer`/`ExposureTracker`/`VolatilityHalt` | Same inline approach | N/A |
| `AccountCanceller` | N/A | `broker.AccountManager` | YES (`account_manager.go:352-366`) |

### 2.3 Specific Wiring Verification

**LiveEngine.ProcessTick** (`internal/engine/live_engine.go:179-288`):
1. Adversarial check → `risk.CheckAdversarial` (L210)
2. ML meta-labeler gating (L227)
3. Kelly fraction scaling (L249, default 0.25) — **Fixed since audit doc**
4. RiskPipeline audit via `e.pipeline.ProcessSignal` (L257-276) — **WIRED**
5. Exit ML dynamic stops (L278)
6. HMM regime updates (L188)

**Backtest Engine.generateSignal** (`internal/backtest/engine.go:967-1106`):
1. Capital > 0 check → volHalt → sizing (Kelly, seasonality)
2. Strategy evaluation → ML gate → position sizing (3% cap)
3. Exposure check inline
4. Pipeline audit via `e.pipeline.ProcessSignal` (L1082-1098) — **WIRED**

### 2.4 Backend Issues Found

#### ISSUE B1: Sharpe Ratio Annualization Inconsistency (HIGH)

`internal/backtest/metrics.go:83-93` (`computeSharpe`) uses `math.Sqrt(252)` regardless of timeframe. For 1-minute bars (390/day), this dramatically under-annualizes. The engine's `calculateSharpe` (`engine.go:1233-1283`) correctly converts to daily returns first.

**Impact:** Frontend metrics display different Sharpe values than engine-calculated ones for intraday backtests.

**Fix:** Convert equity to daily returns in `computeSharpe` before annualizing, or pass `barsPerDay` as parameter.

#### ISSUE B2: PropFirmEnforcer Interface Gap (MEDIUM)

`PropFirmEnforcer` has `IsHalted` and `HaltReason` as **fields** (not methods), so it does not fully satisfy `PropFirmGate` interface at compile time. The backtest engine accesses these as `e.ftmo.IsHalted` (field) rather than `e.ftmo.IsHalted()` (method). Works at runtime but lacks compile-time safety.

**Fix:** Add `IsHalted() bool` and `HaltReason() string` methods that return the fields, plus `var _ risk.PropFirmGate = (*PropFirmEnforcer)(nil)` compile assertion.

#### ISSUE B3: Unused FillModel Interface (LOW)

`internal/model/fill.go:5` defines `FillModel` interface with `MidPriceFill` and `ProbabilisticFill` implementations. The `Engine` struct has `fillModel` field (`engine.go:225`) but it's never called during `Run()`. The active fill implementation is `FillSimulator` (`backtest/slippage.go:19`).

**Impact:** Code drift — two parallel fill abstractions with one unused.

#### ISSUE B4: Monte Carlo from Trades vs Random Walk (MEDIUM)

`RunMonteCarloFromTrades` (`monte_carlo.go:245`) correctly bootstraps actual trade PnL. But `RunMonteCarloWithContext` (`monte_carlo.go:310`) uses pure random walks. The router at `router.go:925` calls `RunMonteCarloWithContext` with config from `MonteCarloFromTrades(nil, 1000, req.Capital)` — passing `nil` trades → falls through to random walk.

**Fix:** Router should call `RunMonteCarloFromTrades` when trades are available.

#### ISSUE B5: Scheduler No Cron Parsing (LOW)

`scheduler.go:151` uses `time.Timer` on 24h loop ignoring the `Schedule` field on `Job`. Not a bug if only daily jobs exist, but misleading.

#### ISSUE B6: No Config Package (LOW)

AGENTS.md lists a `config/` Go package but `internal/config/` doesn't exist. Configuration is scattered across env vars in individual packages (`db/repository.go`, `backtest/execution_config.go`, `backtest/light_optimizer.go`, `api/middleware/middleware.go`, `monitor/ws_hub.go`).

---

## PART III — REACT FRONTEND (`web/src/`)

### 3.1 Prior Audit Doc Cross-Reference

The `frontend-backtest-audit-2026-07-24.md` identified 20 issues. Status as of 2026-07-27:

| Audit Item | Priority | Status |
|-----------|----------|--------|
| **P0-1:** Fix hardcoded system status | CRITICAL | **Partially resolved** — 4/6 use real data, 2 still hardcoded (`Kill Switch Active`, `Auth Enforced` at `monitor/OverviewTab.tsx:97,99`) |
| **P0-2:** Delete duplicate CredentialManagement | CRITICAL | **Resolved** — file deleted, route redirects to IntegrationsPage |
| **P0-3:** Fix Monte Carlo from trades | CRITICAL | **Not fixed** — router still calls random walk variant |
| **P1-1:** Delete redundant DataSources | HIGH | **Resolved** — file deleted, route redirects to SettingsPage |
| **P1-2:** Remove Indicators nav item | HIGH | **Resolved** — removed from sidebar |
| **P1-3:** Integrate OptimizationPanel into BacktestHub | HIGH | **Not implemented** — standalone page and API remain separate |
| **P1-4:** Implement MAE/MFE | HIGH | **Already working** — engine.go:641-889 computes MAE/MFE from actual price data (audit doc was incorrect) |
| **P1-5:** Restructure navigation | HIGH | **Resolved** — 3 groups, 13 items implemented |
| **P1-6:** Add warm-up period | HIGH | **Not implemented** |
| **P1-7:** Fix Sharpe annualization | HIGH | **Not fixed** — metrics.go still ignores barsPerDay |
| **P2 items (6)** | MEDIUM | **Not implemented** |
| **P3 items (5)** | LOW | **Not implemented** |

### 3.2 Frontend Issues Found

#### ISSUE F1: Two Hardcoded Status Indicators (MEDIUM)

`web/src/pages/monitor/OverviewTab.tsx:97` — `Kill Switch Active` = `ok: true` (hardcoded)
`web/src/pages/monitor/OverviewTab.tsx:99` — `Auth Enforced` = `ok: true` (hardcoded)

The kill switch state is available via `/api/v1/risk/status` which is already fetched in `MonitorPage.tsx:50`. Auth status can be derived from middleware presence.

#### ISSUE F2: LWC Rule #13 Violations (LOW)

`web/src/pages/ChartingHub.tsx:267` — `chart.applyOptions({ width: chartContainerRef.current.clientWidth })` — should use `chart.resize(width, height)`
`web/src/pages/ChartingHub.tsx:283` — same violation in a rAF callback

#### ISSUE F3: LWC Rule #15 Violation (LOW)

`web/src/pages/ChartingHub.tsx:281` — `requestAnimationFrame` with no `cancelAnimationFrame` in useEffect cleanup.

#### ISSUE F4: 22 Redirect Routes (LOW)

`web/src/App.tsx:62-118` — 22 redirect routes, including 6 legacy ones (`/admin/health`, `/admin/logs`, `/admin/propfirm`, `/admin/symbols`, `/audit`, `/users`). Functionally harmless but adds code clutter.

#### ISSUE F5: SimulatePage 3 Tabs Use Basic MetricCard Grid (LOW)

Calibrate-regime, ticks, and inject-signal tabs render results as basic MetricCard grids rather than structured tables or visualization components.

---

## PART IV — TEST COVERAGE GAPS

### 4.1 Python Tests (46 files, ~80% module coverage)

**Well tested:**
- Kelly sizing (28 test methods), Brier/Murphy (15), Wilson (11), EWMA (22)
- Domain models (all 19), GKR IR loading/validation, calibration audit
- Pre-flight (all 12 checks), Attribution (4 dimensions)
- Guardian smoke tests (16 critical path tests)

**Notable gaps:**

| Module | Gap | Risk |
|--------|-----|------|
| `orca/ports/temporal.py` | **No tests** | HIGH — temporal contract validation |
| `orca/ml/drift_detection.py` | **No tests** | HIGH — PSI-based drift detection, production-critical |
| `orca/ml/inference.py` | **No tests** | MEDIUM — ML inference subprocess |
| `orca/ml/regime_inference.py` | **No tests** | MEDIUM — regime inference |
| `orca/ml/exit_inference.py` | **No tests** | MEDIUM — exit inference |
| `orca/ml/purge_cv.py` | **No tests** | MEDIUM — purged CV |
| `orca/ml/dataset.py` | **No tests** | MEDIUM — core data structures |
| `orca/ml/train/hmm_enhanced.py` | **No tests** | MEDIUM |
| `orca/ml/train/hierarchical.py` | **No tests** | MEDIUM |
| `orca/ml/train/exit_labels.py` | **No tests** | MEDIUM |
| `orca/optimize/bayesian.py` | **No tests** | MEDIUM — Optuna-based optimization |
| `orca/ir/diff.py` | **No tests** | MEDIUM — strategy diff engine |
| `orca/simulation/calibrate.py` | **No tests** | MEDIUM — 432 lines, parameter calibration |
| `orca/simulation/synthetic.py` | **No tests** | MEDIUM — Heston + Jump Diffusion |
| `orca/backtest/monte_carlo.py` | **No tests** | MEDIUM — Python MC |
| `orca/train/hmm.py` | **No tests** | MEDIUM — HMM training |
| `orca/config/feature_flags.py` | **No tests** | LOW |
| `orca/common/` | **No tests** | LOW |

### 4.2 Go Tests (65 files, good coverage in critical areas)

**Well tested:**
- KillSwitch: 200-goroutine re-entrancy test, 100-goroutine concurrent test (`kill_switch_test.go`)
- RiskPipeline: 10 test functions with mock gates, all rejection paths (`pipeline_test.go:99-266`)
- PropFirm: daily loss, DD, consistency, regime multipliers, size chains
- Strategy registry: all 8+ strategy types, factory isolation, param reset
- Monte Carlo: determinism, insufficient returns, summary stats
- Backtest fidelity: fill probability/price bounds, latency, fees
- LiveEngine: pipeline integration, per-account isolation, reconcile fill
- Ring buffer: concurrent push/pop with 1000 goroutines
- PositionSizer, ExposureTracker, VolatilityHalt, CapitalPool

**Critical gaps:**

| Gap | File | Risk |
|-----|------|------|
| **Security** (JWT, password hashing, token rotation) | `internal/security/` | **CRITICAL** — no test files |
| **Alpaca broker adapter** | `internal/broker/alpaca/` | HIGH — no test files |
| **DB repository** | `internal/db/repository_test.go` | HIGH — 39 lines, tests struct fields not DB operations |
| **AdminPage E2E** | — | HIGH — zero test coverage |

### 4.3 Frontend Tests (23 files, severe page-level gaps)

**Well tested:** All 7 Zustand stores, WebSocket hooks, chart hooks (useChart, useChartKeyboard, useIndicator), chart utilities, error boundary.

**Critical gaps:**

| Gap | Detail |
|-----|--------|
| **Playwright E2E** | `web/e2e/` is **completely empty** — 0 files despite AGENTS.md claiming 49 e2e tests |
| **MonitorPage (4 tabs)** | No component tests |
| **BacktestHub (4 views)** | No component tests |
| **StrategyHub (3 tabs)** | No component tests |
| **ChartingHub** | No component tests |
| **AdminPage (9 tabs)** | No component tests |
| **Deploy/PromoteToLiveWizard** | No tests for 3-step gated deploy |
| **All auth pages** | No dedicated page component tests |
| **LiveMonitor, CrosshairTooltip charts** | No tests |
| **IntegrationsPage, AccountsPage, SettingsPage** | No tests |

### 4.4 Tests That May Be Testing the Wrong Thing

- `internal/db/repository_test.go:10-38` — Tests struct field access, not actual DB operations. Essentially a compile check.
- `internal/engine/live_risk_test.go:33-62` — `TestLiveEngine_PipelineIntegration` registers no strategies, checks 0 signals produced — a tautology.
- `web/src/__tests__/equityCurveChart.test.tsx` — 304 lines heavily mocking lightweight-charts. Tests mock behavior more than real chart behavior.
- `web/src/__tests__/pageSmoke.test.tsx` — Only renders EmergencyPage with all deps mocked.

---

## PART V — CROSS-LANGUAGE & INTEGRATION

### 5.1 Go→Python Bridge

Verified via subprocess calls in Go handlers:
- `orca validate` — called from strategy validation
- `orca calibrate` — called from `CalibratePage`
- `orca preflight` — called from `PromoteToLiveWizard`
- `orca attribute` — called from `AttributionPage`
- `orca data-validate` — called from admin

**Status:** All integrations wired correctly.

### 5.2 Go→React Bridge

- WebSocket at `GET /ws` via `gorilla/websocket` (`internal/monitor/ws_hub.go`)
- REST API at `/api/v1/*` with ~50+ endpoints
- JWT AuthMiddleware on protected routes
- 2FA middleware on emergency routes

**Status:** Wired correctly. WSHub uses pub/sub channels with JWT validation.

### 5.3 Database Schema

- BIGINT fixed-point pricing with `PRICE_SCALE_F = 100_000` — **Rule #2 compliant**
- Tables: `candles`, `symbols`, `trade_executions`, `regime_logs`, `backtest_runs`, `strategies`, `providers`, `provider_symbols`, `settings`, `audit_log`, `kill_switch_history`, `consistency_logs`, `universe_config`, `universe_state`, `matrix_progress`, `backtest_results`
- Repository uses `pgx/v5` with connection pooling

---

## PART VI — OVERLAPPING FUNCTIONALITY & CONFLICTS

| # | Duplication/Overlap | Severity | Detail |
|---|---------------------|----------|--------|
| 1 | `FillModel` vs `FillSimulator` | LOW | Two parallel fill abstractions in `model/fill.go` and `backtest/slippage.go`. Only `FillSimulator` is used. |
| 2 | `computeSharpe` in `engine.go` vs `metrics.go` | HIGH | Different annualization — one uses daily returns, the other uses bar returns × sqrt(252). Produces conflicting values. |
| 3 | `CapitalPoolSim` vs `CapitalPoolManager` | LOW | Both embed `propfirm.PoolState` but duplicate drawdown/balance methods. Some consolidation already done via `pool_base.go`. |
| 4 | `OptimizationPanel` (standalone) vs `OptimizationTab` (in BacktestHub) | MEDIUM | Two UIs for the same concept, separate APIs (`/api/v1/optimize/*` vs backtest endpoints). |
| 5 | 22 redirect routes in App.tsx | LOW | 6 legacy + 16 functional redirects. Clutters the router without breaking anything. |
| 6 | Python `orca/optimize/monte_carlo.py` vs `orca/backtest/monte_carlo.py` | LOW | Two Monte Carlo implementations in Python, one likely superseded. |

---

## PART VII — PRIORITIZED REMEDIATION PLAN

### Wave 1 — Critical Fixes (Day 1, ~4h)

| # | Issue | File(s) | Effort |
|---|-------|---------|--------|
| 1.1 | Fix Sharpe annualization in `metrics.go` — convert to daily returns before annualizing | `internal/backtest/metrics.go:83-93` | 0.5h |
| 1.2 | Fix Monte Carlo router to call `RunMonteCarloFromTrades` when trades exist | `internal/api/router.go:925` | 0.5h |
| 1.3 | Add `IsHalted() bool` / `HaltReason() string` methods + `var _ risk.PropFirmGate` check to `PropFirmEnforcer` | `internal/backtest/propfirm_enforcer.go` | 0.25h |
| 1.4 | Add `var _ risk.CapitalGate` compile assertion to `CapitalPoolSim` | `internal/backtest/capital_pool_sim.go` | 0.1h |
| 1.5 | Fix 2 hardcoded status indicators in OverviewTab | `web/src/pages/monitor/OverviewTab.tsx:97,99` | 0.5h |
| 1.6 | Fix ChartingHub LWC violations (resize → applyOptions, rAF cleanup) | `web/src/pages/ChartingHub.tsx:267,281,283` | 0.5h |
| 1.7 | Run full test suite to verify no regressions | `make test` | 1h |

### Wave 2 — Test Coverage (Day 2, ~6h)

| # | Issue | Effort |
|---|-------|--------|
| 2.1 | Create Playwright E2E tests for 5 critical flows (monitor, backtest run, promote-to-live, emergency stop, login) | 3h |
| 2.2 | Add unit tests for `orca/ml/drift_detection.py` | 1h |
| 2.3 | Add unit tests for `orca/ports/temporal.py` | 0.5h |
| 2.4 | Add unit tests for `orca/ml/inference.py` subprocess flows | 1h |
| 2.5 | Expand `internal/db/repository_test.go` with actual DB integration (testcontainers or in-memory) | 0.5h |

### Wave 3 — Feature Completeness (Day 3, ~6h)

| # | Issue | Effort |
|---|-------|--------|
| 3.1 | Add warm-up period to backtest engine (`WarmUpBars` config, skip signal generation during warmup) | 2h |
| 3.2 | Integrate OptimizationPanel into BacktestHub (or add Optimize mode to runner) | 3h |
| 3.3 | Remove/replace `FillModel` interface — either integrate it or delete it | 1h |

### Wave 4 — Polish (Day 4-5, ~8h)

| # | Issue | Effort |
|---|-------|--------|
| 4.1 | Clean up 22 redirect routes in App.tsx (keep only functional ones, remove legacy) | 0.5h |
| 4.2 | Upgrade SimulatePage calibrate-regime/ticks/inject-signal tabs with structured UI | 2h |
| 4.3 | Add drawdown duration visualization to equity curve | 2h |
| 4.4 | Add portfolio-level correlation analysis for multi-strategy backtests | 3h |
| 4.5 | Add volume-dependent slippage to fill model | 1.5h |
| 4.6 | Add Kelly fraction to live position sizing (if not already applied via pipeline) | Verified already applied via pipeline |

---

## PART VIII — COMPLIANCE SUMMARY (AGENTS.md Rules)

| Rule | Description | Status |
|------|-------------|--------|
| #1 | No reimplementation of canonical math in Go | **PASS** — all math in Python `orca/math/`, `orca/sizing/` |
| #2 | No IEEE 754 float for order prices | **PASS** — `Decimal` in Python, `fixed.Fixed`/BIGINT in Go, `PRICE_SCALE_F=100_000` in DB |
| #3 | No legacy YAML — only `.gkr.yaml` | **PASS** — `orca/ir/loader.py` loads `.gkr.yaml`, preflight checks for them |
| #4 | No deploy without preflight | **PASS** — `orca preflight --strict`, 12-point checklist |
| #5 | Quarterly calibration audits | **PASS** — `orca calibrate`, Murphy decomposition + Platt scaling |
| #6 | Fractional Kelly k=0.25 with 3 attenuators | **PASS** — `kelly_with_attenuators(multiplier=0.25, ...)`, all 3 applied |
| #7 | Immutable domain models | **PASS** — 19/19 Pydantic models use `ConfigDict(frozen=True, extra="forbid")` |
| #8 | Kill-switch re-entrancy guard | **PASS** — triple-check: `halted`, `isLocked` CAS, `killSwitchReady`. Tested with 200 goroutines |
| #9 | No perfect fills at mid-price | **PASS** — `FillSimulator` models spread, slippage, partial fills, volume impact |
| #10 | No panic/throw for recoverable errors | **PASS** — CLI uses try/except with graceful exits; Go returns errors |
| #11 | `setData()` only for initial load | **PASS** — `update()` used for incremental updates, `setData()` for full replacement |
| #12 | `fitContent()` only on user action | **PASS** — called on timeframe change, '0' key press, initial load |
| #13 | `chart.resize()` not `applyOptions({width})` | **VIOLATED** — `ChartingHub.tsx:267,283` uses `applyOptions({width})` |
| #14 | `setVisibleLogicalRange()` not `barSpacing` | **PASS** — `useChartKeyboard.ts:26-33` uses logical range |
| #15 | `requestAnimationFrame` cleanup | **VIOLATED** — `ChartingHub.tsx:281` has no cancel in cleanup |
| #16 | `Map.get()` not `Array.find()` in crosshair | **PASS** — all crosshair hooks use O(1) Map lookups |
| #17 | RiskPipeline not bypassed | **PASS** — wired into both backtest and live engines |
| #18 | Per-account strategy isolation | **PASS** — `LiveEngine.RegisterAccountStrategies` creates per-account registries |

**2 of 18 rules have minor violations** (#13, #15), both in `ChartingHub.tsx`, low risk.

---

## Appendix: File Reference Index

### Python — 60 source files in 19 packages
```
orca/cli.py (889 lines, 16 commands)
orca/models/ (4 files: strategy.py, trade.py, risk.py, __init__.py)
orca/math/ (3 files: brier.py, platt.py, wilson.py)
orca/sizing/ (2 files: kelly.py, volatility.py)
orca/calibration/ (3 files: audit.py, platt.py, cli.py)
orca/preflight/ (1 file: checklist.py)
orca/attribution/ (2 files: slicer.py, cli.py)
orca/ir/ (8 files: canonical.py, compiler.py, diagnostics.py, diff.py, loader.py, params_export.py, schema.py, validator.py)
orca/hash/ (3 files: common.py, graph.py, verify.py)
orca/simulation/ (12 files)
orca/ml/ (10 files + train/ subdir with 8 files)
orca/optimize/ (7 files)
orca/vectorbt/ (5 files)
orca/config/, orca/common/, orca/db/, orca/backtest/, orca/risk/, orca/train/, orca/ports/, orca/data_quality/
```

### Go — 65+ test files, 200+ source files in 20+ packages
```
internal/risk/ (27 files incl. tests)
internal/backtest/ (47 files, engine.go is 1892 lines)
internal/engine/ (6 files, live_engine.go is 461 lines)
internal/propfirm/ (5 files)
internal/broker/ (13 files)
internal/api/ (29 files, router.go is 1892 lines)
internal/db/ (8 files, repository.go is 1097 lines)
internal/strategy/ (24 files)
internal/ingest/ (21 files)
internal/monitor/ (8 files)
internal/scheduler/ (2 files)
internal/model/ (5 files: fill.go, fee.go, latency.go, recorder.go, lifecycle.go)
internal/ml/, internal/config/ (does not exist), internal/hash/, internal/llm/, internal/security/
```

### React — 23 test files, 80+ source files
```
web/src/pages/ (20+ page components)
web/src/components/ (40+ domain components + 31 shadcn/ui + 7 layout + 1 deploy)
web/src/charts/ (9 chart components + utilities)
web/src/hooks/ (16 hooks)
web/src/stores/ (7 Zustand stores)
web/src/api/ (3 files: client.ts [413 lines], optimize.ts, middleware.ts)
web/src/types/ (2 files: api.ts [665 lines], ws.ts [122 lines])
web/src/App.tsx (184 lines, 22+ routes)
```

---

## IMPLEMENTATION SUMMARY (2026-07-27)

The remediation plan from Part VII was executed in 4 waves (Waves 1-4) plus post-implementation cleanup. Below is the final status.

### Wave 1 — Critical Fixes

| # | Issue | Status | Files |
|---|-------|--------|-------|
| 1.1 | Sharpe annualization — `equityToDailyReturns` groups equity by day before `sqrt(252)` | **FIXED** | `internal/backtest/metrics.go` (+32 lines) |
| 1.2 | Monte Carlo — boots from OOS window returns instead of random walks | **FIXED** | `internal/api/router.go:925-934` (+6 lines) |
| 1.3 | PropFirmEnforcer interface compliance — 4 methods + `var _ risk.PropFirmGate` | **FIXED** | `internal/backtest/propfirm_enforcer.go`, `engine.go`, `propfirm_enforcer_test.go` |
| 1.4 | CapitalPoolSim `var _ risk.CapitalGate` assertion — replaced anonymous structs | **FIXED** | `internal/backtest/capital_pool_sim.go` (+6, -13 lines) |
| 1.5 | Hardcoded status indicators — `riskStatus` + authStore for Kill Switch/Auth | **FIXED** | `web/src/pages/monitor/OverviewTab.tsx`, `MonitorPage.tsx` |
| 1.6 | LWC violations — `chart.resize()` replaces `applyOptions({width})`, rAF cleanup | **FIXED** | `web/src/pages/ChartingHub.tsx:267,283,281` |

### Wave 2 — Test Coverage

| # | Task | Status | New Tests |
|---|------|--------|-----------|
| 2.1 | Playwright E2E — verified 10 spec.cjs files exist covering 5 critical flows | Already present | 0 |
| 2.2 | Drift detection tests (PSI, classify, retrain) | **ADDED** | 23 tests in `tests/test_drift_detection.py` |
| 2.3 | Temporal validation tests (8 scenarios) | **ADDED** | 9 tests in `tests/test_temporal.py` |
| 2.4 | ML inference fallback path tests | **ADDED** | 6 tests in `tests/test_inference.py` |
| 2.5 | DB repository tests — config, price roundtrip, JSON, type checks | **EXPANDED** | 13 tests (up from 2) in `internal/db/repository_test.go` |

### Wave 3 — Feature Completeness

| # | Issue | Status |
|---|-------|--------|
| 3.1 | Warm-up period — `WarmUpBars` field wired at engine.go:712-718 | Already implemented |
| 3.2 | OptimizationPanel integrated into BacktestHub as `mode='optimize'` | Already integrated. Cleaned up: deleted `OptimizationPanel.tsx`, updated `CommandPalette` |
| 3.3 | Removed unused `fillModel` field from `Engine` and `EngineMulti` structs | **FIXED** — `engine.go:225`, `engine.go:1483`, `fidelity_test.go` |

### Wave 4 — Polish

| # | Issue | Status |
|---|-------|--------|
| 4.1 | Clean up redirect routes — removed `/audit` and `/users` legacy routes | **FIXED** |
| 4.2 | SimulatePage upgrade — already has structured UI (MetricCard, Table, Badge) | Already done |
| 4.3 | Drawdown duration — shaded area exists, added `max_drawdown_duration_days` metric | **FIXED** |
| 4.4 | Portfolio correlation analysis | Deferred (large scope) |
| 4.5 | Volume-dependent slippage — at `slippage.go:62-64` | Already implemented |
| P3-5 | CAGR metric — `computeCAGR` added, registered in metrics | **FIXED** |

### Prior Audit Doc Items That Were Already Implemented

| Audit Claim | Reality |
|-------------|---------|
| P0-3: Monte Carlo uses random walks | `RunMonteCarloFromTrades` bootstraps from actual trade PnL; only one handler path (optimized walk-forward) used random walks — now fixed |
| P1-4: MAE/MFE hardcoded to 0 | `engine.go:641-889` computes MAE/MFE from actual price data |
| P1-5: Navigation needs restructuring | 3 groups, 13 items already implemented in Sidebar |
| P1-6: No warm-up period | `WarmUpBars` config field wired at `engine.go:712-718` |
| Playwright E2E tests are empty | 10 `.spec.cjs` files exist in `web/e2e/` (audit searched for `.spec.ts`) |
| SimulatePage uses raw JSON | All 7 tabs use MetricCard/Table/Badge components |

### Current State

- **Compliance**: 18/18 AGENTS.md rules pass (Rules 13/15 violations fixed in Wave 1)
- **Test suite**: 528 Python tests (7 skipped), 24/24 Go packages passing
- **Total new tests added**: 51 (38 Python + 13 Go)
- **Files modified**: 14 Go files, 6 frontend files, 5 test files created
- **Files deleted**: 1 (OptimizationPanel.tsx)
- **Dead code removed**: FillModel field from Engine/EngineMulti

