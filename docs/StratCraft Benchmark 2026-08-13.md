# StratCraft → ORca_algo Cross-System Feature Benchmark

**Prepared:** 2026-08-13
**Scope:** Full StratCraft archive (`docs/stratcraft-master.zip`, extracted read-only) vs. ORca_algo working tree. No files modified in either system.

---

## 1. Executive Summary

StratCraft is a **self-hosted, server-rendered (Node/Express + Handlebars) + Rust-engine** system for daily-candle backtesting, parameter optimization, out-of-sample verification, and Alpaca paper/live execution over a **10,000+ ticker universe**. ORca_algo is a **Go API + Python math/IR layer + React SPA** system for **intraday prop-firm trading** over an 18-symbol universe, with a canonical RiskPipeline, prop-firm gates, kill-switch, and ML meta-labeling.

The two systems have **complementary strengths**, not a simple gap in one direction:

- **ORca_algo is materially stronger** in live-risk management (RiskPipeline, prop-firm FTMO gates, kill-switch re-entrancy guard, volatility halt, exposure tracker, per-account isolation), intraday data (WebSocket → ring buffer → bars), ML *inference* (XGBoost PWin meta-labeling, regime classifier), regime detection (HMM 4-state + 6-state enhancer), statistical rigor (Bonferroni/BH multiple-testing correction, block bootstrap), and broker breadth (Alpaca + IBKR + Paper).

- **StratCraft is materially stronger** in anti-overfit *parameter/template selection*, large-universe data management, corporate-action lifecycle, UI-driven ML model *training*, operational admin tooling, and trade-level analytics.

**Highest-value improvement opportunities for ORca_algo** (ranked):
1. Adopt StratCraft's **layered anti-overfit scoring** — cross-sectional ticker split + verify window + balance penalty + parameter-neighborhood stability score (currently ORca_algo has walk-forward + multiple-testing but no cross-sectional OOS or plateau-preference scoring).
2. Adopt StratCraft's **template-level composite ranking** with verification multiplier.
3. Adopt StratCraft's **corporate-actions lifecycle** (broker sync → split/dividend application to live trades → UI).
4. Adopt StratCraft's **start-timing (entry-date sensitivity) analysis**.
5. Adopt StratCraft's **UI-driven ML model training + model-management admin**.

Details follow.

---

## 2. Structured Catalog of StratCraft Components

### 2.1 Frontend (UI) Layer

| # | Component | Detail |
|---|---|---|
| F1 | Stack | Server-rendered **Handlebars** + **Bootstrap 5** + **Chart.js** + **Plotly 2.27** + **TradingView** embed (no SPA framework) |
| F2 | Dashboard | Ranked strategy leaderboard (sortable, CAGR-sorted), account overview, CAGR-by-period chart, scope/period filter toggles |
| F3 | Templates gallery | Composite **Template Score (0–100)** with popover breakdown; best Calmar/CAGR/Sharpe; top gainers |
| F4 | Templates charts | 12 **Plotly scatter clouds** (performance, verification-vs-backtest, balance train-vs-validation) with legend sync |
| F5 | Template detail | Parameter table, strategies list, **backtest-cache bubble/scatter charts**, full cache table with per-parameter filtering + sorting + sticky headers |
| F6 | Strategy detail | History table, **engine-vs-live comparison** (implied slippage, limit penetration, expense-ratio gap), signal-line + confidence-max charts, recent signals |
| F7 | Backtest detail partial | Equity curve (log), SPY/QQQ benchmark, drawdown, daily-return histogram, cash-%, volume-segment profitability, entry-fill-gap analysis, top-10 days, top-50 profit/loss, ticker dot plot, winner/loser duration |
| F8 | Trade drill-down | Per-trade chart with **simulated future price**, entry/stop/target lines, trade-change history (append-only), corporate actions |
| F9 | Trade-date insights | Rank of moves on a given date (opened/active/closed) |
| F10 | Start-timing page | Entry-date sensitivity, forward returns 1W/1M/3M/6M, correlation vs SPY/QQQ, outcomes-by-market-state, scatter |
| F11 | Ticker directory | 10k+ ticker list, search/sort/pagination, Plotly analytics (candle-count, tradable-%, price-volume scatter), common-name-words chart |
| F12 | Ticker detail | Candle chart + volume, corporate actions, backtest heatmap, signal simulation, TradingView |
| F13 | Strategy operations/skips | Operation history w/ filters; **backtest-vs-planOperations skip-reason comparison** |
| F14 | Admin: Jobs | Enqueue/cancel/logs, job timeline, remote-optimizer job table |
| F15 | Admin: LightGBM | Train model form (hyperparams), manual tree-text save, model table with validation metrics (precision/hit/NDCG@k) |
| F16 | Admin: Users | Invite, mTLS lockdown management, session revocation |
| F17 | Admin: Database | Backup/restore, clear-all-*, bulk ticker add/delete, backtest-cache export/import/prune |
| F18 | Admin: Settings | Grouped settings form (boolean/textarea/number) |
| F19 | Admin: Logs/Deployment | System+PM2 logs, CPU/memory charts, server update/restart/SSH toggle |
| F20 | Auth | Passwordless **email OTP** (6-digit), invite links, device-type tracking |
| F21 | PWA | manifest.json + favicon set |

