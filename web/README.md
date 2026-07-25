# `web/` — React 18 + TypeScript 5 Trading Dashboard

Single Page Application with Tailwind CSS 4, shadcn/ui components, and TradingView Lightweight Charts. Provides real-time trading monitoring, backtest execution, strategy management, walk-forward analysis, and broker/symbol configuration. Uses WebSocket for live data and a shared `useLiveRiskData` hook.

[↑ Back to Root README](../README.md)

## Tech Stack

- **React 18** + **TypeScript 5**
- **Vite 5** — build tool and dev server with `@tailwindcss/vite` plugin
- **Tailwind CSS 4** — utility-first CSS with trading-optimized dark theme
- **shadcn/ui** — 13 accessible components (Button, Card, Dialog, Tabs, Table, Badge, Input, Select, Label, Skeleton, Tooltip, Textarea, AlertDialog)
- **lightweight-charts 5** — financial charting (candlestick, equity curve, CVD, volume profile, Monte Carlo)
- **Zustand 5** — state management (auth, cache, trades, indicators, WebSocket data)
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

## Pages

| Page | Path | Description |
|------|------|-------------|
| **CommandCenter** | `/` | Merged Dashboard + LiveTrading + Risk. 4 tabs: Overview (9 KPIs + equity curve + risk bars), Positions, Orders, Risk (emergency stop/resume + regime history) |
| **ExecutionPage** | `/execution` | Order placement (market/limit/stop) with toast notifications |
| **BacktestPage** | `/backtest` | Single or matrix backtest config with `?strategy=` query param support |
| **BacktestHistory** | `/backtest/history` | Backtest run list with compare mode and multi-select equity overlay |
| **BacktestDetail** | `/backtest/history/:id` | Full analytics: Overview, Trades, Optimization, Comparison, Walk-Forward, Analytics tabs |
| **StrategiesPage** | `/strategies` | GKR strategy catalog with validate/clone/delete actions |
| **StrategyEditor** | `/strategies/:id/edit` | Strategy configuration editor with Quick Backtest button |
| **OptimizationPanel** | `/optimize` | Walk-forward optimization with multi-metric gate and parameter sensitivity heatmap |
| **AccountsPage** | `/accounts` | Multi-account broker management |
| **PropFirmPage** | `/propfirm` | Prop firm profile management (FTMO, TopStep, E8, TFT) |
| **MarketDataPage** | `/market-data` | Live market data with candles, CVD, divergence detection, and "Live Ticks Only" toggle |
| **IndicatorsPage** | `/indicators` | Technical indicator configuration with IndicatorConfigModal |
| **CalibratePage** | `/calibrate` | Calibration audit pipeline |
| **AttributionPage** | `/attribution` | PnL attribution with Wilson CI slices |
| **SimulatePage** | `/simulate` | Synthetic data generation, calibration, and validation |
| **AdminPage** | `/admin` | 6-tab admin panel: System Health, Users, Audit Log, Error Logs, Email Test, Seed Data |
| **UniversePage** | `/admin/universe` | Symbol universe management |
| **SymbolAdminPage** | `/symbols` | Symbol/Provider/Credential CRUD management |
| **SettingsPage** | `/settings` | User settings with risk limits and notification config |
| **EmergencyPage** | `/emergency` | Standalone no-auth mobile kill-switch page (< 50 KB) |
| **LoginPage** | `/` (unauthenticated) | JWT login with loading state, password visibility toggle, validation, forgot password link, register link |
| **RegisterPage** | `/register` | New user registration |
| **ForgotPasswordPage** | `/forgot-password` | Password reset email request |
| **ResetPasswordPage** | `/reset-password` | Password reset with token |
| **TwoFAPage** | `/2fa` | TOTP 2FA setup and verification |
| **LLMSettings** | `/llm` | LLM provider configuration (OpenAI, Anthropic, Ollama) |
| **WebhookConfig** | `/webhooks` | Webhook URL + event subscription config with test-fire |
| **CredentialManagement** | `/credentials` | API credential management with rotation |
| **BrokerManagement** | `/brokers` | Broker connection status and supported adapters |
| **NotificationSettings** | `/notifications` | Email/Push/Telegram notification preferences |
| **DataSources** | `/data-sources` | Market data source selection |

