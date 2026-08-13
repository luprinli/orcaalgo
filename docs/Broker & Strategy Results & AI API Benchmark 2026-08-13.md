# Broker, Strategy Results & AI API — Cross-System Benchmark & Implementation Plan

**Prepared:** 2026-08-13
**Scope:** Orca_algo vs. the StratCraft reference (read-only archive). Three comparisons — (1) broker connection, (2) strategy-result details, (3) AI API connection — plus a concrete, file-level implementation plan that reuses existing Orca_algo primitives to avoid code duplication.

---

## Executive Summary

| Area | Orca_algo strength | StratCraft strength | Net |
|---|---|---|---|
| Broker | Capability-manifest registry, priority fallback, health checks, 3 drivers (Alpaca/IBKR/Paper) | Per-account credentials/environment, dispatch preflight, stop-loss lifecycle, granular liquidation, asset/clock/corporate-action sync, honest connection snapshot | Adopt StratCraft's *operations depth* onto Orca_algo's *routing skeleton* |
| Strategy results | Richer risk metrics (Sortino, VaR95, CVaR95, Ulcer, ProfitFactor), Monte Carlo, monthly heatmap, regime stats, promote-to-live | Richer trade-distribution metrics (median/avg trade PnL $ + %, best/worst, duration, concurrent trades, unique tickers), SPY/QQQ benchmark overlay, niche analytics | Adopt the missing trade-distribution metrics + benchmark overlay |
| AI API | Has an LLM client (OpenAI/Anthropic/Ollama) + test endpoint + settings UI | **None** (only local LightGBM) | Orca_algo's LLM is a half-wired stub (key ignored, no consumer) — adopt a BYOK model |

**Top 5 actions (priority order):**
1. Per-account broker credentials + environment (unblocks dynamic frontend broker management).
2. Fix the LLM "test" path to honor the user-supplied key + endpoint, then build BYOK key management on the existing vault.
3. Add StratCraft's trade-distribution metrics to `internal/metrics`.
4. Broker dispatch preflight (idempotency + state reconciliation) to complement RiskPipeline.
5. Complete the stubbed broker `CredentialHandler`/`ProviderHandler` endpoints.

---

# Part 1 — Broker Connection Comparison

## 1.1 Backend catalog

| Concern | StratCraft | Orca_algo | Delta |
|---|---|---|---|
| **Abstraction** | `AccountConnector` interface (`supports()`, optional `fetchSnapshot/fetchPositions/dispatchOperation/liquidatePositions`) | `Adapter` (required methods) + `CapableAdapter` (`Manifest()` + `HealthCheck()`) + `Capability` enum | Orca_algo better — capability manifests + fallback |
| **Routing** | `connectors.find(c => c.supports(provider))` | `BrokerDriverRegistry.Resolve(cap)` → priority-ordered healthy adapter, `ResolveWithFallback` (3 retries) | Orca_algo better |
| **Brokers** | Alpaca only | Alpaca + IBKR + Paper | Orca_algo better |
| **Credentials** | Per-account `apiKey`/`apiSecret` + `environment` (paper/live) | Global env vars (`ALPACA_API_KEY/SECRET`); `db.Account` has no credentials | **StratCraft better — key gap** |
| **Base URL** | Per-account `liveUrl`/`paperUrl` | Env `ALPACA_BASE_URL` / `ALPACA_PAPER` | ~Equal |
| **Connection check** | `fetchSnapshot()` → `status: ready/error/unsupported` + counts | `HealthCheck()` + `TestProvider` (hardcoded `latency=55`, fake `100000`) | StratCraft better; Orca_algo stub |
| **Account snapshot** | cash, long/short MV, equity, liquidation value, open position/order counts (truncation flags), currency, source | `{ID, Balance, Equity, BuyingPower, DailyPL, Status}` | StratCraft better |
| **Positions** | `{ticker, side(long/short), qty, marketValue, avgEntryPrice, costBasis, unrealizedPnl}` | `{Symbol, Quantity, AvgEntryPrice, MarketValue, UnrealizedPL}` (no side/cost basis) | StratCraft better |
| **Order placement** | `dispatchOperation` with **OTO stop bracket**, market-price-cap limit, price rounding | `PlaceOrder` (market/limit/stop; no bracket, `alpacaOrderRequest` lacks `order_class`/`stop_loss`) | StratCraft better |
| **Stop lifecycle** | find-open-stop (qty+side+stop), **PATCH replace**, cancel-stop-before-close, OTO leg extraction | None | **StratCraft better — key gap** |
| **Order-state eval** | Rust `evaluate_order` → Filled/Partial/Cancelled + fill price/time | `CapReconcileFills`/`FillProvider.GetFills` declared, not implemented | StratCraft better |
| **Dispatch preflight** | 5 checks → granular `skipped` reasons (blocking trade, pending op, position exists, open order, buying power) | None (RiskPipeline gates risk, not idempotency) | **StratCraft better — complements RiskPipeline** |
| **Liquidation** | dry-run, cancel-all, latest price, deviation bands, per-position result, email summary | `CloseAllPositions` (blanket `DELETE /positions`) | **StratCraft better — key gap** |
| **Assets/clock/corp actions** | `AlpacaAssetService` (assets, clock, 13 corporate-action types, 429 retry) | None in broker layer (R3 table exists, no sync) | **StratCraft better** |
| **Latest price** | `fetchLatestTradePrice` (data API) | Polygon WS ingest + candle endpoints | ~Equal |
| **Pagination** | generic `collectPaginatedResource` + truncation flags | `retry` package only | StratCraft better |
| **Error normalization** | Alpaca restrictions `failed → skipped`; axios-aware message | Generic error strings | StratCraft better |
| **Duplication** | **Two Alpaca clients** (TS + Rust), duplicated pagination/delay | Single Go layer | Orca_algo better |

