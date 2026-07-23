# `internal/` — Go Backend

The Go layer handles all I/O-bound and orchestration work: HTTP API serving, broker integration, market data ingestion, backtest execution, risk management, scheduling, and database access. Shares `PositionSizer` with live and backtest paths for sizing consistency.

[↑ Back to Root README](../README.md)

## Sub-Packages

### `internal/api/` — REST API & WebSocket

HTTP API server using the Gin framework. 50+ endpoints organized by domain.

| File | Purpose |
|------|---------|
| `router.go` | Main Gin router — mounts all handlers, middleware, WebSocket upgrade. Backtest configs include PropFirmEnabled, GateProfile, StopLoss, TakeProfit, FixedSeed |
| `auth_handler.go` | JWT authentication (login, register, 2FA, password reset) |
| `admin_handler.go` | Admin operations (seed, health, audit logs, user management) |
| `market_data_handler.go` | Market data endpoints (candles, ticks) |
| `symbol_handler.go` | Symbol CRUD with feed assignment |
| `provider_handler.go` | Broker/data provider CRUD |
| `webhook_handler.go` | TradingView/ChartInk webhook receivers |
| `credential_handler.go` | API credential management |
| `propfirm_handler.go` | Vendor-agnostic prop firm profile management |
| `backtest_history_handler.go` | Backtest result storage and retrieval |
| `universe_handler.go` | Universe snapshot and configuration |
| `settings_handler.go` | User settings (notifications, risk) |
| `data_source_handler.go` | Market data source management |
| `middleware/middleware.go` | JWT auth middleware, CORS, rate limiting |

### `internal/broker/` — Broker Adapters

Plugin registry pattern with `Adapter` interface.

| File | Purpose |
|------|---------|
| `adapter.go` | `Adapter` interface: PlaceOrder, CancelOrder, CloseAllPositions, GetPositions, GetAccount |
| `registry.go` | Plugin registry with CancelAllOrders/CloseAllPositions across all adapters |
| `alpaca/adapter.go` | Alpaca Markets broker (REST + WebSocket) |
| `ibkr/adapter.go` | Interactive Brokers adapter (TWS/Gateway) |
| `paper/adapter.go` | Paper trading adapter with simulated fills (0.1% commission) |

### `internal/ingest/` — Market Data Ingestion

| File | Purpose |
|------|---------|
| `ws_client.go` | Polygon.io WebSocket client with auto-reconnection |
| `ring_buffer.go` | Lock-free ring buffer (16,384 ticks), shared with market ingest pipeline |
| `vix_client.go` | VIX data fetcher (Polygon REST) → feeds HMM modulation + PositionSizer |
| `sentiment.go` | Fear & Greed Index fetcher (Alternative.me API) → feeds sentiment gating + PositionSizer |

### `internal/backtest/` — Backtest Engine

| File | Purpose |
|------|---------|
| `engine.go` | Core backtest loop: load candles, regime/VIX/sentiment logs, run strategy with shared PositionSizer, compute metrics + adverse selection rate, apply Multi-Metric Gate |
| `strategy_runner.go` | Z-score mean reversion with exit signal propagation |
| `propfirm_enforcer.go` | Vendor-agnostic PropFirmEnforcer: daily loss, drawdown, consistency, regime multipliers. Returns ComplianceReport (was FTMOReport). Uses RuleBreach (was FTMOBreach) |
| `seasonality.go` | Seasonality overlay: Jan 2.0×, Sep 0.5×, turn-of-month 1.5×, Nov/Dec 1.25×, US holiday calendar |
| `slippage.go` | Fill simulator with FixedSeed support and TCA (SimulateFillWithTCA: slippage vs mid, vs last) |
| `walk_forward.go` | Rolling window train/test with PassedCompliance tracking |
| `optimized_walk_forward.go` | Parameter optimization with IVS robustness, optimized params applied to OOS test |
| `monte_carlo.go` | Go→Python subprocess for Monte Carlo pass-probability |
| `batch_runner.go` | Matrix backtest orchestrator with Warnings and GatePassed propagation |
| `multi_metric_gate.go` | Multi-Metric Gate with Default/Lenient/Strict profiles, auto-applied via ApplyGate |

**Strategy runners** (`internal/backtest/strategy/`):

