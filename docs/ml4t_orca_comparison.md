# Orca ↔ Machine-Learning-for-Trading (ML4T) Comparison & Leverage Analysis

**Date:** 2026-08-14
**Scope:** `data/machine-learning-for-trading-main` (Stefan Jansen, *ML for Trading 3rd ed.*) vs. Orca (`Orca_algo`), with **Orca as the exclusive target** for improvement recommendations.
**Constraint (non-negotiable):** every recommendation must *not* add unnecessary complexity. All changes below prioritize streamlining existing workflows, reducing technical debt, and hardening reliability/accuracy over adding unproven or niche features. Every recommendation is mapped to Orca's existing stack (Go `internal/`, Python `orca/`, React `web/`) and existing conventions (HP #1–#18, `orca preflight`/`calibrate`, RiskPipeline, GKR IR).

---

## 1. Executive Summary of Key Findings

ML4T is an **educational book-companion repository** (27 chapters + 9 end-to-end case studies) that also ships a genuinely production-grade *research* spine: a content-addressed SQLite run-log registry, calendar-aware walk-forward CV with purge/embargo, and one of the most complete anti-overfitting toolkits in open source (Deflated Sharpe Ratio, Rademacher Anti-Serum, CSCV Probability-of-Backtest-Overfitting, MinTRL, block bootstrap, Newey-West/HAC SE, conformal prediction). It explicitly declares itself *not* a production trading system.

Orca is a **production multi-language trading platform** (Go services + Python domain/math + React dashboard) with real broker integrations, a CI-enforced RiskPipeline, TimescaleDB storage, capital-pool/prop-firm enforcement, and hardened operational controls (kill-switch re-entrancy guard, circuit breakers, rate limiting, JWT revocation, typed errors).

The comparison yields a clear, asymmetrical picture:

