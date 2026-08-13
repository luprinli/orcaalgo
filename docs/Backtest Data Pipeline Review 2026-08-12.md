# Backtest Data Pipeline — End-to-End Infrastructure Review

**Date:** 2026-08-12
**Scope:** Read-only review of the full backtest data pipeline — from source ingestion and normalization through downstream generation (resampling, synthetic gap-fill, feature/label construction, dataset splitting) to backtest execution — evaluated against industry-standard quant research data-infrastructure practices. No code changes made.
**Author:** Quant Data Engineer (read-only engagement).

---

## 1. Executive Summary

The pipeline has a **strong conceptual foundation** — BIGINT fixed-point pricing, a canonical 18-symbol universe, a versioned migration system, a source-column + `SourceValues` provenance abstraction, GKR config hashing, and genuine lookahead defenses (point-in-time temporal guard, purged K-fold CV, temporal IR contracts). These are real strengths to preserve.

However, **provenance and idempotency are inconsistently implemented across the multiple ingestion paths**, and this is the direct cause of the data-integrity failures observed in recent backtests. Four findings dominate:

1. **CRITICAL — Source labeling is broken at the primary Python ingestion path.** `orca/data/db_integration.py:upsert_candles` does **not** write the `source` column (it silently falls back to the migration default `'yahoo'`) and uses `ON CONFLICT DO NOTHING`, while `scripts/stooq_seed.py` correctly writes `source='stooq'` and uses `ON CONFLICT DO UPDATE`. Two ingestion paths have contradictory semantics for the same table.

2. **CRITICAL — The `source` column is not part of the uniqueness key** (`idx_candles_unique_bar` is `(symbol_id, timeframe, time)` only). Multiple sources with **incompatible price scales** coexist per bar, and `SourceValues("stooq")` merges the legacy `seed` synthetic fixture with real `stooq` bars, producing ~7–10× discontinuities (NVDA) and wrong index levels (^_US) — the direct cause of the absurd 132%-return / $1,406-average-win results.

3. **HIGH — `generation_id` is non-deterministic and unpersisted.** `seed_all.compute_generation_id` hashes a config that includes `datetime.utcnow()`, so the "deterministic generation ID" changes on every run; it is never written to the `candles.generation_id` column, and backtest results carry no data-version reference. Data lineage is therefore nominal, not operational.

4. **HIGH — Resetting is range-only, not source/symbol-scoped.** `seed_all(reset=True)` issues `DELETE … WHERE time BETWEEN`, so partial reseeds leave stale bars and orphaned symbols; there is no way to rebuild a single symbol/source idempotently.

**Verdict:** The pipeline is not yet trustworthy for reproducible research. Promotion of any backtest to orchestration/live should remain blocked until the provenance/idempotency defects (P0) are remediated and a cost-corrected, out-of-sample-validated baseline is produced.

A fifth, cross-cutting recommendation (§3.7) is the **single highest-leverage reliability lever**: consolidate the two-provider pipeline (Yahoo + stooq) onto **stooq alone** by adding stooq daily data. This eliminates the Yahoo/stooq scale and split-adjustment conflicts at their root rather than papering over them.

---

## 2. Best-Practice Principle Catalog

Legend: ✅ meets · 🟡 partially meets · ❌ fails.

