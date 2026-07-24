# OrcaAlgo Frontend Audit Report

**Date:** 2026-07-23
**Auditor:** Senior Frontend Architect & UX Specialist
**Version:** 1.0.0
**Scope:** Full web dashboard — 35 pages, 31 components, 12 hooks, 7 stores, 13 chart modules
**Stack:** React 18.3, TypeScript 5.5, Lightweight Charts 5.2, Zustand 5.0, Vite 5.4

---

## Executive Summary

The OrcaAlgo dashboard is functionally broad but architecturally fragmented. **35 pages** implement trading workflows across backtesting, live monitoring, risk management, optimization, and administration. The chart infrastructure (Lightweight Charts) is well-integrated with custom hooks for crosshair, drawing tools, and keyboard shortcuts. The WebSocket singleton with listener registry is architecturally sound.

However, the application suffers from three systemic problems:

1. **Information architecture fragmentation:** 43 routes with heavy overlap between `Dashboard`, `LiveTrading`, `RiskPage`, and `StatusPage`. Core quantitative trader workflows (backtest analysis → optimization → strategy comparison → live deployment) require navigating through 4-5 separate routes with no breadcrumb or wizard guidance.

2. **Redundant data fetching:** 5 pages independently poll/live-subscribe to the same `risk` WebSocket channel. `LiveTrading` and `Dashboard` both fetch `live.metrics()` and `live.equity('90d')` on independent 10s/15s intervals — duplicate API calls per session.

3. **Inconsistent implementation patterns:** 6 stub pages with zero functionality, 5 pages using raw `fetch()` bypassing the shared auth/loader middleware, `OptimizationPage` is a 3-line wrapper around `OptimizationPanel`, and 10+ files use inline `style={{display:'flex',flexDirection:'column',gap:X}}` instead of CSS classes.

### Severity Summary

| Severity | Count | Description |
|----------|-------|-------------|
| **P0 — Critical** | 3 | Broken auth flow (stub pages), optimize.ts no auth, duplicate WebSocket subscriptions |
| **P1 — High** | 8 | Page overlap/redundancy, missing navigation flows, inconsistent API patterns |
| **P2 — Medium** | 11 | Stub pages, inline styles, polling intervals, missing loading states |
| **P3 — Low** | 7 | Routes, component naming, dead code |

---

## 1. Architecture Overview

### 1.1 Route Table (43 routes)

| Route | Component | Type | Status |
|-------|-----------|------|--------|
| `/` | `Dashboard` | Core | Functional |
| `/live` | `LiveTrading` | Core | Functional |
| `/live/market` | `LiveMarket` | Core | Functional |
| `/execution` | `ExecutionPage` | Core | Functional |
| `/status` | `StatusPage` | Core | Minimal (health only) |
| `/risk` | `RiskPage` | Core | Functional |
| `/backtest` | `BacktestPage` | Core | Functional (matrix + single) |
| `/backtest/history` | `BacktestHistory` | Core | Functional (compare mode) |
| `/backtest/history/:id` | `BacktestDetail` | Core | Functional (full analytics) |
| `/strategies` | `StrategiesPage` | Core | Functional (catalog + instances) |
| `/strategies/:id` | `StrategyEditor` | Core | Functional |
| `/strategies/:id/edit` | `StrategyEditor` | Core | Duplicate route |
| `/strategies/edit/:id` | `StrategyEditor` | Core | Duplicate route |
| `/strategies/new` | → redirect | Core | Redirect |
| `/optimize` | `OptimizationPage` | Core | Thin wrapper |
| `/accounts` | `AccountsPage` | Management | Functional |
| `/propfirm` | `PropFirmPage` | Management | Functional |
| `/market-data` | `MarketDataPage` | Data | Functional |
| `/indicators` | `IndicatorsPage` | Data | Functional |
| `/data-sources` | `DataSources` | Config | Stub (local state only) |
| `/brokers` | `BrokerManagement` | Config | Stub |
| `/symbols` | `SymbolAdminPage` | Admin | Functional |
| `/calibrate` | `CalibratePage` | Analysis | Functional |
| `/attribution` | `AttributionPage` | Analysis | Functional |
| `/simulate` | `SimulatePage` | Analysis | Functional |
| `/admin` | `AdminPage` | Admin | Functional (6 tabs) |
| `/admin/health` | → redirect | Admin | Redirect |
| `/admin/logs` | → redirect | Admin | Redirect |
| `/admin/propfirm` | `PropFirmPage` | Admin | Duplicate route |
| `/admin/symbols` | `SymbolAdminPage` | Admin | Duplicate route |
| `/admin/universe` | `UniversePage` | Admin | Functional |
| `/audit` | → redirect | Admin | Redirect |
| `/users` | → redirect | Admin | Redirect |
| `/credentials` | `CredentialManagement` | Config | Stub |
| `/webhooks` | `WebhookConfig` | Config | Stub |
| `/llm` | `LLMSettings` | Config | Stub |
| `/2fa` | `TwoFAPage` | Auth | Functional |
| `/settings` | `SettingsPage` | Config | Functional |
| `/notifications` | `NotificationSettings` | Config | Stub |
| `/login` | `LoginPage` | Auth | Functional |
| `/register` | `RegisterPage` | Auth | Functional |
| `/forgot-password` | `ForgotPasswordPage` | Auth | Stub |
| `/reset-password` | `ResetPasswordPage` | Auth | Stub |

**Issues:**
- **6 redirect-only routes** — `/admin/health`, `/admin/logs`, `/audit`, `/users`, `/strategies/new`, `/admin/propfirm`/`/admin/symbols`
- **3 duplicate routes** for `StrategyEditor` — `/strategies/:id`, `/strategies/:id/edit`, `/strategies/edit/:id`
- **8 stub pages** with zero or placeholder functionality
- **1 wrapper page** (`OptimizationPage` — 3 lines wrapping `OptimizationPanel`)

### 1.2 Component Hierarchy

```
App.tsx
├── UnauthenticatedApp
│   ├── LoginPage / RegisterPage / ForgotPasswordPage / ResetPasswordPage
├── AuthenticatedApp
│   ├── Sidebar (inline in App.tsx)
│   ├── AppHeader
│   └── 35 page components
│       ├── Shared: MetricCard, FormField, ConfirmDialog, SkeletonLoader, ErrorCard, ErrorBoundary
│       ├── Charts: EquityCurveChart, LiveMonitorChart, CandlesChart, CVDChart, MonteCarloChart,
│       │           DailyReturnsChart, CalendarHeatmap, YearlySummaryTable, CrosshairTooltip
│       ├── Backtest: BacktestConfigBar, MatrixProgressBar, MatrixResultsPanel, OverviewTab,
│       │             TradesTab, OptimizationTab, ComparisonTab, MonteCarloCards, ResourceGauges
│       └── Deploy: PromoteToLiveWizard
```

### 1.3 Data Flow

