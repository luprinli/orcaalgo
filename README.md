# OrcaAlgo — Polyglot Algorithmic Prop Trading System

**Version**: 1.1.0 · **Auth**: JWT enforced + WS origin validated · **Prices**: types.Price (int64×100000) · **Preflight**: 12 checks · **Guardrails**: pre-commit hook + change audit + env guard

A high-performance algorithmic trading platform purpose-built for **prop firm challenge compliance** (FTMO, TopStep, E8, TFT). Uses a multi-language architecture: **Go** for orchestration and execution, **Python** for strategy IR and canonical mathematics, and **React + TypeScript** for real-time dashboards.

## Stack Constitution

| Component | Language | Role |
|-----------|----------|------|
| Strategy IR, Math, Calibration | **Python 3.11+** | Pydantic v2 domain models, GKR strategy IR, Kelly/Brier/Platt/Wilson/EWMA, calibration audit, PnL attribution, pre-flight, HMM training, data quality validation |
| API, Broker, Ingest, Scheduler | **Go 1.25** | HTTP API (Gin), broker adapters (Alpaca/Paper/IBKR), WebSocket market data ingestion (Polygon.io → ring buffer), WebSocket hub, backtest engine (event-driven + walk-forward), risk management (RiskPipeline + CapitalGate + PropFirmGate + KillSwitch), Monte Carlo bootstrapping, DB repository, LLM integration, per-account strategy isolation, capability-based broker routing |
| Web Dashboard | **React 18 + TypeScript 5 + Tailwind CSS 4 + shadcn/ui** | SPA with lightweight-charts, WebSocket live feed, MonitorPage (Dashboard + Risk + Signals), BacktestHub (Runner + History + Detail + Optimize), StrategyHub (Catalog + Instances + Editor), ChartingHub (Candles + Indicators), IntegrationsPage (Brokers + Credentials + Symbols), backtest matrix runner with gate status, walk-forward analysis, trade analytics, parameter sensitivity, Monte Carlo charts, broker/account/symbol/strategy management, emergency mobile page |
| Time-Series Storage | **PostgreSQL + TimescaleDB** | Hypertables, BIGINT fixed-point price storage, append-only audit logging |

