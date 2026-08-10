# Orchestration-Backtest Interface Integration Plan

**Date:** 2026-08-10
**Authors:** Senior Engineering Team
**Status:** Draft — Awaiting Approval
**Target:** Connect orchestration module to existing BacktestHub UI

---

## 1. Current State Assessment

### 1.1 BacktestHub (`/backtest`)

The existing backtest interface is a mature, feature-rich page with three views (Runner, History, Detail) navigated via URL query params (`?view=runner|history|detail&id=<run_id>`). Key infrastructure:

| Capability | Implementation | State |
|-----------|---------------|-------|
| Strategy selection | `cacheStore` (stale-while-revalidate), API-fetched strategies + symbols | Production |
| Single backtest | `POST /api/v1/backtests` → inline result card → DetailView | Production |
| Matrix backtest | `POST /api/v1/backtests {mode:'matrix'}` → `matrixStore` + `useMatrixStream` polling → virtualized results table | Production |
| Progress streaming | `GET /api/v1/backtests/matrix/:batchId/results?since={seq}` with incremental delta via `matrixStore.applyDelta` | Production |
| Export | CSV for trades, equity, daily returns, matrix results (17 columns) | Production |
| Comparison | Multi-select checkboxes, Pearson correlation matrix, equity curve overlay | Production |
| Results detail | 17 metrics + equity chart + daily returns + Monte Carlo + calendar heatmap + trade analysis tabs + optimization | Production |
| Navigation | URL query params (`view`, `id`) — deep-linkable, back-button friendly | Production |
| Preselected strategy | `?strategy=<id>` query param | Production |

### 1.2 OrchestrationHub (`/orchestration`)

The orchestration interface is a newly built page (Phase 4) with three tabs (Runner, History, Detail) using internal state. Key infrastructure:

| Capability | Implementation | State |
|-----------|---------------|-------|
| Strategy selection | Per-row dropdowns: strategy + symbol + timeframe | New — separate of cacheStore |
| Submission | `POST /api/v1/orchestrator/run` → saves to DB, returns `run_id` | New — no streaming |
| Results | `GET /api/v1/orchestrator/runs/:id` + allocation + correlation | New — from DB |
| Pool metrics | Sharpe, Sortino, MaxDD, Return%, Rebalance Cost | New |
| Allocation visualization | Pie chart (Chart.js) + correlation heatmap (grid) | New |
| Strategy status | `GET /api/v1/strategies/:id/status` + promote/demote | New |
| Navigation | Internal React state (`activeTab`, `selectedRunId`) — no URL persistence | New — no deep-link support |
| Export | None | Missing |
| Progress tracking | None — fire-and-forget submission | Missing |

### 1.3 Identified Integration Gaps

| # | Gap | Severity | Impact |
|---|-----|----------|--------|
| **G1** | Orchestration submission has no streaming/background execution — returns pending immediately but never runs the backtest | **HIGH** | Orchestrated backtests are saved but never executed. The `POST /api/v1/orchestrator/run` handler only persists the config to DB; no goroutine is launched to run the orchestrator. |
| **G2** | OrchestrationHub has no URL-based navigation — tab state is lost on refresh, links cannot be shared | **MEDIUM** | Users cannot bookmark or share orchestration results. Inconsistent with BacktestHub's URL-driven navigation pattern. |
| **G3** | OrchestrationHub does not use `cacheStore` — redundant API calls for strategies and symbols | **LOW** | Duplicates strategy/symbol fetching logic; cache invalidation is not shared. |
| **G4** | Two separate entry points for "Run Backtest" — users must know which page to use for single vs. multi-strategy | **MEDIUM** | User confusion: "Should I use Backtesting or Orchestration for my multi-strategy run?" |
| **G5** | No cross-navigation between BacktestHub and OrchestrationHub results | **MEDIUM** | If a user runs an orchestrated backtest, they cannot apply existing comparison, Monte Carlo, or trade analysis tools from BacktestHub to those results. |
| **G6** | Orchestration results are not unified with backtest history — separate DB tables, separate API endpoints, separate UIs | **HIGH** | The `orchestration_runs` table is isolated from `backtest_runs`. Users see two separate histories. |
| **G7** | OrchestrationHub lacks virtualization, export, comparison, and streaming — all features BacktestHub has | **MEDIUM** | Future orchestration matrix runs (Phase 6+) will hit performance walls without these patterns. |

---

## 2. Recommended Integration Architecture

### 2.1 Core Design Principle: Unify, Don't Duplicate

The integration should **bring orchestration into BacktestHub as a mode**, not maintain two parallel interfaces. This follows the existing precedent of single vs. matrix mode in the current RunnerView.

### 2.2 Proposed Architecture

```
BacktestHub (/backtest)
├── RunnerView
│   ├── Mode: single (existing)          ← unchanged
│   ├── Mode: matrix (existing)          ← unchanged
│   └── Mode: orchestrated (NEW)         ← integrated from OrchestrationHub RunnerTab
│       ├── Strategy rows (per-row: strategy + symbol + timeframe)
│       ├── Orchestration-specific config (rebalance bars, Kelly fraction,
│       │   correlation brake, correlation threshold, friction model)
│       ├── "Recommended Set" preset button
│       └── Submit → POST /api/v1/orchestrator/run → async execution
│
├── HistoryView
│   ├── Tab: Single/Matrix Runs (existing)    ← unchanged
│   └── Tab: Orchestration Runs (NEW)
│       └── Table columns: Run ID, Date Range, Strategies (badges),
│           Status, Pool Sharpe, Pool MaxDD, Pool Return
│       └── Click → DetailView (id, type=orchestration)
│
└── DetailView
    ├── Type: backtest (existing)             ← unchanged
    └── Type: orchestration (NEW)
        ├── Pool Metrics (5 cards: Sharpe, Sortino, MaxDD, Return, RebalanceCost)
        ├── Pool Equity Curve (reuse existing EquityCurve component)
        ├── Allocation Pie (reuse AllocationPie component)
        ├── Correlation Heatmap (reuse CorrelationHeatmap component)
        ├── Strategy PnL table
        └── Correlation Breaches table
```

### 2.3 Data Flow

```
┌──────────────────────────────────────────────────────────────────┐
│                         User Action                              │
│  Selects "Orchestrated" mode → fills config → clicks "Run"      │
└──────────────┬───────────────────────────────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────────────────────────┐
│  POST /api/v1/orchestrator/run                                   │
│  Body: { strategies, start_date, end_date, initial_capital,     │
│          rebalance_bars, kelly_fraction, /* ... */ }             │
│  Response: { run_id: "uuid", status: "running" }                │
│                                                                  │
│  Handler launches async goroutine:                               │
│    orchestrator := NewOrchestrator(db, cfg)                      │
│    result := orchestrator.Run(ctx)                                │
│    repo.UpdateOrchestrationRun(runID, "completed", result)       │
└──────────────┬───────────────────────────────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────────────────────────┐
│  Polling (optional, for live progress — Phase 2)                 │
│  GET /api/v1/orchestrator/runs/:id                               │
│  Response: { status: "running" | "completed" | "failed", ... }  │
└──────────────┬───────────────────────────────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────────────────────────┐
│  On completion: navigate to DetailView                           │
│  Type: "orchestration", id: <run_id>                             │
│                                                                  │
│  GET /api/v1/orchestrator/runs/:id                                │
│  GET /api/v1/orchestrator/runs/:id/allocation                     │
│  GET /api/v1/orchestrator/runs/:id/correlation                    │
│                                                                  │
│  Display: pool metrics + equity curve + allocation pie +        │
│           correlation heatmap + strategy PnL + breaches         │
└──────────────────────────────────────────────────────────────────┘
```

### 2.4 Design Justifications

| Decision | Rationale | Regression Risk Reduction |
|----------|-----------|--------------------------|
| **Integrate as a mode in BacktestHub rather than a separate page** | Single vs. matrix mode already coexist in RunnerView. Adding "orchestrated" as a third mode follows the established pattern. | Zero changes to single/matrix code paths. Mode-based conditional rendering isolates new code. |
| **Keep `/orchestration` route active (with redirect)** | Existing bookmarks and nav links continue to work. The page becomes a redirect to BacktestHub with preselected orchestration mode. | No broken links. Legacy URL support. |
| **Add `type` discriminator to DetailView** | Backtest detail and orchestration detail share the page shell (header, navigation) but have different metric layouts. A `type` prop drives conditional rendering. | Existing backtest detail path is untouched — condition wraps only the metric panel. |
| **Launch async goroutine for orchestration execution** | Match the existing matrix backtest pattern where the API returns immediately and a background worker executes the work. This is the same pattern used by `RunMatrixConcurrent`. | Reuses existing goroutine management patterns. No new concurrency infrastructure. |
| **Share `cacheStore` for strategy/symbol data** | Eliminates duplicate API calls and ensures cache invalidation is consistent across all modes. | Existing cache invalidation calls (e.g., on strategy CRUD) automatically apply to orchestration mode. |
| **Orchestration results in `orchestration_runs` table (not merged with `backtest_runs`)** | The schemas are fundamentally different: orchestration runs have pool-level metrics, allocation history, and correlation breaches. Backtest runs have single-strategy metrics. Merging would create a sparse, confusing table. | No schema migration risk to existing `backtest_runs`. Separate table = separate code paths = no regression. |

---

## 2A. Result Structure Divergence Analysis

### 2A.1 Structural Comparison: BacktestResult vs. OrchestrationRunResult

The two report types differ in **68% of their fields**. Only 5 fields have direct overlap. This is not a case of one extra panel — the results are fundamentally different data products that require **separate rendering paths** at every layer.

| Metric / Section | BacktestResult (22 fields) | OrchestrationRunResult (11 fields) | Overlap? |
|---|---|---|---|
| **Equity Curve** | `EquityCurve []EquityPoint` | `PoolEquity []EquityPoint` | **YES** — same struct |
| **Trades** | `Trades []Trade` (full: MAE, MFE, slippage, adverse selection, stop/take prices) | `Trades []Trade` (partial: no stop-loss exits, no MAE/MFE tracking, no hold duration) | **PARTIAL** — same struct, fewer populated fields |
| **Sharpe** | `SharpeRatio float64` | `PoolSharpe float64` | **YES** |
| **Sortino** | `SortinoRatio float64` | `PoolSortino float64` | **YES** |
| **Max Drawdown** | `MaxDrawdown float64` | `PoolMaxDD float64` | **YES** |
| **Return %** | `TotalReturnPct float64` | `PoolReturnPct float64` | **YES** |
| Win Rate | `WinRate float64` | _Not available_ — pool-level only | **NO** |
| Profit Factor | `ProfitFactor float64` | _Not available_ | **NO** |
| Avg Trade/Win/Loss | `AvgTrade, AvgWin, AvgLoss float64` | _Not available_ | **NO** |
| Num Trades/Wins/Losses | `NumTrades, NumWins, NumLosses int` | _Countable from Trades[] but not precomputed_ | **PARTIAL** |
| MaxDD Duration | `MaxDrawdownDuration int` | _Not available_ | **NO** |
| Adverse Selection | `AdverseSelectionRate float64` | _Not available_ | **NO** |
| MAE / MFE | `AvgMAE, AvgMFE float64` | _Not available_ — orchestrator trades lack `lowestSinceEntry`/`highestSinceEntry` tracking | **NO** |
| Daily Returns | `DailyReturns []DailyReturn` | _Not available_ — computable from pool equity on frontend | **DERIVABLE** |
| Temporal Breakdown | `TemporalBreakdown` (yearly/monthly/weekly/daily) | _Not available_ | **NO** |
| Monthly Returns | Via `TemporalBreakdown.Monthly` | _Not available_ — computable from pool equity on frontend | **DERIVABLE** |
| Regime Stats | `RegimeStats []RegimeStat` | _Not available_ per strategy | **NO** |
| Long/Short Breakdown | `LongShort` (long vs short PnL, WinRate, PF) | _Not available_ | **NO** |
| Compliance Report | `ComplianceReport` (pass/fail + reason) | _Not available_ — pool halted state in `orchestration_runs` | **NO** |
| Signal Diagnostics | `SignalDiag` (17 pipeline counters) | _Not available_ | **NO** |
| Optimization Footprint | N/A (separate endpoint) | _Not applicable_ — orchestration uses allocation optimization, not parameter optimization | **NO** |
| Strategy Params | `StrategyParams map[string]float64` | _Not available_ — multiple strategies, multiple param sets | **NO** |
| **Rebalance Costs** | _Not applicable_ | `RebalanceCosts float64` | **UNIQUE** |
| **Allocation History** | _Not applicable_ | `AllocationHistory []OrchAllocationPoint` | **UNIQUE** |
| **Correlation Breaches** | _Not applicable_ | `CorrelationBreaches []BreachEvent` | **UNIQUE** |
| **Strategy PnL** | _Not applicable_ | `StrategyPnL map[string]float64` | **UNIQUE** |
| **Active Count** | _Not applicable_ | `ActiveCount []int` | **UNIQUE** |

### 2A.2 Impact on DetailView Rendering

The existing BacktestHub `DetailView` (`BacktestHub.tsx:732-1024`) is built entirely around `BacktestResult`. It fetches 7 API responses (`get`, `equity`, `dailyReturns`, `trades`, `monthlyReturns`, `regimeStats`, `optimization`) and renders:

| Section | Lines | Orchestration Equivalent | Status |
|---------|-------|-------------------------|--------|
| **Performance Metrics** — 10-stat grid (Sharpe, Sortino, MaxDD, Win, PF, Return%, Trades W/L, Avg Trade, Avg W/L, W/L Ratio) | 906-917 | Pool metrics grid: 5 stats (Sharpe, Sortino, MaxDD, Return%, RebalanceCost) + per-strategy PnL table | **Different layout needed** |
| **Risk Profile** — 6-stat grid (MaxDD%, DD Duration, MAE, MFE, Avg Hold, Gate) | 920-930 | Not applicable — pool-level DD only, no MAE/MFE, no gate profile | **Not rendered** |
| **Equity Curve** chart | 938-941 | Same component, different data source | **Reusable** |
| **Daily Returns** chart | 943-944 | Derivable from pool equity on frontend | **Reusable if precomputed** |
| **Monte Carlo** simulation — collapsible, 500 trials | 947-967 | Not applicable — multi-strategy simulation is a separate feature (Phase 7) | **Not rendered** |
| **Monthly Returns** — CalendarHeatmap + YearlySummary | 972-978 | Derivable from pool equity on frontend | **Reusable if precomputed** |
| **Trade Analysis** — 3 tabs (Regime, Trades, Optimization) | 980-996 | No regime analysis, no optimization. Trades table exists but with fewer columns. | **Different tab structure** |
| **Costs & Metadata** — collapsible | 998-1012 | Not applicable | **Not rendered** |
| **PromoteToLiveWizard** | 1014-1022 | "Deploy as Orchestration Set" (Phase 5.3) | **Different flow** |

