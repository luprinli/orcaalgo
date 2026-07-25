# `web/` — React 18 + TypeScript 5 Trading Dashboard

Single Page Application with Tailwind CSS 4, shadcn/ui components, and TradingView Lightweight Charts. Provides real-time trading monitoring, backtest execution, strategy management, walk-forward analysis, and broker/symbol configuration. Uses WebSocket for live data and a shared `useLiveRiskData` hook.

[↑ Back to Root README](../README.md)

## Tech Stack

- **React 18** + **TypeScript 5**
- **Vite 5** — build tool and dev server with `@tailwindcss/vite` plugin
- **Tailwind CSS 4** — utility-first CSS with trading-optimized dark theme
- **shadcn/ui** — 31 accessible components (Button, Card, Dialog, Tabs, Table, Badge, Input, Select, Label, Skeleton, Tooltip, Textarea, AlertDialog, Accordion, Avatar, Breadcrumb, Checkbox, Collapsible, Command, DropdownMenu, HoverCard, Input, Label, Popover, Progress, RadioGroup, ScrollArea, Separator, Sheet, Slider, Switch, ToggleGroup)
- **lightweight-charts 5** — financial charting (candlestick, equity curve, CVD, volume profile, Monte Carlo)
- **Zustand 5** — state management (auth, cache, trades, indicators, matrix, timeframe, WebSocket data)
- **@tanstack/react-table** — sortable/filterable data tables
- **plotly.js** — parameter sensitivity heatmaps
- **WebSocket** — real-time data via custom `useWebSocket` hook

## Trading Theme

Dark-mode optimized for extended monitoring sessions:
- Background: `#090d14` (deep navy)
- Card surface: `#111827`
- Primary accent: `#3b82f6` (electric blue)
- High-density spacing: 12px cards, 8px gaps, 32px buttons
- Tabular numerics: `font-variant-numeric: tabular-nums` on all data elements
- See `src/lib/trading-theme.ts` for full design token reference

## Navigation — 3 Groups, 13 Items

```
Trading Desk:
  Dashboard (/)              → MonitorPage — 4 tabs (Overview, Positions & Orders, Risk, Signals)
  Execution (/execution)     → Order placement + active orders
  Backtesting (/backtest)    → BacktestHub — Runner (Matrix/Single/Optimize) + History + Detail + Promote-to-Live
  Strategies (/strategies)   → StrategyHub — Catalog + Instances + Editor

Analysis:
  Charts (/charting)         → ChartingHub — Candles + Indicators
  Calibration (/calibrate)   → Brier score audit pipeline
  Attribution (/attribution) → PnL slicing with Wilson CIs
  Simulation (/simulate)     → 7 sub-tabs: generate, calibrate, validate, calibrate-regime, ticks, inject-signal, validate-regime

Settings:
  System (/settings)         → SettingsPage — 4 tabs (Trading, Webhooks, Notifications, LLM)
  Integrations (/integrations) → Brokers, Providers & Symbols, Credentials
  Accounts (/accounts)       → Trading accounts + Prop Firm profiles
  Admin (/admin)             → 9-tab admin: Health, Users, Audit, Errors, Email, Seed, ML Models, Reconciliation, Data Quality
  Emergency (/emergency)     → Mobile-friendly kill-switch (no auth required)
```

Still accessible via redirect: `/2fa`, `/propfirm`

## Pages

| Page | Path | Description |
|------|------|-------------|
| **MonitorPage** | `/` | Merged Dashboard + LiveTrading + Risk. 4 tabs: Overview (9 KPIs + equity + risk bars), Positions & Orders, Risk (emergency stop/resume + regime), Signals. Real-time via WebSocket + REST polling. System status indicators (broker, data feed, DB, WS) now use real health endpoint data |
| **ExecutionPage** | `/execution` | Order placement (market/limit/stop/stop_limit) with active orders table |
| **BacktestHub** | `/backtest` | **Runner**: Matrix/Single/Optimize modes with strategy multi-select, symbol input, timeframe checkboxes, optimize fields (objective, train/test years, step months, max combos). OptimizationPanel integrated into Optimize mode. **History**: Table with lazy-loaded metrics, compare mode with correlation matrix, rerun/delete. **Detail**: 17 metrics in collapsible groups (Primary/Advanced/Costs), equity curve, daily returns, Monte Carlo, calendar heatmap, yearly summary, regime breakdown, trade list, optimization, live comparison, Promote-to-Live 3-step wizard |
| **StrategyHub** | `/strategies` | Catalog (template strategies), Instances (created strategy instances), Editor (create/edit with params) |
| **ChartingHub** | `/charting` | Candles (interactive chart + tick table with timeframe/range selector), Indicators (computation + overlay management) |
| **IntegrationsPage** | `/integrations` | 3 tabs: Brokers (connection status), Providers & Symbols (CRUD), Credentials (CRUD + rotation). Consolidated from separate CredentialManagement, DataSources, Brokers, Symbols pages |
| **AccountsPage** | `/accounts` | Trading account CRUD with broker selection and prop firm profile linking |
| **PropFirmPage** | `/propfirm` | Prop firm profile management (FTMO, TopStep, E8, TFT) with status display |
| **SimulatePage** | `/simulate` | 7 sub-tabs: generate synthetic data, calibrate HMM, validate data, calibrate regime, generate ticks, inject signal, validate regime. Metric cards replaced raw JSON output |
| **CalibratePage** | `/calibrate` | Brier score decomposition, bin stats table, calibration audit runner |
| **AttributionPage** | `/attribution` | PnL attribution by side/price/edge with Wilson confidence intervals |
| **SettingsPage** | `/settings` | 4 tabs: Trading (risk params + general), Webhooks, Notifications, LLM |
| **AdminPage** | `/admin` | 9-tab admin panel: System Health, Users, Audit Log, Error Logs, Email Test, Seed Data, ML Models, Reconciliation, Data Quality. Runtime metrics card added |
| **EmergencyPage** | `/emergency` | Standalone no-auth mobile kill-switch page |
| **LoginPage** | `/` (unauthenticated) | JWT login with loading state, password visibility toggle, validation, forgot password link, register link |
| **RegisterPage** | `/register` | New user registration |
| **ForgotPasswordPage** | `/forgot-password` | Password reset email request |
| **ResetPasswordPage** | `/reset-password` | Password reset with token |
| **TwoFAPage** | `/2fa` | TOTP 2FA setup and verification |