### 2.2 Backend (Server) Layer

| # | Component | Detail |
|---|---|---|
| B1 | Framework | Express + Handlebars SSR, single `Server` class wiring 16 route modules |
| B2 | Auth/Session | OTP login, `sha256:`-hashed session tokens (cookie `httpOnly`), `request_quotas` rate limit (5/24h email, 10/24h IP) |
| B3 | CSRF | Cookie-based token, double-submit, multipart exemption + re-validation |
| B4 | Encryption | AES-256-GCM `enc:v1:` for broker creds + sensitive settings; `DATABASE_KEY` (32-byte) |
| B5 | Email (Resend) | OTP, invite, **operation-dispatch summary**, **liquidation summary**, cert-bundle, remote-optimizer failure; anti-phishing emoji prefix |
| B6 | JobScheduler | 14 job types, single-worker, retry backoff, `optimize` preemption, idle optimization w/ cooldown |
| B7 | Jobs chain | engine-compile → ticker-sync → corporate-actions → candle-sync → generate-signals → reconcile-trades → backtest-active → plan-operations → dispatch-operations |
| B8 | Scoring (`paramScore.ts`) | Eligibility (≥20 trades) → percentile core → exponential drawdown penalty → **neighborhood stability** → **balance penalty** → final score |
| B9 | Scoring (`templateScore.ts`) | Per-period return/consistency/risk/liquidity → length+recency weighting → **verification multiplier 0.8–1.2×** |
| B10 | Backtest comparison | Implied slippage, limit penetration, expense-ratio gap, entry-price gap |
| B11 | Remote optimization | Hetzner server provisioning via SSH/SFTP, market-data.bin upload, remote `engine optimize`, email completion, self-delete |
| B12 | Alpaca services | Account connector (open/close/update-stop/liquidate), asset service (assets, clock, corporate actions), market-order price cap, OTO stop orders |
| B13 | Cache API | `/api/backtest/check|store|best` with `x-backtest-secret` (timing-safe), shared in-process + remote cache |

### 2.3 Strategy Layer