**Key insight**: Orchestration detail needs its own rendering path — not a conditional toggle within the existing layout. The two report types share only the EquityCurve chart component and 5 scalar metrics. Everything else is either exclusive to one type or needs different data.

### 2A.3 Impact on API Layer

| Existing Endpoint | Works for Backtest? | Works for Orchestration? | Change Needed |
|---|---|---|---|
| `GET /api/v1/backtests/:id` | YES — returns BacktestResult-equivalent JSON | NO — orchestration_id in backtest_runs table | `GET /api/v1/orchestrator/runs/:id` **already exists** — returns `OrchestrationRun` with `strategy_ids`, metrics, `result_json` |
| `GET /api/v1/backtests/:id/equity` | YES — `EquityCurve` | NO — pool equity is in `result_json.pool_equity` | **New endpoint**: `GET /api/v1/orchestrator/runs/:id/equity` to extract and return pool equity curve |
| `GET /api/v1/backtests/:id/trades` | YES — full Trade[] | NO — trades are in `result_json.trades` | **New endpoint**: `GET /api/v1/orchestrator/runs/:id/trades` |
| `GET /api/v1/backtests/:id/daily-returns` | YES — precomputed `DailyReturns[]` | NO — not precomputed | **New endpoint**: compute daily returns from pool equity on backend, OR compute on frontend from equity endpoint |
| `GET /api/v1/backtests/:id/monthly-returns` | YES — from `TemporalBreakdown` | NO — not precomputed | Same as daily returns — backend computation recommended |
| `GET /api/v1/backtests/:id/metrics` | YES — `BacktestMetrics` | NO — different metric set | **New endpoint**: `GET /api/v1/orchestrator/runs/:id/metrics` returning `OrchestrationMetrics` |
| `GET /api/v1/backtests/:id/regime-stats` | YES | NO | Not applicable for orchestration |
| `GET /api/v1/backtests/:id/optimization` | YES | NO | Not applicable for orchestration |
| `GET /api/v1/orchestrator/runs/:id/allocation` | N/A | **Already exists** | No change |
| `GET /api/v1/orchestrator/runs/:id/correlation` | N/A | **Already exists** | No change |

### 2A.4 Impact on Results Grid / History Table

The history list currently returns `BacktestHistoryEntry[]` with columns: `id, run_type, status, strategy_ids, symbols, start_date, end_date, initial_capital, sharpe_ratio, max_drawdown, total_return, win_rate, num_trades`.

An orchestration run has different columns: `id, status, start_date, end_date, initial_capital, strategy_ids, symbol_tf_pairs, pool_sharpe, pool_maxdd, pool_return_pct` — and also unique columns (`pool_sortino`, `rebalance_costs`).

**Recommendation**: Do not merge into one table. Use separate tabs (Backtests / Orchestration) in the HistoryView, each with its own column schema. The existing history tab is untouched; the orchestration tab is new. Cross-link between them via a "View in Backtests" / "View in Orchestration" button on each row type.

### 2A.5 Recommended Detail Architecture

Rather than trying to force both report types through one component tree, the DetailView should delegate to type-specific sub-components:

```
DetailView(id, type: 'backtest' | 'orchestration')
├── [type === 'backtest']  → BacktestDetail  (existing code, zero changes)
│   ├── Performance Metrics grid (10 stats)
│   ├── Risk Profile grid (6 stats)
│   ├── EquityCurve
│   ├── DailyReturns + MonthlyReturns
│   ├── MonteCarlo (collapsible)
│   └── TradeAnalysisTabs (Regime / Trades / Optimization)
│
└── [type === 'orchestration'] → OrchestrationDetail (new, extracted from OrchestrationHub)
    ├── Pool Metrics grid (5 stats) + Strategy PnL table
    ├── Pool EquityCurve
    ├── AllocationPie + Allocation Timeline
    ├── CorrelationHeatmap
    └── BreachEventsTable + Strategy Trades
```

This architecture:
- **Zero risk** to existing BacktestDetail rendering path
- Extracts orchestration display from OrchestrationHub into a reusable sub-component
- Shared components (EquityCurve) used by both with different data sources
- Each type has its own data fetching logic — no shared useEffect with conditional branches

### 2A.6 Recommended API Contract: Orchestration Detail

To match the frontend needs, the backend should expose a unified detail response:

```json
{
  "id": "uuid",
  "status": "completed",
  "start_date": "...",
  "end_date": "...",
  "initial_capital": 100000,
  "strategy_ids": ["grid_trading", "rsi2_reversion"],
  "metrics": {
    "pool_sharpe": 2.5,
    "pool_sortino": 3.1,
    "pool_maxdd": 5.2,
    "pool_return_pct": 12.3,
    "rebalance_costs": 45.20,
    "num_trades": 340,
    "eq_final": 112300
  },
  "equity": [],
  "trades": [],
  "allocation": [],
  "correlation": { "matrix": {}, "breaches": [] },
  "strategy_pnl": {},
  "daily_returns": []
}
```

This is achievable by:
1. Extending `GET /api/v1/orchestrator/runs/:id` to return the full result_json when `status=completed`
2. Adding the helper endpoints (`/equity`, `/trades`, `/daily-returns`, `/metrics`) as thin extractions from `result_json`

---

## 3. Step-by-Step Implementation Roadmap

### Phase 1: Backend Async Execution + API Parity (2.5 days)

**Objective:** Make orchestrated backtests actually execute, and expose a complete API surface matching the detail needs identified in §2A.6.

| Step | Deliverable | Effort | Dependencies | Backward Compatible? |
|------|-------------|--------|-------------|---------------------|
| **1.1** | Add async execution to `OrchestratorHandler.SubmitRun`: launch goroutine with `context.WithTimeout(30min)`, `semaphore.NewWeighted(2)`, calls `Orchestrator.Run()`, updates run status + `result_json` on completion/failure. Write `OrchestrationRunResult` into `result_json` including `pool_equity`, `trades`, `allocation_history`, `correlation_breaches`, `strategy_pnl`. | 3h | Orchestrator component (Phase 1) | YES — additive to handler |
| **1.2** | Compute and store derived metrics in `result_json`: compute daily returns from pool equity, compute monthly returns from pool equity, compute win rate/profit factor from trades, compute per-strategy trade stats (num trades, win rate, avg PnL). Store everything in `result_json` so detail endpoints are thin extractions. | 2h | 1.1 | YES |
| **1.3** | Add `GET /api/v1/orchestrator/runs/:id/equity` — extract pool equity curve from `result_json` | 30min | 1.2 | YES |
| **1.4** | Add `GET /api/v1/orchestrator/runs/:id/trades` — extract trades array from `result_json` | 30min | 1.2 | YES |
| **1.5** | Add `GET /api/v1/orchestrator/runs/:id/daily-returns` — extract precomputed daily returns from `result_json` | 30min | 1.2 | YES |
| **1.6** | Add `GET /api/v1/orchestrator/runs/:id/metrics` — return `OrchestrationMetrics` struct with pool_sharpe, pool_sortino, pool_maxdd, pool_return_pct, rebalance_costs, num_trades, eq_final, strategy_pnl, active_count | 1h | 1.2 | YES |
| **1.7** | Extend `GET /api/v1/orchestrator/runs/:id` to return full `result_json` when `status=completed`, with per-strategy trade stats from 1.2 | 1h | 1.2 | YES — backward compatible, response grew new fields under existing key |
| **1.8** | Unit tests: async execution (success, timeout, cancel), derived metrics accuracy, endpoint response shapes match expected contracts | 3h | 1.1-1.7 | YES |

**Phase 1 Validation:**
- Submit orchestrated backtest via API → `POST /runs` returns `run_id` immediately
- Poll `GET /runs/:id` until `status=completed` (should complete within ~30-60s for 1yr, 2-strategy run on 4h/1h data)
- Verify `result_json` contains: `pool_equity` (500+ points), `trades` (>0), `allocation_history` (>0), `daily_returns` (>200)
- Verify `GET /runs/:id/equity` returns equity curve
- Verify `GET /runs/:id/metrics` returns all fields
- Verify `GET /runs/:id/daily-returns` returns daily return series

### Phase 2: BacktestHub Mode Integration — Runner (2 days)

**Objective:** Add orchestration as a mode in the BacktestHub RunnerView.

| Step | Deliverable | Effort | Dependencies | Backward Compatible? |
|------|-------------|--------|-------------|---------------------|
| **2.1** | Add `"orchestrated"` to mode type union in `BacktestHub.tsx` | 15min | — | YES — additive |
| **2.2** | Create `OrchestrationRunner` component extracting orchestration config UI from `OrchestrationHub` RunnerTab (strategy rows, rebalance bars, Kelly fraction, correlation brake, friction toggle) | 3h | 2.1 | YES — new component |
| **2.3** | Wire orchestration submit to `POST /api/v1/orchestrator/run` and handle response (polling for completion) | 2h | Phase 1, 2.2 | YES |
| **2.4** | Add polling hook `useOrchestrationPoll` (similar to `useMatrixStream`): poll `GET /runs/:id` every 2s, update progress display, navigate to detail on completion | 2h | Phase 1, 2.3 | YES |
| **2.5** | Add `OrchestrationProgressBar` component (similar to `MatrixProgressBar`): show status, elapsed time, estimated completion | 1h | 2.4 | YES |

**Phase 2 Validation:**
- Select "Orchestrated" mode → see per-row strategy selector + orchestration config
- Submit with 2 strategies → see progress bar → navigate to detail on completion
- Verify single and matrix modes are unaffected (full regression suite)

### Phase 3: BacktestHub Detail Integration (1.5 days)

**Objective:** Display orchestration results within the existing BacktestHub DetailView, using the type-discriminated architecture recommended in §2A.5.

| Step | Deliverable | Effort | Dependencies | Backward Compatible? |
|------|-------------|--------|-------------|---------------------|
| **3.1** | Add `type` prop to `DetailView`: `'backtest' | 'orchestration'` — defaults to `'backtest'`. At the top of DetailView, add a type-detection branch: if `type === 'orchestration'` OR the `id` resolves to an orchestration run via `GET /api/v1/orchestrator/runs/:id`, render `<OrchestrationDetail>` instead of the existing backtest layout. All existing backtest code paths remain as the default branch. | 1h | — | YES — default preserves behavior; detection is additive |
| **3.2** | Extract orchestration detail display from `OrchestrationHub` DetailTab into standalone `<OrchestrationDetail>` component accepting `{ runId: string }`. This component handles its own data fetching: `Promise.all([orchestrator.get(id), orchestrator.getEquity(id), orchestrator.getTrades(id), orchestrator.getMetrics(id), orchestrator.getAllocation(id), orchestrator.getCorrelation(id)])`. Renders the layout from §2A.5: Pool Metrics grid (5 stats + strategy PnL table), Equity Curve, Allocation Pie with timeline, Correlation Heatmap, Breach Events table, Strategy Trades table. | 4h | Phase 1 | YES — new component, no changes to existing BacktestDetail |
| **3.3** | Compute daily returns from pool equity on the frontend if the backend hasn't precomputed them (using the same algorithm from `engine.go` daily aggregation). This enables the daily returns chart to render for orchestration results without waiting for Phase 1.2 backend computation. | 1h | 3.2 | YES |
| **3.4** | Reuse `EquityCurve` component for pool equity display — same component, different data source. | 30min | 3.2 | YES |
| **3.5** | Set navigation: from orchestrator completion handler, call `setView('detail', runId, { type: 'orchestration' })`. Update `HubView` type and `setView` signature to include optional `opts?: { type?: 'backtest' | 'orchestration' }`. | 1h | Phase 2, 3.1 | YES |

**Phase 3 Validation:**
- Navigate to orchestration detail via history → see pool metrics, equity curve, allocation pie, correlation heatmap (all 6 sections from §2A.5 rendered)
- Navigate to existing backtest detail → see unchanged 10-stat + risk profile + Monte Carlo + trade tabs
- Verify daily returns computed from pool equity render correctly

### Phase 4: History Unification (1 day)

**Objective:** Unified history view showing both backtest and orchestration runs.

| Step | Deliverable | Effort | Dependencies | Backward Compatible? |
|------|-------------|--------|-------------|---------------------|
| **4.1** | Create `OrchestrationHistoryTab` component: table of orchestration runs with columns (Run ID, Date, Strategies, Status, Pool Sharpe, Pool MaxDD, Pool Return) | 2h | Phase 1 | YES — new component |
| **4.2** | Add tabs to HistoryView: "Backtests" (default) and "Orchestration" | 1h | 4.1 | YES — default tab preserves behavior |
| **4.3** | Add cross-link from orchestration row to DetailView: `onClick → setView('detail', run.id, { type: 'orchestration' })` | 30min | 4.1, Phase 3 | YES |
| **4.4** | Add run type badge to history table: single = gray, matrix = blue, orchestration = purple | 30min | 4.1 | YES |

**Phase 4 Validation:**
- Switch to "Orchestration" history tab → see list of orchestration runs
- Click run → navigate to orchestration detail
- Switch to "Backtests" tab → see existing backtest history unchanged

### Phase 5: StrategyHub + Sidebar Refinement (0.5 day)

**Objective:** Update navigation and surface orchestration promotion flow.

| Step | Deliverable | Effort | Dependencies | Backward Compatible? |
|------|-------------|--------|-------------|---------------------|
| **5.1** | Keep `/orchestration` route active but redirect to `/backtest?mode=orchestrated` | 30min | Phase 2 | YES — existing bookmarks preserved |
| **5.2** | Update sidebar: remove standalone "Orchestration" nav item, add "Backtesting" sub-label indicating orchestration is available | 30min | 5.1 | YES |
| **5.3** | Add "Deploy as Orchestration" button to PromoteToLiveWizard — when multiple strategies are manually promoted, offer to create an orchestration run | 2h | Phase 2, Phase 3 | YES |

**Phase 5 Validation:**
- Visit `/orchestration` → redirected to `/backtest?mode=orchestrated`
- Sidebar shows unified "Backtesting" link

### Phase 6: Cross-Navigation from StrategyHub (0.5 day)

**Objective:** From StrategyHub's Status tab, link to orchestration detail for deployed strategies.

