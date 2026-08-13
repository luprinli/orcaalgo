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

**18-symbol prop-firm universe** (equities, forex, crypto, indices). Real data replaces synthetic wherever available. Stooq dataset (`data/stooq/`, 31K files) provides real intraday bars; Yahoo provides 5-year daily history.

| Data Type | Source | Coverage |
|-----------|--------|----------|
| Candles (1d) | Yahoo Finance real daily | 5-year (2021-07 → 2026-08), all 18 symbols |
| Candles (1h) | **Stooq real hourly** + calibrated synthetic | Real 2-year (2024-07 → 2026-08); synthetic 3-year gap (2021-07 → 2024-07) |
| Candles (4h) | Resampled from stooq 1h + calibrated synthetic | Same as 1h |
| Candles (5m) | **Stooq real 5-min** + calibrated synthetic | Real 5-month (2026-03 → 2026-08); synthetic 4.7-year gap |
| Candles (15m/30m) | Resampled from stooq 5m + calibrated synthetic | Same as 5m |
| VIX | Yahoo ^VIX, **BIGINT** (migration 000039) | 1,284 rows, 5-year |
| Regime | Volatility/trend-based inference | 400+ rows |
| Sentiment | Synthetic + Alternative.me backfill | 9,000+ rows |

**Synthetic generation** (`scripts/stooq_synthetic.py`): Unconstrained Geometric Brownian Motion calibrated from per-symbol stooq σ (Close-to-Close + High-Low range, EWMA λ=0.94), with a soft blend toward the daily Close in the final 50% of the session. Paths are free to break through daily High/Low — no artificial clipping. Source labels: `stooq` (real), `stooq-resampled` (resampled from real), `stooq-calibrated` (synthetic gap-fill), `yahoo` (real 1d).

### Key Files Added (2026-08-12)

| File | Purpose |
|------|---------|
| `scripts/stooq_discovery.py` | Walk stooq tree → `data/stooq/manifest.json` (18-symbol mapping) |
| `scripts/stooq_seed.py` | Stream stooq 1h + 5m CSVs into candles (source='stooq') |
| `scripts/stooq_resample.py` | 1H→4H + 5m→15m/30m resampling (source='stooq-resampled') |
| `scripts/stooq_synthetic.py` | Unconstrained-GBM gap-fill calibrated from stooq σ/μ (source='stooq-calibrated') |
| `internal/db/migrations/000040_stooq_source.up.sql` | `source` column + unique (symbol_id, timeframe, time) constraint |
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
| `scripts/orchestrate.py` `_run_stooq_pipeline` | Orchestrator delegates intraday data to the stooq pipeline |

### Migrations

