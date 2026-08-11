# Pending Future Improvements — Implementation Plan

**Date:** 2026-08-10
**Status:** In Progress
**Total items:** 20 | **Estimated effort:** ~45h

---

## 1. Execution Order

Items are ordered by (impact × 1/effort) — best bang for the buck first.

### Sprint 1 — Quick Wins (~5h)

| Step | Item | Effort | Description |
|------|------|--------|-------------|
| **1.1** | **P5** — Monte Carlo chart in OrchestrationDetail | 1h | Wire pool daily returns from `result_json.daily_returns` into the existing `MonteCarloChart` component. Fix the "insufficient daily returns" message by ensuring daily returns flow through the API. |
| **1.2** | **P4** — OrchMatrix results panel wiring | 1.5h | Connect `OrchMatrixResultsPanel` to the `orchestrator.list()` API so matrix results are visible after `POST /orchestrator/matrix` completes. Add a "View Matrix Results" section to the OrchestrationRunner that polls for completion and shows the panel. |
| **1.3** | **P13** — Wire VIX acceleration detector | 0.5h | Add `vixDetector.Feed(rawVIX, smoothVIX)` call in orchestrator's `Run()` loop before passing VIX to strategies. The detector bypasses smoothing when `|ΔVIX| > 5.0`. |
| **1.4** | **P9** — Deeper context cancellation | 0.5h | Add `select { case <-ctx.Done(): return nil, ctx.Err() }` inside the entry and exit sub-loops in `Run()`, not just at the bar-level. |
| **1.5** | **P20** — O(1) regime lookup | 1h | Build a `map[int64]int8` during `LoadRegimeLogs` in the orchestrator. Replace the O(n) `getRegimeAt()` linear scan with O(1) map lookup. |
| **1.6** | **P18** — Fix TEXT[] serialization in backtest history repo | 0.5h | Apply the same `stringsToPgArray`/`pgArrayToStrings` fix from `repository_orchestration.go` to `repository_backtest_history.go` for `strategy_ids` columns. |

### Sprint 2 — Core Correctness (~8h)

| Step | Item | Effort | Description |
|------|------|--------|-------------|
| **2.1** | **P1** — Fractional share support in sizing | 1.5h | Add a `AllowFractional` flag to `ProcessSignalRequest`. When set, skip the integer share rounding in `RequestCapital`. Wire through orchestrator config and the `fillQty < 1` guard. Default false to preserve existing behavior. |
| **2.2** | **P10** — Run light optimizer on top-3 combos | 3h | Submit 3 single-strategy matrix backtests with `WirePipeline=true` and `SkipLightOptimize=false` for: grid_trading ES 4h, grid_trading NQ 1h, grid_trading ES 1h. Compare optimized vs default params. Store results in `param_versions` table. |
| **2.3** | **P12** — Re-run matrix with E3 spread model | 1h | Submit a matrix backtest for the top-5 combos with `FrictionModel="realistic"` (E3 per-asset-class spreads). Compare Sharpe vs the v10 pre-E3 results. Document the spread-adjusted Sharpe. |
| **2.4** | **P19** — Route exit signals through RiskPipeline | 1h | Modify `generateSignalForExit()` at `engine.go:1284` to call `e.pipeline.ProcessSignal()` instead of `sr.Evaluate()` directly. Add `SkipCapitalCheck` flag for stop-loss exits that should bypass capital authorization. |
| **2.5** | **P14** — Wire `RecordSlippageObservation` into live engine | 1.5h | Call `engine.RecordSlippageObservation(symbol, expectedPrice, actualPrice)` from the broker fill callback in `live_engine.go` or from `ReconcileLiveFill`. |

### Sprint 3 — UX & Integration (~8h)

| Step | Item | Effort | Description |
|------|------|--------|-------------|
| **3.1** | **P2** — "Promote to Orchestration" from matrix results | 2h | Add a "Promote to Orch" button on each matrix result row (in `MatrixResultsPanel`). On click, switch to Orch mode and pre-fill the strategy row with the combo's strategy, symbol, and timeframe. Store the preselected combo in URL state so it survives mode switches. |
| **3.2** | **P6** — Allocation timeline scrubber | 2h | Add a timeline slider to the `AllocationPie` component. On slider change, re-render the pie chart for the selected timestamp. Fetch allocation data from the API on selection. |
| **3.3** | **P8** — OrchMatrix `batch_id` column + migration 000034 | 1h | Add `batch_id TEXT` column to `orchestration_runs`. Add `GET /orchestrator/runs?batch_id=xxx` query param to `ListRuns`. |
| **3.4** | **P7** — Frontend wiring for OrchMatrix | 3h | When `RunOrchestratorMatrix` is called from the UI, start polling `GET /orchestrator/runs` filtered by batch_id. Display results in `OrchMatrixResultsPanel` as they stream in. Add a "View Matrix" button in the OrchestrationRunner that shows results for the last submitted matrix batch. |

