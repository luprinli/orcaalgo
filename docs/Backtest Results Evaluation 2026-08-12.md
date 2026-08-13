# Backtest Results Evaluation — Range Comparison & Promotion Readiness

**Subject:** `data/.backtest_results/matrix_results _from2025.csv` and `matrix_results_from2023.csv` (1,944 rows each = 18 strategies × 18 symbols × 6 timeframes)
**Date:** 2026-08-12
**Scope:** Read-only evaluation of the two latest matrix result sets; validation of the recent framework fixes and assessment of suitability for promotion to orchestration/live.
**Classification:** Internal — Quantitative Risk / Backtest Validation

---

## Verdict

**Not suitable for promotion to orchestration/live.** The recent framework code fixes are validated and working, but the results are dominated by a **data-integrity failure**: the legacy `seed` synthetic fixture is merged with real `stooq` bars at incompatible price scales, producing absurd metrics (e.g. 132% return, $1,406 average win). In addition, the stooq pipeline is incomplete (no resampled/calibrated intraday data), and all results are in-sample with no multiple-testing correction.

---

## 1. Validation of Recent Changes — PASSED

Both files (`DataSource=stooq`, `EngineVersion=dev`) confirm the P0/P1 fixes are working end-to-end:

| Check | `_from2025` | `_from2023` | Status |
|---|---|---|---|
| Kelly in `Params` | `[0.25]`, 1,944/1,944 rows | `[0.25]`, 1,944/1,944 | ✅ P0-2 |
| No-data rows | 0 | 0 | ✅ P0-1 |
| `Wins+Losses != Trades` | 0 | 0 | ✅ P1-1 |
| Slippage > 100 bps | 0 | 0 | ✅ P0-4 |
| PF=0 with 100% win rate | 0 | 0 | ✅ P1-1 |
| Gate passes | 8 | 15 | improved |
| Positive-Sharpe combos | 45 | 73 | — |

The engine/metrics layer is correct. The remaining problems are in the data layer.

---

## 2. Result Summary (both windows)

| Metric | `_from2025` (~1y) | `_from2023` (~3y) |
|---|---|---|
| Rows | 1,944 | 1,944 |
| Zero-trade rows | 1,068 | 969 |
| Gate passes | 8 | 15 |
| Positive Sharpe | 45 | 73 |
| Median CandleCount | 3,354 | 4,698 |
| Anomaly rows (AvgWin > $500) | 2 | 2 |

The 3-year window produces more gate passes and positive-Sharpe combos (more history → more trades), but the same structural anomalies appear in both.

---

## 3. Critical Findings Blocking Promotion

### 3.1 CRITICAL — Seed/stooq price-scale mismatch

`SourceValues("stooq")` (`internal/db/repository_candles.go:52`) returns
`["stooq", "stooq-resampled", "stooq-calibrated", "yahoo", "seed"]`, merging the
legacy `seed` synthetic fixture with real `stooq` bars. The two sources use
incompatible price scales:

| Symbol/TF | `seed` close range | `stooq` close range | Ratio |
|---|---|---|---|
| NVDA 5m | 235 → 365 | 1,643 → 2,364 | ~10× |
| ^_US 1h | 45,307 → 64,689 | 2,439 → 3,630 | ~15× (reversed) |

Merging a 10–15× gap produces the implausible rows:

| Anomaly | Value |
|---|---|
| `opening_range_breakout ^_US 1h` (from2025) | Sharpe 24.1, **Return 132.1%**, PF 55.5 |
| `ma_crossover NVDA 5m` (from2025) | **AvgWin $1,406** (~70%/trade) |
| `keltner_macd NVDA 5m` | AvgWin $562 (~28%/trade) |
| `volatility_harvesting NVDA 1h` (from2023) | AvgWin $1,168 on 1 trade |

Root-cause hypotheses:
- **NVDA**: `seed` synthetic is post-split (June 2024 10:1) while `stooq` 5m is unadjusted pre-split → ~10× gap.
- **^_US**: wrong stooq index mapping (2,439–3,630 is not the S&P 500; `seed` is ~10× too high) → reversed ~15× gap.