| # | Component | Detail |
|---|---|---|
| S1 | Template system | JSON templates (9) + `StrategyRegistry`; default strategies auto-created; LightGBM-derived templates |
| S2 | Strategy engine | Rust `Strategy` trait: `generate_signal → Buy/Sell/Hold + confidence`; entry next-open; exit via stop/time/signal |
| S3 | Strategies (9) | RSI, MACD, Williams %R, ADX, ATR breakout, PSAR, Weighted Momentum, Buy-and-Hold, **LightGBM** |
| S4 | Indicators (~17) | SMA, EMA, MACD, ADX/DMI, RSI, Bollinger, VWAP, ATR (SMA + Wilder + 5-SMA), ann. vol, SuperTrend, Keltner, squeeze, OBV, A/D, MFI |
| S5 | Trading rules | Min-dollar-volume gate, position sizing (fixed/confidence/vol-target/conf+vol), stop loss (percent/ATR), ATR trailing, gap-aware stop fill, max-holding-days |
| S6 | Fill model | Market (next open + slippage + cap) and limit (discount + penetration + fill-score), leverage-aware buying power, short borrow fee, ETF expense ratio, liquidity-scaled slippage |
| S7 | Metrics (24) | Total return, CAGR, Sharpe (2% rf), Calmar, MaxDD, win rate, avg/median trade PnL, duration, concurrency, best/worst, ticker count |
| S8 | Optimizer | Auto-detect numeric params → multi-start (Halton) → one-hop local search → drawdown-feasibility gate → SHARPE/CAGR objective |
| S9 | Verify/Balance | Verify on unseen window (all tickers); balance train-vs-validation CAGR → overfit penalty |
| S10 | LightGBM | 51-feature vector (incl. cross-sectional percentiles/z-scores), LambdaRank training (`label_gain`), precision@10/NDCG@10, model bias in logit space |

### 2.4 Data Layer

| # | Component | Detail |
|---|---|---|
| D1 | Providers (3) | EODHD (default), Tiingo, Alpaca — daily bars, automatic adjusted-close scaling, 429 retry, token redaction |
| D2 | Candle sync | SPY-anchored reference date, 11-yr full reload, mismatch detection → full reload, weekend filter, market-clock gate, bounded concurrency, per-candle `disabled` flags (f/p/v) |
| D3 | Universe | 10k+ from Alpaca active equities; SHA-256 → uint32/0xffffffff < ratio **training/validation split** (SPY/QQQ forced validation) |
| D4 | Asset classification | Name-heuristic → 11 types (equity, ETF, inverse, commodity-trust, bond, income, leveraged 2/3/5x) + expense ratios |
| D5 | Corporate actions | Alpaca sync (13 types), account-trade-scoped, trade quantity/stop adjustment, name-change successor, ratio labels |
| D6 | Snapshot | bincode `market-data.bin` (v5) for offline optimize/verify/balance |

### 2.5 Infrastructure Layer

| # | Component | Detail |
|---|---|---|
| I1 | Deployment | Hetzner/Ubuntu `deploy.sh`: nginx + Let's Encrypt + PM2 + Postgres + Rust + LightGBM, 20GB swap, non-root user |
| I2 | Security | mTLS client-cert lockdown, UFW, fail2ban, nginx rate limiting, security headers + CSP, log rotation |
| I3 | Config | `.env` (DATABASE_URL/KEY, DOMAIN), DB-seeded settings (48 keys) |
| I4 | Testing | Jest (repo/scoring/encryption), Rust snapshot pipeline tests, LightGBM template tests |
| I5 | Observability | system_logs + PM2 log viewer, CPU/memory charts |

---

## 3. Side-by-Side Comparison Matrix

Legend: **Full** = equivalent or stronger exists · **Partial** = weaker/less-robust equivalent · **None** = no equivalent.