| # | Principle | Rating | Evidence |
|---|---|---|---|
| P1 | Fixed-point / numeric precision (no float for prices) | ✅ | BIGINT `open_raw…close_raw`, `PRICE_SCALE_F = 100_000` (Go `repository_core.go:129`, Python `stooq_seed.py:34`); VIX migrated to BIGINT (migration 000039). HP#2 compliant. |
| P2 | Single source of truth for universe | ✅ | `configs/universe.json` loaded by `orca/universe_config.py` and `config.Tickers()`; `DefaultSymbols` aligned to it. |
| P3 | Versioned schema (migrations) | ✅ | Numbered up/down migrations 000001–000040. |
| P4 | Config versioning + hashing | ✅ | GKR `.gkr.yaml` validation, `StrategyHash`, `orca validate`; deterministic canonical-JSON hash. |
| P5 | Parameter versioning | ✅ | `strategy_params_version` table (migration 000030), JSONB params, IS/OOS metrics. |
| P6 | Data lineage / provenance tracking | 🟡 | `source` + `generation_id` columns exist, but not populated by the primary Python path; `generation_id` non-deterministic (§3.1, §3.2). |
| P7 | Dataset↔result version linkage | ❌ | Backtest results carry `SchemaVersion`/`EngineVersion` but no data `generation_id`; a result cannot be tied to the exact data that produced it. |
| P8 | Idempotent data generation | 🟡 | Unique bar constraint + `ON CONFLICT` exist, but `upsert_candles` uses `DO NOTHING` (revisions never propagate) vs `stooq_seed` `DO UPDATE` (inconsistent); reset is range-only (§3.3). |
| P9 | Stage-wise data validation | 🟡 | `validate_integrity.py` (4 checks) exists but silently returns `passed=True` on exceptions and lacks source/scale-consistency checks (§3.4). |
| P10 | Lookahead bias prevention (point-in-time) | 🟡 | `LoadCandlesUpTo` caps `end` via `temporal.GetMaxTime`; temporal IR contracts (`orca/ports/temporal.py`); but the Go feature-store/ML path is not verified point-in-time end-to-end. |
| P11 | Corporate action (split/dividend) adjustment | ❌ | `c.AdjustmentFactor = 1.0` always (`repository_candles.go:101-102`, "No split/dividend adjustment column exists"). Direct cause of NVDA seed-vs-stooq mismatch. |
| P12 | Source-aware loaders | 🟡 | `LoadCandlesFiltered`/`SourceValues` exist, but `SourceValues("stooq")` includes the incompatible `"seed"` fixture (§3.5). |
| P13 | Deterministic/reproducible execution | 🟡 | `FixedSeed`/`NewFillSimulatorWithSeed` exist, but the default fill simulator is wall-clock-seeded (`slippage.go`); default runs are non-reproducible. |
| P14 | Temporal (non-random) dataset splitting | ✅ | `orca/ml/dataset.py:split_temporal` (train→val→test, earliest→latest). |
| P15 | Leakage-safe CV (purge/embargo) | ✅ | `orca/ml/purge_cv.py` (purged K-fold + embargo). |
| P16 | Multiple-testing correction | ✅ | `orca/sizing/multiple_testing.py` (Bonferroni + BH) and `block_bootstrap.py`. (Exists; not yet applied to matrix promotion.) |
| P17 | Result plausibility guardrails | ✅ | `plausibility.go`, `multi_metric_gate.go`, `validate-matrix.ps1`, anti-pattern scan (18 rules). |
| P18 | Synthetic-data calibration & labeling | 🟡 | `stooq_synthetic.py` calibrates from per-symbol σ/μ (EWMA λ=0.94) and labels `stooq-calibrated`; but the legacy `seed` GBM has no calibration provenance and an incompatible base price (§3.5). |
| P19 | Pipeline orchestration idempotency | ❌ | `orchestrate.py --reset-reseed` aborts mid-pipeline on non-fatal errors (Unicode/DB-password) leaving partial state; retry does not re-run skipped downstream steps (§3.6). |

---

## 3. Component-by-Component Evaluation

### 3.1 Source ingestion & normalization

**Components:** `orca/data/seed_all.py` (Yahoo 1d + 5m + VIX), `scripts/stooq_seed.py` (stooq 1h + 5m), `scripts/stooq_discovery.py` (manifest).

- **Strengths:** streaming ingest (stooq_seed), 100,000× fixed-point settle, `auto_adjust=True` on Yahoo 1d, deterministic manifest mapping.
- **Gaps:**
  - `upsert_candles` (`db_integration.py:44`) never sets `source` → every Yahoo/seed_all bar is stored as the migration default `'yahoo'`, erasing its true origin.
  - `upsert_candles` inserts `ON CONFLICT DO NOTHING` (line 101) — the opposite of `stooq_seed.py`'s `ON CONFLICT … DO UPDATE` (lines 124–130). A symbol's bars written by the "seed" path can never be corrected by a later, better source on the same `(symbol_id, timeframe, time)`.
  - Yahoo 5m is resampled to 15m/30m/1h/4h (`seed_all.py:263-266`) and stored via the same `upsert_candles` (no `source`), so all resampled intraday from the Yahoo path is also mislabeled `'yahoo'`.