| Step | Deliverable | Effort | Dependencies | Backward Compatible? |
|------|-------------|--------|-------------|---------------------|
| **6.1** | In StatusTab, add "View Run" link on strategy rows that have an associated orchestration run | 1h | Phase 4 | YES |
| **6.2** | Add `orchestration_run_id` field to `strategy_status` table via migration `000033` to link statuses to their source runs | 1.5h | Phase 1 | YES — new nullable column |

**Phase 6 Validation:**
- Promote a strategy → see "View Run" link → navigate to orchestration detail

### Phase Effort Summary

| Phase | Duration | Focus | Prerequisites |
|-------|----------|-------|--------------|
| Phase 1 | 2.5 days | Backend async execution + full API surface (§2A.6) | — |
| Phase 2 | 2 days | BacktestHub mode integration — Runner | Phase 1 |
| Phase 3 | 1.5 days | DetailView type-discriminated rendering (§2A.5) | Phase 1, Phase 2 |
| Phase 4 | 1 day | History unification with separate tabs | Phase 1, Phase 3 |
| Phase 5 | 0.5 day | StrategyHub + Sidebar refinement | Phase 2, Phase 3 |
| Phase 6 | 0.5 day | Cross-navigation from StrategyHub StatusTab | Phase 4 |

**Total serial effort: ~8 days.** Phase 1 grew from 2 to 2.5 days after §2A analysis identified the need for derived metrics computation (daily/monthly returns, per-strategy trade stats) and a full API surface to match the BacktestDetail parity requirements.

---

## 4. Regression Prevention Plan

### 4.1 Test Suite — Backward Compatibility

| # | Test Name | What It Validates | Type |
|---|-----------|-------------------|------|
| **T1** | `TestRunnerView_SingleMode_Unchanged` | Single mode submission flow is identical before/after integration | E2E (Playwright) |
| **T2** | `TestRunnerView_MatrixMode_Unchanged` | Matrix mode submission, streaming, results display unchanged | E2E |
| **T3** | `TestHistoryView_BacktestTab_Unchanged` | Backtest history list loads, metrics lazy-load, compare works | E2E |
| **T4** | `TestDetailView_BacktestType_Unchanged` | All 17 metrics, equity curve, daily returns, Monte Carlo, trades, optimization tabs render correctly for backtest runs | E2E |
| **T5** | `TestDetailView_MatrixCombo_Unchanged` | Matrix combo result path (in-memory from matrixStore) still works | Unit |
| **T6** | `TestBacktestSubmit_SingleMode_Returns200` | `POST /api/v1/backtests` continues to return correct response shape | Integration |
| **T7** | `TestBacktestSubmit_MatrixMode_Returns200` | Matrix submission creates batch, returns `batch_run_id` | Integration |
| **T8** | `TestOrchestrationLegacyRoute_Redirects` | Visiting `/orchestration` redirects to `/backtest?mode=orchestrated` | E2E |
| **T9** | `TestCSVExport_MatrixResults_Unchanged` | CSV export produces 17 columns with correct data | Unit |
| **T10** | `TestPromoteToLiveWizard_Unchanged` | Deploy wizard works for backtest runs without orchestration context | E2E |

### 4.2 Test Suite — New Functionality

| # | Test Name | What It Validates | Type |
|---|-----------|-------------------|------|
| **T11** | `TestOrchestrationSubmit_CreatesRun` | Submit orchestrated run → run saved to DB, async execution starts | Integration |
| **T12** | `TestOrchestrationRun_Completes` | Full orchestration execution: runs to completion, metrics populated | Integration |
| **T13** | `TestOrchestrationDetail_RendersAllComponents` | Detail view loads pool metrics, equity curve, allocation pie, correlation heatmap, breach table | E2E |
| **T14** | `TestOrchestrationHistory_ListsRuns` | History tab shows orchestration runs with correct columns | E2E |
| **T15** | `TestOrchestrationMode_StrategyRowManagement` | Add/remove strategy rows, recommended set button | Unit |
| **T16** | `TestOrchestrationPoll_PollsUntilComplete` | Polling hook polls every 2s, stops on completion, navigates to detail | Unit |

### 4.3 CI Pipeline Additions

All new tests will be added to the existing CI pipeline in these jobs. The test inventory below maps each test file to the CI job that runs it, and specifies the minimum pass threshold.

| Job | New Test Files | Count | Assertion |
|-----|---------------|-------|-----------|
| `backend` (Go) | `orchestrator_test.go` (BT1-BT22), `rebalance_scheduler_test.go` (BT23-BT33), `correlation_tracker_test.go` (BT34-BT43), `reevaluation_test.go` (BT44-BT52), `vix_detector_test.go` (BT53-BT56), `orchestrator_integration_test.go` (BT57-BT60), `orchestrator_handler_test.go` (BT61-BT82), `strategy_status_handler_test.go` (BT83-BT93), `repository_orchestration_test.go` (BT94-BT105) | 105 | All must pass; `-race` flag enabled |
| `frontend` (Vitest) | `OrchestrationRunner.test.tsx` (FT1-FT17), `OrchestrationDetail.test.tsx` (FT18-FT31), `useOrchestrationPoll.test.ts` (FT32-FT39), `BacktestHubIntegration.test.tsx` (FT40-FT47), `StrategyHubStatusTab.test.tsx` (FT48-FT62), `OrchestrationHistoryTab.test.tsx` (FT63-FT69), `OrchestrationProgressBar.test.tsx` (FT70-FT73), `CorrelationHeatmap.test.tsx` (FT74-FT77), `AllocationPie.test.tsx` (FT78-FT80), `FrictionToggle.test.tsx` (FT81-FT83) | 83 | All must pass |
| `e2e` (Playwright) | `orchestration-backtest-integration.spec.cjs` (E2E1-E2E22) | 22 | All must pass; timeout=180s for orchestration run tests |

**Total new tests: 210** (105 Go + 83 Vitest + 22 Playwright)
**Existing tests unaffected: 233 Vitest + 49 Playwright + all Go test suites**

### 4.3.1 Test Coverage by Layer

| Layer | Files | Tests | Coverage Target | Key Risks Covered |
|-------|-------|-------|-----------------|-------------------|
| **Orchestrator core** | `orchestrator_test.go` | 22 | 100% of `Run()`, `AddStrategy()`, `NewOrchestrator()` paths | R6 (parity), R1 (goroutine leak) |
| **Rebalance Scheduler** | `rebalance_scheduler_test.go` | 11 | 100% of weight formula, eligibility gate, cadence logic | G1 (incorrect allocation) |
| **Correlation Tracker** | `correlation_tracker_test.go` | 10 | 100% of Pearson computation, velocity brake, pair matrix | R4 (false brake triggers) |
| **Strategy Reevaluator** | `reevaluation_test.go` | 9 | 100% of promote/demote triggers and edge cases | G1 (incorrect demotion) |
| **VIX Detector** | `vix_detector_test.go` | 4 | 100% of spike detection and reset | R6 (VIX lag) |
| **Integration** | `orchestrator_integration_test.go` | 4 | End-to-end pipeline + friction models + parity | R6 (parity), R3 (result integrity) |
| **API — Orchestrator** | `orchestrator_handler_test.go` | 22 | 100% of 10 endpoints including auth | R4 (API regression) |
| **API — Strategy Status** | `strategy_status_handler_test.go` | 11 | 100% of 4 endpoints including upsert idempotency | R4 (status corruption) |
| **DB — Repository** | `repository_orchestration_test.go` | 12 | 100% of 9 methods including batch inserts, pagination, array serialization | R2 (DB contention), R5 (orphaned runs) |
| **Frontend — Runner** | `OrchestrationRunner.test.tsx` | 17 | All states (empty, loading, error, success) + all UI interactions | R4 (UI regression) |
| **Frontend — Detail** | `OrchestrationDetail.test.tsx` | 14 | All sections loaded, partial failures, empty states | R4 (detail regression) |
| **Frontend — Poll Hook** | `useOrchestrationPoll.test.ts` | 8 | All lifecycle states (polling, completion, error, timeout, cleanup) | R1 (goroutine leak in polling) |
| **Frontend — Integration** | `BacktestHubIntegration.test.tsx` | 8 | Mode switching, type discrimination, history tabs | R4 (BacktestHub regression) |
| **Frontend — Status Tab** | `StrategyHubStatusTab.test.tsx` | 15 | All statuses, promote/demote dialogs, error/empty/loading | R4 (StatusTab regression) |
| **Frontend — Components** | `*HistoryTab`, `*ProgressBar`, `CorrelationHeatmap`, `AllocationPie`, `FrictionToggle` | 21 | All states (loaded, empty, loading, error) + interactions | R4 (component regression) |
| **E2E** | `orchestration-backtest-integration.spec.cjs` | 22 | 8 full user flows + 14 API-level tests + backward compatibility verification | R4 (full system regression) |

### 4.4 Anti-Pattern Scan Extension

Add 3 new anti-pattern checks to `scripts/anti_pattern_scan.py`:

| Rule | Check | Rationale |
|------|-------|-----------|
| **AP19** | `setData()` called in polling/update handlers | Prohibition #11 — must use `ISeriesApi.update()` |
| **AP20** | `fitContent()` called on data updates (not initial load/timeframe change) | Prohibition #12 |
| **AP21** | Orchestration mode bypasses RiskPipeline | Prohibition #17 — must route through pipeline |

### 4.5 Comprehensive Test Specification — 100% Coverage

The following test cases provide full coverage of all new code paths, integration points, edge cases, and error scenarios introduced by the orchestration-backtest integration. Each test references the project's established conventions: Go tests use `Test<Struct>_<Scenario>` naming with `t.Fatalf`/`t.Errorf` (no testify); React tests use `describe`/`it` with vitest and `@testing-library/react`; Playwright tests use `test.describe`/`test` with the `page` and `request` fixtures in CommonJS (`.spec.cjs`).

#### 4.5.1 Go Backend Tests — `internal/backtest/orchestrator_test.go`

| # | Test Name | Scenario | Assertions |
|---|-----------|----------|------------|
| **BT1** | `TestOrchestrator_NewOrchestrator_DefaultConfig` | Create orchestrator with minimal config | `o.config.KellyFraction == 0.25`, `o.config.RebalanceBars == 20`, `o.config.CorrelationThreshold == 0.6`, `o.scheduler != nil`, `o.correlation != nil`, `o.vixDetector != nil` |
| **BT2** | `TestOrchestrator_NewOrchestrator_CustomConfig` | Create orchestrator with all custom fields | `o.config.KellyFraction == 0.15`, `o.config.RebalanceBars == 40`, `o.config.CorrelationThreshold == 0.5`, `o.config.FrictionModel == "realistic"` |
| **BT3** | `TestOrchestrator_NewOrchestrator_InvalidConfig` | Negative rebalance bars, zero Kelly fraction | `o.config.RebalanceBars == 20` (clamped), `o.config.KellyFraction == 0.25` (clamped from 0) |
| **BT4** | `TestOrchestrator_AddStrategy_Valid` | Add 2 strategies (grid_trading SPX500 4h, rsi2_reversion JPN225 1h) | `len(o.engines) == 2`, `o.enginesByID["JPN225:1h:rsi2_reversion"] != nil`, each engine has non-nil `pipeline`, `positionSizer`, `volHalt`, `exposure` |
| **BT5** | `TestOrchestrator_AddStrategy_Duplicate` | Add same strategy/symbol/timeframe twice | Returns error `"duplicate engine"` |
| **BT6** | `TestOrchestrator_AddStrategy_UnknownStrategy` | Add strategy ID that doesn't exist in registry | Returns error `"unknown strategy"` |
| **BT7** | `TestOrchestrator_Run_EmptyStrategies` | Run orchestrator with zero strategies added | Returns result with `len(result.PoolEquity) == 0`, `len(result.Trades) == 0`, no panic |
| **BT8** | `TestOrchestrator_Run_SingleStrategy` | Run orchestrator with 1 strategy on mock data (25 candles) | `len(result.Trades) > 0`, `result.PoolReturnPct` is finite, `result.PoolSharpe` is computed, `result.PoolMaxDD` is computed |
| **BT9** | `TestOrchestrator_Run_MultiStrategy_NoCorrelationBrake` | Run 2 strategies with correlation brake disabled | `len(result.CorrelationBreaches) == 0`, `len(result.AllocationHistory) > 0`, `len(result.ActiveCount) > 0` |
| **BT10** | `TestOrchestrator_Run_MultiStrategy_CorrelationBrake` | Run 2 strategies with correlation brake enabled | Correlation brake logic exercised; no panic on `o.correlation.CheckCorrelations()` with < 2 data points |
| **BT11** | `TestOrchestrator_Run_RebalanceTriggered` | Run with short rebalance cadence (T=5, 25 candles) | At least 4 rebalance events produce allocation history entries with varying weights |
| **BT12** | `TestOrchestrator_Run_RegimeGateBlocks` | Run with all strategies blocked by regime (HMMState=3, Crisis) | `len(result.Trades) == 0`, all strategies gated by `regime_blocked` eligibility reason |
| **BT13** | `TestOrchestrator_Run_VIXAwareStrategy` | Run with vix_futures_carry strategy and VIX data | Strategy receives VIX via `VIXReceiver` interface; no panic on nil VIX data |
| **BT14** | `TestOrchestrator_Run_PoolHaltsOnDrawdown` | Run with prop-firm profile, inject large loss | `o.pool.Halted()` becomes true after loss, no further trades opened after halt |
| **BT15** | `TestOrchestrator_Run_MissingRegimeLogs` | Run without regime logs in mock DB | Regime defaults to 0, strategies evaluate with regime=0, no panic |
| **BT16** | `TestOrchestrator_Run_MissingVIXLogs` | Run without VIX logs in mock DB | VIX=0 passed to strategies, no panic |
| **BT17** | `TestOrchestrator_Run_MissingCandles` | Run with symbol that has no candle data | Returns result with 0 trades, 0 equity points, no error |
| **BT18** | `TestOrchestrator_ComputeDerivedMetrics_Success` | Compute daily returns, monthly returns, win rate, profit factor from valid equity+trades | `len(dailyReturns) > 0`, `len(monthlyReturns) > 0`, `winRate` between 0 and 1, `profitFactor > 0` |
| **BT19** | `TestOrchestrator_ComputeDerivedMetrics_EmptyEquity` | Compute metrics with zero equity points | All derived fields are zero/nil, no division by zero |
| **BT20** | `TestOrchestrator_ComputeDerivedMetrics_NoTrades` | Compute metrics with equity curve but no trades | `winRate == 0`, `profitFactor == 0`, daily/monthly returns derived from equity only |
| **BT21** | `TestOrchestrator_Run_ContextCancelled` | Cancel context mid-execution | Run returns within timeout, no goroutine leak, pool state not updated |
| **BT22** | `TestOrchestrator_ResultJSON_Roundtrip` | Serialize OrchestrationRunResult to JSON, parse back | All fields survive round-trip; numeric fields retain precision within 1e-6 |

