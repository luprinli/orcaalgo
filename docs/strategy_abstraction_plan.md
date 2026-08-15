# Strategy Abstraction Plan — Eliminate Recurring Bug Classes

**Date:** 2026-08-15
**Status:** Plan (not yet implemented)
**Goal:** Convert the bug classes discovered across the strategy audits into shared abstractions + CI-enforced guards, so no current or future strategy can silently reintroduce them. Clean code and best practice are prioritized over convenience: each abstraction removes a whole *class* of bug, not just the one instance we fixed.

---

## 0. The bug classes (what we are defending against)

| # | Bug class | Instance we already fixed | Abstraction |
|---|---|---|---|
| 1 | Canonical indicator silently wrong | close-to-close `ATR` named "ATR" | indicator library + golden tests + no-inline-math lint |
| 2 | Gating duplicated in runners | `if regime == 3 { return nil }` in 14 runners | anti-pattern Rule 14 |
| 3 | Price-vs-distance units confusion | `peak − StopLoss(price)` trailing stop | typed `StopModel` |
| 4 | Inverted/illegal parameterization | grid `TP < SL` | param sanity validator |
| 5 | Overtrading (no frequency cap) | ORB re-entry, scalp 10/day | `DailyTradeCounter` |
| 6 | Ring-buffer linear-indexing | `indicator.*` over a circular buffer | `HistoryBuffer` |
| 7 | Funnel counting drift | `StrategyNil` double-count on panic | funnel-sum invariant test |
| 8 | Session-clock default | scalp timezone `0` (UTC) | shared `Session` type |

---

## 1. Canonical indicator library + golden tests

**Purpose.** One source of truth for every indicator. A bug in an indicator silently corrupts *every* strategy that uses it (the `ATR` case). No runner may compute an indicator inline.

**Design.**
- `internal/strategy/indicators.go` remains the only indicator file. Deprecate the close-to-close `ATR` (or rename to `CloseToCloseATR`) so the misleading name can't be re-imported; the canonical stop/vol indicator is `TrueRangeATR`.
- New `internal/strategy/indicators_golden_test.go`: a golden/reference-value test for **every** exported function (`TrueRangeATR`, `EMA`, `SMA`, `Mean`, `StdDev`, `ZScore`, `ADX`, `RSI`, `RSI2`, `MACD`, `BollingerBands`, `StochasticOscillator`, `IchimokuCloud`, `DonchianChannel`, `KeltnerChannel`, `WilliamsR`, `Aroon`, `MFI`, `VWAP`, `ForceIndex`, `ChandelierExit`, `OBV`, `CMF`, `ChoppinessIndex`). Each test asserts a hand-computed or reference value on a small deterministic input (pattern: `TestTrueRangeATR_UsesTrueRange`).
- Extend `scripts/anti_pattern_scan.py` with **Rule 14a** (lint): flag indicator math inside `internal/strategy/*_runner.go` — i.e. rolling sums/`math.Sqrt`/`math.Abs`/`math.Max` computed directly over `PriceHistory`/`closeHistory`/`HighHistory` in a runner file. Indicator computation belongs in `indicators.go`.

**Tests / acceptance.** `go test ./internal/strategy/` includes the golden suite; `anti_pattern_scan.py` fails on a runner containing inline indicator math.

**Effort.** Medium (one golden test per indicator; ~24 tests). **No behavior change.**

---

## 2. Anti-pattern Rule 14 — "no gating in runners"

**Purpose.** Enforce the architectural invariant that caused two silent bugs (regime triplication, funnel gap): **a runner is a pure signal generator; the RiskPipeline decides whether to act.**

**Design.**
- `scripts/anti_pattern_scan.py` new `check_rule_14`: for each `internal/strategy/*_runner.go`, flag a `return nil` (or a `nil` signal) whose decision references the `regime` parameter or a global market-state gate (volatility halt / choppiness / ADX *used as a hard block*).
  - Precise, low-false-positive rule: flag `if regime == <n> { return nil }` / `if regime != <n> { return nil }` (regime gating — the pipeline already gates on regime). Vol/halt gating is flagged only when the condition is a *global* state (`isHighVol`, `VolatilityHalt`, `ChoppinessIndex > threshold`) not a *signal* condition (z-score, RSI crossover, EMA cross).