### 3.2 Raw storage & lineage

**Component:** `candles` hypertable (`open_raw…close_raw BIGINT`, `source`, `generation_id`, `scenario`, unique `(symbol_id, timeframe, time)`).

- **Strengths:** BIGINT prices; unique bar constraint; `source` + `generation_id` columns present (migration 000040); compression/retention policies.
- **Gaps:**
  - `generation_id` is **never written** by any ingestion path (`upsert_candles` INSERT omits it; `stooq_seed.py` INSERT omits it). Lineage is dead schema.
  - `compute_generation_id` (`seed_all.py:39-42`) hashes a config including `"timestamp": str(datetime.utcnow())` — **non-deterministic** despite the docstring "Deterministic generation ID".
  - The unique key excludes `source`, so a corrected/revised bar for one source collides with another source's bar and is dropped (or, via `DO UPDATE`, overwrites it and flips the label).

### 3.3 Resampling, synthetic generation & reset semantics

**Components:** `orca/data/resample.py`, `scripts/stooq_resample.py`, `scripts/stooq_synthetic.py`, `seed_all(reset=True)`.

- **Strengths:** `stooq_synthetic.py` calibrates per-symbol σ from Close-to-Close + High-Low (EWMA λ=0.94), soft-blends toward the daily close, and correctly labels `stooq-calibrated`; NYSE-calendar-aware trading-day logic exists.
- **Gaps:**
  - `reset=True` truncates by `time` range only (`seed_all.py:222`) — it cannot scope a reset to a symbol or source, so a partial reseed leaves stale bars for untouched symbols and orphaned `symbols` rows (no re-activation is performed by the Python path).
  - The stooq pipeline's resample/gap-fill outputs (`stooq-resampled`, `stooq-calibrated`) are **absent from the DB** — only `stooq` 1h/5m and `seed` are present. The earlier pipeline abort (§3.6) left these steps unrun and they were not re-run on retry.

### 3.4 Feature engineering & label creation

**Components:** `orca/ml/dataset.py` (FeatureDataset, `split_temporal`), `orca/ml/purge_cv.py`, `orca/ml/train/meta_labeling.py`, Go `FeatureStore`.

- **Strengths:** temporal split (not random); purged K-fold with embargo; label meta-labeling documented as lookahead-safe; `FeatureDataset.metadata` carries `win_rate` and split labels.
- **Gaps:**
  - Feature-vector persistence (migration 000016 `feature_vectors`, 000021 `feature_store_state`) exists, but the Go engine's feature-store path is not verified point-in-time end-to-end (no automated leakage test in the anti-pattern/guardian suites was observed).
  - No data-`generation_id` is recorded on datasets, so a dataset cannot be tied to the candle generation that produced it.

### 3.5 Dataset splitting & lookahead prevention

- **Strengths:** `LoadCandlesUpTo` (`repository_candles.go:117-122`) enforces a point-in-time cap via `temporal.GetMaxTime`; `orca/ports/temporal.py` performs static temporal-contract validation on the GKR IR; purged CV with embargo prevents label overlap.
- **Gaps:** the strongest lookahead guards are in the **Python ML/IR** layer; the **Go matrix backtest** path relies on warm-up bars + the (recently fixed) per-symbol regime index, but there is no single enforced point-in-time contract across the whole Go execution path.

### 3.6 Orchestration & reproducibility

**Components:** `scripts/orchestrate.py` (venv → Docker → reseed → build → services), `internal/backtest/engine.go` fill simulator.