## Shared Components

### Layout (`src/components/layout/`)
- `Sidebar` — 3-group/13-item collapsible navigation with icons
- `PageHeader` — title + subtitle + badge + actions slot
- `PageSection` — card wrapper with error/warning variants
- `ErrorBanner` — standardized error display with retry/dismiss
- `SkeletonRow` / `PageSkeleton` — animated loading states
- `DynamicBreadcrumb` — auto-generated navigation breadcrumbs

### Charts (`src/charts/`)
- `EquityCurveChart` — equity line with drawdown area and trade markers
- `LiveMonitorChart` — real-time candlestick with indicators, fullscreen, export
- `CandlesChart` — static candlestick with OHLCV crosshair tooltip
- `CVDChart` — cumulative volume delta with divergence markers
- `DailyReturnsChart` — daily return distribution
- `MonteCarloChart` — percentile bands with path simulation
- `CalendarHeatmap` — monthly returns visualization
- `YearlySummaryTable` — yearly performance breakdown
- `CrosshairTooltip` — shared chart tooltip utility

### Backtest (`src/components/backtest/`)
- `OverviewTab` — regime stats, warnings card
- `TradesTab` — filterable by month, paginated trade list
- `OptimizationTab` — optimization footprint display
- `ComparisonTab` — life vs backtest comparison
- `MatrixResultsPanel` — streaming results table with sort/filter + "View" detail links
- `MatrixProgressBar` — real-time matrix progress
- `CancelButton` — cancel in-flight matrix runs
- `ResourceGauges` — heap/CPU utilization gauges
- `MonteCarloSummaryCard` / `MonteCarloHistograms` / `MonteCarloContextCard` — MC result components

### Deploy (`src/components/deploy/`)
- `PromoteToLiveWizard` — 3-step gated deploy: Quality Gates → Pre-Flight → Deploy (account selector with balance display, capital allocation slider)

### Hooks (`src/hooks/`)
- `useLiveRiskData` — shared WebSocket + REST risk data with fallback
- `useAdaptivePolling` — visibility-aware, market-hours-gated exponential backoff
- `useWebSocket` — WebSocket connection with per-channel subscription
- `useChart` / `useChartUpdate` / `useChartKeyboard` / `useCrosshair` / `useCandleAggregation` — chart lifecycle hooks
- `useIndicator` / `useIndicatorRenderer` — indicator computation + rendering
- `useEmergencyControl` — emergency stop/resume with 2FA
- `useMatrixStream` — real-time matrix results streaming
- `useParameterSensitivity` — parameter sensitivity heatmap data

### Stores (`src/stores/`)
- `authStore` / `wsStore` / `tradeStore` / `indicatorStore` / `matrixStore` / `timeframeStore` / `cacheStore`

## API Layer (`src/api/`)
- `client.ts` — typed API client with auth headers + 401 refresh + request tracking (~50 endpoints)
- `optimize.ts` — optimization endpoints (submit, status, results)

## Development

```bash
cd web
npm install                      # Install dependencies (requires Node 20+)
npm run dev                      # Dev server (:5173, proxied to :8080)
npm run build                    # Production build → dist/ (Vite + Tailwind)
npx tsc --noEmit                 # TypeScript type check
npx vitest --run                 # Run 217 unit/component tests
npx playwright test              # Run 49 E2E tests (requires dev server + API mocking)
npx eslint .                     # Lint
```

## Configuration

- `vite.config.ts` — Vite with `@tailwindcss/vite` + API proxy
- `components.json` — shadcn/ui configuration
- `tsconfig.json` — TypeScript strict mode
- `playwright.config.cjs` — Chromium headless E2E tests (port 5174)
