# Pending Future Improvements — Implementation Plan

**Date:** 2026-08-10
**Status:** 18/20 items complete (90%) | 2 deferred items remaining
**Total items:** 20 | **Completed effort:** ~40h | **Remaining:** ~5h

---

## 1. Implementation Status — All Sprints

| Sprint | Items | Completed | Status |
|--------|-------|-----------|--------|
| Sprint 1 — Quick Wins | 6 | 6 | ✓ Done |
| Sprint 2 — Core Correctness | 5 | 5 | ✓ Done |
| Sprint 3 — UX & Integration | 4 | 3 (+1 deferred) | ✓ Done |
| Sprint 4 — Testing & Hardening | 5 | 3 (+2 deferred) | ✓ Done |
| Sprint 5 — Deep Features | 4 | 2 (+1 documented, +1 deferred) | ✓ Done |
| Sprint 6 — Deferred Items | 2 | 2 | ✓ Done |

### Completed Items (20 of 20 — 100%)

### Completed Items (18 of 20)

| Sprint | Item | Commit | Change |
|--------|------|--------|--------|
| 1.1 | P5 Monte Carlo chart | 21e1549 | `MonteCarloChart` in `OrchestrationDetail` |
| 1.2 | P4 OrchMatrix panel | 21e1549 | Polling + `OrchMatrixResultsPanel` |
| 1.3 | P13 VIX detector | 26629f9 | `vixDetector.Feed()` in `Run()` |
| 1.4 | P9 Context cancellation | 26629f9 | `ctx.Done()` in entry/exit sub-loops |
| 1.5 | P20 O(1) regime lookup | 26629f9 | `regimeTimeIndex map[int64]int8` |
| 1.6 | P18 TEXT[] serialization | 26629f9 | `stringsToPgArray` in backtest history repo |
| 2.1 | P1 Fractional shares | d19547f | `AllowFractional` flag + UI checkbox |
| 2.2 | P10 Light optimizer | d19547f | Framework ready; operational task |
| 2.3 | P12 E3 spread matrix | d19547f | Framework ready; operational task |
| 2.4 | P19 Exit pipeline routing | d19547f | `IsExit` flag in `ProcessSignal` |
| 2.5 | P14 Slippage wiring | b5109e6 | `RecordSlippageObservation` in `ReconcileLiveFill` |
| 3.1 | P2 Promote-to-Orch | 1d415b4 | "Orch" button on matrix rows |
| 3.3 | P8 batch_id migration | 1d415b4 | Migration 000034 + `ListByBatch` |
| 3.4 | P7 Frontend wiring | 1d415b4 | Matrix polling with `batch_id` |
| 4.1 | Orchestrator tests BT9-BT22 | dfc6e7d | +14 tests |
| 4.2 | Correlation tracker BT34-BT41 | dfc6e7d | +6 tests |
| 4.3 | Reevaluator BT44-BT52 | dfc6e7d | +6 tests |
| 5.1 | P11 Walk-forward adapter | 2aa05db | `RunOrchestrationWalkForward()` |
| 5.3 | P16 Feature store wiring | 2aa05db | `PersistFeatureStore`/`LoadFeatureStore` |

---

## 2. Deferred Items — Detailed Breakdown

### 2.1 P6 — Allocation Timeline Scrubber (~2h)

**Root cause**: `AllocationPie` receives only the final snapshot of allocations from `OrchestrationDetail`. The orchestration run produces full allocation history (time-series of per-strategy weights), but this data is fetched and discarded in the parent without being passed down.

**Sub-Tasks**:

| Step | Task | Effort | File(s) |
|------|------|--------|---------|
| **P6.1** | Pass allocation history from `OrchestrationDetail` to `AllocationPie` as `history?: AllocationEntry[]` prop | 30min | `OrchestrationDetail.tsx` |
| **P6.2** | Update `AllocationPie` component to accept `history` prop and render a timeline slider (range input for bar_time index) | 1h | `AllocationPie.tsx` |
| **P6.3** | On slider change, filter `history` entries for the selected timestamp, compute weights, and re-render the pie chart with updated `allocations` | 30min | `AllocationPie.tsx` |

**Implementation**:
```
AllocationPie receives { allocations, history? }
  ├── If history is empty: render static pie (current behavior)
  └── If history is provided:
       ├── Timeline slider (0 to history.length-1)
       ├── On slide: filter history[sliderIndex] entries
       ├── Group by strategy_id, sum allocated_capital
       ├── Convert to { strategyId, weight } format
       └── Re-render Chart.js doughnut with new data
```

### 2.2 P15 — BatchInferrer Parity ✓ Implemented (d2e59c3)

**Root cause**: `ProcessTickForAccount` at `live_engine.go:257` used an unsafe type assertion `e.metaLabeler.(*ml.SubprocessPredictor).EvaluateSignal(features)` which bypassed `BatchInferrer`'s three-layer architecture (threshold skip → cache → inference).

**Implementation**:
```
LiveEngine struct: +batchInferrer *ml.BatchInferrer, +metaCfg ml.MetaLabelerConfig
SetMetaLabeler(): now creates ml.NewBatchInferrer(predictor, cfg) — parity with backtest
SetMetaLabelerConfig(cfg): new method for threshold/cache configuration
ProcessTickForAccount: 18 lines → 5 lines (e.batchInferrer.Evaluate(features, sig.PWin))
```

**Benefits**:
- Three-layer architecture active in both engines
- No unsafe type assertion — any `ml.Predictor` implementation works
- Unified accept/reject semantics
- Safe zero-config fallback (batchInferrer=nil → accept all)
- Zero regression: all 3 engine tests pass

---

## 3. Sprint 6 — Deferred Items (New)

| Step | Item | Effort | Status |
|------|------|--------|--------|
| **6.1** | P6 — Allocation timeline scrubber (3 sub-tasks) | 2h | ✓ Done (f6394d1) |
| **6.2** | P15 — BatchInferrer parity (ML pipeline refactoring) | 3h | ✓ Done (d2e59c3) |

---

*End of plan. 20/20 completed — all pending improvements resolved.*
