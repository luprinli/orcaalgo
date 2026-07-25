# OrcaAlgo — Agent Instructions & Tech Stack Compliance

## Stack Constitution (Immutable)

The approved technology stack per `docs/stack-critique-and-recommendations.md`:

| Component | Language | Role | Spec Ref |
|-----------|----------|------|----------|
| Strategy IR, Math, Calibration | **Python** (3.11+) | Domain models, GKR strategy IR, canonical math (Kelly, Brier, Platt, Wilson, EWMA), calibration audit, PnL attribution, pre-flight checklist | §2.1.2, §3, §5, §9, §10 |
| API, Broker, Ingest, Scheduling | **Go** (1.25) | HTTP API (Gin), broker integration (Alpaca/IBKR/Paper), market data ingestion (WebSocket→ring buffer), WebSocket hub, background scheduler, DB repository | §2.1.3, §6, §7, §11 |
| Web Dashboard | **React + TypeScript** | SPA with lightweight-charts, WebSocket live feed, strategy/backtest/execution pages | §11 |
| Time-Series Storage | **PostgreSQL + TimescaleDB** | Hypertables (market_ticks, candles, trade_executions), BIGINT fixed-point price storage, compression (7d) + retention (30d) policies | §6.8, §7 |
| Audit Log (consider) | **SQLite** | Append-only WAL mode for lightweight deployments | §7.1 |

## Hard Prohibitions

These are **NEVER** permitted. Violations block PR merge.

| # | Rule | Source |
|---|------|--------|
| 1 | **Do not reimplement canonical math functions in Go.** Kelly, Brier, Platt, Wilson, EWMA exist in `orca/sizing/` and `orca/math/`. Reference them via subprocess or import. | Antipattern #2 |
| 2 | **Do not use IEEE 754 float for order prices.** Use `BIGINT` with scale factor in PostgreSQL, `Decimal` in Python, `fixed.Fixed` in Go (recommended). | Antipattern #5 |
| 3 | **Do not commit strategy configs in legacy YAML format.** All strategies must be `.gkr.yaml` with versioning, hashing, and type validation. | §5.1 |
| 4 | **Do not deploy to production without pre-flight.** `orca preflight` must pass with zero failures before any live deployment. | §9.3 |
| 5 | **Do not skip calibration audits.** Quarterly `orca calibrate` runs are mandatory for all probability-emitting models. | §9.2 |
| 6 | **Do not use full Kelly in production.** Fractional Kelly (k=0.25) with all three attenuators (edge discount, fractional multiplier, hard caps) is mandatory. | §3.1.3 |
| 7 | **Do not mutate domain models.** All Pydantic models use `ConfigDict(frozen=True, extra="forbid")`. All Go structs use unexported fields with constructor-only initialization. | §2.1.2 |
| 8 | **Do not bypass the kill-switch re-entrancy guard.** `isLocked` + `killSwitchReady` must both be checked before any kill-switch execution. | §4.2.2 |
| 9 | **Do not assume perfect fills at mid-price.** Backtests must model maker fill prices, fill probability, spread crossing, fees, and adverse selection. | §9.1.3 |
| 10 | **Do not panic/throw for recoverable errors.** Return errors. Only unrecoverable startup failures may terminate. | Antipattern #10 |
| 11 | **Do not use `setData()` for incremental chart updates.** Use `ISeriesApi.update()` for real-time / polling updates. `setData()` is for initial load or full data replacement only. | §D audit, lightweight-charts docs |
| 12 | **Do not call `fitContent()` on every data update.** `fitContent()` resets the user's scroll/zoom position. Call it only on initial load, timeframe change, or explicit user action (e.g., "Reset view" button). | §D audit, lightweight-charts docs |
| 13 | **Do not use `applyOptions({ width })` for chart resize.** Use `chart.resize(width, height)`. `applyOptions` re-applies all chart options on every dimension change — `resize()` is the correct zero-overhead API. | §D audit, lightweight-charts docs |
| 14 | **Do not use `barSpacing` mutation for keyboard zoom.** Use `getVisibleLogicalRange()` + `setVisibleLogicalRange()` on the time scale. Mutating `barSpacing` conflicts with the chart's internal range calculation. | §D audit |
| 15 | **Do not leave `requestAnimationFrame` un-cancelled.** All `requestAnimationFrame` calls in chart hooks (`useChartUpdate`, drawing tools) must be cancelled in the `useEffect` cleanup via `cancelAnimationFrame`. | React best practice |
| 16 | **Do not use `Array.find()` in crosshair handlers.** Crosshair `subscribeCrosshairMove` fires at 60fps. Build `Map<time, value>` lookups in a `useEffect` and use O(1) `.get()` in the handler. | Performance |

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
- Risk management (kill-switch, rate limiting, memory guard)
- Background scheduler
- LLM client abstraction
- Backtest execution engine
- Monitor/metrics/telemetry

### What goes in React (`web/`)
- Dashboard, Live Monitor, Execution pages
- Backtest Runner and Strategy pages
- Broker and Symbol management pages
- Chart components (CVD, Equity, Volume Profile)
- Risk components (Status, Emergency Stop, Regime Gauge)
- WebSocket provider for real-time data
- Auth pages (Login, 2FA Setup)

## Cross-Language Integration Rules