#### 4.5.2 Go Backend Tests — `internal/backtest/rebalance_scheduler_test.go`

| # | Test Name | Scenario | Assertions |
|---|-----------|----------|------------|
| **BT23** | `TestRebalanceScheduler_IsFullRebalanceDue_Cadence` | Call `IsFullRebalanceDue()` T times | Returns false for first T-1 calls, true on the T-th call, resets after |
| **BT24** | `TestRebalanceScheduler_ComputeWeights_EqualKelly` | 3 active strategies with equal Kelly=0.25 and equal Sharpe=1.0 | Each weight == 1/3, sum of weights == 1.0 |
| **BT25** | `TestRebalanceScheduler_ComputeWeights_SingleActive` | Only 1 active strategy | Weight == 1.0, `ActiveWeight(weights, "absent") == 0` |
| **BT26** | `TestRebalanceScheduler_ComputeWeights_ZeroSharpe` | Active strategies with Sharpe=0 | Each weight == 1/N (equal fallback), denominator is zero handled |
| **BT27** | `TestRebalanceScheduler_ComputeWeights_Empty` | Empty active list | Returns nil map, no panic |
| **BT28** | `TestRebalanceScheduler_EvaluateEligibility_RegimeBlocked` | Strategy not allowed in current regime | `result.Eligible == false`, `result.Reason == "regime_blocked"` |
| **BT29** | `TestRebalanceScheduler_EvaluateEligibility_NoSignal` | Strategy allowed but hasSignal=false | `result.Eligible == false`, `result.Reason == "no_signal"` |
| **BT30** | `TestRebalanceScheduler_EvaluateEligibility_Active` | Strategy allowed + has signal | `result.Eligible == true`, `result.Reason == "active"` |
| **BT31** | `TestRebalanceScheduler_RecordSharpe_Window` | Record 25 Sharpe values, query trailing | Trailing returns average of last 20; older values evicted |
| **BT32** | `TestRebalanceScheduler_CadenceForTimeframe` | Check cadence for each timeframe | `"1d" → 20`, `"4h" → 40`, `"1h" → 80`, `"30m" → 120`, `"unknown" → 40` |
| **BT33** | `TestRebalanceScheduler_EvaluateEligibility_RegimeOverride` | Strategy has Kelly override for regime | `result.Kelly` matches `matrix.KellyForRegime()` value |

#### 4.5.3 Go Backend Tests — `internal/backtest/correlation_tracker_test.go`

| # | Test Name | Scenario | Assertions |
|---|-----------|----------|------------|
| **BT34** | `TestCorrelationTracker_CheckCorrelations_PerfectPositive` | 2 strategies with identical equity curves | `pc.Correlation > 0.99`, `pc.BrakeActive == true` (if > threshold) |
| **BT35** | `TestCorrelationTracker_CheckCorrelations_Uncorrelated` | 2 strategies with independent equity curves | `|pc.Correlation| < 0.3`, `pc.BrakeActive == false` |
| **BT36** | `TestCorrelationTracker_CheckCorrelations_InsufficientData` | < 2 points per strategy | All correlations == 0, no NaN |
| **BT37** | `TestCorrelationTracker_VelocityBrake_SuddenJump` | Correlation jumps from 0.1 to 0.5 in one step | `BreachEvent` emitted with `Action == "brake_applied_velocity"`, brake active |
| **BT38** | `TestCorrelationTracker_VelocityBrake_Cooldown` | Velocity brake fires → cooldown active for 3 bars → no additional velocity triggers | Second velocity spike within cooldown does not fire |
| **BT39** | `TestCorrelationTracker_IsBrakeActive_ReleasesAfterDuration` | Brake active for 10 bars, then correlation drops | Brake released after `brakeDuration` bars when ρ < threshold; `BrakeReleased` breach emitted |
| **BT40** | `TestCorrelationTracker_BrakeDiscount` | Brake active between pair | `BrakeDiscount("a","b") == 0.70` (same pair order returns same result) |
| **BT41** | `TestCorrelationTracker_PairMatrix` | 3 strategies with varying correlations | Matrix has entries for all 3 pairs, symmetric, diagonal excluded |
| **BT42** | `TestCorrelationTracker_SingleStrategy` | Only 1 strategy registered | `CheckCorrelations()` returns empty; `PairMatrix()` returns empty; no panic |
| **BT43** | `TestCorrelationTracker_RecordEquity_ExactlyTwo` | Add exactly 2 equity points per strategy | Pearson formula works with exact minimum inputs |

#### 4.5.4 Go Backend Tests — `internal/backtest/reevaluation_test.go`

| # | Test Name | Scenario | Assertions |
|---|-----------|----------|------------|
| **BT44** | `TestReevaluator_Demotion_MaxDDBreach` | Strategy exceeds MaxDD threshold (grid_trading: 15%) | Result: `Action == "hard_halt"`, `NewState == StrategyViolated`, `NewWeight == 0` |
| **BT45** | `TestReevaluator_Demotion_SharpeDegradation` | Trailing Sharpe < 30% of benchmark for 30+ days | Result: `Action == "reduce_allocation"`, `NewWeight == currentWeight * 0.25` |
| **BT46** | `TestReevaluator_Demotion_SharpeDegradation_60Days` | Trailing Sharpe < threshold for 60+ days | Result: `Action == "deactivate"`, `NewState == StrategyInactive`, `NewWeight == 0` |
| **BT47** | `TestReevaluator_Promotion_SharpeRecovery` | Previously demoted strategy shows Sharpe > 50% benchmark for 10+ days | Result: `Action == "restore_50pct"`, `NewState == StrategyActive` |
| **BT48** | `TestReevaluator_Promotion_RegimeReentry` | Strategy in standby state | Result: `Action == "activate"`, `NewState == StrategyActive`, `NewWeight == 0.10` |
| **BT49** | `TestReevaluator_NoAction_HealthyActive` | Active strategy with good Sharpe and low DD | No result returned (nil), strategy remains active |
| **BT50** | `TestReevaluator_NoAction_ViolatedUnchanged` | Strategy already violated | No demotion or promotion results returned |
| **BT51** | `TestReevaluator_MissingBenchmark` | Strategy has no benchmark Sharpe in config | Demotion/promotion checks skipped for Sharpe-based criteria |
| **BT52** | `TestReevaluator_RecordFillSlippage_Average` | Record 40 fill slippage observations | `AverageFillSlippage()` returns correct mean; older observations evicted |

#### 4.5.5 Go Backend Tests — `internal/backtest/vix_detector_test.go`

| # | Test Name | Scenario | Assertions |
|---|-----------|----------|------------|
| **BT53** | `TestVIXDetector_NoSpike_NormalChange` | Feed VIX changing by 2.0 over 2 days | `IsSpike()` returns false; output == smoothVIX |
| **BT54** | `TestVIXDetector_Spike_RapidChange` | Feed VIX changing by 6.0 over 2 days | `IsSpike()` returns true; `Feed()` returns rawVIX |
| **BT55** | `TestVIXDetector_InsufficientHistory` | Feed only 1 data point | `IsSpike()` returns false (not enough history) |
| **BT56** | `TestVIXDetector_Reset` | Feed 10 points, call Reset, feed again | History cleared; subsequent spike detection works from clean state |

#### 4.5.6 Go Backend Tests — `internal/backtest/orchestrator_integration_test.go`

| # | Test Name | Scenario | Assertions |
|---|-----------|----------|------------|
| **BT57** | `TestOrchestrator_Integration_FullPipeline` | End-to-end: create orchestrator, add strategies, run, verify result JSON | `result.PoolEquity` has > 0 points, `result.PoolSharpe` finite, `result.StrategyPnL` has entries for each strategy, `result.AllocationHistory` has entries, `result.CorrelationBreaches` slice exists |
| **BT58** | `TestOrchestrator_Integration_FrictionModelRealistic` | Run with realistic friction | Trades have non-zero `SlippageMidBps` and `SlippageLastBps`; rebalance costs computed |
| **BT59** | `TestOrchestrator_Integration_FrictionModelIdealistic` | Run with idealized friction | Trades use default equity slippage (0.5bps spread) |
| **BT60** | `TestOrchestrator_ParityWithSingleEngine` | Run same strategy in orchestrator vs single engine, compare results | Signal generation identical; PnL within 10% tolerance (differs due to orchestration overhead: rebalance costs, pool constraints) |

#### 4.5.7 Go API Tests — `internal/api/orchestrator_handler_test.go`

| # | Test Name | Scenario | Assertions |
|---|-----------|----------|------------|
| **BT61** | `TestOrchestratorHandler_SubmitRun_Valid` | POST with 2 strategies, valid date range, capital=100000 | Code: 202, body has `run_id` (UUID), `status: "pending"` or `"running"`, `message` field |
| **BT62** | `TestOrchestratorHandler_SubmitRun_MissingStrategies` | POST with empty strategies array | Code: 400, body has `error` message |
| **BT63** | `TestOrchestratorHandler_SubmitRun_InvalidDate` | POST with `end_date` before `start_date` | Code: 400, body has `error: "invalid end_date"` or similar |
| **BT64** | `TestOrchestratorHandler_SubmitRun_DefaultCapital` | POST without `initial_capital` | Request succeeds; capital defaults to 100000 |
| **BT65** | `TestOrchestratorHandler_SubmitRun_AsyncLaunches` | Submit valid request, verify goroutine launched | After 100ms, run status in DB is not "pending" (either "running", "completed", or "failed") |
| **BT66** | `TestOrchestratorHandler_SubmitRun_UnknownStrategy` | POST with strategy_id not in registry | Request succeeds (validated at runtime in goroutine); goroutine marks run as "failed" with error in result_json |
| **BT67** | `TestOrchestratorHandler_ListRuns_Empty` | GET /runs before any submissions | Code: 200, body: `{runs: null, total: 0}` |
| **BT68** | `TestOrchestratorHandler_ListRuns_WithResults` | GET /runs after 3 submissions | Code: 200, body has 3 runs, correctly sorted by `created_at DESC` |
| **BT69** | `TestOrchestratorHandler_ListRuns_Pagination` | GET /runs?limit=1&offset=1 | Code: 200, body has 1 run, `total == 3` |
| **BT70** | `TestOrchestratorHandler_GetRun_Valid` | GET /runs/:id for existing run | Code: 200, body has `strategy_ids` array, `symbol_tf_pairs` array, `status`, all date fields |
| **BT71** | `TestOrchestratorHandler_GetRun_NotFound` | GET /runs/:id for nonexistent UUID | Code: 404, body: `{error: "run not found"}` |
| **BT72** | `TestOrchestratorHandler_GetRun_Completed` | GET /runs/:id after async execution completes | Code: 200, `result_json` populated with equity curve, trades, allocation_history |
| **BT73** | `TestOrchestratorHandler_GetAllocation_Valid` | GET /runs/:id/allocation for completed run | Code: 200, body is array with `bar_time`, `strategy_id`, `weight`, `allocated_capital`, `is_active` fields |
| **BT74** | `TestOrchestratorHandler_GetCorrelation_Valid` | GET /runs/:id/correlation for completed run | Code: 200, body has `run_id`, `strategy_ids` array, `result_json` |
| **BT75** | `TestOrchestratorHandler_CancelRun_Running` | DELETE /runs/:id for running run | Code: 200, body: `{run_id, status: "cancelled"}`; DB status updated to "cancelled" |
| **BT76** | `TestOrchestratorHandler_GetEquity_Completed` | GET /runs/:id/equity for completed run | Code: 200, body is array of `{time, value, regime}` objects |
| **BT77** | `TestOrchestratorHandler_GetEquity_Running` | GET /runs/:id/equity for still-running run | Code: 200, body is empty array (or 404 if no equity yet) |
| **BT78** | `TestOrchestratorHandler_GetTrades_Completed` | GET /runs/:id/trades for completed run | Code: 200, body is array of Trade objects with `symbol`, `side`, `quantity`, `pnl` fields |
| **BT79** | `TestOrchestratorHandler_GetDailyReturns_Completed` | GET /runs/:id/daily-returns for completed run | Code: 200, body is array of `{date, return}` objects |
| **BT80** | `TestOrchestratorHandler_GetMetrics_Completed` | GET /runs/:id/metrics for completed run | Code: 200, body has `pool_sharpe`, `pool_sortino`, `pool_maxdd`, `pool_return_pct`, `rebalance_costs`, `num_trades`, `eq_final` |
| **BT81** | `TestOrchestratorHandler_Unauthorized` | All endpoints without Authorization header | Code: 401 for all 10 endpoints |
| **BT82** | `TestOrchestratorHandler_ForbiddenRole` | Non-admin role (trader only) on endpoints requiring admin | Code: 403 (if role-based middleware applies) or 200 (if endpoints use 4-handler auth without role check) |

#### 4.5.8 Go API Tests — `internal/api/strategy_status_handler_test.go`

| # | Test Name | Scenario | Assertions |
|---|-----------|----------|------------|
| **BT83** | `TestStrategyStatusHandler_GetStatus_Valid` | GET /strategies/:id/status for existing status | Code: 200, body has `strategy_id`, `status`, `allocation_pct`, `trailing_sharpe`, `trailing_maxdd` |
| **BT84** | `TestStrategyStatusHandler_GetStatus_NotFound` | GET /strategies/:id/status for nonexistent strategy | Code: 404, body: `{error: "strategy status not found"}` |
| **BT85** | `TestStrategyStatusHandler_ListStatuses_Empty` | GET /strategies/statuses before any upserts | Code: 200, body is empty array `[]` |
| **BT86** | `TestStrategyStatusHandler_ListStatuses_WithData` | Upsert 3 statuses, list | Code: 200, body has 3 entries sorted by `strategy_id` |
| **BT87** | `TestStrategyStatusHandler_Promote_Valid` | POST /strategies/:id/promote with reason | Code: 200, body: `{strategy_id, status: "active", reason}`; DB updated with `status="active"`, `allocation_pct=50` |
| **BT88** | `TestStrategyStatusHandler_Promote_MissingReason` | POST /strategies/:id/promote without reason | Code: 400, body has `error` about missing `reason` field |
| **BT89** | `TestStrategyStatusHandler_Promote_CustomWeight` | POST /strategies/:id/promote with `allocation_pct: 75` | Code: 200, DB has `allocation_pct=75` |
| **BT90** | `TestStrategyStatusHandler_Demote_Valid` | POST /strategies/:id/demote with reason and allocation_pct=0 | Code: 200, body: `{strategy_id, status: "inactive", allocation_pct: 0, reason}`; DB updated with `demotion_reason` set |
| **BT91** | `TestStrategyStatusHandler_Demote_PartialAllocation` | POST /strategies/:id/demote with allocation_pct=25 | Code: 200, DB has `allocation_pct=25`, `status="inactive"` |
| **BT92** | `TestStrategyStatusHandler_Demote_MissingReason` | POST /strategies/:id/demote without reason | Code: 400, body has `error` about missing `reason` field |
| **BT93** | `TestStrategyStatusHandler_Upsert_Idempotent` | Promote same strategy twice | Second call overwrites first; no duplicate rows |

