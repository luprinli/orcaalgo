# ORB / Grid / Scalp — Diagnosis & Theoretical Alignment

**Date:** 2026-08-15
**Scope:** `opening_range_breakout`, `orb_15m`, `grid_trading`, `vol_grid`, `session_scalp`, `volume_scalp` — the families that remained strongly negative after the true-ATR / trailing-stop / vol-filter / regime fixes (Sharpe −4.7 to −11 across 5k–105k trades). This document diagnoses *why* they deviate from theoretical expectations and what was changed.

---

## 1. Executive summary

The mean-reversion family was fixed by the prior round and now has edge (28 positive `intraday_mr` combos). The ORB/grid/scalp families were still broken for a **different** reason: they **overtrade** with a **negative risk-reward asymmetry**. Three root causes, three fixes:

| Strategy | Observed | Root cause | Fix |
|---|---|---|---|
| ORB (`opening_range_breakout`, `orb_15m`) | 862 trades/combo, Sharpe −11 | no daily trade cap → re-enters after every stop-out | **one entry per day** (`tradedToday`) |
| Grid (`vol_grid`, `grid_trading`) | 1,162 trades/combo, Sharpe −7 | **inverted R:R** (TP 0.5% < SL 1.5%) | **TP 1.0% / SL 0.5%** (2:1 reward:risk) |
| Session scalp | 416 trades/combo, Sharpe −4.7 | wrong session window (UTC) + 10 trades/day cap | **timezone −4 ET** (done) + **3 trades/day** |

---

## 2. ORB — opening-range breakout

### Theoretical expectation
A canonical ORB takes **one position per day** at the range break, holds to stop/target/end-of-day, with a ~2:1 reward:risk. It is a *low-frequency, single-shot* strategy. A healthy ORB does **~1 trade/day (~250/year)**, not 5+.

### What the code was doing
1. **No daily trade cap.** After a stop-out, the next bar that closed back outside the range re-entered — all day long. This is the overtrading bug: the "one decision a day" discipline of ORB was replaced with "breakout ping-pong."
2. **`RangeMinutes` is a bar count, not minutes.** `barsInRange >= RangeMinutes` counts *bars*, so on 5m `range_minutes=5` forms a 25-minute range, on 15m a 75-minute range. (Naming/clarity issue; the semantic "first N bars" is defensible but mislabeled.)
3. **End-of-day exit holds overnight.** `CloseExitMinutes=390` is dead; the position is force-closed only at the *next day's first bar*, carrying overnight gap risk for an intraday strategy.
4. **`EntryBufferPct` units.** `0.1` → `0.1%` (a thin confirmation buffer — acceptable, but the ParamDef range `0.01–1.0` is actually 0.01%–1.0%).

### Fix implemented
- Added `tradedToday` — the runner now takes **at most one entry per day** and resets it on the day rollover. Re-entry after stop/target/EOD is blocked.

### Regression test
`TestOrbRunner_OneTradePerDay` (`internal/strategy/regression_test.go`) feeds a 5-bar range → breakout entry → stop-out exit → asserts the same-day re-entry is blocked.

---

## 3. Grid — mean-reversion grid

### Theoretical expectation
A grid makes money by **mean reversion in a range**: buy at the lower levels, sell at the adjacent upper level. For positive expectancy the **take-profit must be ≥ the stop-loss** (or the grid must have a strong reversion tilt). The classic grid captures the *oscillation* of a range-bound asset, not the trend.

### What the code was doing
1. **Inverted risk-reward.** `TakeProfitPct=0.5`, `StopLossPct=1.5` — each win paid **+0.5%** while each loss cost **−1.5%**. Even at a 75% win rate this is `0.75·0.5 − 0.25·1.5 = 0` *before* costs — and negative after slippage+fees. This is "picking up pennies in front of a steamroller" and is the direct cause of the −7 Sharpe.
2. **Reference-price re-centering chases trends.** Every 100 bars with no open position, the grid re-centers on the current price — so in a trending market it locks in losses and re-anchors into the trend's path.
3. **`MaxOpen=10`** amplifies the negative expectancy ten-fold.

### Fix implemented
- Flipped to `TakeProfitPct=1.0` (the adjacent grid level) and `StopLossPct=0.5` (half-level stop) → a **2:1 reward:risk** per grid fill, matching the ORB convention and giving the mean-reversion edge room to be net-positive after costs.

---

## 4. Session scalp

### Theoretical expectation
A session-open scalp takes **1–3 trades** in the first 90 minutes, high win rate, tight R:R, profiting from the opening momentum/range break.

### What the code was doing
1. **Wrong session window.** `TimezoneOffset` defaulted to `0` (UTC), so the 9:30–11:00 window was evaluated on the wrong clock hours for US equities. (Fixed in the prior round to `−4` ET.)
2. **10 trades/day cap** — far above the 1–3 a session scalp should take; combined with a tight 1.0-ATR stop, it bled fees.

### Fix implemented
- `MaxTradesPerDay` 10 → **3**.

---

## 5. Cross-cutting issues (documented, not yet changed)

| Issue | Impact | Recommendation |
|---|---|---|
| `RangeMinutes` / `entry_buffer_pct` are "bars"/"percent" but named "minutes"/"pct" | search-space/units confusion | rename params or add per-timeframe conversion |
| ORB holds overnight (no true EOD close) | overnight gap risk | add a session-clock EOD exit |
| Grid re-centers in trends | chases trends, locks losses | gate grid to Calm/range (already regime Calm-only) and drop the 100-bar re-center, or make re-center regime-aware |
| `volume_scalp` (−5.0 Sharpe) not yet audited | — | separate follow-up |

---

## 6. Validation protocol (before re-running the matrix)

1. `go test ./internal/strategy/ -run "TestOrbRunner_OneTradePerDay|TestTrendRunner|TestGridRunner"`.
2. Re-run the matrix and compare `Trades`, `Sharpe`, and the new `ExitReasons`/`NilError` columns:
   - ORB `Trades`/combo should drop from ~862 toward ~250 (one/day).
   - Grid `Sharpe` should move from −7 toward 0+ (2:1 R:R).
   - Session scalp `Trades` should drop from ~416 toward ~150 and the session window should now be ET.
3. Confirm the funnel shows fewer `StrategyNil`-from-wrong-session and no `NilError` (no panics).
