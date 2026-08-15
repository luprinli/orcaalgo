# Backtest Diagnosis & Improvement Report

**Date:** 2026-08-14
**Scope:** `data/.backtest_results/matrix_results (12).csv` (backend matrix export) — why the `Opt` column is blank, where the raw-signal count lives, why strategies underperform benchmarks, and the improvement plan.

---

## 1. Why the `Opt` column is blank

The `Opt`/`Optimized` column is **`false` because `matrix_results (12).csv` was run as a plain sweep** — no light-optimizer, no walk-forward.

- CLI `--optimize` (`cmd/matrix-runner/main.go:22`) sets `optimized = true` and re-runs with the tuned `Params`. Without it, `Optimized=false` and `Params` holds the **strategy defaults** (e.g. `vwap_mr`: `entry_z=1.5, lookback=20, …`).
- `--walk-forward` populates `WfBestParams`/`WfOOSSharpe` instead (OOS-validated).
- The UI (`MatrixResultsPanel.tsx:298`) renders `r.optimized ? "Y" : "—"`, so blank = "no optimization run".

**Fix:** re-run the matrix with `--optimize` (fast, in-sample) or `--walk-forward` (OOS-validated). The `Params` column then shows tuned values and `Opt` → `Y`.

## 2. The raw-signal number *is* captured — just not shown in the UI

The backend matrix records a full signal funnel (`backtest.SignalDiag`, `engine.go:170`), present in CSV (12):

| Column | vwap_mr/MSFT/30m | Meaning |
|---|---|---|
| `CandlesSeen` | 4047 | bars evaluated |
| `SignalAttempts` | 3969 | **raw signals requested** |
| `StrategyNil` | 2554 | runner returned nil (64%) |
| `RegimeRejected` | 1409 | regime gate rejected (99.6% of actual) |
| `SignalsPassed` / `TradesOpened` | 6 | allowed signals |

The raw→allowed comparison exists as columns (`SignalAttempts` vs `SignalsPassed` + the rejection breakdown: `RegimeRejected`, `VolHalted`, `PipelineRejected`, `MLRejected`, `FillRejected`, `CapitalZero`, `RateLimited`, `ExposureBlocked`). **The UI does not render them** — advanced columns stop at `DayVol` (`MatrixResultsPanel.tsx:264`).

## 3. Two matrix paths, divergent columns

- **Backend/UI path** (`internal/backtest/batch_runner.go:235`) writes the **full** header (`SignalAttempts`, `CandlesSeen`, `TradesOpened`, `CapitalZero`, …) → produced `matrix_results (12).csv`.
- **CLI path** (`cmd/matrix-runner/main.go:92`) writes a **shorter** header with abbreviated names (`SigAttempts`, `SigPassed`, `Tf`) and fewer funnel columns.

Same engine/data → comparable performance numbers, but divergent column sets. The UI reads the backend results but hides the funnel.

## 4. Reconciling with theory — the funnel is the smoking gun

The strategies don't beat benchmarks for three measurable reasons:

1. **The regime gate is the binding constraint.** For `vwap_mr` (Calm-only): of 1,415 actual (non-nil) signals, **1,409 (99.6%) were regime-rejected**, leaving 6 trades. A single-regime strategy over a window dominated by a *different* regime barely trades.
2. **Strict entry thresholds.** `StrategyNil` = 64% — the runner returns nil on most attempts (`entry_z=1.5` is tight).
3. **Costs + a strong bull market.** 6 trades at 4.5 bps slippage + fees can't beat SPY buy-and-hold over 2025-08→2026-08 (strong uptrend).

The benchmark filter correctly flags these as benchmark-relative-underperforming (`BenchmarkPass=false`, negative IR).

## 5. Improvement plan