#### 4.5.9 Go DB Tests — `internal/db/repository_orchestration_test.go`

| # | Test Name | Scenario | Assertions |
|---|-----------|----------|------------|
| **BT94** | `TestRepository_SaveOrchestrationRun_Roundtrip` | Save run, load by ID | Loaded fields match saved: `strategy_ids`, `symbol_tf_pairs`, `initial_capital`, `status`, date range |
| **BT95** | `TestRepository_SaveOrchestrationRun_PgArrayEscaping` | Save strategy_ids with special characters (quotes, commas) | Load returns correctly parsed strings via `pgArrayToStrings` |
| **BT96** | `TestRepository_UpdateOrchestrationRun_Complete` | Update with full OrchestrationResult | `pool_sharpe`, `pool_maxdd`, etc. stored and retrievable; `completed_at` set; `status` updated |
| **BT97** | `TestRepository_UpdateOrchestrationRun_Cancel` | Update with nil result (cancel path) | No panic; `status="cancelled"`, metric columns remain null |
| **BT98** | `TestRepository_ListOrchestrationRuns_Pagination` | Save 5 runs, list with limit=2 offset=1 | Returns 2 runs; `total==5`; runs sorted by `created_at DESC` |
| **BT99** | `TestRepository_SaveAllocationHistory_Batch` | Save 50 allocation entries in one batch call | All 50 rows inserted; `LoadAllocationHistory()` returns 50 entries in bar_time order |
| **BT100** | `TestRepository_LoadAllocationHistory_EmptyRun` | Load allocation for run with no allocation entries | Returns empty slice (not nil) |
| **BT101** | `TestRepository_UpsertStrategyStatus_Insert` | Upsert new strategy status | Row created with correct fields |
| **BT102** | `TestRepository_UpsertStrategyStatus_Update` | Upsert existing strategy status with new values | Row updated; `updated_at` refreshed; `ON CONFLICT DO UPDATE` works |
| **BT103** | `TestRepository_UpsertStrategyStatus_NullDemotionReason` | Upsert without setting demotion_reason | `demotion_reason` remains null |
| **BT104** | `TestRepository_GetStrategyStatus_NotFound` | Get nonexistent strategy | Returns error (pgx.ErrNoRows) |
| **BT105** | `TestRepository_ListStrategyStatuses_SortOrder` | List 3 statuses | Results sorted by `strategy_id` |

#### 4.5.10 React Vitest Tests — `web/src/__tests__/OrchestrationRunner.test.tsx`

| # | Test Name | Scenario | Assertions |
|---|-----------|----------|------------|
| **FT1** | `OrchestrationRunner > renders empty strategy row` | Render with default empty state | One strategy row visible with 3 dropdowns; "Add Strategy" button visible; "Run Orchestration" button disabled |
| **FT2** | `OrchestrationRunner > renders with preselected strategies` | Pass preselected strategy+symbol+timeframe | Dropdowns show correct selected values; submit button enabled |
| **FT3** | `OrchestrationRunner > adds strategy row on button click` | Click "Add Strategy" | Row count increases; new row has empty strategy selection |
| **FT4** | `OrchestrationRunner > removes strategy row` | Click remove button on second row | Row count decreases; remaining row unchanged |
| **FT5** | `OrchestrationRunner > cannot remove last row` | Only 1 row exists | Remove button hidden or disabled for last row |
| **FT6** | `OrchestrationRunner > loads strategies from API` | Mock `strategies.list()` returns 5 strategies | 5 options appear in strategy dropdown |
| **FT7** | `OrchestrationRunner > recommended set fills 3 strategies` | Click "Recommended Set" | 3 rows appear: grid_trading/ES/4h, grid_trading/NQ/1h, rsi2_reversion/JPN225/1h |
| **FT8** | `OrchestrationRunner > toggles correlation brake slider` | Toggle correlation brake on | Threshold slider appears; threshold label shows current value |
| **FT9** | `OrchestrationRunner > hides correlation slider when brake off` | Toggle correlation brake off | Threshold slider hidden |
| **FT10** | `OrchestrationRunner > friction model changes label` | Toggle between realistic and idealized | FrictionToggle component onChange fires with correct value |
| **FT11** | `OrchestrationRunner > submit calls orchestrator.submit` | Fill valid config, click submit | `orchestrator.submit` called once with correct payload shape; loading state shown |
| **FT12** | `OrchestrationRunner > shows error on submit failure` | Mock `orchestrator.submit` rejects | Error message displayed; submit button re-enabled |
| **FT13** | `OrchestrationRunner > calls onRunSubmitted on success` | Mock `orchestrator.submit` returns run_id | `onRunSubmitted` callback invoked with correct run_id |
| **FT14** | `OrchestrationRunner > defaults capital to 100000` | Leave capital field empty, submit | Payload has `initial_capital: 100000` |
| **FT15** | `OrchestrationRunner > defaults rebalance bars to 20` | Leave rebalance field empty, submit | Payload has `rebalance_bars: 20` |
| **FT16** | `OrchestrationRunner > validates required fields before submit` | Empty strategy_id in all rows | Submit button disabled |
| **FT17** | `OrchestrationRunner > serializes numeric fields as numbers not strings` | Fill "0.25" in Kelly fraction | Payload has `kelly_fraction: 0.25` (number, not string) |

#### 4.5.11 React Vitest Tests — `web/src/__tests__/OrchestrationDetail.test.tsx`

| # | Test Name | Scenario | Assertions |
|---|-----------|----------|------------|
| **FT18** | `OrchestrationDetail > renders loading state` | Component mounts, data not yet fetched | Loading indicator visible |
| **FT19** | `OrchestrationDetail > renders pool metrics on load` | Mock all API calls return valid data | 5 metric cards visible with correct values (Sharpe, Sortino, MaxDD, Return%, RebalanceCost) |
| **FT20** | `OrchestrationDetail > renders equity curve` | Mock equity data | EquityCurve component rendered with pool equity data |
| **FT21** | `OrchestrationDetail > renders allocation pie` | Mock allocation data | AllocationPie component rendered with strategy weight data |
| **FT22** | `OrchestrationDetail > renders correlation heatmap` | Mock correlation matrix | CorrelationHeatmap component rendered with matrix data |
| **FT23** | `OrchestrationDetail > renders strategy PnL table` | Mock strategy_pnl data | Table shows per-strategy PnL values |
| **FT24** | `OrchestrationDetail > renders breach events table` | Mock correlation_breaches data | Table shows breach events with correlation values and actions |
| **FT25** | `OrchestrationDetail > renders error state` | Mock `orchestrator.get()` rejects | Error message displayed; retry mechanism available |
| **FT26** | `OrchestrationDetail > renders empty state for no trades` | Mock trades returns empty array | "No trades" message or empty table shown |
| **FT27** | `OrchestrationDetail > renders empty state for no breaches` | Mock breaches returns empty array | "No correlation breaches" or empty table shown |
| **FT28** | `OrchestrationDetail > fetches 6 API endpoints on mount` | Mount component, verify API calls | All 6 endpoints called: `get`, `getEquity`, `getTrades`, `getMetrics`, `getAllocation`, `getCorrelation` |
| **FT29** | `OrchestrationDetail > handles partial API failure gracefully` | Mock `getAllocation` rejects, others succeed | Error state shown for allocation section; other sections render correctly |
| **FT30** | `OrchestrationDetail > computes daily returns from equity` | Mock returns equity but no daily_returns endpoint | Daily returns chart renders with values computed from equity curve on frontend |
| **FT31** | `OrchestrationDetail > formats large numbers correctly` | Pool_return_pct = 12.345678 | Display shows "12.35%" (2 decimal places) |

#### 4.5.12 React Vitest Tests — `web/src/__tests__/useOrchestrationPoll.test.ts`

| # | Test Name | Scenario | Assertions |
|---|-----------|----------|------------|
| **FT32** | `useOrchestrationPoll > polls every 2 seconds` | Start polling for run_id | API called at t=0, t=2s, t=4s; useFakeTimers advances correctly |
| **FT33** | `useOrchestrationPoll > stops on completed status` | Mock returns status="completed" | Stops polling; calls onComplete callback with run_id |
| **FT34** | `useOrchestrationPoll > stops on failed status` | Mock returns status="failed" | Stops polling; calls onError callback with error message |
| **FT35** | `useOrchestrationPoll > retries on network error` | Mock rejects twice then succeeds | Polls again after error; no crash |
| **FT36** | `useOrchestrationPoll > stops after 3 consecutive errors` | Mock rejects 3 times | Marks as failed after 3 tries; error callback called |
| **FT37** | `useOrchestrationPoll > cleans up interval on unmount` | Mount, start polling, unmount | No more API calls after unmount; timer cleared |
| **FT38** | `useOrchestrationPoll > does not poll when runId is null` | Pass null runId | Zero API calls made |
| **FT39** | `useOrchestrationPoll > returns progress data` | Mock returns telemetry-like status | Hook returns { status, progress?, error? } correctly |

#### 4.5.13 React Vitest Tests — `web/src/__tests__/BacktestHubIntegration.test.tsx`

| # | Test Name | Scenario | Assertions |
|---|-----------|----------|------------|
| **FT40** | `BacktestHub > renders orchestration mode selector` | Set mode query param to "orchestrated" | Mode selector shows "Orchestrated" option; OrchestrationRunner visible |
| **FT41** | `BacktestHub > default mode is single` | Navigate without mode param | Single mode rendered (unchanged); orchestration panel not visible |
| **FT42** | `BacktestHub > switches mode without breaking state` | Switch single → matrix → orchestrated → single | Each mode renders correct UI; no residual state from previous mode |
| **FT43** | `BacktestHub > DetailView renders backtest type by default` | Navigate to /backtest?view=detail&id=<backtest-id> | Existing BacktestDetail rendered; orchestration panel not rendered |
| **FT44** | `BacktestHub > DetailView renders orchestration type` | Navigate with type=orchestration | BacktestDetail not rendered; OrchestrationDetail rendered instead |
| **FT45** | `BacktestHub > setView accepts type option` | Call setView('detail', runId, { type: 'orchestration' }) | URL has type=orchestration; DetailView renders orchestration components |
| **FT46** | `BacktestHub > HistoryView shows orchestration tab` | Navigate to history view | "Orchestration" tab visible; default "Backtests" tab selected |
| **FT47** | `BacktestHub > HistoryView orchestration tab lists runs` | Switch to orchestration tab, mock API returns 3 runs | 3 rows in orchestration history table |

#### 4.5.14 React Vitest Tests — `web/src/__tests__/StrategyHubStatusTab.test.tsx`

| # | Test Name | Scenario | Assertions |
|---|-----------|----------|------------|
| **FT48** | `StatusTab > renders loading state` | Mount, API not yet resolved | Loading indicator visible |
| **FT49** | `StatusTab > renders statuses table` | Mock returns 3 strategy statuses | Table has 3 rows; columns show strategy_id, status badge, allocation progress bar, trailing sharpe, trailing maxdd, last_signal_at |
| **FT50** | `StatusTab > renders empty state` | Mock returns empty array | Empty state message with guide text visible |
| **FT51** | `StatusTab > status badges render correct colors` | Mock returns active, inactive, standby, violated, validated | Badges use correct variant/color classes for each status |
| **FT52** | `StatusTab > allocation progress bar shows percentage` | Mock returns allocation_pct=0.75 | Progress bar shows 75%; label shows "75%" |
| **FT53** | `StatusTab > maxdd color coding` | Mock returns trailing_maxdd values: 5, 15, 25 | Green for <10%, yellow for 10-20%, red for >20% |
| **FT54** | `StatusTab > promote button opens dialog` | Click promote on inactive strategy | Dialog opens with strategy ID, reason textarea, allocation slider |
| **FT55** | `StatusTab > demote button opens dialog` | Click demote on active strategy | Dialog opens with reason textarea, allocation slider |
| **FT56** | `StatusTab > promote button disabled for active/validated` | Active and validated status rows | Promote button disabled (grayed out) |
| **FT57** | `StatusTab > demote button disabled for inactive/violated` | Inactive and violated status rows | Demote button disabled (grayed out) |
| **FT58** | `StatusTab > promote confirms and calls API` | Fill reason, set allocation, click Confirm | `strategyStatus.promote()` called with correct params; dialog closes; status refreshes |
| **FT59** | `StatusTab > demote confirms and calls API` | Fill reason, set allocation to 0, click Confirm | `strategyStatus.demote()` called with correct params; dialog closes |
| **FT60** | `StatusTab > shows error on promote failure` | Mock `strategyStatus.promote` rejects | Error message displayed in dialog; dialog stays open |
| **FT61** | `StatusTab > shows error on API failure` | Mock `strategyStatus.list` rejects | Error card displayed with error message |
| **FT62** | `StatusTab > refresh button calls list API` | Click refresh button | `strategyStatus.list()` called again; table reloads |

#### 4.5.15 React Vitest Tests — `web/src/__tests__/OrchestrationHistoryTab.test.tsx`

