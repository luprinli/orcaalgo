# OrcaAlgo — Agent Instructions & Tech Stack Compliance

## Stack Constitution (Immutable)

The approved technology stack:

| Component | Language | Role |
|-----------|----------|------|
| Strategy IR, Math, Calibration | **Python** (3.11+) | Domain models, GKR strategy IR, canonical math (Kelly, Brier, Platt, Wilson, EWMA), calibration audit, PnL attribution, pre-flight checklist |
| API, Broker, Ingest, Scheduling | **Go** (1.25) | HTTP API (Gin), broker integration (Alpaca/IBKR/Paper), market data ingestion (WebSocket→ring buffer), WebSocket hub, background scheduler, DB repository, RiskPipeline + CapitalGate + PropFirmGate + SignalGate, CapitalPoolSim/CapitalPoolManager, BaseCapitalPool, KillSwitch + multi-account iteration, per-account strategy isolation, light optimizer wiring, Monte Carlo bootstrapping, walk-forward framework |
| Web Dashboard | **React 18 + TypeScript 5 + Tailwind CSS 4 + shadcn/ui** | SPA with lightweight-charts, WebSocket live feed, MonitorPage (4 tabs), BacktestHub (Runner + History + Detail + Optimize), StrategyHub (3 tabs), ChartingHub (Candles + Indicators), IntegrationsPage, Accounts, Admin (9 tabs), Emergency |
| Time-Series Storage | **PostgreSQL + TimescaleDB** | Hypertables (market_ticks, candles, trade_executions), BIGINT fixed-point price storage, compression (7d) + retention (30d) policies |
| Audit Log (consider) | **SQLite** | Append-only WAL mode for lightweight deployments |

## Hard Prohibitions

These are **NEVER** permitted. Violations block PR merge.

| # | Rule | Source |
|---|------|--------|
| 1 | **Do not reimplement canonical math functions in Go.** Kelly, Brier, Platt, Wilson, EWMA exist in `orca/sizing/` and `orca/math/`. Reference them via subprocess or import. | Antipattern #2 |
| 2 | **Do not use IEEE 754 float for order prices.** Use `BIGINT` with scale factor in PostgreSQL, `Decimal` in Python, `fixed.Fixed` in Go (recommended). | Antipattern #5 |
| 3 | **Do not commit strategy configs in legacy YAML format.** All strategies must be `.gkr.yaml` with versioning, hashing, and type validation. | §5.1 |
| 4 | **Do not deploy to production without pre-flight.** `orca preflight` must pass with zero failures before any live deployment. | §9.3 |
| 5 | **Do not skip calibration audits.** Quarterly `orca calibrate` runs are mandatory for all probability-emitting models. | §9.2 |
| 6 | **Do not use full Kelly in production.** Fractional Kelly (k=0.25) with all three attenuators (edge discount, fractional multiplier, hard caps) is mandatory. Applied in both backtest and live paths. | §3.1.3 |
| 7 | **Do not mutate domain models.** All Pydantic models use `ConfigDict(frozen=True, extra="forbid")`. All Go structs use unexported fields with constructor-only initialization. | §2.1.2 |
| 8 | **Do not bypass the kill-switch re-entrancy guard.** `isLocked` + `killSwitchReady` must both be checked before any kill-switch execution. Trigger propagates to `MultiAccountCapitalPool.MarkAllViolated()`. | §4.2.2 |
| 9 | **Do not assume perfect fills at mid-price.** Backtests must model maker fill prices, fill probability, spread crossing, fees, adverse selection, and volume-dependent slippage. | §9.1.3 |
| 10 | **Do not panic/throw for recoverable errors.** Return errors. Only unrecoverable startup failures may terminate. | Antipattern #10 |
| 11 | **Do not use `setData()` for incremental chart updates.** Use `ISeriesApi.update()` for real-time / polling updates. `setData()` is for initial load or full data replacement only. | lightweight-charts docs |
| 12 | **Do not call `fitContent()` on every data update.** `fitContent()` resets the user's scroll/zoom position. Call it only on initial load, timeframe change, or explicit user action. | lightweight-charts docs |
| 13 | **Do not use `applyOptions({ width })` for chart resize.** Use `chart.resize(width, height)`. | lightweight-charts docs |
| 14 | **Do not use `barSpacing` mutation for keyboard zoom.** Use `getVisibleLogicalRange()` + `setVisibleLogicalRange()` on the time scale. | lightweight-charts docs |
| 15 | **Do not leave `requestAnimationFrame` un-cancelled.** All `requestAnimationFrame` calls in chart hooks must be cancelled in the `useEffect` cleanup via `cancelAnimationFrame`. | React best practice |
| 16 | **Do not use `Array.find()` in crosshair handlers.** Crosshair `subscribeCrosshairMove` fires at 60fps. Build `Map<time, value>` lookups in a `useEffect` and use O(1) `.get()` in the handler. | Performance |
| 17 | **Do not bypass the RiskPipeline in backtest or live paths.** `Engine.generateSignal` and `LiveEngine.ProcessTick` both route through `RiskPipeline.ProcessSignal` / `ReconcileFill`. New risk checks must be added to the pipeline, not duplicated in each engine. | Architecture |
| 18 | **Do not share strategy instances across accounts in live trading.** Use `RegisterAccountStrategies(accountID)` to create factory-isolated instances per account. | Per-account isolation |