## 1.2 Frontend catalog

| Concern | StratCraft (Handlebars SSR) | Orca_algo (React SPA) | Delta |
|---|---|---|---|
| Add account | `create-account.hbs`: name, provider, **environment**, apiKey/secret, excluded tickers/keywords | IntegrationsPage (Providers & Symbols + Credentials) + AccountsPage | Orca_algo splits across tabs; no per-account env/creds |
| Account detail | `account.hbs`: snapshot badge, strategies, operations, live trades, **uncovered positions**, allocation warnings, liquidation, reconcile | AccountsPage: account + prop-firm link | StratCraft richer |
| Connection status | snapshot badge `ready/error/unsupported` | "connection status" + test (stubbed) | ~Equal |
| Capability gating | liquidation shown only for Alpaca | none | StratCraft better |
| Risk display | uncovered positions + allocation warnings | MonitorPage Risk tab (RiskPipeline) | Different |

## 1.3 Broker patterns to adopt (not adopt)

**Adopt (priority):**
- **P0** Per-account credentials + environment + excluded tickers/keywords.
- **P0** Rich `AccountSnapshot` + honest `TestProvider`/`GetAccount` (remove stubs).
- **P0** Complete `CredentialHandler.ListCredentials/RotateCredential` (currently hardcoded).
- **P1** Dispatch preflight with granular skip reasons (complements RiskPipeline).
- **P1** Stop-loss lifecycle: OTO bracket, `ReplaceOrder` (PATCH), find-stop matching, order-state eval.
- **P1** Broker data service: assets, clock, latest price, corporate-action sync (feeds R3).
- **P2** Granular liquidation (dry-run + deviation bands), reusing R11's `dispatch_summary.go`.
- **P2** Pagination + truncation flags + `failed→skipped` error mapping.
- **P2** Capability-driven frontend gating (surface `Manifest().Capabilities`).

**Do NOT adopt:**
- Two parallel broker clients (TS + Rust) — keep Orca_algo's single Go layer.
- Raw-string `apiKey`/`apiSecret` in the account type — keep the vault.
- String-match `supports(provider)` dispatch — keep capability manifests.
- Hardcoded handler fallbacks — these are defects to remove, not a pattern.
- `float64` price/qty arithmetic — keep `types.Price`.

---

# Part 2 — Strategy Result Details Comparison

## 2.1 Backend metrics

**StratCraft `PerformanceCalculator`** (`engine/src/performance.rs`) — 24 metrics, trade-distribution rich:

| Group | Fields |
|---|---|
| Core | total_trades, winning_trades, losing_trades, win_rate |
| Return | total_return ($), cagr |
| Risk | sharpe_ratio (2% rf, 252), calmar_ratio (cagr/|DD|), max_drawdown ($ + %) |
| **Trade distribution** | **avg_trade_pnl ($)**, **avg_trade_pnl_percent**, **median_trade_pnl ($)**, **median_trade_pnl_percent**, **best_trade**, **worst_trade** |
| Duration | **median_trade_duration**, **avg_trade_duration** |
| Concurrency | **median_concurrent_trades**, **avg_concurrent_trades** |
| Breadth | **total_tickers** (unique symbols) |
| Win/loss split | avg_winning_pnl ($ + %), avg_losing_pnl ($ + %) |

Notable formula: `pnl_percent = pnl / exposure` where `exposure = |price × qty|` (per-trade return-on-exposure, **not** pnl/initial-capital).

**Orca_algo `metrics.Calculator`** (`internal/metrics/calculator.go` + `types.go`) — risk-metric rich:

| Group | Fields |
|---|---|
| Core | NumTrades, WinRate, ProfitFactor |
| Return | CAGR, TotalReturn, DailyPnL, DailyPnLPct |
| Risk | Sharpe (configurable rf), **Sortino**, Calmar, MaxDrawdown (+ peak/trough idx), **VaR95**, **CVaR95**, **UlcerIndex**, DrawdownPct |
| Cost | CommissionBps, TotalCommission |

**Gap summary:** Orca_algo has Sortino/VaR/CVaR/Ulcer/ProfitFactor (better tail-risk); StratCraft has the whole trade-distribution set (median/avg trade PnL, best/worst, duration, concurrent trades, unique tickers, win/loss split) that Orca_algo lacks entirely.

## 2.2 Frontend detail page

**StratCraft** (`backtest-detail.hbs` + `backtestCharts.ts`): log equity curve with **SPY/QQQ benchmark overlay**, drawdown, daily-return histogram, cash-%, **volume-segment profitability**, **entry-fill-gap analysis**, top-10 days, top-50 profit/loss, **ticker dot plot**, winner/loser duration, trade drill-down (change history), backtest-cache table with per-param filters + bubble/scatter charts.

