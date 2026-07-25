# Per-Strategy Optimization in Matrix Backtests — Implementation & Regression Protection

**Date:** 2026-07-25
**Scope:** Five critical remediations restoring and wiring the light optimizer into production paths.

---

## Architecture Overview

The light optimizer (`RunLightOptimize`) runs a bounded, train/test-split parameter sweep for a single strategy on a representative subset of symbols, then applies the best-found parameters to all combos of that strategy in the matrix. This fixes the "flat parameter sensitivity" anti-pattern — previously every combo ran with default strategy params, producing a flat surface that masked true performance variation.

### Call Graph

```
Frontend submitOptimizationRun()
  → POST /api/v1/optimize/run
    → submitBacktestWithOptimization (async goroutine)
      → RunLightOptimize(ctx, db, cfg)
        → applyLightOptDefaults
        → lightOptCacheGet/cachePut (in-memory, SHA-256 keyed, TTL 7d)
        → splitWindow (67%/33% train/test)
        → generateCandidates (bounded enumeration or random sample)
        → evalLightParams (per-candidate backtest w/ composite score)
        → plateauPatience early stop
        → OOS validation fallback (best params vs defaults on test set)
      → final backtest with best params → bestMetric (Sharpe)

Matrix runner (RunMatrixConcurrent)
  → per unique strategy:
    → RunLightOptimize(ctx, db, lightCfg)
  → per combo:
    → BacktestConfig.StrategyParams = optimizedParams[strategyID]
```

---

## Remediation 1: Wire `RunLightOptimize` into `RunMatrixConcurrent`

**File:** `internal/backtest/batch_runner.go:168-235`

Before the matrix dispatch loop, unique strategies are collected. For each unseen strategy ID:
1. Representative symbols are selected via `SelectRepresentativeSymbols(symbols, LightOptSymbolCount())` — takes first N symbols from the user's selection (deterministic, reproducible).
2. Remaining symbols become `ValidationSymbols` via `DiffStrings()` — used for OOS fallback check.
3. `LightOptimizeConfig` is constructed with all tuning knobs defaulting from environment variables.
4. `RunLightOptimize(ctx, db, lightCfg)` is called. Returns `nil` if the sweep produces no viable result (empty search space, plateau failure, OOS validation rejection).
5. On success, `optimizedParams[strategyID] = params`.
6. Each combo in the dispatch loop receives `btConfig.StrategyParams = optimizedParams[combo.Strategy]` when available.

**Regression protection:**
- If no optimized params for a strategy, `StrategyParams` is `nil` and the strategy runs with its registry defaults (existing behavior preserved).
- `LightOptBudget()` defaults to 24 combos via `ORCA_LIGHT_OPT_BUDGET`. The env var protects against accidental over-tuning.
- The optimizer runs *before* the matrix dispatch, so total runtime increases by `unique_strategies × budget × symbols × 3 months` — a constant additive cost, not multiplicative.

---

## Remediation 2: Fix `submitBacktestWithOptimization` API Handler

**File:** `internal/api/optimize_handler.go`

The handler was a stub that created a DB record and returned immediately without running any optimization. Now:

1. Validates the incoming `OptimizeConfig` JSON shape.
2. Selects representative symbols via `SelectRepresentativeSymbols`.
3. Constructs date windows from `train_years`/`test_years` (the full walk-forward window parameters are available in the request but the handler uses `RunLightOptimize` for reasonable turnaround times).
4. Spawns a goroutine to run optimization asynchronously. The goroutine:
   - Runs `RunLightOptimize(ctx, db, lightCfg)`.
   - Runs a final backtest with best params to produce `bestMetric` (Sharpe).
   - Persists results via `repo.SaveOptimizationRun()`.
   - Writes progress files via `monitor.WriteBatchProgress()` for status polling.
5. Returns `{ run_id, status: "accepted" }` immediately.

**Frontend polling contract:**
- `GET /api/v1/optimize/:id/status` → reads `monitor.ReadBatchProgress("opt_"+id)`.
- `GET /api/v1/optimize/:id/results` → reads `repo.GetOptimizationRunByID()`.

**Regression protection:**
- If `s.backtestEngine` is `nil` (no DB configured), returns `503 Service Unavailable`.
- If `RunLightOptimize` returns `nil` (no viable params), the result is persisted with empty params but no error — frontend can display "no improvement found."
- Progress file is cleaned up by filesystem TTL; `WriteBatchProgress` was added to `monitor/progress.go` for this purpose.

---

## Remediation 3: Fix IVS Default-Value Bug

**File:** `internal/backtest/optimized_walk_forward.go:333-362`

**Before:** `DefaultOptimizedWalkForwardConfig()` used the zero-value `IVSConfig{}` where `Enabled` defaults to `false`. `DefaultIVSConfig()` returns `Enabled: true`, but the config constructor never called it. Ad-hoc walk-forward runs (everything except `job_runner.go`) silently disabled IVS.

