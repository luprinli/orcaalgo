# OrcaAlgo — Agent Instructions & Tech Stack Compliance

## Stack Constitution (Immutable)

The approved technology stack:

| Component | Language | Role |
|-----------|----------|------|
| Strategy IR, Math, Calibration | **Python** (3.11+) | Domain models, GKR strategy IR, canonical math (Kelly, Brier, Platt, Wilson, EWMA), calibration audit, PnL attribution, pre-flight checklist, block bootstrap MC, multiple testing correction, NYSE holiday calendar, data integrity validation |
| API, Broker, Ingest, Scheduling | **Go** (1.25) | HTTP API (Gin), broker integration (Alpaca/IBKR/Paper), market data ingestion (WebSocket→ring buffer), WebSocket hub, background scheduler, DB repository, RiskPipeline + CapitalGate + PropFirmGate + SignalGate, CapitalPoolSim/CapitalPoolManager, BaseCapitalPool, KillSwitch + multi-account iteration, per-account strategy isolation, light optimizer wiring (run + sensitivity report), Monte Carlo bootstrapping, walk-forward framework, VIX BIGINT migration |
| Web Dashboard | **React 18 + TypeScript 5 + Tailwind CSS 4 + shadcn/ui** | SPA with lightweight-charts, WebSocket live feed, MonitorPage (4 tabs), BacktestHub (Runner + History + Detail + Optimize), StrategyHub (3 tabs), ChartingHub (Candles + Indicators), IntegrationsPage, Accounts, Admin (9 tabs), Emergency |
| Time-Series Storage | **PostgreSQL + TimescaleDB** | Hypertables (market_ticks, candles, trade_executions), BIGINT fixed-point price storage (candles + VIX), compression (7d) + retention (30d) policies |
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
| 17 | **Do not bypass the RiskPipeline in backtest or live paths.** `Engine.generateSignal` and `LiveEngine.ProcessTick` both route through `RiskPipeline.ProcessSignal` / `ReconcileFill`. New risk checks must be added to the pipeline, not duplicated in each engine. CI-enforced via anti-pattern Rule 11. | Architecture |
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
- Data pipeline: seed-all, resampling, regime inference, VIX ingestion, data integrity validation
- Sentiment backfill (Alternative.me Fear & Greed Index)
- NYSE holiday calendar for trading day filtering
- Block bootstrap Monte Carlo for performance estimation
- Multiple testing correction (Bonferroni + Benjamini-Hochberg)

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
- Backtest execution engine (event-driven, walk-forward, Monte Carlo, light optimizer, parameter sensitivity)
- Per-account strategy isolation (`internal/engine/live_engine.go`)
- Matrix runner with `--optimize` and `--walk-forward` flags
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

1. **Go → Python:** Go calls Python via `os/exec` subprocess for `orca validate`, `orca calibrate`, `orca preflight`, `orca attribute`, `orca validate-data-integrity`.
2. **Go → React:** WebSocket on `/ws` (gorilla/websocket). Push risk status every 5s, ticks every 50ms. REST API on `/api/v1/*`.

## Verification Gates

### Pre-Commit