```
WebSocket (Singleton WSConnectionManager)
  ├── channel: "risk" → RiskPage, Dashboard, LiveTrading (3 subscribers, same channel)
  ├── channel: "ticks" → MarketDataPage, LiveMarket, LiveTrading (3 subscribers)
  ├── channel: "cvd" → MarketDataPage
  ├── channel: "divergence" → MarketDataPage
  └── channel: "pnl_history" → (registered but no subscriber identified)

REST (client.ts)
  ├── backtests.* — 21 endpoints (most heavily used namespace)
  ├── live.*     — 5 endpoints
  ├── strategies.* — 12 endpoints
  ├── orders.*   — 4 endpoints
  ├── risk.*     — 3 endpoints
  ├── accounts.* — 4 endpoints
  ├── propfirm.* — 6 endpoints
  ├── admin.*    — 7 endpoints
  └── 10 other namespaces (1-5 endpoints each)

REST (raw fetch — bypasses auth/loader middleware)
  ├── optimize.ts — 5 endpoints (no auth headers)
  ├── SymbolAdminPage — ALL endpoints (raw fetch with inline auth)
  ├── AdminPage — seed + email endpoints (raw fetch)
  ├── LoginPage — auth/login (raw fetch, expected for login)
  ├── TwoFAPage — 2fa/setup, 2fa/verify (raw fetch)
```

---

## 2. Critical Issues (P0)

### P0-1: `optimize.ts` Bypasses Authentication

**Location:** `web/src/api/optimize.ts`

All 5 functions (`startOptimization`, `getOptimizationStatus`, `getOptimizationResults`, `submitOptimizationRun`, `listOptimizationRuns`) use raw `fetch()` with no `Authorization` header, no request ID header, and no 401 refresh handling. This means:
- Optimization runs can be submitted without authentication if the Go backend doesn't enforce it
- No consistent error handling or request tracking
- No global loader integration

**Severity:** P0
**Impact:** Security gap, inconsistent UX (no loading states for optimization requests)

### P0-2: Duplicate WebSocket Subscriptions

**Location:** `RiskPage`, `Dashboard`, `LiveTrading`

Three pages independently subscribe to the `risk` WebSocket channel:
- `RiskPage` — `useWebSocket('risk', { maxReconnects: 30, reconnectInterval: 2000 })`
- `Dashboard` — `useWebSocket('risk', ...)`
- `LiveTrading` — `useWebSocket('ticks', ...)` + polling `risk.status()`

While the `WSConnectionManager` deduplicates actual socket connections, each subscription triggers its own handler chain. If multiple pages are open in tabs or the app grows to include risk data in more components, the handler overhead multiplies.

**Severity:** P0
**Impact:** Redundant data processing, potential handler race conditions

### P0-3: Stub Auth Pages with No Functionality

**Location:** `ForgotPasswordPage`, `ResetPasswordPage`

Both pages display forms but hardcode success without calling any API:
- `ForgotPasswordPage` — sets `msg` state to "Password reset link sent" on button click
- `ResetPasswordPage` — sets `msg` state to "Password successfully reset" then shows back-to-login button

**Severity:** P0
**Impact:** Broken user experience — users cannot recover accounts

---

## 3. High-Severity Issues (P1)

### P1-1: Dashboard / LiveTrading / RiskPage Overlap

**Location:** `Dashboard.tsx`, `LiveTrading.tsx`, `RiskPage.tsx`

These three pages render overlapping data:

| Data | Dashboard | LiveTrading | RiskPage |
|------|-----------|-------------|----------|
| Balance/Equity | ✅ | ✅ | ✅ |
| Daily PnL | ✅ | ✅ | ✅ |
| Drawdown | ✅ | ❌ | ✅ |
| Sharpe/WinRate/ProfitFactor | ✅ | ✅ | ❌ |
| Risk limits (progress bars) | ✅ | ❌ | ✅ |
| Emergency stop buttons | ❌ | ❌ | ✅ |
| Equity curve chart | ✅ | ✅ | ❌ |
| Positions table | ❌ | ✅ | ❌ |
| Orders table | ❌ | ✅ | ❌ |
| Regime indicator | ✅ | ❌ | ✅ |
| Trade count | ✅ | ❌ | ❌ |

`Dashboard` and `LiveTrading` both independently poll `live.metrics()` and `live.equity('90d')` on separate intervals (15s vs 10s). `RiskPage` additionally polls `risk.status()` on 10s.

**Severity:** P1
**Impact:** Redundant API calls, scattered risk/monitoring data requiring 3 page switches, inconsistent refresh rates

### P1-2: Backtest Config Duplication

**Location:** `BacktestPage.tsx`, `OptimizationPanel.tsx`

Both pages implement backtest configuration UIs with overlapping fields:
- `BacktestPage` mode="optimize" → strategy select, objective select, parameter search space table, run button
- `OptimizationPanel` → strategy select, symbol input, objective select, max combinations, train/test years, step months, capital, parameter ranges table, run button

These are two separate implementations of the same optimization config workflow with different sets of fields.

**Severity:** P1
**Impact:** Duplicate maintenance burden, inconsistent feature sets

### P1-3: Inconsistent API Access Patterns

**Location:** `SymbolAdminPage.tsx`, `AdminPage.tsx`

`SymbolAdminPage` uses raw `fetch()` with inline `Authorization: Bearer ${localStorage.orca_token}` for ALL endpoints — 10+ distinct API calls. This completely bypasses:
- The shared `request()` function's 401 refresh logic
- Request ID tracking
- Global loader middleware
- Typed request/response validation

`AdminPage` similarly uses raw `fetch()` for seed and email test endpoints.

**Severity:** P1
**Impact:** Token expiry not handled, no typed safety, inconsistent error handling

### P1-4: Missing Navigation Between Related Views

Core quantitative trader workflows require navigating through disconnected pages:

**Workflow: Backtest Analysis → Live Deployment**
```
BacktestPage → BacktestHistory → BacktestDetail → PromoteToLive → LiveTrading
```
This requires 4-5 separate page navigations with no wizard/progress indicator. Only `BacktestDetail` has a "Promote to Live" button.

**Workflow: Strategy Development → Optimization**
```
StrategiesPage → StrategyEditor → BacktestPage → OptimizationPanel
```
No link from `StrategyEditor` to `BacktestPage` pre-filled with the strategy.

**Severity:** P1
**Impact:** High cognitive load, users must remember ids/names across pages

### P1-5: OptimizationPage is an Unnecessary Wrapper

**Location:** `OptimizationPage.tsx`

```tsx
export default function OptimizationPage() {
  return (
    <div className="main" style={{ maxWidth: 1000 }}>
      <OptimizationPanel />
    </div>
  );
}
```

This 3-line component adds a route and a navigation entry with zero functional value. `OptimizationPanel` could be rendered directly at `/optimize`.

**Severity:** P1
**Impact:** Route bloat, increased bundle (extra module boundary)

### P1-6: StatusPage is Redundant with AdminPage Health Tab

**Location:** `StatusPage.tsx`, `AdminPage.tsx` (health tab)

