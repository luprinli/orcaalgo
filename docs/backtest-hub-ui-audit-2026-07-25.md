# Backtest Hub UI Audit — Functional Integrity, UX Flow & Space Efficiency

**Date:** 2026-07-25
**Auditor:** System Architecture Review  
**Scope:** `http://localhost:5173/backtest` — Runner, History, Detail views, all tabs, components, and the promote-to-live wizard.

---

## 1. Executive Summary

The backtest hub captures the *spirit* of the original design — multi-mode runner, matrix dispatch, streaming results, rich detail analytics, and a promote-to-live wizard. However, six specific structural issues create friction in the end-to-end user flow, and two missing capabilities prevent the "one-shot" backtest-to-live pipeline the design intended.

**Overall Rating:** 7/10 — functional and dense, but mode-switching leaks state, navigation between views is fragmented, and the optimize mode lost its strategy selector during refactoring.

---

## 2. Component-by-Component Audit

### 2.1 Runner View (`/backtest`)

**What it renders:**
- Header: "Backtest Runner" + "History" button (top-right)
- Mode radio buttons: Matrix | Single | Optimize
- Form grid (2 columns): Start Date, End Date, Symbols, Capital
- Source/Gate selectors: Data Source (synthetic), Gate Profile (none)
- Timeframes checkboxes (1d, 1h, 5m) — **only visible in Matrix mode**
- Optimize fields (Objective, Max Combinations, Train/Test Years, Step Months) — **only visible in Optimize mode**
- Strategies checkbox list (11 items) — **visible in Matrix and Single modes**
- Footer: combo count hint + Run button
- Results: Matrix progress + results panel, or single result card, or optimize progress/results card

**Strengths:**
- Three modes clearly distinguishable
- Mode-switching hides/shows relevant fields
- Matrix results stream in real-time via WebSocket polling
- Optimize mode has progress bar and results table

**Issues Found:**

| # | Issue | Severity | Lines |
|---|-------|----------|-------|
| R1 | **Optimize mode hides the strategy selector (checkbox list).** The mode silently uses the first strategy from the previous mode's selection. A user switching from Matrix (with `ma_crossover` checked) to Optimize will optimize `ma_crossover` without knowing which strategy is being optimized. | **High** | 333–345 (strategies section wrapped in `mode !== 'optimize'`) |
| R2 | **Optimize mode hides the date range.** Only train/test years and step months are shown, but the user cannot see the actual start/end dates that will be used. The backend derives dates from `now - trainYears - testYears` to `now - testYears`, but this is invisible to the user. | **Medium** | 262–267 (date fields wrapped in `mode !== 'optimize'`) |
| R3 | **11 strategies in a flat vertical list with no categorization.** Strategies like `intraday_mr`, `trend_following`, `grid_trading`, `session_scalp` are presented in raw snake_case names with no type grouping, no search, no select-all. For pro users testing 5+ strategies simultaneously, this is friction. | **Medium** | 333–345 |
| R4 | **Symbol field is a plain text input** with placeholder "SPX500,NAS100" — no autocomplete, no validation of comma-separated formatting, no error on empty. | **Low** | 268 |
| R5 | **Timeframes are hardcoded to 3 options** (1d, 1h, 5m). There's no 15m, 30m, 4h. Users must modify source code to add timeframes. | **Low** | 49 (constant `ALL_TIMEFRAMES`) |
| R6 | **No "Run Backtest" flow shows the strategy/symbol that will be used when in Single mode with multi-select.** If a user checks 3 strategies but selects "Single" mode, only the first strategy runs — silently ignoring the others. No warning. | **Medium** | 163 (single mode uses `strategies[0]` only) |
| R7 | **Matrix results rows have no "View Detail" link.** Users see Sharpe/Sortino/MaxDD per combo in the matrix results table but cannot click through to see full detail for a specific combo. | **High** | MatrixResultsPanel component |
| R8 | **Single-mode result card shows only 5 metrics** — no link to a richer detail view, no ability to drill into trades or equity curve from here. | **Medium** | 374–389 |

---

### 2.2 History View (`/backtest/history` or `?view=history`)