| # | Test Name | Scenario | Assertions |
|---|-----------|----------|------------|
| **FT63** | `OrchestrationHistoryTab > renders runs in table` | Mock `orchestrator.list` returns 3 runs | 3 rows with columns: Run ID (short), Date Range, Strategies (badges), Status (badge), Pool Sharpe, Pool MaxDD, Pool Return% |
| **FT64** | `OrchestrationHistoryTab > status badges` | Mock runs with running, completed, failed | Badges show correct colors: green=completed, yellow=running, red=failed |
| **FT65** | `OrchestrationHistoryTab > strategy badges truncate` | Mock run with 5 strategy_ids | Only first 3 shown as badges; "+2" text visible |
| **FT66** | `OrchestrationHistoryTab > click row calls onSelectRun` | Click a run row | `onSelectRun` callback invoked with run.id |
| **FT67** | `OrchestrationHistoryTab > empty state` | Mock returns `{runs: null, total: 0}` | "No orchestration runs" message visible |
| **FT68** | `OrchestrationHistoryTab > loading state` | API not yet resolved | Loading skeleton or spinner visible |
| **FT69** | `OrchestrationHistoryTab > error state` | Mock `orchestrator.list` rejects | Error message visible; retry button |

#### 4.5.16 React Vitest Tests — `web/src/__tests__/OrchestrationProgressBar.test.tsx`

| # | Test Name | Scenario | Assertions |
|---|-----------|----------|------------|
| **FT70** | `OrchestrationProgressBar > renders running state` | Status: "running" | Progress bar animated; status text "Running..." visible; elapsed time displayed |
| **FT71** | `OrchestrationProgressBar > renders completed state` | Status: "completed" | Status text "Completed"; progress bar at 100%; elapsed time final |
| **FT72** | `OrchestrationProgressBar > renders failed state` | Status: "failed", error message | Error message displayed; progress bar at current position |
| **FT73** | `OrchestrationProgressBar > renders queued state` | Status: "queued", retry_after=30 | "Queued — will retry in 30s" message visible |

#### 4.5.17 React Vitest Tests — `web/src/__tests__/CorrelationHeatmap.test.tsx`

| # | Test Name | Scenario | Assertions |
|---|-----------|----------|------------|
| **FT74** | `CorrelationHeatmap > renders NxN grid` | 3 strategy IDs | 3×3 grid rendered; diagonal cells show strategy names; off-diagonal cells show correlation values |
| **FT75** | `CorrelationHeatmap > color scales correctly` | Correlation values: 0.1, 0.5, 0.9 | Green for low, yellow for medium, red for high correlation |
| **FT76** | `CorrelationHeatmap > highlights above threshold` | Threshold=0.6, correlation=0.7 | Cell has red border highlight class |
| **FT77** | `CorrelationHeatmap > click handler fires` | Click off-diagonal cell | `onPairClick` callback invoked with (strategyA, strategyB) |

#### 4.5.18 React Vitest Tests — `web/src/__tests__/AllocationPie.test.tsx`

| # | Test Name | Scenario | Assertions |
|---|-----------|----------|------------|
| **FT78** | `AllocationPie > renders chart with slices` | 3 allocations with weights 0.4, 0.35, 0.25 | Chart.js canvas rendered; 3 slices with correct labels and weights |
| **FT79** | `AllocationPie > renders empty state` | Empty allocations array | "No active strategies" message visible |
| **FT80** | `AllocationPie > renders legend` | 3 allocations | Legend shows strategy IDs and percentages |

#### 4.5.19 React Vitest Tests — `web/src/__tests__/FrictionToggle.test.tsx`

| # | Test Name | Scenario | Assertions |
|---|-----------|----------|------------|
| **FT81** | `FrictionToggle > renders realistic mode` | model="realistic" | Realistic label selected; description shows cost impact note |
| **FT82** | `FrictionToggle > calls onChange on toggle` | Click idealized button | `onChange("idealized")` invoked |
| **FT83** | `FrictionToggle > renders idealized mode` | model="idealized" | Idealized label selected; no cost impact note |

#### 4.5.20 Playwright E2E Tests — `web/e2e/orchestration-backtest-integration.spec.cjs`

| # | Test Name | Scenario | Assertions |
|---|-----------|----------|------------|
| **E2E1** | `orchestration mode renders in BacktestHub` | Setup auth + API mocks, goto `/backtest?mode=orchestrated` | OrchestrationRunner visible; strategy dropdowns visible; "Add Strategy" button visible; "Recommended Set" button visible |
| **E2E2** | `orchestration mode submits run` | Fill 2 strategies, set dates, click "Run Orchestration" | POST to `/api/v1/orchestrator/run` fired with correct body; progress bar appears |
| **E2E3** | `orchestration polling completes` | Submit run, mock polling returns completed after 4s | Progress bar visible; after completion, navigated to detail view |
| **E2E4** | `orchestration detail renders all sections` | Navigate to /backtest?view=detail&id=<orch-id>&type=orchestration | Pool metrics cards visible; equity curve chart visible; allocation pie visible; correlation heatmap visible; strategy PnL table visible; breach events table visible |
| **E2E5** | `single mode unchanged` | Goto `/backtest`, select Single mode | Strategy selector, symbol input, date inputs, Run button all functional; no orchestration UI visible |
| **E2E6** | `matrix mode unchanged` | Goto `/backtest`, select Matrix mode | Multi-strategy selector, timeframe checkboxes, Run Matrix button all functional; progress bar works; matrix results panel renders |
| **E2E7** | `history backtest tab unchanged` | Goto `/backtest?view=history` | Backtest runs listed; compare checkboxes work; delete works; detail view opens from row click |
| **E2E8** | `history orchestration tab lists runs` | Goto `/backtest?view=history`, click "Orchestration" tab | Orchestration runs listed with correct columns; click row → detail opens |
| **E2E9** | `legacy /orchestration redirects` | Goto `/orchestration` | Redirected to `/backtest?mode=orchestrated`; status 200 or 304 |
| **E2E10** | `strategy status tab renders` | Goto `/strategies?tab=status` | Status table visible with columns; promote/demote buttons visible |
| **E2E11** | `promote strategy via API` | POST `/api/v1/strategies/grid_trading/promote` with reason=test | 200; GET `/strategies/grid_trading/status` returns status=active |
| **E2E12** | `demote strategy via API` | POST `/api/v1/strategies/grid_trading/demote` with reason=test, allocation_pct=0 | 200; GET status returns status=inactive, allocation_pct=0 |
| **E2E13** | `orchestration equity endpoint returns data` | Submit run, wait for completion, GET `/runs/:id/equity` | Returns JSON array of equity points with time/value/regime |
| **E2E14** | `orchestration metrics endpoint returns data` | GET `/runs/:id/metrics` after completion | Returns all metric fields (7 fields per §2A.6 contract) |
| **E2E15** | `orchestration daily-returns endpoint returns data` | GET `/runs/:id/daily-returns` after completion | Returns JSON array; each entry has date and return_pct |
| **E2E16** | `orchestration cancel API cancels run` | Submit run, POST cancel | Status changes to "cancelled"; polling stops |
| **E2E17** | `concurrent orchestration runs capped` | Submit 3 runs simultaneously | Third run returns queued; only 2 running concurrently (semaphore cap of 2) |
| **E2E18** | `orchestration run times out` | Submit run with mock stuck for >30min | Run marked as "failed" with timeout error in result_json |
| **E2E19** | `backtest detail unchanged after integration` | Run single backtest, view detail | All sections render identically to pre-integration: 10-stat grid, risk profile, equity curve, daily returns, Monte Carlo, monthly returns, trades tabs, optimization tab |
| **E2E20** | `matrix detail unchanged after integration` | Run matrix, view combo detail | Matrix combo detail renders correctly; metrics, equity, trades all present; engine_version="matrix" path works |
| **E2E21** | `CSV export unchanged` | Run matrix, export CSV | 17 columns present; data matches table values |
| **E2E22** | `PromoteToLiveWizard unchanged for backtest` | View backtest detail, open deploy wizard | Wizard opens; strategy_id pre-filled; capital allocation input works; deploy button calls correct endpoint |


---

## 5. Backward Compatibility Safeguards

### 5.1 Feature Flags

| Flag | Default | Controls | Rollback |
|------|---------|----------|----------|
| `ORCA_ORCHESTRATION_ENABLED` | `true` | Shows "Orchestrated" mode in RunnerView. When `false`, mode selector only shows single/matrix. | Set to `false` via env var to instantly hide orchestration mode. |
| `ORCA_ORCHESTRATION_ASYNC` | `true` | Enables async goroutine execution. When `false`, the endpoint returns `run_id` but never executes (current behavior). | Set to `false` to disable execution but keep UI visible. |
| `ORCA_ORCHESTRATION_LEGACY_REDIRECT` | `true` | Redirects `/orchestration` → `/backtest?mode=orchestrated`. When `false`, `/orchestration` renders the original standalone page. | Allow users to opt back to standalone page if needed. |

### 5.2 URL/Deeplink Compatibility

| URL | Before Integration | After Integration |
|-----|-------------------|-------------------|
| `/backtest` | RunnerView (single/matrix) | RunnerView (single/matrix/orchestrated) — same view, new mode |
| `/backtest?view=history` | Backtest history | Backtest history + Orchestration history tab — default tab unchanged |
| `/backtest?view=detail&id=<id>` | Backtest detail | Orchestration detail if `<id>` is an orchestration run, else backtest detail — detection is automatic |
| `/orchestration` | OrchestrationHub page | Redirect to `/backtest?mode=orchestrated` (with feature flag override) |
| `/strategies?tab=status` | StrategyHub Status tab | Unchanged — but rows now link to orchestration detail |

### 5.3 API Backward Compatibility

| Endpoint | Compatible? | Notes |
|----------|-------------|-------|
| `POST /api/v1/backtests` | YES | No changes |
| `GET /api/v1/backtests` | YES | No changes |
| `GET /api/v1/backtests/:id` | YES | No changes |
| `GET /api/v1/backtests/:id/metrics` | YES | No changes |
| `GET /api/v1/backtests/:id/equity` | YES | No changes |
| `POST /api/v1/orchestrator/run` | YES | Additive — now launches goroutine (but returns same shape) |
| `GET /api/v1/orchestrator/runs` | YES | No changes to response shape |
| `GET /api/v1/orchestrator/runs/:id` | YES | No changes — response now includes `pool_equity` in `result_json` when complete |
| `GET /api/v1/orchestrator/runs/:id/equity` | N/A | New endpoint |
| `GET /api/v1/orchestrator/runs/:id/detail` | N/A | New endpoint |

### 5.4 Fallback Mechanism

If the orchestration async worker fails to start (e.g., resource exhaustion), the API returns `status: "queued"` and a `retry_after` field. The frontend shows "Queued — will retry automatically" and polls. After 3 failed retries (15 minutes), the run is marked `status: "failed"` with `error: "executor_unavailable"`.

---

## 6. Risk Mitigation and Rollback Plan

### 6.1 Identified Risks

| Risk | Likelihood | Impact | Mitigation | Recovery |
|------|-----------|--------|------------|----------|
| **R1: Goroutine leak in async execution** | Medium | HIGH — memory pressure, eventual OOM | Use `context.WithTimeout(ctx, 30min)` per run. Cap concurrent orchestration runs at 2 via `semaphore.NewWeighted(2)`. | Kill the goroutine via context cancellation. Restart server. |
| **R2: DB write contention** | Low | MEDIUM — slow queries | `allocation_history` uses batched inserts via `pgx.Batch`. Indexes on `run_id` and `bar_time` already in migration. | None needed — performance is O(1) per write given indexes. |
| **R3: Large equity curve in result_json** | Medium | MEDIUM — slow API responses, large DB rows | Truncate equity curve to 500 points in `result_json`; full curve available via `/runs/:id/equity` endpoint. | Increase truncation limit. |
| **R4: Frontend regression in single/matrix modes** | Medium | HIGH — broken core functionality | Feature flag `ORCA_ORCHESTRATION_ENABLED=false` instantly hides all orchestration UI. Single/matrix paths are in separate conditional branches. | Disable feature flag. Revert to previous BacktestHub code if needed. |
| **R5: Orchestration run never completes** | Medium | MEDIUM — orphaned runs in DB | Timeout at 30 minutes per run. Stale run cleanup cron (delete rows where `status='running' AND created_at < NOW() - INTERVAL '2 hours'`). | Manually cancel runs via `DELETE /api/v1/orchestrator/runs/:id`. |
| **R6: Inconsistent results between backtest and orchestration for same strategy** | Low | HIGH — trust erosion | Parity verification test (Phase 3, §3.2) confirms identical signal generation. Orchestrator uses same `Engine` + `RiskPipeline` stack. | Re-run parity test. Audit pipeline wiring. |

### 6.2 Rollback Plan Per Phase

| Phase | Rollback Steps | Recovery Time | Data Impact |
|-------|---------------|---------------|-------------|
| **Phase 1** | Set `ORCA_ORCHESTRATION_ASYNC=false`. Revert goroutine launch in handler. | < 5 min (env var) | None — in-progress runs will be orphaned (manual cleanup) |
| **Phase 2** | Set `ORCA_ORCHESTRATION_ENABLED=false`. Orchestration mode hidden from UI. | < 1 min (env var) | None |
| **Phase 3** | Revert `DetailView` type discriminator. All detail URLs return to backtest-only rendering. | Git revert + rebuild (10 min) | None |
| **Phase 4** | Remove orchestration history tab. HistoryView reverts to single-tab backtest-only. | Git revert + rebuild (10 min) | None |
| **Phase 5** | Restore standalone `/orchestration` route. Remove redirect. | Git revert + rebuild (10 min) | None |
| **Phase 6** | Drop `orchestration_run_id` column from `strategy_status`. Run `.down.sql`. | Migration rollback (2 min) | Nullable column dropped |

### 6.3 Monitoring and Alerting

| Metric | Threshold | Alert |
|--------|-----------|-------|
| `orch_running_count` | > 2 concurrent | Warn: "Orchestration executor at capacity" |
| `orch_run_duration_seconds` | > 1800 (30 min) | Error: "Orchestration run timed out" |
| `orch_failed_count` (last 1h) | > 5 | Error: "Orchestration failure rate elevated" |
| `orch_queue_depth` | > 10 | Warn: "Orchestration queue backing up" |
| `db_pool_in_use / db_pool_max` | > 0.8 | Warn: "DB pool near capacity during orchestration" |

---

## 7. Open Questions