`StatusPage` fetches `system.health()` and displays a metric grid. `AdminPage` health tab fetches `admin.health()` AND `admin.systemHealth()` — superset of the same data.

**Severity:** P1
**Impact:** Two routes serving overlapping system health data

### P1-7: CalendarHeatmap Renders as Standalone SVG with No Lightweight Charts Integration

**Location:** `charts/CalendarHeatmap.tsx`

The calendar heatmap is a custom SVG implementation rather than using Lightweight Charts primitives. It lacks:
- Synchronized crosshair with the equity curve chart
- Time scale alignment with other chart components
- Consistent theme/styling with the chart library

**Severity:** P1
**Impact:** Visual inconsistency, missing cross-chart interaction

### P1-8: LiveMarket and MarketDataPage Overlap

**Location:** `LiveMarket.tsx`, `MarketDataPage.tsx`

Both pages subscribe to WebSocket tick data:
- `LiveMarket` — `/live/market` — simple tick table with last price
- `MarketDataPage` — `/market-data` — candles + CVD + divergence + tick table

`LiveMarket` is a subset of `MarketDataPage` functionality with no unique features.

**Severity:** P1
**Impact:** Route duplication, user confusion about which market view to use

---

## 4. Medium-Severity Issues (P2)

### P2-1: 8 Stub Pages

| Page | Current State |
|------|---------------|
| `LLMSettings` | Static description text only |
| `WebhookConfig` | Static description text only |
| `CredentialManagement` | Static description text only |
| `BrokerManagement` | Static description text only |
| `NotificationSettings` | Toggle checkboxes, no persistence |
| `DataSources` | Local state buttons, no persistence |
| `ForgotPasswordPage` | Hardcoded success, no API |
| `ResetPasswordPage` | Hardcoded success, no API |

**Recommendation:** Either implement or hide from navigation until ready.

### P2-2: 10+ Pages Use Inline Styles Instead of CSS

```tsx
style={{ display: 'flex', flexDirection: 'column', gap: 10 }}
style={{ display: 'flex', flexDirection: 'column', gap: 24 }}
style={{ maxWidth: 600 }}
style={{ maxWidth: 450 }}
```

These inline styles appear in `LoginPage`, `ExecutionPage`, `SettingsPage`, `BacktestPage`, `MarketDataPage`, `TwoFAPage`, `StrategiesPage`, `AdminPage`, `SimulatePage`, and `OptimizationPage`.

**Impact:** No style reusability, inconsistent spacing, cannot be themed via CSS variables.

### P2-3: Fixed Polling Intervals

| Page/Component | Endpoint | Interval |
|----------------|----------|----------|
| `LiveTrading` | `live.metrics()`, `live.equity()`, others | 10s |
| `Dashboard` | `live.metrics()`, `live.equity()` | 15s |
| `RiskPage` | `risk.status()` | 10s |
| `OptimizationPanel` | `getOptimizationStatus()` | 2s |
| `useMatrixStream` | `matrixResultsSince()` | 1500ms |

None of these intervals are adaptive (back off when idle, speed up during active trading). The matrix polling is hardcoded at 1500ms regardless of matrix size.

### P2-4: Missing Loading States

Several pages lack loading indicators:
- `StatusPage` — shows "Loading..." text, not a skeleton
- `AttributionPage` — shows "Running attribution..." text
- `CalibratePage` — shows "Running calibration..." text
- `LiveMarket` — shows "Waiting for market data..." text

`BacktestHistory` is the only page using `<SkeletonLoader/>`.

### P2-5: No Shared Layout for Metric Grids

The pattern `<div className="metric-grid">{metrics.map(m => <MetricCard ... />)}</div>` is repeated in 15+ pages. The grid columns vary:
- `Dashboard`: `grid-template-columns: repeat(3, 1fr)` (9 metrics)
- `BacktestDetail`: `grid-template-columns: repeat(5, 1fr)` (15 metrics)
- `LiveTrading`: `grid-template-columns: repeat(3, 1fr)` (6 metrics)
- `RiskPage`: `grid-template-columns: repeat(3, 1fr)` (6 metrics)
- `CalibratePage`: `grid-template-columns: repeat(3, 1fr)` (6 metrics)

No centralized grid configuration or responsive breakpoints.

### P2-6: StrategyEditor Route Confusion

Three routes map to the same component:
```
/strategies/:id
/strategies/:id/edit
/strategies/edit/:id
```

The third route uses inverted parameter order and could cause route matching ambiguity.

### P2-7: No Data Caching Between Pages

When navigating from `BacktestHistory` → `BacktestDetail` → back to `BacktestHistory`, the full list is re-fetched. There is no `useQuery`-style cache, no stale-while-revalidate, and no shared data layer between related pages.

### P2-8: `useWebSocket` Effect Has `[]` Dependencies

The WebSocket connection effect uses empty dependency array, meaning channel subscriptions are never re-synced after initial connect. The `channelsRef` provides the current channel set, but `syncSubscriptions` is only called on connect/reconnect events, not when the calling component's channels change.

### P2-9: `i18n/` Directory Has Minimal Coverage

Only `translation.json` exists in `locales/en/`. Most pages import `useTranslation` but only `LoginPage` and `PropFirmPage` have actual translated strings via `t()`. 30+ pages import the hook but use hardcoded English strings.

### P2-10: `TradingChartSection` Component Unclear Purpose

The `components/TradingChartSection.tsx` exists but no page component imports it. Potentially dead code.

### P2-11: No React.lazy Code Splitting

All 35 pages are eagerly loaded. No `React.lazy()` or `Suspense` boundaries. The initial bundle includes every page component regardless of whether the user ever visits `/admin`, `/calibrate`, or `/simulate`.

---

## 5. Low-Severity Issues (P3)

### P3-1: Sidebar Defined Inline in App.tsx

The sidebar navigation (30+ links) is defined as a large JSX block inside `App.tsx` rather than in a separate `Sidebar` component. This makes the routing file unnecessarily large (250+ lines).

### P3-2: `wsStore` is Purely Passive

The WebSocket store (`wsStore`) holds data but has no `connect`/`disconnect`/`subscribe` methods. It's completely decoupled from the `WSConnectionManager`. Data flows: `WSConnectionManager` → component handler → `wsStore.setX()`. This split of concerns is clean but means two modules must be coordinated for every WebSocket feature.

### P3-3: `toastStore` Uses `getState()` Imperatively

The `showToast()` helper uses `useToastStore.getState().addToast()` outside of React's render cycle, which is valid Zustand usage but can cause issues with React 18 strict mode double-invocation.

### P3-4: `useCandleAggregation` Not Used in MarketDataPage

`MarketDataPage` fetches candles at a fixed timeframe but doesn't offer client-side aggregation. `IndicatorsPage` uses `useCandleAggregation` and `TimeframeChips` but `MarketDataPage` does not — inconsistent feature set.

### P3-5: `chartUtils.ts` Exports Overlapping with Chart Components

`equityToLineData()` and `candlesToData()` are exported utilities but `EquityCurveChart` and `CandlesChart` have their own internal data conversion logic.