**What it renders:**
- Header: "Backtest History" + Runner / Compare / Refresh buttons
- Table: ID, Type, Strategies, Symbols, Sharpe, Max DD, Win Rate, Trades, Return, Started, Status, Actions
- Empty state when no runs
- Compare mode: multi-select rows → correlation matrix + equity overlay
- Delete with confirmation dialog

**Strengths:**
- Lazy metrics loading (per-entry, not all-at-once)
- Compare mode with correlation matrix and equity overlay
- Rerun and delete actions per row

**Issues Found:**

| # | Issue | Severity | Notes |
|---|-------|----------|-------|
| H1 | **No pagination controls** — limit hardcoded to 50 rows (line 436). Older entries are invisible once >50 runs exist. | **Medium** | Line 436 |
| H2 | **No column-based search or filter** — users cannot filter the history table by strategy name, symbol, or date range. Only "Compare" mode allows selecting rows. | **Medium** | — |
| H3 | **Compare mode uses equity curve comparison, not trade-level analysis** — good for visual comparison but doesn't help with per-trade attribution. | Low | Lines 608–718 |
| H4 | **"Backtest Runner" link in history view just navigates to `/backtest`** — but resets the runner form. No way to go from history → pre-filled runner. | Low | Line 520 |

---

### 2.3 Detail View (`/backtest/history/:id?view=detail`)

**What it renders:**
- Top bar: Back button, Export CSV buttons (Trades/Equity/Returns), Promote to Live button
- Warnings card (if any)
- 17 metric cards in 3-column grid
- Charts: Yearly Summary, Equity Curve, Daily Returns, Monte Carlo (3 sub-components), Calendar Heatmap
- Tabs: Overview (regime stats), Trades (filterable by month, paginated), Optimization, Comparison (live vs backtest)
- PromoteToLiveWizard (3-step modal)

**Strengths:**
- Rich analytics: 17 metrics, 5 chart types, daily/monthly views
- Monte Carlo with simulation paths + histograms + context card
- Calendar heatmap click-to-filter trades
- Export to CSV for all 3 data types
- Full Promote-to-Live wizard inline

**Issues Found:**

| # | Issue | Severity | Lines |
|---|-------|----------|-------|
| D1 | **17 metric cards is overwhelming.** Metrics like `cagr`, `calmar`, `var_95`, `cvar_95`, `trading_volume`, `commission_bps`, `total_commission`, `pass_probability` are all rendered with equal visual weight. A user scanning for "should I deploy this?" has to process 17 numbers. | **Medium** | 856–874 |
| D2 | **Comparison tab only shows live-vs-backtest, not multi-run comparison.** The History view's compare mode handles multi-run comparison, but the Detail view's "Comparison" tab only handles live-vs-backtest. Two different comparison experiences in the same app. | **Medium** | 930 (ComparisonTab) vs 608–718 (History compare) |
| D3 | **Monte Carlo histogram components render even when no MC data exists.** `MonteCarloSummaryCard`, `MonteCarloHistograms`, `MonteCarloContextCard` are conditionally rendered (lines 894–903) but the `mcResult` state is always null unless a specific API call succeeds. | Low | 894–903 |
| D4 | **No "Re-run with different params" button.** Users must go back to History → Rerun. There's no way to tweak parameters from the detail view. | Low | — |
| D5 | **Detail view loads ALL data in parallel** — metrics, equity, daily returns, trades, monthly returns, regime stats, optimization, live comparison. For large backtests, 8 concurrent API calls may saturate the backend. | Low | 765–807 |
| D6 | **Monthly returns fetch is fire-and-forget with no error handling** (lines 786–787). If the endpoint fails, the yearly summary table silently renders nothing. | Low | 786–787 |

---

### 2.4 Promote-to-Live Wizard

**What it renders:**
1. Step 1 — Quality Gates: Sharpe ≥ 1.0, MaxDD ≤ 8%, PassProb ≥ 80%, PF ≥ 1.5. Override checkbox (2FA).
2. Step 2 — Pre-Flight Checklist: DB connectivity, engine ready, data quality, synthetic contamination check.
3. Step 3 — Deploy: Account selector (Alpaca Paper/Live), capital allocation slider (5-100%), risk rules notice.

**Strengths:**
- 3-step gated process with clear pass/fail indicators
- Pre-flight checklist validates operational readiness
- Capital allocation slider with percentage