| File | Strategy | Key Features |
|------|----------|-------------|
| `orb_runner.go` | Opening Range Breakout | Entry buffer, ATR stops, time-of-day exit, regime gate (blocks Calm+Crisis) |
| `trend_runner.go` | Trend Following | EMA crossover, ADX confirmation, trailing stop, exit signal propagation |
| `grid_runner.go` | Grid Trading | Multi-level grid, take-profit, regime gate (blocks Crisis), exit signal propagation |
| `session_scalp_runner.go` | Session Scalping | 9:30-11:00 ET window, volume-confirmed breakout, 2:1 R:R, time exit, regime-gated |
| `registry.go` | Strategy Registry | Plugin registry with DefaultRegistry() registering all runners |

### `internal/risk/` — Risk Management

| File | Purpose |
|------|---------|
| `kill_switch.go` | Kill-switch: atomic halted flag, re-entrancy guard, broker cancel/close |
| `rate_limiter.go` | Dynamic rate limiter using Redis sorted sets, regime-aware limits |
| `trading_controls.go` | **OrderRateLimiter** (10/sec per symbol), **VolatilityHalt** (z-score >3.0), **ExposureTracker** (max leverage 5×, per-symbol 25%) |
| `position_sizer.go` | Shared PositionSizer: Kelly (k=0.25), regime, VIX, sentiment, correlation scaling. Used by both backtest and live paths |
| `hmm_validation.go` | HMM parameter sanity checks: emission SD ordering, transition plausibility, row sums |
| `memory_guard.go` | Security memory guard framework |
| `credential.go` | API key rotation tracking, vault integration |

### `internal/db/` — Database Layer

PostgreSQL + TimescaleDB access via `pgx/v5`.

| File | Purpose |
|------|---------|
| `repository.go` | Full CRUD: strategies (7 types), symbols, providers, trades, regime logs, candles |
| `seeder.go` | Development seed data: strategies, symbols, regime logs |
| `fixtures.go` | Fixtures: 7 strategies (grid, mean_reversion, breakout, trend, scalp, stat_arb, vol_arb), 17 Stooq symbols |
| `migrations/` | SQL migration files (golang-migrate compatible). Initial schema with strategies, symbols, candles, regime_logs |

### `internal/propfirm/` — Prop Firm Profiles

Vendor-agnostic profile system. Single `Profile` struct supports FTMO, TopStep, E8, TFT via YAML configs.

| File | Purpose |
|------|---------|
| `profile.go` | Profile struct: MaxDailyLossPct, MaxDrawdownPct, DrawdownType, RegimeMultipliers, etc. DefaultProfile() returns FTMO-compatible defaults |

### `internal/scheduler/` — Background Jobs

| File | Purpose |
|------|---------|
| `scheduler.go` | Goroutine-based scheduler: VIX fetch (60s), sentiment fetch (3600s), WebSocket risk broadcast (5s with live regime), key rotation, daily reset |

### `internal/monitor/` — Monitoring & Telemetry

| File | Purpose |
|------|---------|
| `ws_hub.go` | WebSocket hub: client registration, per-channel broadcast |
| `datapipeline.go` | Data pipeline: ring buffer → CVD → WebSocket |
| `telegram.go` | Telegram bot alerts: KillSwitch, RegimeChanged, CredentialExpiry |
| `metrics.go` | Prometheus metrics registration |

### `internal/config/` — Configuration

| File | Purpose |
|------|---------|
| `config.go` | YAML config loader with env override |
| `strategy_config.go` | Strategy params loader |
| `feature_flags.go` | Feature flag system |

### `internal/universe/` — Symbol Universe

| File | Purpose |
|------|---------|
| `manager.go` | Universe manager: refresh, override, activate configs |
| `filters.go` | Dynamic trigger thresholds including NewsSentimentAbsMin |
| `market_snapshot.go` | Market data snapshot with VIX and FearGreedIndex |

## Dependencies

- `github.com/gin-gonic/gin` — HTTP framework
- `github.com/gorilla/websocket` — WebSocket
- `github.com/jackc/pgx/v5` — PostgreSQL driver
- `github.com/prometheus/client_golang` — Prometheus metrics
- `gopkg.in/yaml.v3` — YAML config parsing

## Testing

```bash
go test ./internal/... -count=1           # Unit tests
go test ./internal/backtest/... -v        # Backtest engine tests (FTMO, exit signals, adverse selection, volatility halt)
go test ./internal/risk/... -v            # Risk tests (PositionSizer VIX/sentiment, HMM validation)
go test -race ./internal/... -count=1     # Race detector
```