**Orca_algo** (`BacktestHub.tsx` DetailView + `components/backtest/*`): Performance Metrics (17 metrics), Risk Profile (DD duration, MAE/MFE, hold time, gate status), Equity Curve, Daily Returns, **Monte Carlo**, **Monthly Returns heatmap**, Trade Analysis (Regime / Trades / Optimization), cost metadata, **live-comparison**, **Promote-to-Live wizard**.

**Gap summary:** Orca_algo is stronger on deployment/reporting (Monte Carlo, heatmap, regime, promote-to-live). StratCraft is stronger on *benchmark context* (SPY/QQQ overlay) and niche trade analytics (volume-segment, entry-fill-gap, top-days, ticker dot plot).

## 2.3 Strategy-results recommendations

1. **Add trade-distribution metrics** (P1) — cheapest, highest signal: median/avg trade PnL ($ + %), best/worst trade, avg/median duration, concurrent trades, unique tickers, win/loss split. Add to `metrics.Calculator`; surface in the Overview tab.
2. **Add benchmark (SPY/QQQ) overlay to the equity curve** (P2) — relative-performance context.
3. Optional niche analytics (entry-fill-gap, volume-segment, top-days) — defer; TradesTab drill-down (R10) already covers most forensics.

---

# Part 3 — AI API Connection & BYOK

## 3.1 Current state

**StratCraft:** no external AI/LLM integration. Its only ML is local LightGBM training (`train_lightgbm.rs`), not a hosted model API.

**Orca_algo** (`internal/llm/client.go`, `internal/api/router.go:1847`, `web/src/pages/SettingsPage.tsx` LLMTab):

- `Client` supports OpenAI / Anthropic / Ollama; `NewClient(provider)` reads the key from env `{PROVIDER}_API_KEY` and hardcodes the base URL.
- `POST /llm/test` accepts `{provider, api_key, base_url, model}` **but ignores `api_key` and `base_url`** — it calls `NewClient(provider)` (env key + hardcoded URL). The user-entered key and endpoint are silently discarded.
- `Chat()` is only invoked by `testLLM` (no strategy-analysis or trade-commentary consumer exists).
- The UI (`LLMTab`) has provider/endpoint/model/api-key/temperature fields and a test button; the key is stored in the plaintext settings JSON (no vault), unmasked.

Net: Orca_algo has an LLM *skeleton* but it is a stub — env-key-only, endpoint ignored, no consumer, plaintext storage.

## 3.2 BYOK recommendation (best practice)

Since neither system has a real BYOK model, adopt the following — which deliberately **reuses the broker credential machinery** (vault + per-user scoping) so nothing is duplicated:

1. **Per-user encrypted keys** stored in the vault (same `risk.VaultProvider` the `CredentialHandler` already uses) under a `llm/{userID}/{provider}` path, with a DB table holding non-secret metadata (`provider`, `base_url`, `model`, `masked_suffix`, timestamps).
2. **Key masking on read** (return only `••••••` + last 4) so the UI never round-trips the secret.
3. **`NewClientWithKey(provider, key, baseURL)`** constructor added to `client.go` (keeps `NewClient` as the env fallback for self-hosted/Ollama deployments — no parallel client).
4. **Fix `testLLM`** to honor the passed key + endpoint (and to *not* require a saved key when testing a key the user just typed).
5. **Add an actual consumer** (the intended "strategy analysis / trade commentary") so the key has a purpose — e.g. a scheduler job that summarizes a backtest via the user's key, gated by a circuit breaker (`internal/breaker`).
6. **Per-user scoping** everywhere (`user_id` on the key row) consistent with the existing multi-tenant data isolation.

---

# Part 4 — File-Level Implementation Plan

Legend: **new** = create file · **edit** = modify existing · (reuse) = no new code, wire existing primitive.

## 4.1 Broker — per-account credentials & environment (P0)

| Action | File | Change |
|---|---|---|
| new | `internal/db/migrations/XXXX_account_credentials.up.sql` | add `environment TEXT NOT NULL DEFAULT 'paper'`, `encrypted_credentials BYTEA`, `excluded_tickers TEXT[]`, `excluded_keywords TEXT[]` to `accounts` |
| edit | `internal/db/accounts.go` | add fields to `Account`, `CreateAccount`, `UpdateAccount`; mask secrets on read |
| edit | `internal/broker/alpaca/adapter.go` | add `NewAdapterWithCredentials(key, secret, baseURL string)` (reuse existing `doRequest`) |
| edit | `internal/broker/ibkr/adapter.go`, `paper/adapter.go` | same constructor pattern (reuse) |
| edit | `internal/broker/account_manager.go` | `RegisterAccount` builds adapter from per-account creds (decrypted via `risk.VaultProvider`) instead of env |
| edit | `internal/api/account_handler.go` | accept/return `environment`, masked creds; never echo the secret |
| edit | `web/src/pages/AccountsPage.tsx`, `web/src/api/client.ts` | account form + fields |

*Anti-duplication:* one adapter constructor per broker; `AccountManager` is the single place that decrypts + injects creds.

## 4.2 Broker — honest snapshot + stub removal (P0)

| Action | File | Change |
|---|---|---|
| edit | `internal/broker/adapter.go` | extend `Account` with `LongMV/ShortMV/Currency/OpenOrders/OpenPositions` |
| edit | `internal/broker/alpaca/adapter.go` | populate the extended `Account` |
| edit | `internal/api/provider_handler.go` | `TestProvider`/`GetAccount` return real data (remove `latency=55`, `100000` fallbacks) |
| edit | `internal/api/credential_handler.go` | `ListCredentials`/`RotateCredential` read/write the vault (remove hardcoded `credential-uuid`, `algo_key_v2`) |
| edit | `web/src/pages/IntegrationsPage.tsx` | render snapshot fields + badge |