- **Where Orca already matches or exceeds ML4T** (preserve, do not rework): walk-forward with purge/embargo, fractional Kelly, fixed-point prices, three-state circuit breakers, deterministic hashing, multiple-testing correction, point-in-time/look-ahead discipline, structured logging, typed errors, coverage gates, mutation testing, and a cost model that *already contains* square-root volume impact, adverse selection, and fill probability.
- **Where ML4T is genuinely ahead and Orca can adopt cheaply** (the leverage): (1) **statistical calibration of the cost model** — ML4T *estimates* spread/impact from data (Corwin–Schultz, Roll, Kyle λ, square-root η fit) while Orca's `SlippageModel` ships calibrated-by-hand constants; (2) **deflated-Sharpe-class selection-bias control** — DSR/RAS/CSCV-PBO, which Orca's Bonferroni+BH+block-bootstrap gate does not cover; (3) **structural CI ratchets** — AST/source scans that fail PRs introducing look-ahead leakage, a pattern Orca partially has (`anti_pattern_scan.py` HP #17) and can extend for ~1 day of effort; (4) **Polars+Parquet+`uv.lock` reproducibility hygiene** as a model for hardening Orca's Python data pipeline and lockfile pinning.

**Highest-leverage conclusion:** Orca does not need ML4T's breadth (RL execution, deep portfolio managers, knowledge graphs, text/NLP). It needs four **targeted, low-effort, high-impact** adoptions: (a) a `orca calibrate-costs` module that fits `SpreadBps`/`VolumeImpactFactor` from the OHLC data already in the DB; (b) a `DeflatedSharpe`/`RAS`/`PBO` addition to `orca/sizing/` feeding the existing `promotion_gate`; (c) a look-ahead/temporal-contract CI ratchet rule; (d) `uv.lock` + single-source tool pinning. These are detailed in §5 and §6.

---

## 2. Full Catalog of Similarities

Organized by component/module. Each entry: ML4T location → Orca location.

### 2.1 Deterministic, content-addressed artifact identity
- **ML4T:** `case_studies/utils/registry/specs.py` — `training_hash = SHA256(canonical_json(spec))[:12]`; canonical JSON sorts keys, rejects non-finite floats, strips non-portable metadata; prediction/backtest hashes chain off parent hashes. SQLite `run_log/registry.db`.
- **Orca:** GKR IR deterministic hashing (canonical JSON + SHA-256) for `.gkr.yaml` strategy configs; `configs/strategies/*.gkr.yaml` with versioning + type validation. Same canonical-JSON + reject-non-finite discipline.

### 2.2 Canonical math kept out of the hot path / single source of truth
- **ML4T:** math lives in the `ml4t-diagnostic` library (DSR, bootstrap, Sharpe SE); case-study code imports, never reimplements.
- **Orca:** HP #1 — Kelly/Brier/Platt/Wilson/EWMA exist only in `orca/sizing/` + `orca/math/`; Go references via subprocess or import. Identical philosophy.

### 2.3 Walk-forward validation with purge/embargo
- **ML4T:** `utils/cv_splits.py` → `WalkForwardCV` (calendar-aware, backward stepping, `label_buffer`/`outcome_horizon` purge, `_purge_holdout_touching_validation`).
- **Orca:** `internal/backtest/optimized_walk_forward.go` — `WalkForwardConfig` with `PurgeTradingDays: 5` and `EmbargoTradingDays: 2` (lines 478–479); `GenerateWalkForwardWindows`; `cmd/matrix-runner --walk-forward`; `internal/scheduler/reoptimization.go` degradation-triggered re-optimization.

### 2.4 Anti-overfitting / multiple-testing correction
- **ML4T:** BH-FDR (`20_strategy_synthesis/02_feature_evaluation.py`); block bootstrap, DSR, RAS, White's Reality Check, CSCV PBO (`case_studies/utils/uncertainty.py`, `ml4t-diagnostic`).
- **Orca:** `orca/sizing/multiple_testing.py` (Bonferroni + Benjamini-Hochberg), `orca/sizing/block_bootstrap.py`, `orca/sizing/promotion_gate.py` (BH + Bonferroni + walk-forward OOS degradation gate). Overlapping coverage; ML4T's DSR/RAS/PBO layer is the part Orca lacks (see §3).

### 2.5 Fractional Kelly with risk attenuation
- **ML4T:** `17_portfolio_construction/04_kelly_criterion.py` — binary/continuous/multi-asset Kelly, fractional multiples (0.25/0.5/1.0), Ledoit-Wolf shrinkage, leverage/ruin domain checks.
- **Orca:** HP #6 — fractional Kelly k=0.25 with three attenuators (edge discount, fractional multiplier, hard caps), applied in both backtest and live. `orca/sizing/kelly.py`.

### 2.6 Fixed-point / non-float price handling
- **ML4T:** Databento prices are integer nanodollars divided by `1e9` (`03_market_microstructure/15_itch_lee_ready.py:136-137`).
- **Orca:** HP #2 — BIGINT scale factor in PostgreSQL, `Decimal` in Python, `fixed.Fixed` in Go; VIX BIGINT migration 000039. Orca's coverage is broader (enforced everywhere, not just one feed).

### 2.7 Three-state circuit breakers
- **ML4T:** `26_mlops_governance/04_circuit_breakers.py` — `BreakerState {CLOSED, OPEN, HALF_OPEN}`, `CircuitBreaker` base, `BreakerManager` aggregate.
- **Orca:** `internal/breaker/circuit_breaker.go` — `CircuitClosed/Open/HalfOpen` with `Allow()/RecordSuccess()/RecordFailure()`, instantiated for telegram/LLM/VIX/sentiment.

### 2.8 Kill-switch / daily-loss / max-drawdown controls
- **ML4T:** `ml4t.backtest.risk` — `MaxDrawdownLimit(warn_threshold)`, `DailyLossLimit`, `MaxPositionsLimit`, `GrossExposureLimit`, `NetExposureLimit`; `SafeBroker` with persistent latched kill-switch (`25_live_trading/10_safety_risk_demo.py`).
- **Orca:** `internal/risk/pipeline.go` (volatility halt → sizing → prop-firm halt → regime gate → sizing → soft halt → exposure → correlation brake → capital authorization), `PropFirmEnforcer` soft/hard halt, kill-switch re-entrancy guard (`isLocked` + `killSwitchReady`), `MultiAccountCapitalPool.MarkAllViolated()`. **Orca is more robust** (multi-account propagation + re-entrancy guard + CI enforcement).

### 2.9 Transaction-cost realism (maker/taker, spread, slippage, fill probability, adverse selection)
- **ML4T:** `18_transaction_costs/_cost_analysis.py` (Corwin–Schultz, Roll, Kyle λ, square-root impact η, fee schedules), `ml4t.backtest.execution` (Linear/SquareRoot/PowerLaw impact, volume participation).
- **Orca:** `internal/backtest/slippage.go` — `SlippageModel{SpreadBps, MaxSlippage, VolumeImpactFactor, AdverseSelectBps}` with square-root term `VolumeImpactFactor * sqrt(quantity/barVolume)` (line 113), `LimitFillProbability`, `CalibrateSlippageModel`; `internal/model/fill.go` (`MidPriceFill`/`ProbabilisticFill`); `internal/notify/dispatch_summary.go` (`LimitFillProbability`, `EstimateSigmaRolling`, `CalculateCashImpact`); `internal/broker/fee.go` (maker/taker, per-share, SEC, asset-class schedules, holding/expense fees). **Same architecture; the delta is calibration (§3.3).**

### 2.10 Point-in-time / look-ahead prevention
- **ML4T:** backward ASOF joins, ALFRED vintage data, survivorship-free universes, `NEXT_BAR` execution, `join_asof(strategy="backward")`.
- **Orca:** temporal contract validation (look-ahead prevention) in Python, `NEXT_BAR`-equivalent execution-delay semantics, source+timeframe priority-ordered candle loading, NYSE holiday calendar.

### 2.11 Data-quality validation gates
- **ML4T:** `utils/data_quality.py` — `check_ohlc_invariants`, `null_rate`, `gap_summary`, `validate_prices`, `validate_labels`, `validate_features`, `validate_modeling_inputs` (raises on CRITICAL).
- **Orca:** `orca validate-data-integrity` cross-pipeline data integrity check; `scripts/stooq_*` seeding with unique (symbol_id, timeframe, time) constraint; pre-flight `data_integrity` checks.

### 2.12 Declarative strategy configuration (not code)
- **ML4T:** three-layer YAML (`setup.yaml` → `training/{label}.yaml` → `config/{model_type}/{preset}.yaml`) + `strategy_spec` dict, identity-hashed.
- **Orca:** `.gkr.yaml` strategy IR with versioning, hashing, type validation (HP #3). Orca's IR is stronger (typed, validated by `orca validate`).

### 2.13 Ruff + pytest conventions
- **ML4T:** `ruff==0.15.14` (E,F,I,UP,B,SIM), `line-length=100`, pytest + papermill.
- **Orca:** ruff (E,F,W,I,N,B,A,S,PYI,RUF,UP), `line-length=100`, pytest `-v --cov`. Same line length, overlapping rule sets.

### 2.14 Structured/typed error signaling
- **ML4T:** custom `ConfigError`; descriptive `ValueError`/`RuntimeError`; `DataNotFoundError`/`DownloadError`/`MissingDependencyError` with actionable instructions (`data/exceptions.py`); `RiskLimitError` in `ml4t-live`.
- **Orca:** `pkg/errors/errors.go` `AppError{Category, Severity, Retryable, UserAction}` persisted to `error_logs`; `slog` structured logging (39 files). **Orca is stronger** (typed severity/retryability + DB persistence + audit middleware).

### 2.15 Regime inference (HMM / volatility-trend)
- **ML4T:** `09_model_based_features/11_hmm_regimes.py` (filtered, leakage-safe HMM state probabilities).
- **Orca:** volatility/trend-based regime inference (400+ rows); `RegimeActivationMatrix` (14 strategies × 4 regimes).

---

## 3. Full Catalog of Differences

Each entry annotated: **[→ Adopt]** = adoptable strength from ML4T; **[★ Preserve]** = Orca strength to keep (do not regress toward ML4T).

### 3.1 Core System Architecture
- **[★ Preserve]** Orca is a production polyglot platform (Go API/broker/ingest/scheduler + Python domain/math + React SPA + TimescaleDB). ML4T is a notebook repo with no running service. No change to Orca's architecture is warranted.
- **[→ Adopt (partial)]** ML4T's *content-addressed run-log registry* (`registry.db` with 8 tables: training/prediction/backtest/cohort/pair metrics) is a more systematic version of Orca's per-strategy config hashing. Orca already hashes configs (GKR) but could gain cheap provenance by extending `strategy_params_version` (existing) with a `spec_hash` → `parent_prediction_hash` lineage. Low priority; Orca's `strategy_params_version` JSONB already covers the core need.

### 3.2 ML Pipeline Design & Implementation
- **[★ Preserve]** Orca's GKR IR + `orca validate` type-checking is stricter than ML4T's YAML-only IR.
- **[★ Preserve]** Orca's RiskPipeline is CI-enforced (HP #17, `WirePipeline()` between `NewEngine()` and `Run()`); ML4T has no equivalent automated wiring gate.
- **[→ Adopt]** **Deflated Sharpe Ratio / Rademacher Anti-Serum / CSCV PBO** (`ml4t-diagnostic` + `case_studies/utils/uncertainty.py`). Orca's `promotion_gate` uses asymptotic Sharpe t-test + BH/Bonferroni + walk-forward OOS degradation, but does **not** deflate for the number of trials beyond BH or account for non-normal Sharpe variance. Adding DSR (Marchenko-Pastur / effective-rank), RAS, and CSCV PBO to `orca/sizing/` is a pure-math, Python-only, ~1–2 day addition that plugs directly into `promotion_gate.py`. **Highest-value ML adoption.**
- **[→ Adopt]** **Conformal prediction for position sizing** (`case_studies/utils/conformal.py`, walk-forward split-conformal widths → `weight ∝ 1/width`). Orca has conformal-adjacent uncertainty via block bootstrap but no conformal sizing. Medium priority (broader change; touches allocation path).
- **[→ Adopt (low)]** **Feature registry metadata** (`ml4t.engineer` 120 self-documenting features). Orca computes indicators in Go (`cinar/indicator`) without a self-describing registry. A lightweight Python dict of feature `{name, category, formula, params}` would improve `orca attribute`/audit traceability. Low complexity, low-medium impact.

### 3.3 Market Data Ingestion & Preprocessing
- **[★ Preserve]** Orca's TimescaleDB hypertables + compression (7d) + retention (30d) + BIGINT fixed-point is production-grade and superior to ML4T's Parquet-file model for a live system. Do **not** migrate to Parquet/Polars for storage.
- **[★ Preserve]** Orca's source+timeframe priority-ordered loader (`SourceValues()`, `DISTINCT ON` highest-priority source wins) prevents price-scale discontinuities — a class of bug ML4T handles more loosely.
- **[→ Adopt]** **Spread & impact *estimation* from data** (Corwin–Schultz high-low spread, Roll serial-covariance spread, Kyle λ via robust regression, square-root η fit `|r| = σ·η·√(V/ADV)`). Orca's `SlippageModel` already *has* the knobs (`SpreadBps`, `VolumeImpactFactor`, `AdverseSelectBps`) but they are hand-set constants; ML4T *calibrates* them. See §5 item R2.
- **[→ Adopt (low)]** **Polars for Python-side batch feature/resample work.** Orca's Python data pipeline (resampling, regime inference, synthetic gap-fill) uses pandas/numpy; ML4T's Polars lazy + row-group pushdown is measurably faster for the same batch jobs. This is optional and scoped to `orca/data/` only (no storage change). Low priority — only if profiling shows the batch jobs are slow.

### 3.4 Backtesting Framework
- **[★ Preserve]** Orca's `FillSimulator` determinism (fixed seed 42) + `SlippageModel` square-root volume impact + limit fill probability + `ComputeImpliedComparison`/`MaxEquityDivergencePct` live-vs-backtest parity. ML4T's fill path is weaker (impact demonstrated standalone, not in the main backtest path).
- **[★ Preserve]** Orca's `FlagImplausibleCombos` matrix plausibility gate and `Trade.Changes` append-only drill-down. ML4T has no direct analogue.
- **[→ Adopt]** **Sharpe standard-error / variance of Sharpe** (Lo 2002; López de Prado 2025 closed-form SE with skew/kurtosis/autocorrelation) and **Newey-West HAC SE**. Orca's `block_bootstrap.py` gives bootstrap CIs but not a closed-form SE; adding `sharpe_se` is ~0.5 day and strengthens `promotion_gate` + backtest detail metrics.

### 3.5 Risk Management Components
- **[★ Preserve]** Orca's capital-pool/prop-firm enforcement (`CapitalPoolSim`, `CapitalPoolManager`, `BaseCapitalPool`, `MultiAccountCapitalPool`, `PropfirmEnforcer` soft/hard halt) is far beyond ML4T's `MaxDrawdownLimit`/`DailyLossLimit`. Do not simplify toward ML4T.
- **[→ Adopt (partial)]** ML4T's **`warn_threshold` on drawdown/daily-loss limits** (warn before trip) and **breaker event audit trail** (`BreakerEvent`). Orca's prop-firm enforcer has soft (50% reduction) and hard (stop) thresholds already; adding a *warn* band and persisting breaker transitions to the SQLite audit log would harden observability for ~0.5 day. Low priority.

### 3.6 Deployment & CI/CD Workflows
- **[★ Preserve]** Orca's 9-job CI (`python`, `backend`, `frontend`, `gkr-validate`, `anti-pattern-scan`, `security`, `e2e-playwright`, `guardian`, `mutation-test`) with coverage gates (Py ≥80%, Go ≥60%), gitleaks + govulncheck, and `mutmut` mutation testing. ML4T has **no coverage gate** and **no mutation testing**. This is a decisive Orca advantage — preserve.
- **[→ Adopt]** **Structural CI ratchet for look-ahead/temporal leakage.** ML4T's `tests/test_leakage_detectors.py` + `tests/test_holdout_boundary.py` are AST/source scans that fail PRs introducing leak-shaped calls. Orca's `scripts/anti_pattern_scan.py` already implements 18 AST-style rules (including HP #17); adding a **Rule 19 (temporal-contract/look-ahead)** scan is a ~1 day, high-value hardening step. See §5 item R3.
- **[→ Adopt (low)]** **`pytest-rerunfailures` + `pytest-timeout`** for flaky-resilience in the Python suite (ML4T uses both with per-notebook `reruns`/`timeout`). Orca's `dev` extra already pins pytest-cov/hypothesis; adding two more dev deps reduces CI flake noise. ~0.5 day.

### 3.7 Test Coverage & QA
- **[★ Preserve]** Orca's 53 Python + 99 Go + 26 vitest + 13 Playwright test files with hard coverage thresholds and mutation testing is quantitatively stronger than ML4T's ~148 unit-guard + notebook-execution approach. Preserve and do not adopt ML4T's "no coverage" stance.
- **[→ Adopt]** **Parameterized "no if-TEST branches" testing discipline.** ML4T injects reduced params via Papermill rather than `if TEST:` branches, so the *same code path* runs in CI and production. Orca's Go tests use table-driven tests well; the analogous Python practice (test via constructor/param injection, not env-guarded branches) is worth encoding as a review norm. No code change; a documentation/AGENTS.md note.

### 3.8 Documentation Standards
- **[→ Adopt (low)]** ML4T's per-module README pattern (each of 27 chapters + 9 case studies + `utils/`/`data/`/`scripts/`/`tests/`/`envs/` has a README with learning objectives and runtime/memory callouts). Orca's `docs/` is currently 5 flat files. A short `docs/` index mapping each subsystem (risk, backtest, broker, ingest, data) to its entry points would materially improve onboarding. Low effort, no new tooling.
- **[★ Preserve]** Orca's AGENTS.md "Stack Constitution" + 18 Hard Prohibitions + verification gates are a stronger normative contract than ML4T's README. Keep.

### 3.9 Dependency Management
- **[→ Adopt]** **Lockfile + single-source tool pinning.** ML4T pins `ruff==0.15.14` in three places and CI fails if they drift (`tests/test_toolchain_pins.py`); uses `uv.lock` committed + `uv export --frozen` for reproducible Docker images. Orca uses hatchling with **no `[tool.uv]`** and no committed lockfile for Python. Adopting a committed `uv.lock` (or `pip-compile` constraints) + a `test_toolchain_pins.py`-style guard removes silent env drift. ~0.5–1 day. See §5 item R4.
- **[★ Preserve]** Orca's Go module (`go 1.25`, `go.sum`) and npm (`package-lock.json`) are already correctly locked. No change.

### 3.10 Performance Optimization Patterns
- **[★ Preserve]** Orca's Go concurrency (matrix runner, walk-forward, event-driven engine) + TimescaleDB + WebSocket push. ML4T's Python/Polars stack is not a performance model for Orca's live path.
- **[→ Adopt (scoped)]** ML4T's **memoized precompute + skip-if-complete caching** (`precompute_weights()` to avoid re-running MVO/HRP per risk-sweep variant; registry skip-if-complete backtest reuse). Orca already caches backtests; extending the same skip-if-complete pattern to `matrix-runner` weight/allocator precompute would cut redundant work in sweeps. Low priority — verify profiling first.

### 3.11 Error Handling & Resilience
- **[★ Preserve]** Orca's `AppError` typed severity/retryability, DB `error_logs`, audit middleware, rate limiting, JWT revocation, and exponential-backoff broker retry are production-grade; ML4T's `utils/` is deliberately thin (no retry/circuit-breaker/logging in shared code). Preserve.
- **[→ Adopt (low)]** ML4T's **startup reconciliation report** (`SafeBroker.connect()` diffs persisted `RiskState` vs broker state → `reconciliation_report`; refuses non-clean start). Orca already has balance reconciliation in pre-deployment gating; ensuring the live engine *refuses* to start on non-clean reconciliation (not just warn) is a ~0.5 day hardening if not already enforced. Verify against `internal/risk/capital_pool.go` reconciliation before acting.

---

## 4. Prioritized Leverage Recommendations

All items are additive to Orca's stack, Python-math-first (HP #1), and avoid new major dependencies or architectural shifts.

### HIGH priority

**H1 — Add Deflated Sharpe / Rademacher Anti-Serum / CSCV PBO to `orca/sizing/`.**
- *Why:* closes the selection-bias gap that BH/Bonferroni + block bootstrap do not cover. Directly strengthens `promotion_gate` and `orca calibrate` audit. ML4T reference: `ml4t-diagnostic` `deflated_sharpe_ratio`, `rademacher_complexity`, `ras_sharpe_adjustment`, `compute_pbo`.
- *How:* new `orca/sizing/deflated_sharpe.py` (pure numpy/scipy — already deps): `deflated_sharpe(returns, n_trials, correlation_method=MP|ER)`, `ras_adjusted_sharpe(returns, n_trials)`, `cscv_pbo(returns, n_splits)`, `min_trl(sharpe, n_trials)`. Wire into `promotion_gate.py` as an *additional* veto (a combo must pass BH *and* DSR/PBO before promotion). Add unit tests mirroring `tests/sizing/`.
- *Preserve invariant:* math stays Python-only (HP #1); Go `multi_metric_gate.go` continues to consume results via subprocess/JSON, never reimplements.

**H2 — `orca calibrate-costs`: fit `SpreadBps`/`VolumeImpactFactor` from DB candles.**
- *Why:* Orca's `SlippageModel` already models square-root impact + adverse selection, but the coefficients are hand-set. Calibrating them per-symbol from the OHLC data already in TimescaleDB directly hardens HP #9 (backtest accuracy) with no new architecture.
- *How:* new `orca/costs/` module: `corwin_schultz(high, low)`, `roll_spread(close)`, `fit_sqrt_impact(returns, volume, adv)` (η via OLS through origin, matching `VolumeImpactFactor * sqrt(q/V)`), `kyle_lambda(tick_rule_signed_flow, price_change)` (optional, tick data only). New Typer CLI `orca calibrate-costs` emitting per-symbol `SpreadBps`/`VolumeImpactFactor` JSON consumed by `SlippageForSymbol` seed data. Integrate into `orca preflight` checklist (optional `--strict` gate flagging symbols whose calibrated spread deviates >X% from configured).
- *Prereq:* none — OHLC candles already in DB.

**H3 — CI ratchet rule for temporal-contract / look-ahead leakage.**
- *Why:* ML4T proves AST/source-scan ratchets are cheap and catch the highest-severity backtest bug class (look-ahead). Orca's `anti_pattern_scan.py` already has the framework (18 rules).
- *How:* add **Rule 19** to `scripts/anti_pattern_scan.py` detecting look-ahead-shaped patterns (e.g., `.shift(-`, `future`/`next`-window access, `setData` misuse already covered by chart scan, signal computed from bar `t+k`). Wire into existing `anti-pattern-scan` CI job. ~1 day.

### MEDIUM priority

**M1 — Closed-form Sharpe SE + Newey-West HAC into `orca/sizing/`** (Lo 2002; López de Prado 2025). Strengthens `promotion_gate` p-values beyond the asymptotic t-test, feeding both the promotion gate and backtest detail metrics. ~0.5 day.

**M2 — Conformal position sizing** (`orca/sizing/conformal.py`). Walk-forward split-conformal width → `weight ∝ 1/width`, optional allocator in the sizing layer. Broader change (touches allocation path + a new allocator enum); align with existing `inverse_vol`/`top_k` allocator patterns before committing. ~2 days.

**M3 — Feature registry metadata** for `orca attribute`/audit traceability. Lightweight dict of `{name, category, formula, params, source}` for the indicators Orca computes (Go `cinar/indicator` + Python features). Improves `orca calibrate`/`orca attribute` reporting. ~1 day, no runtime dependency.

**M4 — Drift monitoring (PSI/KS) for probability-emitting models.** ML4T `26_mlops_governance/01_drift_monitoring.py` (PSI + KS with watch p-value). Orca's `orca calibrate` is quarterly; a cheap PSI/KS report on holdout-vs-recent predictions would catch regime drift between audits. ~1–2 days, feeds `orca calibrate` output.

### LOW priority

**L1 — `warn_threshold` band on prop-firm limits + breaker event audit trail** persisted to SQLite audit log. ~0.5 day.

**L2 — Polars for `orca/data` batch resampling/regime jobs** (only if profiling shows slowness). Optional; do not change storage.

**L3 — Skip-if-complete precompute in `matrix-runner`** for allocator/weight recomputation across risk-sweep variants.

**L4 — Docs index** mapping subsystem → entry points (pattern borrowed from ML4T per-module READMEs). No tooling.

---

## 5. Low-Effort High-Impact Change Registry

**Definition:** low-effort = ≤ 2 engineering days; high-impact = measurable improvement to latency, throughput, error resilience, maintainability, or backtest accuracy, with no new major dependencies or architectural shifts.

| ID | Change description | Expected measurable impact | Effort | Prerequisites |
|----|--------------------|---------------------------|--------|---------------|
| R1 | **Deflated Sharpe + RAS + CSCV PBO** in `orca/sizing/deflated_sharpe.py`; wire as extra veto in `promotion_gate.py` | Reduces false-positive strategy promotions (selection-bias); measurable via `promotion_gate` survivor count dropping on real matrix sweeps | 1–2 d | numpy/scipy (already deps) |
| R2 | **`orca calibrate-costs`** — Corwin-Schultz/Roll spread + square-root η fit from DB candles; feed `SlippageForSymbol` seeds | Tighter backtest accuracy (HP #9): calibrated spread/impact vs. hand-set constants; measurable as reduced backtest-vs-replay parity drift | 1–2 d | OHLC candles in TimescaleDB (present) |
| R3 | **Anti-pattern Rule 19** (temporal-contract/look-ahead) in `scripts/anti_pattern_scan.py` | Prevents look-ahead regressions at PR time; measurable as new CI failures on leak-shaped code | 1 d | existing anti-pattern scan harness |
| R4 | **Python lockfile + toolchain pin guard** (`uv.lock` or `pip-compile` constraints + `tests/test_toolchain_pins.py`) | Eliminates silent env/dependency drift; reproducible builds; measurable as deterministic CI | 0.5–1 d | none |
| R5 | **Sharpe SE (Lo/LdP-2025) + Newey-West HAC** in `orca/sizing/`; surface in backtest detail metrics | More honest Sharpe CIs; measurable as tighter confidence bands on reported Sharpe | 0.5 d | numpy/scipy |
| R6 | **`pytest-rerunfailures` + `pytest-timeout`** in `dev` extra + pytest `addopts` | Fewer CI flakes; lower false-negative churn; measurable as reduced rerun rate | 0.5 d | none |
| R7 | **`warn_threshold` band + breaker transition audit trail** (SQLite) on prop-firm/risk limits | Earlier operator warning before hard halt; auditable breaker history; measurable as observability coverage | 0.5 d | existing SQLite audit log |
| R8 | **Startup reconciliation refuses non-clean state** (verify + harden existing `capital_pool` reconciliation) | Prevents silent state divergence at boot; measurable as blocked dirty starts | 0.5 d | verify current reconciliation behavior |
| R9 | **Feature registry metadata** (dict of `{name, category, formula, params}`) for indicators | Better `orca attribute`/audit traceability; lower maintainability debt | 1 d | none |
| R10 | **Docs index** (`docs/README.md` subsystem map) | Faster onboarding; measurable as reduced time-to-orient for new contributors | 0.5 d | none |

---

## 6. Phased Implementation Roadmap

Aligned with Orca's existing development roadmap (which has delivered audit remediation, backtest remediation, benchmark-driven enhancements, and broker/AI/strategy-results enhancements). Quick wins first; each phase is independently shippable and CI-verifiable.

### Phase 0 — Quick wins (Week 1, ~2–3 days total)
1. **R6** — add `pytest-rerunfailures` + `pytest-timeout`; reduce CI flake immediately.
2. **R5** — Sharpe SE + Newey-West HAC in `orca/sizing/`; surface in backtest detail metrics.
3. **R10** — docs index in `docs/README.md`.
4. **R7** — warn-threshold band + breaker audit trail.
- *Gate:* `ruff check orca/ tests/ && mypy orca/ && pytest tests/ -v`; `go build ./... && go test ./internal/... -count=1`.

### Phase 1 — Backtest-accuracy hardening (Weeks 2–3, ~3 days total)
5. **R1** — DSR/RAS/PBO in `orca/sizing/`; wire into `promotion_gate.py` as additional veto.
6. **R2** — `orca calibrate-costs` (spread/impact calibration) + `orca preflight` integration.
- *Gate:* `orca validate configs/strategies/*.gkr.yaml`; `python scripts/anti_pattern_scan.py` (zero violations); `pytest tests/guardian/ -v`; add unit tests under `tests/sizing/`.

### Phase 2 — CI ratchet & reproducibility (Week 4, ~1.5 days total)
7. **R3** — Anti-pattern Rule 19 (temporal-contract scan) + CI wiring.
8. **R4** — Python lockfile + toolchain pin guard.
- *Gate:* CI `anti-pattern-scan` + `python` jobs green; `test_toolchain_pins.py` passes.

### Phase 3 — Observability & provenance (Weeks 5–6, ~2 days total)
9. **R8** — harden startup reconciliation refusal.
10. **R9** — feature registry metadata for `orca attribute`.
11. **M4** — PSI/KS drift report folded into `orca calibrate` output.
- *Gate:* `orca calibrate` run clean; `orca attribute` reports new feature metadata.

### Phase 4 — Deferred / optional (only if warranted)
12. **M2** — conformal position sizing (needs allocator-path design review first).
13. **L2/L3** — Polars batch acceleration + matrix skip-if-complete precompute (profile-gated).

**Sequencing rationale:** Phase 0–1 items are pure Python math/CLI additions that immediately improve backtest accuracy and anti-overfit rigor without touching the Go hot path or any storage schema; Phase 2 adds CI enforcement to lock in those gains; Phase 3 hardens observability; Phase 4 items are intentionally deferred because they carry the highest complexity-to-value ratio and are only justified by profiling or explicit demand.

### 6.1 Frontend Enhancements (per backend change)

Every backend change that produces data a user must see gets a matching React surface. These follow the existing `web/src/api/client.ts` client-method + typed-response pattern; no new UI framework, no new dependencies.

| Backend change | Frontend file(s) | Enhancement |
|----------------|------------------|-------------|
| R1/R5 — DSR, PBO, Sharpe SE in `promotion_gate` / backtest detail | `web/src/components/backtest/OverviewTab.tsx`, `web/src/components/deploy/PromoteToLiveWizard.tsx`, `web/src/api/client.ts` | Add a "Statistical robustness" panel: `sharpe_se`, `deflated_sharpe_ratio`, `expected_max_sharpe`, `cscv_pbo`, `min_trl`. Wizard step 3 gates the Deploy button on `deflated_sharpe_ratio >= 0.95` (mirrors CLI `--require-dsr`) |
| R2 — `orca calibrate-costs` per-symbol coefficients | `web/src/pages/IntegrationsPage.tsx` (Providers & Symbols tab) or Admin "Cost Calibration" tab; `web/src/api/client.ts` | Read-only table of `spread_bps / roll_spread_bps / impact_eta / calibrated_at` per symbol+timeframe, sourced from `GET /admin/cost-calibration` |
| R7 — breaker transition audit trail | `web/src/pages/AdminPage.tsx` (Audit tab) | Add "Breaker events" filter to the existing audit log table (event_type = `breaker_transition`), showing trip/warn transitions + payload |
| R9 — feature registry metadata | `web/src/pages/CalibratePage.tsx` / `AttributionPage.tsx` | Display self-documenting feature metadata (`name/category/formula/params`) in `orca attribute` output |

**Backend endpoints required (Go, additive — no existing route/schema changed):**
- `GET /admin/cost-calibration` → `db.ListCostCalibration` (backed by migration 000047).
- Extend `GET /backtests/:id/metrics` (or `/backtests/:id`) to include `sharpe_se`, `deflated_sharpe_ratio`, `cscv_pbo` — computed by shelling out to the new Python CLI (`orca promote-gate`-style subprocess), **never** reimplementing the math in Go (HP #1).
- Breaker events are appended to the existing SQLite `audit_events` (event_type `breaker_transition`) and already surface through the existing `internal/api/audit_handler.go`; the frontend only adds a filter.

### 6.2 Migrations & DB Actions

| ID | Migration / action | Target store | Notes |
|----|--------------------|--------------|-------|
| M-047 | `000047_cost_calibration` (`cost_calibration` table: `symbol_id`, `timeframe`, `spread_bps`, `roll_spread_bps`, `impact_eta`, `adverse_select_bps`, `estimator`, `calibrated_at`, `UNIQUE(symbol_id, timeframe)`) | PostgreSQL/TimescaleDB | **Implemented** (up + down). Consumed by Go `SlippageForSymbol` seeding |
| M-049 | `000049` reserved for `breaker_events` if Postgres persistence is chosen | PostgreSQL | **Deferred** — breaker transitions use the existing SQLite `audit_events` (no new Postgres table); add a Postgres table only if operators need SQL-queryable breaker history |
| DB-1 | `internal/db/repository_cost.go` — `UpsertCostCalibration` / `ListCostCalibration` / `GetCostCalibrationForSymbol` | Go repository | Backs `GET /admin/cost-calibration`; seeded by `orca calibrate-costs` (or a follow-up Go subprocess call) |
| DB-2 | `internal/backtest/slippage.go` — load per-symbol `SlippageModel` from `cost_calibration` when present, else fall back to the existing hand-set presets | Go | Keeps `SlippageForSymbol` as the default; adds an override path, no behavior change until rows exist |
| DB-3 | `internal/audit/audit_log.go` — emit `event_type="breaker_transition"` from `internal/breaker/circuit_breaker.go` on CLOSED→OPEN / OPEN→HALF_OPEN transitions (with `warn_threshold` in payload) | SQLite | R7; append-only, no schema change |

### 6.3 Implementation Status (2026-08-14)

Implemented and verified (`pytest` 45 passed on new/changed modules, `ruff` clean, `mypy` clean, `go build ./...` + `go vet` + `gofmt` clean, `tsc --noEmit` clean):

- **R5** `orca/sizing/sharpe_stats.py` — `sharpe_se`, `sharpe_se_from_stats`, `sharpe_variance`, `newey_west_se` (+ tests).
- **R1** `orca/sizing/deflated_sharpe.py` — PSR/DSR/expected-max/MinTRL/CSCV-PBO (+ tests).
- **R1 wiring** — `promotion_gate.py` reports `sharpe_se`/`deflated_sharpe_ratio`/`expected_max_sharpe`/`n_dsr_significant`; `orca promote-gate --require-dsr/--dsr-threshold`.
- **R2** `orca/costs/` + `orca calibrate-costs` CLI (+ tests).
- **R2/DB-1** `internal/db/cost_calibration.go` (`Upsert`/`List`/`Get`) + `000047_cost_calibration` migration; `GET /api/v1/admin/cost-calibration` handler + route.
- **R2/DB-2** `internal/backtest/slippage.go` `ApplyCalibratedCosts` (data-calibrated override with graceful fallback) (+ tests).
- **R7/DB-3** `internal/breaker/circuit_breaker.go` warn-threshold + transition observer (+ tests), plus a package-level `SetGlobalObserver` wired at startup in `cmd/orca-server/main.go` to slog + the SQLite `audit_events` log (event_type `breaker_transition`).
- **R1/R5 surface** `orca/sizing/robustness.py` (`backtest_robustness_stats`) + `orca backtest-stats` CLI (+ tests); `GET /api/v1/backtests/:id/robustness` (Go shells out to `orca backtest-stats` over stdin — HP #1); frontend `backtests.robustness()` + a "Statistical Robustness" card grid in `OverviewTab` (Sharpe SE, 95% CI, Deflated Sharpe, MinTRL).
- **R6** `pytest-rerunfailures`/`pytest-timeout` in `dev` extra + `--timeout=300` addopts.
- **R3** `scripts/anti_pattern_scan.py` Rule 12 (look-ahead / temporal-contract) wired into the scanner.
- **R4** `tests/test_toolchain_pins.py` (dev-tool drift guard).
- **R9** `orca/features/__init__.py` feature metadata registry + `orca features` CLI (+ tests).
- **R10** `docs/README.md` subsystem map + comparison-doc link.
- **Frontend** `web/src/api/client.ts` `admin.costCalibration()` + Admin "Cost Calibration" tab (read-only table).

Pre-existing issue fixes:
- `[tool.ruff.lint.per-file-ignores]` → `"tests/**" = ["S101"]` (idiomatic `assert` in tests no longer fails the `ruff check tests/` gate).
- `[tool.mypy] python_version = "3.12"` (aligns with CI interpreter; fixes the numpy-stub `type`-statement parse error seen on Python 3.14 environments).
- `orca/sizing/volatility.py` unused `ewma_volatility` import + over-length line removed (re-export preserved via `__all__`).
- `scripts/test_related.py` GO_PACKAGE_MAP gained `internal/breaker/` and `internal/universe/` (guardian test green).

Remaining (documented, non-blocking): a fresh matrix re-run against real stooq data; CSCV-PBO in the UI only once a multi-strategy return matrix is plumbed (currently the endpoint deflates a single backtest's Sharpe by `n_trials`).

---

## Appendix: Key Source References

**ML4T (source of adoptable patterns):**
- `case_studies/utils/uncertainty.py` — DSR/RAS/PBO/block-bootstrap/HAC/Sharpe-SE
- `ml4t-diagnostic` (PyPI) — `deflated_sharpe_ratio`, `rademacher_complexity`, `compute_pbo`, `WalkForwardCV`
- `18_transaction_costs/_cost_analysis.py` — Corwin-Schultz, Roll, Kyle λ, square-root η
- `case_studies/utils/conformal.py` — walk-forward split-conformal sizing
- `utils/cv_splits.py` — purge/embargo walk-forward
- `utils/data_quality.py` — OHLC/null/gap validation gates
- `tests/test_leakage_detectors.py`, `tests/test_holdout_boundary.py` — structural CI ratchets
- `pyproject.toml` + `uv.lock` + `tests/test_toolchain_pins.py` — lockfile + pin discipline

**Orca (target):**
- `internal/backtest/slippage.go` — `SlippageModel` (SpreadBps/VolumeImpactFactor/AdverseSelectBps, square-root impact)
- `internal/backtest/optimized_walk_forward.go:478-479` — `PurgeTradingDays`/`EmbargoTradingDays`
- `orca/sizing/{block_bootstrap,kelly,multiple_testing,promotion_gate,volatility}.py` — existing math/anti-overfit layer
- `orca/math/{brier,ewma,platt,wilson}.py` — canonical math (HP #1)
- `scripts/anti_pattern_scan.py` — 18-rule CI ratchet (extend with Rule 19)
- `internal/broker/fee.go` — maker/taker/asset-class fee schedules
- `.github/workflows/ci.yml` — 9-job CI incl. `mutation-test`