**Issues Found:**

| # | Issue | Severity | Notes |
|---|-------|----------|-------|
| P1 | **Account selector lists "Alpaca Paper" and "Alpaca Live" but doesn't show account balance or available capital.** User can allocate 100% without knowing how much capital is actually in the account. | **Medium** | Wizard step 3 |
| P2 | **No confirmation of the strategy being deployed.** The wizard shows `strategyName` in the title but doesn't reiterate it at the final "Deploy" step. User could accidentally deploy the wrong strategy. | **Medium** | Wizard step 3 |
| P3 | **No comparison of live account performance vs backtest before deployment.** The Detail view has a live comparison tab, but the wizard doesn't reference it. | Low | — |
| P4 | **Gate override requires 2FA but shows no 2FA input field** — just a checkbox that says "(requires 2FA)". Unclear how 2FA is enforced. | **Medium** | Wizard line 105 |

---

## 3. Navigation Flow Analysis

### Current Flow

```
/backtest (Runner)
  ├── mode=matrix → Run Matrix → streaming results table
  │     └── each row: no click-to-detail
  ├── mode=single → Run Backtest → inline result card (5 metrics)
  │     └── no link to detail
  ├── mode=optimize → Run Optimization → inline progress + results
  │     └── best params table, no link to detail
  └── "History" button → /backtest/history?view=history
        ├── "View" link → /backtest/history/:id?view=detail
        │     ├── overview tab → regime stats
        │     ├── trades tab → filterable trade list
        │     ├── optimization tab → optimization footprint
        │     ├── comparison tab → live vs backtest
        │     └── "Promote to Live" button → 3-step wizard
        ├── "Rerun" → navigate to runner with prefilled config?
        │     └── currently just calls backtests.rerun(id) and navigates to detail
        ├── "Compare" → select rows → correlation matrix + equity overlay
        └── "Refresh" → reload list
```

**Two Fragmented Comparison Experiences:**
1. History view compare: multi-run equity correlation
2. Detail view comparison tab: live-vs-backtest

These should be unified into a single comparison framework.

**Missing "Promote to Live" Shortcut:**
- No way to go from runner → configure → directly promote without running a full backtest
- No way to promote a strategy from the runner's optimization results

---

## 4. Prioritized Remediation Plan

### P0 — Must Fix (UX Broken)

| # | Issue | Fix | Effort | Impact |
|---|-------|-----|--------|--------|
| **R1** | Optimize mode hides strategy selector | Show strategy selector (single-select dropdown) in Optimize mode. Use the first selected strategy as default but let user change it. | 0.5h | Prevents silent strategy mismatch |
| **R7** | Matrix results rows have no "View Detail" link | Add a "View" action link per row in `MatrixResultsPanel` that navigates to detail view with the combo's backtest ID. | 1h | Users can drill into winning combos |
| **P2** | No strategy name in final deploy step | Display `strategyName` prominently in Step 3 of the wizard, with a "Strategy: X — Backtest: Y — Account: Z" summary before the final Deploy button. | 0.5h | Prevents wrong-strategy deployment |

### P1 — High Priority (Significant UX Friction)

| # | Issue | Fix | Effort | Impact |
|---|-------|-----|--------|--------|
| **R6** | Single mode silently ignores extra strategies | Show a warning banner when >1 strategy is checked in Single mode: "Only the first strategy (X) will be run." Alternatively, add an `onSelect` that trims to one. | 0.5h | Prevents silent data loss |
| **P1** | Account selector doesn't show balance | Fetch account balance via `accounts.get(accountId)` and show "Available: $X" below the selector. | 0.5h | Informed capital allocation |
| **P4** | 2FA gate override unclear | Either implement a real 2FA code input, or change the label to "Override gates (skip quality checks)" and remove the "(requires 2FA)" text if 2FA is not actually enforced. | 0.5h | Clarity on security vs convenience |
| **D1** | 17 metric cards overwhelming | Group metrics into collapsible sections: Primary (Sharpe, MaxDD, WinRate, ProfitFactor, Trades, Return), Advanced (Sortino, Calmar, CAGR, VaR, CVaR, PassProb, AvgDDDuration), Costs (Commission, TotalFees, Volume). Default: Advanced and Costs collapsed. | 1.5h | Faster at-a-glance decision |
| **R8** | Single result card shows only 5 metrics | Add a "View Full Report" button that navigates to the detail view for that backtest. | 0.5h | Bridge from runner to detail |
| **D6** | Monthly returns fetch silently fails | Add `.catch()` with a warning state that shows "Monthly returns unavailable" instead of silently rendering nothing. | 0.25h | Graceful degradation |