- **Gaps:**
  - `orchestrate.py` aborted the first reseed mid-pipeline (DB password, then UTF-8 logging), leaving partial state; the retry did not re-run the aborted stooq resample/calibrate steps. The pipeline is not resumable or all-or-nothing.
  - The default fill simulator seeds from wall clock; without `FixedSeed`, two runs of the same combo produce different Sharpe/Sortino/return. Reproducibility is opt-in, not default.

### 3.7 Provider Consolidation — Retire Yahoo, Adopt Stooq Daily

**Verified current state:** the stooq tree contains only `5_us_txt`/`5_world_txt` (5-minute) and `h_us_txt`/`h_world_txt` (hourly) — there is **no daily stooq data downloaded**. Consequently Yahoo (`seed_all.fetch_candles_yahoo`) is today the **sole 1d source** and the **sole VIX source**, and it *also* fetches 5m and resamples it to 15m/30m/1h/4h — overlapping and conflicting with stooq 1h/5m.

**Premise correction:** "stooq already provides 1d" is not true of the current repo. However, stooq's public dataset family includes daily data (`d_us_txt` / `d_world_txt`) in the **same** CSV format (`<TICKER>,<PER>,<DATE>,<TIME>,<OPEN>,<HIGH>,<LOW>,<CLOSE>,<VOL>,<OPENINT>`), so consolidation is a *drop-in addition* (download + discovery + a small parser tweak for the empty `TIME` field), not a rewrite.

**Recommendation: Yes — retire Yahoo and consolidate on stooq.** This is the single highest-leverage complexity reduction available.

| Timeframe | After (consolidated) | Before (current) |
|---|---|---|
| 1d | stooq daily (`d_*_txt`) | Yahoo (sole source) |
| 1h | stooq hourly | Yahoo-resampled **and** stooq (conflict) |
| 5m | stooq 5m | Yahoo 5m **and** stooq (conflict) |
| 4h | stooq 1h → 4h resample | Yahoo-resampled **and** stooq-resampled (conflict) |
| 15m / 30m | stooq 5m → resample | Yahoo-resampled **and** stooq-resampled (conflict) |
| VIX | stooq `^vix` daily (verify) | Yahoo (sole source) |

**Quantified complexity reduction:**

1. Providers: **2 → 1**; no more cross-provider normalization.
2. Ingestion code paths: removes the Yahoo fetch + Yahoo-5m-resample + VIX + synthetic-sentiment branch of `seed_all.py`; leaves one parser (`_parse_stooq_line`).
3. Source labels collapse to `stooq` / `stooq-resampled` / `stooq-calibrated`; `"yahoo"` and `"seed"` are dropped from `SourceValues` entirely (directly resolving R2).
4. Price scale: one convention (100,000×, stooq-native); the Yahoo-vs-stooq 1d/intraday scale mismatch disappears.
5. Corporate-action convention: **one** convention (stooq is unadjusted across *all* timeframes), which removes the *Yahoo `auto_adjust=True` 1d vs stooq unadjusted intraday* mismatch that produced the NVDA discontinuity. (Requires R4 to be scheduled with this change.)
6. Network flakiness eliminated: stooq = local files → **offline, reproducible** seeding; no yfinance rate-limit/retry surface.
7. `seed_all.py` shrinks to: read stooq daily → resample → infer regime/sentiment.

**Caveats and verification gates before committing:**

1. **Coverage check (mandatory):** confirm stooq `d_us_txt`/`d_world_txt` cover all 18 symbols — especially forex majors (`eurusd`, `gbpusd`, …), crypto (`btc.v`, `eth.v`), and indices (`^spx`/`^dax`). The world tree already proves stooq carries currencies/crypto/indices intraday, but daily coverage must be verified per symbol.
2. **VIX alternative:** verify stooq `^vix` daily exists; if not, keep a single isolated VIX fetch (do **not** keep Yahoo for anything else).
3. **Split adjustment (R4) becomes mandatory:** stooq daily is unadjusted, so a proper adjustment factor is required to avoid NVDA-class jumps. This is a net win — one convention instead of two — but must ship in the same change.
4. **Regime inputs:** switch regime inference from Yahoo 1d closes to stooq daily closes (same function, new source).
5. **Coverage gap:** stooq 1h = ~2-year, 5m = ~5-month; the `stooq-calibrated` synthetic gap-fill still covers the remaining 5-year window (unchanged).