1. **Go → Python:** Go calls Python via `os/exec` subprocess for `orca validate`, `orca calibrate`, `orca preflight`, `orca attribute`.
2. **Go → React:** WebSocket on `/ws` (gorilla/websocket). Push risk status every 5s, ticks every 50ms. REST API on `/api/v1/*`.

## Spec Reference Quick Links

| Topic | Spec Section | Source File |
|-------|-------------|-------------|
| Bounded context map | §2.1.1 | — |
| Immutable domain models | §2.1.2 | `orca/models/` |
| Event-driven architecture | §2.1.3 | `internal/monitor/ws_hub.go` |
| Factory + plugin registry | §2.3 | `internal/broker/registry.go`, `internal/broker/broker_driver.go` |
| Kelly criterion (all variants) | §3.1 | `orca/sizing/kelly.py` |
| Brier score / Murphy decomposition | §3.2.1 | `orca/math/brier.py` |
| Platt scaling | §3.2.2 | `orca/math/platt.py` |
| Wilson CI | §3.2.3 | `orca/math/wilson.py` |
| EWMA volatility | §3.3.1 | `orca/sizing/volatility.py` |
| Triple barrier labeling | §3.4 | go-trader-main reference |
| CUSUM detection | §3.5 | go-trader-main reference |
| Drawdown (HWM trailing) | §3.9 | `internal/propfirm/rules.go` |
| Triple safety net | §4.1 | `internal/risk/kill_switch.go` |
| Kill-switch protocol | §4.2 | `internal/risk/kill_switch.go` |
| GKR strategy IR | §5.1 | `orca/ir/schema.py` |
| Three-layer hashing | §5.1.1 | `orca/hash/graph.py` |
| Temporal validation | §5.1.2 | `orca/ports/temporal.py` |
| Fixed-point arithmetic | §6.8 | PostgreSQL BIGINT, `decimal` type |
| Append-only audit log | §7.1 | TimescaleDB hypertables |
| Configuration hierarchy | §8.1 | `configs/` |
| Profile gating | §8.4 | `orca/ir/validator.py` profiles |
| Purged CV | §9.1.1 | research reference |
| Calibration audit | §9.2 | `orca/calibration/audit.py` |
| Pre-flight checklist | §9.3 | `orca/preflight/checklist.py` |
| PnL attribution | §9.4 | `orca/attribution/slicer.py` |
| Content-addressable hashing | §10.3 | `orca/hash/common.py` |
| Source of truth rule | §10.4 | This document |
| Dashboard panels | §11.1.1 | `web/src/components/`, `web/src/pages/` |
| Visual risk signaling | §11.1.2 | `web/src/components/risk/` |
| Structured logging | §11.3 | Go `slog`, Python `structlog` |

## Verification Gate (Pre-Commit)

Before committing any change, verify:

1. **Python:** `ruff check orca/ tests/ && mypy orca/ && pytest tests/ -v`
2. **Go:** `go build ./... && go test ./internal/... -v -count=1 && golangci-lint run ./...`
3. **GKR validation:** `orca validate configs/strategies/*.gkr.yaml`
4. **Anti-pattern scan:** `python scripts/anti_pattern_scan.py` — zero violations
5. **LWC chart scan:** `node scripts/scan-chart-patterns.mjs` — zero errors (warnings allowed)
6. **Guardian tests:** `pytest tests/guardian/ -v` — all critical paths pass

## CI/CD Pipeline

The project has a fully automated CI/CD pipeline (`.github/workflows/ci.yml`) with:

| Job | Language | Gates |
|-----|----------|-------|
| `python` | Python | ruff, mypy, pytest (coverage ≥ 80%) |
| `backend` | Go | golangci-lint, go vet, test (race + coverage ≥ 60%), E2E |
| `frontend` | React/TS | ESLint, tsc, vite build |
| `gkr-validate` | GKR IR | Strategy validation for all `.gkr.yaml` files |
| `anti-pattern-scan` | All | 10 hard prohibition enforcement |
| `security` | All | Gitleaks + Go vulnerability scan |
| `guardian` | Python + Go | Regression smoke tests (20 critical paths) |
| `mutation-test` | Python | Mutation testing on `orca/sizing/`, `orca/math/` (main only) |

Pre-deployment gating (`.github/workflows/pre-deploy.yml`):

- `orca preflight --strict` — 12-point checklist
- GKR strategy hash verification
- `orca calibrate` — calibration audit
- Config hash integrity check
- Kill-switch E2E test
- Balance reconciliation

Scheduled dependency audit (`.github/workflows/dependency-audit.yml`: weekly):

- `govulncheck`, `pip-audit`, `npm audit`

## Kilo Commands for CI/CD

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

See `.kilo/command/` for full command definitions and `.kilo/agent/` for specialized CI/CD diagnostic agents.

## Implementation Priority

| Priority | Task |
|----------|------|
| P0 | All 10 hard prohibitions enforced |
| P0 | `orca validate` passes on all `.gkr.yaml` strategies |
| P0 | Kill-switch with re-entrancy guard operational |
| P0 | Fixed-point price storage in PostgreSQL confirmed |
| P1 | `orca calibrate` pipeline runs quarterly |
| P1 | `orca preflight` gates all live deployments |
| P1 | `orca attribute` produces Wilson CI slices |
| P2 | Go fixed-point migration (`github.com/robaho/fixed`) |
| P2 | SQLite append-only audit log |