### P2 — Medium Priority (Polish & Convenience)

| # | Issue | Fix | Effort | Impact |
|---|-------|-----|--------|--------|
| **R3** | Flat 11-item strategy list | Group strategies by type (Mean Reversion, Trend Following, Breakout, Grid) with collapsible accordion groups. Add "Select All" / "Deselect All" buttons. | 2h | Pro workflow for 5+ strategies |
| **R2** | Optimize mode hides date range | Show a read-only date summary derived from train/test years: "Training: 2022-01–2024-01, Test: 2024-01–2025-01". | 0.5h | Transparency into optimization window |
| **H1** | No pagination in history | Add pagination controls (Previous/Next) or infinite scroll. Simple approach: add offset/limit to fetch. | 1.5h | Access to older runs |
| **H2** | No column filters in history | Add a filter bar above the table: Strategy dropdown, Symbol text input, Date range pickers. Filter client-side from loaded data. | 1.5h | Quick finding of past runs |
| **D2** | Two separate comparison experiences | Unify: the Detail view's "Comparison" tab should accept an optional list of backtest IDs to compare against (making it useful for both live comparison and multi-run comparison). | 3h | Single mental model |
| **D4** | No "Re-run with different params" | Add a "Clone & Retune" button that navigates to Runner with the strategy, symbols, date range pre-filled. | 1h | Fast iteration |

### P3 — Low Priority (Nice-to-Have)

| # | Issue | Fix | Effort | Impact |
|---|-------|-----|--------|--------|
| **R4** | Symbol input needs autocomplete | Add symbol autocomplete from the symbols API or cache store. Show matching symbols as the user types. | 2h | Error prevention |
| **R5** | Timeframes hardcoded to 3 | Make timeframe list configurable via backend endpoint or at least via constants file. | 1h | Flexibility |
| **D3** | Monte Carlo loads conditionally with no feedback | Add a loading skeleton or "Compute Monte Carlo" button if no data exists yet. | 0.5h | Loading feedback |
| **D5** | 8 concurrent API calls on detail load | Batch the non-critical data (regime stats, optimization, live comparison) into a second wave after primary data loads. | 1h | Perceived speed |
| **H3** | Compare mode only uses equity overlay | Add a trade-level comparison table (Win Rate by regime, Average Trade by month, etc.). The data is already loaded. | 2h | Deeper comparison |
| **H4** | "Backtest Runner" link resets form | Use `window.history.back()` or store last runner config in sessionStorage so returning to runner restores previous setup. | 0.5h | Context preservation |

---

## 5. Summary

| Category | Score | Notes |
|----------|-------|-------|
| **Functional Correctness** | 8/10 | All views render and respond to mode switches. Optimize mode strategy leak (R1) is the only functional bug. |
| **Information Architecture** | 7/10 | Three views (runner/history/detail) is correct, but the transitions between them are fragmented. No direct runner→detail or history→prefilled-runner flow. |
| **Space Efficiency** | 7/10 | Runner form is compact. Detail view's 17 metric cards could use collapsible groups. Matrix results table is dense but scrollable. |
| **Workflow Fidelity** | 6/10 | The "select strategies → optimize per strategy → run matrix → compare results → promote best" pipeline exists in pieces but lacks smooth transitions. Users must manually navigate between views and remember state. |
| **Deploy Readiness** | 7/10 | Promote-to-Live wizard has proper gates but lacks account balance visibility, strategy confirmation at final step, and 2FA enforcement clarity. |

**Total estimated remediation effort:** 22h (P0: 2h, P1: 4.25h, P2: 9.5h, P3: 7h)

**Recommended first-wave (8h):** R1, R7, P2, R6, D1, R8 — these 6 items fix the most impactful UX gaps with minimal effort.