Trading-optimized dark theme (deep navy #090d14) with high-density layouts, tabular numerics, and shadcn/ui components (Button, Card, Dialog, Tabs, Table, Badge, Input, Select, Label, Skeleton, Tooltip, Textarea, AlertDialog, Accordion, Sheet, Popover, Radio Group, Scroll Area, Separator, Switch, Toggle Group)

## Quick Start

```bash
# Clone and install Python package
git clone <repo-url> && cd Orca_algo
pip install -e ".[dev]"

# Start infrastructure
docker compose up -d postgres redis

# Validate GKR strategy configs
orca validate configs/strategies/intraday_mr.gkr.yaml

# Validate data quality
orca data-validate --universe

# Start server (API :8080, Metrics :9090, WS /ws)
go run ./cmd/orca-server

# Start frontend (:5173, proxied to :8080)
cd web && npm install && npm run dev

# Paper trading mode (no broker keys needed)
PAPER_TRADING=true go run ./cmd/orca-server

# Pre-flight checklist (run before any live deployment)
orca preflight

# Calibration audit (run quarterly)
orca calibrate --since 90d

# PnL attribution
orca attribute --since 90d
```

## Key Features

### Frontend Dashboard — 3 Groups, 13 Navigation Items
- **Trading Desk**: Dashboard (Monitor), Execution, Backtesting (Runner + History + Detail + Optimize), Strategies (Catalog + Instances + Editor)
- **Analysis**: Charts (Candles + Indicators), Calibration, Attribution, Simulation
- **Settings**: System, Integrations, Accounts, Admin, Emergency

### Multi-User & Multi-Account
- User Management — Registration, login (JWT + 2FA), password reset via email, email verification
- Admin Panel — 9-tab admin: Health, Users, Audit, Errors, Email, Seed, ML Models, Reconciliation, Data Quality
- Multi-Account Broker — Multiple accounts per user (FTMO $100k + TopStep $150k simultaneously)
- Per-Account Capital Pools — Independent daily loss, drawdown, and correlation tracking per account via `MultiAccountCapitalPool`
- **Per-Account Strategy Isolation** — Factory-created isolated strategy instances with independent state (indicator buffers, rolling windows, open positions)
- Per-User Data Isolation — All resources scoped by `user_id` with row-level filtering

### Notification System
- Multi-Channel — Telegram (multi-chat), Email (SMTP), Push (WebSocket)
- Per-User Settings — Enable/disable channels, configure event filters per channel
- Test Endpoint — Send test notifications to verify configuration
- Built-in Templates — Password reset, email verification, trading event notifications

### Prop Firm Compliance (Multi-Firm — FTMO, TopStep, E8, TFT)
- Vendor-agnostic `PropfirmEnforcer` with multi-profile support
- 5% daily loss limit with automatic halt
- 10% maximum drawdown (HWM trailing) — configurable per firm
- 30% consistency rule with size multiplier reduction
- Regime-aware position sizing via shared `PositionSizer` (1.0× / 0.85× / 0.75× / 0.50×)
- VIX-based position scaling (0.50× at VIX>35, 0.75× at VIX>28, 0.90× at VIX>20)
- Sentiment-based position scaling via Fear & Greed Index (0.50× at extremes, 0.75× at edges)
- Kelly fractional sizing (k=0.25) with per-trade cap (2%) and total exposure cap (30%)
- Daily reset scheduler with notification alerts

### Per-Strategy Light Optimization in Matrix Backtests
- Bounded parameter sweep per unique strategy on representative symbol subset
- Train/test split (67%/33%) with OOS validation fallback
- Composite scoring (Sharpe 0.5 + MinDD 0.3 + ProfitFactor 0.2)
- In-memory SHA-256-keyed optimization cache with TTL expiry
- Early stopping (5-combo plateau patience)
- Prop-firm rule propagation (0.1× score multiplier on breaches)
- Optimized params applied to all combos of each strategy in the matrix

### Real-Time Regime Detection
- 4-state Hidden Markov Model (Calm / Trending / High Vol / Crisis)
- Python `hmmlearn` training pipeline → Go runtime HMM forward algorithm
- VIX-modulated emission parameters (SD widened at VIX>25)
- Per-tick confidence scoring
- Regime-gated strategy selection

### Strategy Framework
- GKR Strategy IR — Declarative YAML with versioning, hashing, temporal validation
- **Intraday Mean Reversion** — Z-score based with ATR-normalized exit
- **Grid Trading** — Regime-gated with Fear & Greed sentiment overlay
- **Trend Following** — EMA crossover with ATR trailing stop and ADX confirmation
- **Opening Range Breakout** — Range detection with volatility scaling and time-of-day exit
- **Session Scalping** — 9:30-11:00 ET window, volume-confirmed breakout
- **Pairs Trading** — Cointegrated pairs with Kalman filter hedge ratio
- **Volatility Harvesting** — VIX threshold-based vol premium harvesting
- 11 total strategies registered in factory table

### Risk Management — Unified RiskPipeline
- **`RiskPipeline`** — Canonical signal audit pipeline shared by backtest `Engine` and live `LiveEngine`
- **`CapitalGate` interface** — Both `CapitalPoolSim` (backtest) and `CapitalPoolManager` (live) implement identical capital authorization contract
- **`PropFirmGate` interface** — Both `PropfirmEnforcer` (backtest) and `propfirm.Manager` (live) implement identical prop-firm check contract
- **`SignalGate` interface** — Shared sizing, exposure, and signal validation
- **`BaseCapitalPool` (`propfirm.PoolState`)** — Shared balance tracking, drawdown computation, position counting — extracted from both pool implementations
- Kill-switch with `isLocked` + `killSwitchReady` re-entrancy guard + multi-account iteration + prop-firm violation propagation (`MarkAllViolated`)
- Adversarial detection (reject spikes >3/5min, unusual size/symbol, after-hours lockout)
- Dynamic rate limiter with circuit breaker (10 orders/sec per symbol)
- Volatility halt based on z-score of rolling returns (halts at |z| > 3.0)
- Max leverage control (gross exposure ≤ 5× equity)
- Per-symbol gross exposure cap (25% of equity)
- Encrypted credential vault (env vars dev, AES-256-GCM production)

### Backtesting
- Event-driven candle backtest engine with deterministic seed support
- TCA (Transaction Cost Analysis): per-trade slippage vs mid-price, slippage vs last-trade, adverse selection rate
- Walk-forward rolling window framework with optimization and IVS robustness checking
- Multi-Metric Gate auto-application (Default/Lenient/Strict profiles) with pass/fail display
- Matrix backtest runner: strategies × symbols × timeframes with streaming results
- Per-strategy light optimization pre-processing (composite scoring + train/test split + early stopping + caching + prop-firm propagation)
- Monte Carlo bootstrapping from actual trade PnL distributions
- MAE/MFE computation per trade (stops optimization, trade quality analysis)
- Warm-up period with indicator state building (prevents garbage signals)
- Volume-dependent slippage modeling
- Kelly fraction applied in both backtest and live paths (k=0.25)
- Sharpe ratio annualized from daily returns (consistent across intraday/daily bars)
- Promote-to-Live 3-step wizard (Quality Gates → Pre-Flight → Deploy)
- FTMO/PropFirm rule enforcement (daily loss, drawdown, consistency)

### Observability & Audit
- Structured Audit Logging — Immutable audit trail for all critical actions
- Error Classification — Structured errors with category, severity, retryability
- Health Monitoring — 60s background checks (DB, broker accounts)
- WebSocket Hub — Real-time broadcasts: risk (with VIX/sentiment), fills, account status, regime, PnL history, matrix progress, signal lifecycle
- Telegram + Email Alerts — Kill-switch, regime changes, drawdown warnings
- Prometheus Metrics — Ticks, orders, latency, regime, kill-switch, PnL, WS connections
- Data Quality Validation — `orca data-validate` checks gaps, outliers, volume sanity, completeness

## API Reference

### Core Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/api/v1/auth/login` | Login (JWT + optional 2FA) |
| `POST` | `/api/v1/auth/register` | Register new user |
| `POST` | `/api/v1/auth/forgot-password` | Request password reset email |
| `POST` | `/api/v1/auth/reset-password` | Reset password with token |
| `GET` | `/api/v1/system/health` | System health (public) |
| `GET` | `/api/v1/accounts` | List broker accounts |
| `POST` | `/api/v1/orders` | Place order (optional `account_id`) |
| `GET` | `/api/v1/positions` | Get positions |
| `GET` | `/api/v1/risk/status` | Live risk snapshot |
| `POST` | `/api/v1/emergency/stop` | Trigger kill-switch (2FA) |
| `GET` | `/api/v1/strategies` | List strategies (CRUD: POST, PUT, DELETE) |
| `POST` | `/api/v1/strategies/validate` | Validate strategy params |
| `POST` | `/api/v1/backtests` | Submit single backtest |
| `POST` | `/api/v1/backtests/matrix` | Submit matrix backtest (streaming) |
| `POST` | `/api/v1/optimizations` | Submit per-strategy light optimization |
| `GET` | `/api/v1/optimizations/:id` | Optimization status (polling) |
| `GET` | `/api/v1/optimizations/:id/results` | Optimization results |
| `GET` | `/api/v1/backtests/:id/metrics` | Backtest metrics |
| `GET` | `/api/v1/backtests/:id/equity` | Backtest equity curve |
| `GET` | `/api/v1/backtests/:id/trades` | Backtest trades |
| `GET` | `/api/v1/backtests/:id/monthly-returns` | Monthly returns |
| `GET` | `/api/v1/backtests/:id/regime-stats` | Regime-stratified stats |
| `GET` | `/api/v1/backtests/:id/optimization` | Optimization footprint |
| `GET` | `/api/v1/backtests/:id/walk-forward` | Walk-forward results |
| `GET` | `/api/v1/backtests/:id/live-comparison` | Live vs backtest comparison |
| `GET` | `/api/v1/candles` | Market data candles |
| `GET` | `/api/v1/propfirm/profiles` | Prop firm profiles |
| `GET` | `/api/v1/propfirm/status` | Live prop firm compliance |
| `GET` | `/api/v1/live/metrics` | Live trading metrics |
| `GET` | `/api/v1/live/equity` | Live equity curve |
| `GET` | `/api/v1/universe/current` | Current symbol universe |
| `GET` | `/api/v1/settings` | User settings management |
| `GET` | `/api/v1/admin/users` | User management (admin) |
| `POST` | `/api/v1/admin/seed` | Seed database (admin) |
| `GET` | `/ws` | WebSocket connection |

### WebSocket Events

| Channel | Interval | Payload |
|---------|----------|---------|
| `risk` | 5s | `{regime, confidence, vix, sentiment, halted, daily_pnl_pct, balance, equity, consistency_multiplier}` |
| `pnl_history` | 5s | `[{time, value}]` — daily PnL history |
| `account_status` | 5s | `{account_id, balance, equity, daily_pnl, high_water_mark}` |
| `backtest_progress` | per-combo | `{batch_id, seq, combo, completed, total, failed, percent, current_task}` — matrix streaming |
| `signal_lifecycle` | event | Signal lifecycle events for live trading |
| `notification` | event | `{type, level, title, message, timestamp}` |
| `fill` | event | `{account_id, broker_order_id, symbol, side, filled_qty, avg_fill_price}` |

## Risk Management Layers

| Layer | Location | Mechanism |
|-------|----------|-----------|
| **RiskPipeline** | `internal/risk/pipeline.go` | Canonical signal audit pipeline: sizing → exposure → capital gate → prop-firm |
| **CapitalGate** | `internal/risk/interfaces.go` | Capital authorization across backtest (`CapitalPoolSim`) and live (`CapitalPoolManager`) |
| **PropFirmGate** | `internal/risk/interfaces.go` | Prop-firm enforcement across backtest (`PropfirmEnforcer`) and live (`propfirm.Manager`) |
| **BaseCapitalPool** | `internal/propfirm/pool_base.go` | Shared balance/DD/DailyPnL/Halted fields + shared helpers |
| **Per-Account Isolation** | `internal/engine/live_engine.go` | Factory-created instances, independent per-account registries |
| **PropFirm Rules** | `internal/propfirm/rules.go` + `internal/backtest/propfirm_enforcer.go` | 5% daily loss, 10% max DD (HWM), consistency, regime multipliers |
| **HMM Regime** | `internal/risk/hmm.go` | 4-state forward algorithm, VIX-modulated, Python-trained |
| **VIX Scaling** | `internal/risk/position_sizer.go` | VIX>35→0.50×, VIX>28→0.75×, VIX>20→0.90× |
| **Volatility Halt** | `internal/risk/trading_controls.go` | Z-score halt at \|z\|>3.0 |
| **Order Rate Limiter** | `internal/risk/trading_controls.go` | 10 orders/sec per symbol sliding window |
| **Exposure Tracker** | `internal/risk/trading_controls.go` | Max leverage 5×, per-symbol 25% |
| **Adversarial Guard** | `internal/risk/adversary.go` | Reject spikes (>3/5min), unusual size/symbol, after-hours |
| **Kill Switch** | `internal/risk/kill_switch.go` | Manual (2FA) + auto, multi-account iteration, pool propagation |

## Environment Variables

| Variable | Purpose | Required |
|----------|---------|----------|
| `ORCA_JWT_SECRET` | JWT signing key | Yes |
| `ORCA_ADMIN_PASSWORD` | Admin user password (auto-migrated to DB) | Yes |
| `ORCA_DB_HOST` / `ORCA_DB_PORT` / etc. | PostgreSQL connection | No |
| `PAPER_TRADING` | Use paper trading adapter | No |
| `ALPACA_API_KEY` / `ALPACA_API_SECRET` | Alpaca broker credentials | No |
| `TELEGRAM_BOT_TOKEN` / `TELEGRAM_CHAT_ID` | Telegram alerts | No |
| `SMTP_HOST` / `SMTP_PORT` / `SMTP_USER` / `SMTP_PASS` / `SMTP_FROM` | Email service | No |
| `ORCA_DATA_MODE` | `stooq` (dev) or `live` (production) | No |
| `ORCA_WS_ORIGINS` | Allowed WebSocket origins | No |
| `ORCA_LIGHT_OPT_BUDGET` | Max combos for light optimizer (default: 24) | No |
| `ORCA_LIGHT_OPT_WINDOW_MONTHS` | Window months for light optimizer (default: 3) | No |

## CLI Commands

```bash
orca validate <path>              # Validate .gkr.yaml strategy configs
orca calibrate --since 90d        # Run calibration audit (quarterly)
orca preflight [--strict]         # Pre-deployment checklist (12 checks)
orca attribute --since 90d        # PnL attribution with Wilson CI
orca simulate calibrate --help    # Simulation subcommands
orca data-validate [--universe]   # Data quality validation
orca hmm-train --since 3650d      # Train HMM on historical data
```

## CI/CD Pipeline

| Job | Scope | Gates |
|-----|-------|-------|
| `python` | `orca/`, `tests/` | ruff, mypy, pytest (coverage ≥ 80%) |
| `backend` | `internal/`, `cmd/` | golangci-lint, go vet, test (race + coverage ≥ 60%), E2E |
| `frontend` | `web/` | ESLint, tsc, vite build, vitest (217 tests), playwright (49 e2e tests) |
| `gkr-validate` | `configs/strategies/` | All `.gkr.yaml` validation |
| `anti-pattern-scan` | All | 10 hard prohibition enforcement |
| `security` | All | Gitleaks + govulncheck |
| `guardian` | Python + Go | 20 critical path regression smoke tests |
| `mutation-test` | `orca/sizing/`, `orca/math/` | Mutation testing (main only) |

## Documentation

- [Frontend Audit Report](docs/frontend-backtest-audit-2026-07-24.md) — Full frontend architecture audit: page catalog, backend mapping, duplication analysis, navigation restructuring
- [Backtest Hub UI Audit](docs/backtest-hub-ui-audit-2026-07-25.md) — BacktestHub component-by-component audit, UX workflow analysis, prioritized remediation plan (P0–P3)
- [Capital Pool Live Wiring Plan](docs/capital-pool-live-wiring-plan-2026-07-25.md) — 6-wave architecture plan: RiskPipeline interfaces, BaseCapitalPool extraction, Engine/LiveEngine/API server wiring, per-account strategy isolation
- [Per-Strategy Optimization Implementation](docs/per-strategy-optimization-implementation-2026-07-25.md) — Light optimizer wiring, API handler fix, IVS bug fix, scoring unification, regression protection
- [Tech Stack Constitution](AGENTS.md) — Language boundaries, 16 hard prohibitions, cross-language integration rules
- [Strategy Configs](configs/strategies/) — GKR IR strategy definitions

## License

Proprietary. All rights reserved.

---

*OrcaAlgo v1.1.0 — Multi-user, multi-account, multi-firm, deterministic backtest-live consistent prop trading platform with unified RiskPipeline*