| StratCraft Component | ORca_algo Equivalent | Status | Gap Annotation |
|---|---|---|---|
| **Frontend** | | | |
| Server-rendered UI (Handlebars/Bootstrap) | React 18 + Tailwind + shadcn/ui SPA | Full | Different stack; ORca_algo is richer (31 UI primitives, 36 pages, WebSocket live) |
| Dashboard (strategy leaderboard + accounts) | MonitorPage (5 tabs, 9 KPIs, live) | Full | ORca_algo stronger (real-time, emergency controls) |
| Template gallery score + popover breakdown | StrategyHub Catalog / matrix comparison | Partial | No cross-template family ranking score; no verification multiplier |
| Template scatter clouds (12) | History compare + correlation matrix | Partial | No verification-vs-backtest / balance clouds |
| Backtest detail (equity/benchmark/drawdown/…) | BacktestHub Detail (17 metrics + 5 charts) | Full | ORca_algo comparable; StratCraft has more niche charts (volume-segment, entry-fill-gap, ticker dot plot) |
| Engine-vs-live comparison (implied slippage/penetration/expense) | `GET /backtests/:id/live-comparison` | Partial | Endpoint exists; StratCraft's implied-slippage/penetration/expense-gap decomposition richer |
| Trade drill-down (simulated future, change history) | TradesTab (basic) | Partial | No per-trade future-path chart or append-only change-history UI |
| Start-timing analysis | — | None | No entry-date sensitivity analysis |
| Ticker directory/analytics (10k) | IntegrationsPage → Symbols + UniversePage | Partial | ORca_algo has 18-symbol universe mgmt, no distribution analytics |
| Job scheduler web UI (enqueue/cancel/logs) | Admin (no job timeline UI) | Partial | Scheduler exists; no enqueue/cancel/timeline UI |
| LightGBM train/manage UI | — | None | No UI-driven model training; training is Python CLI/scripts |
| DB admin (backup/restore/cache ops) | AdminPage (11 tabs) | Partial | No backup/restore or backtest-cache export/import/prune |
| Email OTP login | JWT + bcrypt + TOTP 2FA | Full | ORca_algo stronger (2FA + token revocation) |
| **Backend** | | | |
| Express API | Gin API `/api/v1/*` | Full | ORca_algo broader (WebSocket hub, 20+ sub-handlers) |
| CSRF | — | None | N/A: Bearer-JWT API is not cookie-CSRF-exposed (by design) |
| AES-GCM credential encryption | `internal/security` + `credential.go` vault | Full | ORca_algo has env vault + AES-CFB file vault; key-rotation check |
| Email (Resend + dispatch/liquidation summaries) | SMTP + Telegram + WebSocket push | Partial | Notifier fan-out richer; no order-dispatch/liquidation summary emails |
| JobScheduler (14 jobs, retry, preempt) | Scheduler (cron) + reoptimization + account-sync | Full | ORca_algo comparable; StratCraft's daily cascade + idle optimize is well-structured |
| Param scoring (stability + balance + verify) | Walk-forward + light-optimizer + IVS + multiple-testing | Partial | **Key gap:** no cross-sectional ticker split, no balance penalty, no neighborhood-stability scoring; IVS ≈ stability only |
| Template scoring + verification multiplier | — | None | No template-family ranking concept |
| Remote optimization (Hetzner) | — | None | No ephemeral off-host optimization |
| **Strategy** | | | |
| Template/JSON registry | GKR IR `.gkr.yaml` (17 configs) + registry | Full | ORca_algo stronger (versioned/hashed/typed IR, frozen models) |
| Strategies (9) | 16–17 strategies | Full | ORca_algo broader + regime-aware; StratCraft has Buy-and-Hold + LightGBM templates ORca_algo lacks |
| Indicators (~17) | ~24 (cinar + hand-rolled) | Full | ORca_algo broader |
| Position sizing (fixed/conf/vol/conf+vol) | PositionSizer (conf/regime/VIX/sentiment/corr) + Kelly | Full | ORca_algo stronger (3-attenuator Kelly) |
| Stop loss (percent/ATR/trailing) | stop_loss.go (none/fixed/atr/trailing, gap-aware) | Full | Equivalent |
| Fill model (slippage, short borrow, expense) | slippage.go (TCA, partial fills) + fee.go | Full | ORca_algo stronger on TCA; lacks ETF expense-ratio + short borrow-fee terms |
| Metrics (24) | metrics.go (Sharpe/Sortino/Calmar/MAE/MFE/…) | Full | ORca_algo stronger |
| Optimizer (multi-start + local search) | grid/random/bayesian + walk-forward + light-opt | Full | Different approach; both strong; ORca_algo adds IVS robustness |
| Verify/Balance (OOS) | walk-forward (time OOS) + OOS validation | Partial | **Key gap:** no ticker-level (cross-sectional) OOS |
| LightGBM LambdaRank + 51-feature ML | XGBoost PWin + regime classifier (21-dim) | Partial | ORca_algo ML inference strong; no ranking-label training pipeline or UI |
| **Data** | | | |
| 3 daily providers w/ adjustment | stooq/yahoo/tiingo fetchers | Full | ORca_algo has real stooq intraday (stronger); adjustment via migration 000042 |
| Candle sync (SPY anchor, flags) | stooq pipeline + validate_integrity | Full | ORca_algo comparable/stronger (data-integrity + source-priority loader) |
| 10k universe + ticker split | 18-symbol universe (config) | None | **Scope difference:** StratCraft's cross-sectional validation requires a large universe; ORca_algo's 18 symbols too few for ticker-level OOS |
| Asset classification + expense ratios | AssetClassForSymbol (fee-only) | Partial | No ETF expense-ratio model in backtest fees |
| Corporate actions (sync + apply + UI) | corporate_actions migration 000042 (table + cumulative adj) | Partial | **Key gap:** no broker sync, no live-trade split/dividend application, no UI |
| bincode snapshot | — | None | Not needed (TimescaleDB + source-aware loaders) |
| **Infrastructure** | | | |
| Hetzner deploy.sh (nginx/mTLS/fail2ban) | Docker Compose + Prometheus + Grafana | Full | Different but ORca_algo equivalent-quality; mTLS not applicable to API-first Docker |
| Security hardening | JWT/TOTP/rate-limit/trace/breaker/token-revocation | Full | ORca_algo stronger for API; StratCraft's nginx mTLS is network-layer |
| Observability | Prometheus + Grafana + SystemHealthTab | Full | ORca_algo stronger |