## 4.3 Broker — dispatch preflight (P1)

| Action | File | Change |
|---|---|---|
| new | `internal/broker/preflight.go` | `Preflight(account, strategyID, symbol, side, qty, price)` → `(skipReason, error)`; 5 checks (blocking trade, pending op, position exists, open order, buying power) |
| edit | `internal/api/router.go` (`placeOrder`) | call `Preflight` before `PlaceOrder`; return `skipped` + reason |

*Anti-duplication:* preflight is broker-agnostic (uses `Adapter.GetPositions` + a repo check); the risk checks stay in RiskPipeline — no overlap.

## 4.4 Broker — stop-loss lifecycle (P1)

| Action | File | Change |
|---|---|---|
| edit | `internal/broker/adapter.go` | add `ReplaceOrder`, `OrderState` eval, bracket fields to `OrderRequest` (`StopLoss *types.Price`, `OrderClass`) |
| edit | `internal/broker/alpaca/adapter.go` | OTO bracket, `PATCH /orders/{id}`, find-open-stop |
| edit | `internal/broker/ibkr/adapter.go`, `paper/adapter.go` | implement or declare unsupported (capability) |
| edit | `internal/broker/broker_driver.go` | add `CapReplaceOrder` capability |

## 4.5 Broker — data service: assets/clock/latest/corporate actions (P1)

| Action | File | Change |
|---|---|---|
| new | `internal/broker/data.go` | `MarketDataProvider` interface (`Assets`, `Clock`, `LatestPrice`, `CorporateActions`) |
| new | `internal/broker/alpaca/data.go` | Alpaca impl (batched + 429 retry via existing `retry` package) |
| edit | `internal/db/corporate_actions.go` | add `UpsertCorporateActionsBatch` (reuse existing upsert) |
| edit | `internal/scheduler/scheduler.go` | add corporate-action sync job calling the above |

## 4.6 Broker — granular liquidation (P2)

| Action | File | Change |
|---|---|---|
| edit | `internal/broker/adapter.go` | add `LiquidatePositions(request) (*LiquidationResult, error)` + result types |
| edit | `internal/broker/alpaca/adapter.go` | dry-run + deviation bands + latest price |
| edit | `internal/api/account_handler.go` | liquidation endpoint + email via `notify.BuildDispatchSummary` (reuse) |

## 4.7 Strategy results — trade-distribution metrics (P1)

| Action | File | Change |
|---|---|---|
| edit | `internal/metrics/calculator.go` | add `MedianTradePnL`, `AvgTradePnL`, `BestTrade`, `WorstTrade`, `MedianDuration`, `MedianConcurrent`, `UniqueTickers`, `AvgWin/Loss` (reuse existing `median`/`average`/`stddev` helpers) |
| edit | `internal/metrics/types.go` | add `TradeDistribution` struct (kept separate from `PerformanceSnapshot` for backward compat) |
| edit | `internal/backtest/engine.go` | populate from `result.Trades` |
| edit | `internal/api/backtest_metrics_handler.go` | expose in the detail response |
| edit | `web/src/types/api.ts`, `web/src/pages/BacktestHub.tsx`, `web/src/components/backtest/OverviewTab.tsx` | render |

## 4.8 Strategy results — benchmark overlay (P2)

| Action | File | Change |
|---|---|---|
| edit | `internal/backtest/engine.go` | load SPY/QQQ benchmark candles via existing `LoadCandlesByTimeframeFiltered` |
| edit | `internal/api/backtest_metrics_handler.go` | return normalized benchmark series |
| edit | `web/src/charts/EquityCurveChart.tsx` | overlay benchmark line |

## 4.9 AI — BYOK key management (P1)

| Action | File | Change |
|---|---|---|
| new | `internal/db/migrations/XXXX_llm_api_keys.up.sql` | `llm_api_keys(id, user_id, provider, vault_path, base_url, model, masked_suffix, timestamps)` |
| new | `internal/db/llm_keys.go` | `Upsert/List/Get/Delete` LLM key (masked read) |
| edit | `internal/llm/client.go` | add `NewClientWithKey(provider, key, baseURL)`; keep `NewClient` (env fallback) |
| new | `internal/api/llm_handler.go` | `GET/POST/DELETE /llm/keys`, `POST /llm/test` (honor passed key/endpoint) — replace the current `s.testLLM` |
| edit | `internal/api/router.go` | route the new handler; remove old `s.testLLM` |
| edit | `web/src/pages/SettingsPage.tsx` (LLMTab) | save/mask/rotate key UI |
| edit | `web/src/api/client.ts` | LLM key CRUD + test methods |
| new (consumer) | `internal/scheduler/llm_jobs.go` | strategy-analysis/trade-commentary job using per-user key (gated by `internal/breaker`) |

*Anti-duplication:* BYOK reuses `risk.VaultProvider` + the `CredentialHandler` pattern; `NewClientWithKey` extends the existing client instead of adding a second one; the consumer reuses `internal/breaker` for circuit protection.

---

## Bottom line