## Language Boundary Rules

### What goes in Python (`orca/`)
- All Pydantic v2 domain models (frozen, validated)
- All GKR strategy IR representation and validation
- All canonical mathematical functions (Kelly, Brier, Platt, Wilson, EWMA)
- Calibration audit pipeline (quarterly `orca calibrate`)
- PnL attribution engine (`orca attribute`)
- Pre-flight checklist (`orca preflight`)
- Temporal contract validation (look-ahead prevention)
- Deterministic hashing (canonical JSON + SHA-256)
- CLI entry points (Typer)

### What goes in Go (`internal/`)
- HTTP API handlers and middleware (Gin)
- Broker adapters (Alpaca, IBKR, Paper)
- Market data WebSocket ingestion and ring buffer
- WebSocket hub for real-time UI updates
- Database repository (pgx/v5 → PostgreSQL)
- **RiskPipeline** — Canonical signal-audit pipeline (`internal/risk/pipeline.go`)
- **CapitalGate/PropFirmGate/SignalGate** — Shared risk interfaces (`internal/risk/interfaces.go`)
- **CapitalPoolSim** — Backtest capital pool (`internal/backtest/capital_pool_sim.go`)
- **CapitalPoolManager** — Live capital pool per account (`internal/risk/capital_pool.go`)
- **BaseCapitalPool** — Shared balance/DD/DailyPnL fields (`internal/propfirm/pool_base.go`)
- **MultiAccountCapitalPool** — Per-account pool router (`internal/risk/multi_account_capital_pool.go`)
- **PropfirmEnforcer** — Backtest prop-firm enforcement (`internal/backtest/propfirm_enforcer.go`)
- **propfirm.Manager** — Live prop-firm manager (`internal/propfirm/profile.go`)
- Kill-switch (re-entrancy guard, multi-account, pool propagation)
- Risk management (adversarial, volatility halt, exposure tracker, rate limiter)
- Background scheduler, LLM client
- Backtest execution engine (event-driven, walk-forward, Monte Carlo, light optimizer)
- Per-account strategy isolation (`internal/engine/live_engine.go`)
- Monitor/metrics/telemetry

### What goes in React (`web/`)
- **MonitorPage** — Dashboard: Overview (9 KPIs + equity + risk bars), Positions & Orders, Risk (emergency stop/resume + regime), Signals
- **BacktestHub** — Runner (Matrix/Single/Optimize modes), History (compare + correlation matrix), Detail (17 metrics + 5 charts + 4 tabs), Promote-to-Live wizard
- **StrategyHub** — Catalog (template strategies), Instances (created instances), Editor (create/edit)
- **ChartingHub** — Candles (interactive chart + tick table), Indicators (computation + overlay)
- **IntegrationsPage** — Brokers, Providers & Symbols, Credentials
- AccountsPage, SettingsPage (4 tabs), AdminPage (9 tabs), EmergencyPage (no-auth)
- CalibratePage, AttributionPage, SimulatePage (7 sub-tabs)
- Chart components (EquityCurve, DailyReturns, MonteCarlo, CalendarHeatmap, YearlySummary, CVD, Candles, LiveMonitor, CrosshairTooltip)
- Backtest components (OverviewTab, TradesTab, OptimizationTab, ComparisonTab, MatrixResultsPanel, MatrixProgressBar, CancelButton, ResourceGauges)
- Deploy component (PromoteToLiveWizard — 3-step gated deploy)
- Auth pages (Login, Register, 2FA Setup, Forgot Password, Reset Password)
- shadcn/ui components (31: accordion, alert-dialog, avatar, badge, breadcrumb, button, card, checkbox, collapsible, command, dialog, dropdown-menu, hover-card, input, label, popover, progress, radio-group, scroll-area, select, separator, sheet, skeleton, slider, switch, table, tabs, textarea, toggle-group, tooltip)