**Why this is the right lever:** every removed ingestion path removes an entire class of failure — format drift, scale mismatch, split-adjustment mismatch, and network non-determinism. A single-provider pipeline is provably simpler to keep idempotent and lineage-correct.

---

## 4. Strengths to Preserve

1. BIGINT fixed-point price storage (100,000×) with a single Go constant `PRICE_SCALE_F` — do not regress to float.
2. Canonical 18-symbol universe in `configs/universe.json` as the single source of truth.
3. Versioned migrations with up/down pairs (000001–000040).
4. `source` column + `SourceValues` selector abstraction (fix its implementation, don't remove it).
5. GKR IR validation + config hashing + `StrategyHash`.
6. Point-in-time `LoadCandlesUpTo` + temporal IR contracts + purged CV.
7. `validate-matrix.ps1` + plausibility gate + anti-pattern scan (18 rules).
8. `validate_integrity.py`'s four-check structure (extend it, don't discard it).

---

## 5. Prioritized Remediation Recommendations

Ranked by impact on data integrity and reproducibility (long-term reliability over quick fixes).

### P0 — Blocking (fix before any further backtest reliance)

| # | Recommendation | Rationale |
|---|---|---|
| R1 | **Make provenance correct end-to-end.** (a) Add `source` and `generation_id` to `upsert_candles`' INSERT and to `stooq_seed.py`; (b) make `compute_generation_id` truly deterministic (drop the `datetime.utcnow()` term; hash symbols+range+parameters only); (c) persist `generation_id` on every inserted/updated candle row. | Single, correct lineage record is the precondition for every other guarantee. |
| R2 | **Make source part of the bar identity.** Add a `source` column to the unique key (or a per-`generation_id` partition) and load the *highest-priority* source only, rather than merging all sources. Concretely: `SourceValues("stooq")` must **not** include `"seed"`; legacy fixtures should be excluded or migrated. | Eliminates the ~7–10× scale discontinuities that produce absurd results. |
| R3 | **Unify ingestion conflict semantics.** Standardize on `ON CONFLICT … DO UPDATE` (latest-wins, idempotent) for all candle writers, or — better — introduce a `valid_from`/`supersedes` lineage model. `DO NOTHING` silently preserves stale bars. | Restores true idempotency and lets revised data propagate. |
| R4 | **Add corporate-action adjustment.** Populate an adjustment factor (or store both raw and adjusted) and apply split/dividend adjustment on load, replacing the identity `AdjustmentFactor = 1.0`. | Fixes NVDA and any future split/divergent series. |
| R5 | **Make reset source/symbol-scoped and pipeline all-or-nothing.** `reset` should delete by (symbol, timeframe, source) and re-activate canonical symbols; orchestrate.py must be resumable or transactional across its 6 reseed steps. | Prevents partial/stale state and orphaned rows. |
| R16 | **Consolidate on a single provider — retire Yahoo, adopt stooq daily.** Add `d_us_txt`/`d_world_txt` ingestion, retire the Yahoo 1d/5m/resample/VIX path, and drop `"yahoo"`/`"seed"` from `SourceValues` (see §3.7). | Removes the root cause of cross-provider scale/split mismatches and eliminates an entire failure class. Highest-leverage reliability lever. |

### P1 — High