- Severity `HIGH`; spec ref `§9.1.3 / strategy_logic_audit §1`.

**Tests / acceptance.** A synthetic regression fixture (a runner with `if regime == 2 { return nil }`) is flagged; the real 14 runners (already cleaned) are not.

**Effort.** Low. **No behavior change** (runners already cleaned).

---

## 3. Typed `StopModel` (price vs distance)

**Purpose.** The trailing-stop bug survived because a stop **price** (`types.Price`) and a stop **distance** (`float64`) are both numeric. Make the distinction structural.

**Design.**
- New `internal/strategy/stop_model.go`:
  ```go
  // distance is ALWAYS relative (float64); stop is ALWAYS absolute (types.Price).
  func StopPrice(entry types.Price, distance float64, side string) types.Price
  func TrailingStop(peak types.Price, distance float64, side string) types.Price
  ```
- Refactor every runner to use these for stop/target placement. `TrendRunner.trailingStopPrice` becomes `TrailingStop`; each `OpenPosition(..., types.PriceFromFloat(price - stopDist), ...)` becomes `OpenPosition(..., StopPrice(price, stopDist, side), ...)`.

**Tests / acceptance.** `TestStopModel_*`: stop is *exactly* `distance` from entry/peak for both sides; a regression test documents the original bug (passing a *price* where a *distance* is expected no longer compiles or produces a nonsensical stop that the test rejects).

**Effort.** Low–medium. **Behavior-neutral refactor** with tests.

---

## 4. Param sanity validator

**Purpose.** Turn inverted/illegal parameterization (grid `TP < SL`) into a hard error at construction and during optimization.

**Design.**
- New `internal/strategy/param_validation.go`:
  ```go
  // ValidateParams returns a list of human-readable violations for a strategy's
  // params; empty when sane. Keyed by the standard cross-strategy param names.
  func ValidateParams(strategyName string, params map[string]float64) []string
  ```
- Cross-cutting checks: `take_profit < stop_loss` (inverted R:R) → error; `entry_z <= exit_z` → error; `atr_multiplier/target_multiplier/position_scale <= 0` → error; `max_hold/max_open/max_trades_per_day/grid_levels < minimum` → error; `kelly_fraction > 1` → error.
- Wire into: (a) a shared `SetParams` wrapper used by all runners, and (b) `internal/backtest/light_optimizer.go` candidate generation (skip/penalize invalid candidates). Add a CI/unit test that runs `ValidateParams` against every strategy's `registryDefaultParams` (via `internal/strategy/registry.go`).

**Tests / acceptance.** `TestValidateParams_RejectsInvertedRR`; `TestAllRegistryDefaultsValid` passes with the *current* (post-fix) defaults; the grid's old `TP 0.5 / SL 1.5` is demonstrably rejected.

**Effort.** Low–medium. **No behavior change** for valid params; hard-fails invalid ones.

---

## 5. `DailyTradeCounter` in `BaseRunner`

**Purpose.** Centralize day-scoped trade frequency so the overtrading class (ORB re-entry, scalp 10/day) is impossible to hand-roll inconsistently.

**Design.**
- Add to `BaseRunner`:
  ```go
  tradeDay    string
  tradesToday int
  // canTrade reports whether another entry is allowed for the given timestamp.
  func (b *BaseRunner) CanTrade(t time.Time, maxPerDay int) bool
  func (b *BaseRunner) RecordTrade(t time.Time)
  ```
- Refactor `OrbRunner` (`tradedToday`) and `SessionScalpRunner` (`dailyTradeCount`/`currentDay`) to use it; adopt it in `volume_scalp`, `dragon_trend`, and any other intraday runner that re-enters intraday.

