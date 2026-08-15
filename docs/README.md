# `docs/` — Project Documentation

Architecture specifications, technical references, and operations guides.

[↑ Back to Root README](../README.md)

## Reference Documents

| File | Content |
|------|---------|
| `benchmark_filter_design.md` | Market-based benchmark promotion filter — design, configurable benchmark selection, lifecycle, and implementation status |
| `ml4t_orca_comparison.md` | Cross-codebase leverage analysis and implementation roadmap (incl. frontend + migrations plan) |
| `grafana/` | Grafana monitoring dashboard setup, JWT authentication, and CI validation documents |
| `runbooks/` | Operational runbooks: startup/shutdown, kill-switch, database migrations, incident response |

## Subsystem Map (onboarding index)

| Subsystem | Entry points |
|-----------|--------------|
| Risk pipeline / gates | `internal/risk/pipeline.go`, `internal/risk/interfaces.go`, `internal/risk/signal_gate_impl.go` |
| Capital pools / prop-firm | `internal/risk/capital_pool.go`, `internal/propfirm/pool_base.go`, `internal/backtest/propfirm_enforcer.go`, `internal/backtest/capital_pool_sim.go` |
| Backtest engine | `internal/backtest/engine.go`, `internal/backtest/slippage.go`, `internal/backtest/optimized_walk_forward.go` |
| Broker adapters | `internal/broker/` (Alpaca/IBKR/Paper), `internal/broker/fee.go`, `internal/broker/retry/retry.go` |
| Market data | `internal/ingest/` (WebSocket → ring buffer), `internal/db/repository_candles.go` |
| Python math / sizing | `orca/sizing/` (Kelly, block bootstrap, multiple testing, DSR, Sharpe SE), `orca/math/`, `orca/costs/` |
| Benchmark filter | `orca/benchmark/` (spec/metrics/filter), `internal/benchmark/` (Go subprocess), `internal/db/benchmark_evals.go` + `benchmark_series.go`, `internal/api/benchmark_eval_handler.go` |
| Migrations | `internal/db/repository_strategies.go` (`RunMigrations` — Go-managed runner), `cmd/migrate/`, `internal/db/migrations/*.up.sql` |
| Strategy IR | `orca/ir/` (loader, validator, compiler), `configs/strategies/*.gkr.yaml` |
| CLI | `orca/cli.py` (validate, calibrate, preflight, attribute, promote-gate, benchmark-filter, backtest-stats, calibrate-costs, ingest-risk-free, features, …) |

## Operational Runbooks

| File | Content |
|------|---------|
| `runbooks/startup-shutdown.md` | Service startup and graceful shutdown procedures |
| `runbooks/kill-switch.md` | Kill-switch activation and recovery procedures |
| `runbooks/database-migrations.md` | Database migration guide |
| `runbooks/incident-response.md` | Incident classification and response protocols |