---

## 4. Prioritized Actionable Recommendations

| # | Priority | StratCraft Component to Adopt | Expected Benefit | Complexity |
|---|---|---|---|---|
| R1 | **P0** | Layered anti-overfit parameter scoring (`DATASET.md` §2–5): ticker-level train/validation split (SHA-256), verify window, balance penalty, neighborhood stability score | Materially reduce overfit/false-positive strategy promotion; complements existing walk-forward + BH correction | **High** |
| R2 | **P0** | Template-level composite ranking with verification multiplier (`templateScore.ts`) | Rank strategy *families* across periods/scopes; better catalog UX | Medium |
| R3 | **P0** | Corporate-actions lifecycle (`corporateActionSyncHandler` + `reconcile_trades.rs` split/dividend/merger handling + `corporate-actions-card.hbs`) | Correctness of equity strategies across splits/dividends; currently only a table + basic cumulative adjustment | Medium |
| R4 | **P1** | Start-timing (entry-date sensitivity) analysis (`backtest_start_timing.rs` + `start-timing.hbs`) | Quantify strategy robustness to start date; richer OOS evidence | Medium |
| R5 | **P1** | UI-driven ML model training + management (`lightgbm.hbs`, `train_lightgbm.rs`, validation metrics precision@k/NDCG) | Non-engineer model lifecycle; model governance; ranking-metric training (reuse existing `model_registry`) | High |
| R6 | **P1** | Engine-vs-live implied slippage/penetration/expense-gap comparison (`backtestComparison.ts`) | Validate backtest fill realism against live fills; calibrate slippage model | Medium |
| R7 | **P2** | Job scheduler web UI (enqueue/cancel/logs/timeline) (`jobs.hbs` + `routes/jobs.ts`) | Operational control over backtest/optimize/reopt jobs | Medium |
| R8 | **P2** | Backtest-cache export/import/prune (`routes/database.ts`) | Portable parameter-result cache across environments; reduce re-optimization | Low |
| R9 | **P2** | DB admin UI (backup/restore/bulk ops) (`database.hbs`) | Safer maintenance of TimescaleDB | Low–Medium |
| R10 | **P2** | Trade drill-down richness: simulated-future chart, entry/stop/target lines, change history (`trade.hbs`, `backtest-detail.hbs`) | Better trade forensics for debugging/trust | Medium |
| R11 | **P2** | Order-dispatch summary email with limit-fill probability (`dispatchSummaryCalculations.ts`, `EmailService`) | Operator awareness of pending orders + expected cash impact | Low |
| R12 | **P2** | ETF expense-ratio + asset-classification modeling in backtest fees (`assetClassification.ts`, `config.rs`) | More realistic long-hold cost modeling | Low |