### Sprint 4 — Testing & Hardening (~12h)

| Step | Item | Effort | Description |
|------|------|--------|-------------|
| **4.1** | **P3** — Complete orchestrator core tests (BT9-BT22) | 4h | Multi-strategy run, rebalance triggering, regime gate blocking, VIX-aware strategy, pool halt on drawdown, missing regime/VIX/candle data, derived metrics (success/empty/no-trades), context cancelled, result JSON roundtrip. |
| **4.2** | **P3** — Complete correlation tracker tests (BT34-BT43) | 2h | Perfect positive, uncorrelated, insufficient data, velocity brake sudden jump, velocity brake cooldown, brake release, brake discount, pair matrix, single strategy, exactly two points. |
| **4.3** | **P3** — Complete reevaluator tests (BT44-BT52) | 1.5h | MaxDD breach, Sharpe degradation (30d), Sharpe degradation (60d), Sharpe recovery, regime reentry, healthy active unchanged, violated unchanged, missing benchmark, fill slippage average. |
| **4.4** | **P3** — Complete frontend component tests (FT1-FT83) | 3h | OrchestrationRunner (17 tests), OrchestrationDetail (14 tests), useOrchestrationPoll (8 tests), BacktestHubIntegration (8 tests), StrategyHubStatusTab (15 tests), other components (21 tests). |
| **4.5** | **P3** — API handler tests (BT61-BT93) | 1.5h | Orchestrator handler (22 tests), strategy status handler (11 tests). |

### Sprint 5 — Deep Features (~12h)

| Step | Item | Effort | Description |
|------|------|--------|-------------|
| **5.1** | **P11** — Walk-forward validation on orchestration set | 4h | Run walk-forward backtest for the top orchestration combo set (grid_trading SPX500 4h + rsi2_reversion JPN225 1h) with re-optimization window. Compute OOS Sharpe degradation. Wire the walk-forward framework (`reoptimization.go`) to accept orchestration config. |
| **5.2** | **P15** — Route BatchInferrer through live engine | 3h | Replace the direct `EvaluateSignal()` call in `ProcessTickForAccount` with a call through `BatchInferrer` for caching and threshold skip. Parity with backtest path. |
| **5.3** | **P16** — Wire `feature_store_persist.go` into live engine lifecycle | 2h | Call `Persist()` on engine shutdown and `LoadFeatureStore()` on startup from `live_engine.go`. |
| **5.4** | **P17** — VIX futures data feed integration | 3h | Ingest VIX futures term structure data into `market_ticks` hypertable. Wire into `vix_futures_carry_runner.go` to replace the spot VIX proxy with actual contango/backwardation signal. |

---

## 2. Dependencies

```
Sprint 1 (independent) ──→ Sprint 2 (depends on 1.3, 1.4)
                               ├── Sprint 3 (depends on 2.1)
                               └── Sprint 4 (depends on 2.2, 2.3)
                                    └── Sprint 5 (depends on 2.2, 2.4, all Sprint 4)
```

---

## 3. Verification per Sprint

| Sprint | Gates |
|--------|-------|
| Sprint 1 | `go build ./...` clean, `npx tsc --noEmit` zero errors, `npx vitest run` 233 pass |
| Sprint 2 | Above + `go test ./internal/backtest/... -count=1` all pass, matrix re-run produces results with Sharpe > 1.0 |
| Sprint 3 | Above + UI manual test: promote-to-orch works, allocation timeline scrubs, matrix results stream in |
| Sprint 4 | Above + `go test ./internal/backtest/... ./internal/api/... -count=1` all pass, 80+ new tests |
| Sprint 5 | Above + walk-forward OOS Sharpe within 50% of IS, live engine starts without errors |

---

## 4. Implementation Status

| Sprint | Steps | Status |
|--------|-------|--------|
| Sprint 1 | 1.1–1.6 | **In Progress** |
| Sprint 2 | 2.1–2.5 | Pending |
| Sprint 3 | 3.1–3.4 | Pending |
| Sprint 4 | 4.1–4.5 | Pending |
| Sprint 5 | 5.1–5.4 | Pending |

---

*End of plan.*