| # | Name | Purpose |
|---|------|---------|
| 000001–000038 | Various | Initial schema through sentiment_logs |
| **000039** | **vix_bigint** | VIX DOUBLE PRECISION → BIGINT (HP #2 compliance, idempotent) |
| **000040** | **stooq_source** | `source` column + unique (symbol_id, timeframe, time) constraint on candles |

### Known Issues

All 102 issues identified in the Production Audit Report (2026-08-11) have been resolved. The 15-item data infrastructure implementation backlog has been fully delivered. The real intraday data pipeline (stooq) provides 2-year 1H/4H coverage and 5-month 5m/15m/30m coverage, with stooq-calibrated synthetic bars filling the remaining 5-year gap. The synthetic generator uses unconstrained GBM with per-symbol σ — no fixed volatility multiplier, no daily OHLC clipping. Historical audit and implementation documents have been cleaned up.

## Backtest Remediation Status (2026-08-12)

Post-audit remediation of `docs/Backtest Readiness Audit matrix_results (7) 2026-08-12.md` is complete. All P0–P3 enhancements (E1–E15) plus the live-path daily-loss parity fix are implemented and verified (`go build/test/vet`, `orca validate` on 18 configs, `tsc --noEmit`).

Key changes agents must preserve:

- **Data loading is source+timeframe aware and priority-ordered.** `db.Repository` exposes `LoadCandlesFiltered(source)` and `LoadCandlesByTimeframeFiltered(timeframe, source)` backed by `SourceValues()` (`stooq` → `stooq`/`stooq-resampled`/`stooq-calibrated`; `yahoo` → `yahoo`; `seed` → `seed`). The loader applies a source-priority `DISTINCT ON` so the highest-priority source wins per bar — the legacy `seed` fixture and the `yahoo` provider are **never** merged into the `stooq` selector (they produced ~7–10x price-scale discontinuities). `backtestRepoAdapter` (`internal/api/router.go`) no longer hard-codes `1d` and no longer silently falls back to synthetic. Do not reintroduce a `LoadCandles`-only path for timeframed runs, and do not add `seed`/`yahoo` back to `SourceValues("stooq")`.
- **Unknown tickers error.** `symbolConfig` returns `(syntheticSymbolConfig, bool)`; `generateSyntheticCandles` returns an error for unmapped tickers. Add new universe tickers to **both** `configs/universe.json` and `symbolConfig`.
- **Daily-loss is per-day.** `PropFirmEnforcer.CheckDailyLoss` uses `DayStartingBalance` (reset in `OnNewDay`); `risk.CapitalPoolManager.RequestCapital` uses `TotalBalance − DailyPnL`. Do not compare cumulative balance against inception `StartingBalance`.
- **Deterministic metrics are ungated.** `MaxDrawdown`, `AvgWin/Loss`, `NumWins/Losses` compute for any non-zero trade count (only `Sharpe`/`Sortino`/`Calmar` remain gated).
- **Matrix plausibility gate.** `backtest.FlagImplausibleCombos` is surfaced as `MatrixResult.Plausibility`. Keep it wired in `RunMatrixConcurrent`.
- **All 17 matrix strategies are IR-backed.** `configs/strategies/*.gkr.yaml` (18 files) all pass `orca validate`. New strategies require a `.gkr.yaml` before entering the matrix.

Remaining follow-ups (documented, non-blocking): none — API walk-forward wiring, the `rsi_divergence` orphan, and the pairs-runner cointegration proxy have all been resolved. The one open item is a **fresh matrix re-run against real stooq data** (the dev seed's 2-decimal intraday rounding limits grid/ORB realism; see the audit report §11).

## Benchmark-Driven Enhancements (2026-08-13)

Post-benchmark remediation of `docs/StratCraft Benchmark 2026-08-13.md` (cross-system feature benchmark vs. the StratCraft reference) is complete. All 12 recommendations (R1–R12) are implemented and verified (`go build/test/vet`, `ruff`/`mypy`/`pytest`, `tsc --noEmit`). Agents must preserve the following:

### Anti-overfit scoring (`orca/scoring/`)

- **Layered parameter scoring** (`param_score.py`): percentile-rank core (Sharpe/Calmar/return, blended with verify-window metrics via geometric mean) × exponential drawdown penalty × neighbourhood-stability score (plateau preference, neutral below `min_pool_for_stability=8`) × balance penalty (asymmetric train→validation CAGR degradation). Pure dict-in/dict-out — unit-testable without DB/API.
- **Template scoring** (`template_score.py`): strategy-family ranking across multiple periods with length/recency weighting and a verification multiplier (0.8×–1.2×) from the best param row's verify metrics.
- **Ticker split** (`ticker_split.py`): deterministic SHA-256 train/validation split (`SPY`/`QQQ` forced to validation). Stable across processes.
- **CLI**: `orca score-params <rows.json>` and `orca score-templates <periods.json> [--verify ...]`. The statistical gates (Bonferroni/BH multiple-testing + walk-forward) remain in `orca/sizing/promotion_gate.py`; the new scoring is the *plateau/balance/cross-sectional* layer those gates do not cover.

### Trade drill-down (R10)

- `backtest.Trade.Changes` is an append-only `[]TradeChange` (`{timestamp, field, from, to, reason}`). Recorded via the single `addChange` helper (`trade_change.go`) at entry, initial stop/target set, trailing-stop ratchets, and both exit paths. Serialized into `backtest_results.trades` JSONB — do not mutate a trade outside `addChange` or the change log will desync from field state.
- `GET /backtests/:id/trades/:tradeId` returns `metrics.TradeDetail` (full `TradeSummary` + changes + reconstructed `lowest_price`/`highest_price` from MAE/MFE). Frontend `TradesTab` rows are clickable → drill-down panel.

### Backtest/live parity & cost modeling

- **ETF expense-ratio modeling (R12)**. `config.ExpenseRatioForTicker`/`ExpenseRatioForAssetClass` map `equity_etf`/`bond_etf`/`commodity_etf` → annual ratios; `broker.BrokerageFeeConfig.CalculateHoldingFee` charges `notional × ratio × yearsHeld`. Wired into **both** engine close paths via the shared `holdingExpenseFee` helper — do not charge expense ratio a second time anywhere.
- **Engine-vs-live comparison (R6)**. `backtest.ComputeImpliedComparison` (implied slippage, penetration, expense gap) and `MaxEquityDivergencePct` in `live_comparison.go`. The `/backtests/:id/live-comparison` handler now returns honest zero-values (not placeholder constants) when no live trades are linked.
- **Start-timing analysis (R4)**. `backtest.RunStartTiming` + pure `GenerateStartTimingWindows`; `POST /backtests/start-timing`.
- **Dispatch summary + limit-fill probability (R11)**. `notify.LimitFillProbability` (uses stdlib `math.Erfc` — no hand-rolled erf), `CalculateCashImpact`, `BuildDispatchSummary` in `dispatch_summary.go`.

### Admin/ops endpoints (R3, R5, R7, R8, R9)

- **Corporate actions (R3)**. `db.UpsertCorporateAction` + `db.ListCorporateActions`; `GET/POST /admin/corporate-actions`; Admin "Corporate Actions" tab.
- **ML model list (R5)**. `ml.ListModels`; `GET /models`; Admin "ML Models" list table.
- **Job scheduler (R7)**. `scheduler.ListJobs` + `RunJobNow` (thread-safe `lastRun` map); `GET /admin/jobs`, `POST /admin/jobs/run`; Admin "Jobs" tab. The `Server` now holds the scheduler via `SetScheduler`.
- **Backtest-cache admin (R8)**. `db.Export/Import/PruneBacktestCache`; `GET /admin/backtest-cache/export`, `POST .../import`, `POST .../prune`.
- **DB backup/restore (R9)**. `GET /admin/database/backup` (`pg_dump`), `POST /admin/database/restore` (`psql`).

### Preserve these invariants

- **Fee/expense parity**: `holdingExpenseFee` (backtest) is the only place the holding cost is computed. If the paper/live broker gains holding-cost accounting, reuse `BrokerageFeeConfig.CalculateHoldingFee` rather than re-deriving it.
- **No reimplemented math**: limit-fill probability uses Go stdlib `math.Erfc`; Kelly/Brier/Platt/Wilson/EWMA remain Python-only (HP #1).
- **Backward compatibility**: all new endpoints are additive; no existing route, schema, or response shape was changed. The `Trade.Changes` field is additive to the serialized trade JSON.
- **Frontend parity**: every backend change has a matching client method (`web/src/api/client.ts`) and Admin tab or detail panel.