**Primary recommendation bundle (R1+R2):** implement a cross-sectional **ticker split + verify-window + balance-penalty + stability-score** layer on top of the existing walk-forward/multiple-testing. This is the single most differentiated capability StratCraft has that ORca_algo currently lacks. Note a **precondition**: it requires expanding the universe beyond 18 symbols (or applying the split at a *strategy-instance* or *parameter-region* level) — see §5 note 3.

---

## 5. StratCraft Components NOT Suitable for Adoption

| Component | Rationale |
|---|---|
| **mTLS client-certificate lockdown** (`MtlsLockdownService`, nginx `ssl_verify_client`) | Network-layer access control tied to Hetzner/nginx. ORca_algo is a Dockerized API with JWT + TOTP 2FA + DB token revocation + an emergency no-auth kill page. mTLS adds device friction and doesn't fit the Prometheus/Grafana/WebSocket model. |
| **Passwordless email-OTP login** (`routes/auth.ts`) | ORca_algo's username/password (bcrypt) + TOTP 2FA is strictly stronger 2FA and is API-first. Email OTP introduces an external email dependency and is a downgrade for an authenticated API client. |
| **10,000+ daily-candle universe + Buy-and-Hold diversification thesis** (`buy_and_hold.rs`, `tickerSyncHandler`) | Fundamentally different scope: ORca_algo is an 18-symbol prop-firm intraday system with hard concentration/drawdown constraints. The "diversify across 10k equities" alpha thesis conflicts with prop-firm risk rules. (The *ticker-split validation technique* is separately adoptable — R1 — but only after universe expansion.) |
| **Hetzner bare-metal `deploy.sh`** (nginx/fail2ban/UFW/sudoers/manual-update cron) | Not portable to ORca_algo's Docker + Compose + Grafana/Prometheus model. The *remote-optimization concept* (R7-adjacent) is adoptable, but not the Hetzner-specific nginx/UFW hardening script. |
| **LightGBM LambdaRank "extreme multi-bagger" label** (5×/252-day target, `label_gain 0,1,3,7,15,31`) | Incompatible alpha thesis with ORca_algo's prop-firm short-hold intraday model + fractional-Kelly sizing. However, the *ML plumbing* (model registry, UI training, ranking metrics) is adoptable (R5) with a different label/objective. |
| **Daily-only candle backtest as primary engine** | ORca_algo is intraday-first (WebSocket ticks → 1m/5m/15m/1h bars) and already supports daily candles; StratCraft's daily-only engine is a downgrade. |
| **allowShortSelling + short-borrow-fee model** | ORca_algo prop-firm profiles are long-biased; a full short-borrow-fee term is low-value for the current compliance scope (minor borrow-fee modeling could be added to `fee.go` if shorts become material). |

---

### Verification Notes
- Both catalogs were produced from full reads of the archive and working tree (read-only; no files written in either system).
- ORca_algo specifics (RiskPipeline order, FTMO profile defaults, FeatureStore 21-dim vector, 42 migrations, 17 `.gkr.yaml` configs, MonitorPage/AdminPage tab counts) were confirmed against source.
- Minor ORca_algo items flagged for separate follow-up (not StratCraft-related): the stale `orca/cli.py ir_compile` import references non-existent `compile_all_go_configs`/`compile_to_go_config`; the documented `CVD` chart component does not exist as a file (only the `WSCVDData` type).
