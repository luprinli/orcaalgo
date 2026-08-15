# Strategy Logic Audit — Deviations, Artifacts & Metric Gaps

**Date:** 2026-08-14
**Scope:** Deep read of every strategy runner, the shared `BaseRunner`, the engine's signal/exit path, and the indicators, to explain why theoretically-profitable strategies are unprofitable in this system, and why the collected metrics don't make it obvious.

---

## 1. Regime gating is triplicated (duplication of effort)

Regime logic lives in **three** independent layers, and the `--regime-off` experiment only disabled two of them — which is exactly why turning it "off" barely changed anything.

| Layer | Location | What it does |
|---|---|---|
| 1. Engine entry gate | `internal/backtest/engine.go:1429` | `if ParticipationForRegime(...) <= 0 { RegimeRejected++; return nil }` |
| 2. Pipeline gate | `internal/risk/pipeline.go:98` | participation gate + per-regime Kelly override |
| 3. Strategy-internal gate | `trend_runner.go:109`, `rsi2_reversion_runner.go:95`, `session_scalp_runner.go:127`, `donchian_breakout_runner.go:87`, `dragon_trend_runner.go:104`, `grid_runner.go:168`, `carry_runner.go:97` | each runner re-checks `if regime == 3 { return nil }` (and `switch regime` in grid) |

`--regime-off` sets `DisableRegimeGate` (layers 1 + 2), but **layer 3 remains**. The 241K "regime rejections" that vanished when the gate was off simply re-surfaced as `StrategyNil` — because the runners *themselves* refuse to trade regime 3 (crisis), and their entry conditions only fire in their natural regime anyway.

**Quant best practice:** there must be **one canonical risk layer**. Strategies should be *pure signal generators* (emit a signal with a confidence), and ALL gating — regime, volatility, exposure, capital, correlation — belongs in the `RiskPipeline` only. Per-strategy regime sensitivity is then a single tunable (the `regime_w_*` participation weights already implemented), not three hand-wired copies.

**Recommendation:** remove the runner-internal `if regime == 3` / `switch regime` blocks and let the pipeline's soft participation weight (0..1) express regime sensitivity uniformly. This eliminates the duplication and makes `--regime-off` actually meaningful.

---

## 2. Strategy-by-strategy artifact catalog

### 2.1 System-wide: `ATR()` is not ATR

`internal/strategy/indicators.go:9`: `func ATR(prices []float64, count int, period int)` takes **only close** and computes the mean absolute close-to-close change — **not** Wilder's True Range (`max(high-low, |high-prevClose|, |low-prevClose|)`).

Every strategy uses this for stop-loss/take-profit distances (`rsi2_reversion_runner.go:109/147`, `session_scalp_runner.go:212`, `trend_runner.go:183`, …). True Range is strictly ≥ close-to-close, so **all stops are systematically tighter than a proper ATR stop**, ignoring gaps and intra-bar range. This produces premature stop-outs in gap/volatile markets — a direct, mechanical source of negative Sharpe.

### 2.2 System-wide: `PushPriceOnly` zeroes high/low/volume

`base_runner.go:84`: `PushPriceOnly` calls `PushPrice(price, 0, 0, 0)`. Any runner that feeds via close-only (`rsi2_reversion_runner.go:104`) permanently writes **high=0, low=0, volume=0** into the history buffers, silently corrupting any high/low/volume-dependent indicator.

### 2.3 MeanReversionRunner (`intraday_mr`, `mean_reversion`, `vwap_mr`)

1. **Broken high-volatility filter** (`mean_reversion.go:131-135`): the guard compares `atrVol/close` (a normalized bar move) against `VolMaxMult·√(histVariance)/close·0.1`, where `histVariance` is the variance of **price levels**, not returns. It's a unit mismatch (bar-move vs price-level std) with an arbitrary `0.1` — effectively noise, so MR trades in high-vol when it shouldn't.
2. **`atrVol` is close-to-close, mislabeled** (`mean_reversion.go:250-262`).
3. **No high/low history at all** — the runner tracks only `closeHistory`/`volumeHistory` (`mean_reversion.go:19-20`), so true ATR is impossible by construction.
4. `vwap_mr` (the only strategy with any positive-Sharpe combos) defaults `EntryZ=1.5, MaxHold=40, TrendPeriod=100` (`registry.go:151-156`) — few signals, and it can only win when the (broken) vol filter lets it through.

### 2.4 TrendRunner (`trend_following`) — the "1 trade" strategy

1. **Trailing-stop bug** (`trend_runner.go:143`): `trailingStop := peakPrice.Float64() - r.StopLoss.Float64()`. `r.StopLoss` is a **price** (entry − stopDist), not a distance, so `trailingStop = peak − (entry − stopDist) = peak − entry + stopDist`. The stop trails `entry − stopDist` below the peak — for a $500 stock that's ~$495 below peak, i.e. **the trailing stop is effectively disabled**; exits happen only on EMA cross or take-profit. This inverts the intended "let winners run, cut losers" asymmetry.
2. **Two-bar confirmation with ADX/chop filters** (`trend_runner.go:174-208`): a cross must be followed by a bar where `ADX ≥ 20` and `Chop ≤ 61.8`, else the pending signal is dropped. Combined with the trailing-stop bug, this yields the observed 1 trade.

### 2.5 RSI2MeanReversionRunner (`rsi2_reversion`) — Sharpe −2.9

1. Stop/target distances from the **close-to-close pseudo-ATR** (§2.1) → tight stops → whipsaw. The 4,405 trades at −2.9 Sharpe are the signature of "right idea, stops too tight, fee bleed".
2. Dead guard `rsi2 > 0` (`rsi2_reversion_runner.go:118,127`) — RSI(2) is never ≤ 0; harmless but signals un-reviewed logic.