## Cross-Language Integration Rules

1. **Go → Python:** Go calls Python via `os/exec` subprocess for `orca validate`, `orca calibrate`, `orca preflight`, `orca attribute`.
2. **Go → React:** WebSocket on `/ws` (gorilla/websocket). Push risk status every 5s, ticks every 50ms. REST API on `/api/v1/*`.

## Verification Gates

### Pre-Commit

1. **Python:** `ruff check orca/ tests/ && mypy orca/ && pytest tests/ -v`
2. **Go:** `go build ./... && go test ./internal/... -v -count=1 && golangci-lint run ./...`
3. **GKR validation:** `orca validate configs/strategies/*.gkr.yaml`
4. **Anti-pattern scan:** `python scripts/anti_pattern_scan.py` — zero violations
5. **LWC chart scan:** `node scripts/scan-chart-patterns.mjs` — zero errors (warnings allowed)
6. **Guardian tests:** `pytest tests/guardian/ -v` — all critical paths pass
7. **Frontend tests:** `cd web && npx tsc --noEmit && npx vitest run && npx playwright test` — `npm test` / `npm run test:watch` / `npm run test:coverage` / `npm run test:e2e` — 233 unit + 49 e2e

### CI/CD Pipeline

| Job | Language | Gates |
|-----|----------|-------|
| `python` | Python | ruff, mypy, pytest (coverage ≥ 80%) |
| `backend` | Go | golangci-lint, go vet, test (race + coverage ≥ 60%), E2E |
| `frontend` | React/TS | ESLint, tsc, vite build, vitest (228 tests), playwright (49 e2e tests) |
| `gkr-validate` | GKR IR | Strategy validation for all `.gkr.yaml` files |
| `anti-pattern-scan` | All | 18 hard prohibition enforcement |
| `security` | All | Gitleaks + Go vulnerability scan |
| `guardian` | Python + Go | Regression smoke tests (20 critical paths) |
| `mutation-test` | Python | Mutation testing on `orca/sizing/`, `orca/math/` (main only) |

### Pre-Deployment Gating

- `orca preflight --strict` — 12-point checklist
- GKR strategy hash verification
- `orca calibrate` — calibration audit
- Config hash integrity check
- Kill-switch E2E test
- Balance reconciliation

### Kilo Commands for CI/CD

| Command | Purpose |
|---------|---------|
| `/ci` | Run full CI pipeline locally |
| `/pr-check` | Pre-PR quality gates (test-related + anti-pattern) |
| `/preflight` | Pre-deployment checklist |
| `/calibrate` | Run calibration audit |
| `/regression-guard` | Run guardian smoke tests |
| `/test-related` | Run tests only for changed files |
| `/anti-pattern` | Scan for hard prohibition violations |
| `/fix-anti-pattern` | Auto-fix common violations |
| `/deploy-gate` | Full pre-deployment verification |

## Current Implementation State (2026-08-03)

### Strategy Portfolio (16 strategies, 14 registered)

| # | Strategy | File | Regimes | Kelly |
|---|----------|------|---------|-------|
| 1 | Grid Trading | `grid_runner.go` | Calm only (disabled by default) | 0.25 |
| 2 | Trend Following | `trend_runner.go` | Trending only | 0.25 |
| 3 | Session Scalp | `session_scalp_runner.go` | Calm, Trending, HighVol | 0.25/0.25/0.15 |
| 4 | Mean Reversion | `mean_reversion.go` | Calm only | 0.25 |
| 5 | ORB | `orb_runner.go` | Trending, HighVol | 0.25/0.15 |
| 6 | Pairs Trading | `pairs_runner.go` | Calm, HighVol | 0.25/0.15 |
| 7 | Volatility Harvesting | `vol_harvesting_runner.go` | HighVol only | 0.15 |
| 8 | Dragon Trend | `dragon_trend_runner.go` | Trending, HighVol | 0.25/0.15 |
| 9 | VWAP MR | `mean_reversion.go` (Mode="vwap") | Calm only | 0.25 |
| 10 | 15-Min ORB | `orb_runner.go` (range_minutes=15) | Trending, HighVol | 0.25/0.15 |
| 11 | Volume Scalp | `volume_scalp_runner.go` | Calm, Trending | 0.25 |
| 12 | VIX Futures Carry | `vix_futures_carry_runner.go` | HighVol only (spot VIX proxy) | 0.15 |
| 13 | Vol-Adjusted Grid | `grid_runner.go` (AdjustByVolatility=true) | Calm only | 0.15 |
| 14 | Ichimoku Cloud | `ichimoku.go` | All (permissive) | 0.25 |
| 15 | Donchian Breakout | `donchian.go` | All (permissive) | 0.25 |
| 16 | Keltner MACD | `keltner.go` | All (permissive) | 0.25 |