**Tests / acceptance.** `TestBaseRunner_DailyTradeCounter` (rollover resets; cap respected). The ORB one-trade-per-day test stays green.

**Effort.** Low. **Behavior-neutral refactor** (same caps, centralized).

---

## 6. `HistoryBuffer` type

**Purpose.** Eliminate ring-buffer linear-indexing bugs (`indicator.*` over a circular buffer). This is the deepest structural fix and is staged last.

**Design.**
- New `internal/strategy/history.go`:
  ```go
  type HistoryBuffer struct { … }
  func (h *HistoryBuffer) Push(p, high, low types.Price, vol float64)
  func (h *HistoryBuffer) Count() int
  // Linearized, oldest→newest windows for the indicator library.
  func (h *HistoryBuffer) LastPrices(n int) []float64
  func (h *HistoryBuffer) LastHighs(n int) []float64
  func (h *HistoryBuffer) LastLows(n int) []float64
  func (h *HistoryBuffer) LastVolumes(n int) []float64
  ```
- Migrate `BaseRunner` to embed `HistoryBuffer` (replacing `PriceHistory/HighHistory/LowHistory/VolumeHistory/HistIdx/HistCount`).
- Phase the migration: (a) introduce `HistoryBuffer` + keep the raw fields for one transition, (b) migrate indicators to accept linear windows, (c) migrate runners to `LastPrices(n)`/`LastHighs(n)`/…, (d) remove the raw fields.
- Note: `indicator.*` (cinar) expects linear slices; `LastX(n)` provides them, so the `count/period` juggling in `EMA/SMA/StdDev/ATR` simplifies away.

**Tests / acceptance.** `TestHistoryBuffer_CircularOrder` (push > size, assert chronological order). Full `go test ./internal/strategy/` stays green after each sub-step.

**Effort.** High (touches every runner). **Behavior-preserving**, done incrementally.

---

## 7. Funnel-sum invariant test

**Purpose.** Catch counting drift (e.g. the `StrategyNil` double-count) as a deterministic test, not a post-hoc metric mystery.

**Design.**
- New `internal/backtest/funnel_invariant_test.go`:
  - Derive the exact accounting equation from the `generateSignal` increment sites (single discovery step; reference `internal/backtest/engine.go`):
    `SignalAttempts == RegimeRejected + NilError + StrategyNil + ExitSignalZeroQty + VolHalted + PipelineRejected + MLRejected + FillRejected + CapitalZero + RateLimited + BaseSizeZero + QuantityTooSmall + ExposureBlocked + SignalsPassed`.
  - Unit test: construct a `SignalDiag` with a known breakdown and assert the identity.
  - Integration test: run a short deterministic backtest and assert `SignalsPassed == TradesOpened` and the sum identity holds.
- Extend `SignalFunnelJSON` to emit the invariant residuals for observability.

**Tests / acceptance.** The identity test fails if any increment site is added/removed without updating the accounting.

**Effort.** Low–medium.

---

## 8. Shared `Session` clock

**Purpose.** One timezone/session mapping so a strategy never ships with a UTC default for a US-equity session.

**Design.**
- New `internal/strategy/session.go`:
  ```go
  type Session struct {
      StartHour, StartMin, EndHour, EndMin int
      TimezoneOffset int // documented: ET = -4 (EDT) / -5 (EST)
  }
  func (s Session) InWindow(t time.Time) bool
  func (s Session) DayKey(t time.Time) string
  ```
- Refactor `SessionScalpRunner` and `OrbRunner` to use it; default `Session` instances for US equities (9:30–16:00 ET) and crypto (24h) are exported constants.

**Tests / acceptance.** `TestSession_InWindow` (ET boundary cases, offset wraparound). The scalp timezone regression test stays green.

**Effort.** Low.

---

## 9. Additional bug classes (second pass)

A second pass over the same audits surfaces more recurring classes. Some are **confirmed** (evidence already in the code), others are **verify-first** (need a discovery step before implementing). Items 9–18 fold into the existing phases below.

### Correctness / execution — verify first