| # | Recommendation | Rationale |
|---|---|---|
| R6 | **Link backtest results to data `generation_id`.** Add the data generation_id to `BacktestResult`/`ComboResult` and emit it in the CSV. | A result becomes auditable against the exact data that produced it. |
| R7 | **Add source/scale-consistency checks to `validate_integrity.py`** and stop swallowing exceptions as `passed=True`. Add a check that the merged price series has no cross-source discontinuities > X%. | Catches the NVDA/^_US class of defect at seed time, not at analysis time. |
| R8 | **Default-deterministic fills.** Make `FixedSeed` the default (or persist the seed per run); keep wall-clock seeding opt-in. | Reproducible results are a precondition for any gate. |
| R9 | **Complete the stooq resample/calibrate steps and verify presence.** After R2, ensure `stooq-resampled` (4h/15m/30m) and `stooq-calibrated` gap-fill are present and non-empty; add a presence assertion to the reseed. | 15m/30m/4h are currently on legacy synthetic data. |
| R10 | **Fix ^_US/^DAX index ticker mapping and NVDA base price.** | Correct absolute levels for index and split-adjusted symbols. |

### P2 — Medium

| # | Recommendation | Rationale |
|---|---|---|
| R11 | Add a leakage test to the guardian/anti-pattern suites that asserts feature-store point-in-time correctness in the Go engine. | Makes P10 a verified guarantee, not an assumption. |
| R12 | Version datasets (a `dataset_id` referencing `generation_id` + feature/label code hash). | Full dataset lineage for ML artifacts. |
| R13 | Record the seed/parameters (σ, μ, λ, base_price) in the synthetic bar metadata (`scenario` column or a sidecar). | Synthetic data becomes reproducible and auditable. |
| R14 | Normalize candle timestamps to exchange-local time (or a declared timezone) at ingestion. | Resolves session-window strategies' UTC-vs-ET mismatch. |
| R15 | Apply multiple-testing correction (Bonferroni/BH) + walk-forward OOS as a mandatory gate before any promote-to-live. | Closes the selection-bias gap on the 45–73 positive-Sharpe combos. |

---

## 6. High-Level Implementation Roadmap

**Phase 0 — Provider consolidation (P0, ~2–3 days + download):**
R16 (retire Yahoo, adopt stooq daily). Steps: (a) download `d_us_txt`/`d_world_txt`; (b) extend `stooq_discovery.py` for the `d_` trees; (c) extend `stooq_seed.py` to parse the empty `TIME` field for daily bars; (d) rewire `seed_all` to read stooq daily + infer regime from it; (e) drop `"yahoo"`/`"seed"` from `SourceValues`. *Gate:* all 18 symbols have verified stooq daily coverage and VIX has a non-Yahoo source. Do this **before** Phase 1 so provenance work targets a single provider.

**Phase 1 — Provenance & identity (P0, ~1–2 weeks):**
R1 → R2 → R3 → R5, then re-seed cleanly. Deliverable: every candle row has a correct `source` and `generation_id`; no source-merging of incompatible scales; reseed is idempotent and scoped. *Gate:* `validate_integrity` (extended per R7) passes with zero source-discontinuity errors.

**Phase 2 — Correctness of levels (P0/P1, ~1 week):**
R4 (corporate actions) → R10 (index/split mapping). Deliverable: NVDA, ^_US, ^DAX price levels agree across sources; no `AdjustmentFactor=1.0` blind spots.

**Phase 3 — Reproducibility & lineage (P1, ~1 week):**
R6 → R8 → R9 → R12 → R13. Deliverable: a backtest result and its dataset are fully reproducible from a pinned `(generation_id, seed, code hash)`.

**Phase 4 — Validation hardening (P1/P2, ~1 week):**
R7 (source/scale checks) → R11 (leakage test) → R14 (timezone) → R15 (multiple-testing gate). Deliverable: the CI/guardian pipeline blocks promotion on data-integrity or leakage violations.

**Phase 5 — Re-baseline & promote (final):**
Clean re-seed → full matrix re-run → cost-adjusted, walk-forward-validated, multiple-testing-corrected gate → promote only surviving combos.

---

## 7. Bottom Line

The pipeline's architecture is fundamentally sound and already contains most of the right primitives (fixed-point storage, canonical universe, versioned schema, temporal guards, plausibility gates). The reliability failures trace to **inconsistent provenance and idempotency across the multiple ingestion paths**, not to a missing concept. Remediation should prioritize R1–R5 (provenance, source identity, conflict semantics, corporate actions, scoped reset) before any further reliance on backtest outputs for promotion decisions.