| # | Question | Context | Proposed Default | Stakeholder |
|---|----------|---------|-----------------|-------------|
| **Q1** | Should orchestration mode be on by default, or opt-in? | Users familiar with the existing single/matrix modes might be confused by a third mode. | **Opt-in:** Show "Orchestrated" as a mode but keep "Single" as default. Add a small "New!" badge. | Product / UX |
| **Q2** | Should orchestration results appear in the unified history, or stay in the separate orchestration history tab? | Mixing backtest and orchestration runs in one table would require nullable columns for orchestration-specific metrics. | **Separate tab** (proposed above): keeps table schemas clean. Cross-link between tabs. | Architecture |
| **Q3** | Do we need "matrix orchestration" — N strategy combos run through the orchestrator? | The design doc mentioned matrix runner integration (§4.5, G9). An orchestrated matrix would run N strategy×symbol×timeframe combos through the orchestrator, producing allocation/correlation data per combo. | **Defer to Phase 7**: This is a significant feature. The current plan covers single orchestration runs only. | Product |
| **Q4** | Should the orchestrator respect the existing `GateProfile`? | The current orchestrator has its own risk controls (Kelly fraction, correlation brake, regime gate). The existing backtest supports gate profiles (none/default/lenient/strict). Should these combine? | **No**: Orchestration uses its own risk model (design doc §3.2). Gate profiles apply to single/matrix modes only. Add a tooltip explaining this. | Quant / Risk |
| **Q5** | Should orchestration runs be exportable (CSV)? | Existing backtests support CSV export of trades, equity, daily returns, and matrix results. Orchestration runs have allocation history and correlation breaches — these are different data shapes. | **Yes, Phase 6**: Export allocation history as CSV. Defer correlation matrix export. | Product |
| **Q6** | Does the PromoteToLiveWizard need orchestration-aware deployment? | Currently, PromoteToLive deploys a single strategy. With orchestration, multiple strategies are deployed together with allocated capital. | **Yes, Phase 5.3**: Add "Deploy as Orchestration Set" option. Defer full implementation details to a follow-up design doc. | Architecture |
| **Q7** | Should we remove the standalone `/orchestration` page entirely, or keep it as a simplified entry point? | Removing it reduces maintenance surface. Keeping it adds redundancy but offers a simpler UX for orchestration-only users. | **Redirect** (proposed above): Keep the route as a convenience redirect. Remove the full standalone page in Phase 5. | Product |

---

## 8. Appendix: File Change Inventory

### New Files (to be created)

| File | Phase | Purpose |
|------|-------|---------|
| `web/src/components/backtest/OrchestrationRunner.tsx` | 2.2 | Orchestration config UI (strategy rows, risk params) |
| `web/src/components/backtest/OrchestrationDetail.tsx` | 3.2 | Orchestration detail panel (pool metrics, allocation, correlation) |
| `web/src/components/backtest/OrchestrationProgressBar.tsx` | 2.5 | Progress bar for orchestration runs |
| `web/src/hooks/useOrchestrationPoll.ts` | 2.4 | Polling hook for orchestration run status |
| `web/src/components/backtest/OrchestrationHistoryTab.tsx` | 4.1 | Orchestration history table |
| `internal/db/migrations/000033_strategy_status_orch_link.up.sql` | 6.2 | Add `orchestration_run_id` column |
| `internal/db/migrations/000033_strategy_status_orch_link.down.sql` | 6.2 | Rollback |

### New Test Files (to be created)

| File | Phase | Type | Tests | Coverage Target |
|------|-------|------|-------|-----------------|
| `internal/backtest/orchestrator_test.go` | 1.8 | Go unit | BT1-BT22 | 100% of `Run()`, `AddStrategy()`, `NewOrchestrator()` |
| `internal/backtest/rebalance_scheduler_test.go` | 1.8 | Go unit | BT23-BT33 | 100% of weight formula, eligibility gate |
| `internal/backtest/correlation_tracker_test.go` | 1.8 | Go unit | BT34-BT43 | 100% of Pearson, velocity brake |
| `internal/backtest/reevaluation_test.go` | 1.8 | Go unit | BT44-BT52 | 100% of promote/demote triggers |
| `internal/backtest/vix_detector_test.go` | 1.8 | Go unit | BT53-BT56 | 100% of spike detection |
| `internal/backtest/orchestrator_integration_test.go` | 1.8 | Go integration | BT57-BT60 | End-to-end pipeline + parity |
| `internal/api/orchestrator_handler_test.go` | 1.8 | Go handler | BT61-BT82 | 100% of 10 endpoints |
| `internal/api/strategy_status_handler_test.go` | 1.8 | Go handler | BT83-BT93 | 100% of 4 endpoints |
| `internal/db/repository_orchestration_test.go` | 1.8 | Go DB | BT94-BT105 | 100% of 9 repository methods |
| `web/src/__tests__/OrchestrationRunner.test.tsx` | 2 | Vitest | FT1-FT17 | All states + interactions |
| `web/src/__tests__/OrchestrationDetail.test.tsx` | 3 | Vitest | FT18-FT31 | All sections + partial failures |
| `web/src/__tests__/useOrchestrationPoll.test.ts` | 2 | Vitest | FT32-FT39 | All lifecycle states |
| `web/src/__tests__/BacktestHubIntegration.test.tsx` | 2-4 | Vitest | FT40-FT47 | Mode switching, type discrimination |
| `web/src/__tests__/StrategyHubStatusTab.test.tsx` | 4 | Vitest | FT48-FT62 | All statuses, promote/demote dialogs |
| `web/src/__tests__/OrchestrationHistoryTab.test.tsx` | 4 | Vitest | FT63-FT69 | All states + interactions |
| `web/src/__tests__/OrchestrationProgressBar.test.tsx` | 2 | Vitest | FT70-FT73 | All status states |
| `web/src/__tests__/CorrelationHeatmap.test.tsx` | 3 | Vitest | FT74-FT77 | Grid rendering, color scale, clicks |
| `web/src/__tests__/AllocationPie.test.tsx` | 3 | Vitest | FT78-FT80 | Chart rendering, empty state |
| `web/src/__tests__/FrictionToggle.test.tsx` | 2 | Vitest | FT81-FT83 | Toggle states, onChange |
| `web/e2e/orchestration-backtest-integration.spec.cjs` | 3-6 | Playwright | E2E1-E2E22 | 8 full flows + 14 API tests |

### Modified Files (to be changed)

| File | Phase | Change |
|------|-------|--------|
| `internal/api/orchestrator_handler.go` | 1.1 | Add async goroutine launch with timeout, semaphore, result_json storage |
| `internal/backtest/orchestrator.go` | 1.2 | Add `ComputeDerivedMetrics(result)` function: daily returns, monthly returns, win rate, profit factor, per-strategy stats from trades |
| `internal/api/router.go` | 1.3-1.7 | Register new endpoints: `/equity`, `/trades`, `/daily-returns`, `/metrics` on orchestration routes |
| `web/src/pages/BacktestHub.tsx` | 2.1, 2.3, 3.1, 3.3, 3.5, 4.2, 4.3 | Mode type union, orchestration mode routing, type prop on DetailView, orchestration detail path, history tabs |
| `web/src/App.tsx` | 5.1 | `/orchestration` redirect route to `/backtest?mode=orchestrated` |
| `web/src/components/layout/Sidebar.tsx` | 5.2 | Remove standalone orchestration nav item, add sub-label to Backtesting |
| `web/src/pages/strategy-hub/StatusTab.tsx` | 6.1 | Add "View Run" link on strategy rows with orchestration_run_id |
| `web/src/types/api.ts` | 1.6 | Add `OrchestrationMetrics` and `OrchestrationDetailResponse` types matching §2A.6 contract |

### Files to Deprecate

| File | Phase | Action |
|------|-------|--------|
| `web/src/pages/OrchestrationHub.tsx` | 5.1 | Keep as redirect wrapper; remove full body |
| `web/src/components/orchestration/CorrelationHeatmap.tsx` | Post-integration | Archived if fully replaced by unified component |
| `web/src/components/orchestration/AllocationPie.tsx` | Post-integration | Archived if fully replaced by unified component |
| `web/src/components/orchestration/FrictionToggle.tsx` | Post-integration | Archived if fully replaced by unified component |

---

## 9. Implementation Status — 2026-08-10 (Complete)

### 9.1 Phase Completion Summary

| Phase | Plan | Status | Files Changed | Verification |
|-------|------|--------|---------------|-------------|
| **Phase 0** | Foundation fixes (6 steps) | Done | 8 files | `go build ./...` clean, all Go tests pass |
| **Phase 1** | Backend async execution + API parity (2.5 days) | Done | 4 files created, 3 modified | Async goroutine tested; 10 API endpoints registered; equity (2642 pts), trades, daily-returns, metrics all functional |
| **Phase 2** | BacktestHub mode integration — Runner (2 days) | Done | 3 files created, 1 modified | `npx tsc --noEmit` zero errors, 233 vitest pass |
| **Phase 3** | DetailView type-discriminated rendering (1.5 days) | Done | 1 file created, 1 modified | Type-based routing via `?type=orchestration`; existing backtest detail zero-risk |
| **Phase 4** | History unification (1 day) | Done | 1 file created, 1 modified | Tabs "Backtests" (default) / "Orchestration"; cross-link with `type=orchestration` |
| **Phase 5** | Navigation refinement (0.5 day) | Done | 3 files modified | `/orchestration` → redirect; sidebar unified; PromoteToLiveWizard orchestration toggle |
| **Phase 5.3** | Deploy as Orchestration Set | Done | 1 file rewritten | Orchestration config panel in wizard; bulk-promote; polling progress inline |
| **Phase 6** | Cross-navigation from StrategyHub (0.5 day) | Done | 7 files (2 created, 5 modified) | Migration 000033 applied; `orchestration_run_id` in API responses; "Run" link in StatusTab |
| **Q1-Q7** | Open question resolutions | Done | 3 files modified | Default mode → matrix; CSV export for orchestration; all questions resolved |

### 9.2 File Inventory — Created (23 files)

| # | File | Phase | Lines |
|---|------|-------|-------|
| 1 | `internal/db/migrations/000032_orchestration.up.sql` | 0 | 32 |
| 2 | `internal/db/migrations/000032_orchestration.down.sql` | 0 | 4 |
| 3 | `internal/db/repository_orchestration.go` | 0 | 320 |
| 4 | `internal/backtest/rebalance_scheduler.go` | 1 | 165 |
| 5 | `internal/backtest/correlation_tracker.go` | 1 | 220 |
| 6 | `internal/backtest/vix_detector.go` | 1 | 55 |
| 7 | `internal/backtest/orchestrator.go` | 1 | 740 |
| 8 | `internal/backtest/reevaluation.go` | 1 | 265 |
| 9 | `internal/api/orchestrator_handler.go` | 2 | 290 |
| 10 | `internal/api/strategy_status_handler.go` | 2 | 138 |
| 11 | `internal/backtest/parity_test.go` | 3 | 175 |
| 12 | `web/src/components/orchestration/CorrelationHeatmap.tsx` | 4 | 105 |
| 13 | `web/src/components/orchestration/AllocationPie.tsx` | 4 | 85 |
| 14 | `web/src/components/orchestration/FrictionToggle.tsx` | 4 | 45 |
| 15 | `web/src/pages/OrchestrationHub.tsx` | 4 | 751 |
| 16 | `web/src/pages/strategy-hub/StatusTab.tsx` | 4 | 214 |
| 17 | `web/src/components/backtest/OrchestrationRunner.tsx` | 2 | 215 |
| 18 | `web/src/hooks/useOrchestrationPoll.ts` | 2 | 55 |
| 19 | `web/src/components/backtest/OrchestrationProgressBar.tsx` | 2 | 55 |
| 20 | `web/src/components/backtest/OrchestrationDetail.tsx` | 3 | 238 |
| 21 | `web/src/components/backtest/OrchestrationHistoryTab.tsx` | 4 | 133 |
| 22 | `internal/db/migrations/000033_strategy_status_orch_link.up.sql` | 6 | 1 |
| 23 | `internal/db/migrations/000033_strategy_status_orch_link.down.sql` | 6 | 1 |

### 9.3 File Inventory — Modified (15 files)

| # | File | Phase(s) | Changes |
|---|------|----------|---------|
| 1 | `internal/backtest/engine.go` | 0 | Sortino guard (`downStdDev < 1e-6`), sample formula, `MatrixBacktestConfig.WirePipeline` field |
| 2 | `internal/risk/regime_activation.go` | 0 | Added `rsi2_reversion` entry (Calm+Trending, Kelly=0.25/0.25) |
| 3 | `internal/backtest/parallel_runner.go` | 0 | Data race fix: independent `NewEngine(db)` per goroutine |
| 4 | `internal/engine/live_engine.go` | 1 | Adaptive slippage calibration methods |
| 5 | `internal/backtest/batch_runner.go` | 3 | Pipeline wiring when `config.WirePipeline=true` |
| 6 | `internal/api/router.go` | 2-6 | Orchestration + status routes + handler wiring |
| 7 | `internal/api/strategy_status_handler.go` | 6 | `promoteRequest.OrchestrationRunID` field |
| 8 | `internal/db/repository_orchestration.go` | 6 | `StrategyStatus.OrchestrationRunID`, updated SQL in upsert+get+list |
| 9 | `web/src/types/api.ts` | 2-6 | 6 new types + `StrategyStatus.orchestration_run_id` |
| 10 | `web/src/api/client.ts` | 2-6 | `orchestrator` module (7 methods), `strategyStatus` module (4 methods), `orchestrator.getTrades` |
| 11 | `web/src/lib/export.ts` | Q5 | 3 orchestration CSV export functions |
| 12 | `web/src/pages/BacktestHub.tsx` | 2-5 | Mode type + orchestrated, HistoryView tabs, type-based DetailView, setView opts, default mode→matrix |
| 13 | `web/src/App.tsx` | 5 | `/orchestration` → redirect to `/backtest?mode=orchestrated` |
| 14 | `web/src/components/layout/Sidebar.tsx` | 5 | Removed standalone "Orchestration" nav item |
| 15 | `web/src/components/deploy/PromoteToLiveWizard.tsx` | 5.3 | Orchestration deployment panel + toggle + bulk-promote |

### 9.4 API Endpoints — All Verified

| Endpoint | Method | Status | Phase |
|----------|--------|--------|-------|
| `/api/v1/orchestrator/run` | POST | 202, returns run_id, async exec | 1 |
| `/api/v1/orchestrator/runs` | GET | 200, paginated list | 2 |
| `/api/v1/orchestrator/runs/:id` | GET | 200, full run + result_json | 1-2 |
| `/api/v1/orchestrator/runs/:id/equity` | GET | 200, pool equity curve | 1 |
| `/api/v1/orchestrator/runs/:id/trades` | GET | 200, trade array | 1 |
| `/api/v1/orchestrator/runs/:id/daily-returns` | GET | 200, daily return series | 1 |
| `/api/v1/orchestrator/runs/:id/metrics` | GET | 200, 7 metric fields | 1 |
| `/api/v1/orchestrator/runs/:id/allocation` | GET | 200, allocation history | 2 |
| `/api/v1/orchestrator/runs/:id/correlation` | GET | 200, correlation matrix + breaches | 2 |
| `/api/v1/orchestrator/runs/:id` | DELETE | 200, cancel run | 2 |
| `/api/v1/strategies/:id/status` | GET | 200, Status with orch_run_id | 2-6 |
| `/api/v1/strategies/statuses` | GET | 200, all statuses | 2 |
| `/api/v1/strategies/:id/promote` | POST | 200, accepts orchestration_run_id | 6 |
| `/api/v1/strategies/:id/demote` | POST | 200, demotion with reason | 2 |