**9. Fill-vs-signal bar offset (look-ahead in execution).**
If a runner decides on bar `t`'s close and fills at bar `t`'s close, that is look-ahead (the price is only known after the bar completes). **Action:** discover the engine's actual fill timing (`FillSimulator`), then add an invariant test that the fill bar strictly follows the signal bar (or document a deliberate same-bar-close exception).

**10. Same-bar stop-vs-target precedence.**
When a single bar's range contains both the stop and the target, the exit outcome is ambiguous. **Action:** discover which check runs first, then enforce one deterministic, conservative rule (stop wins on the same bar) across every runner.

### State / determinism — confirmed

**11. Deterministic iteration (`grid_runner.go` `range openPositions`).**
The grid iterates `openPositions map[int]*gridPosition` to close positions and derives `exitSide` from the last-visited entry — Go map order is random, so a multi-close bar is non-deterministic. **Fix:** iterate a sorted key slice (or store positions in an ordered slice).

**12. Concurrent runner isolation (`Get` vs `Create`).**
`Registry.Get` returns a shared singleton; `Create` returns a fresh factory instance. Any backtest/matrix path that uses `Get` shares mutable runner state across goroutines → raced, non-deterministic results. **Fix:** an anti-pattern rule (or debug assertion) that backtest/matrix paths only use `Create`.

### Typing / convention — confirmed

**13. Typed `Signal` (replace the `Quantity == 0` sentinel).**
The "0 quantity = exit/flat" convention is overloaded and fragile — a 0-quantity *entry* is silently swallowed (`ExitSignalZeroQty`). **Fix:** `Signal.Action` enum (`Entry` / `Exit` / `None`) and remove the quantity sentinel.

**14. Regime enum (named constants).**
`regime int8` is compared to magic numbers (`0/1/2/3`) across `RegimeExitMults`, the grid switch, and the matrix. **Fix:** `RegimeCalm/Trending/HighVol/Crisis` constants + a lint that rejects raw `regime == N`.

