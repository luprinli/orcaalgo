# `cmd/` — Entry Points

Go binary entry points for the OrcaAlgo platform.

[↑ Back to Root README](../README.md)

## Binaries

### `orca-server` — HTTP API Server

Main application server exposing REST API, WebSocket, and Prometheus metrics.

```bash
go run ./cmd/orca-server
# API:     http://localhost:8080
# Metrics: http://localhost:9091/metrics
# WS:      ws://localhost:8080/ws
```

Environment variables:
- `PAPER_TRADING=true` — paper trading mode (no broker keys required)
- `ORCA_DATA_MODE=mock` — simulated tick feed for development
- See `.env.example` for full configuration

### `orca-cli` — Go CLI Utilities

Auxiliary Go CLI for server operations.

```bash
go run ./cmd/orca-cli health     # Health check
go run ./cmd/orca-cli migrate    # Database migration status
```
