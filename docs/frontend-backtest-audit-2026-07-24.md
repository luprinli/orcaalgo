# OrcaAlgo Frontend & Backtesting/Live Workflow Audit Report

**Date:** 2026-07-24
**Auditor:** System Architecture Review
**Scope:** Full frontend audit + full backtesting-to-live-trading workflow + industry best-practices gap analysis

---

## PART I — FRONTEND AUDIT

### 1. Catalog of All Frontend Assets

#### 1.1 Route-Page Map (23 unique routes)

| Route | Page Component | Auth Required | Lazy Loaded |
|---|---|---|---|
| `/` | `MonitorPage` | Yes | No |
| `/execution` | `ExecutionPage` | Yes | No |
| `/backtest` | `BacktestHub` | Yes | No |
| `/backtest/history` | `BacktestHub` (history view) | Yes | No |
| `/backtest/history/:id` | `BacktestHub` (detail view) | Yes | No |
| `/strategies` | `StrategyHub` | Yes | No |
| `/optimize` | `OptimizationPanel` | Yes | No |
| `/accounts` | `AccountsPage` | Yes | No |
| `/propfirm` | `PropFirmPage` | Yes | No |
| `/market-data` | `ChartingHub` | Yes | No |
| `/charting` | `ChartingHub` (alias) | Yes | No |
| `/indicators` | Redirect → `/market-data?tab=indicators` | Yes | N/A |
| `/simulate` | `SimulatePage` | Yes | Yes |
| `/calibrate` | `CalibratePage` | Yes | Yes |
| `/attribution` | `AttributionPage` | Yes | Yes |
| `/settings` | `SettingsPage` | Yes | No |
| `/credentials` | `CredentialManagement` | Yes | No |
| `/brokers` | `IntegrationsPage` | Yes | No |
| `/symbols` | `IntegrationsPage` | Yes | No |
| `/data-sources` | `DataSources` | Yes | No |
| `/admin` | `AdminPage` | Yes | Yes |
| `/admin/universe` | `UniversePage` | Yes | Yes |
| `/emergency` | `EmergencyPage` | **No** (unauthenticated bypass) | No |
| `/status` | `StatusPage` | Yes | No |
| `/2fa` | `TwoFAPage` | Yes | No |
| `/login`, `/register`, `/forgot-password`, `/reset-password` | Auth pages | No | No |

**16 additional legacy redirect routes** (e.g., `/live`, `/risk`, `/webhooks`, `/llm`, `/notifications`, `/audit`, `/users`, `/admin/health`, `/admin/propfirm`, etc.) — all redirect internally.

#### 1.2 Sidebar Navigation (4 groups, 20 items)

```
Trading:
  Monitor (/)               → MonitorPage with 4 tabs
  Execution (/execution)    → ExecutionPage (order form + active orders)
  Backtest (/backtest)      → BacktestHub (runner + history + detail)
  Strategies (/strategies)  → StrategyHub (catalog + instances + editor)
  Optimize (/optimize)      → OptimizationPanel (standalone param optimization)
  Accounts (/accounts)      → AccountsPage (CRUD for trading accounts)
  Prop Firms (/propfirm)    → PropFirmPage (profiles + status)

Data & Analysis:
  Market Data (/market-data) → ChartingHub (candles + indicators)
  Indicators (/indicators)   → Redirect to /market-data?tab=indicators
  Simulate (/simulate)       → SimulatePage (7 sub-tabs)
  Calibration (/calibrate)   → CalibratePage (Brier score audit)
  Attribution (/attribution) → AttributionPage (PnL slicing)

Configuration:
  Settings (/settings)        → SettingsPage (4 tabs)
  Credentials (/credentials)  → CredentialManagement
  Brokers (/brokers)          → IntegrationsPage (brokers tab)
  Symbols (/symbols)          → IntegrationsPage (providers+symbols tab)
  Data Sources (/data-sources)→ DataSources (3-button page)

Admin:
  Admin Panel (/admin)        → AdminPage (9 tabs)
  Universe (/admin/universe)  → UniversePage
  2FA Setup (/2fa)            → TwoFAPage
  Emergency (/emergency)      → EmergencyPage (no auth)
```

#### 1.3 Component Inventory

**Layout components** (7): `Sidebar`, `PageHeader`, `PageSection`, `PageSkeleton`, `SkeletonRow`, `ErrorBanner`, `DynamicBreadcrumb`

**UI primitives (shadcn)** (31): accordion, alert-dialog, avatar, badge, breadcrumb, button, card, checkbox, collapsible, command, dialog, dropdown-menu, hover-card, input, label, popover, progress, radio-group, scroll-area, select, separator, sheet, skeleton, slider, switch, table, tabs, textarea, toggle-group, tooltip

