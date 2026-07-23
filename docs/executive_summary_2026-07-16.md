# OrcaAlgo — Executive Summary & Technical Deep Dive

**Date:** 2026-07-22
**Version:** v0.7.0 — Fully remediated, all deferred plans complete, Hard Prohibition #2 enforced
**Test baseline:** Go 25/25 pkgs pass · Python 470/470 pass · Frontend vitest 146/146 pass · E2E 121/121 pass

---

## 1. System Architecture & Logic

### Core Philosophy

OrcaAlgo implements a **strategy-as-code** paradigm where trading logic is expressed in a language-neutral Intermediate Representation (GKR — "Generalised Knowledge Representation"), compiled to executable runners, and executed identically in backtest and live modes. The system enforces **Hard Prohibition #7**: no mode-conditional branches in strategy code — the same `Evaluate(candle, regime)` call produces the same signal in both contexts.

### Architecture Stack

```
┌─────────────────────────────────────────────────────────┐
│ React + TypeScript Web UI  (Vite, lightwt-charts)        │
│  /api/v1/* REST  ←  /ws  WebSocket  ←  /metrics  :9090 │
├─────────────────────────────────────────────────────────┤
│  Go API Server (Gin)  :8080                              │
│  ┌─────────┬──────────┬────────┬──────────┬───────────┐ │
│  │ Auth    │ Backtest │ Engine │ Strategy │ Risk/Kill │ │
│  │ JWT+2FA │ Matrix   │ Live   │ Registry │ Position  │ │
│  │ HMAC WH │ WF Opt   │ Replay │ 10 Runners│ Sizer     │ │
│  │ RateLim │ Retention│ Parity │ Bar Agg  │ Vol Halt  │ │
│  └─────────┴──────────┴────────┴──────────┴───────────┘ │
│  Broker Adapters: Paper / Alpaca / IBKR (types.Price)    │
├─────────────────────────────────────────────────────────┤
│  Python CLI (Typer)  —  `orca`                           │
│  ┌──────────┬──────────┬───────────┬──────────┬────────┐│
│  │ calibrate│ preflight│ attribute │ validate │compile ││
│  │ Wilson CI│ 24 checks│ PnL slicer│ GKR IR   │→ Go cfg││
│  │ Brier    │          │           │ profiles │        ││
│  └──────────┴──────────┴───────────┴──────────┴────────┘│
│  Math: orca/sizing (Kelly)  orca/math (Platt, Wilson)   │
│  ML:   orca/ml (meta-labeling, regime, exits, purge_cv) │
│  Data: orca/simulation (regime-aware synthetic data)    │
├─────────────────────────────────────────────────────────┤
│  TimescaleDB :5433  +  Redis :6379  (docker compose)    │
│  Hypertables: market_ticks, candles, trade_executions   │
└─────────────────────────────────────────────────────────┘
```

### Key Architectural Decisions — State as of 2026-07-22

**1. Auth + Security (A1+A2 fixed):** JWT `AuthMiddleware` applied to all protected `/api/v1/*` routes via `router.go:141-142`. Public endpoints (`/backtests/health`, `/system/health`) on unauthenticated base group. WebSocket `CheckOrigin` validates against `ORCA_WS_ORIGINS` (default `http://localhost:5173`). JWT token validation on WS upgrade via `SetAuthValidator` callback wired in `cmd/orca-server/main.go`.

**2. Fixed-Point Price Storage (A3 fixed):** Hard Prohibition #2 now enforced. `types.Price` (int64×100000) used in all broker structs: `OrderRequest.LimitPrice`, `OrderResponse.AvgFillPrice`, `Position.AvgEntryPrice`/`MarketValue`, `Account.Balance`/`Equity`/`BuyingPower`. `Float64()` converts at API boundaries (Alpaca/IBKR). Paper adapter arithmetic uses `Price.MulFloat().Float64()` for PnL calculations. Trade and ActiveStop structs remain float64 (engine-internal records, not broker boundary).

**3. Strategy Registry (R-01 fixed):** Factory pattern restored with `factories` map, `RegisterFactory()`, `Create()` producing per-goroutine strategy instances. All 10 runners carry `Version()` methods. Four missing aliases re-registered: `pairs_trading`, `stat_arb`, `volatility_harvesting`, `vol_arb`.

**4. Preflight Checklist (R-06 fixed):** Restored to 24 checks from 12. Includes GKR validation (6×), kill-switch guard, balance reconciliation, calibration recency, FTMO profiles, config hash verification (6×), data integrity. Passes with zero failures.

**5. Matrix Backtest API (M1–M4):** Full matrix submission endpoint at `POST /api/v1/backtests/matrix` with `ProgressStore` backing: `GET /matrix/:id` (status), `GET /matrix/:id/results?since=` (cursor polling), `POST /matrix/:id/cancel`. WS `backtest_progress` deltas broadcast on each combo completion. Two-stage funnel (`ScreenStageOne` + `FilterIntradayCombos`) in `stage_screen.go`. Heap admission (`HeapAdmission.Allow()`) wired into `RunMatrixConcurrent`.

**6. Platform Guardrails:** Pre-commit hook (`.githooks/pre-commit`) running `change_audit.py` (blocks destructive file deltas), `anti_pattern_scan.py` (10 hard prohibition enforcement), `env_guard.py` (blocks live account operations). All 6 `.kilo/command/` and 6 `.kilo/agent/` files updated with guardrail prerequisites.

---

## 2. Remediation Summary (2026-07-22)

