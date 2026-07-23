# `web/` — React 18 + TypeScript 5 Trading Dashboard

Single Page Application providing real-time trading monitoring, backtest execution, strategy management, and broker/symbol configuration. Uses WebSocket for live data (with VIX/sentiment/regime) and lightweight-charts for visualization.

[↑ Back to Root README](../README.md)

## Tech Stack

- **React 18** + **TypeScript 5**
- **Vite** — build tool and dev server
- **lightweight-charts** — financial charting (CVD, equity curve, volume profile, daily PnL)
- **WebSocket** — real-time data via custom `WebSocketProvider`

## Pages

| Page | Path | Description |
|------|------|-------------|
| **LiveDashboard** | `/live` | Live risk gauge with VIX/sentiment, equity curve, daily PnL, CVD chart, return distribution |
| **Dashboard** | `/` | Regime gauge, risk status card, consistency bar, emergency stop |
| **UnifiedDashboard** | `/dashboard` | Integrated multi-panel dashboard |
| **BacktestRunner** | `/backtest` | Config form with 7 strategies + gate profile selector, results summary with warning/gate columns, equity curve, regime performance |
| **StrategyDetail** | `/strategy/:id` | Individual strategy configuration (7 types: mean_reversion, breakout, trend, grid, scalp, stat_arb, vol_arb) |
| **StrategyComparison** | `/strategies` | Sortable comparison grid, multi-select, regime exposure panels |
| **BrokerManagement** | `/brokers` | Broker adapter management (Alpaca, Paper, IBKR) |
| **SymbolManagement** | `/symbols` | Symbol CRUD and feed assignment |
| **Optimization** | `/optimization` | Walk-forward optimization results with Compliance pass/fail |
| **PropFirmProfiles** | `/propfirm` | Vendor-agnostic prop firm profile management (FTMO, TopStep, E8, TFT) |
| **Settings** | `/settings` | Risk limits, consistency thresholds, notification config |
| **Login** | `/login` | JWT authentication |
| **TwoFASetup** | `/2fa` | TOTP-based 2FA setup |
| **LLMSettings** | `/llm` | LLM integration configuration |
| **WebhookConfig** | `/webhooks` | TradingView/ChartInk webhook configuration |
| **CredentialManagement** | `/credentials` | API credential management and rotation |
| **DataSourceManagement** | `/data-sources` | Market data source configuration |

## Components

### Risk Components (`src/components/risk/`)

| Component | Props | Purpose |
|-----------|-------|---------|
| `RiskStatusCard` | `dailyLossUsed, drawdownUsed, dailyLimitPct, maxDDPct, balance, equity` | Live risk metrics with color coding (ok/warn/crit) |
| `EmergencyStop` | `halted, onTrigger, onResume` | Kill-switch trigger/resume with 2FA token input |
| `RegimeGauge` | `regime, confidence, vix?, sentiment?` | HMM regime + VIX gauge (green/blue/yellow/red) + Fear & Greed index with label |
| `ConsistencyBar` | `dailyPnLPct, thresholdPct` | Consistency rule progress bar with outlier detection |

### Chart Components (`src/components/charts/`)

| Component | Props | Library |
|-----------|-------|---------|
| `CVDChart` | `data, height` | Cumulative Volume Delta with divergence markers |
| `DailyPnLChart` | `data, height, threshold` | Daily P&L bar chart with loss limit threshold |
| `EquityCurve` | `data, height` | Equity curve with drawdown shading |
| `VolumeProfile` | `data, height` | Price-level volume distribution with POC/VAH/VAL |

### Backtest Components (`src/components/backtest/`)

| Component | Props | Purpose |
|-----------|-------|---------|
| `BacktestForm` | `onSubmit, loading` | Strategy (7 types), symbol, date range, capital, gate profile config. Matrix or single mode |
| `BacktestResults` | `results, totalCombos` | Sortable/filterable table with columns: Symbol, Strategy, TF, Return, Sharpe, Sortino, MaxDD, Win%, PF, AvgTrade, Trades, **Warnings**, **Gate Status** (✓/✗/—), **ASR** (Adverse Selection Rate). Filters: hide 0-trade, only gate-passing, only with data |
| `BacktestMetrics` | `result` | Sharpe, Sortino, MaxDD, Win Rate, Profit Factor display |
| `RegimePerformanceTable` | `regimeStats` | Per-regime breakdown (trades, win rate, return, DD) |
| `OptimizationResults` | `windows, loading` | Walk-forward window results with Compliance column |

### Common Components (`src/common/`)

| File | Purpose |
|------|---------|
| `WebSocketProvider.tsx` | React context providing `useWS()` hook with `subscribe(channel, callback)` |
| `ErrorBoundary.tsx` | Error boundary with fallback UI |
| `StatusBadge.tsx` | Color-coded status indicator (ok/warn/crit) |
| `hooks.ts` | Shared custom React hooks |

## WebSocket Integration

The `WebSocketProvider` connects to `ws://localhost:8080/ws` and exposes:

```tsx
const { connected, subscribe } = useWS()

useEffect(() => {
  const unsub = subscribe('risk', (data) => {
    // data: { regime, confidence, vix, sentiment, halted, daily_pnl_pct, balance, equity, ... }
  })
  return unsub
}, [subscribe])
```

Available channels: `risk` (with VIX/sentiment), `cvd`, `orderbook`, `pnl_history`, `ticks`, `positions`, `orders`, `strategy_metrics`

## Development

```bash
cd web
npm install               # Install dependencies
npm run dev               # Dev server (:5173, proxied to :8080)
npm run build             # Production build → dist/
npm run lint              # ESLint
npx tsc --noEmit          # TypeScript type check
```

## Configuration

- `vite.config.ts` — Vite with API proxy to `localhost:8080`
- `tsconfig.json` — TypeScript strict mode
- `eslint.config.js` — ESLint with React + TypeScript rules