### P3-6: `workers/monteCarlo.worker.ts` Not Locatable

The Monte Carlo worker is referenced by `MonteCarloChart` as `new Worker(new URL('../workers/monteCarlo.worker.ts', import.meta.url))` — this is correct for Vite but means the worker is a separate chunk that must be tested independently.

### P3-7: `components/TradingViewProvider.tsx` Contains No TradingView Integration

Despite the name, this component does not import TradingView's charting library. It's likely a placeholder or misnamed.

---

## 6. Quantitative Trader UX Assessment

### 6.1 Strengths

- **Equity curve with trade markers:** `EquityCurveChart` renders buy/sell arrows and benchmark overlays — essential for strategy evaluation.
- **Monte Carlo simulation:** `MonteCarloChart` with Web Worker-based computation, P5/P25/P50/P75/P95 bands, and summary statistics.
- **Comparison mode:** `BacktestHistory` supports multi-select with equity curve overlay — critical for strategy comparison.
- **Calendar heatmap:** Monthly returns heatmap with click-to-filter — good for seasonality analysis.
- **Matrix streaming:** `MatrixResultsPanel` with incremental delta updates — avoids blocking UI during large parameter sweeps.
- **Sensitivity analysis:** `useParameterSensitivity` hook provides Sharpe gradient visualization.
- **Trade list with filtering:** `TradesTab` provides sortable/filterable trade data.

### 6.2 Critical Gaps

1. **No walk-forward analysis view:** `BacktestDetail` shows optimization and comparison tabs but no walk-forward equity curve with IS/OOS shading.
2. **No strategy correlation matrix:** When comparing multiple strategies, no correlation heatmap or pairwise metrics table.
3. **No regime-contextualized metrics:** `BacktestDetail` fetches `regimeStats` but doesn't display regime-filtered Sharpe/MaxDD breakdowns.
4. **No parameter sensitivity surface plot:** `useParameterSensitivity` returns a flat list — no 2D heatmap for parameter interaction effects.
5. **No trade distribution analytics:** No histogram of trade durations, no PnL distribution chart, no MAE/MFE analysis.
6. **Risk indicators lack context:** Drawdown progress bars show "% used" but don't show at what price level the limit would be breached or how many losing trades remain.

### 6.3 Navigation Cognitive Load

A typical quant workflow requires:

| Task | Pages Visited | Clicks |
|------|---------------|--------|
| Run backtest → analyze results | BacktestPage → BacktestHistory → BacktestDetail | 3+ |
| Compare two strategies | BacktestHistory (select, compare) | 2-3 |
| Optimize strategy params | BacktestPage (switch to optimize mode) OR OptimizationPage | 1-2 |
| Deploy strategy live | BacktestDetail → PromoteToLive → LiveTrading | 3 |
| Check live risk during trading | Dashboard OR RiskPage OR LiveTrading | 1-2 |

**Total for full workflow: 8-12 page transitions.** A well-designed dashboard should reduce this to 2-3 context switches.

---

## 7. Charting Audit (Lightweight Charts Implementation)

OrcaAlgo uses TradingView Lightweight Charts (LWCs) 5.2 as its sole charting library across 13 chart modules. This section audits the charting implementation against TradingView-grade UX standards — the benchmark for professional algorithmic trading dashboards.

### 7.1 Chart Component Inventory

| Component | Type | Data | Crosshair | Trade Markers | Indicators | Fullscreen |
|-----------|------|------|-----------|---------------|------------|------------|
| `EquityCurveChart` | Line + Area | Equity data | ✅ | ✅ | ❌ | ❌ |
| `LiveMonitorChart` | Candlestick + Volume | Live ticks | ✅ | ✅ | ✅ | ✅ |
| `CandlesChart` | Candlestick + Volume | Static candles | ❌ | ❌ | ❌ | ❌ |
| `CVDChart` | Histogram | CVD data | ❌ | ❌ | ❌ | ❌ |
| `DailyReturnsChart` | Histogram | Returns | ❌ | ❌ | ❌ | ❌ |
| `MonteCarloChart` | Lines + Areabands | MC percentiles | ✅ | ❌ | ❌ | ❌ |
| `CalendarHeatmap` | Custom SVG | Monthly returns | N/A | ❌ | ❌ | ❌ |
| `YearlySummaryTable` | HTML Table | Derived from returns | N/A | ❌ | ❌ | ❌ |
| `CrosshairTooltip` | Overlay div | Aggregated | N/A | ❌ | ❌ | ❌ |
| `MarkerManager` | Utility | — | N/A | ✅ | ❌ | ❌ |

### 7.2 Feature Gap Analysis Against TradingView Standards