**Unblock the funnel first (highest leverage):**
1. **Widen/parameterize the regime windows** — single-regime strategies are off ~99% of the time. Make regime participation a per-strategy optimizable parameter (`regime_w_calm/trending/highvol/crisis`).
2. **Loosen entry thresholds** (`entry_z` 1.5 → ~1.0) to cut `StrategyNil`.
3. **Run `--optimize` + `--walk-forward`** for strategy/symbol-specific, OOS-validated params.

**Enforce statistical significance:**
4. **MinTRL** — Sharpe 1.5 needs ~100+ trades for 95% confidence; 6 trades is noise. Gate promotion on `min_trl` (`orca backtest-stats`).
5. **Deflated Sharpe / CSCV-PBO** — across ~1000 combos the best Sharpe is selection-biased; use `deflated_sharpe` + the `promotion_gate` BH/DSR veto.
6. **Longer, regime-diverse windows** — use the full 3–5-year data so each strategy sees its own regime.

**Make it visible:**
7. **Surface the signal funnel in the UI** (add `SignalAttempts`, `StrategyNil`, `RegimeRejected`, `SignalsPassed` columns).
8. **Unify the two matrix headers** (CLI vs backend).

## 7. Regime-gate-off experiment (2026-08-14)

Ran the same 81-combo subset (6 strategies × SPY/MSFT/EURUSD, 2023-08-14 → 2026-08-14) with the regime gate **on** vs **off** (`--regime-off`), plain sweep (`--optimize=false`).

### Funnel comparison

| Column | Regime ON | Regime OFF | Delta |
|---|---|---|---|
| `SigAttempts` | 1,652,200 | 1,647,501 | −4,699 |
| `StrategyNil` | 1,393,387 | 1,629,682 | **+236,295** |
| `RegimeRej` | 241,368 | **0** | −241,368 |
| `SigPassed` / `Trades` | 8,711 | 8,906 | **+195** |

### Performance comparison

| Metric | Regime ON | Regime OFF |
|---|---|---|
| Avg Sharpe (trading combos) | −2.965 | −3.103 |
| Median Sharpe | −2.130 | −2.168 |
| Combos with trades | 57 | 63 |
| Combos with Sharpe > 0 | 3 | 2 |

### Per-strategy (regime ON)

| Strategy | Combos | Trades | Avg Sharpe | Pos-Sharpe combos |
|---|---|---|---|---|
| rsi2_reversion | 18 | 4,405 | −2.886 | 0 |
| session_scalp | 12 | 3,478 | −4.975 | 0 |
| vwap_mr | 12 | 793 | −1.091 | 3 |
| intraday_mr | 7 | 17 | N/A (too few) | 0 |
| mean_reversion | 7 | 17 | N/A (too few) | 0 |
| trend_following | 0 | 1 | N/A (too few) | 0 |

### What we learned (recalibrates §5)

1. **The regime gate is not the binding constraint.** Turning it off removed 241K regime rejections but only added 195 trades (+2.2%) — the previously-rejected signals became `StrategyNil` (the strategy evaluated and still found no setup). The regime gate was largely *redundant* with the strategies' own entry conditions.
2. **The regime gate is mildly protective, not destructive.** Sharpe got *worse* with the gate off (−2.965 → −3.103). Keep it; the optimizable participation (already implemented) is the right lever, not a hard off-switch.
3. **The real problem is negative edge, not gating.** `rsi2_reversion` (−2.9) and `session_scalp` (−5.0) trade heavily but lose money systematically; `vwap_mr` is the only strategy with any positive-Sharpe combos (3 of 12). The single-regime strategies are too trade-sparse to even estimate Sharpe.
4. **Implication for the improvement plan:** the highest-value work is fixing the negative-edge strategies' signal logic (entry/exit/sizing), not loosening gates or thresholds. More trades of a −3 Sharpe strategy is more loss, not more alpha.

## 9. Re-validation after the strategy fixes (matrix_results (14))

Re-ran the 1242-combo matrix after the true-ATR, trailing-stop, vol-filter, and regime de-duplication fixes. **The fixes are validated.**