## Shared Components

### Layout (`src/components/layout/`)
- `PageHeader` — title + subtitle + badge (ok/err/warn) + actions slot
- `MetricGrid` — responsive KPI grid (3/4/5 columns)
- `PageSection` — card wrapper with title and error/warning variants
- `ErrorBanner` — standardized error display with retry/dismiss
- `SkeletonRow` / `PageSkeleton` — animated loading states
- `Sidebar` — extracted from App.tsx with collapsible nav groups

### Charts (`src/components/charts/`)
- `EquityCurveChart` — equity line with benchmark overlay and trade markers
- `LiveMonitorChart` — real-time candlestick with indicators, fullscreen, export, timeframe chips
- `CandlesChart` — static candlestick with OHLCV crosshair tooltip
- `CVDChart` — cumulative volume delta with divergence markers and crosshair
- `DailyReturnsChart` — daily return distribution with crosshair
- `MonteCarloChart` — percentile bands with Web Worker computation
- `CalendarHeatmap` — monthly returns visualization
- `YearlySummaryTable` — yearly performance breakdown
- `CrosshairTooltip` / `MarkerManager` — shared chart utilities

### Backtest (`src/components/backtest/`)
- `BacktestConfigBar` — backtest configuration form
- `MatrixProgressBar` / `MatrixResultsPanel` — matrix backtest streaming
- `OverviewTab` / `TradesTab` / `OptimizationTab` / `ComparisonTab` — BacktestDetail tabs
- `AnalyticsTab` — PnL distribution, MAE/MFE, win rate by day/hour, rolling Sharpe
- `WalkForwardTab` — per-window IS/OOS Sharpe with compliance badges
- `ParameterSensitivityHeatmap` — plotly.js 2D parameter interaction heatmap
- `OptimizationConfigForm` — shared optimization config for BacktestPage + OptimizationPanel
- `MonteCarloCards` / `MonteCarloContextCard` / `MonteCarloSummaryCard` — MC result components

### UI (`src/components/ui/`) — shadcn
- `button`, `card`, `dialog`, `tabs`, `table`, `badge`, `input`, `select`, `label`, `skeleton`, `tooltip`, `textarea`, `alert-dialog`

### Hooks (`src/hooks/`)
- `useLiveRiskData` — shared WebSocket + REST risk data with fallback
- `useAdaptivePolling` — visibility-aware, market-hours-gated exponential backoff
- `useChart` / `useChartUpdate` / `useChartCrosshair` / `useChartKeyboard` — chart lifecycle hooks

### Stores (`src/stores/`)
- `cacheStore` — stale-while-revalidate cache for strategies/symbols/accounts
- `authStore` / `wsStore` / `tradeStore` / `indicatorStore` / `toastStore` / `timeframeStore`

## API Layer (`src/api/`)
- `client.ts` — typed API client with auth headers + 401 refresh + request tracking
- `optimize.ts` — optimization endpoints (refactored to use shared `request()` with auth)
- All SymbolAdminPage raw `fetch()` calls migrated to typed client

## WebSocket Integration

```tsx
import { useLiveRiskData } from '../hooks/useLiveRiskData'

const { riskData, connected, isHalted, refetch } = useLiveRiskData()
// riskData: { halted, balance, equity, daily_pnl_pct, drawdown_used, regime, vix, sentiment, ... }
```

Available channels: `risk`, `ticks`, `cvd`, `divergence`, `pnl_history`, `account_status`

## Development

```bash
cd web
npm install                      # Install dependencies (requires Node 20+)
npm run dev                      # Dev server (:5173, proxied to :8080)
npm run build                    # Production build → dist/ (Vite + Tailwind)
npx tsc --noEmit                 # TypeScript type check
npx vitest --run                 # Run 201 unit/component tests
npx playwright test              # Run E2E tests (requires :5173 + :8080)
npx eslint .                     # Lint
```

## Configuration

- `vite.config.ts` — Vite with `@tailwindcss/vite` + API proxy
- `components.json` — shadcn/ui configuration
- `tsconfig.json` — TypeScript strict mode
- `playwright.config.cjs` — Chromium headless E2E tests
