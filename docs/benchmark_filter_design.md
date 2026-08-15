# Market-Based Benchmark Filter — Design & Integration Report

**Date:** 2026-08-14
**Status:** Design proposal (no code changes)
**Scope:** A mandatory promotion gate that requires every promotion-eligible strategy to clear a market-based benchmark comparison before it can be promoted.

---

## 1. Purpose

Orca currently has a benchmark *overlay* (`GET /api/v1/backtests/:id/benchmark`, hardcoded SPY/QQQ base-100 curves in `internal/api/backtest_metrics_handler.go:139`), but no benchmark *filter*. A strategy with a strong but market-relative weak Sharpe can still pass the promotion gate (`orca/sizing/promotion_gate.py`), which today checks only BH/Bonferroni significance, Deflated Sharpe Ratio, walk-forward OOS degradation, and trade distribution.

This filter adds a **market-relative hurdle**: to be promoted, a strategy must produce returns that are economically meaningful *relative to* an appropriate benchmark — not merely positive in isolation. It is a **mandatory, fail-closed gate** for all promotion-eligible strategies, sitting alongside (not replacing) the existing statistical gates.

---

## 2. Design Principles

These are non-negotiable, aligned with the Orca stack constitution (`AGENTS.md`).

1. **Math stays in Python (HP #1).** Benchmark metrics (beta, alpha, information ratio, tracking error, up/down capture, active Sharpe, benchmark-relative drawdown) live in a new `orca/benchmark/` module. Go shells out via subprocess — it never reimplements the metrics, mirroring `internal/backtest/strategy_hash.go` and the new `internal/api/backtest_metrics_handler.go` robustness subprocess.

2. **Mandatory = fail-closed, not fail-open.** Missing benchmark data, an unknown ticker, or an un-declared benchmark **blocks promotion** with an actionable diagnostic. There is no silent "no benchmark → pass". The only escape hatch is an explicit, audited pre-flight override (see §7.4), never a default.

3. **Declared before results, never tuned after.** The benchmark choice must be a versioned part of the strategy config (`.gkr.yaml`) that is hashed (canonical JSON + SHA-256, HP #3). Selecting a benchmark *after* seeing results to make a strategy look good is the benchmark-selection equivalent of overfitting and must be structurally impossible: the hash of the config (including `benchmark:`) is fixed before the matrix run.

4. **Point-in-time / no look-ahead (HP #9).** Benchmark returns are sampled on the exact trading-calendar intersection with the strategy equity curve; no forward-fill, no alignment that lets the benchmark "know" the future. This reuses the source+timeframe priority-ordered loader, *not* the legacy `LoadCandles` path (which AGENTS.md §Backtest Remediation explicitly warns produces ~7–10× price-scale discontinuities).

5. **Fixed-point prices (HP #2).** Benchmark candles are ingested as BIGINT fixed-point like every other candle; no float price fields.

6. **Statistical rigor is inherited, not duplicated.** The benchmark comparison is itself a hypothesis test, so it must be deflated with the existing machinery (DSR/block bootstrap/`orca/sizing/robustness.py`), and corrected for the number of benchmark metrics evaluated (Bonferroni/BH via `orca/sizing/multiple_testing.py`).

7. **Frozen domain models (HP #7).** The benchmark spec is a Pydantic model with `ConfigDict(frozen=True, extra="forbid")`.

---

## 3. Filter Semantics

### 3.1 Primary metrics

A benchmark filter should be driven by a small set of *primary* metrics with hard thresholds, plus a *secondary* set reported for context. Recommended primary set:

| Metric | Definition | Why |
|--------|-----------|-----|
| **Information Ratio** | `mean(active return) / std(active return)`, annualized | The core relative-quality metric; scale-invariant |
| **Annualized Alpha (CAPM)** | intercept of `r_strat − rf = α + β(r_bench − rf) + ε`, annualized | Isolates skill from beta exposure |
| **Active Sharpe** | annualized Sharpe of `r_strat − r_bench` | Direct "beating the benchmark" quality |
| **Benchmark-relative excess total return** | `CAGR_strat − CAGR_bench` over the window | The economic headline |

Recommended **default pass rule** (all configurable): a strategy passes if, over the evaluation window,
- Information Ratio ≥ `0.4`, **and**
- annualized alpha ≥ `0`, **and**
- active Sharpe ≥ `0` (or, stricter, its Deflated active-Sharpe ≥ `0` at the matrix trial count).

These thresholds are deliberately permissive defaults; they block only *clearly* benchmark-relative-unskilled strategies. Teams tune them per asset class.

### 3.2 Secondary (reported, not gating)

- Beta, tracking error, up-capture / down-capture ratio, benchmark-relative max drawdown, `% of periods strategy > benchmark`, correlation.
- `win_rate_vs_benchmark` (share of overlapping periods the strategy's return exceeds the benchmark's) — a useful sanity flag but *not* a primary gate (a high-Sharpe low-win-rate strategy can still be valid).

### 3.3 Statistical treatment

- **Active-return series** (`r_strat − r_bench`) is the input to the existing `orca/sizing/robustness.py::backtest_robustness_stats` → reports active Sharpe SE, CI, Deflated active-Sharpe, and MinTRL.
- The filter verdict is itself gated by **Deflated active-Sharpe ≥ 0** at the matrix trial count, so a benchmark-beating result that is merely selection noise does not pass.
- When multiple benchmark metrics are evaluated, apply BH-FDR across them before declaring failure (mirrors `promotion_gate`).

---

## 4. Benchmark Selection Options (configurable)

### 4.1 Taxonomy

| `kind` | Description | Default comparator |
|--------|-------------|--------------------|
| `equity_index` | Broad market equity proxy | `SPY` (S&P 500) |
| `growth_index` | Growth/tech equity proxy | `QQQ` (Nasdaq-100) |
| `sector_etf` | Sector-specific ETF (XLK, XLF, XLE, XLV, XLY, XLP, XLI, XLB, XLU, XLRE, XLC) | chosen per strategy domain |
| `buy_hold` | Broad-market buy-and-hold of the strategy's **own universe** (equal-weight or cap-weight) | strategy universe |
| `risk_free` | Risk-free hurdle for absolute-return / market-neutral strategies | 3M T-bill / SOFR |
| `custom` | User-defined ticker(s) or a weighted portfolio | user-supplied |

### 4.2 Absolute-return strategies (`risk_free`)

Market-neutral and absolute-return strategies should **not** be compared to SPY (that punishes low-beta skill). For `kind: risk_free`, the filter becomes:

- **excess return** = `r_strat − r_risk_free`, and
- pass = `excess Sharpe ≥ 0` **and** `annualized excess return ≥ hurdle` (e.g., `hurdle = 0` or a configurable floor such as 3-month T-bill + 2%).

This requires a risk-free series. **Note:** Orca does not currently ingest a risk-free curve — `metrics.NewCalculator(0.05)` hardcodes 5%. This is a prerequisite: ingest a short-rate series (e.g., `^IRX`/`^FVX` via the stooq pipeline, or FRED `DGS3MO`) into a new `benchmark_series` table or as a synthetic constant. Flag this as the single largest new dependency.

### 4.3 Custom tickers / indices

`kind: custom` accepts:
- a single ticker (must resolve in the universe or a benchmark catalog), or
- a portfolio: `tickers: [A, B, C]` with `weights: [0.5, 0.3, 0.2]` (equal-weight if omitted), rebalanced at a configured cadence (e.g., daily) with optional transaction cost drag.

Validation (`orca validate`): tickers must exist; weights must be non-negative and sum to 1 (within 1e-6); a `timeframe`/`cadence` must be declared; unknown tickers are rejected (never silently synthetic-filled).

### 4.4 Defaults

- Beta-exposed (trend, ORB, grid, session scalp, pairs, Ichimoku, Donchian, Keltner, …): `kind: equity_index`, `ticker: SPY` (override to `sector_etf` or `growth_index` per strategy domain).
- Absolute-return / market-neutral (mean reversion, VWAP MR, vol harvesting, VIX carry): `kind: risk_free`.
- The default is codified in `configs/universe.json` / a benchmark catalog, so a strategy with **no** `benchmark:` block fails validation loudly rather than silently inheriting a wrong comparator.

---

## 5. Configuration Model (`.gkr.yaml`)

Proposed additive `benchmark:` block (versioned and hashed like the rest of the IR):

```yaml
benchmark:
  kind: equity_index | growth_index | sector_etf | buy_hold | risk_free | custom
  ticker: SPY            # for equity_index/growth_index/sector_etf/risk_free(hurdle source)
  tickers: [SPY, QQQ]    # for custom portfolio
  weights: [0.6, 0.4]    # optional; equal-weight if omitted
  rebalance: 1d          # portfolio rebalance cadence
  risk_free_hurdle: 0.02 # annualized excess-return floor for kind=risk_free
  thresholds:            # optional overrides of the default pass rule
    information_ratio: 0.4
    alpha: 0.0
    active_sharpe: 0.0
```

Validation rules (new in `orca/ir/validator.py`):
- `kind` is a known enum; `ticker`/`tickers` resolve; weights valid; thresholds are finite and in sane ranges; `risk_free` requires a hurdle source.
- The resolved spec is folded into the instance hash so a benchmark change produces a different config hash (detectable in CI).

---

## 6. Data Alignment & Point-in-Time Rules

1. **Calendar intersection.** Build the return series on the **intersection** of strategy equity timestamps and benchmark trading days (drop non-common days, never forward-fill across a missing benchmark day). For intraday strategies, compare at the strategy's decision/aggregation frequency.
2. **Source+timeframe aware loading.** Use `db.Repository.LoadCandlesByTimeframeFiltered(timeframe, source)` (with `SourceValues("stooq")`), *not* `LoadCandles`, so benchmark bars come from the highest-priority source and never merge `seed`/`yahoo` scales into `stooq`.
3. **Total-return vs price-return.** Benchmark series should be **total-return** where possible; if only price-return is available, document it and (for equity indices) add a representative dividend yield so beta-exposed strategies are not unfairly advantaged.
4. **No survivorship.** SPY/QQQ are survivorship-biased indices; document this and prefer `buy_hold` of the strategy's own (survivorship-aware) universe for the most honest comparison.
5. **Cost convention.** The benchmark is a passive buy-and-hold with **zero trading cost** (or a single stated entry fee). The strategy is evaluated net of its own fees/slippage. Do not charge the benchmark the strategy's cost model.

---

## 7. Integration Points

### 7.1 Python — `orca/benchmark/` (new module)

- `spec.py` — frozen `BenchmarkSpec` Pydantic model.
- `metrics.py` — `compute_benchmark_metrics(strategy_returns, benchmark_returns, risk_free=None) -> BenchmarkReport` (beta, alpha, IR, active Sharpe, tracking error, capture ratios, relative drawdown, win-rate-vs-benchmark).
- `filter.py` — `apply_benchmark_filter(report, spec) -> FilterVerdict` (pass/fail + reasons + deflated active-Sharpe).
- `cli.py` command `orca benchmark-filter <matrix.csv> --spec ...` and integration into `orca promote-gate` as an additional veto (consistent with `--require-dsr`).

### 7.2 Go — mandatory gate wiring

- **`internal/backtest/multi_metric_gate.go`** (`EvaluateOOSMultiMetric`): add a benchmark-relative metric to the standard so the Go gate blocks promotion when the filter fails.
- **`internal/backtest/reevaluation.go`** (`StrategyReevaluator`): already consumes `benchmarkSharpe map[string]float64`; extend it to consume the full benchmark verdict so live *re*-promotion/demotion also honors the filter.
- **Matrix runner** (`cmd/matrix-runner`): write benchmark comparison columns (e.g., `BenchmarkIR`, `BenchmarkAlpha`, `BenchmarkActiveSharpe`, `BenchmarkPass`) so the CSV gate can consume them.
- **API** (`internal/api/backtest_metrics_handler.go`): extend `getBacktestBenchmark` to (a) use the filtered loader, (b) accept a configurable symbol/portfolio, and (c) return the filter verdict alongside the overlay.
- **Subprocess boundary**: a `runBenchmarkFilter` helper (mirroring `runBacktestStats`) shells out to `orca benchmark-filter` over stdin with the aligned return series.

### 7.3 Persistence (new migration)

`benchmark_evals` table (mirrors `000047_cost_calibration` style): `strategy_id`, `benchmark_spec_hash`, `window_start/end`, primary/secondary metrics, `passed`, `evaluated_at`. Backed by `db.Repository` upsert/list; surfaced via `GET /api/v1/admin/benchmark-evals` and the backtest detail.

### 7.4 Pre-flight & CI enforcement

- **`orca preflight`** adds a check: every promoted strategy has a `benchmark:` block and a passing `benchmark_evals` row within the calibration window (fail-closed in `--strict`).
- **Anti-pattern scan** adds **Rule 13**: promotion paths (`NewEngine`→`Run`→promote, and `apply_promotion_gate`/`EvaluateOOSMultiMetric`) must consult the benchmark filter — CI-enforced, exactly like Rule 11 (`WirePipeline`).
- **Guardian test**: a critical path asserting "promotion is blocked when below benchmark".

---

## 8. Testing Strategy

### 8.1 Unit (pure math, Python)

- **Reference-invariance tests**: a strategy that is exactly `k × SPY` must yield `beta = k`, `alpha ≈ 0`, `IR = 0`, `active Sharpe ≈ 0`; a cash-only strategy vs `risk_free` must yield `excess ≈ 0`.
- **Determinism/seed**: same inputs → same `BenchmarkReport`; `FrozenDataclass` immutability.
- **Bounds**: beta/IR/weights finite; weights sum to 1; relative drawdown ∈ [−1, 0].
- **Edge cases**: single-period, zero-variance benchmark, flat benchmark, short histories, missing days.

### 8.2 Alignment / no-look-ahead

- **Calendar-misalignment test**: overlapping-but-different calendars produce an intersection (or a hard error), never a forward-filled benchmark.
- **Source-priority test**: benchmark loaded via `LoadCandlesByTimeframeFiltered` never merges `seed`/`yahoo` into `stooq` (mirrors the existing `parity_test.go`).

### 8.3 Parity & property

- **Python↔Go parity** (mirrors `internal/backtest/parity_test.go`): the Go subprocess result matches a direct `orca benchmark-filter` invocation byte-for-byte.
- **Property-based** (`hypothesis`): monotonicity (adding a positive constant drift to the strategy improves alpha/IR), invariance under scaling, symmetry of active-return sign.
- **Golden fixtures**: a curated strategy-vs-SPY fixture with hand-computed beta/alpha/IR checked in `testdata/`.

### 8.4 Guardian / e2e

- `pytest tests/guardian/`: promotion blocked when below benchmark; `orca preflight` fails when a promoted strategy lacks a benchmark config.
- Playwright: `PromoteToLiveWizard` shows the benchmark verdict and disables Deploy when failing.

---

## 9. Anti-Patterns & Risks to Avoid

1. **Benchmark cherry-picking.** Enforce "declared before results, hashed in the IR" (HP #3). Report the benchmark spec hash in every result; a hash mismatch after the fact is a promotion blocker.
2. **Wrong comparator.** Never default a market-neutral strategy to SPY. Use `risk_free`. Codify the default per strategy domain in the benchmark catalog.
3. **Survivorship-blind benchmarks.** SPY/QQQ exclude delisted names; prefer `buy_hold` of the survivorship-aware universe for honesty, or document the bias.
4. **Double-counting costs.** The benchmark is passive (zero cost); the strategy is net. Mixing them double-penalizes the strategy.
5. **Price-return-only benchmarks** unfairly favor dividend-paying strategies. Use total return or a documented dividend adjustment.
6. **Legacy `LoadCandles` path.** Reintroducing it for benchmark bars re-creates the ~7–10× price-scale bug (AGENTS.md). Use the filtered loader.
7. **Thresholds hardcoded in Go.** Keep thresholds in config/Python; Go only enforces the verdict, not the math (HP #1).

---

## 10. User-Facing Configuration UX

| Surface | Change |
|---------|--------|
| **CLI** | `orca validate` reports benchmark config; `orca benchmark-filter` runs the filter standalone; `orca promote-gate` gains `--benchmark` and reports the verdict per survivor. |
| **BacktestHub → Runner** | A "Benchmark" selector (default SPY, options QQQ / sector ETFs / broad B&H / risk-free / custom ticker+weights), persisted into the run config and hashed. |
| **BacktestHub → Detail** | Extend the existing `EquityCurveChart` benchmark overlay (SPY/QQQ) to any configured benchmark, and add a "Benchmark Gate" panel showing primary metrics + pass/fail. |
| **Promote-to-Live wizard** | Step 3 shows the benchmark verdict; Deploy is disabled unless `passed == true` (mirrors the DSR gate). |
| **Admin** | A "Benchmark Evals" read-only table (from `benchmark_evals`) for audit. |

---

## 11. Phased Roadmap (implementation only when approved)

1. **Phase 0 — math + CLI.** `orca/benchmark/` (spec/metrics/filter) + `orca benchmark-filter` + `promote-gate` veto, with full unit/parity/property tests. No Go/UI changes. ✅ **Implemented.**
2. **Phase 1 — data + persistence.** Ingest benchmark series (incl. risk-free `^IRX`/FRED), `benchmark_evals` migration, repository methods, `getBacktestBenchmark` rework to filtered loader + configurable symbols. ✅ **Implemented** — `000049_benchmark_filter` migration (consolidates `benchmark_evals` + `benchmark_series`), `internal/db/benchmark_evals.go` + `benchmark_series.go`, `orca ingest-risk-free` (`^IRX`), `GET /api/v1/admin/benchmark-evals`, `getBacktestBenchmark` reworked to the stooq-filtered loader + `?symbols=`, `POST /api/v1/backtests/:id/benchmark-eval` (risk-free + equity paths → `orca benchmark-filter` → persists verdict), and the Admin "Benchmark Evals" table.
3. **Phase 2 — mandatory wiring.** `MultiMetricGate` + `StrategyReevaluator` + matrix-runner columns + `runBenchmarkFilter` subprocess; anti-pattern Rule 13; `orca preflight` check. 🟡 **Mostly implemented** — `MultiMetricStandard.MinInformationRatio` + `EvaluateBacktestMultiMetricWithBenchmark`, `StrategyReevaluator.SetBenchmarkPassed` (blocks promotion on a recorded `false`), anti-pattern Rule 13 (structural guard on `promotion_gate.py`), `orca preflight` benchmark checks (15/16), and the backtest-detail "Benchmark Gate" panel + `backtests.benchmarkEval()` client method. **Remaining:** matrix-runner `BenchmarkPass`/`BenchmarkIR`/`BenchmarkAlpha` column export (see note below).
4. **Phase 3 — UX.** BacktestHub selector + detail panel + wizard gating + Admin table; Playwright coverage. 🟡 Detail panel + Admin table + wizard gating done; runner selector + Playwright remain (§14).

> **Note on matrix-runner columns (Phase 2 remainder):** computing the benchmark verdict per combo requires a Python subprocess per combination, which is prohibitively slow for large sweeps. Recommended design: compute the verdict **once per (symbol, timeframe)** (the benchmark is identical across strategies on the same symbol), cache it in-memory, and write the same `BenchmarkPass`/`BenchmarkIR`/`BenchmarkAlpha` to every row for that symbol — a ~1 subprocess-per-symbol cost instead of per-combo.

---

## 14. Deferred Items — Subtask Breakdown & Implementation Plan

### 14.1 Matrix-runner benchmark columns (2c) — ✅ implemented

1. **1.1 Shared `internal/benchmark` package.** Extract the subprocess evaluation (`Evaluate`) and spec hashing (`SpecHash`) out of `internal/api` so both the HTTP API and `cmd/matrix-runner` share one Go entry point (HP #1: math stays in Python). ✅
2. **1.2 API refactor.** `benchmark_eval_handler.go` now calls `benchmark.Evaluate` / `benchmark.SpecHash`; the local `runBenchmarkFilter`/`benchmarkSpecHash`/`benchmarkVerdictJSON` were removed. ✅
3. **1.3 Matrix-runner columns.** `--benchmark SPY` flag → loads the benchmark 1d stooq series **once per run** (shared across combos), then per combo aligns `result.DailyReturns` and appends `BenchmarkPass`/`BenchmarkIR`/`BenchmarkAlpha` to the CSV (header, success row, and error row). The subprocess still runs per combo (the verdict compares the strategy, not just the symbol), but the benchmark data load is O(1) per run. ✅

### 14.2 BacktestHub runner benchmark selector (2.2) — ✅ implemented

1. **2.2.1** Runner form now has a Benchmark selector (kind: SPY/QQQ/Risk-Free/Custom + symbol input) in `BacktestHub.tsx`; the choice auto-syncs the symbol per kind. ✅
2. **2.2.2** `BacktestRequest` gained `benchmark_kind`/`benchmark_symbol`; the run request persists them into `backtest_runs.config` JSONB (both matrix and single paths in `submitBacktest`). ✅
3. **2.2.3** `getBacktestBenchmarkEval` reads the run's stored benchmark spec (`benchmarkFromConfig`) as the default `kind`/`benchmark_symbol`, falling back to equity_index/SPY. The `getBacktestBenchmark` overlay uses the configurable `?symbols=` param. ✅

### 14.3 Playwright e2e coverage (2.3) — ✅ implemented

1. **2.3.1–2.3.3** `web/e2e/benchmark-filter.spec.cjs` covers: the runner accepting a benchmark spec, the `benchmark-eval` endpoint returning a verdict (with graceful 503 degradation when the orca toolchain is absent), and the `admin/benchmark-evals` endpoint returning a list. ✅

> **Note on 2.3:** the spec is written defensively (mirrors the existing `backtest-report.spec.cjs` "endpoint unavailable without DB/toolchain" pattern) and was syntax-checked (`node --check`) but not executed here, since e2e requires a running server + browser.

---

## 12. Open Decisions (with recommended defaults)

| Decision | Recommendation |
|----------|----------------|
| Default pass rule | IR ≥ 0.4 **and** alpha ≥ 0 **and** Deflated active-Sharpe ≥ 0 (permissive, configurable) |
| Missing benchmark data behavior | **Fail-closed** (block promotion) with pre-flight override for audit |
| Risk-free source | `^IRX` via stooq pipeline, FRED `DGS3MO` fallback; store in `benchmark_series` |
| Benchmark cost convention | Passive, zero-cost, total-return |
| Single vs. multiple benchmarks | Allow one primary benchmark per strategy (declared); the "multiple benchmarks" concern is handled by the per-domain catalog default, not by testing many benchmarks post-hoc |
| Where the verdict is authoritative | Python filter is the single source of truth; Go consumes the verdict (HP #1) |

---

## 13. Lifecycle of Benchmark-Relative-Unskilled Strategies

**Rule: retire, never discard.** A failed comparison is negative information, not noise. Decommission the live deployment, retain the evidence.

### Why the record must survive

1. **Honest trial-count accounting.** DSR / Rademacher Anti-Serum / CSCV-PBO (`orca/sizing/`) need the *total* trial count — including failures — to deflate survivors' Sharpe. Dropping failed strategies understates trials and inflates DSR, reintroducing the survivorship bias the filter exists to remove. Even a one-row tombstone (strategy id, spec hash, verdict, reason) keeps the count honest.
2. **Negative alpha is informative.** Persistently benchmark-lagging strategies (IR < 0, α < 0) are inversion candidates — the inverse exposure, or their features as a negative signal (mind costs, capacity, short constraints).
3. **Failures diagnose the research process.** A cluster of long-only equity strategies failing SPY with β≈1 is a *crowding* signal about the pipeline, not about any single strategy.

### Tiered disposition

| Disposition | Condition | Retain | Action |
|-------------|-----------|--------|--------|
| `retired` | Failed filter; β≈1, α≈0, IR≈0 (pure beta replicator) | Full record + verdict | Decommission; proves redundancy with a cheaper buy-and-hold |
| `archived` (reuse candidate) | Failed on average but regime-conditional or uncorrelated skill | Full return series + regime labels | Mine in `orca calibrate` post-mortem; feed `RegimeActivationMatrix`; test inversion/attribution |
| `terminal` | Post-mortem: no inversion, no regime, no diversifier value, cost-driven | Tombstone row only | Stop re-evaluating; tombstone preserves trial accounting |

### Standalone skill vs. portfolio contribution

The benchmark filter measures **standalone relative skill**, not **portfolio value**. A low-IR but uncorrelated strategy may still diversify. Industry practice separates the two gates:

- **Benchmark filter** → removes redundant beta and negative alpha.
- **Portfolio-level gate** (correlation / marginal contribution to the existing book) → decides whether a *surviving* (or low-but-uncorrelated) strategy earns capital.

Promotion is therefore **necessary but not sufficient**: a strategy that passes the benchmark filter can still be rejected for crowding/correlation at the portfolio layer.

### Retention mechanics (mapped to Orca)

- **State, not delete:** add a `disposition` field (`active`/`retired`/`archived`/`terminal`) alongside the existing `is_active` in `strategy_params_version`; never `DELETE`.
- **`benchmark_evals` is the tombstone:** it already persists verdict + spec hash; add `disposition` and a `post_mortem` note.
- **Storage lifecycle:** metadata/verdicts retained indefinitely; full daily-return series kept for a bounded window (default 90 days) unless flagged `archived`, then compressed — reusing the TimescaleDB compression (7d) / retention (30d) policies and the `backtest-cache/prune` admin path.
- **Quarterly post-mortem:** fold a "graveyard review" into `orca calibrate`: surface inversion candidates, attribute failures (signal-vs-cost), mine regime-conditional skill, promote the handful worth reuse to `archived`, mark the rest `terminal`.

### What is actually dropped

Only the **live deployment / promotion eligibility** — never the record. Nothing leaves the evidence store until a documented post-mortem marks a strategy `terminal`, and even then the one-row tombstone is retained.

### Anti-patterns

- **Deleting failed trials** → self-inflicted survivorship that biases DSR/RAS.
- **Re-promoting a failed strategy under a new benchmark** → blocked by the declared-before-results config hash (benchmark is part of the hashed `.gkr.yaml`).
- **Keeping an unskilled strategy live "for diversification"** without actually running the portfolio-level gate.