**15. Percent-vs-fraction unit convention (folds into #4).**
`entry_buffer_pct`/`grid_spacing_pct` are percent (`/100`), while `position_scale`/`kelly_fraction` are fractions — the same literal `0.5` means different magnitudes across params. **Fix:** explicit `_pct` / `_bps` / `_frac` suffixes; the param validator (#4) enforces the numeric range matches the declared unit.

**16. Warm-up vs indicator minimum.**
A strategy's `histCount >= lookback` warm-up can be shorter than its indicator's own minimum (Ichimoku needs 52 bars, ADX needs `2×period`), yielding garbage values early. **Fix:** a per-strategy `minWarmUp` derived from its indicators (or a shared `WarmUpBars` helper + lint).

### Parity / robustness — verify first

**17. Live/backtest param plumbing parity.**
We fixed `regime_w_*` parity, but the other optimizable params (`stop_mult_highvol/crisis`, `profit_mult_trending`, and per-strategy params) must also reach the live engine. **Action:** a parity test asserting the live `SetParams` path applies exactly the map the backtest optimizer emits.

**18. Numeric guards (NaN/Inf/zero).**
Division by a zero-volume/zero-price bar can produce NaN/Inf that silently corrupts indicators. **Fix:** a shared `finite()` guard and the convention that every indicator returns 0 on non-finite input (with a test).

---

## Phased rollout (clean code first)

| Phase | Items | Rationale |
|---|---|---|
| **0 — baseline** | `go build/test`, `anti_pattern_scan`, `pytest` green | confirm the starting point |
| **1 — guards (additive, CI-enforced, zero behavior change)** | #2 (Rule 14), #4 (param validator + #15 units), #7 (funnel invariant), #1 (golden tests), #14 (regime enum), #18 (numeric guards) | converts silent bugs into CI/test failures without touching runtime behavior |
| **2 — typed helpers + confirmed fixes** | #3 (StopModel), #5 (DailyTradeCounter), #11 (grid determinism), #13 (typed `Signal`), #16 (warm-up) | small, test-backed refactors; remove price/distance, overtrading, non-determinism, and sentinel classes |
| **3 — structural** | #8 (Session) then #6 (HistoryBuffer), #12 (runner isolation) | #8/#12 are small; #6 is the largest, done last with incremental migration |
| **4 — discovery-first (verify, then implement)** | #9 (fill offset), #10 (stop/target precedence), #17 (live parity) | each needs a code-reading step before the fix; landed as invariant tests, not blind changes |

### Phase 1 status — ✅ implemented (2026-08-15)

- **#14 Regime enum** — `internal/strategy/regime.go` (`RegimeCalm/Trending/HighVol/Crisis` + `RegimeName`); `RegimeExitMults` + grid switch now use named constants.
- **#18 Numeric guards** — `finite()` in `indicators.go`; `PushPrice` skips non-finite candles; `Mean`/`sampleStd` guard non-finite/negative.
- **#4 Param validator** — `internal/strategy/param_validation.go` (`ValidateParams` hard errors + `WarnParams` inverted-R:R advisory); `TestAllRegistryDefaultsValid` passes.
- **#1 Golden tests** — `internal/strategy/indicators_golden_test.go` (golden + invariant for every indicator); extracted `sampleStd` from 4 duplicated inline `math.Sqrt(variance…)` sites.
- **#2 Anti-pattern Rules 14/15** — `check_rule_14` (no `if regime == …` gating in runners), `check_rule_15` (no inline `math.Sqrt(variance…)` std); both currently report 0 violations.
- **#7 Funnel invariant** — `internal/backtest/funnel_invariant_test.go`; verified `accounted == SignalAttempts` and `SignalsPassed == TradesOpened` across 4 strategies on a wired-pipeline backtest.

### Phase 2 status — ✅ implemented (2026-08-15)

- **#11 Grid determinism** — `grid_runner.go` now iterates `openPositions` in sorted key order (was a random-order map `range`, making multi-close `exitSide` non-deterministic).
- **#5 DailyTradeCounter** — `BaseRunner.CanTrade(t, maxPerDay)` / `RecordTrade(t)`; `OrbRunner` (one/day) and `SessionScalpRunner` (3/day) refactored off their ad-hoc daily counters.
- **#3 StopModel** — `internal/strategy/stop_model.go` (`StopPrice`/`TargetPrice`/`TrailingStop`); `TrendRunner` refactored (trailing stop + entry). Price-vs-distance is now structural.
- **#13 Typed Signal** — `Signal.Action` enum (`SignalNone/Entry/Exit`) replaces the `Quantity == 0` sentinel; all 18 runners set `Action`, the engine keys off `Action`, and the signal-action constants are mirrored into `backtest`.
- **#16 Warm-up** — verified the indicator-heavy strategies' warm-up guards already exceed their indicator minimums; added `TestWarmUpNoEarlySignals`. **Also found + fixed a real bug**: `IchimokuRunner` passed `PriceHistory` as highs/lows to `IchimokuCloud` — now `HighHistory/LowHistory/PriceHistory`.

### Phase 3 status — ✅ implemented (2026-08-15)

- **#8 Session** — `internal/strategy/session.go` (`Session` + `InWindow`/`DayKey`/`NewETSession`); `SessionScalpRunner` refactored to use it.
- **#12 Runner isolation** — verified the backtest engine already uses `Create` (fresh) not `Get` (shared singleton); added `TestRegistry_CreateFreshInstances`.
- **#6 HistoryBuffer / circular-buffer fix** — `internal/strategy/history.go` (`HistoryBuffer` + `linearWindow` + `LastX(n)`), `BaseRunner.LinearPrices/Highs/Lows/Volumes(n)`. Migrated all 24 `indicator.*` wrapper call sites (ma_crossover, rsi2, keltner, ichimoku, carry, momentum, donchian) to linearized windows — fixing the scrambled-window bug once the ring buffer wraps. **Also found + fixed two more close-for-high/low bugs** (`KeltnerChannel` and the `IchimokuCloud` prev-bar computation both passed `PriceHistory` as highs/lows).
- **Test recalibration** — `TestE2E_*` thresholds updated: the circular-buffer + close-for-high/low fixes remove the spurious signals the scrambled window used to emit on synthetic daily data.

### Phase 4 status — ✅ implemented (2026-08-15)

- **#9 Fill-vs-signal bar offset — LOOK-AHEAD FOUND AND FIXED.** The engine filled entries at the *signal bar's close* (the close it used to decide) — classic look-ahead. Implemented **NEXT_BAR execution**: a signal on bar t is deferred and filled at bar t+1's open (`executeEntry` helper + `pendingEntries` map + end-of-data flush), so the entry can no longer look-ahead. `TestNextBarEntry_DelayedFill` verifies fills are strictly after the first candle and that `SignalsPassed == TradesOpened` holds with the deferred fill.
- **#10 Same-bar stop-vs-target — already correct, locked with a test.** The engine checked stop before take-profit (conservative stop-first). Extracted `resolveStopTarget` and added `TestResolveStopTarget_StopFirst` / `_TakeProfitOnly`.
- **#17 Live/backtest param parity — GAP FOUND AND FIXED.** The backtest applied `SetRegimeExitParams` (`stop_mult_highvol/crisis`, `profit_mult_trending`) but the live engine did not. `LiveEngine.RegisterAccountStrategies` now applies them, mirroring the backtest path.

> **Impact note:** NEXT_BAR (#9) is the single biggest correctness change in this plan — it changes every backtest's entry price and therefore every metric. A fresh matrix re-run is required to re-establish the baseline, and any previously-promoted strategy must be re-validated against the corrected fill model.

**Ordering principle:** every phase leaves the build/tests/scan green. Items 1, 2, 4, 7, 14, 15, 18 are *guards* (make future bugs fail CI); items 3, 5, 11, 13, 16, 8, 6, 12 are *abstractions/fixes* (make the bug class unrepresentable); items 9, 10, 17 are *verifications* (confirm correctness and lock it with an invariant test).

---

## Invariants (must hold after each phase)

1. A runner **emits signals only** — no regime/global-gating, no inline indicator math.
2. All **absolute prices** are `types.Price`; all **distances** are `float64` (enforced by `StopModel`).
3. Every indicator is **canonical and golden-tested**, and returns `0` on non-finite input.
4. Every strategy's **default params pass `ValidateParams`** (including percent/fraction unit checks).
5. The **funnel columns sum** to `SignalAttempts`.
6. Intraday runners respect a **shared daily trade cap**.
7. **Regime values are named constants**, never raw `int8` comparisons.
8. **Signals are typed** (`Action` enum), never a quantity sentinel.
9. **Backtests are deterministic**: no map-iteration-order dependence, no shared runner state (`Create`, not `Get`).
10. **Execution is point-in-time**: fills never precede the signal bar (verified; exceptions documented).
11. `go build ./...`, `go test ./internal/...`, `anti_pattern_scan.py`, `pytest tests/` all pass.

## Risks / notes

- **Rule 14a (no-inline-math) must not flag legitimate signal logic.** The rule targets *market-state gating*, not *signal conditions* (z-score, crossover). Fixtures + the real runner set are used to tune the rule and avoid false positives.
- **`HistoryBuffer` migration is the riskiest item.** It is staged with a dual-field transition period and incremental commits, never a big-bang rewrite.
- **The funnel-sum identity must be derived from code, not guessed.** Phase 1 includes a discovery step that reads every `signalDiag.*++` site before asserting the equation.
- **Phase 4 items (fill offset, stop/target, live parity) are the highest-stakes.** A wrong conclusion here silently invalidates every backtest; each is landed as an *invariant test* derived from the discovered behavior, and any behavior change is committed separately with a before/after matrix comparison.
- **Typed `Signal` (#13) and `HistoryBuffer` (#6) both touch every runner.** They are sequenced in different phases so a partial rollout never leaves the tree broken.