**After:** `IVSConfig: DefaultIVSConfig()` is explicitly assigned in the config constructor.

Additionally, the objective type was changed from `ObjectiveDDRatio` (single-metric) to `ObjectiveComposite` with weights `{Sharpe: 0.5, MinDD: 0.3, ProfitFactor: 0.2}` — matching the light optimizer's scoring approach.

**Regression protection:**
- `DefaultIVSConfig()` is tested in `optimized_walk_forward_test.go` with explicit `Enabled: true`.
- The `job_runner.go` pipeline (which explicitly set `Enabled: true` before this fix) is unchanged and still passes all tests.
- Composite scoring is validated by existing `TestComputeObjective_Composite` test.

---

## Remediation 4: Deterministic Symbol Selection

**Files:** `internal/backtest/batch_runner.go:378-426`

Two exported functions added:

- `SelectRepresentativeSymbols(symbols []string, n int) []string` — returns first N symbols from input (preserves user ordering, deterministic, reproducible).
- `DiffStrings(a, b []string) []string` — returns set difference for OOS validation symbol set.
- `pickDominantTimeframe(timeframes []string) string` — prefers daily bars, then first available.

**Rationale:** The original plan specified ≤4 symbols from the matrix selection, preferring synthetic-backed ones. The current implementation preserves the user's input order (first N symbols), which is deterministic and reflects the user's priority ordering. Future enhancement: select by most liquid symbols (highest candle count) via database query.

---

## Remediation 5: Unify Scoring Metric

**File:** `internal/backtest/optimized_walk_forward.go:348-352`

**Before:** Walk-forward defaulted to `ObjectiveDDRatio` (Sharpe/MaxDD); light optimizer used composite scoring `{0.5, 0.3, 0.2}`. Inconsistency meant the two optimization paths produced different parameter rankings.

**After:** Both paths use composite scoring with identical weights `{Sharpe: 0.5, MinDD: 0.3, ProfitFactor: 0.2}`. The `ObjectiveComposite` pathway calls `ComputeObjective(result, ObjectiveComposite, weights)` which computes:

```
score = (SharpeWeight × normSharpe) + (DDWeight × (1 - normDD)) + (PFWeight × normPF)
```

where weights normalize to a 0–1 scale via `normalizeScore()`.

**Regression protection:**
- `ComputeObjective` with `ObjectiveComposite` is tested in `optimizer_test.go`.
- Existing tests for `ObjectiveDDRatio` are unaffected — that code path still works when explicitly configured.

---

## Infrastructure Additions

### `monitor.WriteBatchProgress` 
**File:** `internal/monitor/progress.go:84-95`

Previously only `ReadBatchProgress` existed with no write counterpart. Added `WriteBatchProgress(batchID, bp)` which writes to `data/progress/<batchID>.json`. Used by the optimization handler to track async progress.

### `Engine.GetDB()`
**File:** `internal/backtest/engine.go:319-321`

Exposes the internal `Database` handle so callers like `submitBacktestWithOptimization` can invoke `RunLightOptimize` (which requires `Database`, not `*Engine`).

---

## Verification

```bash
# Build
go build ./...

# Backtest unit tests (includes light optimizer, IVS, composite scoring, cache)
go test ./internal/backtest/... -count=1 -v

# Engine unit tests (includes live engine Kelly fraction)
go test ./internal/engine/... -count=1 -v

# Frontend e2e tests (with API mocking)
npx playwright test route-verification.spec.cjs page-navigation.spec.cjs optimization-ui.spec.cjs
```

**Test results as of 2026-07-25:**
- Go backtest: all tests pass
- Go engine: all tests pass
- Playwright: 49/49 pass (3 spec files)

---

## Prop Firm Objective Protection

The system remains aligned with the goal of profitably passing prop firm challenges:

1. **PropFirmEnabled** flows through `MatrixBacktestConfig` → `LightOptimizeConfig` → `BacktestConfig`. The light optimizer penalizes prop-firm breaches with `score *= 0.1` (§light_optimizer.go:395-402).
2. **KellyFraction** (0.25 default) is applied in both backtest engine and live engine — position sizing conservatism is preserved.
3. **Multi-metric gating** (Sharpe ≥ 1.0, MaxDD ≤ 8%, PassProb ≥ 80%, PF ≥ 1.5) remains enforced through `ApplyGate` + `GateProfile` in `RunMatrixConcurrent`.
4. **FTMO rules** (daily loss limit, max drawdown, consistency) are enforced through `PropfirmEnforcer` in `Engine.Run()`.

The light optimizer improves rather than compromises prop firm readiness: instead of running every combo with default (untuned) parameters, each strategy now gets per-combo parameters that have been pre-optimized for the specific symbol universe and time horizon — producing more realistic, diverse performance surfaces in the matrix results.