### Architecture Highlights

- **RiskPipeline** (`internal/risk/pipeline.go`): Canonical signal-audit path shared by backtest and live engines. ProcessSignal order: volatility halt → sizing → prop-firm halt → regime gate → sizing (Kelly + multipliers) → soft halt → exposure check → cross-strategy correlation brake → capital authorization.

- **SignalGateImpl** (`internal/risk/signal_gate_impl.go`): Concrete implementation wrapping VolatilityHalt, PositionSizer, ExposureTracker, and OrderRateLimiter. Used by both engines.

- **RegimeActivationMatrix** (`internal/risk/regime_activation.go`): 14 strategies × 4 regimes with per-regime Kelly multiplier overrides. Default mappings from the original Strategy Selection & Deployment Architecture (2026-08-10).

- **PropFirmEnforcer** (`internal/backtest/propfirm_enforcer.go`): Soft halt (positions reduced 50%) at configurable daily loss threshold, hard halt (trading stopped) at configurable limit. Per-profile configurable via `propfirm.Profile`.

- **Walk-Forward Automation** (`internal/scheduler/reoptimization.go`): Degradation-triggered daily re-optimization. Checks OOS Sharpe degradation (>20%) or parameter age (>90 days). Light optimizer integration with automatic version save/activate.

- **Parameter Versioning** (`internal/db/parameter_version_repo.go`): `strategy_params_version` table with JSONB params, IS/OOS metrics, active flag, and activate/deactivate API endpoints.

- **Phase 8 Completed**: All 7 alternative/complementary strategies implemented. Multi-Asset StatArb replaced by simpler PairsRunner (cointegration spread). VIX Futures Carry uses spot VIX as contango proxy. Vol-Adjusted Grid uses ATR/VIX dynamic spacing.

### Key Files Added (2026-08-03)

| File | Phase | Purpose |
|------|-------|---------|
| `internal/risk/signal_gate_impl.go` | -1 | Shared SignalGate for both engines |
| `internal/risk/regime_activation.go` | 1 | Strategy × regime matrix |
| `internal/risk/regime_activation_test.go` | 1 | Matrix + pipeline tests |
| `internal/strategy/vol_harvesting_runner.go` | 4 | Volatility harvesting |
| `internal/strategy/pairs_runner.go` | 4 | Cointegration spread trading |
| `internal/strategy/dragon_trend_runner.go` | 8 | Multi-EMA trend (8,21,50,200) |
| `internal/strategy/volume_scalp_runner.go` | 8 | Volume-confirmed scalp |
| `internal/strategy/vix_futures_carry_runner.go` | 8 | VIX contango proxy |
| `internal/db/migrations/000030_parameter_versions.up.sql` | 5 | Parameter version table |
| `internal/db/parameter_version_repo.go` | 5 | Version CRUD |
| `internal/scheduler/reoptimization.go` | 5 | Degradation-triggered re-optimization |
| `internal/api/param_version_handler.go` | 5 | API for param versions |
| `web/src/components/backtest/RegimeActivationMatrix.tsx` | 6 | Strategy × regime UI grid |
| `web/src/components/backtest/SoftHaltGauge.tsx` | 6 | Daily loss gauge |
| `web/src/pages/ParamVersionPage.tsx` | 6 | Parameter version management |
| `web/src/__tests__/regimeMatrix.test.tsx` | 7 | Frontend tests |
| `docs/Senior Quantitative Audit Report 2026-08-02.md` | — | Full audit report |
| `docs/Multi-Strategy Orchestration Design 2026-08-10.md` | — | Strategy selection & deployment architecture |
| `docs/Orchestration-Backtest Integration Plan 2026-08-10.md` | — | Orchestration integration plan |

### Known Issues

1. **Live engine BatchInferrer parity resolved** — `ProcessTickForAccount` now routes through `batchInferrer.Evaluate()` (three-layer: threshold skip → cache → inference). Parity with backtest engine established in commit d2e59c3. Removed unsafe type assertion; any `ml.Predictor` implementation works.

2. **`internal/ml/feature_store_persist.go`** — Now callable via `LiveEngine.PersistFeatureStore()` and `LoadFeatureStore()` methods. Wire into live engine lifecycle when ready (shutdown/startup hooks).

3. **VIX Futures Carry uses spot VIX proxy** — No VIX futures data feed is ingested. Strategy uses `SetVIX()` with spot VIX as contango signal until futures data is available.