| Feature | TradingView Standard | OrcaAlgo Status | Severity | Action |
|---------|---------------------|-----------------|----------|--------|
| **Crosshair tooltip (OHLC + indicators)** | Unified tooltip displaying OHLCV + all active indicator values at crosshair position | ⚠️ Partial — only `LiveMonitorChart` and `EquityCurveChart` implement crosshair; `CandlesChart`, `CVDChart`, `DailyReturnsChart` lack it entirely | P0 | Implement `subscribeCrosshairMove` on all chart components; standardize `CrosshairTooltip` to accept OHLCV + indicator rows |
| **Multi-timeframe synchronized crosshairs** | Moving crosshair on one chart updates all visible charts to the same timestamp | ❌ Missing | P1 | When multiple charts share a container (e.g., `CandlesChart` + `CVDChart`), synchronize via shared `timeScale().subscribeVisibleLogicalRangeChange()` |
| **Symbol search on chart** | Typeahead search in chart header for rapid symbol switching | ❌ Missing — symbol selection is always on a separate page section or outside the chart boundary | P1 | Add inline `<SymbolSearch />` component to chart headers (`LiveMonitorChart`, `CandlesChart`, `MarketDataPage`) |
| **Timeframe selector on chart toolbar** | Clickable timeframe chips (1m, 5m, 15m, 1h, 4h, 1D, 1W) in the chart's top-left toolbar | ⚠️ Partial — `TimeframeChips` exists but lives in page headers, not on the chart canvas | P1 | Move `TimeframeChips` into chart-level toolbar; make it a prop of chart components |
| **Indicator management on chart** | Add/remove indicators via a modal or sidebar anchored to the chart, not a separate page | ❌ Missing — indicator config is on `/indicators` (a separate route/page); `LiveMonitorChart` supports inline indicator computation but with no config UI | P0 | Extract indicator management from `IndicatorsPage` into a reusable `<IndicatorConfigModal />` that can be triggered from any chart's toolbar button |
| **Drawing tools (trendlines, horizontals, rectangles)** | Toolbar with: trendline, horizontal line/ray, rectangle, Fibonacci retracement, text annotation | ⚠️ Partial — only trendline drawing is implemented via `useDrawingTool` hook; no tool palette, no persistent annotations, no snap-to-price | P2 | Expand `useDrawingTool` with mode palette (trendline/horizontal/rect); add annotation persistence via marks |
| **Chart theme consistency** | CSS variables drive all color decisions; dark/light mode via single toggle | ⚠️ Partial — `chartConfig.ts` reads 7 CSS variables for chart colors, but page-level containers use hardcoded hex values (`#1a1a2e`, `var(--danger)`) that do not update with theme changes | P1 | Replace all hardcoded hex/var() in page containers with Tailwind `dark:` utilities or CSS variable tokens; audit 30+ files for `style={{color:'var(--X)'}}` patterns |
| **OHLCV header overlay** | Semi-transparent header at chart top showing current bar's O, H, L, C, V in monospace font | ✅ Present — `OHLCVHeader` component renders on crosshair move | Keep | — |
| **Trade markers (entry/exit arrows)** | Green up-arrow at entry, red down-arrow at entry; circles at exits; hover tooltip with trade details | ✅ Present — `MarkerManager.tradesToMarkers()` + `useTradeTooltip` | Keep | — |
| **Volume profile (visible range)** | Histogram on right price scale showing volume-at-price for the visible range | ❌ Missing — `volume_profile.go` exists in Go backend but has no frontend renderer | P2 | Implement `VolumeProfileChart` component using lightweight-charts histogram series |
| **Chart performance (1K-10K candles)** | Smooth pan/zoom at 60fps with 10,000+ OHLCV bars | Unknown — no benchmark test exists; `chartUtils.ts` performs `dedupeByTime` which is O(n) per render; `LiveMonitorChart` uses `requestAnimationFrame` batching via `useChartUpdate` — good pattern but unmeasured | P1 | Add performance benchmark: load 5,000/10,000 candles, measure render time and scroll FPS; add `React.memo` on chart components; verify `react-window` virtualization is not interfering with LWCs canvas updates |
| **Cross-chart time synchronization** | Pan/zoom one chart → all charts in view follow the same time range | ❌ Missing | P2 | Implement via shared `timeScale().subscribeVisibleLogicalRangeChange()` + broadcast pattern |
| **Price scale synchronization** | Multiple chart panes share a synchronized right price scale for overlay comparisons | ❌ Missing — `EquityCurveChart` uses independent price scales for equity line (right) and drawdown area (left), but no cross-chart sync | P2 | Add `priceScale('right').applyOptions({ autoScale: true })` with shared scale reference across stacked charts |
| **Responsive chart sizing** | Charts resize to fill container on viewport changes | ✅ Present — `useChart` hook implements `ResizeObserver` | Keep | — |
| **Keyboard shortcuts** | `+/-` zoom, arrow keys pan, `0` fit content, `Ctrl+Z` undo drawing | ✅ Present — `useChartKeyboard` hook | Keep | — |
| **Export (PNG/SVG)** | Right-click → save as image; or toolbar button | ⚠️ Partial — `LiveMonitorChart` supports PNG export; no other chart component does | P1 | Standardize export via a shared `useChartExport` hook or chart toolbar button across all chart components |
| **Undo/redo draw actions** | Drawing tool undo stack | ❌ Missing | P3 | Add to `useDrawingTool` when expanded |

### 7.3 Chart Architecture Assessment

**Strengths:**
- `useChart` factory hook correctly handles `ResizeObserver`, theme mutation observation, and cleanup on unmount — LWCs integration is idiomatic
- `useChartUpdate` (requestAnimationFrame batching) is the correct pattern for high-frequency tick updates
- `chartConfig.ts` centralizes defaults and color extraction from CSS variables
- `chartUtils.ts` data conversion utilities are correct and include deduplication
- `MarkerManager` cleanly separates marker creation from chart rendering