### 3.2 CRITICAL — Incomplete stooq pipeline

The candles table contains only two sources:

| Source | Timeframes present |
|---|---|
| `seed` (legacy synthetic) | 1d, 1h, 4h, 30m, 15m, 5m (all) |
| `stooq` (real) | **1h and 5m only** |

There is **no `stooq-resampled`** (1h→4h, 5m→15m/30m) **and no `stooq-calibrated`**
gap-fill. Consequently:
- 15m/30m/4h backtests run entirely on the legacy `seed` synthetic (uncalibrated scale).
- Even 1h/5m merge `seed` with `stooq`, reproducing the §3.1 scale mismatch.

The resampling/gap-fill steps of the stooq pipeline did not persist (the earlier reseed attempt aborted mid-pipeline on the Unicode/DB-password bugs; the retry did not re-run the intraday resample/calibrate steps).

### 3.3 HIGH — In-sample only, multiple testing uncorrected

No walk-forward/OOS columns; all Sharpe are in-sample. Across 1,944 combos × 2 windows, 45–73 positive-Sharpe combos are consistent with the null before Bonferroni/Benjamini-Hochberg. The head of the table (Sharpe 24, 8, 4.5) is the expected tail of a large scan, not evidence of edge.

### 3.4 MEDIUM — Low-sample / degenerate signals

- `volatility_harvesting` / `vix_futures_carry` show PF=999 (all-win sentinel) with 5–16 trades — correctly flagged now, but not promotable.
- `trend_following` (8–9 trading rows), `intraday_mr` (21–27), `dragon_trend` (35–38), `donchian_breakout` (59–69) are effectively dead in both windows.
- `session_scalp AUDUSD 1h` (Sharpe 8.0, 94% win rate, 100 trades) is suspiciously high and likely regime-/data-dependent.

---

## 4. Promising Candidates (for follow-up, not promotion)

- **`rsi2_reversion` on liquid names at 1h** is the only strategy consistent across **both** windows:
  - SPY 1h: Sharpe 4.4 / +9.7% (2025), TSLA 1h 3.5, QQQ 1h 3.2, AUDUSD 1h 4.6, TLT 1h 2.5.
  - Still in-sample and uncorrected; the leading candidate only.
- `ma_crossover` (SPY/^_US 1h) and `vol_grid` (TLT/QQQ 1d) show plausible Sharpe 2.3–2.8 in the 3-year window.

---

## 5. Required Fixes Before Any Promotion Decision

| # | Fix | Severity |
|---|---|---|
| 1 | Remove `seed` from `SourceValues("stooq")` (and update `repository_test.go`); the legacy fixture must not contaminate real-data runs | CRITICAL |
| 2 | Complete the stooq pipeline — run `stooq_resample.py` (1h→4h, 5m→15m/30m) and `stooq_synthetic.py` (calibrated gap-fill) so intraday timeframes have real/calibrated data | CRITICAL |
| 3 | Fix NVDA split-adjustment and `^_US` stooq ticker mapping (resolve the ~10–15× scale discrepancies) | CRITICAL |
| 4 | Run `--walk-forward` + multiple-testing correction on `rsi2_reversion`/`ma_crossover`/`vol_grid` candidates before gate re-evaluation | HIGH |
| 5 | Re-run matrix on clean data and re-baseline gate/Sharpe results | HIGH |

---

## 6. Bottom Line

The framework remediation (P0/P1) is complete and validated: Kelly emission, canonical 18-symbol universe, asset-class fees, slippage metric, profit-factor/breakeven accounting, grid disable, and per-symbol regime gating all work correctly in the new results. However, the results are **not yet reliable for live/orchestration promotion** because the data layer merges incompatible `seed` and `stooq` price scales (producing absurd returns and average wins), the stooq intraday pipeline is incomplete for 15m/30m/4h, and no out-of-sample or multiple-testing discipline has been applied. Promotion must be gated on fixes 1–5 above.