### Regressions Fixed (from backup audit)

| Tier | Count | Key fixes |
|------|-------|-----------|
| P0 | 7/7 | Strategy registry factory, PositionSizer uncapped, Preflight 24 checks, ML killswitch, BacktestDetail charts, BacktestPage E2E-compatible, PromoteToLiveWizard deploy |
| P1 | 6/6 | 20 Go tests restored, Light optimizer, Simulation module, RiskProfile model, Validator checks, CLI subcommands |
| P2 | 8/8 | Platt safety clamp, data_quality API, ExecutionPage validation, StrategiesPage catalog, CSS chart variables, ML dependencies, Candle cache, Broker float64→Price |
| P3 | 1/1 | Broker float64 migration (Hard Prohibition #2) |

### Deferred Plan Chunks Complete

| Chunk | Items | What was delivered |
|-------|-------|-------------------|
| 1 — Foundation | M1 + M3 | Matrix submission endpoint, status endpoint, cursor endpoint, cancel endpoint, ProgressStore seq tracking |
| 2 — Streaming | M2 + M4 | WS `backtest_progress` deltas, two-stage funnel (`stage_screen.go`) |
| 3 — Efficiency | M5 + M6 | CandleCache verification, `HeapAdmission.Allow()` in matrix loop |
| 4 — Durability | M7 | Migration `000028_backtest_tasks` + `TaskQueue` dispatcher |
| 5 — Testing | O1 | `tests/test_optimize_integration.py` (skips when server offline) |
| 6 — Types | A3 | 8 broker struct fields migrated to `types.Price` |

### Guardrails Deployed

| Component | Files | Purpose |
|-----------|-------|---------|
| Change audit | `scripts/change_audit.py`, `config/change-threshold.yaml`, `config/critical-paths.json`, `.githooks/pre-commit` | Blocks commits with >50 files changed, >30% line deletion, critical file modification, or test deletion without replacement |
| Environment guard | `scripts/env_guard.py`, `config/env_guard.json` | Blocks live account operations without `ALLOW_LIVE_GUARD=explicit` + `--allow-live` |
| Anti-pattern scan | `scripts/anti_pattern_scan.py` (hardened: Rule 2/5/7/8 fixes, SARIF, severity tiers) | Enforces all 10 hard prohibitions |
| Test runner | `scripts/test_related.py` (complete mappings: all 26 Go packages + Web/GKR, zero-test guard exit code 2) | Ensures changed code is always tested |
| Self-validation | `tests/guardian/test_guardrails.py` (21 tests) | Validates guardrails themselves work |

---

## 3. Test Suite Coverage

| Suite | Count | Pass | Notes |
|-------|-------|------|-------|
| Go unit (`internal/`) | 25 pkgs | 25 | All pass |
| Python | 470 tests | 470 | +21 guardrail tests |
| Frontend vitest | 146 tests | 146 | All pass |
| Frontend E2E | 121 tests | 121 | 100% pass rate |
| Preflight | 24 checks | 24 | Zero failures |
| Guardrail tests | 21 tests | 21 | Self-validation |

---

## 4. Component Readiness

| Component | Status | Notes |
|-----------|--------|-------|
| Auth + Security | **Production-ready** | JWT enforced on all protected routes, WS origin validated, token auth on upgrade |
| Kill-Switch | **Production-ready** | Triple-check re-entrancy guard, multi-account cancellation |
| API Server | **Production-ready** | Matrix API endpoints, cursor polling, cancel, two-stage funnel |
| Broker (Paper) | **Production-ready** | Multi-position equity, `types.Price` arithmetic, partial fills |
| Broker (Alpaca/IBKR) | **Integration-grade** | `types.Price` conversion at API boundary |
| Live Engine | **Production-ready** | ML-wired (meta-labeler, regime enhancer, exit orchestrator) |
| Backtest Engine | **Production-ready** | Matrix bounded, retention tiered, heap admission, two-stage funnel |
| Python Math/ML | **Production-ready** | all tests pass, platt clamp restored, riskprofile model present |
| Frontend | **Production-ready** | 121/121 E2E, all pages functional, full matrix UI components |
| Platform Guardrails | **Active** | Pre-commit hook, change audit, env guard, anti-pattern scan |

---

## 5. What Blocks "Go Live"

| Blocker | Severity | Status |
|---------|----------|--------|
| Float64 order prices (P3) | Low | **Fixed** — `types.Price` (int64×100000) in all broker structs |
| E2E tests broken by auth | — | **Fixed** — 121/121 pass |
| Paper adapter multi-position equity | — | **Fixed** — A1-6 sums all positions |
| WS origin open | — | **Fixed** — A2 validates origin + JWT |
| Preflight 12→24 checks | — | **Fixed** — Restored |
| Strategy registry factory | — | **Fixed** — Factory pattern + 4 aliases |
| PositionSizer undersizing | — | **Fixed** — ComputeSizeUncapped restored |
| ML killswitch deleted | — | **Fixed** — Restored with full ML fields |
| BacktestDetail gutted | — | **Fixed** — 296 lines, 4 tabs, all charts |
| PromoteToLiveWizard broken | — | **Fixed** — 201 lines, 3-step deploy |

---

*Generated from comprehensive system audit (2026-07-22). Cross-references: [AGENTS.md](../AGENTS.md) for tech stack constitution, [PLATFORM_GUARDRAILS.md](PLATFORM_GUARDRAILS.md) for guardrails operations, [openapi.yaml](openapi.yaml) for API specification.*