**Weaknesses:**
- Chart components are inconsistent — `LiveMonitorChart` is fully featured (crosshair + OHLCV header + indicators + export + fullscreen + drawing + keyboard), while `CandlesChart` renders bare candlesticks with no interaction
- No shared chart toolbar pattern — each chart component independently implements (or doesn't) its own controls
- `CalendarHeatmap` is a custom SVG detached from LWCs — no time synchronization, no consistent theming
- Chart export is inconsistent — `LiveMonitorChart` supports PNG via `dom-to-image` but `EquityCurveChart` does not
- Monte Carlo simulation runs in a Web Worker (correct) but the worker file path depends on Vite's URL resolution — this breaks if the worker is moved

### 7.4 Charting Recommendations Summary

| Priority | Task | Components Affected |
|----------|------|---------------------|
| **P0** | Add crosshair tooltip to `CandlesChart`, `CVDChart`, `DailyReturnsChart` | 3 |
| **P0** | Extract indicator management from `IndicatorsPage` to reusable chart modal | 3 |
| **P1** | Add inline symbol search to chart headers | 4 |
| **P1** | Move timeframe selector from page header to chart toolbar | 5 |
| **P1** | Replace hardcoded colors with CSS variables across all containers | 30+ |
| **P1** | Add chart performance benchmarks (5K/10K candles) | All |
| **P1** | Standardize chart export across all chart components | 4 |
| **P2** | Multi-chart time synchronization | 2 |
| **P2** | Drawing tool palette (trendline/horizontal/rectangle) | 1 |
| **P2** | Volume Profile chart component | 1 |
| **P3** | Undo/redo for drawings | 1 |

---

## 8. Responsive & Multi-Device Strategy

### 8.1 Device Tier Classification

| Tier | Resolution | Viewport | Target Users | Design Strategy |
|------|-----------|----------|-------------|-----------------|
| **Primary** | 1920×1080 | Desktop (single monitor) | Core trading workflow — backtest analysis, live monitoring, strategy configuration | Full-featured: metric grids at 3-5 columns, charts at full width, sidebar expanded |
| **Secondary** | 2560×1440 / dual 1920×1080 | Desktop (multi-monitor) | Power users — chart on primary, risk dashboard on secondary | Grid snapping for multi-window layouts; detachable chart panes |
| **Tertiary** | 1440×900 | Laptop | Portfolio review, quick checks, emergency access | Metric grids collapse to 2 columns; charts reduce height to 50% of viewport; sidebar collapses to icon-only |
| **Fallback** | 1280×800 | Tablet landscape | Emergency monitoring — kill-switch access, PnL check, position liquidation | Minimal mode: single-column layout; only risk controls, order form, and live PnL visible; charts hidden |
| **Not supported** | < 768px width | Mobile portrait | N/A | Show a static "Desktop Required" screen with a link to the desktop URL. Do not attempt to render charts or tables below this breakpoint — the information density required for quantitative trading cannot be served on a mobile portrait viewport. |

### 8.2 Responsive Implementation with Tailwind

With the recommended Tailwind CSS 4 migration, responsive breakpoints are applied as utility prefixes:

```
// Primary (desktop): default — no prefix
<div className="grid grid-cols-3 gap-6">

// Tertiary (laptop 1440px): xl: breakpoint
<div className="grid grid-cols-3 xl:grid-cols-2 gap-6">

// Fallback (tablet 1280px): lg: breakpoint
<div className="grid grid-cols-3 xl:grid-cols-2 lg:grid-cols-1 gap-6">

// Not supported (< 768px): md: breakpoint
<div className="hidden md:block">
  {/* All trading content */}
</div>
<div className="block md:hidden">
  <DesktopRequiredMessage />
</div>
```

**Tailwind breakpoint configuration:**
```typescript
// tailwind.config.ts
theme: {
  screens: {
    'lg': '1280px',   // Tablet landscape fallback
    'xl': '1440px',   // Laptop
    '2xl': '1920px',  // Desktop primary
    '3xl': '2560px',  // Multi-monitor
  }
}
```

### 8.3 Component-Level Responsive Rules

| Component | Primary (≥1920px) | Tertiary (1440px) | Fallback (1280px) |
|-----------|-------------------|-------------------|-------------------|
| **Metric grids** | 3-5 columns | 2-3 columns | 2 columns, condensed cards |
| **Charts** | Full viewport width, 400-600px height | Full width, 300px height | Hidden (`hidden lg:block`) |
| **Data tables** | Full width, 10-15 rows visible | Full width, 8 rows visible | Full width, 5 rows visible |
| **Sidebar** | Expanded (200px), icons + labels | Collapsed (60px), icons only | Hidden; hamburger menu |
| **Forms** | 2-column layout | Single column | Single column, condensed spacing |
| **BacktestConfigBar** | Inline expanded | Collapsible sections | Single-column stacked |
| **MatrixResultsPanel** | 4-column sortable grid | 2-column grid | Single-column list |
| **Risk controls** | Always visible | Always visible | Always visible (emergency access) |
| **Trade markers** | Full resolution | Full resolution | Reduced marker size |

### 8.4 Critical Mobile/Emergency Requirements

The one justified mobile use case is **emergency kill-switch access**. An authorized user must be able to:

1. Log in from a mobile browser
2. View current PnL and drawdown status
3. Trigger emergency stop (with 2FA)
4. Verify positions are closing

This requires a dedicated lightweight emergency page (`/emergency`) that:
- Loads zero charting libraries (saves 300+ KB)
- Fetches only `risk.status()` and `positions.list()`
- Renders a single-column layout with: balance, PnL, drawdown bar, emergency stop button
- Total page weight under 50 KB uncompressed

### 8.5 Multi-Monitor Strategy

For power users running the dashboard across 2-3 monitors:

- **Monitor 1 (primary):** Charts + backtest interface (Command Center or BacktestDetail)
- **Monitor 2 (secondary):** Risk dashboard + positions (open a second browser window at `/?view=risk-only`)
- **Monitor 3 (tertiary):** Live market data (`/market-data?view=compact`)

Support this via URL query parameters that control which panels are visible:
```
/?view=risk-only          → Only renders risk panel (no charts, no tables)
/?view=positions-only     → Only renders positions table
/market-data?view=compact  → Hides config section, shows only chart + ticks
```

---

## Appendix A: Complete Page Inventory

| # | Page | Lines (approx) | API Calls | WebSocket | Status |
|---|------|---------------|-----------|-----------|--------|
| 1 | `Dashboard` | 120 | 2 | risk | ✅ Active |
| 2 | `LiveTrading` | 150 | 6 | ticks | ✅ Active |
| 3 | `RiskPage` | 160 | 4 | risk | ✅ Active |
| 4 | `BacktestPage` | 330 | 1 | — | ✅ Active |
| 5 | `BacktestHistory` | 280 | 4 | — | ✅ Active |
| 6 | `BacktestDetail` | 350 | 9 | — | ✅ Active |
| 7 | `StrategiesPage` | 280 | 5 | — | ✅ Active |
| 8 | `StrategyEditor` | 200 | 6 | — | ✅ Active |
| 9 | `ExecutionPage` | 200 | 3 | — | ✅ Active |
| 10 | `MarketDataPage` | 160 | 1 | ticks+cvd+divergence | ✅ Active |
| 11 | `IndicatorsPage` | 250 | 3 | — | ✅ Active |
| 12 | `LiveMarket` | 90 | 0 | ticks | ✅ Active |
| 13 | `SimulatePage` | 280 | 3 | — | ✅ Active |
| 14 | `CalibratePage` | 160 | 1 | — | ✅ Active |
| 15 | `AttributionPage` | 150 | 1 | — | ✅ Active |
| 16 | `OptimizationPanel` | 200 | 3 | — | ✅ Active |
| 17 | `OptimizationPage` | 10 | 0 | — | ⚠️ Wrapper |
| 18 | `AccountsPage` | 200 | 5 | — | ✅ Active |
| 19 | `PropFirmPage` | 250 | 6 | — | ✅ Active |
| 20 | `AdminPage` | 300 | 8 | — | ✅ Active |
| 21 | `SymbolAdminPage` | 250 | 8 | — | ✅ Active |
| 22 | `UniversePage` | 200 | 6 | — | ✅ Active |
| 23 | `StatusPage` | 60 | 1 | — | ⚠️ Redundant |
| 24 | `SettingsPage` | 120 | 2 | — | ✅ Active |
| 25 | `TwoFAPage` | 120 | 2 | — | ✅ Active |
| 26 | `LoginPage` | 60 | 1 | — | ✅ Active |
| 27 | `RegisterPage` | 70 | 1 | — | ✅ Active |
| 28 | `ForgotPasswordPage` | 50 | 0 | — | ❌ Stub |
| 29 | `ResetPasswordPage` | 60 | 0 | — | ❌ Stub |
| 30 | `LLMSettings` | 30 | 0 | — | ❌ Stub |
| 31 | `WebhookConfig` | 30 | 0 | — | ❌ Stub |
| 32 | `CredentialManagement` | 30 | 0 | — | ❌ Stub |
| 33 | `BrokerManagement` | 30 | 0 | — | ❌ Stub |
| 34 | `DataSources` | 70 | 0 | — | ❌ Stub |
| 35 | `NotificationSettings` | 50 | 0 | — | ❌ Stub |

---

## Appendix B: API Call Frequency Map

| Endpoint | Callers | Calls per page load |
|----------|---------|---------------------|
| `backtests.metrics(id)` | BacktestDetail, BacktestHistory (per row) | 1 + N |
| `backtests.equity(id)` | BacktestDetail, BacktestHistory (compare) | 1-3 |
| `live.metrics()` | Dashboard (15s), LiveTrading (10s) | 2 concurrent polls |
| `live.equity('90d')` | Dashboard (15s), LiveTrading (10s) | 2 concurrent polls |
| `risk.status()` | RiskPage (10s) | 1 poll |
| `GET /api/v1/symbols` | SymbolAdminPage, IndicatorsPage, MarketDataPage | 3 per session |
| `backtests.list()` | BacktestHistory | 1 |
| `strategies.list()` | StrategiesPage, BacktestPage, StrategyEditor | 3 per session |
| `orders.list()` | ExecutionPage, LiveTrading (10s) | 1 + poll |
| `positions.list()` | LiveTrading (10s) | 1 + poll |

---

## Appendix C: UI Framework Evaluation & Recommendation

### C.1 Current Stack Assessment

The project currently has **no UI component library or CSS framework**. Styling is implemented through a mix of:

| Approach | Usage | Files |
|----------|-------|-------|
| CSS classes (`.card`, `.metric-grid`, `.btn-primary`, `.flex-between`) | Global stylesheet | `index.css` |
| Inline `style={{...}}` objects | Form layouts, spacing overrides | 10+ files |
| Lightweight Charts built-in theming | Chart colors via CSS variables | `chartConfig.ts` |

**Strengths:** Zero external CSS dependency, fast initial load, full control.
**Weaknesses:** No design system, inconsistent spacing/typography, 10+ files with inline styles, no responsive breakpoints, no dark/light theme toggle, no accessible component primitives (modals, tooltips, dropdowns), no data table with sorting/filtering.

### C.2 Constraints for Algorithmic Trading Dashboards

The following constraints are specific to trading dashboards and differ from general web applications:

| Constraint | Weight | Description |
|------------|--------|-------------|
| **Data table performance** | Critical | Trade lists (500-50K rows), backtest results matrices, order books — require virtualization, sorting, filtering, and streaming updates without blocking the main thread |
| **Real-time updates** | Critical | WebSocket tick data, live PnL, risk status — UI must handle 20+ updates/second without jank or stale closures |
| **Dark theme** | Mandatory | Trading desks operate in low-light environments; light themes cause eye strain during extended monitoring sessions |
| **Chart integration** | Mandatory | Lightweight Charts is already integrated and must remain the charting solution — any UI framework must coexist with LWCs' canvas-based rendering |
| **Form density** | High | Strategy configs have 20-50 parameters; optimization configs have parameter range tables — forms must be compact and responsive |
| **Bundle size** | High | Traders often run the dashboard alongside heavy backtest processes; the UI should not compete for memory |
| **Accessibility** | Medium | Keyboard navigation for power users, but not a regulatory requirement |
| **Multi-monitor** | Medium | Dashboard often spans 2-3 monitors — responsive grid layout is more important than mobile breakpoints |
| **i18n readiness** | Low | Single-language (English) for now, but framework choice should not preclude future localization |

### C.3 Candidate Evaluation

Six frameworks were evaluated against the constraints above. Each was scored on a 1-5 scale (5 = best fit) weighted by constraint importance.

#### C.3.1 Tailwind CSS + shadcn/ui (Recommended)

| Constraint | Score | Rationale |
|------------|-------|-----------|
| Data table performance | 4 | shadcn/ui provides headless table primitives; pair with **TanStack Table** for sorting/filtering/virtualization |
| Real-time updates | 5 | Zero overhead — utility classes compile to atomic CSS at build time; no runtime CSS-in-JS cost |
| Dark theme | 5 | Native dark mode via `dark:` prefix; CSS variables for seamless theme switching without JS |
| Chart integration | 5 | Non-invasive — utility classes style the container, not the canvas; no z-index or stacking conflicts |
| Form density | 4 | shadcn/ui form components are compact and composable; pairs with react-hook-form+zod (already in use) |
| Bundle size | 5 | Build-time CSS generation + tree-shaking; shadcn/ui is copy-paste (not a dependency); ~15 KB gzipped for full component set |
| Accessibility | 4 | shadcn/ui components built on Radix primitives — WCAG 2.1 AA compliant out of the box |
| Multi-monitor | 5 | Responsive utilities (`lg:`, `xl:`, `2xl:`) map naturally to monitor widths |
| i18n readiness | 4 | No blocking factor — utility classes are language-agnostic |
| **Weighted total** | **4.7** | Best overall fit for this specific application profile |

**Key advantages over alternatives:**
- **Tailwind eliminates all inline styles** — the audit found 10+ files using `style={{display:'flex',flexDirection:'column',gap:X}}`. Every one becomes `flex flex-col gap-4`.
- **shadcn/ui is not a dependency** — components are copied into `src/components/ui/`, giving full control over implementation. No version-lock, no breaking changes.
- **Dark theme is a single `dark:` prefix** — `bg-white dark:bg-gray-900` replaces complex theme logic.
- **TanStack Table integration is first-class** — shadcn/ui's table component is a thin wrapper around TanStack Table v8 headless primitives.
- **Migration is incremental** — Tailwind can be added alongside existing CSS; inline styles can be replaced file-by-file.

#### C.3.2 Ant Design (antd)

| Constraint | Score | Rationale |
|------------|-------|-----------|
| Data table performance | 5 | `<Table>` component is best-in-class: built-in sorting, filtering, pagination, virtualization, row selection |
| Real-time updates | 3 | CSS-in-JS (emotion) runtime adds per-render overhead; observable on 20+ updates/sec |
| Dark theme | 4 | `ConfigProvider` with `theme={{ algorithm: theme.darkAlgorithm }}` — functional but heavy |
| Chart integration | 3 | Opinionated design system competes with Lightweight Charts' canvas rendering for z-index/styling control |
| Form density | 5 | `<Form>` component with layout modes, validation integration, compact density option |
| Bundle size | 2 | ~300 KB gzipped (tree-shaken); adds significant weight even with selective imports |
| Accessibility | 5 | Full ARIA support, keyboard navigation |
| Multi-monitor | 3 | Responsive grid via `Row`/`Col` but not as flexible as utility classes |
| i18n readiness | 5 | Built-in `ConfigProvider` locale support |
| **Weighted total** | **3.5** | Excellent component quality but too heavy for a real-time trading dashboard; CSS-in-JS runtime conflicts with high-frequency updates |

#### C.3.3 MUI (Material UI)

| Constraint | Score | Rationale |
|------------|-------|-----------|
| Data table performance | 5 | `DataGrid` (MUI X) — premium data grid with streaming updates, column pinning, aggregation |
| Real-time updates | 2 | CSS-in-JS (emotion) runtime; worst performer on high-frequency update benchmarks |
| Dark theme | 5 | `ThemeProvider` with `mode: 'dark'` — mature theming system |
| Chart integration | 3 | Material Design aesthetic conflicts with trading dashboard conventions; heavy opinionated styling |
| Form density | 3 | Material Design spacing defaults create low information density; requires significant override |
| Bundle size | 1 | ~400 KB gzipped (tree-shaken); largest of all candidates; DataGrid requires separate MUI X license |
| Accessibility | 5 | Industry-leading ARIA support |
| Multi-monitor | 3 | `Grid2` component but Material breakpoints target mobile/tablet, not multi-monitor |
| i18n readiness | 5 | Built-in locale support |
| **Weighted total** | **2.9** | Excellent for admin dashboards but CSS-in-JS performance overhead and low information density disqualify for trading use |

#### C.3.4 Tremor

| Constraint | Score | Rationale |
|------------|-------|-----------|
| Data table performance | 3 | Basic `<Table>` component; no built-in sorting, filtering, or virtualization |
| Real-time updates | 5 | Tailwind-based (compile-time CSS); zero runtime overhead |
| Dark theme | 5 | Native Tailwind dark mode |
| Chart integration | 2 | Opinionated chart components (recharts-based) compete with Lightweight Charts; would need to disable/override |
| Form density | 2 | Limited form components; focused on dashboard widgets, not configuration interfaces |
| Bundle size | 5 | Tailwind + small component library; ~40 KB gzipped |
| Accessibility | 2 | Limited ARIA coverage; not a priority for the library |
| Multi-monitor | 5 | Tailwind responsive utilities |
| i18n readiness | 3 | No built-in support |
| **Weighted total** | **2.7** | Built for dashboards but dashboard-widget-focused, not trading-workflow-focused. Lacks the form density and data table sophistication required. |

#### C.3.5 AG Grid Community

AG Grid is evaluated as a **data table supplement**, not a UI framework replacement. It would pair with any of the above CSS frameworks.

| Constraint | Score | Rationale |
|------------|-------|-----------|
| Data table performance | 5 | Industry benchmark; handles 100K+ rows with virtualization, streaming updates, column pinning, aggregation, pivoting, Excel-like filtering |
| Real-time updates | 5 | `applyTransaction()` API for delta-based updates without full re-render; `asyncTransaction` for WebSocket streams |
| Dark theme | 4 | Built-in dark themes (Alpine Dark, Balham Dark); customizable via CSS variables |
| Chart integration | 5 | Independent of charting library |
| Form density | N/A | Not a form framework |
| Bundle size | 3 | ~200 KB gzipped (community edition, tree-shaken) |
| Accessibility | 5 | Full WCAG 2.1 AA compliance |
| Multi-monitor | 5 | Responsive column sizing |
| i18n readiness | 5 | Built-in locale support for 40+ languages |
| **Weighted total** | **4.4** | **Strongly recommended as a supplement** for trade lists, backtest result matrices, order tables, and universe management grids where TanStack Table would require significant custom implementation |

#### C.3.6 Keep Current (No Framework)

| Constraint | Score | Rationale |
|------------|-------|-----------|
| Data table performance | 1 | No built-in sorting, filtering, or virtualization. Each table is manually implemented with `<table>` elements. |
| Real-time updates | 5 | No overhead (but no structured update pattern either) |
| Dark theme | 2 | CSS variables used for charts but no systematic dark mode across all components |
| Chart integration | 5 | Already integrated |
| Form density | 1 | Inline styles create inconsistent layouts; no grid system for form fields |
| Bundle size | 5 | Zero overhead (but maintenance cost is externalized) |
| Accessibility | 1 | No ARIA attributes, no focus management, no keyboard navigation patterns |
| Multi-monitor | 2 | No responsive breakpoints |
| i18n readiness | 2 | react-i18next integrated but no component-level locale support |
| **Weighted total** | **2.8** | Acceptable for MVP but not viable for production trading dashboard. Technical debt accumulates with every new page. |

### C.4 Firm Recommendation

**Primary: Tailwind CSS 4 + shadcn/ui + TanStack Table v8**

This combination is the optimal choice for the OrcaAlgo dashboard for the following reasons:

1. **Compile-time CSS** — Zero runtime overhead during high-frequency WebSocket updates. Unlike CSS-in-JS solutions (MUI, Ant Design), Tailwind generates static CSS at build time. No JavaScript execution per-render for styling.

2. **Non-invasive to Lightweight Charts** — Utility classes style the React wrapper elements (containers, headers, controls) without touching the canvas. Both Ant Design and MUI have opinionated box models that frequently conflict with canvas-based libraries.

3. **shadcn/ui is copy-paste, not a dependency** — Components live in the project's `src/components/ui/` directory. This means full control, no breaking changes from upstream, and the ability to modify component internals for trading-specific behaviors (e.g., number formatting, color thresholds, streaming data patterns).

4. **TanStack Table addresses the #1 weakness** — The current codebase has 12+ pages with custom `<table>` implementations, none with sorting, filtering, or virtualization. TanStack Table v8 (headless) provides all three while leaving rendering to shadcn/ui's table primitives. Migration from raw `<table>` to TanStack Table is mechanical and per-component.

5. **Gradual migration path** — Tailwind coexists with existing CSS. Pages can be migrated one at a time. Inline styles can be replaced with utility classes as each file is touched.

6. **Dark theme is trivial** — `dark:bg-gray-900` in Tailwind is the entire dark mode implementation. No theme provider, no context, no runtime switching cost.

**Secondary: AG Grid Community (supplement for data-heavy tables)**

For trade lists (BacktestDetail TradesTab, BacktestHistory), universe symbol grids, and admin tables where row counts exceed 1,000, AG Grid Community provides:
- Server-side row model for lazy loading from paginated APIs
- `applyTransaction()` for WebSocket-driven delta updates (live trade feed)
- Column pinning, aggregation, and Excel-like filtering without custom implementation
- The community edition is MIT-licensed with zero cost

**Migration cost estimate:**

| Phase | Work | Effort |
|-------|------|--------|
| 1. Install Tailwind | `npm install tailwindcss @tailwindcss/vite` + config | 30 min |
| 2. Install shadcn/ui | `npx shadcn@latest init` + add components (button, card, dialog, tabs, table, form, badge, select, input) | 1 hour |
| 3. Replace global CSS | Map `.card`, `.metric-grid`, `.btn-*`, `.flex-between` to Tailwind utilities; delete `index.css` rules except chart variables | 2 hours |
| 4. Migrate shared components | `MetricCard`, `FormField`, `ConfirmDialog`, `SkeletonLoader`, `AppHeader` | 4 hours |
| 5. Migrate high-traffic pages | `Dashboard`, `LiveTrading`, `BacktestDetail`, `BacktestPage` | 8 hours |
| 6. Migrate remaining pages | 30+ pages, incremental, one per PR | 20 hours |
| 7. Add TanStack Table | Replace raw `<table>` in `TradesTab`, `BacktestHistory`, admin pages | 6 hours |
| 8. Optional: AG Grid | Replace TanStack Table for trade lists (>1000 rows) and universe grids | 4 hours |
| **Total** | | **~45 hours** (can be parallelized across contributors) |

### C.5 What NOT to Migrate

The following existing libraries should be **kept and not replaced**:

| Library | Reason to Keep |
|---------|---------------|
| **Lightweight Charts 5.2** | Industry-standard trading chart library; already well-integrated with custom hooks; no alternative matches its performance for OHLCV rendering |
| **Zustand 5.0** | Minimal, fast state management; correct choice for trading dashboards (no boilerplate, no providers, atomic selectors) |
| **react-hook-form + zod** | Already used in 4 pages; best-in-class for form-heavy config interfaces; pairs directly with shadcn/ui form components |
| **react-window 2.2** | Already used via `useWindowedRows`; TanStack Table can optionally use react-window for row virtualization |
| **react-hot-toast** | Lightweight toast library; shadcn/ui has its own `<Sonner>` but migration is optional |
| **plotly.js-dist-min** | Only if used for 3D parameter surface plots; otherwise consider removing to reduce bundle |