### 2.6 SessionScalpRunner (`session_scalp`) — Sharpe −5.0

1. **Timezone default is 0 (UTC)** (`session_scalp_runner.go:49`), but the strategy targets the 9:30–11:00 **ET** session (its own ParamDef comment says "ET = -4/-5"). By default the "opening range" is formed on the wrong clock hours for US equities — trading a pre-/post-market window with no liquidity, or the wrong session entirely.
2. **`EntryBufferPct` units bug** (`session_scalp_runner.go:222`): the param is described as a "percentage buffer" with default `0.1`, but usage divides by 100 → `0.001` = 0.1%. The optimizer sweeps `[0.01, 0.5]` but the applied buffer is 0.01%–0.5% — effectively zero, so entries fire on a single tick past the range (noise).
3. Stop/target from close-to-close pseudo-ATR (§2.1).

---

## 3. Why these aren't obvious from the collected metrics

1. **`StrategyNil` conflates "correctly no trade" with "panic/recovered"** — `engine.go:1438-1446` increments `StrategyNil` both when `Evaluate` returns nil *and* when it panics inside the recover. 1.39M nils tell you nothing about *why*.
2. **No "why nil" breakdown.** `StrategyNil` is a single integer. There is no per-filter reason (trend filter vs vol filter vs session window vs no-crossover), so the broken vol filter (§2.3.1) and the strict trend confirmation (§2.4.2) are invisible behind one big number.
3. **No exit-quality breakdown.** The funnel counts trades, but not *how* they closed (stop-loss vs take-profit vs time-exit vs signal-reverse). A strategy that always stops out would look identical in aggregate to one that takes profits, yet be unprofitable.
4. **No stop/ATR sanity diagnostics.** Nothing records stop distance vs realized range, so a too-tight stop (§2.1) or a disabled trailing stop (§2.4.1) produces "normal-looking" trades with no flag.
5. **Regime duplication is invisible.** The three gate layers collapse into one `RegimeRejected`/`StrategyNil` count, so nobody can see that the same decision is made three times (and that `--regime-off` only removed two of them).
6. **Aggregate-only funnel.** Everything is a column total per combo; there is no per-signal trace and no distribution (e.g., median bars-held, stop-hit rate) that would expose systematic artifacts.

---

## 4. Metric improvements to capture these insights

| # | New metric | Captures | Effort |
|---|---|---|---|
| M1 | **`StrategyNil` reason breakdown** — instrument each runner to report why it returned nil (`trend_filter`, `vol_filter`, `session_window`, `no_crossover`, `below_threshold`, `warmup`) | exposes §2.3.1, §2.4.2, §2.6.1 directly | low–med |
| M2 | **Exit-reason breakdown** — `stop_loss` / `take_profit` / `time_exit` / `signal_reverse` / `eod` counts + median bars-held | exposes §2.1 (tight stops) and §2.4.1 (disabled trailing stop) | low |
| M3 | **Stop/target sanity stats** — mean stop distance, mean target distance, stop-hit rate, in ATR units | a "stop-hit rate ≈ 90%" or "stop distance ≈ 0.2×true-range" flag | low |
| M4 | **Separate `NilError` from `NilNoSetup`** — count panics/recoveries distinctly | exposes runtime bugs in runners | trivial |
| M5 | **Per-layer gate breakdown** — count rejections at each of the three regime layers separately | makes the duplication visible | low |
| M6 | **True-ATR indicator** + a diagnostic that logs the close-to-close vs true-range ratio | validates indicator correctness | low |

---

## 5. Priority recommendations

1. **De-duplicate regime gating** (remove runner-internal `regime == 3`; single canonical pipeline gate) — fixes the `--regime-off` blind spot and matches quant best practice. ✅ **Implemented** (14 runners cleaned; `regime == 3` blocks removed).
2. **Replace the close-to-close `ATR` with true range** (pass high/low) — fixes the systematically-too-tight stops across every strategy. ✅ **Implemented** — `TrueRangeATR` rewritten circular-buffer-aware; all 12+ runners switched; close-only runners now store high/low.
3. **Fix the TrendRunner trailing-stop arithmetic** and the **SessionScalp timezone/buffer units** — two confirmed bugs. ✅ **Implemented** — trailing stop now uses the entry-time ATR *distance* (`trailingStopPrice`); SessionScalp timezone default is `-4` (ET).
4. **Fix the MeanReversion vol filter** — replace the unit-mismatched filter with a proper return-volatility measure. ✅ **Implemented** — `returnVol` (std of returns) replaces the price-level-variance comparison; dead `VolPeriod`/`histVariance`/`atrVol` removed.
5. **Add M2 + M4 + M5 metrics first** (cheapest, highest signal) to make the next round of artifacts visible before running more optimization. ✅ **M4 implemented** (`NilError` separate from `StrategyNil`, fixes a double-count on panic); **M2 implemented** (`ExitReasons` breakdown: stop_loss/take_profit/time_exit/signal_reverse/end_of_data/pnl_clamped). M5 (per-layer gate breakdown) remains.

### Regression tests added (`internal/strategy/regression_test.go`)

- `TestTrueRangeATR_UsesTrueRange` / `_ExceedsCloseToCloseOnGaps` / `_CircularBuffer`
- `TestMeanReversion_ReturnVol`
- `TestGridRunner_NoRegimeCrisisEarlyReturn`
- `TestTrendRunner_TrailingStopPriceUsesDistance`