**Domain components** (40+): backtest/* (17 components), deploy/* (1), strategy-hub/* (3), monitor/* (4), layout/* (7), charts/* (9), plus standalone: `Watchlist`, `TradingViewProvider`, `TimeframeChips`, `SymbolSearch`, `StatusCards`, `SkeletonLoader`, `ParamEditor`, `OHLCVHeader`, `MultiSelect`, `MetricCard`, `IndicatorConfigModal`, `FormField`, `ErrorCard`, `ErrorBoundary`, `EmptyState`, `ConfirmDialog`, `CommandPalette`, `ChartOverlayButtons`, etc.

**Hooks** (14): `useWebSocket`, `useChartUpdate`, `useChartKeyboard`, `useIndicator`, `useIndicatorRenderer`, `useDrawingTool`, `useCrosshair`, `useCandleAggregation`, `useLiveRiskData`, `useLiveData`, `useEmergencyControl`, `useTradeTooltip`, `useParameterSensitivity`, `useMatrixStream`, `useAdaptivePolling`, `useWindowedRows`

**Stores** (7): `authStore`, `wsStore`, `tradeStore`, `indicatorStore`, `matrixStore`, `timeframeStore`, `cacheStore`

**Charts** (9): `EquityCurveChart`, `DailyReturnsChart`, `MonteCarloChart`, `CalendarHeatmap`, `YearlySummaryTable`, `CandlesChart`, `LiveMonitorChart`, `CVDChart`, `CrosshairTooltip`

**API modules** (2): `api/client.ts` (413 lines, ~50 endpoints), `api/optimize.ts` (64 lines, 4 functions)

---

### 2. Mapping: Frontend → Backend Capabilities

| Frontend Page/Tab | Backend Endpoint(s) Used | Go Package(s) | Status |
|---|---|---|---|
| **MonitorPage** (Overview) | `live.metrics()`, `live.equity()`, `positions.list()`, `orders.list()`, `risk.status()`, `WS /ws` | `engine/live_engine.go`, `broker/`, `risk/` | Functional |
| **MonitorPage** (Risk) | `risk.emergencyStop()`, `risk.emergencyResume()`, `admin.killHistory()`, `monitor.regimeHistory()` | `risk/kill_switch.go` | Functional |
| **MonitorPage** (Signals) | `signals.list()` | `strategy/` | Functional |
| **ExecutionPage** | `orders.place()`, `orders.list()`, `orders.cancel()` | `broker/adapter.go` | Functional |
| **BacktestHub** (Runner) | `backtests.run()`, `WS matrix stream` | `backtest/batch_runner.go`, `backtest/engine.go` | Functional |
| **BacktestHub** (History) | `backtests.list()`, `backtests.metrics()`, `backtests.equity()`, `backtests.delete()`, `backtests.rerun()` | `backtest/`, `db/` | Functional |
| **BacktestHub** (Detail) | `backtests.metrics()`, `.equity()`, `.dailyReturns()`, `.trades()`, `.regimeStats()`, `.monthlyReturns()`, `.optimization()`, `.walkForward()`, `.liveComparison()` | `backtest/engine.go`, `backtest/optimizer.go`, `backtest/walk_forward.go`, `backtest/monte_carlo.go` | Functional |
| **BacktestHub** (Promote) | `strategies.preflight()`, `strategies.deploy()` | `backtest/job_runner.go` | Functional |
| **StrategyHub** | `strategies.list()`, `.create()`, `.update()`, `.delete()`, `.clone()`, `.validate()`, `.reload()`, `.fromGkr()` | `strategy/registry.go` | Functional |
| **OptimizationPanel** | `optimize/submitOptimizationRun()`, `.getOptimizationStatus()`, `.getOptimizationResults()` | `backtest/optimizer.go`, `backtest/light_optimizer.go` | Functional (separate `/api/v1/optimize/*`) |
| **ChartingHub** (Candles) | `candles.get()`, `WS /ws ticks` | `ingest/`, `db/` | Functional |
| **ChartingHub** (Indicators) | `indicators.list()`, `.compute()`, `.startStream()`, `.stopStream()` | `strategy/indicators.go` | Functional |
| **AccountsPage** | `accounts.list()`, `.create()`, `.delete()`, `.setDefault()`, `brokers.list()` | `broker/account_manager.go` | Functional |
| **PropFirmPage** | `propfirm.profiles.list()`, `.create()`, `.update()`, `.delete()`, `.active.get()`, `.active.set()`, `.status()` | `propfirm/`, `backtest/propfirm_enforcer.go` | Functional |
| **SimulatePage** | `simulate.generate()`, `.calibrate()`, `.validate()`, `.calibrateRegime()`, `.generateTicks()`, `.injectSignal()`, `.validateRegime()` | `engine/` (Python bridge) | Partially functional |
| **CalibratePage** | `calibrate.run()` | Python `orca/calibration/audit.py` (subprocess) | Functional |
| **AttributionPage** | `attribution.run()` | Python `orca/attribution/slicer.py` (subprocess) | Functional |
| **SettingsPage** | `settings.get()`, `.update()`, `.testNotification()`, `.testLLM()` | `api/`, `notify/` | Functional |
| **CredentialManagement** | `credentials.list()`, `.create()`, `.rotate()` | `api/` (credential store) | Functional |
| **IntegrationsPage** | `brokers.list()`, `providers.list()/.create()/.delete()/.test()`, `symbols.list()/.create()/.delete()`, `credentials.list()/.create()/.rotate()` | `broker/`, `ingest/`, `api/` | Functional |
| **DataSources** | `settings.get()`, `settings.update()` | — (redundant with Settings) | Thin/redundant |
| **AdminPage** | `admin.health()`, `.systemHealth()`, `.users()`, `.auditLogs()`, `.errorLogs()`, `.seed()`, `models.*`, `reconciliation.*`, `dataValidate.*`, kill history | `api/`, `db/`, `model/` | Functional |
| **UniversePage** | `universe.current()`, `.configs()`, `.override()`, `.refresh()`, `.createConfig()`, `.activateConfig()` | `universe/` | Functional |
| **EmergencyPage** | `risk.emergencyStop()`, `risk.emergencyResume()` | `risk/kill_switch.go` | Functional |
| **StatusPage** | `system.health()` | `api/` | Functional (thin) |

---

### 3. Duplication, Overlap & Redundancy

#### 3.1 CRITICAL: CredentialManagement Page Duplicated in IntegrationsPage

**Files:** `web/src/pages/CredentialManagement.tsx` (161 lines) vs `web/src/pages/IntegrationsPage.tsx` (518 lines, includes credentials tab)

Both implement identical credential CRUD: list, create (name + provider + api_key + secret_key), rotate. Same providers, same UI pattern, same fields. The credential form appears twice: once as a standalone page at `/credentials` and once as a tab inside `/brokers` (IntegrationsPage).

**Recommendation:** Delete `CredentialManagement.tsx`. Merge its route into IntegrationsPage. Keep `/credentials` as a redirect to `/integrations?tab=credentials`.

**Effort:** 0.5h | **UX Impact:** High (eliminates navigation confusion)

#### 3.2 HIGH: DataSources Page Redundant with Settings

**File:** `web/src/pages/DataSources.tsx` (42 lines)

The entire DataSources page is three buttons (Alpaca/Stooq/Mock) that update `settings.general.data_source`. This is the *exact same* functionality as the "Data Source" dropdown in SettingsPage → General tab.

**Recommendation:** Delete `DataSources.tsx`. Remove `/data-sources` from sidebar. The data source selector in Settings and Backtest Runner already cover this.

**Effort:** 0.25h | **UX Impact:** Medium (removes one redundant nav item)

#### 3.3 HIGH: Indicators and Market Data as Separate Nav Items

`/indicators` redirects to `/market-data?tab=indicators` (App.tsx:89), yet both appear as separate sidebar items under "Data & Analysis" group. Users see two items that go to the same page.

**Recommendation:** Remove the "Indicators" nav item from the sidebar. Indicators is already a tab inside ChartingHub. No redirect needed.

**Effort:** 0.1h | **UX Impact:** Medium (removes a misleading duplicate nav item)

#### 3.4 MEDIUM: Two Optimization UIs

- `OptimizationPanel` (`/optimize`) — standalone page with its own API (`/api/v1/optimize/*`)
- `OptimizationTab` inside `BacktestHub` detail view — shows optimization footprint from `backtests.optimization(id)`

The standalone `OptimizationPanel` submits optimization runs via a *separate* API (`api/optimize.ts`) from the backtest API. It uses different types (`OptimizeConfig`, `OptimizationResult`) than the backtest detail view's `OptimizationFootprint`. Yet both conceptually achieve the same thing: parameter optimization.

**Recommendation:** Integrate `OptimizationPanel` into `BacktestHub` as an additional view or a modal accessible from the backtest runner. Unify the APIs so `POST /api/v1/backtests` supports `optimize: true` mode instead of a separate `/api/v1/optimize/*` namespace. The OptimizationPanel should accept a `strategy_id` from URL params to pre-fill.

**Effort:** 4h | **UX Impact:** High (unifies optimization into the backtest flow, reduces navigation)

#### 3.5 MEDIUM: Route Aliases Create Confusion

- `/market-data` AND `/charting` both render `ChartingHub` (App.tsx:88,90)
- `/brokers` AND `/symbols` both render `IntegrationsPage` (App.tsx:92-93)

**Recommendation:** Consolidate to canonical routes. Remove aliases or redirect them cleanly. Recommended canonical routes: `/charting` for ChartingHub, `/integrations` for IntegrationsPage.

**Effort:** 0.5h | **UX Impact:** Low

#### 3.6 LOW: Risk Emergency Controls in Two Places

- `EmergencyPage` (`/emergency`): Standalone mobile-optimized emergency stop/resume with 2FA
- `RiskTab` inside `MonitorPage`: Same emergency stop/resume with 2FA

This is justified duplication because EmergencyPage is accessible without auth (for emergencies).

**Recommendation:** Keep both. Ensure `EmergencyPage` loads from same risk hook. No action needed.

#### 3.7 LOW: Settings Fragmentation

Five separate nav items deal with configuration: Settings, Credentials, Brokers, Symbols, Data Sources. Users must navigate to 5 different pages to configure the system, spread across two sidebar groups.

**Recommendation:** Consolidate into a unified "System Configuration" page with sub-tabs or sections. This is part of the navigation restructuring recommended below.

**Effort:** 6h | **UX Impact:** High

---

### 4. Skeleton-Only Pages or Pages With No Functional Data

#### 4.1 CRITICAL: MonitorPage System Status Panel — Hardcoded Fake Data

**File:** `web/src/pages/monitor/OverviewTab.tsx:91-106`

All six status indicators (`brokerOnline`, `dataFeedActive`, `killSwitchActive`, `dbConnected`, `authEnforced`, `wsConnected`) are hardcoded `ok: true`. None of these values come from any API. The System Health endpoint (`/api/v1/system/health`) exists and returns real data but is never called here.

#### 4.2 LOW: SimulatePage Sub-Tabs (4 of 7) Use Raw JSON Output

The calibrate-regime, ticks, inject-signal, and validate-regime tabs render results as un-styled `<pre>{JSON.stringify(result)}</pre>`. These function but have no proper UI components, tables, or data visualization. They look like developer debug panels.

#### 4.3 LOW: StatusPage is Extremely Thin

`StatusPage` (26 lines) calls `system.health()` and dumps all fields as `MetricCard` components. It works but provides no actionable information beyond what could be a small status badge.

---

### 5. Recommended Navigation Restructuring

**Current:** 4 groups, 20 items, multiple overlapping concepts, config scattered across 2 groups.

**Proposed:** 3 groups, 13 items, all core functionality accessible in ≤ 2 clicks.

```
Trading Desk:
  Dashboard (/)              → MonitorPage (overview + positions + risk + signals)
  Execution (/execution)     → ExecutionPage (orders + positions)
  Backtesting (/backtest)    → BacktestHub (runner + history + detail + optimize + promote)
  Strategies (/strategies)   → StrategyHub (catalog + instances + editor)

Analysis:
  Charts (/charting)         → ChartingHub (candles + indicators + watchlist)
  Calibration (/calibrate)   → CalibratePage
  Attribution (/attribution) → AttributionPage
  Simulation (/simulate)     → SimulatePage

Settings:
  System (/settings)         → SettingsPage (trading + webhooks + notifications + LLM + data source)
  Integrations (/integrations) → IntegrationsPage (brokers + providers + symbols + credentials)
  Accounts (/accounts)       → AccountsPage (trading accounts + prop firm profiles)
  Admin (/admin)             → AdminPage (health + users + audit + errors + db + universe + reconciliation + status)
```

**Removed from nav:** Indicators, Data Sources, Credentials, Brokers, Symbols, Optimize, Prop Firms (merged into Accounts), Status (merged into Admin).

**Rationale:** The Optimize standalone page merges into Backtesting. Prop Firms move under Accounts (they are account profiles). Credentials, Brokers, Symbols, and Data Sources all merge into Integrations. Status page absorbed into Admin → Health tab.

---

## PART II — BACKTESTING & LIVE TRADING WORKFLOW AUDIT

### 6. Current End-to-End Backtest Flow

```
1. CONFIGURE → BacktestHub RunnerView
   - Select strategies (multi-select checkboxes)
   - Enter symbols (comma-separated text)
   - Set date range, capital, data source, gate profile
   - Choose single vs matrix mode (matrix adds timeframe dimension)
   - Click "Run Backtest" or "Run Matrix"

2. EXECUTE → Go Backend (backtest/batch_runner.go)
   - Cartesian product of (strategies × symbols × timeframes)
   - Bounded concurrency (semaphore.Weighted, max 8 workers)
   - Heap admission control (memory pressure throttling)
   - For each combo: Engine.Run() with full pipeline
     ├─ Prefetch regime/VIX/sentiment logs
     ├─ Merge multi-symbol candles by time
     ├─ Bar-by-bar loop:
     │  ├─ Universe filtering (point-in-time snapshots)
     │  ├─ Regime filter (HMM-based)
     │  ├─ Stop-loss/take-profit check
     │  ├─ Adverse selection detection (next-bar close vs entry)
     │  ├─ ML dynamic exit adjustments
     │  ├─ Signal generation (strategy runner + multi-layer gating):
     │  │  ├─ Capital > 0 check
     │  │  ├─ Volatility halt (z-score threshold)
     │  │  ├─ Regime sizing multiplier
     │  │  ├─ Seasonal overlay
     │  │  ├─ Kelly multiplier (k=0.25 default)
     │  │  ├─ Meta-label ML gate (PWin filtering)
     │  │  └─ Position sizing (confidence × regime × VIX × sentiment)
     │  └─ Fill simulation (slippage + partial fill + latency)
     ├─ Close remaining positions at end
     └─ Compute metrics + gate evaluation

3. STREAM → WebSocket to MatrixResultsPanel
   - Real-time progress bar
   - Filterable/sortable results table (Sharpe, Sortino, DD, Win Rate, etc.)

4. ANALYZE → BacktestHub DetailView
   - Full metrics card grid (15 metrics)
   - Equity curve chart
   - Daily returns distribution chart
   - Monte Carlo simulation (500 sims, 252 forward days)
   - Calendar heatmap (monthly returns)
   - Yearly summary table
   - Regime breakdown table
   - Trade list (filterable by month, paginated)
   - Optimization footprint
   - Live vs backtest comparison

5. DEPLOY → PromoteToLiveWizard (3-step)
   - Step 1: Quality gates (Sharpe ≥ 1.0, MaxDD ≤ 8%, PassProb ≥ 80%, PF ≥ 1.5)
   - Step 2: Pre-flight checklist (DB connectivity, engine ready, data quality, synthetic contamination)
   - Step 3: Select account + capital allocation %, deploy
```

### 7. Live Trading Workflow

```
1. TICK INGEST → WebSocket → RingBuffer (16384 capacity, lock-free SPSC)
2. BAR AGGREGATION → 1m/5m/15m/1h rolling windows
3. REGIME DETECTION → HMMTracker.Update() on each bar
4. SIGNAL GENERATION → GlobalRegistry().EvaluateAll()
5. SIGNAL FILTERING → Adversarial check → ML meta-label → PWin sizing
6. ORDER ROUTING → Capability-based broker adapter (Alpaca/IBKR/Paper)
7. RISK MONITORING → Kill-switch (re-entrancy guarded), order rate limiter,
                     volatility halt, exposure tracker, daily loss limit
8. DATA → PostgreSQL + TimescaleDB hypertables for ticks/candles/executions
```

### 8. Backtest vs Live Parity Analysis

| Feature | Backtest | Live | Parity |
|---|---|---|---|
| Strategy signal generation | `StrategyRunner.Evaluate()` | `GlobalRegistry().EvaluateAll()` | ✅ Same strategy runner |
| Fill simulation | `FillSimulator` (slippage + latency + partial fill) | Real broker fills | ⚠️ Different fill model |
| Commission | `BrokerageFeeConfig` per side (bps + SEC) | `BrokerageFeeConfig` from adapter | ✅ Shared config |
| Regime detection | Pre-computed HMM logs | Real-time `HMMTracker` | ✅ Same model |
| Position sizing | `ComputeSizeUncapped()` | `ComputeSize()` | ⚠️ Uncapped vs capped |
| Stop-loss | `CheckStopHit()` per bar | `LiveEngine` stop monitoring | ✅ Same logic |
| Kelly fraction | Configurable k (default 0.25) | Not applied live | ❌ Gap |
| ML meta-label gate | `PWin > threshold` | Same gate | ✅ Shared |
| Prop firm rules | `PropfirmEnforcer` | Not enforced live (manual) | ❌ Gap |
| Multi-strategy | `EngineMulti` + `CapitalPoolSim` | `LiveEngine` single-strat only | ❌ Gap |

### 9. Industry Best-Practices Gap Analysis (QuantStart Framework)

| Best Practice | OrcaAlgo Status | Gap Severity |
|---|---|---|
| **Optimization bias mitigation** | ✅ IVS stability scoring, walk-forward purging, embargo periods, parameter sensitivity heatmap, bounded search spaces | Good |
| **Look-ahead bias prevention** | ⚠️ Regime detection may leak future data (intraday bars use same-day regime computed from full-day data), high/low values in stop-loss checking | Medium |
| **Survivorship bias mitigation** | ✅ Point-in-time universe snapshots (`LoadUniverseSnapshots`), active symbol filtering per bar | Good |
| **Transaction cost modeling** | ✅ Commission bps + SEC fee + spread slippage + partial fill + latency | Good |
| **Market impact modeling** | ❌ No volume-dependent impact or order book depth simulation | Medium |
| **Warm-up period** | ❌ No explicit warm-up exclusion period; indicators may produce garbage on first N bars | Medium |
| **Data-snooping/overfitting defense** | ✅ Walk-forward with purged CV, IVS robustness, multi-metric gating | Good |
| **Monte Carlo from trades** | ❌ `MonteCarloFromTrades()` ignores trade data; uses random walks instead of bootstrapped trade PnL | High |
| **Sharpe ratio consistency** | ❌ Engine uses `sqrt(252 * barsPerDay)`; metrics uses `sqrt(252)` — inconsistent annualization | Medium |
| **MAE/MFE calculation** | ❌ Hardcoded to 0 in handler (`backtest_metrics_handler.go:195`); not computed by engine | High |
| **Short-selling constraints** | ❌ No borrow cost, no short-squeeze modeling | Low |
| **Correlation/diversification analysis** | ❌ Multi-strategy engine runs independently; no portfolio-level risk decomposition | Low |
| **Psychological tolerance bias** | ❌ No drawdown duration visualization, no behavioral risk warnings, no forward stress testing | Medium |
| **Calibration audit (probability models)** | ✅ Quarterly `orca calibrate` with Brier + reliability + resolution + Wilson CI | Good |
| **Fill model redundancy** | ❌ Two parallel fill implementations: `FillSimulator` (used) vs `FillModel` interface (unused) | Low |
| **Fill probability on exit** | ❌ 5% partial fill probability only on entry; exit fills always get full quantity | Medium |
| **Data pipeline integrity** | ✅ Preflight rejects synthetic data contamination, data validation checks exist | Good |
| **Regime robustness** | ⚠️ VIX > 25 warning re: pre-computed regime accuracy, but no real-time regime model switching | Low |
| **News trading** | ⚠️ Hardcoded time windows (8:30-8:35 ET, etc.) without actual economic calendar integration | Low |

---

## PART III — PRIORITIZED REMEDIATION PLAN

### Priority Legend
- **P0** — Critical: security, data integrity, or significantly misleading UX
- **P1** — High: substantial UX friction, duplicate code, or backtesting accuracy
- **P2** — Medium: polish, consolidation, or minor gaps
- **P3** — Low: nice-to-have enhancements

---

### P0 Items (Must Fix)

#### P0-1: Fix Hardcoded System Status in Monitor Overview
**File:** `web/src/pages/monitor/OverviewTab.tsx:91-106`
**Problem:** Six status indicators show hardcoded `ok: true`. Users see "green dots" that never reflect actual system health.
**Fix:** Call `system.health()` (`GET /api/v1/system/health`) and map real component statuses to the indicators.
**Effort:** 1h | **UX Impact:** Critical — eliminates false positive status reporting

#### P0-2: Delete Duplicate CredentialManagement Page
**Files:** `web/src/pages/CredentialManagement.tsx`, `web/src/components/layout/Sidebar.tsx`, `web/src/App.tsx`
**Problem:** Identical credential CRUD duplicated in standalone page and IntegrationsPage tab.
**Fix:** Delete `CredentialManagement.tsx`, remove `/credentials` from sidebar, redirect `/credentials` to `/integrations?tab=credentials` in App.tsx.
**Effort:** 0.5h | **UX Impact:** High — removes one redundant nav item, eliminates confusion

#### P0-3: Fix Monte Carlo from Trade Data
**File:** `internal/backtest/monte_carlo.go:230-242`
**Problem:** `MonteCarloFromTrades()` receives trade data but ignores it, generating random walks instead. The frontend Monte Carlo display is misleading — it shows simulated random walk outcomes, not bootstrapped trade performance.
**Fix:** Implement bootstrap resampling of actual trade PnL values. For each simulation, sample with replacement from the trade PnL list to build an equity curve.
**Effort:** 2h | **UX Impact:** Critical — fixes fundamentally misleading Monte Carlo results

---

### P1 Items (High Priority)

#### P1-1: Delete Redundant DataSources Page
**File:** `web/src/pages/DataSources.tsx`
**Fix:** Remove page, sidebar item, and route. Data source selection already exists in Settings → General and Backtest Runner.
**Effort:** 0.25h | **UX Impact:** Medium

#### P1-2: Remove "Indicators" from Sidebar Navigation
**File:** `web/src/components/layout/Sidebar.tsx:61`
**Fix:** Remove the Indicators nav item (it already redirects to Market Data → indicators tab).
**Effort:** 0.1h | **UX Impact:** Medium

#### P1-3: Integrate OptimizationPanel into BacktestHub
**Files:** `web/src/pages/OptimizationPanel.tsx`, `web/src/api/optimize.ts`, `web/src/App.tsx`, `web/src/components/layout/Sidebar.tsx`
**Fix:** Add "Optimize" as a mode in BacktestHub RunnerView (alongside single/matrix), or as a modal triggered from the runner. Unify `POST /api/v1/optimize/run` and `POST /api/v1/backtests` under a single endpoint with `optimize: true`. Remove the standalone `/optimize` route and sidebar item.
**Effort:** 4h | **UX Impact:** High — unifies backtesting + optimization into one workflow

#### P1-4: Implement MAE/MFE Calculation
**File:** `internal/backtest/engine.go` (add to metrics computation), `internal/api/backtest_metrics_handler.go:195`
**Problem:** Maximum Adverse Excursion and Maximum Favorable Excursion are hardcoded to 0.
**Fix:** Track `worstPrice` and `bestPrice` for each open position during the bar loop, compute MAE = entry_price - worst_price (for longs), MFE = best_price - entry_price (for longs). Report per-trade and aggregate.
**Effort:** 2h | **UX Impact:** High — enables stop-loss optimization and trade quality analysis

#### P1-5: Restructure Navigation (3 Groups, 13 Items)
**Files:** `web/src/components/layout/Sidebar.tsx`, `web/src/App.tsx`
**Fix:** Implement the proposed navigation structure from §5. Move Prop Firms into Accounts page as a sub-tab. Merge standalone routes into their logical parents.
**Effort:** 6h | **UX Impact:** Very High — core workflow accessible in 2-3 clicks from any page

#### P1-6: Add Warm-Up Period to Backtest Engine
**File:** `internal/backtest/engine.go` (around line 496, the bar loop entry)
**Problem:** Strategies start evaluating from the first candle. Indicators with long lookback periods (e.g., 200-period MA) produce garbage signals.
**Fix:** Add `WarmUpBars` to `BacktestConfig` (default: max indicator warmup). Skip signal generation during warmup, only update indicator state. Mark warmup period on equity curve.
**Effort:** 2h | **UX Impact:** High — prevents garbage signals from contaminating early results

#### P1-7: Fix Sharpe Ratio Annualization Inconsistency
**Files:** `internal/backtest/engine.go:1136-1169`, `internal/backtest/metrics.go`
**Problem:** Engine computes annualized Sharpe as `mean * sqrt(252 * barsPerDay) / std`. The registered metrics framework hardcodes `sqrt(252)`, producing different numbers for intraday backtests.
**Fix:** Standardize on daily returns → `sqrt(252)` annualization. Convert intraday equity curve to daily returns before computing Sharpe in the engine.
**Effort:** 1.5h | **UX Impact:** Medium — fixes metric inconsistency

---

### P2 Items (Medium Priority)

#### P2-1: Replace SimulatePage Raw JSON Output with Proper UI
**Files:** `web/src/pages/SimulatePage.tsx` (tabs: calibrate-regime, ticks, inject-signal, validate-regime)
**Fix:** Replace `<pre>{JSON.stringify(result)}</pre>` with structured tables and metric cards matching the app's design system.
**Effort:** 3h | **UX Impact:** Medium

#### P2-2: Merge StatusPage into AdminPage Health Tab
**File:** `web/src/pages/StatusPage.tsx`
**Fix:** Move SystemHealth display into AdminPage → Health tab. Remove `/status` route and sidebar link.
**Effort:** 1h | **UX Impact:** Low

#### P2-3: Add Drawdown Duration Visualization
**Files:** `web/src/charts/EquityCurveChart.tsx`, `web/src/components/backtest/OverviewTab.tsx`
**Problem:** No visualization of drawdown duration (how long underwater periods last). This addresses the "psychological tolerance bias" from QuantStart.
**Fix:** Add underwater plot (periods where equity < HWM shaded red) to equity curve. Add a "Max Drawdown Duration" metric card.
**Effort:** 2h | **UX Impact:** Medium — critical for behavioral risk assessment

#### P2-4: Add Volume-Dependent Slippage to Fill Model
**File:** `internal/backtest/slippage.go`
**Problem:** Slippage uses fixed spread + random factor, independent of trade volume relative to market volume.
**Fix:** Add `volume_factor = 1 + (trade_qty / avg_bar_volume) * impact_coefficient` to the slippage calculation.
**Effort:** 2h | **UX Impact:** Medium

#### P2-5: Add Portfolio-Level Correlation Analysis for Multi-Strategy
**Files:** `internal/backtest/engine.go` (EngineMulti), `web/src/components/backtest/ComparisonTab.tsx`
**Problem:** Multi-strategy backtest runs strategies independently; no correlation matrix or diversification benefit analysis.
**Fix:** Compute pairwise correlation matrix of strategy returns, report in OverviewTab and ComparisonTab.
**Effort:** 3h | **UX Impact:** Medium

#### P2-6: Route Alias Cleanup
**Files:** `web/src/App.tsx`
**Fix:** Remove `/charting` alias (use `/market-data`), remove `/brokers` and `/symbols` aliases (use `/integrations`). Redirect all old routes with `replace`.
**Effort:** 0.5h | **UX Impact:** Low

#### P2-7: Add Kelly Fraction to Live Trading
**File:** `internal/engine/live_engine.go`
**Problem:** Kelly multiplier is applied in backtests but not in live trading. The sizing method differs (capped vs uncapped), creating a parity gap.
**Fix:** Apply `DefaultKellyMultiplier` (0.25) to live position sizing, matching the backtest's `kelly_fraction` config.
**Effort:** 1h | **UX Impact:** Medium — ensures live trading matches backtest sizing assumptions

---

### P3 Items (Low Priority)

#### P3-1: Integrate Economic Calendar for News Trading Windows
**File:** `internal/backtest/propfirm_enforcer.go:170-186`
**Fix:** Replace hardcoded time windows with a configurable economic calendar lookup.
**Effort:** 4h | **UX Impact:** Low

#### P3-2: Remove Redundant FillModel Interface
**File:** `internal/model/fill.go`, `internal/backtest/engine.go`
**Fix:** Either integrate `FillModel` into the engine (replacing `FillSimulator`) or remove the unused interface.
**Effort:** 2h | **UX Impact:** Low (code cleanup)

#### P3-3: Add Exit Partial-Fill Probability
**File:** `internal/backtest/slippage.go` (around line 73), `internal/backtest/engine.go` (around line 634)
**Fix:** Apply the same 5% partial fill probability to exits that already exists for entries.
**Effort:** 1h | **UX Impact:** Low

#### P3-4: Add Short-Selling Constraints (Borrow Cost)
**File:** `internal/backtest/engine.go`
**Fix:** Add `ShortBorrowFeeBps` to `BacktestConfig`. Apply to short positions in addition to standard commission.
**Effort:** 1.5h | **UX Impact:** Low

#### P3-5: Add CAGR to Metrics Output
**File:** `internal/backtest/metrics.go`, `web/src/types/api.ts`
**Fix:** Compute `CAGR = (final_equity / initial_capital)^(1/years) - 1`. The frontend already has the metric card but the backend may not compute it consistently.
**Effort:** 0.5h | **UX Impact:** Low

---

## Summary: Effort vs Impact Matrix

| ID | Item | Effort | UX Impact | Risk |
|---|---|---|---|---|
| P0-1 | Fix hardcoded system status | 1h | Critical | None |
| P0-2 | Delete duplicate CredentialManagement | 0.5h | High | None |
| P0-3 | Fix Monte Carlo from trades | 2h | Critical | Medium (core engine change) |
| P1-1 | Delete redundant DataSources | 0.25h | Medium | None |
| P1-2 | Remove Indicators nav item | 0.1h | Medium | None |
| P1-3 | Integrate OptimizationPanel into BacktestHub | 4h | High | Medium (refactor) |
| P1-4 | Implement MAE/MFE | 2h | High | Low |
| P1-5 | Restructure navigation | 6h | Very High | Medium (broad refactor) |
| P1-6 | Add warm-up period | 2h | High | Medium (engine change) |
| P1-7 | Fix Sharpe annualization | 1.5h | Medium | Low |
| P2-1 | Replace SimulatePage JSON | 3h | Medium | Low |
| P2-2 | Merge StatusPage into Admin | 1h | Low | None |
| P2-3 | Add drawdown duration viz | 2h | Medium | Low |
| P2-4 | Volume-dependent slippage | 2h | Medium | Low |
| P2-5 | Portfolio correlation analysis | 3h | Medium | Low |
| P2-6 | Route alias cleanup | 0.5h | Low | None |
| P2-7 | Kelly fraction in live trading | 1h | Medium | Low |
| P3-1 | Economic calendar integration | 4h | Low | Medium |
| P3-2 | Remove redundant FillModel | 2h | Low | Low |
| P3-3 | Exit partial-fill probability | 1h | Low | Low |
| P3-4 | Short-selling constraints | 1.5h | Low | Low |
| P3-5 | Add CAGR metric | 0.5h | Low | None |

**Total estimated effort:** ~40h
**P0-P1 critical path:** ~18h (can be completed in 2-3 days by one developer)

---

## Recommended Implementation Order

1. **Day 1 (8h):** P0-1, P0-2, P1-1, P1-2, P1-5 (navigation restructure is the anchor task)
2. **Day 2 (8h):** P1-3 (optimization integration), P1-6 (warm-up period), P0-3 (Monte Carlo fix)
3. **Day 3 (8h):** P1-4 (MAE/MFE), P1-7 (Sharpe fix), P2-1 (SimulatePage UI), P2-2, P2-6
4. **Day 4-5 (16h):** P2-3, P2-4, P2-5, P2-7, P3 items as time permits