1. **Python:** `ruff check orca/ tests/ && mypy orca/ && pytest tests/ -v`
2. **Go:** `go build ./... && go test ./internal/... -v -count=1 && golangci-lint run ./...`
3. **GKR validation:** `orca validate configs/strategies/*.gkr.yaml`
4. **Anti-pattern scan:** `python scripts/anti_pattern_scan.py` — zero violations (18 rules including HP #17 CI check)
5. **LWC chart scan:** `node scripts/scan-chart-patterns.mjs` — zero errors (warnings allowed)
6. **Guardian tests:** `pytest tests/guardian/ -v` — all critical paths pass
7. **Frontend tests:** `cd web && npx tsc --noEmit && npx vitest run && npx playwright test` — `npm test` / `npm run test:watch` / `npm run test:coverage` / `npm run test:e2e` — 233 unit + 49 e2e

### CI/CD Pipeline

| Job | Language | Gates |
|-----|----------|-------|
| `python` | Python | ruff, mypy, pytest (coverage ≥ 80%) |
| `backend` | Go | golangci-lint, go vet, test (race + coverage ≥ 60%), E2E |
| `frontend` | React/TS | ESLint, tsc, vite build, vitest (233 tests), playwright (49 e2e tests) |
| `gkr-validate` | GKR IR | Strategy validation for all `.gkr.yaml` files |
| `anti-pattern-scan` | All | 18 hard prohibition enforcement (includes HP #17 Rule 11) |
| `security` | All | Gitleaks + Go vulnerability scan |
| `guardian` | Python + Go | Regression smoke tests (20 critical paths) |
| `mutation-test` | Python | Mutation testing on `orca/sizing/`, `orca/math/` (main only) |

### Pre-Deployment Gating

- `orca preflight --strict` — 12-point checklist
- GKR strategy hash verification
- `orca calibrate` — calibration audit
- `orca validate-data-integrity` — cross-pipeline data integrity check
- Config hash integrity check
- Kill-switch E2E test
- Balance reconciliation
- Backtest-vs-replay parity verification

## Audit Remediation Summary (2026-08-11)

Total issues resolved: 102 (from Production Audit Report 2026-08-11). All 11 CRITICAL, 35 HIGH, 38 MEDIUM, 18 LOW resolved.

### Key New Packages

| Package | Purpose |
|---------|---------|
| `internal/breaker/` | Circuit breakers on telegram/VIX/sentiment |
| `internal/api/middleware/rate_limit.go` | API rate limiting |
| `internal/api/middleware/trace.go` | Request tracing |
| `internal/db/token_revocation.go` | JWT token revocation |

### Key New Infrastructure

- Rate limiting middleware, login brute-force protection
- Circuit breakers on telegram/VIX/sentiment
- Readiness probe `/readyz`, 30s drain timeout
- Config validation at startup, `ORCA_ENVIRONMENT` flag
- DB-based token revocation

### Library Upgrades

| Component | Before | After |
|-----------|--------|-------|
| JWT | Hand-rolled | `golang-jwt/jwt/v5` |
| TOTP | Hand-rolled | `pquerna/otp` |

### Infrastructure Changes

- Dockerfile: non-root user added
- Redis removed from docker-compose
- Compression/retention added to candles hypertable (migration 000036)

### ML Parity

- FeatureStore integration in backtest engine
- MTM equity tracking
- PWin formula unified between backtest and live paths

### Python Fixes

11 math/model fixes: Monte Carlo, EWMA, Kelly, Poisson, random seeds, validator, compiler.

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

## Current Implementation State (2026-08-12)

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

- **RiskPipeline** (`internal/risk/pipeline.go`): Canonical signal-audit path shared by backtest and live engines. ProcessSignal order: volatility halt → sizing → prop-firm halt → regime gate → sizing (Kelly + multipliers) → soft halt → exposure check → cross-strategy correlation brake → capital authorization. **CI-enforced**: Anti-pattern Rule 11 checks that `WirePipeline()` is called between `NewEngine()` and `Run()`.

- **SignalGateImpl** (`internal/risk/signal_gate_impl.go`): Concrete implementation wrapping VolatilityHalt, PositionSizer, ExposureTracker, and OrderRateLimiter. Used by both engines.

- **RegimeActivationMatrix** (`internal/risk/regime_activation.go`): 14 strategies × 4 regimes with per-regime Kelly multiplier overrides.

- **PropFirmEnforcer** (`internal/backtest/propfirm_enforcer.go`): Soft halt (positions reduced 50%) at configurable daily loss threshold, hard halt (trading stopped) at configurable limit.

- **Walk-Forward Automation** (`internal/scheduler/reoptimization.go`): Degradation-triggered daily re-optimization. Checks OOS Sharpe degradation (>20%) or parameter age (>90 days).

- **Parameter Versioning** (`internal/db/parameter_version_repo.go`): `strategy_params_version` table with JSONB params, IS/OOS metrics, active flag.

### Data Infrastructure (2026-08-12)

| Data Type | Source | Status |
|-----------|--------|--------|
| Candles | Yahoo Finance 1d + 5m, resampled to 15m/30m/1h/4h | 36,693 bars, 475 combos BPD ≤5% |
| VIX | Yahoo ^VIX + OU model fallback, **BIGINT** (migration 000039) | 1,255 rows, 5-year history |
| Regime | Volatility/trend-based inference from candle data | 8,841 rows across 7 symbols |
| Sentiment | Synthetic from returns + Alternative.me backfill | 110 rows synthetic, full history available |

### Key Files Added (2026-08-12)

| File | Purpose |
|------|---------|
| `scripts/anti_pattern_scan.py` Rule 11 | HP #17 CI check — detects NewEngine()→Run() without WirePipeline() |
| `internal/backtest/parity_test.go` | Backtest-vs-replay parity: batch/streaming determinism, pipeline signal parity |
| `cmd/matrix-runner/main.go` `--optimize` | Per-combo light optimizer in matrix sweeps |
| `cmd/matrix-runner/main.go` `--walk-forward` | Per-combo walk-forward validation (WfISSharpe/WfOOSSharpe columns) |
| `orca/data/validate_integrity.py` | Cross-pipeline data integrity validation CLI |
| `orca/data/sentiment_backfill.py` | Historical sentiment backfill from Alternative.me Fear & Greed Index |
| `orca/data/nyse_calendar.py` | NYSE holiday calendar (Gauss algorithm, observed holidays) |
| `internal/backtest/light_optimizer.go` `RunParameterSensitivity` | Per-parameter sensitivity scores + robust/moderate/sensitive classification |
| `orca/simulation/validate.py` `generate_first` | Fix circular dependency in validate_strategy_coverage |
| `orca/data/seed_all.py` `_write_manifest` | `data/.manifest.json` auto-generated on every seed-all |
| `internal/db/migrations/000039_vix_bigint.up.sql` | VIX DOUBLE PRECISION → BIGINT (scale 10000, idempotent) |
| `orca/sizing/block_bootstrap.py` | Block bootstrap Monte Carlo with temporal dependency preservation |
| `orca/sizing/multiple_testing.py` | Bonferroni + Benjamini-Hochberg multiple testing correction |
| `orca/simulation/tick_disaggregator.py` `get_symbol_ticks_per_minute` | Per-symbol liquidity-configured tick generation |
| `orca/simulation/generate_1m.py` _get_us_trading_days | NYSE holiday calendar integration in synthetic pipeline |

### Migrations

| # | Name | Purpose |
|---|------|---------|
| 000001–000038 | Various | Initial schema through sentiment_logs |
| **000039** | **vix_bigint** | VIX DOUBLE PRECISION → BIGINT (HP #2 compliance, idempotent) |

### Known Issues

All 102 issues identified in the Production Audit Report (2026-08-11) have been resolved. The 15-item data infrastructure implementation backlog (E-4, E-23, E-26, E-36, D-6, D-8, D-9, D-10, D-12, D-15, E-19b, E-24, E-27, E-28, E-36b) has been fully delivered. Historical audit and implementation documents have been cleaned up.