Orca_algo already has the better *pluggability skeleton* (capability registry, fallback, health checks, 3 brokers) and the better *tail-risk metrics*. The high-leverage work is: **(1)** per-account broker credentials (P0), **(2)** completing the LLM test path + a BYOK key store on the existing vault (P1), **(3)** adding StratCraft's trade-distribution metrics (P1), and **(4)** the broker dispatch preflight + stop-loss lifecycle (P1). Every change reuses an existing primitive (vault, `retry`, `breaker`, `notify`, `metrics.Calculator`, `LoadCandlesByTimeframeFiltered`) to avoid duplication.

---

# Part 5 — System-Wide Alignment Review & Enhanced Plan

This section reviews Part 4 against the **Stack Constitution** (`AGENTS.md`): the 18 hard prohibitions (HP #1–#18), the Python/Go/React language boundaries, the pre-commit verification gates, and the pre-deployment gating. It then adds the cross-cutting areas the original plan missed.

## 5.1 Isolated vs. system-wide classification

| Workstream | Scope | Why |
|---|---|---|
| 4.7 Trade-distribution metrics | **System-wide (medium)** | Metrics feed `orca/sizing/promotion_gate.py`, `orca calibrate`, and the promotion UI — not just the detail page. Adding a metric but not plumbing it into promotion/calibration would create a second, disconnected evaluation path. |
| 4.8 Benchmark overlay | Isolated | Single handler + chart; no shared state. |
| 4.3 Dispatch preflight | **System-wide (high)** | Must integrate with **RiskPipeline** (HP #17) without becoming a parallel risk path; the anti-pattern scan (Rule 11) enforces `WirePipeline()` between `NewEngine()` and `Run()`. Any new live-order entry point must route through the canonical pipeline. |
| 4.1 Per-account credentials | **System-wide (high)** | Touches DB + repo + adapter + account manager + API + frontend + vault + RBAC + audit + preflight. Interacts with HP #18 (per-account strategy isolation) and the multi-account capital pool. |
| 4.2 Rich snapshot + stub removal | **System-wide (medium)** | The snapshot must feed `CapitalPoolManager` balance reconciliation (pre-deployment "balance reconciliation" gate), the WebSocket hub, and the frontend — not just the provider page. |
| 4.4 Stop-loss lifecycle | **System-wide (high)** | Changes the `Adapter` interface (ripples to all 3 drivers), **requires a matching backtest model** for backtest/live parity (HP #9), must record changes via `trade_change.go`, and must persist stop-leg order IDs. |
| 4.5 Broker data service | **System-wide (medium)** | Symbol mapping must integrate with `internal/universe` + `configs/universe.json` + symbol CRUD (single source of truth); corporate actions feed R3 + `orca validate-data-integrity`; the clock feeds session gating (adversarial after-hours checks). |
| 4.6 Liquidation | **System-wide (high)** | Must respect the kill-switch re-entrancy guard (HP #8) and propagate via `MultiAccountCapitalPool.MarkAllViolated()`; shares the emergency path rather than adding a parallel one. |
| 4.9 BYOK | **System-wide (high)** | DB + repo + vault + client + API + rate-limit + breaker + audit + masking + frontend + i18n + Prometheus + config validation + an actual consumer + preflight. |

## 5.2 Constitution alignment

| Workstream | Relevant rule/gate | Compliance action |
|---|---|---|
| 4.1/4.2/4.6 | **HP #2** (no IEEE float for order prices) | Extended `Account` (`LongMV/ShortMV`), `LiquidationRequest.limitPrice`, and any price in the snapshot **must use `types.Price`**, not `float64`. Only percentages/qty may be `float64`. |
| 4.3/4.4 | **HP #17** (RiskPipeline) + anti-pattern Rule 11 | Preflight is *additive* and placed **before** `RiskPipeline.ProcessSignal`/`ReconcileFill`; it must not duplicate sizing/halt logic. New order path must pass the CI scan (no `NewEngine()→Run()` without `WirePipeline()`). |
| 4.1 | **HP #18** (per-account isolation) | Per-account credentials must flow through `RegisterAccountStrategies(accountID)` factory isolation; no shared adapter state across accounts. |
| 4.6 | **HP #8** (kill-switch re-entrancy guard) | Liquidation shares `isLocked` + `killSwitchReady`; a manual liquidation is an emergency action, not a separate unguarded path. |
| 4.7 | **HP #1** (canonical math in Python) | The new trade-distribution metrics (median/avg PnL, duration, concurrency) are **not** in the canonical set (Kelly/Brier/Platt/Wilson/EWMA) and may live in Go. Do **not** add any of the five canonical formulas here. |
| 4.9 | **HP #10** (no panic for recoverable errors) + secrets policy | LLM/broker call failures return errors; secrets are never logged; keys are masked on every read. |
| All | Pre-commit gates | Go: `go build ./... && go test ./internal/... && golangci-lint`. Python: `ruff && mypy && pytest`. Frontend: `tsc && vitest && playwright`. Anti-pattern scan: `scripts/anti_pattern_scan.py` (zero new violations). |
| 4.1/4.9 | Pre-deployment gating | New `orca preflight --strict` checks: broker credential presence + vault health, LLM key integrity, and (already present) balance reconciliation must now reconcile against the richer snapshot. |

## 5.3 Cross-cutting areas missing from the original plan (add these)

1. **Migration hygiene.** `XXXX_` → concrete `000043_account_credentials` and `000044_llm_api_keys`, each with a `.down.sql`, idempotent, matching the BIGINT/`types.Price` and `user_id` FK `ON DELETE CASCADE` conventions of 000001–000042.

2. **Per-user RBAC (IDOR protection).** Every new secret-bearing endpoint (`/accounts/*`, `/llm/keys/*`) must scope by `user_id` the way `ListAccountsByUser` does; admin-only routes (`/admin/*`) remain behind `AuthMiddleware` + role check. The vault path must be namespaced per user (`accounts/{userID}/{accountID}`, `llm/{userID}/{provider}`) — not a flat global path.

3. **Secrets lifecycle.** Masked reads (last-4 only), no-logging, rotation for *both* account credentials and LLM keys (extend the existing `CredentialHandler` rotation pattern; do not write a second mechanism). Store **`vault_path` + masked metadata** in the DB row — the plan's `encrypted_credentials BYTEA` should instead reference the vault (consistent with `CredentialHandler`).

4. **Rate limiting + circuit breaker on external calls.** Apply `internal/api/middleware/rate_limit.go` to `/llm/*` and broker-test routes; wrap the LLM client (not just the consumer job) in `internal/breaker` with per-user quotas; extend the breaker to a new `llm` circuit (the existing set is telegram/VIX/sentiment).

5. **Audit logging.** Credential/store/rotate/delete and liquidation actions must emit audit events via the existing audit handler (`internal/api` audit + `audit_error_logs`), mirroring how token revocation and kill-switch events are recorded.

6. **WebSocket propagation.** Rich snapshot + connection status should be pushed over the existing `monitor.WSHub` (risk status already pushes every 5s) so IntegrationsPage/MonitorPage update live rather than poll-only.

7. **Risk & capital-pool integration.** The richer `Account` (equity, liquidation value, long/short MV) must feed `CapitalPoolManager` reconciliation and `BaseCapitalPool` (balance/DD/DailyPnL), so the pre-deployment "balance reconciliation" gate uses real broker numbers.

8. **Backtest/live parity for stop brackets.** The stop-loss lifecycle (4.4) must add the *same* OTO/bracket semantics to the backtest engine (`internal/backtest/engine.go` `activeStops` + `StopLossConfig`), with a `parity_test.go` case asserting backtest == replay. Without this, HP #9 ("do not assume perfect fills") and the "backtest-vs-replay parity" gate are violated.

9. **Symbol mapping single source of truth.** Broker assets must populate `internal/universe` + `configs/universe.json` + the symbol CRUD (IntegrationsPage → Providers & Symbols), not create a parallel symbol table.

10. **Python-side updates.** `orca/preflight/` gains checks (broker creds present, vault health, LLM key integrity, masked-key integrity); `orca/sizing/promotion_gate.py` + `orca calibrate` consume the new trade-distribution metrics so promotion/calibration and the detail page stay in lockstep.

11. **Config validation at startup.** New config keys (LLM base URL/model/temperature, broker environment defaults) are validated at startup (the existing `ORCA_ENVIRONMENT` + startup-config-validation pattern).

12. **i18n.** New UI strings (BYOK key form, connection badge states, new metric labels) need keys in the locale files used by `t('llm:description')`.

13. **Observability.** LLM cost/latency/token usage and broker call latency → Prometheus (`monitor/metrics`), consistent with existing telemetry.

14. **Tests + anti-pattern scan.** Add `internal/broker/preflight_test.go`, extend `internal/llm/client_test.go` and `internal/metrics/calculator_test.go`, add `internal/api/llm_handler_test.go`, plus a frontend e2e case; run the anti-pattern scan to confirm zero new HP #2/#17 violations.

15. **Documentation.** Update `AGENTS.md` "Current Implementation State" (the established pattern after each enhancement batch), root `README.md`, and the affected package READMEs (`internal/broker`, `internal/llm`, `internal/metrics`, `orca`).

## 5.4 Revised file-change list (system-wide annotations)

The original Part 4 tables remain valid; the following **additional** files must change to keep each workstream system-consistent (marked ✚ = added here):

| Workstream | Additional files (✚) |
|---|---|
| 4.1 credentials | ✚ `internal/db/migrations/000043_account_credentials.{up,down}.sql` · ✚ `internal/api/middleware/` (user-scope guard) · ✚ audit handler · ✚ `orca/preflight/` (credential check) · ✚ `internal/security` (masking helper reuse) |
| 4.2 snapshot | ✚ `internal/risk/capital_pool.go` + `internal/propfirm/pool_base.go` (reconciliation) · ✚ `monitor.WSHub` push |
| 4.3 preflight | ✚ `internal/risk/pipeline.go` wiring point · ✚ `scripts/anti_pattern_scan.py` Rule 11 test case |
| 4.4 stop lifecycle | ✚ `internal/backtest/engine.go` (bracket parity) · ✚ `internal/backtest/parity_test.go` · ✚ `internal/backtest/trade_change.go` (stop-change recording) · ✚ `internal/db` (stop-leg order IDs) |
| 4.5 data service | ✚ `internal/universe` + `configs/universe.json` + `internal/api/symbol_handler.go` · ✚ `orca/data/validate_integrity.py` |
| 4.6 liquidation | ✚ `internal/risk` kill-switch guard + `MultiAccountCapitalPool.MarkAllViolated()` · ✚ audit handler |
| 4.7 metrics | ✚ `orca/sizing/promotion_gate.py` + `orca/calibration/` (consume) |
| 4.9 BYOK | ✚ `internal/db/migrations/000044_llm_api_keys.{up,down}.sql` · ✚ `internal/api/middleware/rate_limit.go` · ✚ `internal/breaker` (llm circuit) · ✚ audit handler · ✚ Prometheus metrics · ✚ i18n locale files · ✚ `orca/preflight/` (LLM key integrity) |

## 5.5 Revised sequencing (dependency-ordered)

1. **Foundation (no new risk):** migration hygiene + `types.Price` corrections + remove handler stubs (4.2) + trade-distribution metrics (4.7, Go-only) + benchmark overlay (4.8).
2. **Secrets/identity (cross-cutting first):** per-account credentials (4.1) *with* RBAC + vault + masking + audit, then BYOK (4.9) reusing that same machinery.
3. **Risk-integrated broker ops:** preflight (4.3) wired into RiskPipeline, then stop-lifecycle (4.4) with backtest parity, then liquidation (4.6) behind the kill-switch guard.
4. **Data & observability:** broker data service (4.5) + symbol mapping + corporate-action sync + WebSocket/Prometheus.
5. **Gates:** Python preflight/calibration/promotion updates, config validation, tests, anti-pattern scan, docs — run the full pre-commit + pre-deployment verification set.

This ordering ensures each system-wide change lands *with* its supporting cross-cutting concern (identity, risk, observability) rather than being applied as an isolated feature.

---

# Part 6 — Test Plan, DB Alignment & Frontend Consumption

## 6.1 Test strategy (unit / integration / e2e)

Each workstream ships with three layers, matching the existing gate (`go test`, `pytest`, `vitest`, `playwright`).

| Workstream | Unit (Go) | Integration (Go, `httptest`) | E2E (Playwright) |
|---|---|---|---|
| 4.7 trade-distribution metrics | `internal/metrics/calculator_test.go` — `ComputeTradeDistribution` (empty, win/loss split, median, best/worst, duration, unique tickers) | `internal/api/backtest_metrics_handler_test.go` — `GET /backtests/:id/trade-distribution` returns correct JSON for a seeded result | `web/e2e/backtest-detail.spec.ts` — navigate to a backtest, assert the Trade Distribution table renders medians/avg/best/worst |
| 4.8 benchmark overlay | — (pure data) | `backtest_metrics_handler_test.go` — benchmark series present + normalized | `backtest-detail.spec.ts` — equity chart shows a second benchmark series |
| 4.3 preflight | `internal/broker/preflight_test.go` — each skip reason (blocking trade, pending op, position exists, open order, buying power) | `order_handler_test.go` — order with existing position returns `skipped` + reason | `web/e2e/orders.spec.ts` — place order against a held symbol shows the skip banner |
| 4.1 per-account credentials | `internal/broker/alpaca/adapter_test.go` — `NewAdapterWithCredentials` builds client; masked read | `account_handler_test.go` — create/update account never echoes the secret; list masks | `web/e2e/accounts.spec.ts` — add account, assert masked `••••` + last-4 |
| 4.4 stop lifecycle | `alpaca/adapter_test.go` — bracket request body; PATCH replace | `order_handler_test.go` — stop replace returns the new stop leg ID | `web/e2e/orders.spec.ts` — move stop updates the row |
| 4.6 liquidation | `alpaca/adapter_test.go` — deviation-band skip + limit price | `account_handler_test.go` — dry-run returns per-position preview | `web/e2e/accounts.spec.ts` — liquidate dry-run shows preview + email sent |
| 4.5 data service | `alpaca/data_test.go` — asset dedupe, clock parse, corporate-action map | `corporate_actions_test.go` — batch upsert idempotent | `web/e2e/admin.spec.ts` — corporate-actions tab lists synced events |
| 4.9 BYOK | `internal/llm/client_test.go` — `NewClientWithKey` uses passed key+baseURL; `internal/api/llm_handler_test.go` — test honors key | `llm_handler_test.go` — save/mask/delete key; rate-limit 429 | `web/e2e/settings.spec.ts` — add key, see masked value, test connection |

**Regression gate per change:** `go build ./... && go vet ./... && go test ./internal/... -count=1`, `ruff check orca/ && mypy orca/ && pytest tests/ -q`, `cd web && npx tsc --noEmit && npx vitest run && npx playwright test`, and `python scripts/anti_pattern_scan.py` (zero new HP #2/#17).

## 6.2 Database alignment (concrete migrations)

| Migration | Purpose | Schema (aligned with 000001–000042 conventions) |
|---|---|---|
| `000043_account_credentials.up.sql` / `.down.sql` | per-account broker credentials + environment | `ALTER TABLE accounts ADD COLUMN environment TEXT NOT NULL DEFAULT 'paper'`, `ADD COLUMN vault_path TEXT NOT NULL DEFAULT ''`, `ADD COLUMN masked_key TEXT NOT NULL DEFAULT ''`, `ADD COLUMN excluded_tickers TEXT[] NOT NULL DEFAULT '{}'`, `ADD COLUMN excluded_keywords TEXT[] NOT NULL DEFAULT '{}'`; `.down` drops the columns |
| `000044_llm_api_keys.up.sql` / `.down.sql` | per-user BYOK LLM keys | `CREATE TABLE llm_api_keys (id UUID PK, user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, provider TEXT NOT NULL, vault_path TEXT NOT NULL, base_url TEXT NOT NULL DEFAULT '', model TEXT NOT NULL DEFAULT '', masked_suffix TEXT NOT NULL DEFAULT '', created_at/updated_at TIMESTAMPTZ)` + index on `(user_id, provider)`; `.down` drops the table |

Rules honored: idempotent, `user_id` FK with `ON DELETE CASCADE`, secrets live in the vault (only `vault_path` + `masked_suffix` in the DB), no `float64` price columns (percentages/quantities only).

## 6.3 Frontend consumption (display of new DB data)

| Data | API | Frontend type (`web/src/types/api.ts`) | Display |
|---|---|---|---|
| Trade distribution | `GET /backtests/:id/trade-distribution` | `TradeDistribution` | `BacktestHub` DetailView — new "Trade Distribution" metrics grid (avg/median PnL $ + %, best/worst, avg/median duration, win/loss split, unique tickers) |
| Benchmark | `GET /backtests/:id/metrics` (extended) | `benchmark: {spy: number[], qqq: number[]}` | `EquityCurveChart` second series |
| Account env/creds | `GET/POST /api/v1/accounts` | `environment`, `masked_key` | `AccountsPage` — environment badge + masked credential display |
| Rich snapshot | `GET /providers/:id/account` (real) | `AccountSnapshot` (extended) | `IntegrationsPage` — connection badge + cash/equity/position counts |
| LLM keys | `GET/POST/DELETE /api/v1/llm/keys` | `LLMKey` | `SettingsPage` LLMTab — key list with mask, add/rotate/delete, test |
| Corporate actions | `GET /admin/corporate-actions` (already exists) | (existing) | Admin Corporate Actions tab (already wired) |

Frontend invariants: every new field is additive to the existing type; masked values are read-only (`••••` + last-4); all new strings go through `t('key:fallback')` i18n.

---

# Part 7 — Implementation Log (2026-08-13)

Begin with **4.7 trade-distribution metrics** (dependency-ordered foundation, Go-only, no risk), then 4.8, then the secrets/identity workstream.

### ✅ 4.7 — Trade-distribution metrics (DONE)
- `internal/metrics/types.go` — `TradeDistribution` struct (additive, no price fields → HP #2 clean).
- `internal/metrics/calculator.go` — `ComputeTradeDistribution` + `median`/`maxValue`/`minValue` helpers (reuses `average`).
- `internal/metrics/calculator_test.go` — 5 unit tests (empty, win/loss split, odd median, no-duration).
- `internal/api/backtest_metrics_handler.go` — shared `loadBacktestTrades` helper (de-duplicates the trade JSON→`TradeSummary` mapping) + `GET /backtests/:id/trade-distribution`; `getBacktestMetrics` refactored to use the helper.
- `internal/api/router.go` — route registered.
- `web/src/types/api.ts` + `web/src/api/client.ts` — `TradeDistribution` type + `backtests.tradeDistribution`.
- `web/src/components/backtest/OverviewTab.tsx` + `web/src/pages/BacktestHub.tsx` — Trade Distribution grid (avg/median P&L, best/worst, avg win/loss, hold times, unique tickers) rendered above Regime Breakdown.

**Validation:** `go build ./...` + `go vet` clean; `go test ./internal/metrics ./internal/api` pass; `tsc --noEmit` clean; `vitest` 233/233 pass; anti-pattern scan — no new violations.

### ✅ 4.9 — BYOK LLM key management (DONE, core)
- `internal/db/migrations/000044_llm_api_keys.{up,down}.sql` — per-user `llm_api_keys` (vault-backed metadata only, `user_id UUID` FK `ON DELETE CASCADE`, `UNIQUE(user_id, provider)`).
- `internal/db/llm_keys.go` — `LLMKey` repo: `Upsert/List/Get/Delete` (secrets never persisted; `vault_path` is `json:"-"`).
- `internal/llm/client.go` — `NewClientWithKey(provider, key, baseURL)` + extracted `defaultBaseURL` (deduplicated the base-URL switch; `NewClient` now delegates to it).
- `internal/api/llm_handler.go` — `GET/POST/DELETE /llm/keys`, `POST /llm/test`. **Fixes the `testLLM` bug**: the passed key + base URL are honored; falls back to the user's stored (vault) key when omitted. Masking via a single `maskSuffix` helper.
- `internal/api/router.go` — `llmHandler` field + wiring; routes registered under `protected` (auth); old `s.testLLM` removed.
- `web/src/types/api.ts` + `web/src/api/client.ts` — `LLMKey` type + `llm` client (`listKeys`/`addKey`/`deleteKey`).
- `web/src/pages/SettingsPage.tsx` — LLMTab "Stored API Keys (BYOK)" card (masked list, save/delete).

**Validation:** `go build`/`go vet` clean; `go test ./internal/llm ./internal/api` pass (incl. `NewClientWithKey` + `maskSuffix`); `tsc --noEmit` clean; `vitest` 233/233 pass; anti-pattern scan clean for new files.

**Remaining cross-cutting hardening (noted in §5.3):** rate-limit middleware on `/llm/*`, an `llm` circuit in `internal/breaker`, audit events for key add/delete, and Prometheus cost/latency metrics — all additive to this core.