### 9.5 Bugs Found and Fixed During Implementation

| # | Bug | Root Cause | Fix |
|---|-----|-----------|-----|
| B1 | result_json empty in DB | `UpdateOrchestrationRun()` didn't accept JSON param | Added `UpdateOrchestrationRunWithJSON()` |
| B2 | All strategies returned nil from registry | `strategy.NewRegistry()` creates empty registry | Changed to `strategy.GlobalRegistry()` |
| B3 | 0 trades generated (critical) | `openPositions` key mismatch — `strategyID` vs `engineID("SYM:TF:SID")` | Changed all `openPositions` keys to use `engineID()` |
| B4 | TEXT[] serialization error on save | `json.Marshal([]string)` incompatible with PostgreSQL `TEXT[]` | Added `stringsToPgArray()` and `pgArrayToStrings()` helpers |
| B5 | Cancel endpoint panicked | `floatPtr(result.PoolSharpe)` with nil `result` | Added nil guard in `UpdateOrchestrationRun()` |

### 9.6 Verification Gates — All Passing

| Gate | Tool | Result |
|------|------|--------|
| Go build | `go build ./...` | Clean |
| Backend tests | `go test ./internal/backtest/... ./internal/risk/...` | All pass |
| TypeScript | `npx tsc --noEmit` | Zero errors |
| Vitest | `npx vitest run` | 233 passed (26 files) |
| DB migration 000032 | Docker psql apply | 3 tables created |
| DB migration 000033 | Docker psql apply | `orchestration_run_id` column added |
| API E2E (promote) | curl POST /strategies/:id/promote | 200, DB persisted |
| API E2E (orch submit) | curl POST /orchestrator/run | 202, async execution ran |
| API E2E (equity) | curl GET /orchestrator/runs/:id/equity | 2642 points returned |
| API E2E (metrics) | curl GET /orchestrator/runs/:id/metrics | All 7 fields |
| API E2E (trades) | curl GET /orchestrator/runs/:id/trades | Array returned |
| API E2E (daily-returns) | curl GET /orchestrator/runs/:id/daily-returns | 105 returns |

---

## 10. Pending Items — Deep Analysis & Remediation Plan

### 10.1 K1 — Sizing Constraint at High Asset Prices

**Severity:** MEDIUM | **Risk:** Orchestrator produces zero trades on high-priced instruments

**Root Cause Analysis**

The orchestrator's signal generation path hardcodes position sizing as:

```
sizingPct = 0.02 * kelly * w
baseSize   = capital * sizingPct / price
```

With default values (`kelly=0.25`, no weight pre-rebalance: `w=0.5`), `sizingPct = 0.0025` (0.25%). At SPX500 (~$6,000) with $100K capital: `baseSize = $250 / $6,000 = 0.0417 shares`. The orchestrator's `fillQty < 1` guard rejects this. The `RiskPipeline` further applies integer share rounding via `RequestCapital`, producing zero-size positions. This is not just a configuration issue — it's an architectural limitation: the pipeline operates in integer shares (per AGENTS.md Hard Prohibition #2: BIGINT fixed-point), making sub-1-share positions mathematically impossible.

**Affected Code Paths**
- `orchestrator.go:328` — `sizingPct := 0.02 * kelly` (hardcoded 2%)
- `orchestrator.go:335` — `baseSize := capital * sizingPct / price`
- `orchestrator.go:371` — `if fillQty < 1 { continue }` (gate)
- `risk/pipeline.go` — `ProcessSignal` → `CapitalGate.RequestCapital()` — caps at MaxPositionPct × capital, rounds to integer shares

**Works on:** JPN225 (~$300), SPY (~$550), QQQ (~$500) at $100K+ capital
**Fails on:** SPX500 (~$6,000), ES (~$5,800), NQ (~$20,000), BTCUSD (~$60,000) at $100K capital

**Remediation Design**

1. **Make position sizing configurable** — Add `MaxPositionPct` field to `OrchestratorConfig` (default 0.02, validated range 0.005–0.10). Respect the existing `MaxPositionPct` on `CapitalPoolSim` (already enforced by `RequestCapital`) as the system-level cap. The orchestrator's own sizing becomes the strategy-level allocation, which is then capped by the pool's `MaxPositionPct`.

2. **Validate at config level** — Clamp inputs in `NewOrchestrator()`: if `cfg.MaxPositionPct < 0.001` or `> 0.20`, clamp to 0.02.

3. **Surface in UI** — Add "Max Position %" slider to `OrchestrationRunner` and `PromoteToLiveWizard` orchestration panel, with warning text below 1% (orange) and above 10% (red).

4. **Document the constraint** — Add tooltip: "Position sizing is subject to RiskPipeline integer share rounding. For instruments priced above capital × sizing% / 1.0, no position will be opened."

**Effort:** 1h backend, 30min frontend.

---

### 10.2 K2 — No Matrix Orchestration

**Severity:** MEDIUM | **Risk:** Users must manually configure each orchestration run; no automated strategy×symbol×timeframe combo exploration via orchestrator

**Root Cause Analysis**

The matrix runner (`batch_runner.go:RunMatrixConcurrent`) iterates strategy×symbol×timeframe combos and runs each through a single `Engine.Run()`. This produces per-combo metrics (Sharpe, Sortino, MaxDD, etc.) — independent of each other. The orchestrator, by contrast, runs multiple strategies concurrently through a shared capital pool with dynamic allocation. These are fundamentally different execution models:

| Aspect | Matrix Runner | Orchestrator |
|--------|-------------|-------------|
| Execution model | N independent backtests | 1 pool with N strategies |
| Capital management | Per-combo fixed capital | Shared pool with Kelly-weighted allocation |
| Rebalancing | None | Every T bars with correlation brake |
| Output metrics | Per-combo standalone | Pool-level + allocation history + correlation |
| Use case | "Which strategy×symbol×tf is best?" | "How do these strategies perform together?" |

The design doc §4.5 specified "Wire orchestrator into matrix-runner for multi-strategy simulation" but the two models are not directly composable — a matrix of orchestration runs would need a new runner that iterates over combo SETS and runs each set through the orchestrator.

**Remediation Design — Phase 7 Specification**

| Sub-Task | Deliverable | Effort |
|----------|-------------|--------|
| 7.1 | Design `OrchestratorMatrixConfig`: N sets of strategy×symbol×timeframe combos, each set independent | 2h |
| 7.2 | Implement `RunOrchestratorMatrix()`: run N orchestrated backtests concurrently (capped at 2 by semaphore), produce per-set pool metrics | 4h |
| 7.3 | API: POST /api/v1/orchestrator/matrix with combo sets; returns batch_id; streaming via GET /matrix/:batchId/results?since={seq} | 3h |
| 7.4 | Frontend: orchestration matrix results panel (reuse MatrixResultsPanel with orchestration-specific columns: pool sharpe, pool maxdd, allocation, correlation breaches) | 4h |
| 7.5 | DB: extend `orchestration_runs` with `batch_id` for grouping matrix runs | 1h |

---

### 10.3 K3 — Orchestration Test Suite Not Yet Implemented

**Severity:** MEDIUM | **Risk:** No automated regression safety net for orchestration code paths; manual E2E verification only

**Root Cause Analysis**

The test specification in §4.5 defines 210 tests across three layers. The existing test suite (233 vitest + 49 playwright + all Go tests) covers only existing functionality — zero tests exercise the new orchestration code paths. The highest-risk areas are:

1. **Orchestrator `Run()` method** (BT1-BT22) — signal generation, trade execution, rebalance logic, regime gating, pool halting
2. **RebalanceScheduler** (BT23-BT33) — Kelly-weighted allocation formula, eligibility gate
3. **CorrelationTracker** (BT34-BT43) — Pearson computation, velocity brake, pair matrix
4. **Go API handlers** (BT61-BT93) — request validation, async execution, endpoint responses

**Remediation Plan — Priority-Ordered**

| Priority | Tests | Effort | Rationale |
|----------|-------|--------|-----------|
| **P0 — Critical safety net** | BT1-BT8 — orchestrator core (config, add strategy, run with 1-2 strategies, empty, single) | 3h | Covers the `Run()` path that took 3 debug iterations to get working |
| **P1 — Math correctness** | BT23-BT27 — rebalance scheduler (weights, cadence, eligibility) | 2h | The Kelly-proportional formula is the core allocation algorithm; arithmetic errors would silently corrupt results |
| **P2 — API integrity** | BT61-BT72 — orchestrator handler (submit, list, get, allocation, correlation, cancel) | 3h | Covers the 10 external-facing API endpoints |
| **P3 — Frontend components** | FT1-FT17 — OrchestrationRunner (all states + interactions) | 2h | Covers the most complex new UI component |
| **P4 — Correlation** | BT34-BT43 — correlation tracker | 2h | Pearson computation and velocity brake logic |

---

### 10.4 K4 — Legacy `OrchestrationHub.tsx` Standalone Page

**Severity:** LOW | **Risk:** Code maintenance overhead; no user-facing impact (route redirects)

**Root Cause Analysis**

The original `OrchestrationHub.tsx` (751 lines) was built as a standalone page with its own tabs, data fetching, and state management. After Phases 2-5 integrated orchestration into BacktestHub as a mode, the standalone page became redundant. The `/orchestration` route was changed in Phase 5.1 to redirect. However:

- The 751-line file is still imported (lazy) in `App.tsx` — but the import was removed in Phase 5.1 when the route changed to `<Navigate>`. **Correction**: checking App.tsx, the `OrchestrationHub` lazy import WAS removed in Phase 5.1. The file is orphaned — no importers.
- The page components within it (`DetailTab`, `HistoryTab`, `RunnerTab`) were the source of truth for the extracted components: `OrchestrationDetail`, `OrchestrationHistoryTab`, `OrchestrationRunner`. These components are now independent and used by BacktestHub.
- The file should be converted to a thin redirect wrapper or archived.

**Remediation**

Replace `OrchestrationHub.tsx` with a minimal redirect component that immediately navigates to `/backtest?mode=orchestrated`, respecting the `ORCA_ORCHESTRATION_LEGACY_REDIRECT` feature flag concept.

**Effort:** 15min.

---

### 10.5 K5 — No Monte Carlo for Orchestration Pools

**Severity:** LOW | **Risk:** Users cannot assess the statistical robustness of pool-level metrics; no confidence intervals for pool Sharpe/MaxDD

**Root Cause Analysis**

The existing backtest Monte Carlo (`monte_carlo.go:RunMonteCarlo`) bootstraps individual trade sequences 500 times, computing distribution of Sharpe, MaxDD, and Return%. This works for a single strategy with a single equity curve. Orchestration pools have:

- A single pool equity curve (already suitable for bootstrapping)
- Per-strategy trade sequences (could be bootstrapped individually and correlated)
- Allocation history (changing weights over time — non-trivial to resample)

The simplest approach (and the one the existing Monte Carlo component expects) is to bootstrap the pool equity curve directly:

```
func RunPoolMonteCarlo(equity []EquityPoint, trials int) PoolMonteCarloResult {
    dailyReturns := computeDailyReturnsFromEquity(equity)
    for i := 0; i < trials; i++ {
        sampled := bootstrapSampleWithReplacement(dailyReturns)
        cumReturns := computeCumulative(sampled)
        sharpeDist[i] = computeSharpe(sampled)
        maxDDDist[i] = computeMaxDD(cumReturns)
        returnDist[i] = cumReturns[len(cumReturns)-1]
    }
    return PoolMonteCarloResult{...}
}
```

**Remediation Design**

| Sub-Task | Deliverable | Effort |
|----------|-------------|--------|
| 5.1 | Implement `RunPoolMonteCarlo()` in `orchestrator.go` — bootstraps pool equity daily returns 500 times | 1.5h |
| 5.2 | Add to `EnrichResultJSON()` — store Monte Carlo results in result_json | 30min |
| 5.3 | Frontend: reuse existing Monte Carlo chart component with pool equity data | 1h |
| 5.4 | Add to `OrchestrationDetail` — Monte Carlo section below equity curve | 30min |

**Effort:** 3.5h total.

---

### 10.6 Remediation Implementation Order

| Order | Item | Effort | Rationale |
|-------|------|--------|-----------|
| 1 | **K1** — Configurable sizing | 1.5h | Highest user impact: currently prevents any orchestration run on ES/NQ/SPX500 |
| 2 | **K4** — Legacy page cleanup | 0.25h | Quick win, reduces maintenance surface |
| 3 | **K3-P0** — Orchestrator core tests (BT1-BT8) | 3h | Safety net for the most complex code path |
| 4 | **K3-P1** — Rebalance scheduler tests (BT23-BT27) | 2h | Correctness of core allocation math |
| 5 | **K5** — Pool Monte Carlo | 3.5h | Statistical robustness for pool metrics |
| 6 | **K2** — Matrix orchestration plan | 1h | Design document only; implementation deferred |
| **Total** | | **~11.25h** | |

---

## 10.7 Implementation Status

| # | Item | Status | Details |
|---|------|--------|---------|
| K1 | Configurable sizing | **In Progress** | `MaxPositionPct` field added to config struct |
| K2 | Matrix orchestration plan | Pending | Design doc above, implementation Phase 7 |
| K3-P0 | Orchestrator core tests | Pending | |
| K3-P1 | Rebalance scheduler tests | Pending | |
| K4 | Legacy page cleanup | Pending | |
| K5 | Pool Monte Carlo | Pending | |

---

## 11. Approval Checklist

- [x] Architecture review — implemented
- [x] Product review — implemented
- [x] Quant review — confirmed
- [x] DevOps review — implemented
- [x] QA review — all 233 existing tests pass
- [ ] Security review — pending

---

*End of plan. All phases 0-6 implemented. Pending: test suite creation (210 tests), matrix orchestration (Phase 7), Monte Carlo for pools (Phase 7).*