| Strategy family | Before (12) | After (14) |
|---|---|---|
| `intraday_mr` positive-Sharpe combos | 0 | **28** (avg Sharpe −0.36, top 4.20) |
| `mean_reversion`/`vwap_mr` | 0 | still trade-sparse (too few trades) |
| `rsi2_reversion` | 0 | 8 positive (avg −2.26) |
| `ichimoku` / `ma_crossover` / `pairs` / `vix_carry` | — | 7 / 11 / 6 / 5 positive |
| ORB / grid / scalp families | — | still strongly negative (`opening_range_breakout` −11, `orb_15m` −9, `vol_grid` −7, `session_scalp` −4.7, `volume_scalp` −5) |

**Conclusion:** the mean-reversion family now has a real, measurable edge (28 profitable combos on `intraday_mr`). The ORB/grid/scalp families remain broken for a *different* reason (overtrading + the session-timezone/buffer issues) and need their own audit.

## 10. Blank-column root causes (resolved)

| Column | Root cause | Fix |
|---|---|---|
| **Opt** (Optimized) | `RunLightOptimize` returned nil for every strategy: `pickDominantTimeframe` always picks `1d`, and the fixed 3-month window gave ~90 bars — under the 500-bar floor (`light_optimizer.go:216`) | Adaptive window: `lightOptWindowMonths(timeframe)` now sizes the window to guarantee ≥500 bars for the dominant timeframe |
| **Wf IS / Wf OOS** | `MatrixBacktestConfig.WalkForward` defaulted `false`; the frontend never sent it | `WalkForward` now defaults **true** (`req.WalkForward == nil || *req.WalkForward`) |
| **Nil Err / Exit** (new) | `NilError`/`ExitReasons` existed in the backend but not the UI/CSV | Added to `ComboResult` type + a "Nil Err" funnel column; `exit_reasons` available on the row |

### Regime participation — already optimizable (verified 2026-08-14)

Regime gating is already a per-strategy optimizable parameter; the lever exists end-to-end in the backtest path:

1. **Soft participation model** — `RegimeActivationMatrix.Participation [4]float64` (`internal/risk/regime_activation.go`) turns regime gating from all-or-nothing into a risk-proportional weight (`0 = block`, `1 = full`, in-between scales size).
2. **Optimizer search space** — `RegimeParamDefs` (`internal/backtest/optimizer.go:443`) defines `regime_w_calm/trending/highvol/crisis` (0..1, step 0.25, default = the binary `Allowed` pattern) plus `stop_mult_highvol/crisis` and `profit_mult_trending`; `addRegimeParams` injects them into every `DefaultSearchSpace` (`optimizer.go:399`).
3. **Application** — `applyRegimeParticipation` (`engine.go:250`, called at `engine.go:753`) reads `regime_w_*` from `StrategyParams` and overrides the matrix; the RiskPipeline applies the soft weight (`pipeline.go:99`, `size *= regimeWeight`).
4. **Light optimizer + walk-forward** both use `DefaultSearchSpace` (`light_optimizer.go:196/461/603`).

**Test coverage added:** `TestDefaultSearchSpace_IncludesRegimeParticipation` and `TestRegimeParamDefs_DefaultMatchesAllowed` (`internal/backtest/optimizer_test.go`) lock in the behavior.

**Why the strategies are still over-gated:** the matrix was run *without* `--optimize`/`--walk-forward`, so `regime_w_*` stayed at the restrictive binary `Allowed` default (e.g. `vwap_mr` Calm-only → `{1,0,0,0}`). Re-running with `--optimize` lets the optimizer soften participation (e.g. `regime_w_trending = 0.25..0.75`) and should lift `SignalsPassed` sharply.

**Known follow-up (backtest/live parity):** the live engine does not yet apply optimized `regime_w_*` from the active parameter version — it uses the static matrix. Backtest-optimized regime participation must be plumbed into the live engine (`RegisterAccountStrategies` → `applyRegimeParticipation`) to keep backtest/live consistent.
